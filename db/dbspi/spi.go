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

// Updater is used to update the entity
type Updater interface {
	Add(column Column, value any) Updater
	AddByMap(columnMap map[Column]any) Updater
	Remove(column Column) Updater
	Params() map[string]any
}

// Ider is the interface for the entity that has an id field
// Generally, it is used with xxById methods from Executor interface
type Ider interface {
	IdFiledName() string
}

type Entity interface {
	TableName() string
}

type Executor[T Entity] interface {
	// Helpful Methods
	// If T implements Ider interface, xxById methods get id field name from Ider.IdFiledName(),
	// otherwise use "id" as the id field name
	GetById(ctx context.Context, id any) (T, error)
	ExistsById(ctx context.Context, id any) (bool, T, error)
	UpdateById(ctx context.Context, id any, updater Updater) error
	DeleteById(ctx context.Context, id any) error

	// Common Methods
	Find(ctx context.Context, query Query, pagenation PaginationConfig) ([]T, error)
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

	// Raw sql methods
	Raw(ctx context.Context, sql string, args ...any) ([]T, error)
	Exec(ctx context.Context, sql string, args ...any) error
}

// Db is the interface for the database
// Generally, you should not use Db methods directly, but use Executor methods instead
type Db interface {
	WithModel(entity any) Db
	WithTableName(tableName string) Db
	Find(ctx context.Context, dest any, query Query, pagenation PaginationConfig) error
	Count(ctx context.Context, query Query) (uint64, error)
	Create(ctx context.Context, entity Entity) error
	Save(ctx context.Context, entity Entity) error
	Update(ctx context.Context, entity Entity) error
	Delete(ctx context.Context, entity Entity) error
	BatchCreate(ctx context.Context, entities any, batchSize int) error
	BatchSave(ctx context.Context, entities any) error
	UpdateByQuery(ctx context.Context, query Query, updater Updater) error
	DeleteByQuery(ctx context.Context, entity Entity, query Query) error
	Raw(ctx context.Context, dest any, sql string, args ...any) error
	Exec(ctx context.Context, sql string, args ...any) error
}
