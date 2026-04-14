package dbhelper

import (
	"fmt"
	"strconv"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ShardingConfig provides a declarative configuration for sharded executors.
// It can be populated from YAML, JSON, or directly in Go code.
//
// YAML example (table-only sharding):
//
//	server:
//	  host: 10.0.0.1
//	  port: 3306
//	  user: root
//	  password: secret
//	  db_name: order_db
//	table:
//	  rule: hash_mod
//	  key_field: shop_id
//	  count: 10
//
// YAML example (db + table sharding on same server):
//
//	server:
//	  host: 10.0.0.1
//	  port: 3306
//	  user: root
//	  password: secret
//	db:
//	  rule: hash_mod
//	  key_field: shop_id
//	  count: 4
//	  prefix: order_db
//	table:
//	  rule: hash_mod
//	  key_field: shop_id
//	  count: 10
//
// YAML example (named db sharding, e.g., by region):
//
//	server:
//	  host: 10.0.0.1
//	  port: 3306
//	  user: root
//	  password: secret
//	db:
//	  rule: named
//	  key_field: region
//	  prefix: order_
//	  suffix: _db
//	  keys: [SG, TH, ID]
//	table:
//	  rule: hash_mod
//	  key_field: shop_id
//	  count: 10
//
// YAML example (multi-server):
//
//	servers:
//	  - key: "0"
//	    host: 10.0.0.1
//	    port: 3306
//	    user: root
//	    password: secret
//	    db_name: order_db_0
//	  - key: "1"
//	    host: 10.0.0.2
//	    port: 3306
//	    user: root
//	    password: secret
//	    db_name: order_db_1
//	db:
//	  rule: hash_mod
//	  key_field: shop_id
//	  count: 2
//	table:
//	  rule: hash_mod
//	  key_field: shop_id
//	  count: 10
type ShardingConfig struct {
	// Server configures a single database server.
	// Use this when all databases reside on the same server.
	// For multi-server setups, use Servers instead.
	Server *ServerConfig `yaml:"server" json:"server"`

	// Servers configures multiple database servers.
	// Each entry maps a routing key to a server with a specific database.
	Servers []NamedServerConfig `yaml:"servers" json:"servers"`

	// Db configures database-level sharding.
	// Omit for table-only sharding (single database, multiple tables).
	//
	// Supported rules:
	//   - "hash_mod": hash(key) % Count → db index. Requires Count and Prefix.
	//   - "named": key → key (direct mapping). Requires Prefix, Suffix, and Keys.
	//   - "range": range-based routing. Requires Prefix and Boundaries.
	Db *DbShardConfig `yaml:"db" json:"db"`

	// Table configures table-level sharding.
	// Omit for db-only sharding (multiple databases, single table per db).
	//
	// Supported rules:
	//   - "hash_mod": hash(key) % Count → table suffix. Requires Count.
	Table *TableShardConfig `yaml:"table" json:"table"`

	// MaxConcurrency limits concurrent goroutines for scatter-gather operations
	// (FindAll / CountAll). 0 means unlimited.
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"`
}

// ServerConfig configures a database server connection.
// Either set DSN for a single connection string, or use Host/Port/User/Password fields.
// DSN takes precedence over individual fields.
type ServerConfig struct {
	DSN      string `yaml:"dsn" json:"dsn"`
	Host     string `yaml:"host" json:"host"`
	Port     uint   `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	DbName   string `yaml:"db_name" json:"db_name"`
	Debug    bool   `yaml:"debug" json:"debug"`

	MaxOpenConns           int `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns           int `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetimeSeconds int `yaml:"conn_max_lifetime_seconds" json:"conn_max_lifetime_seconds"`
}

// NamedServerConfig extends ServerConfig with a routing key for multi-server setups.
type NamedServerConfig struct {
	ServerConfig `yaml:",inline" json:",inline"`
	Key          string `yaml:"key" json:"key"`
}

// DbShardConfig configures database-level sharding.
type DbShardConfig struct {
	// Rule specifies the sharding algorithm: "hash_mod", "range", or "named".
	Rule string `yaml:"rule" json:"rule"`

	// KeyField is the DB column name used to extract the sharding value
	// from the ShardingKey. Required.
	KeyField string `yaml:"key_field" json:"key_field"`

	// Count is the number of database shards (for "hash_mod").
	Count int `yaml:"count" json:"count"`

	// Prefix is the database name prefix.
	//   hash_mod: generates {Prefix}_{0}, {Prefix}_{1}, ...
	//   named:    generates {Prefix}{key}{Suffix}
	//   range:    generates {Prefix}_{0}, {Prefix}_{1}, ...
	Prefix string `yaml:"prefix" json:"prefix"`

	// Suffix is appended after the key (for "named" rule only).
	Suffix string `yaml:"suffix" json:"suffix"`

	// Keys lists the explicit routing keys (for "named" rule only).
	// Each key maps to database "{Prefix}{key}{Suffix}".
	Keys []string `yaml:"keys" json:"keys"`

	// Boundaries defines range boundaries (for "range" rule only).
	// Creates len(Boundaries)+1 shards.
	// Example: [1000, 2000] →
	//   key < 1000 → shard 0, 1000 <= key < 2000 → shard 1, key >= 2000 → shard 2
	Boundaries []int64 `yaml:"boundaries" json:"boundaries"`
}

// TableShardConfig configures table-level sharding.
type TableShardConfig struct {
	// Rule specifies the sharding algorithm: "hash_mod".
	Rule string `yaml:"rule" json:"rule"`

	// KeyField is the DB column name used to extract the sharding value
	// from the ShardingKey. Required.
	KeyField string `yaml:"key_field" json:"key_field"`

	// Count is the number of table shards.
	Count int `yaml:"count" json:"count"`

	// Format is the suffix format string for table names.
	// Default: "_%08d" → order_tab_00000000, order_tab_00000001, ...
	// Example: "_%02d" → order_tab_00, order_tab_01, ...
	Format string `yaml:"format" json:"format"`
}

// NewShardedExecutorFromConfig creates a sharded executor from declarative configuration.
// This eliminates the need to manually construct DbTargets, sharding rules, and options.
//
// Example (table-only, minimal code):
//
//	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
//	    Server: &dbhelper.ServerConfig{
//	        Host: "10.0.0.1", Port: 3306, User: "root", Password: "pass", DbName: "order_db",
//	    },
//	    Table: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
//	})
//
// Example (db + table sharding):
//
//	executor := dbhelper.NewShardedExecutorFromConfig(&Order{}, dbhelper.ShardingConfig{
//	    Server: &dbhelper.ServerConfig{
//	        Host: "10.0.0.1", Port: 3306, User: "root", Password: "pass",
//	    },
//	    Db:    &dbhelper.DbShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 4, Prefix: "order_db"},
//	    Table: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
//	})
func NewShardedExecutorFromConfig[T dbspi.Entity](entity T, cfg ShardingConfig) dbspi.Executor[T] {
	var opts []ShardOption

	opts = append(opts, WithDbs(buildDbTargets(cfg)))

	if rule := buildDbRule(cfg.Db); rule != nil {
		opts = append(opts, WithDbRule(rule))
	}

	if rule := buildTableRule(cfg.Table); rule != nil {
		opts = append(opts, WithTableRule(rule))
	}

	if cfg.Db != nil && cfg.Db.KeyField != "" {
		opts = append(opts, WithDbKeyField(cfg.Db.KeyField))
	}

	if cfg.Table != nil && cfg.Table.KeyField != "" {
		opts = append(opts, WithTableKeyField(cfg.Table.KeyField))
	}

	if cfg.MaxConcurrency > 0 {
		opts = append(opts, WithMaxConcurrency(cfg.MaxConcurrency))
	}

	return NewShardedExecutor(entity, opts...)
}

func serverConfigOpts(server ServerConfig) []DbConfigOption {
	var opts []DbConfigOption
	if server.MaxOpenConns > 0 {
		opts = append(opts, WithMaxOpenConns(server.MaxOpenConns))
	}
	if server.MaxIdleConns > 0 {
		opts = append(opts, WithMaxIdleConns(server.MaxIdleConns))
	}
	if server.ConnMaxLifetimeSeconds > 0 {
		opts = append(opts, WithConnMaxLifetimeSeconds(server.ConnMaxLifetimeSeconds))
	}
	if server.Debug {
		opts = append(opts, WithDebugMode(server.Debug))
	}
	return opts
}

func newDbFromServer(server ServerConfig, dbName string) dbspi.Db {
	opts := serverConfigOpts(server)
	if server.DSN != "" {
		return NewDbFromDSN(server.DSN, opts...)
	}
	return NewDb(NewDbConfig(server.Host, server.Port, server.User, server.Password, dbName, opts...))
}

func buildDbTargets(cfg ShardingConfig) []dbspi.DbTarget {
	if len(cfg.Servers) > 0 {
		targets := make([]dbspi.DbTarget, len(cfg.Servers))
		for i, s := range cfg.Servers {
			targets[i] = dbspi.DbTarget{
				Key: s.Key,
				Db:  newDbFromServer(s.ServerConfig, s.DbName),
			}
		}
		return targets
	}

	if cfg.Server == nil {
		panic("sharding config requires either Server or Servers")
	}

	if cfg.Db != nil {
		switch cfg.Db.Rule {
		case "hash_mod":
			targets := make([]dbspi.DbTarget, cfg.Db.Count)
			for i := 0; i < cfg.Db.Count; i++ {
				dbName := fmt.Sprintf("%s_%d", cfg.Db.Prefix, i)
				targets[i] = dbspi.DbTarget{
					Key: strconv.Itoa(i),
					Db:  newDbFromServer(*cfg.Server, dbName),
				}
			}
			return targets

		case "named":
			targets := make([]dbspi.DbTarget, len(cfg.Db.Keys))
			for i, key := range cfg.Db.Keys {
				dbName := cfg.Db.Prefix + key + cfg.Db.Suffix
				targets[i] = dbspi.DbTarget{
					Key: key,
					Db:  newDbFromServer(*cfg.Server, dbName),
				}
			}
			return targets

		case "range":
			count := len(cfg.Db.Boundaries) + 1
			targets := make([]dbspi.DbTarget, count)
			for i := 0; i < count; i++ {
				dbName := fmt.Sprintf("%s_%d", cfg.Db.Prefix, i)
				targets[i] = dbspi.DbTarget{
					Key: strconv.Itoa(i),
					Db:  newDbFromServer(*cfg.Server, dbName),
				}
			}
			return targets

		default:
			panic(fmt.Sprintf("unsupported db sharding rule: %q", cfg.Db.Rule))
		}
	}

	return SingleDb(newDbFromServer(*cfg.Server, cfg.Server.DbName))
}

func buildDbRule(cfg *DbShardConfig) dbspi.DbShardingRule {
	if cfg == nil {
		return nil
	}
	switch cfg.Rule {
	case "hash_mod":
		return NewHashModDbRule(cfg.Count)
	case "named":
		return NewDirectDbRule()
	case "range":
		return NewRangeDbRule(cfg.Boundaries)
	default:
		panic(fmt.Sprintf("unsupported db sharding rule: %q", cfg.Rule))
	}
}

func buildTableRule(cfg *TableShardConfig) dbspi.TableShardingRule {
	if cfg == nil {
		return nil
	}
	switch cfg.Rule {
	case "hash_mod":
		if cfg.Format != "" {
			return NewHashModTableRuleWithFormat(cfg.Count, cfg.Format)
		}
		return NewHashModTableRule(cfg.Count)
	default:
		panic(fmt.Sprintf("unsupported table sharding rule: %q", cfg.Rule))
	}
}
