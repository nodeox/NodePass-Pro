package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 定义结构化日志接口。
type Logger struct {
	zap    *zap.Logger
	sugar  *zap.SugaredLogger
	closer io.Closer
}

// Config 定义日志配置。
type Config struct {
	Level      string // debug, info, warn, error
	OutputPath string // 日志文件路径（可选）
	Prefix     string // 日志前缀
}

// New 创建新的结构化日志器。
func New(cfg Config) (*Logger, error) {
	if cfg.Level == "" {
		cfg.Level = "info"
	}

	// 解析日志级别
	level := zapcore.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	// 配置编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 配置输出
	var cores []zapcore.Core
	var closer io.Closer

	// 控制台输出
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		level,
	)
	cores = append(cores, consoleCore)

	// 文件输出（可选）
	if cfg.OutputPath != "" {
		// 确保目录存在
		dir := filepath.Dir(cfg.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(cfg.OutputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		closer = file

		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		fileCore := zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(file),
			level,
		)
		cores = append(cores, fileCore)
	}

	// 创建 logger
	core := zapcore.NewTee(cores...)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	if cfg.Prefix != "" {
		zapLogger = zapLogger.Named(cfg.Prefix)
	}

	return &Logger{
		zap:    zapLogger,
		sugar:  zapLogger.Sugar(),
		closer: closer,
	}, nil
}

// Close 关闭日志器并释放资源。
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	_ = l.zap.Sync()
	if l.closer != nil {
		return l.closer.Close()
	}
	return nil
}

// Debug 记录调试级别日志。
func (l *Logger) Debug(msg string, fields ...interface{}) {
	if l == nil || l.sugar == nil {
		return
	}
	l.sugar.Debugw(msg, fields...)
}

// Info 记录信息级别日志。
func (l *Logger) Info(msg string, fields ...interface{}) {
	if l == nil || l.sugar == nil {
		return
	}
	l.sugar.Infow(msg, fields...)
}

// Warn 记录警告级别日志。
func (l *Logger) Warn(msg string, fields ...interface{}) {
	if l == nil || l.sugar == nil {
		return
	}
	l.sugar.Warnw(msg, fields...)
}

// Error 记录错误级别日志。
func (l *Logger) Error(msg string, fields ...interface{}) {
	if l == nil || l.sugar == nil {
		return
	}
	l.sugar.Errorw(msg, fields...)
}

// Printf 提供与标准 log.Logger 兼容的接口。
func (l *Logger) Printf(format string, args ...interface{}) {
	if l == nil || l.sugar == nil {
		return
	}
	l.sugar.Infof(format, args...)
}

// With 添加结构化字段。
func (l *Logger) With(fields ...interface{}) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{
		zap:    l.zap,
		sugar:  l.sugar.With(fields...),
		closer: l.closer,
	}
}

// Zap 返回底层的 zap.Logger（用于高级用法）。
func (l *Logger) Zap() *zap.Logger {
	if l == nil {
		return nil
	}
	return l.zap
}

// Sugar 返回底层的 zap.SugaredLogger（用于高级用法）。
func (l *Logger) Sugar() *zap.SugaredLogger {
	if l == nil {
		return nil
	}
	return l.sugar
}

// StdLogger 返回一个兼容标准库 log.Logger 的适配器。
func (l *Logger) StdLogger() *log.Logger {
	if l == nil {
		return log.New(os.Stdout, "", log.LstdFlags)
	}
	// 创建一个 writer 适配器
	writer := &zapWriter{logger: l}
	return log.New(writer, "", 0) // 不需要前缀和标志，zap 已经处理
}

// zapWriter 实现 io.Writer 接口，将写入转发到 zap logger。
type zapWriter struct {
	logger *Logger
}

func (w *zapWriter) Write(p []byte) (n int, err error) {
	if w.logger != nil && w.logger.sugar != nil {
		w.logger.sugar.Info(string(p))
	}
	return len(p), nil
}
