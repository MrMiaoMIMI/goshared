package dbsp

import (
	"context"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

type errorTableStore[T dbspi.Entity] struct {
	err error
}

func NewErrorTableStore[T dbspi.Entity](err error) dbspi.TableStore[T] {
	return errorTableStore[T]{err: err}
}

func (e errorTableStore[T]) Shard(*dbspi.ShardingKey) (dbspi.TableStore[T], error) {
	return e, e.err
}

func (e errorTableStore[T]) GetById(context.Context, any) (T, error) {
	var zero T
	return zero, e.err
}

func (e errorTableStore[T]) ExistsById(context.Context, any) (bool, T, error) {
	var zero T
	return false, zero, e.err
}

func (e errorTableStore[T]) UpdateById(context.Context, any, dbspi.Updater) error {
	return e.err
}

func (e errorTableStore[T]) DeleteById(context.Context, any) error {
	return e.err
}

func (e errorTableStore[T]) Find(context.Context, dbspi.Query, dbspi.Pagination) ([]T, error) {
	return nil, e.err
}

func (e errorTableStore[T]) Exists(context.Context, dbspi.Query) (bool, T, error) {
	var zero T
	return false, zero, e.err
}

func (e errorTableStore[T]) Count(context.Context, dbspi.Query) (uint64, error) {
	return 0, e.err
}

func (e errorTableStore[T]) Create(context.Context, T) error {
	return e.err
}

func (e errorTableStore[T]) Save(context.Context, T) error {
	return e.err
}

func (e errorTableStore[T]) Update(context.Context, T) error {
	return e.err
}

func (e errorTableStore[T]) Delete(context.Context, T) error {
	return e.err
}

func (e errorTableStore[T]) BatchCreate(context.Context, []T, int) error {
	return e.err
}

func (e errorTableStore[T]) BatchSave(context.Context, []T) error {
	return e.err
}

func (e errorTableStore[T]) UpdateByQuery(context.Context, dbspi.Query, dbspi.Updater) error {
	return e.err
}

func (e errorTableStore[T]) DeleteByQuery(context.Context, dbspi.Query) error {
	return e.err
}

func (e errorTableStore[T]) FirstOrCreate(context.Context, T, dbspi.Query) (T, error) {
	var zero T
	return zero, e.err
}

func (e errorTableStore[T]) Raw(context.Context, string, ...any) ([]T, error) {
	return nil, e.err
}

func (e errorTableStore[T]) Exec(context.Context, string, ...any) error {
	return e.err
}

func (e errorTableStore[T]) FindAll(context.Context, dbspi.Query, int) ([]T, error) {
	return nil, e.err
}

func (e errorTableStore[T]) CountAll(context.Context, dbspi.Query) (uint64, error) {
	return 0, e.err
}

type errorSoftDeleteTableStore[T dbspi.Entity] struct {
	errorTableStore[T]
}

func NewErrorSoftDeleteTableStore[T dbspi.Entity](err error) dbspi.SoftDeleteTableStore[T] {
	return errorSoftDeleteTableStore[T]{errorTableStore: errorTableStore[T]{err: err}}
}

func (e errorSoftDeleteTableStore[T]) SoftDeleteById(context.Context, any) error {
	return e.err
}

func (e errorSoftDeleteTableStore[T]) SoftDeleteByQuery(context.Context, dbspi.Query) error {
	return e.err
}

func (e errorSoftDeleteTableStore[T]) RestoreById(context.Context, any) error {
	return e.err
}

func (e errorSoftDeleteTableStore[T]) RestoreByQuery(context.Context, dbspi.Query) error {
	return e.err
}

func (e errorSoftDeleteTableStore[T]) FindNotDeleted(context.Context, dbspi.Query, dbspi.Pagination) ([]T, error) {
	return nil, e.err
}

func (e errorSoftDeleteTableStore[T]) CountNotDeleted(context.Context, dbspi.Query) (uint64, error) {
	return 0, e.err
}

func (e errorSoftDeleteTableStore[T]) ExistsByIdNotDeleted(context.Context, any) (bool, T, error) {
	var zero T
	return false, zero, e.err
}

func (e errorSoftDeleteTableStore[T]) ExistsNotDeleted(context.Context, dbspi.Query) (bool, T, error) {
	var zero T
	return false, zero, e.err
}
