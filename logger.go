package main

import (
	"fmt"
	"log"
	"os"
)

type LogLevel int

const (
	LOG_EXIT LogLevel = iota
	LOG_CRI
	LOG_ERR
	LOG_WARN
	LOG_INFO
	LOG_DEBUG
)

var levelmap = [...]string{
	"EXIT",
	"CRITICAL",
	"ERROR",
	"WARN",
	"INFO",
	"DEBUG",
}

func (level LogLevel) String() string {
	return levelmap[int(level)]
}

type Logger struct {
	log   *log.Logger
	level LogLevel
}

func NewLogger(level LogLevel) *Logger {
	return &Logger{
		log:   log.New(os.Stdout, "snet:", log.LUTC|log.LstdFlags),
		level: level,
	}
}

func (l *Logger) l(level LogLevel, v ...interface{}) {
	if l.level >= level {
		l.log.Println(append([]interface{}{fmt.Sprintf("%s:", level)}, v...)...)
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
