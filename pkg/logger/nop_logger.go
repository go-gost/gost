package logger

var (
	nop = &nopLogger{}
)

func Nop() Logger {
	return nop
}

type nopLogger struct{}

func (l *nopLogger) WithFields(fields map[string]interface{}) Logger {
	return l
}

func (l *nopLogger) Debug(args ...interface{}) {
}

func (l *nopLogger) Debugf(format string, args ...interface{}) {
}

func (l *nopLogger) Info(args ...interface{}) {
}

func (l *nopLogger) Infof(format string, args ...interface{}) {
}

func (l *nopLogger) Warn(args ...interface{}) {
}

func (l *nopLogger) Warnf(format string, args ...interface{}) {
}

func (l *nopLogger) Error(args ...interface{}) {
}

func (l *nopLogger) Errorf(format string, args ...interface{}) {
}

func (l *nopLogger) Fatal(args ...interface{}) {
}

func (l *nopLogger) Fatalf(format string, args ...interface{}) {
}

func (l *nopLogger) GetLevel() LogLevel {
	return ""
}

func (l *nopLogger) IsLevelEnabled(level LogLevel) bool {
	return false
}
