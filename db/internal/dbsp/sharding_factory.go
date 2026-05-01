package dbsp

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp/expr"
)

// ShardOption configures a sharded executor.
type ShardOption func(*shardConfig)

type shardConfig struct {
	dbs            []dbspi.DbTarget
	dbRule         dbspi.DbShardingRule
	tableRule      dbspi.TableShardingRule
	maxConcurrency int
}

// NewShardedExecutorWithOptions creates a sharded executor with the given entity and options.
func NewShardedExecutorWithOptions[T dbspi.Entity](entity T, opts ...ShardOption) dbspi.Executor[T] {
	cfg := &shardConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return NewShardedExecutor(entity, ShardedExecutorConfig{
		Dbs:            cfg.dbs,
		DbRule:         cfg.dbRule,
		TableRule:      cfg.tableRule,
		MaxConcurrency: cfg.maxConcurrency,
	})
}

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

// WithMaxConcurrency sets the max number of concurrent goroutines for scatter-gather operations.
func WithMaxConcurrency(n int) ShardOption {
	return func(c *shardConfig) {
		c.maxConcurrency = n
	}
}

// BuildExprDbRule creates an expression-based DB sharding rule from a name template
// and expand expressions.
func BuildExprDbRule(nameExpr string, expandExprs []string) (dbspi.DbShardingRule, error) {
	tmpl, err := expr.ParseTemplate(nameExpr)
	if err != nil {
		return nil, fmt.Errorf("parse db name_expr %q: %w", nameExpr, err)
	}
	expands, err := expr.ParseExpands(expandExprs)
	if err != nil {
		return nil, fmt.Errorf("parse db expand_exprs: %w", err)
	}
	autoInferIdentityComputes(tmpl, expands)
	return NewExprDbRule(tmpl, expands), nil
}

// BuildExprTableRule creates an expression-based table sharding rule from a name template
// and expand expressions.
func BuildExprTableRule(nameExpr string, expandExprs []string) (dbspi.TableShardingRule, error) {
	tmpl, err := expr.ParseTemplate(nameExpr)
	if err != nil {
		return nil, fmt.Errorf("parse table name_expr %q: %w", nameExpr, err)
	}
	expands, err := expr.ParseExpands(expandExprs)
	if err != nil {
		return nil, fmt.Errorf("parse table expand_exprs: %w", err)
	}
	return NewExprTableRule(tmpl, expands)
}

// MustBuildExprDbRule is the panic-on-error variant of BuildExprDbRule.
func MustBuildExprDbRule(nameExpr string, expandExprs ...string) dbspi.DbShardingRule {
	rule, err := BuildExprDbRule(nameExpr, expandExprs)
	if err != nil {
		panic(fmt.Sprintf("NewExprDbRule: %v", err))
	}
	return rule
}

// MustBuildExprTableRule is the panic-on-error variant of BuildExprTableRule.
func MustBuildExprTableRule(nameExpr string, expandExprs ...string) dbspi.TableShardingRule {
	rule, err := BuildExprTableRule(nameExpr, expandExprs)
	if err != nil {
		panic(fmt.Sprintf("NewExprTableRule: %v", err))
	}
	return rule
}

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

// DbTargetEntry represents a single entry for generating DbTarget.
type DbTargetEntry struct {
	Key    string
	DbName string
}

