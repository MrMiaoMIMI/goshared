package convutil

import "testing"

type customInt int
type customString string

func TestToScalar(t *testing.T) {
	n, err := To[customInt]("42")
	if err != nil {
		t.Fatalf("To[customInt]: %v", err)
	}
	if n != 42 {
		t.Fatalf("To[customInt] = %d, want 42", n)
	}

	b, err := To[bool]("yes")
	if err != nil {
		t.Fatalf("To[bool]: %v", err)
	}
	if !b {
		t.Fatalf("To[bool] = false, want true")
	}

	s, err := To[customString]([]byte("hello"))
	if err != nil {
		t.Fatalf("To[customString]: %v", err)
	}
	if s != "hello" {
		t.Fatalf("To[customString] = %q, want hello", s)
	}
}

func TestToOverflow(t *testing.T) {
	if _, err := To[int8](128); err == nil {
		t.Fatalf("To[int8](128) expected overflow error")
	}
	if _, err := To[uint](-1); err == nil {
		t.Fatalf("To[uint](-1) expected negative conversion error")
	}
}

func TestToOr(t *testing.T) {
	if got := ToOr[int]("bad", 7); got != 7 {
		t.Fatalf("ToOr fallback = %d, want 7", got)
	}
}
