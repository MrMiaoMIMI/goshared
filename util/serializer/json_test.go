package serializer

import (
	"encoding/json"
	"testing"
)

type sample struct {
	Name string         `json:"name"`
	Meta map[string]int `json:"meta"`
}

func TestStringAndUnmarshal(t *testing.T) {
	raw, err := String(sample{Name: "svc", Meta: map[string]int{"a": 1}})
	if err != nil {
		t.Fatalf("String: %v", err)
	}
	if !Valid([]byte(raw)) {
		t.Fatalf("String produced invalid JSON: %s", raw)
	}

	var got sample
	if err := Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Name != "svc" || got.Meta["a"] != 1 {
		t.Fatalf("got = %#v", got)
	}
}

func TestCloneDeepCopies(t *testing.T) {
	src := sample{Name: "svc", Meta: map[string]int{"a": 1}}
	got, err := Clone(src)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}

	got.Meta["a"] = 2
	if src.Meta["a"] != 1 {
		t.Fatalf("Clone did not deep copy nested map")
	}
}

func TestUnmarshalUseNumber(t *testing.T) {
	var got map[string]any
	if err := UnmarshalUseNumber([]byte(`{"id":123}`), &got); err != nil {
		t.Fatalf("UnmarshalUseNumber: %v", err)
	}
	if _, ok := got["id"].(json.Number); !ok {
		t.Fatalf("id type = %T, want json.Number", got["id"])
	}
}
