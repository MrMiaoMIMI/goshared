package dbhelper

import (
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

// NewDb creates a Db Instance
func NewDb(dbConfig dbspi.DbConfig) dbspi.Db {
	return dbsp.NewGormDb(dbConfig)
}

// NewDbConfig creates a DbConfig, used for NewDb
func NewDbConfig(host string, port uint, user string, password string, dbName string) dbspi.DbConfig {
	return dbsp.NewDbConfig(host, port, user, password, dbName)
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
