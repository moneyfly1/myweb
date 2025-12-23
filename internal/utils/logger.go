package utils

import (
	"log"
	"os"
	"path/filepath"
)

// Logger 日志记录器
type Logger struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	warnLog  *log.Logger
}

var AppLogger *Logger

// InitLogger 初始化日志
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

	AppLogger = &Logger{
		infoLog:  log.New(infoFile, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLog: log.New(errorFile, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLog:  log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime),
	}

	return nil
}

// Info 记录信息日志
func (l *Logger) Info(format string, v ...interface{}) {
	if l != nil && l.infoLog != nil {
		l.infoLog.Printf(format, v...)
	}
}

// Error 记录错误日志
func (l *Logger) Error(format string, v ...interface{}) {
	if l != nil && l.errorLog != nil {
		l.errorLog.Printf(format, v...)
	}
}

// Warn 记录警告日志
func (l *Logger) Warn(format string, v ...interface{}) {
	if l != nil && l.warnLog != nil {
		l.warnLog.Printf(format, v...)
	}
}

// LogUserActivity 记录用户活动
func LogUserActivity(userID uint, activityType, description string) {
	if AppLogger != nil {
		AppLogger.Info("用户活动: user_id=%d, type=%s, description=%s", userID, activityType, description)
	}
}

// LogAudit 记录审计日志（仅记录到文件）
func LogAudit(userID uint, actionType, resourceType string, resourceID uint, description string) {
	if AppLogger != nil {
		AppLogger.Info("审计日志: user_id=%d, action=%s, resource=%s:%d, description=%s",
			userID, actionType, resourceType, resourceID, description)
	}
}

// LogInfo 记录信息日志（如果 AppLogger 未初始化，则使用标准 log）
func LogInfo(format string, v ...interface{}) {
	if AppLogger != nil {
		AppLogger.Info(format, v...)
	} else {
		log.Printf("[INFO] "+format, v...)
	}
}

// LogWarn 记录警告日志（如果 AppLogger 未初始化，则使用标准 log）
func LogWarn(format string, v ...interface{}) {
	if AppLogger != nil {
		AppLogger.Warn(format, v...)
	} else {
		log.Printf("[WARN] "+format, v...)
	}
}

// LogErrorMsg 记录错误日志消息（如果 AppLogger 未初始化，则使用标准 log）
func LogErrorMsg(format string, v ...interface{}) {
	if AppLogger != nil {
		AppLogger.Error(format, v...)
	} else {
		log.Printf("[ERROR] "+format, v...)
	}
}
