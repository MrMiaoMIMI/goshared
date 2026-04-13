package example

import (
	"context"
	"fmt"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// Order model for sharding examples
/*
建表语句：
CREATE TABLE `order_tab_00000000` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `shop_id` bigint(20) NOT NULL,
  `amount` bigint(20) NOT NULL,
  `status` int(11) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
 CREATE TABLE `order_tab_00000001` LIKE `order_tab_00000000`;
 CREATE TABLE `order_tab_00000002` LIKE `order_tab_00000000`;
 CREATE TABLE `order_tab_00000003` LIKE `order_tab_00000000`;
 CREATE TABLE `order_tab_00000004` LIKE `order_tab_00000000`;
 CREATE TABLE `order_tab_00000005` LIKE `order_tab_00000000`;
 CREATE TABLE `order_tab_00000006` LIKE `order_tab_00000000`;
 CREATE TABLE `order_tab_00000007` LIKE `order_tab_00000000`;
 CREATE TABLE `order_tab_00000008` LIKE `order_tab_00000000`;
 CREATE TABLE `order_tab_00000009` LIKE `order_tab_00000000`;
*/
type Order struct {
	ID     int64 `gorm:"primaryKey"`
	ShopID int64 `gorm:"column:shop_id"`
	Amount int64 `gorm:"column:amount"`
	Status int   `gorm:"column:status"`
}

func (*Order) TableName() string {
	return "order_tab"
}

func (*Order) IdFiledName() string {
	return "id"
}

// ==================== Example 1: Table-only sharding via Shard() ====================

func Test_Sharding_TableOnly_WithShard(t *testing.T) {
	ctx := context.Background()
	db := testNewDb()

	// order_tab sharded into 8 tables by shop_id: order_tab_00 ~ order_tab_07
	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewHashModTableRule(10)),
	)

	shopID := int64(12345)

	// Shard() returns the resolved executor
	shardedExec, err := executor.Shard(shopID)
	if err != nil {
		t.Fatalf("Shard failed: %v", err)
	}

	// GetById on the resolved executor (routes to order_tab_01 since 12345 % 8 = 1)
	order, err := shardedExec.GetById(ctx, 1001)
	t.Logf("Example 1a: GetById via Shard(): order=%v, err=%v", order, err)

	// Find on the resolved executor
	orders, err := shardedExec.Find(ctx, nil, nil)
	t.Logf("Example 1b: Find via Shard(): orders=%v, err=%v", orders, err)

	// Create on the resolved executor
	err = shardedExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})
	t.Logf("Example 1c: Create via Shard(): err=%v", err)
}

// ==================== Example 2: Table-only sharding via ctx ====================

func Test_Sharding_TableOnly_WithCtx(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewHashModTableRule(8)),
	)

	shopID := int64(12345)

	// Inject sharding key into context
	ctx := dbspi.WithShardingKey(context.Background(), shopID)

	// Directly call CRUD methods — sharding key extracted from ctx
	order, err := executor.GetById(ctx, 1001)
	t.Logf("Example 2a: GetById via ctx: order=%v, err=%v", order, err)

	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Example 2b: Find via ctx: orders=%v, err=%v", orders, err)

	err = executor.Create(ctx, &Order{ShopID: 12345, Amount: 200})
	t.Logf("Example 2c: Create via ctx: err=%v", err)
}

// ==================== Example 3: Missing sharding key ====================

func Test_Sharding_MissingKey(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewHashModTableRule(8)),
	)

	// Call without sharding key → should return ErrShardingKeyRequired
	ctx := context.Background()
	_, err := executor.GetById(ctx, 1001)
	t.Logf("Example 3: GetById without sharding key: err=%v", err)
	// Expected: err = "sharding key is required: use Shard(key) or pass via WithShardingKey(ctx, key)"
}

// ==================== Example 4: Scatter-gather ====================

func Test_Sharding_FindAll(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewHashModTableRule(10)),
	)

	ctx := context.Background()

	// FindAll queries all 8 shard tables concurrently, no batching
	allOrders, err := executor.FindAll(ctx, nil, 0)
	t.Logf("Example 4a: FindAll (no batch): count=%d, err=%v", len(allOrders), err)

	// FindAll with batching: fetch 100 rows per batch from each shard
	allOrders, err = executor.FindAll(ctx, nil, 100)
	t.Logf("Example 4b: FindAll (batch=100): count=%d, err=%v", len(allOrders), err)

	// CountAll counts across all 8 shard tables
	total, err := executor.CountAll(ctx, nil)
	t.Logf("Example 4c: CountAll: total=%d, err=%v", total, err)
}

// ==================== Example 5: Database + Table sharding ====================

