package dbspi

import (
	"context"
)

// Condition is an empty interface, for different implementations
// For gorm implementation, it will implement gormExpression interface
type Condition any

// Column is the table column
type Column interface {
	Name() string
}

// Field is the table column and query condition builder, it is used to build Query and Updater
type Field[T any] interface {
	// Field is a column
	Column

	// Common Methods
	IsNull() Condition
	IsNotNull() Condition
	Eq(v *T) Condition
	NotEq(v *T) Condition
	In(v []T) Condition
	NotIn(v []T) Condition

	// Number Methods
	Gt(v *T) Condition
	GtEq(v *T) Condition
	Lt(v *T) Condition
	LtEq(v *T) Condition
	Between(min, max *T) Condition

	// String Methods
	Like(v *string) Condition
	NotLike(v *string) Condition
	StartsWith(v *string) Condition
	EndsWith(v *string) Condition
	Contains(v *string) Condition
	NotContains(v *string) Condition
}

// Query is used to query entities from the database
type Query interface {
	Condition
}

// Updater builds column updates for UpdateById and UpdateByQuery.
//
// Use dbhelper.NewUpdater to create values understood by the default table store
// implementation.
type Updater interface {
	Set(column Column, value any) Updater
	SetMap(columnMap map[Column]any) Updater
	Remove(column Column) Updater
}

// Entity is a database model that declares its logical table name.
type Entity interface {
	TableName() string
}

// DatabaseGroupKeyProvider is an optional interface for entities to declare which
// database configuration key they belong to.
// Used by Manager to route entities to the correct database/sharding group.
// If not implemented, DefaultDatabaseGroupKey is used.
type DatabaseGroupKeyProvider interface {
	DatabaseGroupKey() string
}

// TableStore provides typed CRUD and query operations for one logical table.
type TableStore[T Entity] interface {
	// Shard explicitly binds subsequent operations to one physical shard.
	//
	// Most business code should rely on auto-key routing through regular
	// methods such as Find, Count, Create, Update, or Delete. Use Shard only
	// when the sharding key cannot be inferred from the entity, query, or
	// context, or for diagnostics, repair, and other advanced operations.
	// For non-sharded TableStore, this is a no-op and returns (self, nil).
	Shard(key *ShardingKey) (TableStore[T], error)

	// ID helpers use IdFieldNameProvider when T implements it. Otherwise they
	// use DefaultIdFieldName.
	GetById(ctx context.Context, id any) (T, error)
	ExistsById(ctx context.Context, id any) (bool, T, error)
	UpdateById(ctx context.Context, id any, updater Updater) error
	DeleteById(ctx context.Context, id any) error

	// Query methods.
	Find(ctx context.Context, query Query, pagination Pagination) ([]T, error)
	Exists(ctx context.Context, query Query) (bool, T, error)
	Count(ctx context.Context, query Query) (uint64, error)

	// Entity mutation methods.
	Create(ctx context.Context, entity T) error
	Save(ctx context.Context, entity T) error
	Update(ctx context.Context, entity T) error
	Delete(ctx context.Context, entity T) error
	BatchCreate(ctx context.Context, entities []T, batchSize int) error
	BatchSave(ctx context.Context, entities []T) error

	// Query-based mutation methods.
	UpdateByQuery(ctx context.Context, query Query, updater Updater) error
	DeleteByQuery(ctx context.Context, query Query) error

	// FirstOrCreate returns the first entity matching the query, creating it if not found.
	FirstOrCreate(ctx context.Context, entity T, query Query) (T, error)

	// Scatter-gather methods across all shards.
	// For non-sharded TableStore, FindAll is equivalent to Find, CountAll is equivalent to Count.
	//
	// FindAll returns ALL matching rows from all shards.
	// batchSize controls the number of rows fetched per batch from each shard.
	// When batchSize > 0, each shard is queried iteratively in batches until exhausted.
	// When batchSize <= 0, each shard is queried all at once (no batching).
	FindAll(ctx context.Context, query Query, batchSize int) ([]T, error)
	CountAll(ctx context.Context, query Query) (uint64, error)
}

// SQLTableStore provides advanced raw SQL execution for a table store.
//
// Prefer TableStore methods for regular business reads and writes. For sharded
// tables, raw SQL execution must be routed explicitly, either by calling Shard
// first or by passing a ShardingKey in ctx.
type SQLTableStore[T Entity] interface {
	// Raw runs a query and scans rows into T.
	Raw(ctx context.Context, sql string, args ...any) ([]T, error)

	// Exec runs a SQL statement without returning rows.
	Exec(ctx context.Context, sql string, args ...any) error
}
