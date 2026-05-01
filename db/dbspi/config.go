package dbspi

// DatabaseConfig is the top-level configuration for all databases.
// It maps database group names to their connection and sharding configurations.
type DatabaseConfig struct {
	Databases map[string]DatabaseEntry `yaml:"databases" json:"databases"`
}

// DatabaseEntry configures a single database or a sharded database group.
type DatabaseEntry struct {
	// Connection: DSN string (takes precedence over individual fields).
	DSN string `yaml:"dsn" json:"dsn"`

	// Connection: individual fields.
	Host     string `yaml:"host" json:"host"`
	Port     uint   `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password" json:"password"`
	DbName   string `yaml:"db_name" json:"db_name"`
	Debug    bool   `yaml:"debug" json:"debug"`

	// Connection pool.
	MaxOpenConns           int `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns           int `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetimeSeconds int `yaml:"conn_max_lifetime_seconds" json:"conn_max_lifetime_seconds"`

	// Database-level sharding (expression-based).
	DbSharding *DbShardConfig `yaml:"db_sharding" json:"db_sharding"`

	// Default table-level sharding (expression-based).
	// Can be overridden per entity via EntityRules.
	TableSharding *TableShardConfig `yaml:"table_sharding" json:"table_sharding"`

	// Per-entity table sharding overrides.
	EntityRules []EntityRule `yaml:"entity_rules" json:"entity_rules"`

	// Multi-server configuration.
	Servers []NamedDbServerConfig `yaml:"servers" json:"servers"`

	// Max concurrent goroutines for scatter-gather.
	MaxConcurrency int `yaml:"max_concurrency" json:"max_concurrency"`
}

// EntityRule defines a table sharding override for a group of tables.
type EntityRule struct {
	// Tables lists the logical table names this rule applies to.
	Tables []string `yaml:"tables" json:"tables"`

	// TableSharding overrides the database-level default for these tables.
	TableSharding *TableShardConfig `yaml:"table_sharding" json:"table_sharding"`

	// MaxConcurrency overrides the database-level default.
	MaxConcurrency *int `yaml:"max_concurrency" json:"max_concurrency"`
}

// DbServerConfig configures a database server connection.
type DbServerConfig struct {
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

// NamedDbServerConfig extends DbServerConfig with a routing key for multi-server setups.
type NamedDbServerConfig struct {
	DbServerConfig `yaml:",inline" json:",inline"`
	Key            string `yaml:"key" json:"key"`
}

// DbShardConfig configures database-level sharding via expressions.
type DbShardConfig struct {
	// NameExpr is the name template for the database name.
	NameExpr string `yaml:"name_expr" json:"name_expr"`

	// ExpandExprs are variable declarations and computations.
	ExpandExprs []string `yaml:"expand_exprs" json:"expand_exprs"`
}

// TableShardConfig configures table-level sharding via expressions.
type TableShardConfig struct {
	// NameExpr is the name template for the physical table name.
	NameExpr string `yaml:"name_expr" json:"name_expr"`

	// ExpandExprs are variable declarations and computations.
	ExpandExprs []string `yaml:"expand_exprs" json:"expand_exprs"`
}

// DbManager is an opaque database manager handle.
// Create one with dbhelper.NewDbManager, then pass it to dbhelper.For or dbhelper.SetDefault.
type DbManager interface {
	DBManager()
}

type PaginationConfig interface {
	WithLimit(limit *int) PaginationConfig
	WithOffset(offset *int) PaginationConfig
	AppendOrder(order OrderConfig) PaginationConfig
	Limit() *int
	Offset() *int
	Orders() []OrderConfig
}

type OrderConfig interface {
	Column() Column
	Desc() bool
}
