package example

import (
	"context"
	"strings"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// RegionalOrder has a region field, used for composite-key auto-extraction tests.
type RegionalOrder struct {
	ID     int64  `gorm:"primaryKey"`
	ShopID int64  `gorm:"column:shop_id"`
	Region string `gorm:"column:region"`
	Amount int64  `gorm:"column:amount"`
}

func (*RegionalOrder) TableName() string   { return "order_tab" }
func (*RegionalOrder) DbKey() string       { return "order_dbs" }
func (*RegionalOrder) IdFiledName() string { return "id" }

// ==================== Auto-extract from Entity ====================

func Test_AutoKey_Create_FromEntity(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	err := executor.Create(ctx, &Order{ShopID: 12345, Amount: 100})
	t.Logf("Create auto-extract from entity: err=%v", err)
}

func Test_AutoKey_Save_FromEntity(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	err := executor.Save(ctx, &Order{ShopID: 12345, Amount: 200})
	t.Logf("Save auto-extract from entity: err=%v", err)
}

func Test_AutoKey_Update_FromEntity(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	err := executor.Update(ctx, &Order{ID: 1, ShopID: 12345, Amount: 300})
	t.Logf("Update auto-extract from entity: err=%v", err)
}

func Test_AutoKey_Delete_FromEntity(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	err := executor.Delete(ctx, &Order{ID: 1, ShopID: 12345})
	t.Logf("Delete auto-extract from entity: err=%v", err)
}

func Test_AutoKey_BatchCreate_FromEntity(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	entities := []*Order{
		{ShopID: 12345, Amount: 100},
		{ShopID: 12345, Amount: 200},
	}
	err := executor.BatchCreate(ctx, entities, 100)
	t.Logf("BatchCreate auto-extract from first entity: err=%v", err)
}

func Test_AutoKey_BatchCreate_EmptySlice(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	err := executor.BatchCreate(ctx, []*Order{}, 100)
	if err != nil {
		t.Fatalf("BatchCreate with empty slice should succeed, got: %v", err)
	}
	t.Log("BatchCreate empty slice: no error (early return)")
}

// ==================== Auto-extract from Query ====================

func Test_AutoKey_Find_FromQuery(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	shopId := int64(12345)
	orders, err := executor.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
	t.Logf("Find auto-extract from query: orders=%v, err=%v", orders, err)
}

func Test_AutoKey_Count_FromQuery(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	shopId := int64(12345)
	count, err := executor.Count(ctx, dbhelper.Q(shopIdField.Eq(&shopId)))
	t.Logf("Count auto-extract from query: count=%d, err=%v", count, err)
}

func Test_AutoKey_Exists_FromQuery(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	shopId := int64(12345)
	exists, order, err := executor.Exists(ctx, dbhelper.Q(shopIdField.Eq(&shopId)))
	t.Logf("Exists auto-extract from query: exists=%v, order=%v, err=%v", exists, order, err)
}

func Test_AutoKey_DeleteByQuery_FromQuery(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	shopId := int64(12345)
	err := executor.DeleteByQuery(ctx, dbhelper.Q(shopIdField.Eq(&shopId)))
	t.Logf("DeleteByQuery auto-extract from query: err=%v", err)
}

// ==================== Auto-extract from Query (nested) ====================

func Test_AutoKey_Find_FromNestedQuery(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")
	statusField := dbhelper.NewField[int]("status")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	shopId := int64(12345)
	status := 1

	// Nested: AND(AND(shop_id=12345, status=1))
	nestedQuery := dbhelper.Q(
		dbhelper.Q(shopIdField.Eq(&shopId), statusField.Eq(&status)),
	)
	orders, err := executor.Find(ctx, nestedQuery, nil)
	t.Logf("Find from nested AND query: orders=%v, err=%v", orders, err)
}

func Test_AutoKey_Find_FromMixedQuery(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")
	statusField := dbhelper.NewField[int]("status")
	amountField := dbhelper.NewField[int64]("amount")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	shopId := int64(12345)
	status1 := 1
	status2 := 2
	amount := int64(100)

	// AND(OR(status=1, status=2), shop_id=12345, amount > 100)
	// shop_id is extractable; status values inside OR are skipped; amount is Gt not Eq
	query := dbhelper.Q(
		dbhelper.Or(statusField.Eq(&status1), statusField.Eq(&status2)),
		shopIdField.Eq(&shopId),
		amountField.Gt(&amount),
	)
	orders, err := executor.Find(ctx, query, nil)
	t.Logf("Find from mixed query (OR + Eq + Gt): orders=%v, err=%v", orders, err)
}

// ==================== Auto-extract from ID ====================

func Test_AutoKey_GetById_FromId(t *testing.T) {
	db := testNewDb()

	// Shard by ID column
	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	order, err := executor.GetById(ctx, int64(1001))
	t.Logf("GetById auto-extract from id: order=%v, err=%v", order, err)
}

func Test_AutoKey_DeleteById_FromId(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	err := executor.DeleteById(ctx, int64(1001))
	t.Logf("DeleteById auto-extract from id: err=%v", err)
}

// ==================== Ctx key + auto-extract aggregation ====================

func Test_AutoKey_CtxAndEntity_SameTable(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	// ctx key shop_id=22345 (%10=5), entity ShopID=12345 (%10=5) → same table → OK
	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(22345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	err := executor.Create(ctx, &Order{ShopID: 12345, Amount: 100})
	if err != nil {
		t.Fatalf("Ctx and entity same table should succeed, got: %v", err)
	}
	t.Logf("Ctx + entity same table: err=%v", err)
}

func Test_AutoKey_CtxAndEntity_CrossShard(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	// ctx key shop_id=99999 (%10=9), entity ShopID=12345 (%10=5) → different tables → error
	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(99999))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	err := executor.Create(ctx, &Order{ShopID: 12345, Amount: 100})
	if err == nil {
		t.Fatal("Expected cross-shard error when ctx key and entity differ")
	}
	if !strings.Contains(err.Error(), "cross-shard") {
		t.Fatalf("Error should mention 'cross-shard', got: %v", err)
	}
	t.Logf("Ctx + entity cross-shard: %v", err)
}

func Test_AutoKey_CtxAndQuery_SameTable(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	// ctx key shop_id=22345 (%10=5), query shop_id=12345 (%10=5) → same table → OK
	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(22345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	shopId := int64(12345)
	orders, err := executor.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
	if err != nil {
		t.Fatalf("Ctx and query same table should succeed, got: %v", err)
	}
	t.Logf("Ctx + query same table: orders=%v, err=%v", orders, err)
}

func Test_AutoKey_CtxAndQuery_CrossShard(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	// ctx key shop_id=99999 (%10=9), query shop_id=12345 (%10=5) → different tables → error
	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(99999))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	shopId := int64(12345)
	_, err := executor.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
	if err == nil {
		t.Fatal("Expected cross-shard error when ctx key and query differ")
	}
	if !strings.Contains(err.Error(), "cross-shard") {
		t.Fatalf("Error should mention 'cross-shard', got: %v", err)
	}
	t.Logf("Ctx + query cross-shard: %v", err)
}

func Test_AutoKey_CtxOnly_StillWorks(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	// Only ctx key, no conflicting sources → should work as before
	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	// Raw/Exec only uses ctx key (no auto-extraction possible)
	err := executor.Exec(ctx, "SELECT 1")
	t.Logf("Ctx-only Raw/Exec: err=%v", err)
}

// ==================== Composite key: auto-extract all fields from entity ====================

func Test_AutoKey_CompositeKey_FromEntity(t *testing.T) {
	dbSG := testNewDb()
	dbTH := testNewDb()

	executor := dbhelper.NewShardedExecutor(&RegionalOrder{},
		dbhelper.WithDbs(dbhelper.NamedDbs(map[string]dbspi.Db{
			"SG": dbSG,
			"TH": dbTH,
		})),
		dbhelper.WithDbRule(dbhelper.NewExprDbRule(
			"${region}",
			"${region} := enum(SG, TH)",
			"${region} = @{region}",
		)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	err := executor.Create(ctx, &RegionalOrder{ShopID: 12345, Region: "SG", Amount: 100})
	t.Logf("Composite key auto-extract from entity (region+shop_id): err=%v", err)
}

// ==================== Missing column: entity lacks required field ====================

func Test_AutoKey_MissingColumn_EntityLacksRegion(t *testing.T) {
	dbSG := testNewDb()
	dbTH := testNewDb()

	// Order struct has no "region" field, but db rule requires @{region}
	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.NamedDbs(map[string]dbspi.Db{
			"SG": dbSG,
			"TH": dbTH,
		})),
		dbhelper.WithDbRule(dbhelper.NewExprDbRule(
			"${region}",
			"${region} := enum(SG, TH)",
			"${region} = @{region}",
		)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	err := executor.Create(ctx, &Order{ShopID: 12345, Amount: 100})
	if err == nil {
		t.Fatal("Expected error for missing region column")
	}
	if !strings.Contains(err.Error(), "region") {
		t.Fatalf("Error should mention missing 'region' column, got: %v", err)
	}
	t.Logf("Missing column error: %v", err)
}

// ==================== Same-table validation (values route to same shard) ====================

func Test_AutoKey_QuerySameTable_DifferentValues(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	// 11111 % 10 = 1, 21111 % 10 = 1 → same table → allowed
	shopId1 := int64(11111)
	shopId2 := int64(21111)

	query := dbhelper.Q(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
	orders, err := executor.Find(ctx, query, nil)
	if err != nil {
		t.Fatalf("Same-table values should not error, got: %v", err)
	}
	t.Logf("Same-table different values: orders=%v, err=%v", orders, err)
}

func Test_AutoKey_QueryCrossShard_DifferentValues(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	// 11111 % 10 = 1, 22222 % 10 = 2 → different tables → error
	shopId1 := int64(11111)
	shopId2 := int64(22222)

	query := dbhelper.Q(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
	_, err := executor.Find(ctx, query, nil)
	if err == nil {
		t.Fatal("Expected cross-shard error")
	}
	if !strings.Contains(err.Error(), "cross-shard") {
		t.Fatalf("Error should mention 'cross-shard', got: %v", err)
	}
	t.Logf("Cross-shard error: %v", err)
}

func Test_AutoKey_QueryCrossShard_NestedDifferentValues(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")
	statusField := dbhelper.NewField[int]("status")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	// 11111 % 10 = 1, 22222 % 10 = 2 → different tables → error
	shopId1 := int64(11111)
	shopId2 := int64(22222)
	status := 1

	conflictQuery := dbhelper.Q(
		shopIdField.Eq(&shopId1),
		dbhelper.Q(shopIdField.Eq(&shopId2), statusField.Eq(&status)),
	)
	_, err := executor.Find(ctx, conflictQuery, nil)
	if err == nil {
		t.Fatal("Expected nested cross-shard error")
	}
	if !strings.Contains(err.Error(), "cross-shard") {
		t.Fatalf("Error should mention 'cross-shard', got: %v", err)
	}
	t.Logf("Nested cross-shard error: %v", err)
}

func Test_AutoKey_QueryNoConflict_SameValue(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	shopId := int64(12345)

	// AND(shop_id=12345, shop_id=12345) -- same value, no conflict
	query := dbhelper.Q(shopIdField.Eq(&shopId), shopIdField.Eq(&shopId))
	orders, err := executor.Find(ctx, query, nil)
	t.Logf("Same value no conflict: orders=%v, err=%v", orders, err)
}

// ==================== Cross-source conflict (FirstOrCreate) ====================

func Test_AutoKey_FirstOrCreate_EntityAndQuery_NoConflict(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	shopId := int64(12345)

	// Entity and query both have shop_id=12345 -- no conflict
	_, err := executor.FirstOrCreate(ctx,
		&Order{ShopID: 12345, Amount: 100},
		dbhelper.Q(shopIdField.Eq(&shopId)),
	)
	t.Logf("FirstOrCreate no conflict (same shop_id): err=%v", err)
}

func Test_AutoKey_FirstOrCreate_EntityAndQuery_CrossShard(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	// Entity shop_id=12345 (% 10 = 5), query shop_id=99999 (% 10 = 9) → different tables
	queryShopId := int64(99999)

	_, err := executor.FirstOrCreate(ctx,
		&Order{ShopID: 12345, Amount: 100},
		dbhelper.Q(shopIdField.Eq(&queryShopId)),
	)
	if err == nil {
		t.Fatal("Expected cross-shard error")
	}
	if !strings.Contains(err.Error(), "cross-shard") {
		t.Fatalf("Error should mention 'cross-shard', got: %v", err)
	}
	t.Logf("FirstOrCreate cross-shard: %v", err)
}

func Test_AutoKey_FirstOrCreate_EntityAndQuery_SameTable(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	// Entity shop_id=12345 (% 10 = 5), query shop_id=22345 (% 10 = 5) → same table → OK
	queryShopId := int64(22345)

	_, err := executor.FirstOrCreate(ctx,
		&Order{ShopID: 12345, Amount: 100},
		dbhelper.Q(shopIdField.Eq(&queryShopId)),
	)
	t.Logf("FirstOrCreate same-table different values: err=%v", err)
}

// ==================== OR query: now extractable with same-target validation ====================

func Test_AutoKey_OrQuery_SameTable(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	// 11111 % 10 = 1, 21111 % 10 = 1 → same table → allowed
	shopId1 := int64(11111)
	shopId2 := int64(21111)

	orQuery := dbhelper.Or(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
	orders, err := executor.Find(ctx, orQuery, nil)
	if err != nil {
		t.Fatalf("OR query with same-table values should succeed, got: %v", err)
	}
	t.Logf("OR query same table: orders=%v, err=%v", orders, err)
}

func Test_AutoKey_InQuery_SameTable(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	// 11111 % 10 = 1, 21111 % 10 = 1 → same table → allowed
	shopId1 := int64(11111)
	shopId2 := int64(21111)

	inQuery := dbhelper.Q(shopIdField.In([]int64{shopId1, shopId2}))
	orders, err := executor.Find(ctx, inQuery, nil)
	if err != nil {
		t.Fatalf("IN query with same-table values should succeed, got: %v", err)
	}
	t.Logf("IN query same table: orders=%v, err=%v", orders, err)
}

func Test_AutoKey_OrQuery_CrossShard(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	// 11111 % 10 = 1, 22222 % 10 = 2 → different tables → error
	shopId1 := int64(11111)
	shopId2 := int64(22222)

	orQuery := dbhelper.Or(shopIdField.Eq(&shopId1), shopIdField.Eq(&shopId2))
	_, err := executor.Find(ctx, orQuery, nil)
	if err == nil {
		t.Fatal("Expected cross-shard error for OR query")
	}
	if !strings.Contains(err.Error(), "cross-shard") {
		t.Fatalf("Error should mention 'cross-shard', got: %v", err)
	}
	t.Logf("OR query cross-shard: %v", err)
}

func Test_AutoKey_InQuery_CrossShard(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	// 11111 % 10 = 1, 22222 % 10 = 2 → different tables → error
	shopId1 := int64(11111)
	shopId2 := int64(22222)

	inQuery := dbhelper.Q(shopIdField.In([]int64{shopId1, shopId2}))
	_, err := executor.Find(ctx, inQuery, nil)
	if err == nil {
		t.Fatal("Expected cross-shard error for OR query")
	}
	if !strings.Contains(err.Error(), "cross-shard") {
		t.Fatalf("Error should mention 'cross-shard', got: %v", err)
	}
	t.Logf("OR query cross-shard: %v", err)
}

func Test_AutoKey_MixedQuery_OrWithAndSameTable(t *testing.T) {
	db := testNewDb()

	shopIdField := dbhelper.NewField[int64]("shop_id")
	statusField := dbhelper.NewField[int]("status")

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	shopId := int64(12345)
	status1 := 1
	status2 := 2

	// AND(OR(status=1, status=2), shop_id=12345)
	// shop_id has one value; status has two values but is not a required sharding column
	query := dbhelper.Q(
		dbhelper.Or(statusField.Eq(&status1), statusField.Eq(&status2)),
		shopIdField.Eq(&shopId),
	)
	orders, err := executor.Find(ctx, query, nil)
	if err != nil {
		t.Fatalf("Mixed AND/OR with non-sharding OR column should succeed, got: %v", err)
	}
	t.Logf("Mixed query OR+AND: orders=%v, err=%v", orders, err)
}

// ==================== Nil query: columns not extractable ====================

func Test_AutoKey_NilQuery_MissingColumn(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = @{shop_id} % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	_, err := executor.Find(ctx, nil, nil)
	if err == nil {
		t.Fatal("Expected missing column error for nil query")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("Error should mention 'missing', got: %v", err)
	}
	t.Logf("Nil query missing column: %v", err)
}

// ==================== DbManager + auto-extract ====================

func Test_AutoKey_DbManager_CreateWithoutExplicitKey(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbhelper.DatabaseConfig{
		Databases: map[string]dbhelper.DatabaseEntry{
			"default": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword, Debug: true,
				DbName: testAppDbName,
			},
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword, Debug: true,
				DbSharding: &dbhelper.DbShardConfig{
					NameExpr:    "order_db_${idx}",
					ExpandExprs: []string{"${idx} := range(0, 2)", "${idx} = @{shop_id} % 2"},
				},
				TableSharding: &dbhelper.TableShardConfig{
					NameExpr:    "order_tab_${index}",
					ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
				},
			},
		},
	})

	orderExec := dbhelper.For(&Order{}, mgr)

	ctx := context.Background()
	err := orderExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})
	t.Logf("DbManager Create without explicit key: err=%v", err)
}

func Test_AutoKey_DbManager_FindWithQuery(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbhelper.DatabaseConfig{
		Databases: map[string]dbhelper.DatabaseEntry{
			"default": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbName: testAppDbName,
			},
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbSharding: &dbhelper.DbShardConfig{
					NameExpr:    "order_db_${idx}",
					ExpandExprs: []string{"${idx} := range(0, 2)", "${idx} = @{shop_id} % 2"},
				},
				TableSharding: &dbhelper.TableShardConfig{
					NameExpr:    "order_tab_${index}",
					ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
				},
			},
		},
	})

	orderExec := dbhelper.For(&Order{}, mgr)

	ctx := context.Background()
	shopId := int64(12345)
	shopIdField := dbhelper.NewField[int64]("shop_id")
	orders, err := orderExec.Find(ctx, dbhelper.Q(shopIdField.Eq(&shopId)), nil)
	t.Logf("DbManager Find with query auto-extract: orders=%v, err=%v", orders, err)
}