func Test_Sharding_DbAndTable(t *testing.T) {
	// Two separate databases
	db0 := testNewDb()
	db1 := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.IndexedDbs(db0, db1)),
		dbhelper.WithDbRule(dbhelper.NewHashModDbRule(2)),
		dbhelper.WithTableRule(dbhelper.NewHashModTableRule(10)),
	)

	shopID := int64(12345)
	ctx := dbspi.WithShardingKey(context.Background(), shopID)

	// DbRule:    12345 % 2 = 1 → db1
	// TableRule: 12345 % 10 = 5 → order_tab_00000005
	order, err := executor.GetById(ctx, 1)
	t.Logf("Example 5: Db+Table sharding: order=%v, err=%v", order, err)
}

// ==================== Example 6: Custom table sharding (by date) ====================

func Test_Sharding_CustomTableRule(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewCustomTableRule(
			func(logicalTable string, key any) (string, error) {
				dateStr, ok := key.(string)
				if !ok {
					return "", fmt.Errorf("expected string date key, got %T", key)
				}
				return fmt.Sprintf("%s_%s", logicalTable, dateStr), nil
			},
		)),
	)

	// Routes to order_tab_20260413
	shardedExec, err := executor.Shard("20260413")
	if err != nil {
		t.Fatalf("Shard failed: %v", err)
	}
	orders, err := shardedExec.Find(context.Background(), nil, nil)
	t.Logf("Example 6: Custom table rule (by date): orders=%v, err=%v", orders, err)
}

// ==================== Example 7: Custom db sharding (by country) ====================

func Test_Sharding_CustomDbRule(t *testing.T) {
	dbSEA := testNewDb()
	dbTW := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.NamedDbs(map[string]dbspi.Db{
			"SEA": dbSEA,
			"TW":  dbTW,
		})),
		dbhelper.WithDbRule(dbhelper.NewCustomDbRule(func(key any) (any, error) {
			country, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("expected string country key, got %T", key)
			}
			switch country {
			case "SG", "MY", "PH":
				return "SEA", nil
			case "TW":
				return "TW", nil
			default:
				return nil, fmt.Errorf("unknown country: %s", country)
			}
		})),
	)

	ctx := dbspi.WithShardingKey(context.Background(), "SG")
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Example 7a: Custom db rule (SG → SEA): orders=%v, err=%v", orders, err)

	ctx = dbspi.WithShardingKey(context.Background(), "TW")
	orders, err = executor.Find(ctx, nil, nil)
	t.Logf("Example 7b: Custom db rule (TW): orders=%v, err=%v", orders, err)
}

// ==================== Example 8: Non-sharded executor (backward compatible) ====================

func Test_Sharding_NonShardedExecutor(t *testing.T) {
	db := testNewDb()

	// Non-sharded executor — same as before
	executor := dbhelper.NewExecutor(db, &User{})

	ctx := context.Background()

	// Shard() is no-op for non-sharded executor
	sameExec, err := executor.Shard(12345)
	t.Logf("Example 8a: Shard() on non-sharded: err=%v (should be nil)", err)

	users, err := sameExec.Find(ctx, nil, nil)
	t.Logf("Example 8b: Find via Shard() on non-sharded: users=%v, err=%v", users, err)

	// FindAll is equivalent to Find for non-sharded executor
	allUsers, err := executor.FindAll(ctx, nil, 0)
	t.Logf("Example 8c: FindAll on non-sharded: users=%v, err=%v", allUsers, err)
}

// ==================== Example 9: Shard error handling ====================

func Test_Sharding_ErrorHandling(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewCustomTableRule(
			func(logicalTable string, key any) (string, error) {
				id, ok := key.(int64)
				if !ok {
					return "", fmt.Errorf("expected int64 key, got %T", key)
				}
				return fmt.Sprintf("%s_%02d", logicalTable, id%8), nil
			},
		)),
	)

	// Shard with wrong type → should return error
	_, err := executor.Shard("not_an_int64")
	t.Logf("Example 9: Shard with wrong key type: err=%v", err)
	// Expected: err contains "expected int64 key, got string"
}

// ==================== Example 10: Struct sharding ====================

func Test_Sharding_StructSharding(t *testing.T) {
	db := testNewDb()

	type StructKey struct {
		ShopId int64
		Status int
	}

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewCustomTableRule(
			func(logicalTable string, key any) (string, error) {
				sk, ok := key.(StructKey)
				if !ok {
					return "", fmt.Errorf("expected StructKey key, got %T", key)
				}
				return fmt.Sprintf("%s_%08d_%02d_tab", logicalTable, sk.ShopId, sk.Status), nil
			},
		)),
	)

	// Find on the resolved executor
	shardedExec, err := executor.Shard(StructKey{ShopId: 12345, Status: 2})
	if err != nil {
		t.Fatalf("Shard failed: %v", err)
	}
	orders, err := shardedExec.Find(context.Background(), nil, nil)
	t.Logf("Example 10: Struct sharding: orders=%v, err=%v", orders, err)
}
