package log

import (
	"fmt"
	base "log"
	"os"
	"runtime"
)

var isDebug bool
var debug *base.Logger
var err *base.Logger
var info *base.Logger

func init() {
	debug = base.New(os.Stdout, "[DEBUG] ", base.LstdFlags)
	err = base.New(os.Stdout, "[ERROR] ", base.LstdFlags)
	info = base.New(os.Stdout, "[INFO]  ", base.LstdFlags)
}

func SetDebug(debug bool) {
	isDebug = debug
}

func Debug(format string, v ...interface{}) {
	if isDebug {
		return
	}

	_, file, line, ok := runtime.Caller(1)
	var fileInfo string
	if ok {
		fileInfo = fmt.Sprintf("%s(%d) ", file, line)
	} else {
		fileInfo = ""
	}

	debug.Printf(fileInfo+format, v...)
}

func Error(format string, v ...interface{}) string {
	_, file, line, ok := runtime.Caller(1)
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
