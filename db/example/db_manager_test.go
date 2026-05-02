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
// No need to manually create Db, DbTarget, or sharding rules.
// ================================================================

// ==================== Entity Definitions ====================

// User is a non-sharded entity. No DbKey() uses dbspi.DefaultDbKey.
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
func (*OrderItem) IdFieldName() string { return dbspi.DefaultIdFieldName }

// OrderDetail shares the same database group but has different table sharding.
type OrderDetail struct {
	ID      int64 `gorm:"primaryKey"`
	OrderID int64 `gorm:"column:order_id"`
	ShopID  int64 `gorm:"column:shop_id"`
	Detail  string
}

func (*OrderDetail) TableName() string   { return "order_detail_tab" }
func (*OrderDetail) DbKey() string       { return "order_dbs" }
func (*OrderDetail) IdFieldName() string { return dbspi.DefaultIdFieldName }

// ==================== DbManager: Non-sharded ====================

func Test_DbManager_Simple(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbspi.DatabaseConfig{
		Databases: map[string]dbspi.DatabaseEntry{
			dbspi.DefaultDbKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbName: testAppDbName,
			},
		},
	})

	userExec := dbhelper.For(&User{}, dbhelper.WithDbManager(mgr))

	ctx := context.Background()
	users, err := userExec.Find(ctx, nil, nil)
	requireNoError(t, err)
	if len(users) == 0 {
		t.Fatal("expected users from default database")
	}
}

func Test_DbManager_ForEnhance_Simple(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbspi.DatabaseConfig{
		Databases: map[string]dbspi.DatabaseEntry{
			dbspi.DefaultDbKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbName: testAppDbName,
			},
		},
	})

	userExec := dbhelper.ForEnhance(&User{}, dbhelper.WithDbManager(mgr))

	ctx := context.Background()
	count, err := userExec.CountWithoutDeleted(ctx, nil)
	requireNoError(t, err)
	if count == 0 {
		t.Fatal("expected non-deleted user count")
	}
}

// ==================== DbManager: DSN mode ====================

func Test_DbManager_DSN(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbspi.DatabaseConfig{
		Databases: map[string]dbspi.DatabaseEntry{
			dbspi.DefaultDbKey: {
				DSN:          testDSN(testAppDbName),
				MaxOpenConns: 200,
			},
		},
	})

	userExec := dbhelper.For(&User{}, dbhelper.WithDbManager(mgr))

	ctx := context.Background()
	users, err := userExec.Find(ctx, nil, nil)
	requireNoError(t, err)
	if len(users) == 0 {
		t.Fatal("expected users from DSN database")
	}
}

// ==================== DbManager: Sharded with reuse ====================

func Test_DbManager_ShardedWithReuse(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbspi.DatabaseConfig{
		Databases: map[string]dbspi.DatabaseEntry{
			dbspi.DefaultDbKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbName: testAppDbName,
			},
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbSharding: &dbspi.DbShardConfig{
					NameExpr:    "order_db_${idx}",
					ExpandExprs: []string{"${idx} := range(0, 2)", "${idx} = @{shop_id} % 2"},
				},
				TableSharding: &dbspi.TableShardConfig{
					NameExpr:    "order_tab_${index}",
					ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
				},
				MaxConcurrency: 5,
				EntityRules: []dbspi.EntityRule{
					{
						Tables: []string{"order_detail_tab"},
						TableSharding: &dbspi.TableShardConfig{
							NameExpr:    "order_detail_tab_${index}",
							ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
						},
					},
				},
			},
		},
	})

	orderExec := dbhelper.For(&Order{}, dbhelper.WithDbManager(mgr))
	itemExec := dbhelper.For(&OrderItem{}, dbhelper.WithDbManager(mgr))
	detailExec := dbhelper.For(&OrderDetail{}, dbhelper.WithDbManager(mgr))

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	_, err := orderExec.Find(ctx, nil, nil)
	requireNoError(t, err)

	_, err = itemExec.Find(ctx, nil, nil)
	requireNoError(t, err)

	_, err = detailExec.Find(ctx, nil, nil)
	requireNoError(t, err)
}

// ==================== DbManager: Named db sharding ====================

func Test_DbManager_NamedDbSharding(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbspi.DatabaseConfig{
		Databases: map[string]dbspi.DatabaseEntry{
			dbspi.DefaultDbKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbName: testAppDbName,
			},
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbSharding: &dbspi.DbShardConfig{
					NameExpr:    "order_${region}_db",
					ExpandExprs: []string{"${region} := enum(SG, TH)", "${region} = @{region}"},
				},
				TableSharding: &dbspi.TableShardConfig{
					NameExpr:    "order_tab_${index}",
					ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
				},
			},
		},
	})

	orderExec := dbhelper.For(&Order{}, dbhelper.WithDbManager(mgr))

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, "SG").
		Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	_, err := orderExec.Find(ctx, nil, nil)
	requireNoError(t, err)
}

// ==================== DbManager: Global default ====================

func Test_DbManager_GlobalDefault(t *testing.T) {
	mgr := dbhelper.NewDbManager(dbspi.DatabaseConfig{
		Databases: map[string]dbspi.DatabaseEntry{
			dbspi.DefaultDbKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbName: testAppDbName,
			},
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DbSharding: &dbspi.DbShardConfig{
					NameExpr:    "order_db_${idx}",
					ExpandExprs: []string{"${idx} := range(0, 2)", "${idx} = @{shop_id} % 2"},
				},
				TableSharding: &dbspi.TableShardConfig{
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
	requireNoError(t, err)
	if len(users) == 0 {
		t.Fatal("expected users from global default manager")
	}

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx = dbspi.WithShardingKey(ctx, sk)
	_, err = orderExec.Find(ctx, nil, nil)
	requireNoError(t, err)
}
