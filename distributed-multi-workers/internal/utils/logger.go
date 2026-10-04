package utils

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

func NewLogger(prefix string) *Logger {
	return &Logger{
		log.New(os.Stdout, prefix+" ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile),
	}
}

func (l *Logger) Info(format string, v ...any) {
	l.Printf("[INFO] "+format, v...)
}

func (l *Logger) Debug(format string, v ...any) {
	l.Printf("[DEBUG] "+format, v...)
}

func (l *Logger) Error(format string, v ...any) {
	l.Printf("[ERROR] "+format, v...)
}
