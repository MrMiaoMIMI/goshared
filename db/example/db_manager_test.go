package example

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ================================================================
// DbManager Example: Top-down configuration-driven database management
//
// One config → one DbManager → all executors.
// Entity declares its database group via DbKey().
// No need to manually create DbConfig, Db, DbTarget, or sharding rules.
// ================================================================

// ==================== Entity Definitions ====================

// User is a non-sharded entity. No DbKey() → uses "default" database.
// (User struct is defined in db_test.go)

// OrderItem shares the same database group as Order.
type OrderItem struct {
	ID      int64 `gorm:"primaryKey"`
	OrderID int64 `gorm:"column:order_id"`
	ShopID  int64 `gorm:"column:shop_id"`
	Name    string
}

func (*OrderItem) TableName() string   { return "order_item_tab" }
func (*OrderItem) DbKey() string       { return "order_dbs" }
func (*OrderItem) IdFiledName() string { return "id" }

// OrderDetail shares the same database group but has different table sharding.
type OrderDetail struct {
	ID      int64 `gorm:"primaryKey"`
	OrderID int64 `gorm:"column:order_id"`
	ShopID  int64 `gorm:"column:shop_id"`
	Detail  string
}

func (*OrderDetail) TableName() string   { return "order_detail_tab" }
func (*OrderDetail) DbKey() string       { return "order_dbs" }
func (*OrderDetail) IdFiledName() string { return "id" }

// ==================== DbManager: Non-sharded ====================

func Test_DbManager_Simple(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbhelper.DatabaseConfig{
		Databases: map[string]dbhelper.DatabaseEntry{
			"default": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbName: testAppDbName,
			},
		},
	})

	userExec := dbhelper.For(&User{}, mgr)

	ctx := context.Background()
	users, err := userExec.Find(ctx, nil, nil)
	t.Logf("DbManager simple: users=%v, err=%v", users, err)
}

// ==================== DbManager: DSN mode ====================

func Test_DbManager_DSN(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbhelper.DatabaseConfig{
		Databases: map[string]dbhelper.DatabaseEntry{
			"default": {
				DSN:          testDSN(testAppDbName),
				MaxOpenConns: 200,
			},
		},
	})

	userExec := dbhelper.For(&User{}, mgr)

	ctx := context.Background()
	users, err := userExec.Find(ctx, nil, nil)
	t.Logf("DbManager DSN: users=%v, err=%v", users, err)
}

// ==================== DbManager: Sharded with reuse ====================

func Test_DbManager_ShardedWithReuse(t *testing.T) {
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
				MaxConcurrency: 5,
				EntityRules: []dbhelper.EntityRule{
					{
						Tables: []string{"order_detail_tab"},
						TableSharding: &dbhelper.TableShardConfig{
							NameExpr:    "order_detail_tab_${index}",
							ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
						},
					},
				},
			},
		},
	})

	orderExec := dbhelper.For(&Order{}, mgr)
	itemExec := dbhelper.For(&OrderItem{}, mgr)
	detailExec := dbhelper.For(&OrderDetail{}, mgr)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	orders, err := orderExec.Find(ctx, nil, nil)
	t.Logf("Order (4×10): orders=%v, err=%v", orders, err)

	items, err := itemExec.Find(ctx, nil, nil)
	t.Logf("OrderItem (4×10, shared): items=%v, err=%v", items, err)

	details, err := detailExec.Find(ctx, nil, nil)
	t.Logf("OrderDetail (4×20, overridden): details=%v, err=%v", details, err)
}

// ==================== DbManager: Named db sharding ====================

func Test_DbManager_NamedDbSharding(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbhelper.DatabaseConfig{
		Databases: map[string]dbhelper.DatabaseEntry{
			"default": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbName: testAppDbName,
			},
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbSharding: &dbhelper.DbShardConfig{
					NameExpr:    "order_${region}_db",
					ExpandExprs: []string{"${region} := enum(SG, TH)", "${region} = @{region}"},
				},
				TableSharding: &dbhelper.TableShardConfig{
					NameExpr:    "order_tab_${index}",
					ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
				},
			},
		},
	})

	orderExec := dbhelper.For(&Order{}, mgr)

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, "SG").
		Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := orderExec.Find(ctx, nil, nil)
	t.Logf("Named db sharding (SG): orders=%v, err=%v", orders, err)
}

// ==================== DbManager: Global default ====================

func Test_DbManager_GlobalDefault(t *testing.T) {
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

	dbhelper.SetDefault(mgr)

	userExec := dbhelper.For(&User{})
	orderExec := dbhelper.For(&Order{})

	ctx := context.Background()
	users, err := userExec.Find(ctx, nil, nil)
	t.Logf("Global default - User: users=%v, err=%v", users, err)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx = dbspi.WithShardingKey(ctx, sk)
	orders, err := orderExec.Find(ctx, nil, nil)
	t.Logf("Global default - Order: orders=%v, err=%v", orders, err)
}
