package dbhelper

import (
	"context"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

// NewDb creates a Db Instance
func NewDb(dbConfig dbspi.DbConfig) dbspi.Db {
	return dbsp.NewGormDb(dbConfig)
}

// DbConfigOption configures DbConfig construction.
type DbConfigOption func(*dbConfigOptions)

type dbConfigOptions struct {
	maxOpenConns           *int
	maxIdleConns           *int
	connMaxLifetimeSeconds *int
	debugMode              *bool
}

// NewDbConfig creates a DbConfig, used for NewDb
func NewDbConfig(host string, port uint, user string, password string, dbName string, opts ...DbConfigOption) dbspi.DbConfig {
	return dbsp.NewDbConfig(host, port, user, password, dbName, toDbspConfigOptions(opts)...)
}

// NewDbConfigFromDSN creates a DbConfig from a raw DSN string.
// Example: NewDbConfigFromDSN("root:pass@tcp(10.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local")
func NewDbConfigFromDSN(dsn string, opts ...DbConfigOption) dbspi.DbConfig {
	return dbsp.NewDbConfigFromDSN(dsn, toDbspConfigOptions(opts)...)
}

// NewDbFromDSN creates a Db instance from a raw DSN string.
func NewDbFromDSN(dsn string, opts ...DbConfigOption) dbspi.Db {
	return NewDb(NewDbConfigFromDSN(dsn, opts...))
}

// WithMaxOpenConns sets the maximum number of open connections to the database.
func WithMaxOpenConns(n int) DbConfigOption {
	return func(o *dbConfigOptions) {
		o.maxOpenConns = &n
	}
}

// WithMaxIdleConns sets the maximum number of idle connections in the pool.
func WithMaxIdleConns(n int) DbConfigOption {
	return func(o *dbConfigOptions) {
		o.maxIdleConns = &n
	}
}

// WithConnMaxLifetimeSeconds sets the maximum lifetime of a connection in seconds.
func WithConnMaxLifetimeSeconds(s int) DbConfigOption {
	return func(o *dbConfigOptions) {
		o.connMaxLifetimeSeconds = &s
	}
}

// WithDebugMode enables or disables GORM debug logging.
func WithDebugMode(debug bool) DbConfigOption {
	return func(o *dbConfigOptions) {
		o.debugMode = &debug
	}
}

func toDbspConfigOptions(opts []DbConfigOption) []dbsp.DbConfigOption {
	values := &dbConfigOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(values)
		}
	}

	internalOpts := make([]dbsp.DbConfigOption, 0, 4)
	if values.maxOpenConns != nil {
		internalOpts = append(internalOpts, dbsp.WithMaxOpenConns(*values.maxOpenConns))
	}
	if values.maxIdleConns != nil {
		internalOpts = append(internalOpts, dbsp.WithMaxIdleConns(*values.maxIdleConns))
	}
	if values.connMaxLifetimeSeconds != nil {
		internalOpts = append(internalOpts, dbsp.WithConnMaxLifetimeSeconds(*values.connMaxLifetimeSeconds))
	}
	if values.debugMode != nil {
		internalOpts = append(internalOpts, dbsp.WithDebugMode(*values.debugMode))
	}
	return internalOpts
}

// NewField creates a T type field, used for query and update
func NewField[T any](columnName string) dbspi.Field[T] {
	return dbsp.NewField[T](columnName)
}

// Q creates a new Query with AND keyword, it is a shortcut for And
func Q(conditions ...dbspi.Condition) dbspi.Query {
	return dbsp.And(conditions...)
}

// And creates a new Query with AND keyword
func And(conditions ...dbspi.Condition) dbspi.Query {
	return dbsp.And(conditions...)
}

// Or creates a new Query with OR keyword
func Or(conditions ...dbspi.Condition) dbspi.Query {
	return dbsp.Or(conditions...)
}

// Not creates a new Query with NOT keyword
func Not(condition dbspi.Condition) dbspi.Query {
	return dbsp.Not(condition)
}

// Select creates a query that only returns specific columns.
// Example:
//
//	nameField := dbhelper.NewField[string]("name")
//	ageField := dbhelper.NewField[int]("age")
//	results, err := executor.Find(ctx, dbhelper.Select(
//	    []dbspi.Column{nameField, ageField},
//	    nameField.Like(ptrutil.Of("%test%")),
//	), nil)
func Select(columns []dbspi.Column, conditions ...dbspi.Condition) dbspi.SelectQuery {
	return dbsp.Select(columns, conditions...)
}

// NewExecutor creates a new Executor for Table T
// Example:
// NewExecutor(db, &User{})
func NewExecutor[T dbspi.Entity](db dbspi.Db, entityInstance T) dbspi.Executor[T] {
	return dbsp.NewExecutor(db, entityInstance)
}

// NewExecutorWithTableName creates a new Executor for Table T with the given table name
// Example:
// NewExecutorWithTableName(db, &User{}, "user_tab_00000001")
func NewExecutorWithTableName[T dbspi.Entity](db dbspi.Db, entityInstance T, tableName string) dbspi.Executor[T] {
	return dbsp.NewExecutorWithTableName(db, entityInstance, tableName)
}

// NewEnhancedExecutor creates a new EnhancedExecutor for Table T
// Example:
// NewEnhancedExecutor(db, &User{})
func NewEnhancedExecutor[T dbspi.Entity](db dbspi.Db, entityInstance T) dbspi.EnhancedExecutor[T] {
	return dbsp.NewExecutor(db, entityInstance)
}

// NewEnhancedExecutorWithTableName creates a new EnhancedExecutor for Table T with the given table name
// Example:
// NewEnhancedExecutorWithTableName(db, &User{}, "user_tab_00000001")
func NewEnhancedExecutorWithTableName[T dbspi.Entity](db dbspi.Db, entityInstance T, tableName string) dbspi.EnhancedExecutor[T] {
	return dbsp.NewExecutorWithTableName(db, entityInstance, tableName)
}

// Transaction runs fn within a database transaction.
// The transaction is committed if fn returns nil; rolled back otherwise.
//
// Example:
//
//	err := dbhelper.Transaction(ctx, db, func(tx dbspi.Db) error {
//	    exec := dbhelper.NewExecutor(tx, &User{})
//	    if err := exec.Create(ctx, user); err != nil {
//	        return err
//	    }
//	    return exec.Create(ctx, profile)
//	})
func Transaction(ctx context.Context, db dbspi.Db, fn dbspi.TxFn) error {
	return db.Transaction(ctx, fn)
}

// NewUpdater creates a new Updater
func NewUpdater() dbspi.Updater {
	return dbsp.NewUpdater()
}

// NewPaginationConfig creates a PaginationConfig
func NewPaginationConfig() dbspi.PaginationConfig {
	return dbsp.NewPaginationConfig()
}

// NewOrderConfig creates an OrderConfig
func NewOrderConfig(column dbspi.Column, desc bool) dbspi.OrderConfig {
	return dbsp.NewOrderConfig(column, desc)
}
