package utils

import (
	"os"

	"github.com/sirupsen/logrus"
)

const (
	WarnLevel  = logrus.WarnLevel
	InfoLevel  = logrus.InfoLevel
	DebugLevel = logrus.DebugLevel
	ErrorLevel = logrus.ErrorLevel
)

var (
	timeFormat = "2006-01-02 15:04:05.999999 -0700"
)

type LogStruct struct {
	log *logrus.Logger
}

type Log interface {
	SetLevel(level logrus.Level)
	SetLogFormat(format logrus.Formatter)
	Info(message string)
	Warn(message string)
	Debug(message string)
	Error(message string)

	Infof(message string, args ...interface{})
	Warnf(message string, args ...interface{})
	Debugf(message string, args ...interface{})
	Errorf(message string, args ...interface{})
}

func NewLog(format string) Log {

	var (logFormat logrus.Formatter)
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(InfoLevel)
	logFieldMap :=  logrus.FieldMap{
		logrus.FieldKeyTime:  "timestamp",
		logrus.FieldKeyLevel: "level",
		logrus.FieldKeyMsg:   "message",
		logrus.FieldKeyFunc:  "caller",
	}


	switch format {
	case "json": 
		logFormat = &logrus.JSONFormatter{
			FieldMap: logFieldMap,
			TimestampFormat: timeFormat,
		}
	case "text":
		logFormat = &logrus.TextFormatter{
			FieldMap: logFieldMap,
			TimestampFormat: timeFormat,
		}
	}

	log.SetFormatter(logFormat)

	return &LogStruct{
		log: log,
	}
}

func (l *LogStruct) SetLevel(level logrus.Level) {
	l.log.SetLevel(level)
}

func (l *LogStruct) SetLogFormat(format logrus.Formatter) {
	l.log.SetFormatter(format)
}

func (l *LogStruct) Info(message string) {
	l.log.Info(message)
}

func (l *LogStruct) Infof(message string, args ...interface{}) {
	l.log.Infof(message, args...)
}

func (l *LogStruct) Warn(message string) {
	l.log.Warn(message)
}

func (l *LogStruct) Warnf(message string, args ...interface{}) {
	l.log.Warnf(message, args...)
}

func (l *LogStruct) Debug(message string) {
	l.log.Debug(message)
}

func (l *LogStruct) Debugf(message string, args ...interface{}) {
	l.log.Debugf(message, args...)
}

func (l *LogStruct) Error(message string) {
	l.log.Error(message)
}

func (l *LogStruct) Errorf(message string, args ...interface{}) {
	l.log.Errorf(message, args...)
}
