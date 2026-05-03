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

// SelectQuery wraps a Query with specific column selection.
type SelectQuery interface {
	Query
	Columns() []Column
}

// Updater is used to update the entity
type Updater interface {
	Set(column Column, value any) Updater
	SetMap(columnMap map[Column]any) Updater
	Remove(column Column) Updater
	Values() map[string]any
}

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

type Executor[T Entity] interface {
	// Shard routes to a specific shard by the given ShardingKey.
	// Returns the resolved Executor bound to the target database and physical table.
	// For non-sharded Executor, this is a no-op and returns (self, nil).
	Shard(key *ShardingKey) (Executor[T], error)

	// Helpful Methods
	// If T implements IdFieldNamer, xxById methods get id field name from IdFieldName(),
	// otherwise use DefaultIdFieldName as the id field name
	GetById(ctx context.Context, id any) (T, error)
	ExistsById(ctx context.Context, id any) (bool, T, error)
	UpdateById(ctx context.Context, id any, updater Updater) error
	DeleteById(ctx context.Context, id any) error

	// Common Methods
	Find(ctx context.Context, query Query, pagination Pagination) ([]T, error)
	Exists(ctx context.Context, query Query) (bool, T, error)
	Count(ctx context.Context, query Query) (uint64, error)
	Create(ctx context.Context, entity T) error
	Save(ctx context.Context, entity T) error
	Update(ctx context.Context, entity T) error
	Delete(ctx context.Context, entity T) error
	BatchCreate(ctx context.Context, entities []T, batchSize int) error
	BatchSave(ctx context.Context, entities []T) error
	UpdateByQuery(ctx context.Context, query Query, updater Updater) error
	DeleteByQuery(ctx context.Context, query Query) error

	// FirstOrCreate returns the first entity matching the query, creating it if not found.
	FirstOrCreate(ctx context.Context, entity T, query Query) (T, error)

	// Raw sql methods
	Raw(ctx context.Context, sql string, args ...any) ([]T, error)
	Exec(ctx context.Context, sql string, args ...any) error

	// Scatter-gather methods across all shards.
	// For non-sharded Executor, FindAll is equivalent to Find, CountAll is equivalent to Count.
	//
	// FindAll returns ALL matching rows from all shards.
	// batchSize controls the number of rows fetched per batch from each shard.
	// When batchSize > 0, each shard is queried iteratively in batches until exhausted.
	// When batchSize <= 0, each shard is queried all at once (no batching).
	FindAll(ctx context.Context, query Query, batchSize int) ([]T, error)
	CountAll(ctx context.Context, query Query) (uint64, error)
}
