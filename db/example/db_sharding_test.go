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

func (*Order) IdFieldName() string {
	return dbspi.DefaultIdFieldName
}

// OrderFields provides type-safe column references for Order.
var OrderFields = struct {
	ShopID dbspi.Field[int64]
	Region dbspi.Field[string]
}{
	ShopID: dbhelper.NewField[int64]("shop_id"),
	Region: dbhelper.NewField[string]("region"),
}

// ==================== Example 1: Table-only sharding via Shard() ====================

func Test_Sharding_TableOnly_WithShard(t *testing.T) {
	ctx := context.Background()
	executor := newOrderShopTableExecutor(10)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))

	shardedExec, err := executor.Shard(sk)
	if err != nil {
		t.Fatalf("Shard failed: %v", err)
	}

	order, err := shardedExec.GetById(ctx, 1001)
	requireNoError(t, err)
	t.Logf("Example 1a: GetById via Shard(): order=%v", order)

	orders, err := shardedExec.Find(ctx, nil, nil)
	requireNoError(t, err)
	t.Logf("Example 1b: Find via Shard(): orders=%v", orders)

	err = shardedExec.Create(ctx, &Order{ShopID: 12345, Amount: 100})
	requireNoError(t, err)
}

// ==================== Example 2: Table-only sharding via ctx ====================

func Test_Sharding_TableOnly_WithCtx(t *testing.T) {
	executor := newOrderShopTableExecutor(8)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	order, err := executor.GetById(ctx, 1001)
	requireNoError(t, err)
	t.Logf("Example 2a: GetById via ctx: order=%v", order)

	orders, err := executor.Find(ctx, nil, nil)
	requireNoError(t, err)
	t.Logf("Example 2b: Find via ctx: orders=%v", orders)

	err = executor.Create(ctx, &Order{ShopID: 12345, Amount: 200})
	requireNoError(t, err)
}

// ==================== Example 3: Missing sharding key ====================

func Test_Sharding_MissingKey(t *testing.T) {
	executor := newOrderShopTableExecutor(8)

	ctx := context.Background()
	_, err := executor.GetById(ctx, 1001)
	requireErrorContains(t, err, "missing")
}

// ==================== Example 4: Scatter-gather ====================

func Test_Sharding_FindAll(t *testing.T) {
	executor := newOrderShopTableExecutor(10)

	ctx := context.Background()

	allOrders, err := executor.FindAll(ctx, nil, 0)
	requireNoError(t, err)
	t.Logf("Example 4a: FindAll (no batch): count=%d", len(allOrders))

	allOrders, err = executor.FindAll(ctx, nil, 100)
	requireNoError(t, err)
	t.Logf("Example 4b: FindAll (batch=100): count=%d", len(allOrders))

	total, err := executor.CountAll(ctx, nil)
	requireNoError(t, err)
	t.Logf("Example 4c: CountAll: total=%d", total)
}

// ==================== Example 5: Database + Table sharding ====================

func Test_Sharding_DbAndTable(t *testing.T) {
	executor := newOrderDbTableExecutor()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	order, err := executor.GetById(ctx, 1)
	requireNoError(t, err)
	t.Logf("Example 5: Db+Table sharding: order=%v", order)
}

// ==================== Example 6: Region-based DB sharding ====================

func Test_Sharding_RegionDb(t *testing.T) {
	executor := newOrderRegionDbExecutor()

	sk := dbspi.NewShardingKey().Set(OrderFields.Region, "SG")
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	requireNoError(t, err)
	t.Logf("Example 6a: Region DB (SG): orders=%v", orders)

	sk = dbspi.NewShardingKey().Set(OrderFields.Region, "TH")
	ctx = dbspi.WithShardingKey(context.Background(), sk)
	orders, err = executor.Find(ctx, nil, nil)
	requireNoError(t, err)
	t.Logf("Example 6b: Region DB (TH): orders=%v", orders)
}

// ==================== Example 7: Non-sharded executor ====================

func Test_Sharding_NonShardedExecutor(t *testing.T) {
	executor := dbhelper.For(&User{}, dbhelper.WithDbManager(testDbManager(testDbName)))

	ctx := context.Background()

	sameExec, err := executor.Shard(nil)
	requireNoError(t, err)

	users, err := sameExec.Find(ctx, nil, nil)
	requireNoError(t, err)
	t.Logf("Example 7b: Find via Shard() on non-sharded: users=%v", users)

	allUsers, err := executor.FindAll(ctx, nil, 0)
	requireNoError(t, err)
	t.Logf("Example 7c: FindAll on non-sharded: users=%v", allUsers)
}

// ==================== Example 8: Composite sharding key (db + table use different fields) ====================

func Test_Sharding_CompositeKey(t *testing.T) {
	executor := newOrderRegionRequiredExecutor()

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, "SG").
		Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	orders, err := executor.Find(ctx, nil, nil)
	requireNoError(t, err)
	t.Logf("Example 8: Composite key (region=SG, shop_id=12345): orders=%v", orders)
}
