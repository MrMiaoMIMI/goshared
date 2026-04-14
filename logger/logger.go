package logger

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const (
	TraceIDKey  contextKey = "trace_id"
	loggerKey   contextKey = "__logger"
)

var (
	globalLogger *zap.Logger
	once         sync.Once
)

// Config logger 配置
type Config struct {
	Level       string // debug, info, warn, error
	Development bool
	Encoding    string // json 或 console
	OutputPaths []string
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Level:       "debug",
		Development: false,
		Encoding:    "json",
	}
}

// Init 初始化全局 logger（仅执行一次）
func Init(cfg Config) {
	once.Do(func() {
		globalLogger = newLogger(cfg)
	})
}

func newLogger(cfg Config) *zap.Logger {
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

	var encoder zapcore.Encoder
	if cfg.Encoding == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	writers := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}
	for _, path := range cfg.OutputPaths {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			writers = append(writers, zapcore.AddSync(f))
		}
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		level,
	)

	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	}

	if cfg.Development {
		opts = append(opts, zap.Development())
	}

	return zap.New(core, opts...)
}

// getLogger 获取全局 logger，未初始化时使用默认配置。
// 使用 sync.Once（通过 Init）保证线程安全且仅初始化一次。
func getLogger() *zap.Logger {
	Init(DefaultConfig())
	return globalLogger
}

// Sync 刷新所有缓冲的日志。应在程序退出前调用。
func Sync() error {
	return getLogger().Sync()
}

func extractTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// loggerFromCtx returns the context-scoped logger (with trace_id pre-baked)
// or falls back to the global logger. No per-call allocation is needed.
func loggerFromCtx(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
			return l
		}
	}
	return getLogger()
}

// IsDebugEnabled returns true if debug level logging is enabled.
func IsDebugEnabled() bool {
	return getLogger().Core().Enabled(zapcore.DebugLevel)
}

// IsInfoEnabled returns true if info level logging is enabled.
func IsInfoEnabled() bool {
	return getLogger().Core().Enabled(zapcore.InfoLevel)
}

// Debug 打印 debug 级别日志
func Debug(ctx context.Context, msg string, fields ...Field) {
	loggerFromCtx(ctx).Debug(msg, toZapFields(fields)...)
}

// Info 打印 info 级别日志
func Info(ctx context.Context, msg string, fields ...Field) {
	loggerFromCtx(ctx).Info(msg, toZapFields(fields)...)
}

// Warn 打印 warn 级别日志
func Warn(ctx context.Context, msg string, fields ...Field) {
	loggerFromCtx(ctx).Warn(msg, toZapFields(fields)...)
}

// Error 打印 error 级别日志
func Error(ctx context.Context, msg string, fields ...Field) {
	loggerFromCtx(ctx).Error(msg, toZapFields(fields)...)
}

// Fatal 打印 fatal 级别日志并退出程序
func Fatal(ctx context.Context, msg string, fields ...Field) {
	loggerFromCtx(ctx).Fatal(msg, toZapFields(fields)...)
}

// Panic 打印日志并 panic
func Panic(ctx context.Context, msg string, fields ...Field) {
	loggerFromCtx(ctx).Panic(msg, toZapFields(fields)...)
}

// DPanic 在 Development 模式下 panic，在 Production 模式下仅记录 error。
// 用于「不应该发生但不至于崩溃生产服务」的场景。
func DPanic(ctx context.Context, msg string, fields ...Field) {
	loggerFromCtx(ctx).DPanic(msg, toZapFields(fields)...)
}

// SugaredLogger 包装了 zap.SugaredLogger，用于格式化日志
type SugaredLogger struct {
	sugar *zap.SugaredLogger
}

func (s *SugaredLogger) Debugf(template string, args ...interface{}) {
	s.sugar.Debugf(template, args...)
}

func (s *SugaredLogger) Infof(template string, args ...interface{}) {
	s.sugar.Infof(template, args...)
}

func (s *SugaredLogger) Warnf(template string, args ...interface{}) {
	s.sugar.Warnf(template, args...)
}

func (s *SugaredLogger) Errorf(template string, args ...interface{}) {
	s.sugar.Errorf(template, args...)
}

// WithContext 返回带有 trace_id 的 SugaredLogger。
// 如果 ctx 已通过 SetTraceID 注入了 logger，则直接复用，无需额外分配。
func WithContext(ctx context.Context) *SugaredLogger {
	return &SugaredLogger{sugar: loggerFromCtx(ctx).Sugar()}
}

// Logger 是带有预设字段的日志实例，适合在请求/任务作用域中复用。
type Logger struct {
	base *zap.Logger
}

// WithFields 创建带有预设字段的 Logger
func WithFields(fields ...Field) *Logger {
	return &Logger{base: getLogger().With(toZapFields(fields)...)}
}

func (l *Logger) Debug(_ context.Context, msg string, fields ...Field) {
	l.base.Debug(msg, toZapFields(fields)...)
}

func (l *Logger) Info(_ context.Context, msg string, fields ...Field) {
	l.base.Info(msg, toZapFields(fields)...)
}

func (l *Logger) Warn(_ context.Context, msg string, fields ...Field) {
	l.base.Warn(msg, toZapFields(fields)...)
}

func (l *Logger) Error(_ context.Context, msg string, fields ...Field) {
	l.base.Error(msg, toZapFields(fields)...)
}

// SetTraceID 将 trace_id 设置到 context 中，同时创建一个预绑定了 trace_id 的 logger
// 并存入 context。后续所有通过该 ctx 的日志调用都会自动携带 trace_id，
// 且不会产生任何额外的 slice 分配或字段拷贝。
//
// 典型用法（在中间件/请求入口处调用一次）：
//
//	ctx = logger.SetTraceID(ctx, requestID)
//	// 后续所有日志自动携带 trace_id
//	logger.Info(ctx, "processing request", logger.String("path", "/api/users"))
func SetTraceID(ctx context.Context, traceID string) context.Context {
	ctx = context.WithValue(ctx, TraceIDKey, traceID)
	derived := getLogger().With(zap.String("trace_id", traceID))
	ctx = context.WithValue(ctx, loggerKey, derived)
	return ctx
}

// GetTraceID 从 context 中获取 trace_id
func GetTraceID(ctx context.Context) string {
	return extractTraceID(ctx)
}
