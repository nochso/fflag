// Package fflag parses [flag.FlagSet] from simple configuration files.
//
// # Syntax
//
// Keys (flag names without the "-" prefix) followed by values.
//
//	flag-name flag-value
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

// Options used for parsing a config file.
type Options struct {
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

// NewDefaultOptions returns default options for use in [Parse].
//
//	Path:                "config.txt"
//	ConfigFlagName:      "config"
//	WriteConfigFlagName: "write-config"
func NewDefaultOptions() *Options {
	return &Options{
		Path:                "config.txt",
		ConfigFlagName:      "config",
		WriteConfigFlagName: "write-config",
	}
}

// WriteFlagSetConfig writes a configuration file to w including both default
// and currently set values (should they differ).
func WriteFlagSetConfig(w io.Writer, fs *flag.FlagSet, ignoreFlags ...string) {
	flags := make(map[string]struct{}, len(ignoreFlags))
	for i := range ignoreFlags {
		flags[ignoreFlags[i]] = struct{}{}
	}
	fmt.Fprint(w, multilineComment(`fflag file syntax:

  flag-name flag-value

where flag-name is an argument name without the "-" prefix.

Comments begin with any of these: # ' ; //

Leading and trailing whitespace is ignored on each line, key and value.`, 1))
	fmt.Fprint(w, "\n\n")
	fs.VisitAll(func(f *flag.Flag) {
		if _, ignore := flags[f.Name]; ignore {
			return
		}
		fmt.Fprintf(w,
			"# %s\n%s\n#\n# default:\n# %s %s\n",
			f.Name,
			multilineComment(f.Usage, 3),
			f.Name,
			f.DefValue,
		)
		if f.DefValue != f.Value.String() {
			fmt.Fprintf(w, "%s %s\n", f.Name, f.Value)
		}
		fmt.Fprintln(w)
	})
}

func multilineComment(s string, indent int) string {
	ind := strings.Repeat(" ", indent)
	return "#" + ind + strings.ReplaceAll(s, "\n", "\n#"+ind)
}

// ErrWriteConfig is returned by [Parse] after the current configuration has been
// to written to [os.Stdout].
var ErrWriteConfig = errors.New("wrote configuration to stdout")

// Parse a config file using [os.Args] into an existing [flag.FlagSet] before parsing the FlagSet itself.
//
// Returns [ErrWriteConfig] if the configuration was written to stdout as requested.
func Parse(fs *flag.FlagSet, o *Options) error {
	return ParseArgs(fs, o, os.Args[1:])
}

// ParseArgs parses a config file using given arguments into an existing [flag.FlagSet] before parsing the FlagSet itself.
//
// Returns [ErrWriteConfig] if the configuration was written to stdout as requested.
func ParseArgs(fs *flag.FlagSet, o *Options, args []string) error {
	if o == nil {
		o = NewDefaultOptions()
	}
	fs.String(o.ConfigFlagName, o.Path, "path to config file")
	if o.WriteConfigFlagName != "" {
		fs.Bool(o.WriteConfigFlagName, false, "write configuration to stdout and exit")
	}
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
	err = fs.Parse(args)
	if err != nil && !errors.Is(err, ErrWriteConfig) {
		return err
	}
	if getFlagWriteConfig(o.WriteConfigFlagName) {
		WriteFlagSetConfig(os.Stdout, fs, o.ConfigFlagName, o.WriteConfigFlagName)
		return ErrWriteConfig
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

func getFlagWriteConfig(writeConfigFlagName string) bool {
	f := flag.NewFlagSet(writeConfigFlagName, flag.ContinueOnError)
	// don't care about -h here and errors are handled by p.visitFlag
	f.SetOutput(io.Discard)
	var writeConfig bool
	f.BoolVar(&writeConfig, writeConfigFlagName, false, "write configuration to stdout")
	f.Parse(os.Args[1:])
	return writeConfig
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
		err = p.scanLine(sc)
		if err != nil {
			p.errors = append(p.errors, err)
		}
	}
	return sc.Err()
}

func (p *parser) scanLine(sc *bufio.Scanner) error {
	p.lineNo++
	k, v := parseLine(sc.Text())
	if k == "" {
		return nil
	}
	p.textFlags[k] = v
	if fl := p.fs.Lookup(k); fl == nil {
		return fmt.Errorf("fflag: config file %q line %d contains unknown flag name %q", p.Path, p.lineNo, k)
	}
	return nil
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
