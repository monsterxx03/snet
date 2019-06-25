// Package logger provide a simple wrapper around standard lib "log".
// Support logging level, and print the correct logging line number.
package logger

import (
	"fmt"
	"log"
	"os"
)

type Level int

const (
	_           = iota
	DEBUG Level = 10 * iota
	INFO
	WARNING
	ERROR
	FATAL
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "Debug"
	case INFO:
		return "Info"
	case WARNING:
		return "Warn"
	case ERROR:
		return "Error"
	case FATAL:
		return "Fatal"
	default:
		return fmt.Sprintf("Unknown log level:%d", l)
	}
}

type Logger struct {
	level Level
	log   *log.Logger
}

func (l *Logger) Debug(v ...interface{}) {
	if l.level <= DEBUG {
		l.log.Output(2, "Debug:"+fmt.Sprintln(v...))
	}
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.level <= DEBUG {
		l.log.Output(2, "Debug:"+fmt.Sprintf(format, v...))
	}
}

func (l *Logger) Info(v ...interface{}) {
	if l.level <= INFO {
		l.log.Output(2, "Info:"+fmt.Sprintln(v...))
	}
}

func (l *Logger) Infof(format string, v ...interface{}) {
	if l.level <= INFO {
		l.log.Output(2, "Info:"+fmt.Sprintf(format, v...))
	}
}

func (l *Logger) Warn(v ...interface{}) {
	if l.level <= WARNING {
		l.log.Output(2, "Warn:"+fmt.Sprintln(v...))
	}
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.level <= WARNING {
		l.log.Output(2, "Warn:"+fmt.Sprintf(format, v...))
	}
}

func (l *Logger) Error(v ...interface{}) {
	if l.level <= ERROR {
		l.log.Output(2, "Error:"+fmt.Sprintln(v...))
	}
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.level <= ERROR {
		l.log.Output(2, "Error:"+fmt.Sprintf(format, v...))
	}
}

func (l *Logger) Fatal(v ...interface{}) {
	l.log.Output(2, "Fatal:"+fmt.Sprintln(v...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.log.Output(2, "Fatal:"+fmt.Sprintf(format, v...))
	os.Exit(1)
}

func NewLogger(level Level) *Logger {
	return &Logger{level, log.New(os.Stdout, "", log.Llongfile|log.LstdFlags)}
}
