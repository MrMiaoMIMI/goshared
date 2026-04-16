package dbsp

import (
	"reflect"
	"strings"
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

// ================== RequiredColumns Tests ==================

func TestExprDbRuleRequiredColumns(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_${region}_db")
	expands, _ := expr.ParseExpands([]string{
		`${region} := enum(SG, TH)`,
		`${region} = @{region}`,
	})
	rule := NewExprDbRule(tmpl, expands)
	cols := rule.RequiredColumns()
	if len(cols) != 1 || cols[0] != "region" {
		t.Fatalf("expected [region], got %v", cols)
	}
}

func TestExprTableRuleRequiredColumns(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_tab_${index}")
	expands, _ := expr.ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} = hash(@{shop_id}) % 10`,
		`${index} = fill(${idx}, 8)`,
	})
	rule, _ := NewExprTableRule(tmpl, expands)
	cols := rule.RequiredColumns()
	if len(cols) != 1 || cols[0] != "shop_id" {
		t.Fatalf("expected [shop_id], got %v", cols)
	}
}

func TestRequiredColumnsComposite(t *testing.T) {
	expands, _ := expr.ParseExpands([]string{
		`${region} := enum(SG, TH)`,
		`${region} = @{region}`,
	})
	dbRule := NewExprDbRule(nil, expands)

	expands2, _ := expr.ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} = hash(@{shop_id}) % 10`,
		`${index} = fill(${idx}, 8)`,
	})
	tableRule, _ := NewExprTableRule(nil, expands2)

	dbCols := dbRule.RequiredColumns()
	tableCols := tableRule.RequiredColumns()

	allCols := make(map[string]bool)
	for _, c := range dbCols {
		allCols[c] = true
	}
	for _, c := range tableCols {
		allCols[c] = true
	}

	if !allCols["region"] || !allCols["shop_id"] || len(allCols) != 2 {
		t.Fatalf("expected {region, shop_id}, got %v", allCols)
	}
}

// ================== ExtractEqColumnsFromQuery Tests ==================

func TestExtractEqColumns_SimpleEq(t *testing.T) {
	shopId := int64(12345)
	query := NewQuery(NewField[int64]("shop_id").Eq(&shopId))
	cols := ExtractEqColumnsFromQuery(query)
	if vals, ok := cols["shop_id"]; !ok || len(vals) != 1 || vals[0] != int64(12345) {
		t.Fatalf("expected shop_id=[12345], got %v", cols)
	}
}

func TestExtractEqColumns_MultipleEq(t *testing.T) {
	shopId := int64(12345)
	status := 1
	query := NewQuery(
		NewField[int64]("shop_id").Eq(&shopId),
		NewField[int]("status").Eq(&status),
	)
	cols := ExtractEqColumnsFromQuery(query)
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %v", cols)
	}
	if cols["shop_id"][0] != int64(12345) || cols["status"][0] != 1 {
		t.Fatalf("unexpected values: %v", cols)
	}
}

func TestExtractEqColumns_NestedAnd(t *testing.T) {
	shopId := int64(12345)
	status := 1
	query := NewQuery(
		And(NewField[int64]("shop_id").Eq(&shopId)),
		And(NewField[int]("status").Eq(&status)),
	)
	cols := ExtractEqColumnsFromQuery(query)
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %v", cols)
	}
}

func TestExtractEqColumns_DeepNesting(t *testing.T) {
	shopId := int64(12345)
	query := NewQuery(And(And(NewField[int64]("shop_id").Eq(&shopId))))
	cols := ExtractEqColumnsFromQuery(query)
	if cols["shop_id"][0] != int64(12345) {
		t.Fatalf("deep nesting: expected shop_id=[12345], got %v", cols)
	}
}

