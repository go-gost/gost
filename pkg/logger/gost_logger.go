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

// Info logs a message at level Info.
func (l *logger) Info(args ...interface{}) {
	l.logger.Log(logrus.InfoLevel, args...)
}

// Infof logs a message at level Info.
func (l *logger) Infof(format string, args ...interface{}) {
	l.logger.Logf(logrus.InfoLevel, format, args...)
}

// Debug logs a message at level Debug.
func (l *logger) Debug(args ...interface{}) {
	l.logger.Log(logrus.DebugLevel, args...)
}

// Debugf logs a message at level Debug.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.logger.Logf(logrus.DebugLevel, format, args...)
}

// Warn logs a message at level Warn.
func (l *logger) Warn(args ...interface{}) {
	l.logger.Log(logrus.WarnLevel, args...)
}

// Warnf logs a message at level Warn.
func (l *logger) Warnf(format string, args ...interface{}) {
	l.logger.Logf(logrus.WarnLevel, format, args...)
}

// Error logs a message at level Error.
func (l *logger) Error(args ...interface{}) {
	l.logger.Log(logrus.ErrorLevel, args...)
}

// Errorf logs a message at level Error.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.logger.Logf(logrus.ErrorLevel, format, args...)
}

// Fatal logs a message at level Fatal then the process will exit with status set to 1.
func (l *logger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

// Fatalf logs a message at level Fatal then the process will exit with status set to 1.
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}
