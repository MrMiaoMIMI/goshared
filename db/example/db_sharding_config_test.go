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
// BEFORE (verbose — many fields to remember):
//
//   executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
//       Server: &dbhelper.ServerConfig{Host: "10.0.0.1", Port: 3306, User: "root", Password: "pass", DbName: "order_db"},
//       Table:  &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
//   })
//
// AFTER (expression-based — self-documenting):
//
//   executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
//       Server: &dbhelper.ServerConfig{Host: "10.0.0.1", Port: 3306, User: "root", Password: "pass", DbName: "order_db"},
//       Table:  &dbhelper.TableShardConfig{
//           NameExpr: "order_tab_${index}",
//           ExpandExprs: []string{
//               "${idx} := range(0, 10)",
//               "${idx} = hash(@{shop_id}) % 10",
//               "${index} = fill(${idx}, 8)",
//           },
//       },
//   })
//
// AFTER (builder — one-liner):
//
//   executor := dbhelper.Sharded(&Order{}).
//       Server("10.0.0.1", 3306, "root", "pass", "order_db").
//       ExprTable("order_tab_${index}",
//           "${idx} := range(0, 10)",
//           "${idx} = hash(@{shop_id}) % 10",
//           "${index} = fill(${idx}, 8)").
//       Build()
//
// ================================================================

const (
	dbUserName  = "root"
	dbPassword  = "123456"
	dbHost      = "127.0.0.1"
	dbPort      = 3306
	orderDbName = "order_db"
	debugMode   = true
)

// ==================== Config-driven Examples ====================

func Test_Config_TableOnly(t *testing.T) {
	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
		Server: &dbhelper.ServerConfig{
			Host: dbHost, Port: dbPort, User: dbUserName, Password: dbPassword,
			DbName: orderDbName, Debug: true,
		},
		Table: &dbhelper.TableShardConfig{
			NameExpr:    "order_tab_${index}",
			ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = @{shop_id} % 10", "${index} = fill(${idx}, 8)"},
		},
	})

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config table-only: orders=%v, err=%v", orders, err)
}

func Test_Config_DbAndTable(t *testing.T) {
	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
		Server: &dbhelper.ServerConfig{
			Host: dbHost, Port: dbPort, User: dbUserName, Password: dbPassword,
			DbName: orderDbName, Debug: true,
		},
		Db: &dbhelper.DbShardConfig{
			NameExpr:    "order_db_${idx}",
			ExpandExprs: []string{"${idx} := range(0, 4)", "${idx} = hash(@{shop_id}) % 4"},
		},
		Table: &dbhelper.TableShardConfig{
			NameExpr:    "order_tab_${index}",
			ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = hash(@{shop_id}) % 10", "${index} = fill(${idx}, 8)"},
		},
	})

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
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
			NameExpr:    "order_${region}_db",
			ExpandExprs: []string{"${region} := enum(SG, TH, ID)", "${region} = @{region}"},
		},
		Table: &dbhelper.TableShardConfig{
			NameExpr:    "order_tab_${index}",
			ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = hash(@{shop_id}) % 10", "${index} = fill(${idx}, 8)"},
		},
	})

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, "SG").
		Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config named dbs: orders=%v, err=%v", orders, err)
}

func Test_Config_MultiServer(t *testing.T) {
	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
		Servers: []dbhelper.NamedServerConfig{
			{Key: "0", ServerConfig: dbhelper.ServerConfig{Host: "10.0.0.1", Port: 3306, User: "root", Password: "pass", DbName: "order_db_0"}},
			{Key: "1", ServerConfig: dbhelper.ServerConfig{Host: "10.0.0.2", Port: 3306, User: "root", Password: "pass", DbName: "order_db_1"}},
		},
		Db: &dbhelper.DbShardConfig{
			NameExpr:    "${idx}",
			ExpandExprs: []string{"${idx} := range(0, 2)", "${idx} = hash(@{shop_id}) % 2"},
		},
		Table: &dbhelper.TableShardConfig{
			NameExpr:    "order_tab_${index}",
			ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = hash(@{shop_id}) % 10", "${index} = fill(${idx}, 8)"},
		},
	})

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
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
		Table: &dbhelper.TableShardConfig{
			NameExpr:    "order_tab_${index}",
			ExpandExprs: []string{"${idx} := range(0, 10)", "${idx} = hash(@{shop_id}) % 10", "${index} = fill(${idx}, 8)"},
		},
	})

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config with conn pool: orders=%v, err=%v", orders, err)
}

// ==================== Builder Examples ====================

func Test_Builder_TableOnly(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass", "order_db").
		ExprTable("order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = hash(@{shop_id}) % 10",
			"${index} = fill(${idx}, 8)",
		).
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder table-only: orders=%v, err=%v", orders, err)
}

func Test_Builder_DbAndTable(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass").
		ExprDb("order_db_${idx}",
			"${idx} := range(0, 4)",
			"${idx} = hash(@{shop_id}) % 4",
		).
		ExprTable("order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = hash(@{shop_id}) % 10",
			"${index} = fill(${idx}, 8)",
		).
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder db+table: orders=%v, err=%v", orders, err)
}

func Test_Builder_NamedDbs(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass").
		ExprDb("order_${region}_db",
			"${region} := enum(SG, TH, ID)",
			"${region} = @{region}",
		).
		ExprTable("order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = hash(@{shop_id}) % 10",
			"${index} = fill(${idx}, 8)",
		).
		Build()

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, "SG").
		Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder named dbs: orders=%v, err=%v", orders, err)
}

func Test_Builder_WithOptions(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		Server("127.0.0.1", 3306, "root", "pass", "order_db").
		ExprTable("order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = hash(@{shop_id}) % 10",
			"${index} = fill(${idx}, 8)",
		).
		ConnPool(200, 20, 1800).
		MaxConcurrency(5).
		Debug().
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder with options: orders=%v, err=%v", orders, err)
}

func Test_Builder_MultiServer(t *testing.T) {
	executor := dbhelper.Sharded(&Order{}).
		AddServer("0", "10.0.0.1", 3306, "root", "pass", "order_db_0").
		AddServer("1", "10.0.0.2", 3306, "root", "pass", "order_db_1").
		ExprDb("${idx}",
			"${idx} := range(0, 2)",
			"${idx} = hash(@{shop_id}) % 2",
		).
		ExprTable("order_tab_${index}",
			"${idx} := range(0, 10)",
			"${idx} = hash(@{shop_id}) % 10",
			"${index} = fill(${idx}, 8)",
		).
		Build()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Builder multi-server: orders=%v, err=%v", orders, err)
}
