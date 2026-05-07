package env

import (
	"testing"
	"time"
)

func TestGetTyped(t *testing.T) {
	t.Setenv("UTIL_ENV_INT", "12")
	t.Setenv("UTIL_ENV_BOOL", "on")

	if got := Get[int]("UTIL_ENV_INT", 0); got != 12 {
		t.Fatalf("Get[int] = %d, want 12", got)
	}
	if got := Get[bool]("UTIL_ENV_BOOL", false); !got {
		t.Fatalf("Get[bool] = false, want true")
	}
	if got := Get[int]("UTIL_ENV_MISSING", 9); got != 9 {
		t.Fatalf("missing fallback = %d, want 9", got)
	}
}

func TestDurationAndSlice(t *testing.T) {
	t.Setenv("UTIL_ENV_DURATION", "150ms")
	t.Setenv("UTIL_ENV_SLICE", "a, b,,c")

	if got := Duration("UTIL_ENV_DURATION", time.Second); got != 150*time.Millisecond {
		t.Fatalf("Duration = %v, want 150ms", got)
	}

	got := Slice("UTIL_ENV_SLICE", ",", nil)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("Slice len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Slice[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
