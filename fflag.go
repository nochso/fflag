// Package fflag parses flag.FlagSet from simple configuration files.
//
// Syntax
//
// Keys (flag names without the ``-'' prefix) followed by values.
//
//   flag-name flag-value
//
// Comments begin with any of these: # ; //
//
// Leading and trailing whitespace is ignored on each line, key and value.
package fflag

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// LogFunc provides a Logger implementation for a Printf style function.
type LogFunc func(string, ...interface{})

// Printf implements the Logger interface for a Printf style function.
func (lf LogFunc) Printf(format string, a ...interface{}) {
	lf(format, a...)
}

// Logger is used to log warnings while parsing a config file.
type Logger interface {
	Printf(string, ...interface{})
}

// Options used for parsing a config file.
type Options struct {
	// Logger logs warnings (unknown keys/names found in file)
	Logger Logger

	// Path is the default config file path.
	//
	// If this file does exist, no error is returned.
	// If a specific file is given using FlagName, the file must exist.
	Path string

	// ConfigFlagName is the name of the flag that points to a config file.
	//
	// If this flag is invoked a non-existing file will return an error.
	ConfigFlagName string

	// WriteConfigFlagName is the name of the flag that causes the current
	// configuration to be printed.
	WriteConfigFlagName string
}

// NewDefaultOptions returns default options for use in Parse.
//
//   Path:           "config.txt"
//   ConfigFlagName: "config"
func NewDefaultOptions() *Options {
	return &Options{
		Path:                "config.txt",
		ConfigFlagName:      "config",
		WriteConfigFlagName: "write-config",
	}
}

func WriteFlagSetConfig(w io.Writer, fs *flag.FlagSet, ignoreFlags ...string) {
	flags := make(map[string]struct{}, len(ignoreFlags))
	for i := range ignoreFlags {
		flags[ignoreFlags[i]] = struct{}{}
	}
	fmt.Fprint(w, multilineComment(`fflag file syntax:

  flag-name flag-value

where flag-name is an argument name without the "-" prefix.

Comments begin with any of these: # ' ; //

Leading and trailing whitespace is ignored on each line, key and value.`))
	fmt.Fprint(w, "\n\n")
	fs.VisitAll(func(f *flag.Flag) {
		if _, ignore := flags[f.Name]; ignore {
			return
		}
		fmt.Fprintf(w, "%s\n//\n// default:\n// %s %s\n", multilineComment(f.Name+": "+f.Usage), f.Name, f.DefValue)
		if f.DefValue != f.Value.String() {
			fmt.Fprintf(w, "%s %s\n", f.Name, f.Value)
		}
		fmt.Fprintln(w)
	})
}

func multilineComment(s string) string {
	return "// " + strings.ReplaceAll(s, "\n", "\n// ")
}

// Parse a config file into an existing FlagSet.
func Parse(fs *flag.FlagSet, o *Options) error {
	if o == nil {
		o = NewDefaultOptions()
	}
	fs.String(o.ConfigFlagName, o.Path, "path to config file")
	configPath := getFlagConfigPath(o.ConfigFlagName)
	fileMustExist := false
	if configPath != "" {
		// a specific config was requested using config flag. insist on the file existing.
		fileMustExist = true
		o.Path = configPath
	}
	p := &parser{
		fileMustExist: fileMustExist,
		fs:            fs,
		Options:       o,
		textFlags:     map[string]string{},
	}
	err := p.parse()
	if err != nil {
		return err
	}

	if o.WriteConfigFlagName != "" && fs.Lookup(o.WriteConfigFlagName).Value.String() == "true" {
		WriteFlagSetConfig(os.Stdout, fs, o.ConfigFlagName, o.WriteConfigFlagName)
	}
	return nil
}

func getFlagConfigPath(configFlagName string) string {
	f := flag.NewFlagSet(configFlagName, flag.ContinueOnError)
	// don't care about -h here and errors are handled by p.visitFlag
	f.SetOutput(io.Discard)
	var configPath string
	f.StringVar(&configPath, configFlagName, "", "path to config file")
	f.Parse(os.Args[1:])
	return configPath
}

type parser struct {
	*Options
	lineNo int
	fs     *flag.FlagSet
	// true when a specific config was requested via -config flag
	fileMustExist bool
	textFlags     map[string]string
	errors        errs
}

func (p *parser) logf(format string, a ...interface{}) {
	if p.Logger == nil {
		return
	}
	p.Logger.Printf(format, a...)
}

func (p *parser) parse() error {
	if err := p.scanTextFlags(); err != nil {
		return err
	}
	p.fs.VisitAll(p.visitFlag)
	if len(p.errors) > 0 {
		return p.errors
	}
	return nil
}

func (p *parser) visitFlag(f *flag.Flag) {
	v, ok := p.textFlags[f.Name]
	if !ok {
		return
	}
	if err := f.Value.Set(v); err != nil {
		err = fmt.Errorf("fflag: error setting flag %q = %q from config file %q: %w", f.Name, v, p.Path, err)
		p.errors = append(p.errors, err)
	}
}

func (p *parser) scanTextFlags() error {
	f, err := os.Open(p.Path)
	if err != nil {
		if !p.fileMustExist && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("fflag: error reading -%s=%q: %w", p.ConfigFlagName, p.Path, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		p.scanLine(sc)
	}
	return sc.Err()
}

func (p *parser) scanLine(sc *bufio.Scanner) {
	p.lineNo++
	k, v := parseLine(sc.Text())
	if k == "" {
		return
	}
	p.textFlags[k] = v
	if fl := p.fs.Lookup(k); fl == nil {
		p.logf("warning: config file %q line %d contains unknown flag name %q", p.Path, p.lineNo, k)
	}
}

func parseLine(line string) (k, v string) {
	line = strings.TrimSpace(line)
	if isComment(line) {
		return "", ""
	}
	parts := strings.SplitN(line, " ", 2)
	k = parts[0]
	if len(parts) == 2 {
		v = strings.TrimSpace(parts[1])
	}
	return
}

func isComment(line string) bool {
	if len(line) == 0 {
		return true
	}
	if len(line) >= 2 && line[:2] == "//" {
		return true
	}
	b := line[0]
	return b == ';' || b == '#' || b == '\''
}

type errs []error

func (e errs) Error() string {
	if len(e) == 1 {
		return e[0].Error()
	}
	if len(e) == 0 {
		return ""
	}
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "%d errors:", len(e))
	for i, err := range e {
		fmt.Fprintf(sb, "\n%d. %v", i, err)
	}
	return sb.String()
}
