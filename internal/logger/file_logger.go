package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)

var fileLogger *lumberjack.Logger

func InitFileLogger() {
	logDir := "../../logs"
	os.MkdirAll(logDir, 0755)
	
	fileLogger = &lumberjack.Logger{
		Filename:   "../../logs/log.txt", 
		MaxSize:    10,             
		MaxBackups: 3,              
		MaxAge:     28,             
		Compress:   true,           
	}
}

func FileLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		
		c.Next()
		
		end := time.Now()
		latency := end.Sub(start)
		
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		
		level := getLevel(statusCode)
		
		logToFile(level, "HTTP Request", map[string]interface{}{
			"status":    statusCode,
			"method":    method,
			"path":      path,
			"query":     query,
			"ip":        clientIP,
			"latency":   latency.String(),
			"userAgent": c.Request.UserAgent(),
			"time":      end.Format(time.RFC3339),
		})
	}
}

func getLevel(status int) string {
	switch {
	case status >= 500:
		return "ERROR"
	case status >= 400:
		return "WARN"
	case status >= 300:
		return "INFO"
	case status >= 200:
		return "DEBUG"
	default:
		return "INFO"
	}
}

func logToFile(level, message string, fields map[string]interface{}) {
	if fileLogger == nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s | %s", level, timestamp, message)

	for k, v := range fields {
        if v != "" && v != nil {
            logEntry += fmt.Sprintf(" | %s=%v", k, v)
        }
    }
    
    logEntry += "\n"
    fileLogger.Write([]byte(logEntry))
}

func BusinessLog(level, message string, fields map[string]interface{}) {
	logToFile(level, message, fields)
}

func Debug(msg string, fields map[string]interface{}) {
	logToFile("DEBUG", msg, fields)
}

func Info(msg string, fields map[string]interface{}) {
	logToFile("INFO", msg, fields)
}

func Warn(msg string, fields map[string]interface{}) {
	logToFile("WARN", msg, fields)
}

func Error(msg string, fields map[string]interface{}) {
	logToFile("ERROR", msg, fields)
}