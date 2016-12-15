package clog

import (
	//    "fmt"
	"log"
	"os"
	"sync"
)

var logMutex sync.Mutex
var with_colors bool = true

type LogLevel uint

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

var color_tags = [...]string{
	"\033[34mDEBUG\033[0m",
	"\033[90mINFO\033[0m",
	"\033[93mWARNING\033[0m",
	"\033[91mERROR\033[0m",
}

var plain_tags = [...]string{
	"DEBUG",
	"INFO",
	"WARNING",
	"ERROR",
}

/*
var logfile *os.File

func Init() {
	logfile,err := os.Open("nocan.log")
	if err!= nil {
		panic("Cloud not open nocan.log")
	}
}
*/

func Log(level LogLevel, format string, v ...interface{}) {
	var (
		tag string
		//id string
	)

	if with_colors {
		//tag = fmt.Sprintf("[\033[35m%s\033[0m] ", id)
		tag += color_tags[level] + " "
	} else {
		//tag = fmt.Sprintf("[%s] ", id)
		tag += plain_tags[level] + " "
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	log.Printf(tag+format, v...)
}

func Warning(format string, v ...interface{}) {
	Log(WARNING, format, v...)
}

func Error(format string, v ...interface{}) {
	Log(ERROR, format, v...)
}
func Fatal(format string, v ...interface{}) {
	Log(ERROR, format, v...)
	os.Exit(1)
}

func Info(format string, v ...interface{}) {
	Log(INFO, format, v...)
}

func Debug(format string, v ...interface{}) {
	Log(DEBUG, format, v...)
}
