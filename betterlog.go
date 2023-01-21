package main

import (
	golog "log"
)

type betterlog struct{}

func (b betterlog) Fatalf(format string, v ...any) {
	golog.Fatalf(format, v...)
}
func (b betterlog) Printf(format string, v ...any) {
	golog.Printf(format, v...)
}

func (b betterlog) Debugf(format string, v ...any) {
	golog.Printf(format, v...)
}
