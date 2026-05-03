package dbspi

import "context"

// SoftDeleteTableStore extends TableStore with soft-delete operations.
type SoftDeleteTableStore[T Entity] interface {
	TableStore[T]

	// SoftDeleteById sets the soft-delete flag to true by id.
	SoftDeleteById(ctx context.Context, id any) error

	// SoftDeleteByQuery sets the soft-delete flag to true by query.
	SoftDeleteByQuery(ctx context.Context, query Query) error

	// RestoreById sets the soft-delete flag to false by id.
	RestoreById(ctx context.Context, id any) error

	// RestoreByQuery sets the soft-delete flag to false by query.
	RestoreByQuery(ctx context.Context, query Query) error

	// FindNotDeleted finds entities where the soft-delete flag is false.
	FindNotDeleted(ctx context.Context, query Query, pagination Pagination) ([]T, error)

	// CountNotDeleted counts entities where the soft-delete flag is false.
	CountNotDeleted(ctx context.Context, query Query) (uint64, error)

	// ExistsByIdNotDeleted checks whether an entity exists by id with the soft-delete flag false.
	ExistsByIdNotDeleted(ctx context.Context, id any) (bool, T, error)

	// ExistsNotDeleted checks whether an entity exists by query with the soft-delete flag false.
	ExistsNotDeleted(ctx context.Context, query Query) (bool, T, error)
}
