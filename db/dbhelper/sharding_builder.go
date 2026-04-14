package dbhelper

import (
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ShardedBuilder provides a fluent API for constructing sharded executors.
// It builds a ShardingConfig internally and delegates to NewShardedExecutorFromConfig.
//
// Example (table-only sharding — most common):
//
//	executor := dbhelper.Sharded(&Order{}).
//	    Server("10.0.0.1", 3306, "root", "pass", "order_db").
//	    HashModTable("shop_id", 10).
//	    Build()
//
// Example (db + table sharding):
//
//	executor := dbhelper.Sharded(&Order{}).
//	    Server("10.0.0.1", 3306, "root", "pass").
//	    HashModDb("shop_id", "order_db", 4).
//	    HashModTable("shop_id", 10).
//	    Build()
//
// Example (named db sharding by region):
//
//	executor := dbhelper.Sharded(&Order{}).
//	    Server("10.0.0.1", 3306, "root", "pass").
//	    NamedDbs("region", "order_", "_db", "SG", "TH", "ID").
//	    HashModTable("shop_id", 10).
//	    Build()
type ShardedBuilder[T dbspi.Entity] struct {
	entity T
	cfg    ShardingConfig
}

// Sharded starts building a sharded executor for the given entity.
func Sharded[T dbspi.Entity](entity T) *ShardedBuilder[T] {
	return &ShardedBuilder[T]{entity: entity}
}

// Server sets the database server connection.
// For table-only sharding, provide the database name as the last argument.
// For db sharding, omit the database name (it comes from HashModDb/NamedDbs/RangeDb).
func (b *ShardedBuilder[T]) Server(host string, port uint, user, password string, dbName ...string) *ShardedBuilder[T] {
	b.cfg.Server = &ServerConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}
	if len(dbName) > 0 {
		b.cfg.Server.DbName = dbName[0]
	}
	return b
}

// AddServer adds a database server for multi-server sharding.
// key is the routing key string that matches the db sharding rule output.
func (b *ShardedBuilder[T]) AddServer(key string, host string, port uint, user, password, dbName string) *ShardedBuilder[T] {
	b.cfg.Servers = append(b.cfg.Servers, NamedServerConfig{
		ServerConfig: ServerConfig{
			Host:     host,
			Port:     port,
			User:     user,
			Password: password,
			DbName:   dbName,
		},
		Key: key,
	})
	return b
}

// Debug enables GORM debug logging for all connections.
// Must be called after Server() or AddServer().
func (b *ShardedBuilder[T]) Debug() *ShardedBuilder[T] {
	if b.cfg.Server != nil {
		b.cfg.Server.Debug = true
	}
	for i := range b.cfg.Servers {
		b.cfg.Servers[i].Debug = true
	}
	return b
}

// ConnPool configures the connection pool for all connections.
// Must be called after Server() or AddServer().
func (b *ShardedBuilder[T]) ConnPool(maxOpen, maxIdle, lifetimeSec int) *ShardedBuilder[T] {
	if b.cfg.Server != nil {
		b.cfg.Server.MaxOpenConns = maxOpen
		b.cfg.Server.MaxIdleConns = maxIdle
		b.cfg.Server.ConnMaxLifetimeSeconds = lifetimeSec
	}
	for i := range b.cfg.Servers {
		b.cfg.Servers[i].MaxOpenConns = maxOpen
		b.cfg.Servers[i].MaxIdleConns = maxIdle
		b.cfg.Servers[i].ConnMaxLifetimeSeconds = lifetimeSec
	}
	return b
}

// HashModDb configures hash-mod database sharding on a single server.
// keyField is the DB column name for extracting the sharding value.
// Generates db names: {prefix}_0, {prefix}_1, ..., {prefix}_{count-1}.
//
// Example:
//
//	.HashModDb("shop_id", "order_db", 4)
//	// Creates databases: order_db_0, order_db_1, order_db_2, order_db_3
func (b *ShardedBuilder[T]) HashModDb(keyField, prefix string, count int) *ShardedBuilder[T] {
	b.cfg.Db = &DbShardConfig{
		Rule:     "hash_mod",
		KeyField: keyField,
		Count:    count,
		Prefix:   prefix,
	}
	return b
}

// NamedDbs configures named database sharding on a single server.
// keyField is the DB column name for extracting the sharding value.
// Generates db names: {prefix}{key}{suffix} for each key.
//
// Example:
//
//	.NamedDbs("region", "order_", "_db", "SG", "TH", "ID")
//	// Creates databases: order_SG_db, order_TH_db, order_ID_db
//	// ShardingKey with region="SG" → routes to order_SG_db
func (b *ShardedBuilder[T]) NamedDbs(keyField, prefix, suffix string, keys ...string) *ShardedBuilder[T] {
	b.cfg.Db = &DbShardConfig{
		Rule:     "named",
		KeyField: keyField,
		Prefix:   prefix,
		Suffix:   suffix,
		Keys:     keys,
	}
	return b
}

// RangeDb configures range-based database sharding on a single server.
// keyField is the DB column name for extracting the sharding value.
// Generates db names: {prefix}_0, {prefix}_1, ..., {prefix}_{len(boundaries)}.
//
// Example:
//
//	.RangeDb("shop_id", "order_db", 1000, 2000)
//	// key < 1000 → order_db_0, 1000 <= key < 2000 → order_db_1, key >= 2000 → order_db_2
func (b *ShardedBuilder[T]) RangeDb(keyField, prefix string, boundaries ...int64) *ShardedBuilder[T] {
	b.cfg.Db = &DbShardConfig{
		Rule:       "range",
		KeyField:   keyField,
		Prefix:     prefix,
		Boundaries: boundaries,
	}
	return b
}

// HashModTable configures hash-mod table sharding.
// keyField is the DB column name for extracting the sharding value.
// Physical table name = entity.TableName() + "_%08d".
//
// Example:
//
//	.HashModTable("shop_id", 10)
//	// order_tab → order_tab_00000000, order_tab_00000001, ..., order_tab_00000009
func (b *ShardedBuilder[T]) HashModTable(keyField string, count int) *ShardedBuilder[T] {
	b.cfg.Table = &TableShardConfig{
		Rule:     "hash_mod",
		KeyField: keyField,
		Count:    count,
	}
	return b
}

// HashModTableWithFormat configures hash-mod table sharding with a custom suffix format.
// keyField is the DB column name for extracting the sharding value.
//
// Example:
//
//	.HashModTableWithFormat("shop_id", 10, "_%02d")
//	// order_tab → order_tab_00, order_tab_01, ..., order_tab_09
func (b *ShardedBuilder[T]) HashModTableWithFormat(keyField string, count int, format string) *ShardedBuilder[T] {
	b.cfg.Table = &TableShardConfig{
		Rule:     "hash_mod",
		KeyField: keyField,
		Count:    count,
		Format:   format,
	}
	return b
}

// MaxConcurrency limits concurrent goroutines for scatter-gather operations
// (FindAll / CountAll). 0 means unlimited.
func (b *ShardedBuilder[T]) MaxConcurrency(n int) *ShardedBuilder[T] {
	b.cfg.MaxConcurrency = n
	return b
}

// Build creates the sharded executor from the accumulated configuration.
func (b *ShardedBuilder[T]) Build() dbspi.Executor[T] {
	return NewShardedExecutorFromConfig(b.entity, b.cfg)
}
