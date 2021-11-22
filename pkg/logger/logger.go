package logger

import (
	"io"
)

// LogFormat is format type
type LogFormat string

const (
	TextFormat LogFormat = "text"
	JSONFormat LogFormat = "json"
)

// LogLevel is Logger Level type
type LogLevel string

const (
	// DebugLevel has verbose message
	DebugLevel LogLevel = "debug"
	// InfoLevel is default log level
	InfoLevel LogLevel = "info"
	// WarnLevel is for logging messages about possible issues
	WarnLevel LogLevel = "warn"
	// ErrorLevel is for logging errors
	ErrorLevel LogLevel = "error"
	// FatalLevel is for logging fatal messages. The system shuts down after logging the message.
	FatalLevel LogLevel = "fatal"
)

type Logger interface {
	WithFields(map[string]interface{}) Logger
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	GetLevel() LogLevel
	IsLevelEnabled(level LogLevel) bool
}

type LoggerOptions struct {
	Output io.Writer
	Format LogFormat
	Level  LogLevel
}

type LoggerOption func(opts *LoggerOptions)

func OutputLoggerOption(out io.Writer) LoggerOption {
	return func(opts *LoggerOptions) {
		opts.Output = out
	}
}

func FormatLoggerOption(format LogFormat) LoggerOption {
	return func(opts *LoggerOptions) {
		opts.Format = format
	}
}

func LevelLoggerOption(level LogLevel) LoggerOption {
	return func(opts *LoggerOptions) {
		opts.Level = level
	}
}
