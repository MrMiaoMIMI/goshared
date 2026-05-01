package example

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// Db sharding should be configured through DbManager:
//
//	mgr := dbhelper.NewDbManager(dbspi.DatabaseConfig{
//	    Databases: map[string]dbspi.DatabaseEntry{
//	        "order_dbs": {
//	            Host: testDbHost, Port: testDbPort, User: testDbUser, Password: testDbPassword,
//	            DbSharding: &dbspi.DbShardConfig{...},
//	            TableSharding: &dbspi.TableShardConfig{...},
//	        },
//	    },
//	})
//	executor := dbhelper.For(&Order{}, mgr)

func Test_Config_TableOnly(t *testing.T) {
	executor := newOrderShopTableExecutor(10)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config table-only: orders=%v, err=%v", orders, err)
}

func Test_Config_DbAndTable(t *testing.T) {
	executor := newOrderDbTableExecutor()

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config db+table: orders=%v, err=%v", orders, err)
}

func Test_Config_NamedDbs(t *testing.T) {
	executor := newOrderNamedDbTableExecutor()

	sk := dbspi.NewShardingKey().
		Set(OrderFields.Region, "SG").
		Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config named dbs: orders=%v, err=%v", orders, err)
}

func Test_Config_MultiServer(t *testing.T) {
	entry := orderShopTableEntry(10)
	entry.Servers = []dbspi.NamedDbServerConfig{
		testNamedServer("0"),
		testNamedServer("1"),
	}
	entry.DbName = ""
	entry.DbSharding = dbShardConfig(
		"${idx}",
		"${idx} := range(0, 2)",
		"${idx} = @{shop_id} % 2",
	)
	executor := managedExecutor(&Order{}, entry)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config multi-server: orders=%v, err=%v", orders, err)
}

func Test_Config_WithConnPool(t *testing.T) {
	entry := orderShopTableEntry(10)
	entry.MaxOpenConns = 200
	entry.MaxIdleConns = 20
	entry.ConnMaxLifetimeSeconds = 1800
	entry.MaxConcurrency = 5
	executor := managedExecutor(&Order{}, entry)

	sk := dbspi.NewShardingKey().Set(OrderFields.ShopID, int64(12345))
	ctx := dbspi.WithShardingKey(context.Background(), sk)
	orders, err := executor.Find(ctx, nil, nil)
	t.Logf("Config with conn pool: orders=%v, err=%v", orders, err)
}
