package dbsp

import (
	"context"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

var (
	_ dbspi.EnhancedExecutor[_tableForCheck] = new(GormExecutor[_tableForCheck])
)

func NewDefaultDeletedFiled() dbspi.Field[bool] {
	return NewField[bool](dbspi.DefaultDeletedFieldName)
}

// SoftDeleteById implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) SoftDeleteById(ctx context.Context, id any) error {
	updater := NewUpdater().Set(e.getDeletedField(e.emptyEntityInstance), true)
	return e.UpdateById(ctx, id, updater)
}

// SoftDeleteByQuery implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) SoftDeleteByQuery(ctx context.Context, query dbspi.Query) error {
	updater := NewUpdater().Set(e.getDeletedField(e.emptyEntityInstance), true)
	return e.UpdateByQuery(ctx, query, updater)
}

// RestoreById implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) RestoreById(ctx context.Context, id any) error {
	updater := NewUpdater().Set(e.getDeletedField(e.emptyEntityInstance), false)
	return e.UpdateById(ctx, id, updater)
}

// RestoreByQuery implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) RestoreByQuery(ctx context.Context, query dbspi.Query) error {
	updater := NewUpdater().Set(e.getDeletedField(e.emptyEntityInstance), false)
	return e.UpdateByQuery(ctx, query, updater)
}

// FindNotDeleted implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) FindNotDeleted(ctx context.Context, query dbspi.Query, pagination dbspi.Pagination) ([]T, error) {
	return e.Find(ctx, e.withNotDeleted(query), pagination)
}

// CountNotDeleted implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) CountNotDeleted(ctx context.Context, query dbspi.Query) (uint64, error) {
	return e.Count(ctx, e.withNotDeleted(query))
}

// ExistsByIdNotDeleted implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) ExistsByIdNotDeleted(ctx context.Context, id any) (bool, T, error) {
	var entity T
	if id == nil {
		return false, entity, nil
	}
	return e.ExistsNotDeleted(ctx, e.buildQueryById(id))
}

// ExistsNotDeleted implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) ExistsNotDeleted(ctx context.Context, query dbspi.Query) (bool, T, error) {
	return e.Exists(ctx, e.withNotDeleted(query))
}

// getDeletedField returns the deleted field from the entity instance.
// If the entity instance implements SoftDeleteFieldNamer interface, it returns the deleted field name from the interface.
// Otherwise, it returns the default deleted field name.
func (e *GormExecutor[T]) getDeletedField(entity dbspi.Entity) dbspi.Field[bool] {
	if namer, ok := entity.(dbspi.SoftDeleteFieldNamer); ok {
		return NewField[bool](namer.DeletedFieldName())
	}
	return NewDefaultDeletedFiled()
}

// withNotDeleted appends a `deleted = false` condition to the given query.
// If the query is nil, it returns a query with only the not-deleted condition.
func (e *GormExecutor[T]) withNotDeleted(query dbspi.Query) dbspi.Query {
	falseVal := false
	notDeletedCond := e.getDeletedField(e.emptyEntityInstance).Eq(&falseVal)
	if query == nil {
		return NewQuery(notDeletedCond)
	}
	return And(query, notDeletedCond)
}
