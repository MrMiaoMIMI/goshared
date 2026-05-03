package dbspi

// DatabaseConfig is the top-level configuration for all databases.
// It maps database group names to their connection and sharding configurations.
// Entities without DatabaseGroupKeyProvider use DefaultDatabaseGroupKey.
type DatabaseConfig struct {
	DatabaseGroups map[string]DatabaseGroupConfig `yaml:"database_groups" json:"database_groups"`
}

// DatabaseGroupConfig configures a single database or a sharded database group.
type DatabaseGroupConfig struct {
	// Connection: DSN string (takes precedence over individual fields).
	DSN string `yaml:"dsn" json:"dsn"`

	// Connection: individual fields.
	Host         string `yaml:"host" json:"host"`
	Port         uint   `yaml:"port" json:"port"`
	User         string `yaml:"user" json:"user"`
	Password     string `yaml:"password" json:"password"`
	DatabaseName string `yaml:"database_name" json:"database_name"`
	Debug        bool   `yaml:"debug" json:"debug"`

	// Connection pool.
	// Zero values use DefaultMaxOpenConns, DefaultMaxIdleConns, and
	// DefaultConnMaxLifetimeSeconds.
	MaxOpenConns           int `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns           int `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetimeSeconds int `yaml:"conn_max_lifetime_seconds" json:"conn_max_lifetime_seconds"`

	// Database-level sharding (expression-based).
	DatabaseSharding *DatabaseShardingConfig `yaml:"database_sharding" json:"database_sharding"`

	// Default table-level sharding (expression-based).
	// Can be overridden per entity via TableRules.
	TableSharding *TableShardingConfig `yaml:"table_sharding" json:"table_sharding"`

	// Per-table sharding overrides.
	TableRules []TableRule `yaml:"table_rules" json:"table_rules"`

	// Multi-server configuration.
	Servers []NamedServerConfig `yaml:"servers" json:"servers"`

	// Max concurrent goroutines for scatter-gather.
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"`
}

// TableRule defines a table sharding override for a group of tables.
type TableRule struct {
	// Tables lists the logical table names this rule applies to.
	Tables []string `yaml:"tables" json:"tables"`

	// TableSharding overrides the database-level default for these tables.
	TableSharding *TableShardingConfig `yaml:"table_sharding" json:"table_sharding"`

	// MaxConcurrency overrides the database-level default.
	MaxConcurrency *int `yaml:"max_concurrency" json:"max_concurrency"`
}

// ServerConfig configures a database server connection.
type ServerConfig struct {
	DSN          string `yaml:"dsn" json:"dsn"`
	Host         string `yaml:"host" json:"host"`
	Port         uint   `yaml:"port" json:"port"`
	User         string `yaml:"user" json:"user"`
	Password     string `yaml:"password" json:"password"`
	DatabaseName string `yaml:"database_name" json:"database_name"`
	Debug        bool   `yaml:"debug" json:"debug"`

	// Zero values use DefaultMaxOpenConns, DefaultMaxIdleConns, and
	// DefaultConnMaxLifetimeSeconds.
	MaxOpenConns           int `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns           int `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetimeSeconds int `yaml:"conn_max_lifetime_seconds" json:"conn_max_lifetime_seconds"`
}

// NamedServerConfig extends ServerConfig with a routing key for multi-server setups.
type NamedServerConfig struct {
	ServerConfig `yaml:",inline" json:",inline"`
	Key          string `yaml:"key" json:"key"`
}

// DatabaseShardingConfig configures database-level sharding via expressions.
type DatabaseShardingConfig struct {
	// NameExpr is the name template for the database name.
	NameExpr string `yaml:"name_expr" json:"name_expr"`

	// ExpandExprs are variable declarations and computations.
	ExpandExprs []string `yaml:"expand_exprs" json:"expand_exprs"`
}

// TableShardingConfig configures table-level sharding via expressions.
type TableShardingConfig struct {
	// NameExpr is the name template for the physical table name.
	NameExpr string `yaml:"name_expr" json:"name_expr"`

	// ExpandExprs are variable declarations and computations.
	ExpandExprs []string `yaml:"expand_exprs" json:"expand_exprs"`
}

// Manager is an opaque database manager handle returned by dbhelper.NewManager.
//
// Do not implement this interface directly. dbhelper only supports Manager
// values returned by dbhelper.NewManager or dbhelper.DefaultManager.
type Manager interface {
	ManagerHandle()
}

type Pagination interface {
	WithLimit(limit *int) Pagination
	WithOffset(offset *int) Pagination
	AppendOrder(order Order) Pagination
	Limit() *int
	Offset() *int
	Orders() []Order
}

type Order interface {
	Column() Column
	Desc() bool
}
