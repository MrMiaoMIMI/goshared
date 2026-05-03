package dbhelper

import (
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

// NewPagination creates a pagination config.
func NewPagination() dbspi.Pagination {
	return dbsp.NewPagination()
}

// OrderBy creates an order config.
func OrderBy(column dbspi.Column, desc bool) dbspi.Order {
	return dbsp.OrderBy(column, desc)
}

// Asc creates an ascending order config.
func Asc(column dbspi.Column) dbspi.Order {
	return dbsp.Asc(column)
}

// Desc creates a descending order config.
func Desc(column dbspi.Column) dbspi.Order {
	return dbsp.Desc(column)
}
