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
	updater := NewUpdater().Add(e.getDeletedField(e.emptyEntityInstance), true)
	return e.UpdateById(ctx, id, updater)
}

// SoftDeleteByQuery implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) SoftDeleteByQuery(ctx context.Context, query dbspi.Query) error {
	updater := NewUpdater().Add(e.getDeletedField(e.emptyEntityInstance), true)
	return e.UpdateByQuery(ctx, query, updater)
}

// RecoverFromDeletedById implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) RecoverFromDeletedById(ctx context.Context, id any) error {
	updater := NewUpdater().Add(e.getDeletedField(e.emptyEntityInstance), false)
	return e.UpdateById(ctx, id, updater)
}

// RecoverFromDeletedByQuery implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) RecoverFromDeletedByQuery(ctx context.Context, query dbspi.Query) error {
	updater := NewUpdater().Add(e.getDeletedField(e.emptyEntityInstance), false)
	return e.UpdateByQuery(ctx, query, updater)
}

// FindWithoutDeleted implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) FindWithoutDeleted(ctx context.Context, query dbspi.Query, pagenation dbspi.PaginationConfig) ([]T, error) {
	return e.Find(ctx, e.withNotDeleted(query), pagenation)
}

// CountWithoutDeleted implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) CountWithoutDeleted(ctx context.Context, query dbspi.Query) (uint64, error) {
	return e.Count(ctx, e.withNotDeleted(query))
}

// ExistsByIdWithoutDeleted implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) ExistsByIdWithoutDeleted(ctx context.Context, id any) (bool, T, error) {
	var entity T
	if id == nil {
		return false, entity, nil
	}
	return e.ExistsWithoutDeleted(ctx, e.buildQueryById(id))
}

// ExistsWithoutDeleted implements dbspi.EnhancedExecutor
func (e *GormExecutor[T]) ExistsWithoutDeleted(ctx context.Context, query dbspi.Query) (bool, T, error) {
	return e.Exists(ctx, e.withNotDeleted(query))
}

// getDeletedField returns the deleted field from the entity instance.
// If the entity instance implements DeletedFieldNamer interface, it returns the deleted field name from the interface.
// Otherwise, it returns the default deleted field name.
func (e *GormExecutor[T]) getDeletedField(entity dbspi.Entity) dbspi.Field[bool] {
	if namer, ok := entity.(dbspi.DeletedFieldNamer); ok {
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
