package dbhelper

import (
	"context"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

// NewField creates a typed table field for query and update builders.
func NewField[T any](columnName string) dbspi.Field[T] {
	return dbsp.NewField[T](columnName)
}

// Q creates a query with AND semantics. It is a shortcut for And.
func Q(conditions ...dbspi.Condition) dbspi.Query {
	return dbsp.And(conditions...)
}

// And creates a query with AND semantics.
func And(conditions ...dbspi.Condition) dbspi.Query {
	return dbsp.And(conditions...)
}

// Or creates a query with OR semantics.
func Or(conditions ...dbspi.Condition) dbspi.Query {
	return dbsp.Or(conditions...)
}

// Not creates a query with NOT semantics.
func Not(condition dbspi.Condition) dbspi.Query {
	return dbsp.Not(condition)
}

// Select creates a query that only returns specific columns.
func Select(columns []dbspi.Column, conditions ...dbspi.Condition) dbspi.SelectQuery {
	return dbsp.Select(columns, conditions...)
}

// NewUpdater creates an updater for update operations.
func NewUpdater() dbspi.Updater {
	return dbsp.NewUpdater()
}

// NewPaginationConfig creates a pagination config.
func NewPaginationConfig() dbspi.PaginationConfig {
	return dbsp.NewPaginationConfig()
}

// NewOrderConfig creates an order config.
func NewOrderConfig(column dbspi.Column, desc bool) dbspi.OrderConfig {
	return dbsp.NewOrderConfig(column, desc)
}

// Transaction runs fn within a database transaction.
// The transaction is committed if fn returns nil, and rolled back otherwise.
func Transaction(ctx context.Context, db dbspi.Db, fn dbspi.TxFn) error {
	return db.Transaction(ctx, fn)
}
