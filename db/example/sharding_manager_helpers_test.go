package example

import (
	"fmt"
	"sync"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

var shardingExecutorCache sync.Map

func defaultTestDatabaseGroupConfig() dbspi.DatabaseGroupConfig {
	return dbspi.DatabaseGroupConfig{
		Host:         testDbHost,
		Port:         testDbPort,
		User:         testDbUser,
		Password:     testDbPassword,
		DatabaseName: testAppDatabaseName,
		Debug:        testDebugMode,
		MaxOpenConns: 2,
		MaxIdleConns: 1,
	}
}

func singleTestDatabaseGroupConfig() dbspi.DatabaseGroupConfig {
	return dbspi.DatabaseGroupConfig{
		Host:         testDbHost,
		Port:         testDbPort,
		User:         testDbUser,
		Password:     testDbPassword,
		DatabaseName: testDatabaseName,
		Debug:        testDebugMode,
		MaxOpenConns: 2,
		MaxIdleConns: 1,
	}
}

func testNamedServer(key string) dbspi.NamedServerConfig {
	return dbspi.NamedServerConfig{
		Key: key,
		ServerConfig: dbspi.ServerConfig{
			Host:         testDbHost,
			Port:         testDbPort,
			User:         testDbUser,
			Password:     testDbPassword,
			DatabaseName: testDatabaseName,
			Debug:        testDebugMode,

			MaxOpenConns: 2,
			MaxIdleConns: 1,
		},
	}
}

func managedExecutor[T dbspi.Entity](entity T, entry dbspi.DatabaseGroupConfig) dbspi.Executor[T] {
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: defaultTestDatabaseGroupConfig(),
			"order_dbs":                   entry,
		},
	})
	return dbhelper.NewExecutor(entity, dbhelper.WithManager(mgr))
}

func cachedManagedExecutor[T dbspi.Entity](key string, entity T, entry dbspi.DatabaseGroupConfig) dbspi.Executor[T] {
	if cached, ok := shardingExecutorCache.Load(key); ok {
		return cached.(dbspi.Executor[T])
	}
	exec := managedExecutor(entity, entry)
	actual, _ := shardingExecutorCache.LoadOrStore(key, exec)
	return actual.(dbspi.Executor[T])
}

func tableShardConfig(nameExpr string, expandExprs ...string) *dbspi.TableShardingConfig {
	return &dbspi.TableShardingConfig{
		NameExpr:    nameExpr,
		ExpandExprs: expandExprs,
	}
}

func dbShardConfig(nameExpr string, expandExprs ...string) *dbspi.DatabaseShardingConfig {
	return &dbspi.DatabaseShardingConfig{
		NameExpr:    nameExpr,
		ExpandExprs: expandExprs,
	}
}

func orderShopTableEntry(count int) dbspi.DatabaseGroupConfig {
	entry := singleTestDatabaseGroupConfig()
	entry.TableSharding = tableShardConfig(
		"order_tab_${index}",
		fmt.Sprintf("${idx} := range(0, %d)", count),
		fmt.Sprintf("${idx} = @{shop_id} %% %d", count),
		"${index} = fill(${idx}, 8)",
	)
	return entry
}

func newOrderShopTableExecutor(count int) dbspi.Executor[*Order] {
	return cachedManagedExecutor(fmt.Sprintf("order-shop-table:%d", count), &Order{}, orderShopTableEntry(count))
}

func newOrderIDTableExecutor(count int) dbspi.Executor[*Order] {
	entry := singleTestDatabaseGroupConfig()
	entry.TableSharding = tableShardConfig(
		"order_tab_${index}",
		fmt.Sprintf("${idx} := range(0, %d)", count),
		fmt.Sprintf("${idx} = @{id} %% %d", count),
		"${index} = fill(${idx}, 8)",
	)
	return cachedManagedExecutor(fmt.Sprintf("order-id-table:%d", count), &Order{}, entry)
}

func newOrderDbTableExecutor() dbspi.Executor[*Order] {
	entry := orderShopTableEntry(10)
	entry.Servers = []dbspi.NamedServerConfig{
		testNamedServer("0"),
		testNamedServer("1"),
	}
	entry.DatabaseName = ""
	entry.DatabaseSharding = dbShardConfig(
		"${idx}",
		"${idx} := range(0, 2)",
		"${idx} = @{shop_id} % 2",
	)
	return cachedManagedExecutor("order-db-table", &Order{}, entry)
}

func newOrderNamedDbTableExecutor() dbspi.Executor[*Order] {
	entry := orderShopTableEntry(10)
	entry.Servers = []dbspi.NamedServerConfig{
		testNamedServer("order_SG_db"),
		testNamedServer("order_TH_db"),
	}
	entry.DatabaseName = ""
	entry.DatabaseSharding = dbShardConfig(
		"order_${region}_db",
		"${region} := enum(SG, TH)",
		"${region} = @{region}",
	)
	return cachedManagedExecutor("order-named-db-table", &Order{}, entry)
}

func newOrderRegionDbExecutor() dbspi.Executor[*Order] {
	entry := singleTestDatabaseGroupConfig()
	entry.Servers = []dbspi.NamedServerConfig{
		testNamedServer("order_SG_db"),
		testNamedServer("order_TH_db"),
	}
	entry.DatabaseName = ""
	entry.DatabaseSharding = dbShardConfig(
		"order_${region}_db",
		"${region} := enum(SG, TH)",
		"${region} = @{region}",
	)
	return cachedManagedExecutor("order-region-db", &Order{}, entry)
}

func newRegionalOrderCompositeExecutor() dbspi.Executor[*RegionalOrder] {
	entry := singleTestDatabaseGroupConfig()
	entry.Servers = []dbspi.NamedServerConfig{
		testNamedServer("SG"),
		testNamedServer("TH"),
	}
	entry.DatabaseName = ""
	entry.DatabaseSharding = dbShardConfig(
		"${region}",
		"${region} := enum(SG, TH)",
		"${region} = @{region}",
	)
	entry.TableSharding = tableShardConfig(
		"order_tab_${index}",
		"${idx} := range(0, 10)",
		"${idx} = @{shop_id} % 10",
		"${index} = fill(${idx}, 8)",
	)
	return cachedManagedExecutor("regional-order-composite", &RegionalOrder{}, entry)
}

func newOrderRegionRequiredExecutor() dbspi.Executor[*Order] {
	entry := singleTestDatabaseGroupConfig()
	entry.Servers = []dbspi.NamedServerConfig{
		testNamedServer("SG"),
		testNamedServer("TH"),
	}
	entry.DatabaseName = ""
	entry.DatabaseSharding = dbShardConfig(
		"${region}",
		"${region} := enum(SG, TH)",
		"${region} = @{region}",
	)
	entry.TableSharding = tableShardConfig(
		"order_tab_${index}",
		"${idx} := range(0, 10)",
		"${idx} = @{shop_id} % 10",
		"${index} = fill(${idx}, 8)",
	)
	return cachedManagedExecutor("order-region-required", &Order{}, entry)
}

func newTableVarExecutor[T dbspi.Entity](entity T, count int) dbspi.Executor[T] {
	entry := singleTestDatabaseGroupConfig()
	entry.TableSharding = tableShardConfig(
		"${table}_${index}",
		fmt.Sprintf("${idx} := range(0, %d)", count),
		fmt.Sprintf("${idx} = @{shop_id} %% %d", count),
		"${index} = fill(${idx}, 8)",
	)
	return cachedManagedExecutor(fmt.Sprintf("table-var:%s:%d", entity.TableName(), count), entity, entry)
}
