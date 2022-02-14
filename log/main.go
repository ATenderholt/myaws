package log

import (
	"fmt"
	base "log"
	"myaws/config"
	"os"
)

var debug *base.Logger
var err *base.Logger
var info *base.Logger

func init() {
	if config.GetSettings().IsDebug() {
		debug = base.New(os.Stdout, "[DEBUG] ", base.LstdFlags)
	}

	err = base.New(os.Stdout, "[ERROR] ", base.LstdFlags)
	info = base.New(os.Stdout, "[INFO]  ", base.LstdFlags)
}

func Debug(format string, v ...interface{}) {
	if config.GetSettings().IsDebug() {
		debug.Printf(format, v...)
	}
}

func Error(format string, v ...interface{}) string {
	msg := fmt.Sprintf(format, v...)
	err.Printf(msg)
	return msg
}

func Info(format string, v ...interface{}) {
	info.Printf(format, v...)
}

func Panic(format string, v ...interface{}) {
	err.Panicf(format, v...)
}
