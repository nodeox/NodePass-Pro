package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	logger, err := New(Config{
		Level:  "info",
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}
	if logger.zap == nil {
		t.Error("Expected non-nil zap logger")
	}
	if logger.sugar == nil {
		t.Error("Expected non-nil sugar logger")
	}
}

func TestNewWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(Config{
		Level:      "debug",
		OutputPath: logPath,
		Prefix:     "test",
	})
	if err != nil {
		t.Fatalf("New() with file failed: %v", err)
	}
	defer logger.Close()

	// 写入日志
	logger.Info("test message", "key", "value")

	// 验证文件存在
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestLogLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(Config{
		Level:      "debug",
		OutputPath: logPath,
		Prefix:     "test",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// 测试各个日志级别
	logger.Debug("debug message", "level", "debug")
	logger.Info("info message", "level", "info")
	logger.Warn("warn message", "level", "warn")
	logger.Error("error message", "level", "error")

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// 验证所有级别的日志都被记录
	if !strings.Contains(logContent, "debug message") {
		t.Error("Debug message not found in log")
	}
	if !strings.Contains(logContent, "info message") {
		t.Error("Info message not found in log")
	}
	if !strings.Contains(logContent, "warn message") {
		t.Error("Warn message not found in log")
	}
	if !strings.Contains(logContent, "error message") {
		t.Error("Error message not found in log")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// 设置为 warn 级别
	logger, err := New(Config{
		Level:      "warn",
		OutputPath: logPath,
		Prefix:     "test",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Debug 和 Info 不应该被记录
	if strings.Contains(logContent, "debug message") {
		t.Error("Debug message should not be logged at warn level")
	}
	if strings.Contains(logContent, "info message") {
		t.Error("Info message should not be logged at warn level")
	}

	// Warn 和 Error 应该被记录
	if !strings.Contains(logContent, "warn message") {
		t.Error("Warn message should be logged")
	}
	if !strings.Contains(logContent, "error message") {
		t.Error("Error message should be logged")
	}
}

func TestPrintf(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(Config{
		Level:      "info",
		OutputPath: logPath,
		Prefix:     "test",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// 测试 Printf（兼容标准 log.Logger）
	logger.Printf("[INFO] Test message: %s=%d", "count", 42)

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "Test message") {
		t.Error("Printf message not found in log")
	}
}

func TestWith(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := New(Config{
		Level:      "info",
		OutputPath: logPath,
		Prefix:     "test",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// 创建带有额外字段的 logger
	childLogger := logger.With("component", "test-component", "version", "1.0")
	childLogger.Info("test message")

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "component") {
		t.Error("Component field not found in log")
	}
	if !strings.Contains(logContent, "test-component") {
		t.Error("Component value not found in log")
	}
}

func TestNilLogger(t *testing.T) {
	var logger *Logger

	// 所有方法应该安全处理 nil
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
	logger.Printf("test")
	_ = logger.Close()
	_ = logger.With("key", "value")
	_ = logger.Zap()
	_ = logger.Sugar()

	// 如果没有 panic，测试通过
}

func TestDefaultLevel(t *testing.T) {
	logger, err := New(Config{
		Prefix: "test",
	})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer logger.Close()

	// 默认级别应该是 info
	logger.Info("info message")
	// 不应该 panic
}

func TestInvalidOutputPath(t *testing.T) {
	// 尝试写入不存在的目录（但会自动创建）
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "subdir", "test.log")

	logger, err := New(Config{
		Level:      "info",
		OutputPath: logPath,
		Prefix:     "test",
	})
	if err != nil {
		t.Fatalf("New() should create parent directories: %v", err)
	}
	defer logger.Close()

	logger.Info("test message")

	// 验证文件被创建
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created in subdirectory")
	}
}
