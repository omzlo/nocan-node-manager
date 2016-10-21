package log

import (
//    "fmt"
    "log"
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
