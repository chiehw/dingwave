package logger

import (
	"log"
	"os"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel = INFO
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorGray    = "\033[90m"
)

func SetLevel(level Level) {
	currentLevel = level
}

func Debug(format string, v ...any) {
	if currentLevel <= DEBUG {
		log.Printf(colorGray+"[DEBUG] "+format+colorReset, v...)
	}
}

func Info(format string, v ...any) {
	if currentLevel <= INFO {
		log.Printf(colorBlue+"[INFO] "+format+colorReset, v...)
	}
}

func Warn(format string, v ...any) {
	if currentLevel <= WARN {
		log.Printf(colorYellow+"[WARN] "+format+colorReset, v...)
	}
}

func Error(format string, v ...any) {
	if currentLevel <= ERROR {
		log.Printf(colorRed+"[ERROR] "+format+colorReset, v...)
	}
}

func Fatal(format string, v ...any) {
	log.Printf(colorRed+"[FATAL] "+format+colorReset, v...)
	os.Exit(1)
}

