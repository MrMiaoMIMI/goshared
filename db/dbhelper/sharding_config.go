package dbhelper

import (
	"fmt"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp/expr"
)

// ShardingConfig provides a declarative configuration for sharded executors.
//
// YAML example (region-based DB + table sharding):
//
//	server:
//	  host: 10.0.0.1
//	  port: 3306
//	  user: root
//	  password: secret
//	db:
//	  name_expr: "order_${region}_db"
//	  expand_exprs:
//	    - "${region} := enum(SG, TH, ID)"
//	    - "${region} = @{region}"
//	table:
//	  name_expr: "order_tab_${index}"
//	  expand_exprs:
//	    - "${idx} := range(0, 1000)"
//	    - "${idx} = @{shop_id} / 1000 % 1000"
//	    - "${index} = fill(${idx}, 8)"
//
// YAML example (hash-mod pattern):
//
//	server:
//	  host: 10.0.0.1
//	  port: 3306
//	  user: root
//	  password: secret
//	db:
//	  name_expr: "order_db_${idx}"
//	  expand_exprs:
//	    - "${idx} := range(0, 4)"
//	    - "${idx} = hash(@{shop_id}) % 4"
//	table:
//	  name_expr: "order_tab_${index}"
//	  expand_exprs:
//	    - "${idx} := range(0, 1000)"
//	    - "${idx} = hash(@{shop_id}) % 1000"
//	    - "${index} = fill(${idx}, 8)"
type ShardingConfig struct {
	// Server configures a single database server.
	Server *ServerConfig `yaml:"server" json:"server"`

	// Servers configures multiple database servers.
	Servers []NamedServerConfig `yaml:"servers" json:"servers"`

	// Db configures database-level sharding via expressions.
	Db *DbShardConfig `yaml:"db" json:"db"`

	// Table configures table-level sharding via expressions.
	Table *TableShardConfig `yaml:"table" json:"table"`

	// MaxConcurrency limits concurrent goroutines for scatter-gather.
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"`
}

// ServerConfig configures a database server connection.
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

// DbShardConfig configures database-level sharding via expressions.
type DbShardConfig struct {
	// NameExpr is the name template for the database name.
	// Only ${var} interpolation is supported. All computation belongs in ExpandExprs.
	// Example: "order_${region}_db"
	NameExpr string `yaml:"name_expr" json:"name_expr"`

	// ExpandExprs are variable declarations and computations.
	// := for enumeration declarations, = for runtime computations.
	// Supports @{col} for column refs, ${var} for variables, func() for functions.
	// Example: ["${region} := enum(SG, TH, ID)", "${region} = @{region}"]
	ExpandExprs []string `yaml:"expand_exprs" json:"expand_exprs"`
}

// TableShardConfig configures table-level sharding via expressions.
type TableShardConfig struct {
	// NameExpr is the name template for the physical table name.
	// Only ${var} interpolation is supported. All computation belongs in ExpandExprs.
	// Example: "order_tab_${index}"
	NameExpr string `yaml:"name_expr" json:"name_expr"`

	// ExpandExprs are variable declarations and computations.
	// Supports @{col} for column refs, ${var} for variables, func() for functions.
	// Example: ["${idx} := range(0, 1000)", "${idx} = @{shop_id} % 1000", "${index} = fill(${idx}, 8)"]
	ExpandExprs []string `yaml:"expand_exprs" json:"expand_exprs"`
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

func buildDbTargets(cfg ShardingConfig) ([]dbspi.DbTarget, error) {
	// Multi-server mode
	if len(cfg.Servers) > 0 {
		targets := make([]dbspi.DbTarget, len(cfg.Servers))
		for i, s := range cfg.Servers {
			targets[i] = dbspi.DbTarget{
				Key: s.Key,
				Db:  newDbFromServer(s.ServerConfig, s.DbName),
			}
		}
		return targets, nil
	}

	if cfg.Server == nil {
		return nil, fmt.Errorf("sharding config requires either Server or Servers")
	}

	// Expression-based DB sharding: enumerate all db names
	if cfg.Db != nil && cfg.Db.NameExpr != "" {
		tmpl, err := expr.ParseTemplate(cfg.Db.NameExpr)
		if err != nil {
			return nil, fmt.Errorf("parse db name_expr: %w", err)
		}
		expands, err := expr.ParseExpands(cfg.Db.ExpandExprs)
		if err != nil {
			return nil, fmt.Errorf("parse db expand_exprs: %w", err)
		}

		// Auto-infer identity computations for ${var} refs in template
		autoInferIdentityComputes(tmpl, expands)

		rule := dbsp.NewExprDbRule(tmpl, expands)
		dbNames, err := rule.EnumerateDbNames()
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

	// No db sharding: single database
	return SingleDb(newDbFromServer(*cfg.Server, cfg.Server.DbName)), nil
}

func buildDbRule(cfg *DbShardConfig) (dbspi.DbShardingRule, error) {
	if cfg == nil || cfg.NameExpr == "" {
		return nil, nil
	}

	tmpl, err := expr.ParseTemplate(cfg.NameExpr)
	if err != nil {
		return nil, fmt.Errorf("parse db name_expr: %w", err)
	}
	expands, err := expr.ParseExpands(cfg.ExpandExprs)
	if err != nil {
		return nil, fmt.Errorf("parse db expand_exprs: %w", err)
	}

	autoInferIdentityComputes(tmpl, expands)

	return dbsp.NewExprDbRule(tmpl, expands), nil
}

func buildTableRule(cfg *TableShardConfig) (dbspi.TableShardingRule, error) {
	if cfg == nil || cfg.NameExpr == "" {
		return nil, nil
	}

	tmpl, err := expr.ParseTemplate(cfg.NameExpr)
	if err != nil {
		return nil, fmt.Errorf("parse table name_expr: %w", err)
	}
	expands, err := expr.ParseExpands(cfg.ExpandExprs)
	if err != nil {
		return nil, fmt.Errorf("parse table expand_exprs: %w", err)
	}

	return dbsp.NewExprTableRule(tmpl, expands)
}

// autoInferIdentityComputes checks if a ${var} in the template matches a := declaration
// but has no corresponding = computation. If so, it auto-generates "${var} = @{var}"
// as an identity computation (passthrough from column to variable).
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
