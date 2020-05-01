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
}

func New(logLevel Level) *Log {
	return &Log{
		logLevel: logLevel,
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

func (l Log) print(logLevel Level, msg ...interface{}) {
	if logLevel < l.logLevel {
		return
	}

	log.SetOutput(os.Stderr)

	switch logLevel {
	case Info:
		log.SetPrefix("[INFO]")
		log.SetOutput(os.Stdout)
	case Warn:
		log.SetPrefix("[WARN]")
	case Error:
		log.SetPrefix("[ERROR]")

		defer os.Exit(1)
	}

	log.Println(msg...)
}
