package logger

import "log"

func Info(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

func Error(format string, args ...interface{}) {
	log.Printf("ERROR: "+format, args...)
}
