package dbspi

import "context"

type EnhancedExecutor[T Entity] interface {
	Executor[T]

	// SoftDeleteById is a method that sets the soft delete flag to true by id
	SoftDeleteById(ctx context.Context, id any) error

	// SoftDelete is a method that sets the soft delete flag to true
	SoftDeleteByQuery(ctx context.Context, query Query) error

	// RestoreById is a method that sets the soft delete flag to false by id
	RestoreById(ctx context.Context, id any) error

	// RestoreByQuery is a method that sets the soft delete flag to false by query
	RestoreByQuery(ctx context.Context, query Query) error

	// FindNotDeleted is a method that finds the entity without the soft delete flag
	FindNotDeleted(ctx context.Context, query Query, pagination Pagination) ([]T, error)

	// CountNotDeleted is a method that counts the entity without the soft delete flag
	CountNotDeleted(ctx context.Context, query Query) (uint64, error)

	// ExistsByIdNotDeleted is a method that checks if the entity exists without the soft delete flag by id
	ExistsByIdNotDeleted(ctx context.Context, id any) (bool, T, error)

	// ExistsNotDeleted is a method that checks if the entity exists without the soft delete flag
	ExistsNotDeleted(ctx context.Context, query Query) (bool, T, error)
}
