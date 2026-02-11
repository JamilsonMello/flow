package flow

import (
	"fmt"
	"log"
)

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...interface{}) {}
func (noopLogger) Info(string, ...interface{})  {}
func (noopLogger) Error(string, ...interface{}) {}

type stdLogger struct{}

func (stdLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[FLOW DEBUG] "+msg, args...)
}

func (stdLogger) Info(msg string, args ...interface{}) {
	log.Printf("[FLOW INFO] "+msg, args...)
}

func (stdLogger) Error(msg string, args ...interface{}) {
	log.Printf("[FLOW ERROR] "+msg, args...)
}

func NewStdLogger() Logger {
	return stdLogger{}
}

type fmtLogger struct{}

func (fmtLogger) Debug(msg string, args ...interface{}) {
	fmt.Printf("[FLOW DEBUG] "+msg+"\n", args...)
}

func (fmtLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[FLOW INFO] "+msg+"\n", args...)
}

func (fmtLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("[FLOW ERROR] "+msg+"\n", args...)
}

func NewFmtLogger() Logger {
	return fmtLogger{}
}
