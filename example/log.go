package main

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	LogLevel int
	Prefix string
}

const (
	TRACE=iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

func (l *Logger) Trace(format string, v ...interface{}) {
	if l.LogLevel >= TRACE {
		log.Printf(fmt.Sprintf("trace: %s: %s\n", l.Prefix, format), v...)
	}
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.LogLevel >= DEBUG {
		log.Printf(fmt.Sprintf("debug: %s: %s\n", l.Prefix, format), v...)
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.LogLevel >= INFO {
		log.Printf(fmt.Sprintf("info: %s: %s\n", l.Prefix, format), v...)
	}
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if l.LogLevel >= WARN {
		log.Printf(fmt.Sprintf("warn: %s: %s\n", l.Prefix, format), v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.LogLevel >= ERROR {
		log.Printf(fmt.Sprintf("error: %s: %s\n", l.Prefix, format), v...)
	}
}

func (l *Logger) Fatal(format string, v ...interface{}) {
	if l.LogLevel >= FATAL {
		log.Printf(fmt.Sprintf("fatal: %s: %s\n", l.Prefix, format), v...)
	}
	os.Exit(1)
}
