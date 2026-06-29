package logger

import (
	"log"
	"os"
)

type Level uint

const (
	Info  Level = iota
	Warn  Level = iota
	Error Level = iota
)

type Log struct {
	logLevel Level
	info     *log.Logger
	warn     *log.Logger
	err      *log.Logger
}

func New(logLevel Level) *Log {
	return &Log{
		logLevel: logLevel,
		info:     log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		warn:     log.New(os.Stderr, "[WARN] ", log.LstdFlags),
		err:      log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
	}
}

func (l Log) Info(msg string) {
	l.print(Info, msg)
}

func (l Log) Warn(msg string) {
	l.print(Warn, msg)
}

func (l Log) WarnErr(msg string, err error) {
	l.print(Warn, msg, err)
}

func (l Log) Fatal(msg string, err error) {
	l.print(Error, msg, err)
}

func (l Log) print(logLevel Level, msg ...any) {
	if logLevel < l.logLevel {
		return
	}

	switch logLevel {
	case Info:
		l.info.Println(msg...)
	case Warn:
		l.warn.Println(msg...)
	case Error:
		l.err.Println(msg...)
		os.Exit(1)
	}
}
