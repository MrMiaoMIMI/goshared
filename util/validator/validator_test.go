package validator

import "testing"

func TestValidateGenericRules(t *testing.T) {
	err := Validate(
		Required("id", int64(10)),
		Range("score", 80.5, 0.0, 100.0),
		In("status", "enabled", "enabled", "disabled"),
		RequiredString("name", " service "),
	)
	if err != nil {
		t.Fatalf("Validate valid rules: %v", err)
	}
}

func TestValidateCollectsErrors(t *testing.T) {
	errs := Collect(
		Required("id", 0),
		Max("size", 11, 10),
		In("status", "bad", "ok", "disabled"),
	)
	if len(errs) != 3 {
		t.Fatalf("Collect len = %d, want 3", len(errs))
	}
	if errs.Err() == nil {
		t.Fatalf("Errors.Err() = nil, want error")
	}
}