func TestExtractEqColumns_OrExtracted(t *testing.T) {
	shopId1 := int64(11111)
	shopId2 := int64(22222)
	query := Or(
		NewField[int64]("shop_id").Eq(&shopId1),
		NewField[int64]("shop_id").Eq(&shopId2),
	)
	cols := ExtractEqColumnsFromQuery(query)
	vals := cols["shop_id"]
	if len(vals) != 2 {
		t.Fatalf("OR query should yield 2 values for shop_id, got %v", cols)
	}
	if vals[0] != int64(11111) || vals[1] != int64(22222) {
		t.Fatalf("expected [11111, 22222], got %v", vals)
	}
}

func TestExtractEqColumns_MixedAndOr(t *testing.T) {
	shopId := int64(12345)
	status1 := 1
	status2 := 2
	// AND(OR(status=1, status=2), shop_id=12345)
	// Both AND and OR branches are traversed for value collection
	query := NewQuery(
		Or(NewField[int]("status").Eq(&status1), NewField[int]("status").Eq(&status2)),
		NewField[int64]("shop_id").Eq(&shopId),
	)
	cols := ExtractEqColumnsFromQuery(query)
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns (shop_id + status), got %v", cols)
	}
	if cols["shop_id"][0] != int64(12345) {
		t.Fatalf("expected shop_id=[12345], got %v", cols["shop_id"])
	}
	statusVals := cols["status"]
	if len(statusVals) != 2 {
		t.Fatalf("expected 2 status values from OR, got %v", statusVals)
	}
}

func TestExtractEqColumns_InExtracted(t *testing.T) {
	shopIds := []int64{1, 2, 3}
	query := NewQuery(
		NewField[int64]("shop_id").In(shopIds),
	)
	cols := ExtractEqColumnsFromQuery(query)
	vals := cols["shop_id"]
	if len(vals) != 3 {
		t.Fatalf("expected 3 values from IN, got %v", cols)
	}
	if vals[0] != int64(1) || vals[1] != int64(2) || vals[2] != int64(3) {
		t.Fatalf("expected [1, 2, 3], got %v", vals)
	}
}

func TestExtractEqColumns_GtSkipped(t *testing.T) {
	amount := int64(100)
	query := NewQuery(
		NewField[int64]("amount").Gt(&amount),
	)
	cols := ExtractEqColumnsFromQuery(query)
	if len(cols) != 0 {
		t.Fatalf("Gt should not be extracted, got %v", cols)
	}
}

func TestExtractEqColumns_InAndEqMixed(t *testing.T) {
	shopIds := []int64{11111, 22222}
	status := 1
	query := NewQuery(
		NewField[int64]("shop_id").In(shopIds),
		NewField[int]("status").Eq(&status),
	)
	cols := ExtractEqColumnsFromQuery(query)
	if len(cols) != 2 {
		t.Fatalf("expected 2 columns, got %v", cols)
	}
	if len(cols["shop_id"]) != 2 {
		t.Fatalf("expected 2 shop_id values from IN, got %v", cols["shop_id"])
	}
	if len(cols["status"]) != 1 || cols["status"][0] != 1 {
		t.Fatalf("expected status=[1], got %v", cols["status"])
	}
}

func TestExtractEqColumns_NilQuery(t *testing.T) {
	cols := ExtractEqColumnsFromQuery(nil)
	if len(cols) != 0 {
		t.Fatalf("nil query should yield 0 columns, got %v", cols)
	}
}

func TestExtractEqColumns_MultipleValues(t *testing.T) {
	shopId1 := int64(11111)
	shopId2 := int64(22222)
	query := NewQuery(
		NewField[int64]("shop_id").Eq(&shopId1),
		NewField[int64]("shop_id").Eq(&shopId2),
	)
	cols := ExtractEqColumnsFromQuery(query)
	vals := cols["shop_id"]
	if len(vals) != 2 {
		t.Fatalf("expected 2 values for shop_id, got %v", cols)
	}
	if vals[0] != int64(11111) || vals[1] != int64(22222) {
		t.Fatalf("expected [11111, 22222], got %v", vals)
	}
}

