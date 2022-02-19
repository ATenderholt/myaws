package log

import (
	"fmt"
	base "log"
	"myaws/config"
	"os"
	"runtime"
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
	if !config.GetSettings().IsDebug() {
		return
	}

	_, file, line, ok := runtime.Caller(0)
	var fileInfo string
	if ok {
		fileInfo = fmt.Sprintf("%s(%d) ", file, line)
	} else {
		fileInfo = ""
	}

	debug.Printf(fileInfo+format, v...)
}

func Error(format string, v ...interface{}) string {
	_, file, line, ok := runtime.Caller(0)
	var fileInfo string
	if ok {
		fileInfo = fmt.Sprintf("%s(%d) ", file, line)
	} else {
		fileInfo = ""
	}

	msg := fmt.Sprintf(fileInfo+format, v...)
	err.Printf(msg)
	return msg
}

func Info(format string, v ...interface{}) {
	info.Printf(format, v...)
}

func Panic(format string, v ...interface{}) {
	err.Panicf(format, v...)
}
