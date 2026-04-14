package dbhelper

import (
	"fmt"
	"strconv"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

// ================== Sharded Executor ==================

// ShardOption configures the sharded executor.
type ShardOption func(*shardConfig)

type shardConfig struct {
	dbs            []dbspi.DbTarget
	dbRule         dbspi.DbShardingRule
	tableRule      dbspi.TableShardingRule
	dbKeyField     string
	tableKeyField  string
	maxConcurrency int
}

// NewShardedExecutor creates a sharded executor with the given entity and options.
// Use WithDbs + WithTableRule for table-only sharding (single or multiple dbs).
// Use WithDbs + WithDbRule + WithTableRule for database + table sharding.
// Use WithDbs + WithDbRule for database-only sharding.
func NewShardedExecutor[T dbspi.Entity](entity T, opts ...ShardOption) dbspi.Executor[T] {
	cfg := &shardConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return dbsp.NewShardedExecutor(entity, dbsp.ShardedExecutorConfig{
		Dbs:            cfg.dbs,
		DbRule:         cfg.dbRule,
		TableRule:      cfg.tableRule,
		DbKeyField:    cfg.dbKeyField,
		TableKeyField: cfg.tableKeyField,
		MaxConcurrency: cfg.maxConcurrency,
	})
}

// ================== Sharding Options ==================

// WithDbs sets the database target list for sharding.
// Each DbTarget maps a routing key string to a Db instance.
// Use SingleDb() or IndexedDbs() for convenience.
func WithDbs(dbs []dbspi.DbTarget) ShardOption {
	return func(c *shardConfig) {
		c.dbs = dbs
	}
}

// WithDbRule sets the database sharding rule.
func WithDbRule(rule dbspi.DbShardingRule) ShardOption {
	return func(c *shardConfig) {
		c.dbRule = rule
	}
}

// WithTableRule sets the table sharding rule.
func WithTableRule(rule dbspi.TableShardingRule) ShardOption {
	return func(c *shardConfig) {
		c.tableRule = rule
	}
}

// WithDbKeyField sets the DB column name for extracting the db sharding value
// from the ShardingKey.
func WithDbKeyField(field string) ShardOption {
	return func(c *shardConfig) {
		c.dbKeyField = field
	}
}

// WithTableKeyField sets the DB column name for extracting the table sharding value
// from the ShardingKey.
func WithTableKeyField(field string) ShardOption {
	return func(c *shardConfig) {
		c.tableKeyField = field
	}
}

// WithMaxConcurrency sets the max number of concurrent goroutines
// for scatter-gather operations (FindAll / CountAll).
// Default 0 means unlimited concurrency.
// Set this to control resource usage when you have many shards.
func WithMaxConcurrency(n int) ShardOption {
	return func(c *shardConfig) {
		c.maxConcurrency = n
	}
}

// ================== Convenience functions for DbTarget ==================

// SingleDb wraps a single Db into a []DbTarget with key "0".
// Use this for table-only sharding (single database, multiple tables).
func SingleDb(db dbspi.Db) []dbspi.DbTarget {
	return []dbspi.DbTarget{{Key: "0", Db: db}}
}

// IndexedDbs creates a []DbTarget with sequential string keys ("0", "1", "2", ...).
// Use this with hash-mod db rules where the target key is a stringified index.
func IndexedDbs(dbs ...dbspi.Db) []dbspi.DbTarget {
	targets := make([]dbspi.DbTarget, len(dbs))
	for i, db := range dbs {
		targets[i] = dbspi.DbTarget{Key: strconv.Itoa(i), Db: db}
	}
	return targets
}

// NamedDbs creates a []DbTarget with string keys.
// Use this with named/direct db rules where the target key is a string (e.g., country code).
func NamedDbs(dbs map[string]dbspi.Db) []dbspi.DbTarget {
	targets := make([]dbspi.DbTarget, 0, len(dbs))
	for name, db := range dbs {
		targets = append(targets, dbspi.DbTarget{Key: name, Db: db})
	}
	return targets
}

// ================== DbTarget generation helpers ==================

// DbTargetEntry represents a single entry for generating DbTarget.
// Key is the routing key string, DbName is the database name on the server.
type DbTargetEntry struct {
	Key    string
	DbName string
}

// GenDbTargets creates []DbTarget from a single database server by generating
// a separate Db connection for each entry's DbName.
// This is useful when multiple databases reside on the same server.
//
// Example:
//
//	targets := dbhelper.GenDbTargets("10.0.0.1", 3306, "root", "pass",
//	    dbhelper.DbTargetEntry{Key: "SG", DbName: "order_sg_db"},
//	    dbhelper.DbTargetEntry{Key: "TH", DbName: "order_th_db"},
//	)
func GenDbTargets(host string, port uint, user, password string, entries ...DbTargetEntry) []dbspi.DbTarget {
	targets := make([]dbspi.DbTarget, len(entries))
	for i, entry := range entries {
		dbConfig := NewDbConfig(host, port, user, password, entry.DbName)
		targets[i] = dbspi.DbTarget{Key: entry.Key, Db: NewDb(dbConfig)}
	}
	return targets
}

// GenDbTargetsByNames creates []DbTarget where each database name also serves as the routing key.
// This is a shortcut for GenDbTargets when key == dbName.
//
// Example:
//
//	targets := dbhelper.GenDbTargetsByNames("10.0.0.1", 3306, "root", "pass",
//	    "order_sg_db", "order_th_db", "order_id_db",
//	)
//	// Result: [{Key:"order_sg_db", Db:...}, {Key:"order_th_db", Db:...}, ...]
func GenDbTargetsByNames(host string, port uint, user, password string, dbNames ...string) []dbspi.DbTarget {
	entries := make([]DbTargetEntry, len(dbNames))
	for i, name := range dbNames {
		entries[i] = DbTargetEntry{Key: name, DbName: name}
	}
	return GenDbTargets(host, port, user, password, entries...)
}

// GenDbTargetsByIndex creates []DbTarget with string keys ("0", "1", "2", ...)
// and database names generated from a prefix + index: "{prefix}_{index}".
// Use with NewHashModDbRule(count).
//
// Example:
//
//	targets := dbhelper.GenDbTargetsByIndex("10.0.0.1", 3306, "root", "pass",
//	    "order_db", 4,
//	)
//	// Result: [{Key:"0", Db:"order_db_0"}, {Key:"1", Db:"order_db_1"}, ...]
func GenDbTargetsByIndex(host string, port uint, user, password string, prefix string, count int) []dbspi.DbTarget {
	entries := make([]DbTargetEntry, count)
	for i := 0; i < count; i++ {
		entries[i] = DbTargetEntry{
			Key:    strconv.Itoa(i),
			DbName: fmt.Sprintf("%s_%d", prefix, i),
		}
	}
	return GenDbTargets(host, port, user, password, entries...)
}

// GenDbTargetsByFunc creates []DbTarget using a user-provided function that
// generates (key, dbName) pairs.
//
// Example (region-based):
//
//	targets := dbhelper.GenDbTargetsByFunc("10.0.0.1", 3306, "root", "pass",
//	    func() []dbhelper.DbTargetEntry {
//	        regions := []string{"SG", "TH", "ID", "MY"}
//	        entries := make([]dbhelper.DbTargetEntry, len(regions))
//	        for i, r := range regions {
//	            entries[i] = dbhelper.DbTargetEntry{Key: r, DbName: "order_" + r + "_db"}
//	        }
//	        return entries
//	    },
//	)
func GenDbTargetsByFunc(host string, port uint, user, password string, fn func() []DbTargetEntry) []dbspi.DbTarget {
	return GenDbTargets(host, port, user, password, fn()...)
}

// GenDbTargetsBySuffix creates []DbTarget with string keys matching the suffixes,
// and database names generated from prefix + suffix.
//
// Example:
//
//	targets := dbhelper.GenDbTargetsBySuffix("10.0.0.1", 3306, "root", "pass",
//	    "order_", "_db", []string{"SG", "TH", "ID"},
//	)
//	// Result: [{Key:"SG", Db:"order_SG_db"}, {Key:"TH", Db:"order_TH_db"}, ...]
func GenDbTargetsBySuffix(host string, port uint, user, password string, prefix, suffix string, keys []string) []dbspi.DbTarget {
	entries := make([]DbTargetEntry, len(keys))
	for i, key := range keys {
		entries[i] = DbTargetEntry{
			Key:    key,
			DbName: prefix + key + suffix,
		}
	}
	return GenDbTargets(host, port, user, password, entries...)
}

// ================== Table Sharding Rules ==================

// NewHashModTableRule creates a hash-mod table sharding rule.
// Physical table name = logicalTable + fmt.Sprintf("_%08d", hash(key) % shardCount)
func NewHashModTableRule(shardCount int) dbspi.TableShardingRule {
	return dbsp.NewHashModTableRule(shardCount)
}

// NewHashModTableRuleWithFormat creates a hash-mod table sharding rule with custom suffix format.
// Example: NewHashModTableRuleWithFormat(10, "_%08d") → order_tab_00000000 ... order_tab_00000009
func NewHashModTableRuleWithFormat(shardCount int, suffixFormat string) dbspi.TableShardingRule {
	return dbsp.NewHashModTableRuleWithFormat(shardCount, suffixFormat)
}

// NewCustomTableRule creates a custom table sharding rule with the given function.
func NewCustomTableRule(fn func(logicalTable string, key dbspi.ShardingValue) (string, error)) dbspi.TableShardingRule {
	return dbsp.NewCustomTableRule(fn)
}

// ================== Db Sharding Rules ==================

// NewHashModDbRule creates a hash-mod database sharding rule.
// Returns target key as a string ("0", "1", ...), use with IndexedDbs().
func NewHashModDbRule(dbCount int) dbspi.DbShardingRule {
	return dbsp.NewHashModDbRule(dbCount)
}

// NewRangeDbRule creates a range-based database sharding rule.
// boundaries defines the upper bound (exclusive) for each shard.
// Returns target key as a string ("0", "1", ...), use with IndexedDbs().
func NewRangeDbRule(boundaries []int64) dbspi.DbShardingRule {
	return dbsp.NewRangeDbRule(boundaries)
}

// NewDirectDbRule creates a direct (pass-through) database sharding rule.
// The sharding value's String() is used directly as the target key to match DbTarget.Key.
func NewDirectDbRule() dbspi.DbShardingRule {
	return dbsp.NewDirectDbRule()
}

// NewCustomDbRule creates a custom database sharding rule with the given function.
// The fn should return a target key string that matches DbTarget.Key.
func NewCustomDbRule(fn func(key dbspi.ShardingValue) (string, error)) dbspi.DbShardingRule {
	return dbsp.NewCustomDbRule(fn)
}
