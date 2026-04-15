package dbhelper

import (
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ShardedBuilder provides a fluent API for constructing sharded executors.
//
// Example (region-based DB + table sharding):
//
//	executor := dbhelper.Sharded(&Order{}).
//	    Server("10.0.0.1", 3306, "root", "pass").
//	    ExprDb("order_${region}_db",
//	        "${region} := enum(SG, TH, ID)",
//	        "${region} = @{region}",
//	    ).
//	    ExprTable("order_tab_${index}",
//	        "${idx} := range(0, 1000)",
//	        "${idx2} = @{shop_id} / 1000",
//	        "${idx} = ${idx2} % 1000",
//	        "${index} = fill(${idx}, 8)",
//	    ).
//	    Build()
//
// Example (table-only sharding):
//
//	executor := dbhelper.Sharded(&Order{}).
//	    Server("10.0.0.1", 3306, "root", "pass", "order_db").
//	    ExprTable("order_tab_${index}",
//	        "${idx} := range(0, 100)",
//	        "${idx} = @{shop_id} % 100",
//	        "${index} = fill(${idx}, 4)",
//	    ).
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
// For db sharding, omit the database name (it comes from the expression).
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

// ExprDb configures expression-based database sharding.
//
// nameExpr is the name template using only ${var} references (e.g., "order_${region}_db").
// expandExprs are variable declarations and computations (e.g.,
// "${region} := enum(SG, TH, ID)", "${region} = @{region}").
func (b *ShardedBuilder[T]) ExprDb(nameExpr string, expandExprs ...string) *ShardedBuilder[T] {
	b.cfg.Db = &DbShardConfig{
		NameExpr:    nameExpr,
		ExpandExprs: expandExprs,
	}
	return b
}

// ExprTable configures expression-based table sharding.
//
// nameExpr is the name template using only ${var} references (e.g., "order_tab_${index}").
// expandExprs are variable declarations and computations (e.g.,
// "${idx} := range(0, 1000)", "${idx} = @{shop_id} % 1000", "${index} = fill(${idx}, 8)").
func (b *ShardedBuilder[T]) ExprTable(nameExpr string, expandExprs ...string) *ShardedBuilder[T] {
	b.cfg.Table = &TableShardConfig{
		NameExpr:    nameExpr,
		ExpandExprs: expandExprs,
	}
	return b
}

// MaxConcurrency limits concurrent goroutines for scatter-gather operations.
func (b *ShardedBuilder[T]) MaxConcurrency(n int) *ShardedBuilder[T] {
	b.cfg.MaxConcurrency = n
	return b
}

// Build creates the sharded executor from the accumulated configuration.
func (b *ShardedBuilder[T]) Build() dbspi.Executor[T] {
	return NewShardedExecutorFromConfig(b.entity, b.cfg)
}
