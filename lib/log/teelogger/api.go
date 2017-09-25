package teelogger

import (
	"fmt"

	"github.com/Symantec/Dominator/lib/log"
)

type flusher interface {
	Flush() error
}

type Logger struct {
	one log.Logger
	two log.Logger
}

func New(one, two log.Logger) *Logger {
	return &Logger{one, two}
}

func (l *Logger) Fatal(v ...interface{}) {
	msg := fmt.Sprint(v...)
	l.one.Print(msg)
	if fl, ok := l.one.(flusher); ok {
		fl.Flush()
	}
	l.two.Fatal(msg)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.one.Print(msg)
	if fl, ok := l.one.(flusher); ok {
		fl.Flush()
	}
	l.two.Fatal(msg)
}

func (l *Logger) Fatalln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	l.one.Print(msg)
	if fl, ok := l.one.(flusher); ok {
		fl.Flush()
	}
	l.two.Fatal(msg)
}

func (l *Logger) Panic(v ...interface{}) {
	msg := fmt.Sprint(v...)
	l.one.Print(msg)
	l.two.Panic(msg)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.one.Print(msg)
	l.two.Panic(msg)
}

func (l *Logger) Panicln(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	l.one.Print(msg)
	l.two.Panic(msg)
}

func (l *Logger) Print(v ...interface{}) {
	msg := fmt.Sprint(v...)
	l.one.Print(msg)
	l.two.Print(msg)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	l.one.Print(msg)
	l.two.Print(msg)
}

func (l *Logger) Println(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	l.one.Print(msg)
	l.two.Print(msg)
}
