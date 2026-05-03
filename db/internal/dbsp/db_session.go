package dbsp

import (
	"context"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

type transactionFunc func(tx dbSession) error

type dbSession interface {
	WithModel(entity any) dbSession
	WithTableName(tableName string) dbSession
	Find(ctx context.Context, dest any, query dbspi.Query, pagination dbspi.Pagination) error
	Count(ctx context.Context, query dbspi.Query) (uint64, error)
	Create(ctx context.Context, entity dbspi.Entity) error
	Save(ctx context.Context, entity dbspi.Entity) error
	Update(ctx context.Context, entity dbspi.Entity) error
	Delete(ctx context.Context, entity dbspi.Entity) error
	BatchCreate(ctx context.Context, entities any, batchSize int) error
	BatchSave(ctx context.Context, entities any) error
	UpdateByQuery(ctx context.Context, query dbspi.Query, updater dbspi.Updater) error
	DeleteByQuery(ctx context.Context, entity dbspi.Entity, query dbspi.Query) error
	FirstOrCreate(ctx context.Context, entity dbspi.Entity, query dbspi.Query) error
	Raw(ctx context.Context, dest any, sql string, args ...any) error
	Exec(ctx context.Context, sql string, args ...any) error
	Transaction(ctx context.Context, fn transactionFunc) error
}

type DatabaseTarget struct {
	Key string
	Db  dbSession
}
