package interfaces

import "log"

// Logger 定义结构化日志接口。
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	With(fields ...interface{}) Logger
	Close() error
	StdLogger() *log.Logger
}
