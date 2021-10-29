package logger

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

type logger struct {
	logger *logrus.Entry
}

// WithFields adds new fields to log.
func (l *logger) WithFields(fields map[string]interface{}) Logger {
	return &logger{
		logger: l.logger.WithFields(logrus.Fields(fields)),
	}
}

// Debug logs a message at level Debug.
func (l *logger) Debug(args ...interface{}) {
	l.log(logrus.DebugLevel, args...)
}

// Debugf logs a message at level Debug.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.logf(logrus.DebugLevel, format, args...)
}

// Info logs a message at level Info.
func (l *logger) Info(args ...interface{}) {
	l.log(logrus.InfoLevel, args...)
}

// Infof logs a message at level Info.
func (l *logger) Infof(format string, args ...interface{}) {
	l.logf(logrus.InfoLevel, format, args...)
}

// Warn logs a message at level Warn.
func (l *logger) Warn(args ...interface{}) {
	l.log(logrus.WarnLevel, args...)
}

// Warnf logs a message at level Warn.
func (l *logger) Warnf(format string, args ...interface{}) {
	l.logf(logrus.WarnLevel, format, args...)
}

// Error logs a message at level Error.
func (l *logger) Error(args ...interface{}) {
	l.log(logrus.ErrorLevel, args...)
}

// Errorf logs a message at level Error.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.logf(logrus.ErrorLevel, format, args...)
}

// Fatal logs a message at level Fatal then the process will exit with status set to 1.
func (l *logger) Fatal(args ...interface{}) {
	l.log(logrus.FatalLevel, args...)
}

// Fatalf logs a message at level Fatal then the process will exit with status set to 1.
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.logf(logrus.FatalLevel, format, args...)
}

func (l *logger) GetLevel() LogLevel {
	return LogLevel(l.logger.Logger.GetLevel().String())
}

func (l *logger) IsLevelEnabled(level LogLevel) bool {
	lvl, _ := logrus.ParseLevel(string(level))
	return l.logger.Logger.IsLevelEnabled(lvl)
}

func (l *logger) log(level logrus.Level, args ...interface{}) {
	lg := l.logger
	if l.logger.Logger.IsLevelEnabled(logrus.DebugLevel) {
		lg = lg.WithField("caller", l.caller(3))
	}
	lg.Log(level, args...)
}

func (l *logger) logf(level logrus.Level, format string, args ...interface{}) {
	lg := l.logger
	if l.logger.Logger.IsLevelEnabled(logrus.DebugLevel) {
		lg = lg.WithField("caller", l.caller(3))
	}
	lg.Logf(level, format, args...)
}

func (l *logger) caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		file = "<???>"
	} else {
		file = filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file))
	}
	return fmt.Sprintf("%s:%d", file, line)
}