// GenDbTargets creates []DbTarget from a single database server.
func GenDbTargets(host string, port uint, user, password string, entries ...DbTargetEntry) []dbspi.DbTarget {
	targets := make([]dbspi.DbTarget, len(entries))
	for i, entry := range entries {
		targets[i] = dbspi.DbTarget{
			Key: entry.Key,
			Db: NewGormDb(dbspi.DbServerConfig{
				Host:     host,
				Port:     port,
				User:     user,
				Password: password,
				DbName:   entry.DbName,
			}),
		}
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

// ShardingConfig provides a declarative configuration for sharded executors.
type ShardingConfig struct {
	// Server configures a single database server.
	Server *dbspi.DbServerConfig `yaml:"server" json:"server"`

	// Servers configures multiple database servers.
	Servers []dbspi.NamedDbServerConfig `yaml:"servers" json:"servers"`

	// Db configures database-level sharding via expressions.
	Db *dbspi.DbShardConfig `yaml:"db" json:"db"`

	// Table configures table-level sharding via expressions.
	Table *dbspi.TableShardConfig `yaml:"table" json:"table"`

	// MaxConcurrency limits concurrent goroutines for scatter-gather.
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"`
}

// NewShardedExecutorFromConfig creates a sharded executor from declarative configuration.
func NewShardedExecutorFromConfig[T dbspi.Entity](entity T, cfg ShardingConfig) dbspi.Executor[T] {
	var opts []ShardOption

	dbs, err := buildDbTargets(cfg)
	if err != nil {
		panic(fmt.Sprintf("sharding config: build db targets: %v", err))
	}
	opts = append(opts, WithDbs(dbs))

	if cfg.Db != nil {
		rule, err := buildDbRule(cfg.Db)
		if err != nil {
			panic(fmt.Sprintf("sharding config: build db rule: %v", err))
		}
		opts = append(opts, WithDbRule(rule))
	}

	if cfg.Table != nil {
		rule, err := buildTableRule(cfg.Table)
		if err != nil {
			panic(fmt.Sprintf("sharding config: build table rule: %v", err))
		}
		opts = append(opts, WithTableRule(rule))
	}

	if cfg.MaxConcurrency > 0 {
		opts = append(opts, WithMaxConcurrency(cfg.MaxConcurrency))
	}

	return NewShardedExecutorWithOptions(entity, opts...)
}

func newDbFromServer(server dbspi.DbServerConfig, dbName string) dbspi.Db {
	if server.DSN == "" && dbName != "" {
		server.DbName = dbName
	}
	return NewGormDb(server)
}

func buildDbTargets(cfg ShardingConfig) ([]dbspi.DbTarget, error) {
	if len(cfg.Servers) > 0 {
		targets := make([]dbspi.DbTarget, len(cfg.Servers))
		for i, s := range cfg.Servers {
			targets[i] = dbspi.DbTarget{
				Key: s.Key,
				Db:  newDbFromServer(s.DbServerConfig, s.DbName),
			}
		}
		return targets, nil
	}

	if cfg.Server == nil {
		return nil, fmt.Errorf("sharding config requires either Server or Servers")
	}

	if cfg.Db != nil && cfg.Db.NameExpr != "" {
		rule, err := buildDbRule(cfg.Db)
		if err != nil {
			return nil, err
		}

		enumerator, ok := rule.(interface {
			EnumerateDbNames() ([]string, error)
		})
		if !ok {
			return nil, fmt.Errorf("db rule cannot enumerate db names")
		}

		dbNames, err := enumerator.EnumerateDbNames()
		if err != nil {
			return nil, fmt.Errorf("enumerate db names: %w", err)
		}

		targets := make([]dbspi.DbTarget, len(dbNames))
		for i, name := range dbNames {
			targets[i] = dbspi.DbTarget{
				Key: name,
				Db:  newDbFromServer(*cfg.Server, name),
			}
		}
		return targets, nil
	}

	return SingleDb(newDbFromServer(*cfg.Server, cfg.Server.DbName)), nil
}

func buildDbRule(cfg *dbspi.DbShardConfig) (dbspi.DbShardingRule, error) {
	if cfg == nil || cfg.NameExpr == "" {
		return nil, nil
	}
	return BuildExprDbRule(cfg.NameExpr, cfg.ExpandExprs)
}

func buildTableRule(cfg *dbspi.TableShardConfig) (dbspi.TableShardingRule, error) {
	if cfg == nil || cfg.NameExpr == "" {
		return nil, nil
	}
	return BuildExprTableRule(cfg.NameExpr, cfg.ExpandExprs)
}

// autoInferIdentityComputes checks if a ${var} in the template matches a := declaration
// but has no corresponding = computation. If so, it auto-generates "${var} = @{var}".
func autoInferIdentityComputes(tmpl *expr.Template, expands *expr.ExpandSet) {
	varRefs := tmpl.CollectVarRefs()
	if len(varRefs) == 0 {
		return
	}

	declaredVars := make(map[string]bool)
	for _, d := range expands.Decls {
		declaredVars[d.VarName] = true
	}

	computedVars := make(map[string]bool)
	for _, c := range expands.Computes {
		computedVars[c.VarName] = true
	}

	for _, varName := range varRefs {
		if declaredVars[varName] && !computedVars[varName] {
			identityExpr, err := expr.ParseExpressionString("@{" + varName + "}")
			if err != nil {
				panic(fmt.Sprintf("autoInferIdentityComputes: failed to parse identity expr for %q: %v", varName, err))
			}
			expands.Computes = append(expands.Computes, &expr.ExpandCompute{
				VarName: varName,
				Expr:    identityExpr,
			})
		}
	}
}

type resolvedDbEntry struct {
	db  dbspi.Db
	dbs []dbspi.DbTarget

	dbRule           dbspi.DbShardingRule
	defaultTableRule dbspi.TableShardingRule
	maxConcurrency   int

	entityOverrides map[string]*entityOverride
}

type entityOverride struct {
	tableRule      dbspi.TableShardingRule
	maxConcurrency *int
}

// DbManager manages database connections and sharding configurations.
type DbManager struct {
	mu      sync.RWMutex
	entries map[string]*resolvedDbEntry
}

// DBManager marks DbManager as the dbspi.DbManager implementation.
func (*DbManager) DBManager() {}

var (
	defaultManager   *DbManager
	defaultManagerMu sync.RWMutex
)

// NewDbManager creates a new DbManager from the given configuration.
func NewDbManager(cfg dbspi.DatabaseConfig) *DbManager {
	mgr := &DbManager{
		entries: make(map[string]*resolvedDbEntry, len(cfg.Databases)),
	}
	for name, entry := range cfg.Databases {
		mgr.entries[name] = resolveDbEntry(entry)
	}
	return mgr
}

// SetDefaultDbManager sets the global default DbManager.
func SetDefaultDbManager(mgr *DbManager) {
	defaultManagerMu.Lock()
	defer defaultManagerMu.Unlock()
	defaultManager = mgr
}

// DefaultDbManager returns the global default DbManager.
func DefaultDbManager() *DbManager {
	defaultManagerMu.RLock()
	defer defaultManagerMu.RUnlock()
	if defaultManager == nil {
		panic("dbhelper: default DbManager not initialized, call dbhelper.SetDefault() first")
	}
	return defaultManager
}

// For creates an Executor for the given entity using the DbManager.
func For[T dbspi.Entity](entity T, managers ...*DbManager) dbspi.Executor[T] {
	var mgr *DbManager
	if len(managers) > 0 && managers[0] != nil {
		mgr = managers[0]
	} else {
		mgr = DefaultDbManager()
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
	maxConcurrency := entry.maxConcurrency

	if override, exists := entry.entityOverrides[tableName]; exists {
		if override.tableRule != nil {
			tableRule = override.tableRule
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
	if tableRule != nil {
		opts = append(opts, WithTableRule(tableRule))
	}
	if maxConcurrency > 0 {
		opts = append(opts, WithMaxConcurrency(maxConcurrency))
	}
	return NewShardedExecutorWithOptions(entity, opts...)
}

// ForEnhance creates an EnhancedExecutor for the given entity using the DbManager.
func ForEnhance[T dbspi.Entity](entity T, managers ...*DbManager) dbspi.EnhancedExecutor[T] {
	exec := For(entity, managers...)
	enhanced, ok := exec.(dbspi.EnhancedExecutor[T])
	if !ok {
		panic("dbhelper: resolved executor does not implement EnhancedExecutor")
	}
	return enhanced
}

func resolveDbEntry(entry dbspi.DatabaseEntry) *resolvedDbEntry {
	resolved := &resolvedDbEntry{
		entityOverrides: make(map[string]*entityOverride),
		maxConcurrency:  entry.MaxConcurrency,
	}

	serverCfg := toDbServerConfig(entry)

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
		dbs, err := buildDbTargets(shardCfg)
		if err != nil {
			panic(fmt.Sprintf("dbhelper: build db targets: %v", err))
		}
		resolved.dbs = dbs
		if entry.DbSharding != nil {
			rule, err := buildDbRule(entry.DbSharding)
			if err != nil {
				panic(fmt.Sprintf("dbhelper: build db rule: %v", err))
			}
			resolved.dbRule = rule
		}
	} else {
		resolved.db = newDbFromServer(serverCfg, entry.DbName)
	}

	if entry.TableSharding != nil {
		rule, err := buildTableRule(entry.TableSharding)
		if err != nil {
			panic(fmt.Sprintf("dbhelper: build table rule: %v", err))
		}
		resolved.defaultTableRule = rule
	}

	for _, rule := range entry.EntityRules {
		override := &entityOverride{
			maxConcurrency: rule.MaxConcurrency,
		}
		if rule.TableSharding != nil {
			tsCfg := rule.TableSharding
			if tsCfg.NameExpr == "" && entry.TableSharding != nil {
				inherited := *tsCfg
				inherited.NameExpr = entry.TableSharding.NameExpr
				tsCfg = &inherited
			}
			tableRule, err := buildTableRule(tsCfg)
			if err != nil {
				panic(fmt.Sprintf("dbhelper: build entity table rule: %v", err))
			}
			override.tableRule = tableRule
		}
		for _, tblName := range rule.Tables {
			resolved.entityOverrides[tblName] = override
		}
	}

	return resolved
}

func toDbServerConfig(entry dbspi.DatabaseEntry) dbspi.DbServerConfig {
	return dbspi.DbServerConfig{
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
