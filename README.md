<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# fflag

```go
import "github.com/nochso/fflag"
```

Package fflag parses flag\.FlagSet from simple configuration files\.

### Syntax

Keys \(flag names without the \`\`\-'' prefix\) followed by values\.

```
flag-name flag-value
```

Comments begin with any of these: \# ; //

Leading and trailing whitespace is ignored on each line\, key and value\.

## Index

- [func Parse(fs *flag.FlagSet, o *Options) error](<#func-parse>)
- [func WriteFlagSetConfig(w io.Writer, fs *flag.FlagSet, ignoreFlags ...string)](<#func-writeflagsetconfig>)
- [type LogFunc](<#type-logfunc>)
  - [func (lf LogFunc) Printf(format string, a ...interface{})](<#func-logfunc-printf>)
- [type Logger](<#type-logger>)
- [type Options](<#type-options>)
  - [func NewDefaultOptions() *Options](<#func-newdefaultoptions>)


## func Parse

```go
func Parse(fs *flag.FlagSet, o *Options) error
```

Parse a config file into an existing FlagSet\.

## func WriteFlagSetConfig

```go
func WriteFlagSetConfig(w io.Writer, fs *flag.FlagSet, ignoreFlags ...string)
```

## type LogFunc

LogFunc provides a Logger implementation for a Printf style function\.

```go
type LogFunc func(string, ...interface{})
```

### func \(LogFunc\) Printf

```go
func (lf LogFunc) Printf(format string, a ...interface{})
```

Printf implements the Logger interface for a Printf style function\.

## type Logger

Logger is used to log warnings while parsing a config file\.

```go
type Logger interface {
    Printf(string, ...interface{})
}
```

## type Options

Options used for parsing a config file\.

```go
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
```

### func NewDefaultOptions

```go
func NewDefaultOptions() *Options
```

NewDefaultOptions returns default options for use in Parse\.

```
Path:           "config.txt"
ConfigFlagName: "config"
```



Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)
