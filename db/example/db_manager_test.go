package example

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ================================================================
// Manager Example: Top-down configuration-driven database management
//
// One config → one Manager → all executors.
// Entity declares its database group via DatabaseGroupKey().
// No need to manually create database sessions or sharding rules.
// ================================================================

// ==================== Entity Definitions ====================

// User is a non-sharded entity. No DatabaseGroupKey() uses dbspi.DefaultDatabaseGroupKey.
// (User struct is defined in db_test.go)

// OrderItem shares the same database group as Order.
type OrderItem struct {
	ID      int64 `gorm:"primaryKey"`
	OrderID int64 `gorm:"column:order_id"`
	ShopID  int64 `gorm:"column:shop_id"`
	Name    string
}

func (*OrderItem) TableName() string        { return "order_item_tab" }
func (*OrderItem) DatabaseGroupKey() string { return "order_dbs" }
func (*OrderItem) IdFieldName() string      { return dbspi.DefaultIdFieldName }

// OrderDetail shares the same database group but has different table sharding.
type OrderDetail struct {
	ID      int64 `gorm:"primaryKey"`
	OrderID int64 `gorm:"column:order_id"`
	ShopID  int64 `gorm:"column:shop_id"`
	Detail  string
}

func (*OrderDetail) TableName() string        { return "order_detail_tab" }
func (*OrderDetail) DatabaseGroupKey() string { return "order_dbs" }
func (*OrderDetail) IdFieldName() string      { return dbspi.DefaultIdFieldName }

// ==================== Manager: Non-sharded ====================

func Test_Manager_Simple(t *testing.T) {
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DatabaseName: testAppDatabaseName,
			},
		},
	})

	userExec := dbhelper.NewExecutor(&User{}, dbhelper.WithManager(mgr))

	ctx := context.Background()
	users, err := userExec.Find(ctx, nil, nil)
	requireNoError(t, err)
	if len(users) == 0 {
		t.Fatal("expected users from default database")
	}
}

func Test_Manager_NewEnhancedExecutor_Simple(t *testing.T) {
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DatabaseName: testAppDatabaseName,
			},
		},
	})

	userExec := dbhelper.NewEnhancedExecutor(&User{}, dbhelper.WithManager(mgr))

	ctx := context.Background()
	count, err := userExec.CountNotDeleted(ctx, nil)
	requireNoError(t, err)
	if count == 0 {
		t.Fatal("expected non-deleted user count")
	}
}

// ==================== Manager: DSN mode ====================

func Test_Manager_DSN(t *testing.T) {
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: {
				DSN:          testDSN(testAppDatabaseName),
				MaxOpenConns: 200,
			},
		},
	})

	userExec := dbhelper.NewExecutor(&User{}, dbhelper.WithManager(mgr))

	ctx := context.Background()
	users, err := userExec.Find(ctx, nil, nil)
	requireNoError(t, err)
	if len(users) == 0 {
		t.Fatal("expected users from DSN database")
	}
}

// ==================== Manager: Sharded with reuse ====================

func Test_Manager_ShardedWithReuse(t *testing.T) {
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DatabaseName: testAppDatabaseName,
			},
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DatabaseSharding: &dbspi.DatabaseShardingConfig{
					NameExpr:    "order_db_${idx}",
					ExpandExprs: []string{"${idx} := range(0, 2)", "${idx} = @{shop_id} % 2"},
				},
				TableSharding: &dbspi.TableShardingConfig{
					NameExpr:    "order_tab_${index}",
					ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
				},
				MaxConcurrency: 5,
				TableRules: []dbspi.TableShardingRuleConfig{
					{
						Tables: []string{"order_detail_tab"},
						TableSharding: &dbspi.TableShardingConfig{
							NameExpr:    "order_detail_tab_${index}",
							ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
						},
					},
				},
			},
		},
	})

	orderExec := dbhelper.NewExecutor(&Order{}, dbhelper.WithManager(mgr))
	itemExec := dbhelper.NewExecutor(&OrderItem{}, dbhelper.WithManager(mgr))
	detailExec := dbhelper.NewExecutor(&OrderDetail{}, dbhelper.WithManager(mgr))

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)

	_, err := orderExec.Find(ctx, nil, nil)
	requireNoError(t, err)

	_, err = itemExec.Find(ctx, nil, nil)
	requireNoError(t, err)

	_, err = detailExec.Find(ctx, nil, nil)
	requireNoError(t, err)
}

// ==================== Manager: Named db sharding ====================

func Test_Manager_NamedDatabaseSharding(t *testing.T) {
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DatabaseName: testAppDatabaseName,
			},
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DatabaseSharding: &dbspi.DatabaseShardingConfig{
					NameExpr:    "order_${region}_db",
					ExpandExprs: []string{"${region} := enum(SG, TH)", "${region} = @{region}"},
				},
				TableSharding: &dbspi.TableShardingConfig{
					NameExpr:    "order_tab_${index}",
					ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
				},
			},
		},
	})

	orderExec := dbhelper.NewExecutor(&Order{}, dbhelper.WithManager(mgr))

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, "SG").
		Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	_, err := orderExec.Find(ctx, nil, nil)
	requireNoError(t, err)
}

// ==================== Manager: Global default ====================

func Test_Manager_GlobalDefault(t *testing.T) {
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DatabaseName: testAppDatabaseName,
			},
			"order_dbs": {
				Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
				DatabaseSharding: &dbspi.DatabaseShardingConfig{
					NameExpr:    "order_db_${idx}",
					ExpandExprs: []string{"${idx} := range(0, 2)", "${idx} = @{shop_id} % 2"},
				},
				TableSharding: &dbspi.TableShardingConfig{
					NameExpr:    "order_tab_${index}",
					ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
				},
			},
		},
	})

	dbhelper.SetDefaultManager(mgr)

	userExec := dbhelper.NewExecutor(&User{})
	orderExec := dbhelper.NewExecutor(&Order{})

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