func TestExtractEqColumns_NestedMultipleValues(t *testing.T) {
	shopId1 := int64(11111)
	shopId2 := int64(22222)
	status := 1
	query := NewQuery(
		NewField[int64]("shop_id").Eq(&shopId1),
		And(NewField[int64]("shop_id").Eq(&shopId2), NewField[int]("status").Eq(&status)),
	)
	cols := ExtractEqColumnsFromQuery(query)
	vals := cols["shop_id"]
	if len(vals) != 2 {
		t.Fatalf("expected 2 shop_id values from nested AND, got %v", cols)
	}
	if vals[0] != int64(11111) || vals[1] != int64(22222) {
		t.Fatalf("expected [11111, 22222], got %v", vals)
	}
	if len(cols["status"]) != 1 || cols["status"][0] != 1 {
		t.Fatalf("expected status=[1], got %v", cols["status"])
	}
}

func TestExtractEqColumns_SameValueCollected(t *testing.T) {
	shopId := int64(12345)
	query := NewQuery(
		NewField[int64]("shop_id").Eq(&shopId),
		NewField[int64]("shop_id").Eq(&shopId),
	)
	cols := ExtractEqColumnsFromQuery(query)
	vals := cols["shop_id"]
	if len(vals) != 2 {
		t.Fatalf("expected 2 raw values (dedup happens later), got %v", vals)
	}
	if vals[0] != int64(12345) || vals[1] != int64(12345) {
		t.Fatalf("expected [12345, 12345], got %v", vals)
	}
}

// ================== MergeIntoMultiValues Tests ==================

func TestMergeIntoMultiValues_DifferentColumns(t *testing.T) {
	entity := map[string]any{"shop_id": int64(12345)}
	query := map[string][]any{"region": {"SG"}}
	merged := mergeIntoMultiValues(entity, query)
	if len(merged) != 2 {
		t.Fatalf("expected 2 columns, got %v", merged)
	}
	if merged["shop_id"][0] != int64(12345) || merged["region"][0] != "SG" {
		t.Fatalf("unexpected merged result: %v", merged)
	}
}

func TestMergeIntoMultiValues_SameColumn(t *testing.T) {
	entity := map[string]any{"shop_id": int64(12345)}
	query := map[string][]any{"shop_id": {int64(12345)}}
	merged := mergeIntoMultiValues(entity, query)
	vals := merged["shop_id"]
	if len(vals) != 2 {
		t.Fatalf("expected 2 raw values, got %v", vals)
	}
}

func TestMergeIntoMultiValues_DifferentValues(t *testing.T) {
	entity := map[string]any{"shop_id": int64(12345)}
	query := map[string][]any{"shop_id": {int64(99999)}}
	merged := mergeIntoMultiValues(entity, query)
	vals := merged["shop_id"]
	if len(vals) != 2 {
		t.Fatalf("expected 2 values (both collected for later validation), got %v", vals)
	}
	if vals[0] != int64(12345) || vals[1] != int64(99999) {
		t.Fatalf("expected [12345, 99999], got %v", vals)
	}
}

// ================== DeduplicateValues Tests ==================

func TestDeduplicateValues_NoDuplicates(t *testing.T) {
	vals := deduplicateValues([]any{int64(1), int64(2), int64(3)})
	if len(vals) != 3 {
		t.Fatalf("expected 3, got %v", vals)
	}
}

func TestDeduplicateValues_WithDuplicates(t *testing.T) {
	vals := deduplicateValues([]any{int64(1), int64(2), int64(1), int64(3), int64(2)})
	if len(vals) != 3 {
		t.Fatalf("expected 3 unique values, got %v", vals)
	}
}

func TestDeduplicateValues_AllSame(t *testing.T) {
	vals := deduplicateValues([]any{int64(42), int64(42), int64(42)})
	if len(vals) != 1 || vals[0] != int64(42) {
		t.Fatalf("expected [42], got %v", vals)
	}
}

// ================== ShardingKeyResolver Tests ==================

type testOrder struct {
	ID     int64 `gorm:"primaryKey"`
	ShopID int64 `gorm:"column:shop_id"`
	Amount int64 `gorm:"column:amount"`
	Status int   `gorm:"column:status"`
}

func (*testOrder) TableName() string   { return "order_tab" }
func (*testOrder) IdFiledName() string { return "id" }

