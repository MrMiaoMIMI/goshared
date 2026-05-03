package example

import (
	"fmt"
	"sync"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

var shardingTableStoreCache sync.Map

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

func managedTableStore[T dbspi.Entity](entity T, entry dbspi.DatabaseGroupConfig) dbspi.TableStore[T] {
	mgr := dbhelper.NewManager(dbspi.DatabaseConfig{
		DatabaseGroups: map[string]dbspi.DatabaseGroupConfig{
			dbspi.DefaultDatabaseGroupKey: defaultTestDatabaseGroupConfig(),
			"order_dbs":                   entry,
		},
	})
	return dbhelper.NewTableStore(entity, dbhelper.WithManager(mgr))
}

func cachedManagedTableStore[T dbspi.Entity](key string, entity T, entry dbspi.DatabaseGroupConfig) dbspi.TableStore[T] {
	if cached, ok := shardingTableStoreCache.Load(key); ok {
		return cached.(dbspi.TableStore[T])
	}
	store := managedTableStore(entity, entry)
	actual, _ := shardingTableStoreCache.LoadOrStore(key, store)
	return actual.(dbspi.TableStore[T])
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

func newOrderShopTableStore(count int) dbspi.TableStore[*Order] {
	return cachedManagedTableStore(fmt.Sprintf("order-shop-table:%d", count), &Order{}, orderShopTableEntry(count))
}

func newOrderIDTableStore(count int) dbspi.TableStore[*Order] {
	entry := singleTestDatabaseGroupConfig()
	entry.TableSharding = tableShardConfig(
		"order_tab_${index}",
		fmt.Sprintf("${idx} := range(0, %d)", count),
		fmt.Sprintf("${idx} = @{id} %% %d", count),
		"${index} = fill(${idx}, 8)",
	)
	return cachedManagedTableStore(fmt.Sprintf("order-id-table:%d", count), &Order{}, entry)
}

func newOrderDbTableStore() dbspi.TableStore[*Order] {
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
	return cachedManagedTableStore("order-db-table", &Order{}, entry)
}

func newOrderNamedDbTableStore() dbspi.TableStore[*Order] {
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
	return cachedManagedTableStore("order-named-db-table", &Order{}, entry)
}

func newOrderRegionDbTableStore() dbspi.TableStore[*Order] {
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
	return cachedManagedTableStore("order-region-db", &Order{}, entry)
}

func newRegionalOrderCompositeTableStore() dbspi.TableStore[*RegionalOrder] {
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
	return cachedManagedTableStore("regional-order-composite", &RegionalOrder{}, entry)
}

func newOrderRegionRequiredTableStore() dbspi.TableStore[*Order] {
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
	return cachedManagedTableStore("order-region-required", &Order{}, entry)
}

func newTableVarTableStore[T dbspi.Entity](entity T, count int) dbspi.TableStore[T] {
	entry := singleTestDatabaseGroupConfig()
	entry.TableSharding = tableShardConfig(
		"${table}_${index}",
		fmt.Sprintf("${idx} := range(0, %d)", count),
		fmt.Sprintf("${idx} = @{shop_id} %% %d", count),
		"${index} = fill(${idx}, 8)",
	)
	return cachedManagedTableStore(fmt.Sprintf("table-var:%s:%d", entity.TableName(), count), entity, entry)
}
