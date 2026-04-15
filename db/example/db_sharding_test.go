package example

import (
	"context"
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
 ...
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

func (*Order) DbKey() string {
	return "order_dbs"
}

func (*Order) IdFiledName() string {
	return "id"
}

// OrderFields provides type-safe column references for Order.
var OrderFields = struct {
	ShopID dbspi.Column
	Region dbspi.Column
}{
	ShopID: dbhelper.NewColumn("shop_id"),
	Region: dbhelper.NewColumn("region"),
}

// ==================== Example 1: Table-only sharding via Shard() ====================

func Test_Sharding_TableOnly_WithShard(t *testing.T) {
	ctx := context.Background()
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = hash(@{shop_id}) % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))

	shardedExec, err := executor.Shard(sk)
	if err != nil {
		t.Fatalf("Shard failed: %v", err)
	}

	order, err := shardedExec.GetById(ctx, 1001)
	t.Logf("Example 1a: GetById via Shard(): order=%v, err=%v", order, err)

	orders, err := shardedExec.Find(ctx, nil, nil)
	t.Logf("Example 1b: Find via Shard(): orders=%v, err=%v", orders, err)

	err = shardedExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})
	t.Logf("Example 1c: Create via Shard(): err=%v", err)
}

// ==================== Example 2: Table-only sharding via ctx ====================

func Test_Sharding_TableOnly_WithCtx(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 8)",
			"${idx} = hash(@{shop_id}) % 8",
			"${index} = fill(${idx}, 8)",
		)),
	)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

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
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 8)",
			"${idx} = hash(@{shop_id}) % 8",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()
	_, err := executor.GetById(ctx, 1001)
	t.Logf("Example 3: GetById without sharding key: err=%v", err)
}

// ==================== Example 4: Scatter-gather ====================

func Test_Sharding_FindAll(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.SingleDb(db)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = hash(@{shop_id}) % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	ctx := context.Background()

	allOrders, err := executor.FindAll(ctx, nil, 0)
	t.Logf("Example 4a: FindAll (no batch): count=%d, err=%v", len(allOrders), err)

	allOrders, err = executor.FindAll(ctx, nil, 100)
	t.Logf("Example 4b: FindAll (batch=100): count=%d, err=%v", len(allOrders), err)

	total, err := executor.CountAll(ctx, nil)
	t.Logf("Example 4c: CountAll: total=%d, err=%v", total, err)
}

// ==================== Example 5: Database + Table sharding ====================

func Test_Sharding_DbAndTable(t *testing.T) {
	db0 := testNewDb()
	db1 := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.IndexedDbs(db0, db1)),
		dbhelper.WithDbRule(dbhelper.NewExprDbRule(
			"${idx}",
			"${idx} := range(0, 2)",
			"${idx} = hash(@{shop_id}) % 2",
		)),
		dbhelper.WithTableRule(dbhelper.NewExprTableRule(
			"order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = hash(@{shop_id}) % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	order, err := executor.GetById(ctx, 1)
	t.Logf("Example 5: Db+Table sharding: order=%v, err=%v", order, err)
}

// ==================== Example 6: Region-based DB sharding ====================

func Test_Sharding_RegionDb(t *testing.T) {
	dbSG := testNewDb()
	dbTH := testNewDb()

	executor := dbhelper.NewShardedExecutor(&Order{},
		dbhelper.WithDbs(dbhelper.NamedDbs(map[string]dbspi.Db{
			"order_SG_db": dbSG,
			"order_TH_db": dbTH,
		})),
		dbhelper.WithDbRule(dbhelper.NewExprDbRule(
			"order_${region}_db",
			"${region} := enum(SG, TH)",
			"${region} = @{region}",
		)),
	)

	sk := dbspi.NewShardingKey().Set(OrderFields.Region, "SG")
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Example 6a: Region DB (SG): orders=%v, err=%v", orders, err)

	sk = dbspi.NewShardingKey().Set(OrderFields.Region, "TH")
	ctx = dbspi.WithShardingKey(context.Background(), sk)
	orders, err = executor.Find(ctx, nil, nil)
	t.Logf("Example 6b: Region DB (TH): orders=%v, err=%v", orders, err)
}

// ==================== Example 7: Non-sharded executor ====================

func Test_Sharding_NonShardedExecutor(t *testing.T) {
	db := testNewDb()

	executor := dbhelper.NewExecutor(db, &User{})

	ctx := context.Background()

	sameExec, err := executor.Shard(nil)
	t.Logf("Example 7a: Shard() on non-sharded: err=%v (should be nil)", err)

	users, err := sameExec.Find(ctx, nil, nil)
	t.Logf("Example 7b: Find via Shard() on non-sharded: users=%v, err=%v", users, err)

	allUsers, err := executor.FindAll(ctx, nil, 0)
	t.Logf("Example 7c: FindAll on non-sharded: users=%v, err=%v", allUsers, err)
}

// ==================== Example 8: Composite sharding key (db + table use different fields) ====================

func Test_Sharding_CompositeKey(t *testing.T) {
	dbSG := testNewDb()
	dbTH := testNewDb()

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
			"${idx} = hash(@{shop_id}) % 10",
			"${index} = fill(${idx}, 8)",
		)),
	)

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, "SG").
		Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Example 8: Composite key (region=SG, shop_id=12345): orders=%v, err=%v", orders, err)
}