func TestBuildShardingKeyResolver_SingleColumn(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_tab_${index}")
	expands, _ := expr.ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} = hash(@{shop_id}) % 10`,
		`${index} = fill(${idx}, 8)`,
	})
	tableRule, _ := NewExprTableRule(tmpl, expands)

	resolver := buildShardingKeyResolver(
		reflect.TypeOf(&testOrder{}),
		"id",
		nil,
		tableRule,
	)
	if resolver == nil {
		t.Fatal("expected non-nil resolver")
	}
	if len(resolver.requiredCols) != 1 || resolver.requiredCols[0] != "shop_id" {
		t.Fatalf("expected [shop_id], got %v", resolver.requiredCols)
	}
}

func TestShardingKeyResolver_FromEntity(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_tab_${index}")
	expands, _ := expr.ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} = hash(@{shop_id}) % 10`,
		`${index} = fill(${idx}, 8)`,
	})
	tableRule, _ := NewExprTableRule(tmpl, expands)

	resolver := buildShardingKeyResolver(
		reflect.TypeOf(&testOrder{}),
		"id",
		nil,
		tableRule,
	)

	order := &testOrder{ShopID: 12345, Amount: 100}
	columns := resolver.fromEntity(order)
	if columns["shop_id"] != int64(12345) {
		t.Fatalf("expected shop_id=12345, got %v", columns)
	}
}

func TestShardingKeyResolver_FromId(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_tab_${index}")
	expands, _ := expr.ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} = hash(@{id}) % 10`,
		`${index} = fill(${idx}, 8)`,
	})
	tableRule, _ := NewExprTableRule(tmpl, expands)

	resolver := buildShardingKeyResolver(
		reflect.TypeOf(&testOrder{}),
		"id",
		nil,
		tableRule,
	)

	columns := resolver.fromId(int64(1001))
	if columns["id"] != int64(1001) {
		t.Fatalf("expected id=1001, got %v", columns)
	}
}

func TestShardingKeyResolver_BuildShardingKey_Success(t *testing.T) {
	tmpl, _ := expr.ParseTemplate("order_tab_${index}")
	expands, _ := expr.ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} = hash(@{shop_id}) % 10`,
		`${index} = fill(${idx}, 8)`,
	})
	tableRule, _ := NewExprTableRule(tmpl, expands)

	resolver := buildShardingKeyResolver(
		reflect.TypeOf(&testOrder{}),
		"id",
		nil,
		tableRule,
	)

	columns := map[string]any{"shop_id": int64(12345)}
	sk, err := resolver.buildShardingKey(columns)
	if err != nil {
		t.Fatal(err)
	}
	v, err := sk.Get("shop_id")
	if err != nil || v != int64(12345) {
		t.Fatalf("expected shop_id=12345 in sharding key, got %v", v)
	}
}

func TestShardingKeyResolver_BuildShardingKey_MissingColumn(t *testing.T) {
	expands1, _ := expr.ParseExpands([]string{
		`${region} := enum(SG, TH)`,
		`${region} = @{region}`,
	})
	dbRule := NewExprDbRule(nil, expands1)

	expands2, _ := expr.ParseExpands([]string{
		`${idx} := range(0, 10)`,
		`${idx} = hash(@{shop_id}) % 10`,
		`${index} = fill(${idx}, 8)`,
	})
	tableRule, _ := NewExprTableRule(nil, expands2)

	resolver := buildShardingKeyResolver(
		reflect.TypeOf(&testOrder{}),
		"id",
		dbRule,
		tableRule,
	)
	if resolver == nil {
		t.Fatal("expected non-nil resolver")
	}

	// Only provide shop_id, missing region
	columns := map[string]any{"shop_id": int64(12345)}
	_, err := resolver.buildShardingKey(columns)
	if err == nil {
		t.Fatal("expected missing column error")
	}
	if !strings.Contains(err.Error(), "region") {
		t.Fatalf("error should mention 'region', got: %v", err)
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error should mention 'missing', got: %v", err)
	}
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
