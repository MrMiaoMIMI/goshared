package dbhelper

import (
	"fmt"
	"strconv"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp/expr"
)

// ================== Sharded Executor ==================

// ShardOption configures the sharded executor.
type ShardOption func(*shardConfig)

type shardConfig struct {
	dbs            []dbspi.DbTarget
	dbRule         dbspi.DbShardingRule
	tableRule      dbspi.TableShardingRule
	maxConcurrency int
}

// NewShardedExecutor creates a sharded executor with the given entity and options.
func NewShardedExecutor[T dbspi.Entity](entity T, opts ...ShardOption) dbspi.Executor[T] {
	cfg := &shardConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return dbsp.NewShardedExecutor(entity, dbsp.ShardedExecutorConfig{
		Dbs:            cfg.dbs,
		DbRule:         cfg.dbRule,
		TableRule:      cfg.tableRule,
		MaxConcurrency: cfg.maxConcurrency,
	})
}

// ================== Sharding Options ==================

// WithDbs sets the database target list for sharding.
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

// WithMaxConcurrency sets the max number of concurrent goroutines
// for scatter-gather operations (FindAll / CountAll).
func WithMaxConcurrency(n int) ShardOption {
	return func(c *shardConfig) {
		c.maxConcurrency = n
	}
}

// ================== Expression Rule Constructors ==================

// NewExprDbRule creates an expression-based DB sharding rule from a name template
// and expand expressions. Panics on parse/validation errors.
//
// Example:
//
//	rule := dbhelper.NewExprDbRule("order_${region}_db",
//	    "${region} := enum(SG, TH, ID)",
//	    "${region} = @{region}",
//	)
func NewExprDbRule(nameExpr string, expandExprs ...string) dbspi.DbShardingRule {
	tmpl, err := expr.ParseTemplate(nameExpr)
	if err != nil {
		panic(fmt.Sprintf("NewExprDbRule: parse name_expr %q: %v", nameExpr, err))
	}
	expands, err := expr.ParseExpands(expandExprs)
	if err != nil {
		panic(fmt.Sprintf("NewExprDbRule: parse expand_exprs: %v", err))
	}
	autoInferIdentityComputes(tmpl, expands)
	return dbsp.NewExprDbRule(tmpl, expands)
}

// NewExprTableRule creates an expression-based table sharding rule from a name template
// and expand expressions. Panics on parse/validation errors.
//
// Example:
//
//	rule := dbhelper.NewExprTableRule("order_tab_${index}",
//	    "${idx} := range(0, 1000)",
//	    "${idx} = @{shop_id} % 1000",
//	    "${index} = fill(${idx}, 8)",
//	)
func NewExprTableRule(nameExpr string, expandExprs ...string) dbspi.TableShardingRule {
	tmpl, err := expr.ParseTemplate(nameExpr)
	if err != nil {
		panic(fmt.Sprintf("NewExprTableRule: parse name_expr %q: %v", nameExpr, err))
	}
	expands, err := expr.ParseExpands(expandExprs)
	if err != nil {
		panic(fmt.Sprintf("NewExprTableRule: parse expand_exprs: %v", err))
	}
	rule, err := dbsp.NewExprTableRule(tmpl, expands)
	if err != nil {
		panic(fmt.Sprintf("NewExprTableRule: %v", err))
	}
	return rule
}

// ================== Convenience functions for DbTarget ==================

// SingleDb wraps a single Db into a []DbTarget with key "0".
func SingleDb(db dbspi.Db) []dbspi.DbTarget {
	return []dbspi.DbTarget{{Key: "0", Db: db}}
}

// IndexedDbs creates a []DbTarget with sequential string keys ("0", "1", "2", ...).
func IndexedDbs(dbs ...dbspi.Db) []dbspi.DbTarget {
	targets := make([]dbspi.DbTarget, len(dbs))
	for i, db := range dbs {
		targets[i] = dbspi.DbTarget{Key: strconv.Itoa(i), Db: db}
	}
	return targets
}

// NamedDbs creates a []DbTarget with string keys.
func NamedDbs(dbs map[string]dbspi.Db) []dbspi.DbTarget {
	targets := make([]dbspi.DbTarget, 0, len(dbs))
	for name, db := range dbs {
		targets = append(targets, dbspi.DbTarget{Key: name, Db: db})
	}
	return targets
}

// ================== DbTarget generation helpers ==================

// DbTargetEntry represents a single entry for generating DbTarget.
type DbTargetEntry struct {
	Key    string
	DbName string
}

// GenDbTargets creates []DbTarget from a single database server.
func GenDbTargets(host string, port uint, user, password string, entries ...DbTargetEntry) []dbspi.DbTarget {
	targets := make([]dbspi.DbTarget, len(entries))
	for i, entry := range entries {
		dbConfig := NewDbConfig(host, port, user, password, entry.DbName)
		targets[i] = dbspi.DbTarget{Key: entry.Key, Db: NewDb(dbConfig)}
	}
	return targets
}

// GenDbTargetsByNames creates []DbTarget where key == dbName.
func GenDbTargetsByNames(host string, port uint, user, password string, dbNames ...string) []dbspi.DbTarget {
	entries := make([]DbTargetEntry, len(dbNames))
	for i, name := range dbNames {
		entries[i] = DbTargetEntry{Key: name, DbName: name}
	}
	return GenDbTargets(host, port, user, password, entries...)
}

// GenDbTargetsByIndex creates []DbTarget with keys "0", "1", "2", ...
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
