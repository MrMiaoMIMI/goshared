package example

import (
	"fmt"
	"sync"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

var shardingExecutorCache sync.Map

func defaultTestDatabaseEntry() dbspi.DatabaseEntry {
	return dbspi.DatabaseEntry{
		Host:         testDbHost,
		Port:         testDbPort,
		User:         testDbUser,
		Password:     testDbPassword,
		DbName:       testAppDbName,
		Debug:        testDebugMode,
		MaxOpenConns: 2,
		MaxIdleConns: 1,
	}
}

func singleTestDatabaseEntry() dbspi.DatabaseEntry {
	return dbspi.DatabaseEntry{
		Host:         testDbHost,
		Port:         testDbPort,
		User:         testDbUser,
		Password:     testDbPassword,
		DbName:       testDbName,
		Debug:        testDebugMode,
		MaxOpenConns: 2,
		MaxIdleConns: 1,
	}
}

func testNamedServer(key string) dbspi.NamedDbServerConfig {
	return dbspi.NamedDbServerConfig{
		Key: key,
		DbServerConfig: dbspi.DbServerConfig{
			Host:     testDbHost,
			Port:     testDbPort,
			User:     testDbUser,
			Password: testDbPassword,
			DbName:   testDbName,
			Debug:    testDebugMode,

			MaxOpenConns: 2,
			MaxIdleConns: 1,
		},
	}
}

func managedExecutor[T dbspi.Entity](entity T, entry dbspi.DatabaseEntry) dbspi.Executor[T] {
	mgr := dbhelper.NewDbManager(dbspi.DatabaseConfig{
		Databases: map[string]dbspi.DatabaseEntry{
			dbspi.DefaultDbKey: defaultTestDatabaseEntry(),
			"order_dbs":        entry,
		},
	})
	return dbhelper.For(entity, dbhelper.WithDbManager(mgr))
}

func cachedManagedExecutor[T dbspi.Entity](key string, entity T, entry dbspi.DatabaseEntry) dbspi.Executor[T] {
	if cached, ok := shardingExecutorCache.Load(key); ok {
		return cached.(dbspi.Executor[T])
	}
	exec := managedExecutor(entity, entry)
	actual, _ := shardingExecutorCache.LoadOrStore(key, exec)
	return actual.(dbspi.Executor[T])
}

func tableShardConfig(nameExpr string, expandExprs ...string) *dbspi.TableShardConfig {
	return &dbspi.TableShardConfig{
		NameExpr:    nameExpr,
		ExpandExprs: expandExprs,
	}
}

func dbShardConfig(nameExpr string, expandExprs ...string) *dbspi.DbShardConfig {
	return &dbspi.DbShardConfig{
		NameExpr:    nameExpr,
		ExpandExprs: expandExprs,
	}
}

func orderShopTableEntry(count int) dbspi.DatabaseEntry {
	entry := singleTestDatabaseEntry()
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
	entry := singleTestDatabaseEntry()
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
	return cachedManagedExecutor("order-db-table", &Order{}, entry)
}

func newOrderNamedDbTableExecutor() dbspi.Executor[*Order] {
	entry := orderShopTableEntry(10)
	entry.Servers = []dbspi.NamedDbServerConfig{
		testNamedServer("order_SG_db"),
		testNamedServer("order_TH_db"),
	}
	entry.DbName = ""
	entry.DbSharding = dbShardConfig(
		"order_${region}_db",
		"${region} := enum(SG, TH)",
		"${region} = @{region}",
	)
	return cachedManagedExecutor("order-named-db-table", &Order{}, entry)
}

func newOrderRegionDbExecutor() dbspi.Executor[*Order] {
	entry := singleTestDatabaseEntry()
	entry.Servers = []dbspi.NamedDbServerConfig{
		testNamedServer("order_SG_db"),
		testNamedServer("order_TH_db"),
	}
	entry.DbName = ""
	entry.DbSharding = dbShardConfig(
		"order_${region}_db",
		"${region} := enum(SG, TH)",
		"${region} = @{region}",
	)
	return cachedManagedExecutor("order-region-db", &Order{}, entry)
}

func newRegionalOrderCompositeExecutor() dbspi.Executor[*RegionalOrder] {
	entry := singleTestDatabaseEntry()
	entry.Servers = []dbspi.NamedDbServerConfig{
		testNamedServer("SG"),
		testNamedServer("TH"),
	}
	entry.DbName = ""
	entry.DbSharding = dbShardConfig(
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
	entry := singleTestDatabaseEntry()
	entry.Servers = []dbspi.NamedDbServerConfig{
		testNamedServer("SG"),
		testNamedServer("TH"),
	}
	entry.DbName = ""
	entry.DbSharding = dbShardConfig(
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
	entry := singleTestDatabaseEntry()
	entry.TableSharding = tableShardConfig(
		"${table}_${index}",
		fmt.Sprintf("${idx} := range(0, %d)", count),
		fmt.Sprintf("${idx} = @{shop_id} %% %d", count),
		"${index} = fill(${idx}, 8)",
	)
	return cachedManagedExecutor(fmt.Sprintf("table-var:%s:%d", entity.TableName(), count), entity, entry)
}
