package log

import (
	base "log"
	"os"
)

var err *base.Logger
var info *base.Logger

func init() {
	err = base.New(os.Stdout, "[ERROR] ", base.LstdFlags)
	info = base.New(os.Stdout, "[INFO]  ", base.LstdFlags)
}

func Error(format string, v ...interface{}) {
	err.Printf(format, v...)
}

func Info(format string, v ...interface{}) {
	info.Printf(format, v...)
}

func Panic(format string, v ...interface{}) {
	err.Panicf(format, v...)
}
