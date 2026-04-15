package dbsp

import (
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp/expr"
)

// ================== ExprDbRule Tests ==================

func TestExprDbRuleRegion(t *testing.T) {
	tmpl, err := expr.ParseTemplate("order_${region}_db")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := expr.ParseExpands([]string{
		`${region} := enum(SG, TH, ID)`,
		`${region} = @{region}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	rule := NewExprDbRule(tmpl, expands)

	tests := []struct {
		region string
		want   string
	}{
		{"SG", "order_SG_db"},
		{"TH", "order_TH_db"},
		{"ID", "order_ID_db"},
	}

	for _, tt := range tests {
		sk := dbspi.NewShardingKey().SetVal("region", tt.region)
		got, err := rule.ResolveDbKey(sk)
		if err != nil {
			t.Fatalf("region=%s: %v", tt.region, err)
		}
		if got != tt.want {
			t.Fatalf("region=%s: expected %q, got %q", tt.region, tt.want, got)
		}
	}
}

func TestExprDbRuleEnumerate(t *testing.T) {
	tmpl, err := expr.ParseTemplate("order_${region}_db")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := expr.ParseExpands([]string{
		`${region} := enum(SG, TH, ID)`,
		`${region} = @{region}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	rule := NewExprDbRule(tmpl, expands)
	names, err := rule.EnumerateDbNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 3 {
		t.Fatalf("expected 3 db names, got %d: %v", len(names), names)
	}

	expected := map[string]bool{
		"order_SG_db": true,
		"order_TH_db": true,
		"order_ID_db": true,
	}
	for _, name := range names {
		if !expected[name] {
			t.Fatalf("unexpected db name: %s", name)
		}
	}
}

func TestExprDbRuleHashMod(t *testing.T) {
	tmpl, err := expr.ParseTemplate("order_db_${idx}")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := expr.ParseExpands([]string{
		`${idx} := range(0, 4)`,
		`${idx} = hash(@{shop_id}) % 4`,
	})
	if err != nil {
		t.Fatal(err)
	}

	rule := NewExprDbRule(tmpl, expands)

	sk := dbspi.NewShardingKey().SetVal("shop_id", int64(42))
	got, err := rule.ResolveDbKey(sk)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected non-empty db key")
	}
	t.Logf("hash-mod db key: %s", got)
}

func TestExprDbRuleEnumerateRange(t *testing.T) {
	tmpl, err := expr.ParseTemplate("order_db_${idx}")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := expr.ParseExpands([]string{
		`${idx} := range(0, 4)`,
		`${idx} = hash(@{shop_id}) % 4`,
	})
	if err != nil {
		t.Fatal(err)
	}

	rule := NewExprDbRule(tmpl, expands)
	names, err := rule.EnumerateDbNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 4 {
		t.Fatalf("expected 4 db names, got %d: %v", len(names), names)
	}

	expected := map[string]bool{
		"order_db_0": true,
		"order_db_1": true,
		"order_db_2": true,
		"order_db_3": true,
	}
	for _, name := range names {
		if !expected[name] {
			t.Fatalf("unexpected db name: %s", name)
		}
	}
}

// ================== ExprTableRule Tests ==================

func TestExprTableRuleResolve(t *testing.T) {
	tmpl, err := expr.ParseTemplate("order_tab_${index}")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := expr.ParseExpands([]string{
		`${idx} := range(0, 1000)`,
		`${idx2} = @{shop_id} / 1000`,
		`${idx} = ${idx2} % 1000`,
		`${index} = fill(${idx}, 8)`,
	})
	if err != nil {
		t.Fatal(err)
	}

	rule, err := NewExprTableRule(tmpl, expands)
	if err != nil {
		t.Fatal(err)
	}

	sk := dbspi.NewShardingKey().SetVal("shop_id", int64(123456789))
	got, err := rule.ResolveTable("order_tab", sk)
	if err != nil {
		t.Fatal(err)
	}
	// 123456789 / 1000 = 123456, 123456 % 1000 = 456, fill(456,8) = "00000456"
	if got != "order_tab_00000456" {
		t.Fatalf("expected 'order_tab_00000456', got %q", got)
	}
}

func TestExprTableRuleShardCount(t *testing.T) {
	tmpl, err := expr.ParseTemplate("order_tab_${index}")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := expr.ParseExpands([]string{
		`${idx} := range(0, 100)`,
		`${idx} = @{shop_id} % 100`,
		`${index} = fill(${idx}, 4)`,
	})
	if err != nil {
		t.Fatal(err)
	}

	rule, err := NewExprTableRule(tmpl, expands)
	if err != nil {
		t.Fatal(err)
	}

	if rule.ShardCount() != 100 {
		t.Fatalf("expected shard count 100, got %d", rule.ShardCount())
	}
}

func TestExprTableRuleShardName(t *testing.T) {
	tmpl, err := expr.ParseTemplate("order_tab_${index}")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := expr.ParseExpands([]string{
		`${idx} := range(0, 100)`,
		`${idx} = @{shop_id} % 100`,
		`${index} = fill(${idx}, 4)`,
	})
	if err != nil {
		t.Fatal(err)
	}

	rule, err := NewExprTableRule(tmpl, expands)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		index int
		want  string
	}{
		{0, "order_tab_0000"},
		{1, "order_tab_0001"},
		{42, "order_tab_0042"},
		{99, "order_tab_0099"},
	}

	for _, tt := range tests {
		got, err := rule.ShardName("order_tab", tt.index)
		if err != nil {
			t.Fatalf("index=%d: %v", tt.index, err)
		}
		if got != tt.want {
			t.Fatalf("index=%d: expected %q, got %q", tt.index, tt.want, got)
		}
	}
}

