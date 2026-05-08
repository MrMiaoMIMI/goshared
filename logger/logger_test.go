package logger

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureWritesJSONLog(t *testing.T) {
	resetGlobalForTest(t)

	logPath := filepath.Join(t.TempDir(), "app.log")
	if err := Configure(Config{
		Level:       LevelDebug,
		Encoding:    EncodingJSON,
		OutputPaths: []string{logPath},
	}); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	Debug(context.Background(), "debug message",
		String("service", "order"),
		Int("count", 2),
	)

	entry := readLastLogEntry(t, logPath)
	if entry["level"] != LevelDebug {
		t.Fatalf("unexpected level: %#v", entry["level"])
	}
	if entry["msg"] != "debug message" {
		t.Fatalf("unexpected message: %#v", entry["msg"])
	}
	if entry["service"] != "order" {
		t.Fatalf("unexpected service field: %#v", entry["service"])
	}
	if entry["count"] != float64(2) {
		t.Fatalf("unexpected count field: %#v", entry["count"])
	}
}

func TestInitCanReconfigureAfterLazyDefault(t *testing.T) {
	resetGlobalForTest(t)

	Info(context.Background(), "before explicit init")

	logPath := filepath.Join(t.TempDir(), "app.log")
	Init(Config{
		Level:       LevelInfo,
		Encoding:    EncodingJSON,
		OutputPaths: []string{logPath},
	})
	Info(context.Background(), "after explicit init")

	entry := readLastLogEntry(t, logPath)
	if entry["msg"] != "after explicit init" {
		t.Fatalf("expected explicit init logger to be used, got %#v", entry["msg"])
	}
}

func TestConfigureRejectsInvalidConfigWithoutReplacingCurrentLogger(t *testing.T) {
	resetGlobalForTest(t)

	logPath := filepath.Join(t.TempDir(), "app.log")
	if err := Configure(Config{
		Level:       LevelInfo,
		Encoding:    EncodingJSON,
		OutputPaths: []string{logPath},
	}); err != nil {
		t.Fatalf("Configure valid: %v", err)
	}

	if err := Configure(Config{Level: "verbose"}); err == nil || !strings.Contains(err.Error(), "invalid logger level") {
		t.Fatalf("expected invalid level error, got %v", err)
	}
	if err := Configure(Config{Encoding: "xml"}); err == nil || !strings.Contains(err.Error(), "invalid logger encoding") {
		t.Fatalf("expected invalid encoding error, got %v", err)
	}
	if err := Configure(Config{OutputPaths: []string{" "}}); err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("expected empty output path error, got %v", err)
	}
	if err := Configure(Config{OutputPaths: []string{filepath.Join(t.TempDir(), "missing", "app.log")}}); err == nil {
		t.Fatalf("expected bad output path error")
	}

	Info(context.Background(), "still current")
	entry := readLastLogEntry(t, logPath)
	if entry["msg"] != "still current" {
		t.Fatalf("invalid config should not replace current logger, got %#v", entry["msg"])
	}
}

func TestSetTraceIDAndWithFieldsShareContextLogger(t *testing.T) {
	resetGlobalForTest(t)

	logPath := filepath.Join(t.TempDir(), "app.log")
	if err := Configure(Config{
		Level:       LevelInfo,
		Encoding:    EncodingJSON,
		OutputPaths: []string{logPath},
	}); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	scoped := WithFields(String("component", "worker"))
	ctx := SetTraceID(context.Background(), "trace-123")
	scoped.Info(ctx, "work done", String("job", "sync"))

	entry := readLastLogEntry(t, logPath)
	if entry[TraceIDField] != "trace-123" {
		t.Fatalf("expected trace_id from context, got %#v", entry[TraceIDField])
	}
	if entry["component"] != "worker" {
		t.Fatalf("expected scoped field, got %#v", entry["component"])
	}
	if entry["job"] != "sync" {
		t.Fatalf("expected call field, got %#v", entry["job"])
	}
}

func TestSetTraceIDHandlesNilContext(t *testing.T) {
	resetGlobalForTest(t)

	ctx := SetTraceID(nil, "trace-nil")
	if got := GetTraceID(ctx); got != "trace-nil" {
		t.Fatalf("unexpected trace id: %q", got)
	}
}

func resetGlobalForTest(t *testing.T) {
	t.Helper()
	clearGlobalLogger()
	t.Cleanup(clearGlobalLogger)
}

func clearGlobalLogger() {
	globalMu.Lock()
	logger := globalLogger
	closer := globalCloser
	globalLogger = nil
	globalCloser = nil
	globalMu.Unlock()

	_ = syncLogger(logger)
	if closer != nil {
		closer()
	}
}

func readLastLogEntry(t *testing.T, path string) map[string]any {
	t.Helper()
	if err := Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[len(lines)-1] == "" {
		t.Fatalf("expected log entry, got %q", string(data))
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("Unmarshal log entry %q: %v", lines[len(lines)-1], err)
	}
	return entry
}
