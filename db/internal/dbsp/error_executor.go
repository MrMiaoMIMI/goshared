package dbsp

import (
	"context"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

type errorExecutor[T dbspi.Entity] struct {
	err error
}

func NewErrorExecutor[T dbspi.Entity](err error) dbspi.Executor[T] {
	return errorExecutor[T]{err: err}
}

func (e errorExecutor[T]) Shard(*dbspi.ShardingKey) (dbspi.Executor[T], error) {
	return e, e.err
}

func (e errorExecutor[T]) GetById(context.Context, any) (T, error) {
	var zero T
	return zero, e.err
}

func (e errorExecutor[T]) ExistsById(context.Context, any) (bool, T, error) {
	var zero T
	return false, zero, e.err
}

func (e errorExecutor[T]) UpdateById(context.Context, any, dbspi.Updater) error {
	return e.err
}

func (e errorExecutor[T]) DeleteById(context.Context, any) error {
	return e.err
}

func (e errorExecutor[T]) Find(context.Context, dbspi.Query, dbspi.Pagination) ([]T, error) {
	return nil, e.err
}

func (e errorExecutor[T]) Exists(context.Context, dbspi.Query) (bool, T, error) {
	var zero T
	return false, zero, e.err
}

func (e errorExecutor[T]) Count(context.Context, dbspi.Query) (uint64, error) {
	return 0, e.err
}

func (e errorExecutor[T]) Create(context.Context, T) error {
	return e.err
}

func (e errorExecutor[T]) Save(context.Context, T) error {
	return e.err
}

func (e errorExecutor[T]) Update(context.Context, T) error {
	return e.err
}

func (e errorExecutor[T]) Delete(context.Context, T) error {
	return e.err
}

func (e errorExecutor[T]) BatchCreate(context.Context, []T, int) error {
	return e.err
}

func (e errorExecutor[T]) BatchSave(context.Context, []T) error {
	return e.err
}

func (e errorExecutor[T]) UpdateByQuery(context.Context, dbspi.Query, dbspi.Updater) error {
	return e.err
}

func (e errorExecutor[T]) DeleteByQuery(context.Context, dbspi.Query) error {
	return e.err
}

func (e errorExecutor[T]) FirstOrCreate(context.Context, T, dbspi.Query) (T, error) {
	var zero T
	return zero, e.err
}

func (e errorExecutor[T]) Raw(context.Context, string, ...any) ([]T, error) {
	return nil, e.err
}

func (e errorExecutor[T]) Exec(context.Context, string, ...any) error {
	return e.err
}

func (e errorExecutor[T]) FindAll(context.Context, dbspi.Query, int) ([]T, error) {
	return nil, e.err
}

func (e errorExecutor[T]) CountAll(context.Context, dbspi.Query) (uint64, error) {
	return 0, e.err
}

type errorEnhancedExecutor[T dbspi.Entity] struct {
	errorExecutor[T]
}

func NewErrorEnhancedExecutor[T dbspi.Entity](err error) dbspi.EnhancedExecutor[T] {
	return errorEnhancedExecutor[T]{errorExecutor: errorExecutor[T]{err: err}}
}

func (e errorEnhancedExecutor[T]) SoftDeleteById(context.Context, any) error {
	return e.err
}

func (e errorEnhancedExecutor[T]) SoftDeleteByQuery(context.Context, dbspi.Query) error {
	return e.err
}

func (e errorEnhancedExecutor[T]) RestoreById(context.Context, any) error {
	return e.err
}

func (e errorEnhancedExecutor[T]) RestoreByQuery(context.Context, dbspi.Query) error {
	return e.err
}

func (e errorEnhancedExecutor[T]) FindNotDeleted(context.Context, dbspi.Query, dbspi.Pagination) ([]T, error) {
	return nil, e.err
}

func (e errorEnhancedExecutor[T]) CountNotDeleted(context.Context, dbspi.Query) (uint64, error) {
	return 0, e.err
}

func (e errorEnhancedExecutor[T]) ExistsByIdNotDeleted(context.Context, any) (bool, T, error) {
	var zero T
	return false, zero, e.err
}

func (e errorEnhancedExecutor[T]) ExistsNotDeleted(context.Context, dbspi.Query) (bool, T, error) {
	var zero T
	return false, zero, e.err
}
