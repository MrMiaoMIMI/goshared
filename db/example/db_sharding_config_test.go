package example

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ================================================================
// Before vs After comparison:
//
// BEFORE (verbose — requires understanding DbTarget, DbConfig, ShardingRule):
//
//   dbConfig := dbhelper.NewDbConfig("10.0.0.1", 3306, "root", "pass", "order_db")
//   db := dbhelper.NewDb(dbConfig)
//   executor := dbhelper.NewShardedExecutor(&Order{},
//       dbhelper.WithDbs(dbhelper.SingleDb(db)),
//       dbhelper.WithTableRule(dbhelper.NewHashModTableRule(10)),
//       dbhelper.WithTableKeyField("shop_id"),
//   )
//
// AFTER (config — just fill a struct):
//
//   executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
//       Server: &dbhelper.ServerConfig{Host: "10.0.0.1", Port: 3306, User: "root", Password: "pass", DbName: "order_db"},
//       Table:  &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
//   })
//
// AFTER (builder — one-liner):
//
//   executor := dbhelper.Sharded(&Order{}).
//       Server("10.0.0.1", 3306, "root", "pass", "order_db").
//       HashModTable("shop_id", 10).
//       Build()
//
// ================================================================

// ==================== Config-driven Examples ====================

func Test_Config_TableOnly(t *testing.T) {
	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
		Server: &dbhelper.ServerConfig{
			Host: "127.0.0.1", Port: 3306, User: "root", Password: "pass",
			DbName: "order_db",
		},
		Table: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
	})

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config table-only: orders=%v, err=%v", orders, err)
}

func Test_Config_DbAndTable(t *testing.T) {
	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
		Server: &dbhelper.ServerConfig{
			Host: "127.0.0.1", Port: 3306, User: "root", Password: "pass",
		},
		Db:    &dbhelper.DbShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 4, Prefix: "order_db"},
		Table: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
	})

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config db+table: orders=%v, err=%v", orders, err)
}

func Test_Config_NamedDbs(t *testing.T) {
	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
		Server: &dbhelper.ServerConfig{
			Host: "127.0.0.1", Port: 3306, User: "root", Password: "pass",
		},
		Db: &dbhelper.DbShardConfig{
			Rule: "named", KeyField: "region", Prefix: "order_", Suffix: "_db",
			Keys: []string{"SG", "TH", "ID"},
		},
		Table: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
	})

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, dbspi.StrVal("SG")).
		Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config named dbs: orders=%v, err=%v", orders, err)
}

func Test_Config_RangeDb(t *testing.T) {
	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
		Server: &dbhelper.ServerConfig{
			Host: "127.0.0.1", Port: 3306, User: "root", Password: "pass",
		},
		Db: &dbhelper.DbShardConfig{
			Rule: "range", KeyField: "shop_id", Prefix: "order_db",
			Boundaries: []int64{10000, 20000},
		},
		Table: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
	})

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(5000))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config range db: orders=%v, err=%v", orders, err)
}

func Test_Config_MultiServer(t *testing.T) {
	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
		Servers: []dbhelper.NamedServerConfig{
			{Key: "0", ServerConfig: dbhelper.ServerConfig{Host: "10.0.0.1", Port: 3306, User: "root", Password: "pass", DbName: "order_db_0"}},
			{Key: "1", ServerConfig: dbhelper.ServerConfig{Host: "10.0.0.2", Port: 3306, User: "root", Password: "pass", DbName: "order_db_1"}},
		},
		Db:    &dbhelper.DbShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 2},
		Table: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
	})

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config multi-server: orders=%v, err=%v", orders, err)
}

func Test_Config_WithConnPool(t *testing.T) {
	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
		Server: &dbhelper.ServerConfig{
			Host: "127.0.0.1", Port: 3306, User: "root", Password: "pass",
			DbName:                 "order_db",
			MaxOpenConns:           200,
			MaxIdleConns:           20,
			ConnMaxLifetimeSeconds: 1800,
			Debug:                  true,
		},
		Table: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
	})

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config with conn pool: orders=%v, err=%v", orders, err)
}

// ==================== Builder Examples ====================

func Test_Builder_TableOnly(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass", "order_db").
		HashModTable("shop_id", 10).
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder table-only: orders=%v, err=%v", orders, err)
}

func Test_Builder_DbAndTable(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass").
		HashModDb("shop_id", "order_db", 4).
		HashModTable("shop_id", 10).
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder db+table: orders=%v, err=%v", orders, err)
}

func Test_Builder_NamedDbs(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass").
		NamedDbs("region", "order_", "_db", "SG", "TH", "ID").
		HashModTable("shop_id", 10).
		Build()

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, dbspi.StrVal("SG")).
		Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder named dbs: orders=%v, err=%v", orders, err)
}

func Test_Builder_RangeDb(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass").
		RangeDb("shop_id", "order_db", 10000, 20000).
		HashModTable("shop_id", 10).
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(5000))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder range db: orders=%v, err=%v", orders, err)
}

func Test_Builder_WithOptions(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass", "order_db").
		HashModTable("shop_id", 10).
		ConnPool(200, 20, 1800).
		MaxConcurrency(5).
		Debug().
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder with options: orders=%v, err=%v", orders, err)
}

func Test_Builder_CustomTableFormat(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass", "order_db").
		HashModTableWithFormat("shop_id", 10, "_%02d").
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder custom format: orders=%v, err=%v", orders, err)
}

func Test_Builder_MultiServer(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		AddServer("0", "10.0.0.1", 3306, "root", "pass", "order_db_0").
		AddServer("1", "10.0.0.2", 3306, "root", "pass", "order_db_1").
		HashModDb("shop_id", "", 2).
		HashModTable("shop_id", 10).
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, dbspi.IntVal(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder multi-server: orders=%v, err=%v", orders, err)
}
