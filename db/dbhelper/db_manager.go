package dbhelper

import (
	"fmt"
	"sync"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ================== Top-level Configuration ==================

// DatabaseConfig is the top-level configuration for all databases.
// It maps database group names to their connection and sharding configurations.
//
// YAML example:
//
//	databases:
//	  default:
//	    host: 10.0.0.1
//	    port: 3306
//	    user: root
//	    password: pass
//	    db_name: my_app_db
//
//	  order_dbs:
//	    host: 10.0.0.1
//	    port: 3306
//	    user: root
//	    password: pass
//	    db_sharding:
//	      rule: hash_mod
//	      key_field: shop_id
//	      count: 4
//	      prefix: order_db
//	    table_sharding:
//	      rule: hash_mod
//	      key_field: shop_id
//	      count: 10
//	    entity_rules:
//	      - tables: [order_detail_tab, order_log_tab]
//	        table_sharding:
//	          rule: hash_mod
//	          key_field: shop_id
//	          count: 20
type DatabaseConfig struct {
	Databases map[string]DatabaseEntry `yaml:"databases" json:"databases"`
}

// DatabaseEntry configures a single database or a sharded database group.
//
// Connection can be specified via DSN or individual fields:
//
//	# DSN mode
//	dsn: "root:pass@tcp(10.0.0.1:3306)/my_app_db"
//
//	# Field mode
//	host: 10.0.0.1
//	port: 3306
//	user: root
//	password: pass
//	db_name: my_app_db
type DatabaseEntry struct {
	// Connection: DSN string (takes precedence over individual fields)
	DSN string `yaml:"dsn" json:"dsn"`

	// Connection: individual fields
	Host     string `yaml:"host" json:"host"`
	Port     uint   `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	DbName   string `yaml:"db_name" json:"db_name"`
	Debug    bool   `yaml:"debug" json:"debug"`

	// Connection pool
	MaxOpenConns           int `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns           int `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetimeSeconds int `yaml:"conn_max_lifetime_seconds" json:"conn_max_lifetime_seconds"`

	// Database-level sharding (shared by all entities in this group).
	// Omit for non-sharded databases.
	DbSharding *DbShardConfig `yaml:"db_sharding" json:"db_sharding"`

	// Default table-level sharding for all entities in this group.
	// Can be overridden per entity via EntityRules.
	TableSharding *TableShardConfig `yaml:"table_sharding" json:"table_sharding"`

	// Per-entity table sharding overrides.
	// Each rule maps a list of table names to a specific table sharding config.
	EntityRules []EntityRule `yaml:"entity_rules" json:"entity_rules"`

	// Multi-server configuration (for cross-server database sharding).
	// When set, Host/Port/User/Password/DSN are ignored for db target generation.
	Servers []NamedServerConfig `yaml:"servers" json:"servers"`

	// Max concurrent goroutines for scatter-gather (FindAll / CountAll).
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"`
}

// EntityRule defines a table sharding override for a group of tables.
type EntityRule struct {
	// Tables lists the logical table names (Entity.TableName()) this rule applies to.
	Tables []string `yaml:"tables" json:"tables"`

	// TableSharding overrides the database-level default for these tables.
	TableSharding *TableShardConfig `yaml:"table_sharding" json:"table_sharding"`

	// MaxConcurrency overrides the database-level default for these tables.
	MaxConcurrency *int `yaml:"max_concurrency" json:"max_concurrency"`
}

// ================== Resolved internal types ==================

type resolvedDbEntry struct {
	// Non-sharded
	db dbspi.Db

	// Sharded
	dbs    []dbspi.DbTarget
	dbRule dbspi.DbShardingRule

	// Key fields (DB column names for extracting from ShardingKey)
	dbKeyField    string
	tableKeyField string

	// Default table sharding for all entities
	defaultTableRule dbspi.TableShardingRule
	maxConcurrency   int

	// Per-entity overrides (keyed by TableName)
	entityOverrides map[string]*entityOverride
}

type entityOverride struct {
	tableRule      dbspi.TableShardingRule
	tableKeyField  string
	maxConcurrency *int
}

// ================== DbManager ==================

// DbManager manages database connections and sharding configurations.
// Create with NewDbManager() and use For() to get executors.
type DbManager struct {
	mu      sync.RWMutex
	entries map[string]*resolvedDbEntry
}

var (
	defaultManager   *DbManager
	defaultManagerMu sync.RWMutex
)

// NewDbManager creates a new DbManager from the given configuration.
//
// Example:
//
//	mgr := dbhelper.NewDbManager(dbhelper.DatabaseConfig{
//	    Databases: map[string]dbhelper.DatabaseEntry{
//	        "default": {Host: "10.0.0.1", Port: 3306, User: "root", Password: "pass", DbName: "my_db"},
//	        "order_dbs": {
//	            Host: "10.0.0.1", Port: 3306, User: "root", Password: "pass",
//	            DbSharding:    &dbhelper.DbShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 4, Prefix: "order_db"},
//	            TableSharding: &dbhelper.TableShardConfig{Rule: "hash_mod", KeyField: "shop_id", Count: 10},
//	        },
//	    },
//	})
func NewDbManager(cfg DatabaseConfig) *DbManager {
	mgr := &DbManager{
		entries: make(map[string]*resolvedDbEntry, len(cfg.Databases)),
	}
	for name, entry := range cfg.Databases {
		mgr.entries[name] = resolveDbEntry(entry)
	}
	return mgr
}

// SetDefault sets the global default DbManager.
// Call this once at application startup.
func SetDefault(mgr *DbManager) {
	defaultManagerMu.Lock()
	defer defaultManagerMu.Unlock()
	defaultManager = mgr
}

// Default returns the global default DbManager.
// Panics if SetDefault has not been called.
func Default() *DbManager {
	defaultManagerMu.RLock()
	defer defaultManagerMu.RUnlock()
	if defaultManager == nil {
		panic("dbhelper: default DbManager not initialized, call dbhelper.SetDefault() first")
	}
	return defaultManager
}

// For creates an Executor for the given entity using the DbManager.
// The entity's DbKey() determines which database group to use (defaults to "default").
// Table sharding is resolved from the database group's default + entity-specific overrides.
//
// If managers are provided, the first one is used. Otherwise, the global default is used.
//
// Example:
//
//	// Using global default manager
//	orderExec := dbhelper.For(&Order{})
//
//	// Using explicit manager
//	orderExec := dbhelper.For(&Order{}, mgr)
func For[T dbspi.Entity](entity T, managers ...*DbManager) dbspi.Executor[T] {
	var mgr *DbManager
	if len(managers) > 0 && managers[0] != nil {
		mgr = managers[0]
	} else {
		mgr = Default()
	}

	key := "default"
	if provider, ok := any(entity).(dbspi.DbKeyProvider); ok {
		key = provider.DbKey()
	}

	mgr.mu.RLock()
	entry, ok := mgr.entries[key]
	if !ok {
		entry, ok = mgr.entries["default"]
	}
	mgr.mu.RUnlock()

	if !ok {
		panic(fmt.Sprintf("dbhelper: database config %q not found (and no \"default\" fallback)", key))
	}

	tableName := entity.TableName()
	tableRule := entry.defaultTableRule
	tableKeyField := entry.tableKeyField
	maxConcurrency := entry.maxConcurrency

	if override, exists := entry.entityOverrides[tableName]; exists {
		if override.tableRule != nil {
			tableRule = override.tableRule
		}
		if override.tableKeyField != "" {
			tableKeyField = override.tableKeyField
		}
		if override.maxConcurrency != nil {
			maxConcurrency = *override.maxConcurrency
		}
	}

	if entry.dbRule == nil && tableRule == nil {
		db := entry.db
		if db == nil && len(entry.dbs) > 0 {
			db = entry.dbs[0].Db
		}
		return NewExecutor(db, entity)
	}

	var opts []ShardOption
	if len(entry.dbs) > 0 {
		opts = append(opts, WithDbs(entry.dbs))
	} else if entry.db != nil {
		opts = append(opts, WithDbs(SingleDb(entry.db)))
	}
	if entry.dbRule != nil {
		opts = append(opts, WithDbRule(entry.dbRule))
	}
	if entry.dbKeyField != "" {
		opts = append(opts, WithDbKeyField(entry.dbKeyField))
	}
	if tableRule != nil {
		opts = append(opts, WithTableRule(tableRule))
	}
	if tableKeyField != "" {
		opts = append(opts, WithTableKeyField(tableKeyField))
	}
	if maxConcurrency > 0 {
		opts = append(opts, WithMaxConcurrency(maxConcurrency))
	}
	return NewShardedExecutor(entity, opts...)
}

// ================== Internal resolution ==================

func resolveDbEntry(entry DatabaseEntry) *resolvedDbEntry {
	resolved := &resolvedDbEntry{
		entityOverrides: make(map[string]*entityOverride),
		maxConcurrency:  entry.MaxConcurrency,
	}

	serverCfg := toServerConfig(entry)

	if entry.DbSharding != nil || len(entry.Servers) > 0 {
		if entry.DbSharding != nil && len(entry.Servers) == 0 && entry.DSN != "" {
			panic("dbhelper: DSN cannot be used with db_sharding on a single server " +
				"(DSN includes the database name). Use Host/Port/User/Password fields instead, " +
				"or use the Servers list with per-server DSN")
		}

		shardCfg := ShardingConfig{
			Db: entry.DbSharding,
		}
		if len(entry.Servers) > 0 {
			shardCfg.Servers = entry.Servers
		} else {
			shardCfg.Server = &serverCfg
		}
		resolved.dbs = buildDbTargets(shardCfg)
		resolved.dbRule = buildDbRule(entry.DbSharding)
		if entry.DbSharding != nil {
			resolved.dbKeyField = entry.DbSharding.KeyField
		}
	} else {
		resolved.db = newDbFromServer(serverCfg, entry.DbName)
	}

	if entry.TableSharding != nil {
		resolved.defaultTableRule = buildTableRule(entry.TableSharding)
		resolved.tableKeyField = entry.TableSharding.KeyField
	}

	for _, rule := range entry.EntityRules {
		override := &entityOverride{
			maxConcurrency: rule.MaxConcurrency,
		}
		if rule.TableSharding != nil {
			override.tableRule = buildTableRule(rule.TableSharding)
			override.tableKeyField = rule.TableSharding.KeyField
		}
		for _, tblName := range rule.Tables {
			resolved.entityOverrides[tblName] = override
		}
	}

	return resolved
}

func toServerConfig(entry DatabaseEntry) ServerConfig {
	return ServerConfig{
		DSN:                    entry.DSN,
		Host:                   entry.Host,
		Port:                   entry.Port,
		User:                   entry.User,
		Password:               entry.Password,
		DbName:                 entry.DbName,
		Debug:                  entry.Debug,
		MaxOpenConns:           entry.MaxOpenConns,
		MaxIdleConns:           entry.MaxIdleConns,
		ConnMaxLifetimeSeconds: entry.ConnMaxLifetimeSeconds,
	}
}
