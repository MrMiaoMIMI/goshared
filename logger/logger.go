package logger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const (
	TraceIDKey contextKey = "trace_id"
	loggerKey  contextKey = "__logger"
)

const (
	// TraceIDField 是日志中记录请求链路 ID 的字段名。
	TraceIDField = "trace_id"

	// LevelDebug 输出 debug 及以上级别日志。
	LevelDebug = "debug"
	// LevelInfo 输出 info 及以上级别日志。
	LevelInfo = "info"
	// LevelWarn 输出 warn 及以上级别日志。
	LevelWarn = "warn"
	// LevelError 输出 error 及以上级别日志。
	LevelError = "error"

	// EncodingJSON 以 JSON 行格式输出日志，适合服务端采集。
	EncodingJSON = "json"
	// EncodingConsole 以更适合本地阅读的文本格式输出日志。
	EncodingConsole = "console"

	// OutputStdout 将日志输出到标准输出。
	OutputStdout = "stdout"
	// OutputStderr 将日志输出到标准错误。
	OutputStderr = "stderr"
)

var (
	globalLogger *zap.Logger
	globalCloser func()
	globalMu     sync.RWMutex
)

// Config 是全局 logger 的启动配置.
//
// 零值可直接使用，默认行为为：
//   - level: info
//   - encoding: json
//   - output_paths: stdout
type Config struct {
	// Level 是最低输出级别，支持 debug、info、warn、error、dpanic、panic、fatal。
	// 为空时使用 info。
	Level string `json:"level" yaml:"level"`
	// Development 开启开发模式。开启后，DPanic 会触发 panic。
	Development bool `json:"development" yaml:"development"`
	// Encoding 是日志输出格式，支持 json 和 console。为空时使用 json。
	Encoding string `json:"encoding" yaml:"encoding"`
	// OutputPaths 是日志输出目标，支持 stdout、stderr 或文件路径。
	// 为空时输出到 stdout。
	OutputPaths []string `json:"output_paths" yaml:"output_paths"`
}

// DefaultConfig 返回适合本地调试的默认配置：debug 级别、JSON 格式、输出到 stdout。
func DefaultConfig() Config {
	return Config{
		Level:       LevelDebug,
		Development: false,
		Encoding:    EncodingJSON,
		OutputPaths: []string{OutputStdout},
	}
}

// Init 初始化全局 logger，配置非法时会 panic。
//
// 服务启动阶段更推荐使用 Configure，因为 Configure 会返回错误，便于将
// 配置问题作为启动失败处理。
func Init(cfg Config) {
	if err := Configure(cfg); err != nil {
		panic(err)
	}
}

// Configure 创建并设置全局 logger。
//
// 调用成功后，后续 Debug/Info/Warn/Error 等包级方法都会使用新的配置。
// 如果服务在读取配置前已经输出过日志，也可以再次调用 Configure 覆盖默认 logger。
func Configure(cfg Config) error {
	logger, closer, err := newLogger(cfg)
	if err != nil {
		return err
	}

	globalMu.Lock()
	previousLogger := globalLogger
	previousCloser := globalCloser
	globalLogger = logger
	globalCloser = closer
	globalMu.Unlock()

	_ = syncLogger(previousLogger)
	if previousCloser != nil {
		previousCloser()
	}
	return nil
}

func newLogger(cfg Config) (*zap.Logger, func(), error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, nil, err
	}
	encoding, err := parseEncoding(cfg.Encoding)
	if err != nil {
		return nil, nil, err
	}
	outputPaths, err := normalizeOutputPaths(cfg.OutputPaths)
	if err != nil {
		return nil, nil, err
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
	if encoding == EncodingConsole {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	writer, closeFn, err := openOutputPaths(outputPaths)
	if err != nil {
		return nil, nil, fmt.Errorf("logger output paths: %w", err)
	}

	core := zapcore.NewCore(
		encoder,
		writer,
		level,
	)

	opts := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	}

	if cfg.Development {
		opts = append(opts, zap.Development())
	}

	return zap.New(core, opts...), closeFn, nil
}

func parseLevel(raw string) (zapcore.Level, error) {
	levelText := strings.TrimSpace(strings.ToLower(raw))
	if levelText == "" {
		levelText = LevelInfo
	}

	var level zapcore.Level
	if err := level.Set(levelText); err != nil {
		return zapcore.InfoLevel, fmt.Errorf("invalid logger level %q: %w", raw, err)
	}
	return level, nil
}

func parseEncoding(raw string) (string, error) {
	encoding := strings.TrimSpace(strings.ToLower(raw))
	if encoding == "" {
		return EncodingJSON, nil
	}
	switch encoding {
	case EncodingJSON, EncodingConsole:
		return encoding, nil
	default:
		return "", fmt.Errorf("invalid logger encoding %q: supported values are %q and %q", raw, EncodingJSON, EncodingConsole)
	}
}

