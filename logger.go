package main

import (
	"log"
	"os"
)

const (
	LOG_EXIT = iota
	LOG_CRI
	LOG_ERR
	LOG_WARN
	LOG_INFO
	LOG_DEBUG
)

type Logger struct {
	log   *log.Logger
	level int
}

func NewLogger(level int) *Logger {
	return &Logger{
		log:   log.New(os.Stdout, "snet:", log.LUTC|log.LstdFlags),
		level: level,
	}
}

func (l *Logger) l(level int, v ...interface{}) {
	if l.level >= level {
		l.log.Println(v...)
	}
}

func (l *Logger) Debug(v ...interface{}) {
	l.l(LOG_DEBUG, v...)
}

func (l *Logger) Info(v ...interface{}) {
	l.l(LOG_INFO, v...)
}

func (l *Logger) Err(v ...interface{}) {
	l.l(LOG_ERR, v...)
}

func (l *Logger) Exit(v ...interface{}) {
	l.l(LOG_EXIT, v...)
	os.Exit(1)
}

func (l *Logger) Warn(v ...interface{}) {
	l.l(LOG_WARN, v...)
}
