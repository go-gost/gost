package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var (
	_ Logger = (*logger)(nil)
)

type logger struct {
	logger *logrus.Entry
}

func newLogger(name string) *logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)

	gl := &logger{
		logger: l.WithFields(logrus.Fields{
			logFieldScope: name,
		}),
	}

	return gl
}

// EnableJSONOutput enables JSON formatted output log.
func (l *logger) EnableJSONOutput(enabled bool) {
	l.logger.Logger.SetFormatter(&logrus.JSONFormatter{})
}

// SetOutputLevel sets log output level
func (l *logger) SetLevel(level LogLevel) {
	lvl, _ := logrus.ParseLevel(string(level))
	l.logger.Logger.SetLevel(lvl)
}

// WithFields adds new fields to log.
func (l *logger) WithFields(fields map[string]interface{}) Logger {
	return &logger{
		logger: l.logger.WithFields(logrus.Fields(fields)),
	}
}

// Debug logs a message at level Debug.
func (l *logger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

// Debugf logs a message at level Debug.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Info logs a message at level Info.
func (l *logger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

// Infof logs a message at level Info.
func (l *logger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Warn logs a message at level Warn.
func (l *logger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

// Warnf logs a message at level Warn.
func (l *logger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

// Error logs a message at level Error.
func (l *logger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

// Errorf logs a message at level Error.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

// Fatal logs a message at level Fatal then the process will exit with status set to 1.
func (l *logger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

// Fatalf logs a message at level Fatal then the process will exit with status set to 1.
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}