func normalizeOutputPaths(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return []string{OutputStdout}, nil
	}

	normalized := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, fmt.Errorf("invalid logger output path: empty path")
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	return normalized, nil
}

func openOutputPaths(paths []string) (zapcore.WriteSyncer, func(), error) {
	writers := make([]zapcore.WriteSyncer, 0, len(paths))
	files := make([]*os.File, 0, len(paths))

	for _, path := range paths {
		switch path {
		case OutputStdout:
			writers = append(writers, zapcore.Lock(os.Stdout))
		case OutputStderr:
			writers = append(writers, zapcore.Lock(os.Stderr))
		default:
			file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				closeFiles(files)
				return nil, nil, fmt.Errorf("open %q: %w", path, err)
			}
			files = append(files, file)
			writers = append(writers, zapcore.Lock(file))
		}
	}

	closeFn := func() {
		closeFiles(files)
	}
	if len(writers) == 1 {
		return writers[0], closeFn, nil
	}
	return zapcore.NewMultiWriteSyncer(writers...), closeFn, nil
}

func closeFiles(files []*os.File) {
	for _, file := range files {
		_ = file.Close()
	}
}

// getLogger 获取全局 logger，未初始化时使用 DefaultConfig。
func getLogger() *zap.Logger {
	globalMu.RLock()
	logger := globalLogger
	globalMu.RUnlock()
	if logger != nil {
		return logger
	}

	globalMu.Lock()
	defer globalMu.Unlock()
	if globalLogger == nil {
		logger, closer, err := newLogger(DefaultConfig())
		if err != nil {
			globalLogger = zap.NewNop()
		} else {
			globalLogger = logger
			globalCloser = closer
		}
	}
	return globalLogger
}

// Sync 刷新所有缓冲的日志。应在程序退出前调用。
func Sync() error {
	return syncLogger(getLogger())
}

func syncLogger(logger *zap.Logger) error {
	if logger == nil {
		return nil
	}
	err := logger.Sync()
	if err == nil || errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.EBADF) {
		return nil
	}
	return err
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

func contextLogger(ctx context.Context) (*zap.Logger, bool) {
	if ctx == nil {
		return nil, false
	}
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok && l != nil {
		return l, true
	}
	return nil, false
}

// loggerFromCtx returns the context-scoped logger (with trace_id pre-baked)
// or falls back to the global logger. No per-call allocation is needed.
func loggerFromCtx(ctx context.Context) *zap.Logger {
	if l, ok := contextLogger(ctx); ok {
		return l
	}
	return getLogger()
}

// IsDebugEnabled 判断当前配置是否会输出 debug 日志。
func IsDebugEnabled() bool {
	return getLogger().Core().Enabled(zapcore.DebugLevel)
}

// IsInfoEnabled 判断当前配置是否会输出 info 日志。
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

// SugaredLogger 提供 Debugf/Infof/Warnf/Errorf 这类格式化日志方法。
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

// WithContext 返回会自动携带 ctx 中 trace_id 的格式化 logger。
func WithContext(ctx context.Context) *SugaredLogger {
	return &SugaredLogger{sugar: loggerFromCtx(ctx).Sugar()}
}

// Logger 是带有固定字段的日志实例，适合在组件、任务或请求作用域中复用。
type Logger struct {
	base   *zap.Logger
	fields []zap.Field
}

// WithFields 创建带有固定字段的 Logger。
//
// 例如组件启动时创建一次：
//
//	workerLog := logger.WithFields(logger.String("component", "worker"))
//	workerLog.Info(ctx, "job finished")
func WithFields(fields ...Field) *Logger {
	zapFields := toZapFields(fields)
	return &Logger{
		base:   getLogger().With(zapFields...),
		fields: zapFields,
	}
}

func (l *Logger) logger(ctx context.Context) *zap.Logger {
	if l == nil {
		return loggerFromCtx(ctx)
	}
	if ctxLogger, ok := contextLogger(ctx); ok {
		if len(l.fields) == 0 {
			return ctxLogger
		}
		return ctxLogger.With(l.fields...)
	}
	if l.base == nil {
		return getLogger().With(l.fields...)
	}
	return l.base
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.logger(ctx).Debug(msg, toZapFields(fields)...)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...Field) {
	l.logger(ctx).Info(msg, toZapFields(fields)...)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.logger(ctx).Warn(msg, toZapFields(fields)...)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...Field) {
	l.logger(ctx).Error(msg, toZapFields(fields)...)
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
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID == "" {
		return ctx
	}
	ctx = context.WithValue(ctx, TraceIDKey, traceID)
	derived := getLogger().With(zap.String(TraceIDField, traceID))
	ctx = context.WithValue(ctx, loggerKey, derived)
	return ctx
}

// GetTraceID 从 context 中获取 trace_id
func GetTraceID(ctx context.Context) string {
	return extractTraceID(ctx)
}
