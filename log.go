package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

const (
	LogLevelErr   = 0
	LogLevelWarn  = 1
	LogLevelInfo  = 2
	LogLevelDebug = 3
)

var (
	logi            *log.Logger
	logFile         *os.File
	defaultLogLevel = LogLevelErr
	errLevels       = []string{"error", "warn", "info", "debug"}
)

func init() {
	logi = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
}

func NewLog() error {
	var err error

	defaultLogLevel = Conf.LogLevel
	// init log
	if Conf.Log != "" {
		logFile, err = os.OpenFile(Conf.Log, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			logi.Printf("os.OpenFile(\"%s\", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644) failed (%s)", Conf.Log, err.Error())
			return err
		}

		logi = log.New(logFile, "", log.LstdFlags)
	}

	return nil
}

func CloseLog() {
	if logFile != nil {
		logFile.Close()
	}
}

func LogError(level int, format string, args ...interface{}) {
	if defaultLogLevel >= level {
		logCore(level, format, args...)
	}
}

func logCore(level int, format string, args ...interface{}) {
	var (
		file string
		line int
		ok   bool
	)

	_, file, line, ok = runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}

	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}

	file = short
	logi.Print(fmt.Sprintf("%s:%d [%s] %s", file, line, errLevels[level], fmt.Sprintf(format, args...)))
}
