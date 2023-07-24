package logging

import (
	"log"
	"os"
)

// i use a custom logger type with an interface so i can keep track of
// of as many different types and levels of loggers with a simple interface

type LoggerLevel int

type Logger interface {
	AddLogger(level LoggerLevel, logger *log.Logger) *log.Logger
	GetLogger(level LoggerLevel) *log.Logger
	RemoveLogger(level LoggerLevel)
	WriteToLogger(level LoggerLevel, msg string, err ...error)
}

const (
	DEFAULT LoggerLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

type CustomLogger struct {
	loggerMap map[LoggerLevel]*log.Logger
}

func NewLogger() *CustomLogger {
	cm := &CustomLogger{loggerMap: make(map[LoggerLevel]*log.Logger)}
	l := log.New(os.Stderr, "DEFAULT: ", log.Ltime)
	cm.loggerMap[DEFAULT] = l
	return cm
}

// will return the needed logger for the level given(will default to DEFAULT level)
func (cm *CustomLogger) GetLogger(level LoggerLevel) *log.Logger {
	l, ok := cm.loggerMap[level]
	if !ok {
		return cm.loggerMap[DEFAULT]
	}
	return l
}

// will add the new logger to the custom loggers map and return the logger
// (if logger already exists will just return it)
func (cm *CustomLogger) AddLogger(level LoggerLevel, logger *log.Logger) *log.Logger {
	l, ok := cm.loggerMap[level]
	if ok {
		return l
	}
	cm.loggerMap[level] = logger
	return cm.loggerMap[level]
}

// remove a logger from custom loggers
func (cm *CustomLogger) RemoveLogger(level LoggerLevel) {
	delete(cm.loggerMap, level)
}

// will write to the logger that mathces the level given, will default to
// DEFAULT level if  given logger does not exist
// will take a optional error parameter, (max of one error)
// the error of this fucnion will be writen to the default logger
func (cm *CustomLogger) WriteToLogger(level LoggerLevel, msg string, err ...error) {
	log := cm.GetLogger(level)
	errSize := len(err)
	if errSize > 1 {
		cm.GetLogger(DEFAULT).Println("to many errors passed to WriteToLogger, want=1, got=", errSize)
	}
	if errSize == 1 {
		log.Println(msg, err[0])
	}
	log.Println(msg)

}
