package dbsp

import (
	"context"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

var (
	_ dbspi.SoftDeleteTableStore[_tableForCheck] = new(GormTableStore[_tableForCheck])
)

func NewDefaultDeletedFiled() dbspi.Field[bool] {
	return NewField[bool](dbspi.DefaultDeletedFieldName)
}

// SoftDeleteById implements dbspi.SoftDeleteTableStore
func (e *GormTableStore[T]) SoftDeleteById(ctx context.Context, id any) error {
	updater := NewUpdater().Set(e.getDeletedField(e.emptyEntityInstance), true)
	return e.UpdateById(ctx, id, updater)
}

// SoftDeleteByQuery implements dbspi.SoftDeleteTableStore
func (e *GormTableStore[T]) SoftDeleteByQuery(ctx context.Context, query dbspi.Query) error {
	updater := NewUpdater().Set(e.getDeletedField(e.emptyEntityInstance), true)
	return e.UpdateByQuery(ctx, query, updater)
}

// RestoreById implements dbspi.SoftDeleteTableStore
func (e *GormTableStore[T]) RestoreById(ctx context.Context, id any) error {
	updater := NewUpdater().Set(e.getDeletedField(e.emptyEntityInstance), false)
	return e.UpdateById(ctx, id, updater)
}

// RestoreByQuery implements dbspi.SoftDeleteTableStore
func (e *GormTableStore[T]) RestoreByQuery(ctx context.Context, query dbspi.Query) error {
	updater := NewUpdater().Set(e.getDeletedField(e.emptyEntityInstance), false)
	return e.UpdateByQuery(ctx, query, updater)
}

// FindNotDeleted implements dbspi.SoftDeleteTableStore
func (e *GormTableStore[T]) FindNotDeleted(ctx context.Context, query dbspi.Query, pagination dbspi.Pagination) ([]T, error) {
	return e.Find(ctx, e.withNotDeleted(query), pagination)
}

// CountNotDeleted implements dbspi.SoftDeleteTableStore
func (e *GormTableStore[T]) CountNotDeleted(ctx context.Context, query dbspi.Query) (uint64, error) {
	return e.Count(ctx, e.withNotDeleted(query))
}

// ExistsByIdNotDeleted implements dbspi.SoftDeleteTableStore
func (e *GormTableStore[T]) ExistsByIdNotDeleted(ctx context.Context, id any) (bool, T, error) {
	var entity T
	if id == nil {
		return false, entity, nil
	}
	return e.ExistsNotDeleted(ctx, e.buildQueryById(id))
}

// ExistsNotDeleted implements dbspi.SoftDeleteTableStore
func (e *GormTableStore[T]) ExistsNotDeleted(ctx context.Context, query dbspi.Query) (bool, T, error) {
	return e.Exists(ctx, e.withNotDeleted(query))
}

// getDeletedField returns the deleted field from the entity instance.
// If the entity instance implements SoftDeleteFieldNameProvider interface, it returns the deleted field name from the interface.
// Otherwise, it returns the default deleted field name.
func (e *GormTableStore[T]) getDeletedField(entity dbspi.Entity) dbspi.Field[bool] {
	if namer, ok := entity.(dbspi.SoftDeleteFieldNameProvider); ok {
		return NewField[bool](namer.SoftDeleteFieldName())
	}
	return NewDefaultDeletedFiled()
}

// withNotDeleted appends a `deleted = false` condition to the given query.
// If the query is nil, it returns a query with only the not-deleted condition.
func (e *GormTableStore[T]) withNotDeleted(query dbspi.Query) dbspi.Query {
	falseVal := false
	notDeletedCond := e.getDeletedField(e.emptyEntityInstance).Eq(&falseVal)
	if query == nil {
		return NewQuery(notDeletedCond)
	}
	return And(query, notDeletedCond)
}
