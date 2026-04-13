package logger

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 定义 context key 类型，避免冲突
type contextKey string

const (
	// TraceIDKey 用于存储 trace_id 的 context key
	TraceIDKey contextKey = "trace_id"
)

var (
	// 全局 logger 实例
	globalLogger *zap.Logger
	once         sync.Once
)

// Config logger 配置
type Config struct {
	Level       string // debug, info, warn, error
	Development bool   // 是否开发模式
	Encoding    string // json 或 console
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Level:       "debug",
		Development: false,
		Encoding:    "json",
	}
}

// Init 初始化全局 logger
func Init(cfg Config) {
	once.Do(func() {
		globalLogger = newLogger(cfg)
	})
}

// newLogger 创建新的 logger 实例
func newLogger(cfg Config) *zap.Logger {
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

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 根据配置选择编码器
	var encoder zapcore.Encoder
	if cfg.Encoding == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建 core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	// 创建 logger
	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1), // 跳过封装层
	}

	if cfg.Development {
		opts = append(opts, zap.Development())
	}

	return zap.New(core, opts...)
}

// getLogger 获取全局 logger，如果未初始化则使用默认配置
func getLogger() *zap.Logger {
	if globalLogger == nil {
		Init(DefaultConfig())
	}
	return globalLogger
}

// extractTraceID 从 context 中提取 trace_id
func extractTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// withTraceID 如果 ctx 中有 trace_id，则添加到 fields 中
func withTraceID(ctx context.Context, fields []zap.Field) []zap.Field {
	traceID := extractTraceID(ctx)
	if traceID != "" {
		newFields := make([]zap.Field, len(fields)+1)
		newFields[0] = zap.String("trace_id", traceID)
		copy(newFields[1:], fields)
		return newFields
	}
	return fields
}

// Debug 打印 debug 级别日志
func Debug(ctx context.Context, msg string, fields ...Field) {
	getLogger().Debug(msg, withTraceID(ctx, toZapFields(fields))...)
}

// Info 打印 info 级别日志
func Info(ctx context.Context, msg string, fields ...Field) {
	getLogger().Info(msg, withTraceID(ctx, toZapFields(fields))...)
}

// Warn 打印 warn 级别日志
func Warn(ctx context.Context, msg string, fields ...Field) {
	getLogger().Warn(msg, withTraceID(ctx, toZapFields(fields))...)
}

// Error 打印 error 级别日志
func Error(ctx context.Context, msg string, fields ...Field) {
	getLogger().Error(msg, withTraceID(ctx, toZapFields(fields))...)
}

// Fatal 打印 fatal 级别日志并退出程序
func Fatal(ctx context.Context, msg string, fields ...Field) {
	getLogger().Fatal(msg, withTraceID(ctx, toZapFields(fields))...)
}

// Panic 打印日志并 panic
func Panic(ctx context.Context, msg string, fields ...Field) {
	getLogger().Panic(msg, withTraceID(ctx, toZapFields(fields))...)
}

// SugaredLogger 包装了 zap.SugaredLogger，用于格式化日志
type SugaredLogger struct {
	sugar *zap.SugaredLogger
}

// Debugf 打印格式化的 debug 日志
func (s *SugaredLogger) Debugf(template string, args ...interface{}) {
	s.sugar.Debugf(template, args...)
}

// Infof 打印格式化的 info 日志
func (s *SugaredLogger) Infof(template string, args ...interface{}) {
	s.sugar.Infof(template, args...)
}

// Warnf 打印格式化的 warn 日志
func (s *SugaredLogger) Warnf(template string, args ...interface{}) {
	s.sugar.Warnf(template, args...)
}

// Errorf 打印格式化的 error 日志
func (s *SugaredLogger) Errorf(template string, args ...interface{}) {
	s.sugar.Errorf(template, args...)
}

// WithContext 返回带有 trace_id 的 SugaredLogger（用于格式化日志）
func WithContext(ctx context.Context) *SugaredLogger {
	traceID := extractTraceID(ctx)
	if traceID != "" {
		return &SugaredLogger{sugar: getLogger().With(zap.String("trace_id", traceID)).Sugar()}
	}
	return &SugaredLogger{sugar: getLogger().Sugar()}
}

// SetTraceID 将 trace_id 设置到 context 中
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 从 context 中获取 trace_id
func GetTraceID(ctx context.Context) string {
	return extractTraceID(ctx)
}