func TestExprTableRuleEnumBased(t *testing.T) {
	tmpl, err := expr.ParseTemplate("order_tab_${region}")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := expr.ParseExpands([]string{
		`${region} := enum(SG, TH, ID)`,
		`${region} = @{region}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	rule, err := NewExprTableRule(tmpl, expands)
	if err != nil {
		t.Fatal(err)
	}

	if rule.ShardCount() != 3 {
		t.Fatalf("expected shard count 3, got %d", rule.ShardCount())
	}

	expected := []string{"order_tab_SG", "order_tab_TH", "order_tab_ID"}
	for i, want := range expected {
		got, err := rule.ShardName("order_tab", i)
		if err != nil {
			t.Fatalf("index=%d: %v", i, err)
		}
		if got != want {
			t.Fatalf("index=%d: expected %q, got %q", i, want, got)
		}
	}
}

func TestExprDbRuleNilKey(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_${region}_db")
	expands, _ := expr.ParseExpands([]string{
		`${region} := enum(SG, TH)`,
		`${region} = @{region}`,
	})
	rule := NewExprDbRule(tmpl, expands)

	_, err := rule.ResolveDbKey(nil)
	if err != dbspi.ErrShardingKeyRequired {
		t.Fatalf("expected ErrShardingKeyRequired, got %v", err)
	}
}

func TestExprTableRuleNilKey(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_tab_${index}")
	expands, _ := expr.ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} = @{shop_id} % 10`,
		`${index} = fill(${idx}, 4)`,
	})
	rule, _ := NewExprTableRule(tmpl, expands)

	_, err := rule.ResolveTable("order_tab", nil)
	if err != dbspi.ErrShardingKeyRequired {
		t.Fatalf("expected ErrShardingKeyRequired, got %v", err)
	}
}

func TestExprTableRuleShardNameOutOfRange(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_tab_${index}")
	expands, _ := expr.ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} = @{shop_id} % 10`,
		`${index} = fill(${idx}, 4)`,
	})
	rule, _ := NewExprTableRule(tmpl, expands)

	_, err := rule.ShardName("order_tab", 10)
	if err == nil {
		t.Fatal("expected error for index out of range")
	}
	t.Logf("error: %v", err)

	_, err = rule.ShardName("order_tab", -1)
	if err == nil {
		t.Fatal("expected error for negative index")
	}
}

func TestExprDbRuleMissingColumn(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_${region}_db")
	expands, _ := expr.ParseExpands([]string{
		`${region} := enum(SG, TH)`,
		`${region} = @{region}`,
	})
	rule := NewExprDbRule(tmpl, expands)

	sk := dbspi.NewShardingKey().SetVal("shop_id", int64(123))
	_, err := rule.ResolveDbKey(sk)
	if err == nil {
		t.Fatal("expected error for missing column @{region}")
	}
	t.Logf("error: %v", err)
}

func TestExprTableRuleSimpleHashMod(t *testing.T) {
	tmpl, err := expr.ParseTemplate("order_tab_${index}")
	if err != nil {
		t.Fatal(err)
	}
	expands, err := expr.ParseExpands([]string{
		`${idx} := range(0, 1000)`,
		`${idx} = hash(@{shop_id}) % 1000`,
		`${index} = fill(${idx}, 8)`,
	})
	if err != nil {
		t.Fatal(err)
	}

	rule, err := NewExprTableRule(tmpl, expands)
	if err != nil {
		t.Fatal(err)
	}

	sk := dbspi.NewShardingKey().SetVal("shop_id", int64(42))
	got1, _ := rule.ResolveTable("order_tab", sk)
	got2, _ := rule.ResolveTable("order_tab", sk)
	if got1 != got2 {
		t.Fatalf("expected deterministic result: %q vs %q", got1, got2)
	}
	t.Logf("hash-mod table: %s", got1)
}
