package libws

import (
	"fmt"
	"io"
	"time"
)

// testLogger implements the logger interface using an io.Writer
type testLogger struct {
	writer io.Writer
	fields map[string]any
}

// NewTestLogger creates a new logger that writes to the provided writer
func newTestLogger(writer io.Writer) logger {
	return &testLogger{
		writer: writer,
		fields: make(map[string]any),
	}
}

func (l *testLogger) WithField(key string, value any) logger {
	newLogger := &testLogger{
		writer: l.writer,
		fields: make(map[string]any),
	}
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

func (l *testLogger) formatFields() string {
	if len(l.fields) == 0 {
		return ""
	}

	result := " ["
	first := true
	for k, v := range l.fields {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s=%v", k, v)
		first = false
	}
	result += "]"
	return result
}

func (l *testLogger) log(level, msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fields := l.formatFields()
	fmt.Fprintf(l.writer, "[%s] %s%s: %s\n", timestamp, level, fields, msg)
}

func (l *testLogger) Debug(args ...any) {
	l.log("DEBUG", fmt.Sprint(args...))
}

func (l *testLogger) Debugf(format string, args ...any) {
	l.log("DEBUG", fmt.Sprintf(format, args...))
}

func (l *testLogger) Debugln(args ...any) {
	l.log("DEBUG", fmt.Sprintln(args...))
}

func (l *testLogger) Info(args ...any) {
	l.log("INFO", fmt.Sprint(args...))
}

func (l *testLogger) Infof(format string, args ...any) {
	l.log("INFO", fmt.Sprintf(format, args...))
}

func (l *testLogger) Infoln(args ...any) {
	l.log("INFO", fmt.Sprintln(args...))
}

func (l *testLogger) Warn(args ...any) {
	l.log("WARN", fmt.Sprint(args...))
}

func (l *testLogger) Warnf(format string, args ...any) {
	l.log("WARN", fmt.Sprintf(format, args...))
}

func (l *testLogger) Warnln(args ...any) {
	l.log("WARN", fmt.Sprintln(args...))
}

func (l *testLogger) Error(args ...any) {
	l.log("ERROR", fmt.Sprint(args...))
}

func (l *testLogger) Errorf(format string, args ...any) {
	l.log("ERROR", fmt.Sprintf(format, args...))
}

func (l *testLogger) Errorln(args ...any) {
	l.log("ERROR", fmt.Sprintln(args...))
}
