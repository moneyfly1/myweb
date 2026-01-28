package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type beijingTimeWriter struct {
	writer io.Writer
}

func (w *beijingTimeWriter) Write(p []byte) (n int, err error) {
	beijingTime := GetBeijingTime().Format("2006/01/02 15:04:05")
	timestamp := fmt.Sprintf("[%s] ", beijingTime)
	_, err = w.writer.Write([]byte(timestamp))
	if err != nil {
		return 0, err
	}
	return w.writer.Write(p)
}

type Logger struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	warnLog  *log.Logger
}

var AppLogger *Logger

func InitLogger(logDir string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	infoFile, err := os.OpenFile(filepath.Join(logDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	errorFile, err := os.OpenFile(filepath.Join(logDir, "error.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	infoWriter := &beijingTimeWriter{writer: infoFile}
	errorWriter := &beijingTimeWriter{writer: errorFile}
	warnWriter := &beijingTimeWriter{writer: os.Stdout}

	AppLogger = &Logger{
		infoLog:  log.New(infoWriter, "[INFO] ", log.Lshortfile),
		errorLog: log.New(errorWriter, "[ERROR] ", log.Lshortfile),
		warnLog:  log.New(warnWriter, "[WARN] ", 0),
	}

	return nil
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l != nil && l.infoLog != nil {
		l.infoLog.Printf(format, v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l != nil && l.errorLog != nil {
		l.errorLog.Printf(format, v...)
	}
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if l != nil && l.warnLog != nil {
		l.warnLog.Printf(format, v...)
	}
}

func LogUserActivity(userID uint, activityType, description string) {
	if AppLogger != nil {
		AppLogger.Info("用户活动: user_id=%d, type=%s, description=%s", userID, activityType, description)
	}
}

func LogAudit(userID uint, actionType, resourceType string, resourceID uint, description string) {
	if AppLogger != nil {
		AppLogger.Info("审计日志: user_id=%d, action=%s, resource=%s:%d, description=%s",
			userID, actionType, resourceType, resourceID, description)
	}
}

func LogInfo(format string, v ...interface{}) {
	if AppLogger != nil {
		AppLogger.Info(format, v...)
	} else {
		log.Printf("[INFO] "+format, v...)
	}
}

func LogWarn(format string, v ...interface{}) {
	if AppLogger != nil {
		AppLogger.Warn(format, v...)
	} else {
		log.Printf("[WARN] "+format, v...)
	}
}

func LogErrorMsg(format string, v ...interface{}) {
	if AppLogger != nil {
		AppLogger.Error(format, v...)
	} else {
		log.Printf("[ERROR] "+format, v...)
	}
}
