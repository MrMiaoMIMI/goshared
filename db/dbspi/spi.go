package dbspi

import (
	"context"
)

// Condition is an empty interface, for different implementations
// For gorm implementation, it will implement gormExpression interface
type Condition interface {}

type Column interface {
    Name() string

	// The following methods are not supported in the current implementation
	// Table() string
	// Alias() string
	// WithTable(table string) Column
	// WithAlias(alias string) Column
}

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


type Query interface {
	Condition
}

type Updater interface {
	Add(column Column, value any) Updater
	AddByMap(columnMap map[Column]any) Updater
	Remove(column Column) Updater
	Params() map[string]any
}

type Tabler interface {
	Table() string
}

type Executor [T Tabler] interface {
	Find(ctx context.Context, query Query, pagenation PaginationConfig) ([]*T, error)
	Count(ctx context.Context, query Query) (int64, error)
	Create(ctx context.Context, value *T) error
	Save(ctx context.Context, value *T) error
	Update(ctx context.Context, query Query, updater Updater) error
	Delete(ctx context.Context, query Query) error
}

type Db interface {
	WithTable(table string) Db
	Find(ctx context.Context, dest any, query Query, pagenation PaginationConfig) error
	Count(ctx context.Context, query Query) (int64, error)
	Create(ctx context.Context, dest any) error
	Save(ctx context.Context, dest any) error
	Update(ctx context.Context, query Query, updater Updater) error
	Delete(ctx context.Context, query Query) error
}