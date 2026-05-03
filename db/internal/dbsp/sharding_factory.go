package dbsp

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp/expr"
)

// ShardOption configures a sharded executor.
type ShardOption func(*shardConfig)

type shardConfig struct {
	dbs            []DatabaseTarget
	dbRule         dbspi.DatabaseShardingRule
	tableRule      dbspi.TableShardingRule
	maxConcurrency int
	commonFields   dbspi.CommonFieldAutoFillOptions
}

// NewShardedExecutorWithOptions creates a sharded executor with the given entity and options.
func NewShardedExecutorWithOptions[T dbspi.Entity](entity T, opts ...ShardOption) dbspi.Executor[T] {
	cfg := &shardConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return NewShardedExecutor(entity, ShardedExecutorConfig{
		Dbs:               cfg.dbs,
		DbRule:            cfg.dbRule,
		TableShardingRule: cfg.tableRule,
		MaxConcurrency:    cfg.maxConcurrency,
		CommonFields:      cfg.commonFields,
	})
}

// WithDbs sets the database target list for sharding.
func WithDbs(dbs []DatabaseTarget) ShardOption {
	return func(c *shardConfig) {
		c.dbs = dbs
	}
}

// WithDbRule sets the database sharding rule.
func WithDbRule(rule dbspi.DatabaseShardingRule) ShardOption {
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

func WithCommonFieldAutoFill(commonFields dbspi.CommonFieldAutoFillOptions) ShardOption {
	return func(c *shardConfig) {
		c.commonFields = commonFields
	}
}

// BuildExprDbRule creates an expression-based DB sharding rule from a name template
// and expand expressions.
func BuildExprDbRule(nameExpr string, expandExprs []string) (dbspi.DatabaseShardingRule, error) {
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
func MustBuildExprDbRule(nameExpr string, expandExprs ...string) dbspi.DatabaseShardingRule {
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

// SingleDb wraps a single Db into a []DatabaseTarget with key "0".
func SingleDb(db dbSession) []DatabaseTarget {
	return []DatabaseTarget{{Key: "0", Db: db}}
}

// IndexedDbs creates a []DatabaseTarget with sequential string keys ("0", "1", "2", ...).
func IndexedDbs(dbs ...dbSession) []DatabaseTarget {
	targets := make([]DatabaseTarget, len(dbs))
	for i, db := range dbs {
		targets[i] = DatabaseTarget{Key: strconv.Itoa(i), Db: db}
	}
	return targets
}

// NamedDbs creates a []DatabaseTarget with string keys.
func NamedDbs(dbs map[string]dbSession) []DatabaseTarget {
	targets := make([]DatabaseTarget, 0, len(dbs))
	for name, db := range dbs {
		targets = append(targets, DatabaseTarget{Key: name, Db: db})
	}
	return targets
}

// DatabaseTargetEntry represents a single entry for generating DatabaseTarget.
type DatabaseTargetEntry struct {
	Key          string
	DatabaseName string
}

// GenDatabaseTargets creates []DatabaseTarget from a single database server.
func GenDatabaseTargets(host string, port uint, user, password string, entries ...DatabaseTargetEntry) []DatabaseTarget {
	targets := make([]DatabaseTarget, len(entries))
	for i, entry := range entries {
		targets[i] = DatabaseTarget{
			Key: entry.Key,
			Db: NewGormDb(dbspi.ServerConfig{
				Host:         host,
				Port:         port,
				User:         user,
				Password:     password,
				DatabaseName: entry.DatabaseName,
			}),
		}
	}
	return targets
}

// GenDatabaseTargetsByNames creates []DatabaseTarget where key == dbName.
func GenDatabaseTargetsByNames(host string, port uint, user, password string, dbNames ...string) []DatabaseTarget {
	entries := make([]DatabaseTargetEntry, len(dbNames))
	for i, name := range dbNames {
		entries[i] = DatabaseTargetEntry{Key: name, DatabaseName: name}
	}
	return GenDatabaseTargets(host, port, user, password, entries...)
}

// GenDatabaseTargetsByIndex creates []DatabaseTarget with keys "0", "1", "2", ...
func GenDatabaseTargetsByIndex(host string, port uint, user, password string, prefix string, count int) []DatabaseTarget {
	entries := make([]DatabaseTargetEntry, count)
	for i := 0; i < count; i++ {
		entries[i] = DatabaseTargetEntry{
			Key:          strconv.Itoa(i),
			DatabaseName: fmt.Sprintf("%s_%d", prefix, i),
		}
	}
	return GenDatabaseTargets(host, port, user, password, entries...)
}

// ShardingConfig provides a declarative configuration for sharded executors.
type ShardingConfig struct {
	// Server configures a single database server.
	Server *dbspi.ServerConfig `yaml:"server" json:"server"`

	// Servers configures multiple database servers.
	Servers []dbspi.NamedServerConfig `yaml:"servers" json:"servers"`

	// Db configures database-level sharding via expressions.
	Db *dbspi.DatabaseShardingConfig `yaml:"db" json:"db"`

	// Table configures table-level sharding via expressions.
	Table *dbspi.TableShardingConfig `yaml:"table" json:"table"`

	// MaxConcurrency limits concurrent goroutines for scatter-gather.
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"`
}

// NewShardedExecutorFromConfig creates a sharded executor from declarative configuration.
func NewShardedExecutorFromConfig[T dbspi.Entity](entity T, cfg ShardingConfig) dbspi.Executor[T] {
	var opts []ShardOption

	dbs, err := buildDatabaseTargets(cfg)
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

func newDbFromServer(server dbspi.ServerConfig, dbName string) dbSession {
	if server.DSN == "" && dbName != "" {
		server.DatabaseName = dbName
	}
	return NewGormDb(server)
}

func buildDatabaseTargets(cfg ShardingConfig) ([]DatabaseTarget, error) {
	if len(cfg.Servers) > 0 {
		targets := make([]DatabaseTarget, len(cfg.Servers))
		for i, s := range cfg.Servers {
			targets[i] = DatabaseTarget{
				Key: s.Key,
				Db:  newDbFromServer(s.ServerConfig, s.DatabaseName),
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

		targets := make([]DatabaseTarget, len(dbNames))
		for i, name := range dbNames {
			targets[i] = DatabaseTarget{
				Key: name,
				Db:  newDbFromServer(*cfg.Server, name),
			}
		}
		return targets, nil
	}

	return SingleDb(newDbFromServer(*cfg.Server, cfg.Server.DatabaseName)), nil
}

func buildDbRule(cfg *dbspi.DatabaseShardingConfig) (dbspi.DatabaseShardingRule, error) {
	if cfg == nil || cfg.NameExpr == "" {
		return nil, nil
	}
	return BuildExprDbRule(cfg.NameExpr, cfg.ExpandExprs)
}

func buildTableRule(cfg *dbspi.TableShardingConfig) (dbspi.TableShardingRule, error) {
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
	db  dbSession
	dbs []DatabaseTarget

	dbRule           dbspi.DatabaseShardingRule
	defaultTableRule dbspi.TableShardingRule
	maxConcurrency   int

	entityOverrides map[string]*entityOverride
}

type entityOverride struct {
	tableRule      dbspi.TableShardingRule
	maxConcurrency *int
}

type txBoundDbRule struct {
	rule      dbspi.DatabaseShardingRule
	targetKey string
}

func (r txBoundDbRule) ResolveDatabaseTargetKey(key *dbspi.ShardingKey) (string, error) {
	got, err := r.rule.ResolveDatabaseTargetKey(key)
	if err != nil {
		return "", err
	}
	if got != r.targetKey {
		return "", fmt.Errorf("cross-shard transaction not allowed: transaction db target %q, operation routes to %q", r.targetKey, got)
	}
	return got, nil
}

type txBoundDbRuleWithColumns struct {
	txBoundDbRule
	provider dbspi.ShardingKeyColumnsProvider
}

func (r txBoundDbRuleWithColumns) RequiredColumns() []string {
	return r.provider.RequiredColumns()
}

// Manager manages database connections and sharding configurations.
type Manager struct {
	mu           sync.RWMutex
	entries      map[string]*resolvedDbEntry
	commonFields dbspi.CommonFieldAutoFillOptions
}

func (*Manager) ManagerHandle() {}

// CommonFieldAutoFillOptions returns the manager-level common-field configuration.
func (m *Manager) CommonFieldAutoFillOptions() dbspi.CommonFieldAutoFillOptions {
	if m == nil {
		return dbspi.DefaultCommonFieldAutoFillOptions()
	}
	return m.commonFields
}

// Transaction starts a transaction for one database group and passes a
// transaction-scoped manager to fn. The scoped manager preserves the selected
// group's table rules but routes all database access through the transaction Db.
func (m *Manager) Transaction(ctx context.Context, dbKey string, shardingKey *dbspi.ShardingKey, commonFields dbspi.CommonFieldAutoFillOptions, fn func(txMgr *Manager) error) error {
	if m == nil {
		m = DefaultManager()
	}
	if dbKey == "" {
		dbKey = dbspi.DefaultDatabaseGroupKey
	}

	m.mu.RLock()
	entry, ok := m.entries[dbKey]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("dbhelper: database config %q not found", dbKey)
	}

	db, targetKey, err := resolveTransactionDb(entry, shardingKey)
	if err != nil {
		return err
	}

	return db.Transaction(ctx, func(txDb dbSession) error {
		txEntry := cloneEntryForTransaction(entry, txDb, targetKey)
		txMgr := &Manager{
			entries: map[string]*resolvedDbEntry{
				dbKey: txEntry,
			},
			commonFields: commonFields.Normalize(),
		}
		return fn(txMgr)
	})
}

func resolveTransactionDb(entry *resolvedDbEntry, shardingKey *dbspi.ShardingKey) (dbSession, string, error) {
	if entry.dbRule != nil {
		if shardingKey == nil {
			return nil, "", fmt.Errorf("dbhelper: transaction on db-sharded database requires WithTransactionShardingKey")
		}
		targetKey, err := entry.dbRule.ResolveDatabaseTargetKey(shardingKey)
		if err != nil {
			return nil, "", fmt.Errorf("resolve transaction db key failed: %w", err)
		}
		db, err := findDatabaseTarget(entry.dbs, targetKey)
		if err != nil {
			return nil, "", err
		}
		return db, targetKey, nil
	}

	if entry.db != nil {
		return entry.db, "0", nil
	}
	if len(entry.dbs) > 0 {
		return entry.dbs[0].Db, entry.dbs[0].Key, nil
	}
	return nil, "", fmt.Errorf("dbhelper: transaction database has no Db target")
}

func findDatabaseTarget(dbs []DatabaseTarget, targetKey string) (dbSession, error) {
	for _, target := range dbs {
		if target.Key == targetKey {
			return target.Db, nil
		}
	}
	return nil, fmt.Errorf("no DatabaseTarget found for key: %s", targetKey)
}

func cloneEntryForTransaction(entry *resolvedDbEntry, txDb dbSession, targetKey string) *resolvedDbEntry {
	txEntry := *entry
	txEntry.db = txDb
	txEntry.dbs = []DatabaseTarget{{Key: targetKey, Db: txDb}}
	if entry.dbRule != nil {
		boundRule := txBoundDbRule{rule: entry.dbRule, targetKey: targetKey}
		if provider, ok := entry.dbRule.(dbspi.ShardingKeyColumnsProvider); ok {
			txEntry.dbRule = txBoundDbRuleWithColumns{txBoundDbRule: boundRule, provider: provider}
		} else {
			txEntry.dbRule = boundRule
		}
	}
	return &txEntry
}

var (
	defaultManager   *Manager
	defaultManagerMu sync.RWMutex
)

// NewManager creates a new Manager from the given configuration.
func NewManager(cfg dbspi.DatabaseConfig, commonFields dbspi.CommonFieldAutoFillOptions) *Manager {
	mgr := &Manager{
		entries:      make(map[string]*resolvedDbEntry, len(cfg.DatabaseGroups)),
		commonFields: commonFields.Normalize(),
	}
	for name, entry := range cfg.DatabaseGroups {
		mgr.entries[name] = resolveDbEntry(entry)
	}
	return mgr
}

// SetDefaultManager sets the global default Manager.
func SetDefaultManager(mgr *Manager) {
	defaultManagerMu.Lock()
	defer defaultManagerMu.Unlock()
	defaultManager = mgr
}

// DefaultManager returns the global default Manager.
func DefaultManager() *Manager {
	defaultManagerMu.RLock()
	defer defaultManagerMu.RUnlock()
	if defaultManager == nil {
		panic("dbhelper: default Manager not initialized, call dbhelper.SetDefaultManager() first")
	}
	return defaultManager
}

// For creates an Executor for the given entity using the Manager.
func For[T dbspi.Entity](entity T, managers ...*Manager) dbspi.Executor[T] {
	var mgr *Manager
	if len(managers) > 0 && managers[0] != nil {
		mgr = managers[0]
	} else {
		mgr = DefaultManager()
	}
	return ForWithCommonFieldAutoFill(entity, mgr, mgr.commonFields)
}

func ForWithCommonFieldAutoFill[T dbspi.Entity](entity T, mgr *Manager, commonFields dbspi.CommonFieldAutoFillOptions) dbspi.Executor[T] {
	if mgr == nil {
		mgr = DefaultManager()
	}
	commonFields = commonFields.Normalize()
	key := dbspi.DefaultDatabaseGroupKey
	if provider, ok := any(entity).(dbspi.DatabaseGroupKeyProvider); ok {
		key = provider.DatabaseGroupKey()
	}

	mgr.mu.RLock()
	entry, ok := mgr.entries[key]
	if !ok {
		entry, ok = mgr.entries[dbspi.DefaultDatabaseGroupKey]
	}
	mgr.mu.RUnlock()

	if !ok {
		panic(fmt.Sprintf("dbhelper: database config %q not found (and no %q fallback)", key, dbspi.DefaultDatabaseGroupKey))
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
		return NewExecutorWithCommonFieldAutoFill(db, entity, commonFields)
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
	opts = append(opts, WithCommonFieldAutoFill(commonFields))
	return NewShardedExecutorWithOptions(entity, opts...)
}

// ForEnhance creates an EnhancedExecutor for the given entity using the Manager.
func ForEnhance[T dbspi.Entity](entity T, managers ...*Manager) dbspi.EnhancedExecutor[T] {
	exec := For(entity, managers...)
	enhanced, ok := exec.(dbspi.EnhancedExecutor[T])
	if !ok {
		panic("dbhelper: resolved executor does not implement EnhancedExecutor")
	}
	return enhanced
}

func ForEnhanceWithCommonFieldAutoFill[T dbspi.Entity](entity T, mgr *Manager, commonFields dbspi.CommonFieldAutoFillOptions) dbspi.EnhancedExecutor[T] {
	exec := ForWithCommonFieldAutoFill(entity, mgr, commonFields)
	enhanced, ok := exec.(dbspi.EnhancedExecutor[T])
	if !ok {
		panic("dbhelper: resolved executor does not implement EnhancedExecutor")
	}
	return enhanced
}

func resolveDbEntry(entry dbspi.DatabaseGroupConfig) *resolvedDbEntry {
	resolved := &resolvedDbEntry{
		entityOverrides: make(map[string]*entityOverride),
		maxConcurrency:  entry.MaxConcurrency,
	}

	serverCfg := toServerConfig(entry)

	if entry.DatabaseSharding != nil || len(entry.Servers) > 0 {
		if entry.DatabaseSharding != nil && len(entry.Servers) == 0 && entry.DSN != "" {
			panic("dbhelper: DSN cannot be used with database_sharding on a single server " +
				"(DSN includes the database name). Use Host/Port/User/Password fields instead, " +
				"or use the Servers list with per-server DSN")
		}

		shardCfg := ShardingConfig{
			Db: entry.DatabaseSharding,
		}
		if len(entry.Servers) > 0 {
			shardCfg.Servers = entry.Servers
		} else {
			shardCfg.Server = &serverCfg
		}
		dbs, err := buildDatabaseTargets(shardCfg)
		if err != nil {
			panic(fmt.Sprintf("dbhelper: build db targets: %v", err))
		}
		resolved.dbs = dbs
		if entry.DatabaseSharding != nil {
			rule, err := buildDbRule(entry.DatabaseSharding)
			if err != nil {
				panic(fmt.Sprintf("dbhelper: build db rule: %v", err))
			}
			resolved.dbRule = rule
		}
	} else {
		resolved.db = newDbFromServer(serverCfg, entry.DatabaseName)
	}

	if entry.TableSharding != nil {
		rule, err := buildTableRule(entry.TableSharding)
		if err != nil {
			panic(fmt.Sprintf("dbhelper: build table rule: %v", err))
		}
		resolved.defaultTableRule = rule
	}

	for _, rule := range entry.TableRules {
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

func toServerConfig(entry dbspi.DatabaseGroupConfig) dbspi.ServerConfig {
	return dbspi.ServerConfig{
		DSN:                    entry.DSN,
		Host:                   entry.Host,
		Port:                   entry.Port,
		User:                   entry.User,
		Password:               entry.Password,
		DatabaseName:           entry.DatabaseName,
		Debug:                  entry.Debug,
		MaxOpenConns:           entry.MaxOpenConns,
		MaxIdleConns:           entry.MaxIdleConns,
		ConnMaxLifetimeSeconds: entry.ConnMaxLifetimeSeconds,
	}
}
