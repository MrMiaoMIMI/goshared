package dbspi

import "context"

type EnhancedExecutor[T Entity] interface {
	Executor[T]

	// SoftDeleteById is a method that sets the soft delete flag to true by id
	SoftDeleteById(ctx context.Context, id any) error

	// SoftDelete is a method that sets the soft delete flag to true
	SoftDeleteByQuery(ctx context.Context, query Query) error

	// RecoverFromDeletedById is a method that sets the soft delete flag to false by id
	RecoverFromDeletedById(ctx context.Context, id any) error

	// RecoverFromDeletedByQuery is a method that sets the soft delete flag to false by query
	RecoverFromDeletedByQuery(ctx context.Context, query Query) error

	// FindWithoutDeleted is a method that finds the entity without the soft delete flag
	FindWithoutDeleted(ctx context.Context, query Query, pagenation PaginationConfig) ([]T, error)

	// CountWithoutDeleted is a method that counts the entity without the soft delete flag
	CountWithoutDeleted(ctx context.Context, query Query) (uint64, error)

	// ExistsByIdWithoutDeleted is a method that checks if the entity exists without the soft delete flag by id
	ExistsByIdWithoutDeleted(ctx context.Context, id any) (bool, T, error)

	// ExistsWithoutDeleted is a method that checks if the entity exists without the soft delete flag
	ExistsWithoutDeleted(ctx context.Context, query Query) (bool, T, error)
}

type OptionalDeletedField interface {
	DeletedFiledName() string
}
