package dbhelper

import (
	"context"
	"fmt"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

// Tx is a transaction-scoped database manager.
//
// Use NewTableStore or NewSoftDeleteTableStore with WithTx to create table stores
// that run on the same underlying database transaction.
type Tx struct {
	manager          *dbsp.Manager
	databaseGroupKey string
	commonFields     dbsp.CommonFieldAutoFillOptions
}

// Transaction runs fn in a single physical database transaction.
//
// The transaction is committed if fn returns nil and rolled back otherwise.
// Use WithTransactionDatabaseGroupKey to select the database group. For db-sharded groups,
// WithTransactionShardingKey is required to select one physical database shard. Local
// transactions do not span multiple database groups or database shards.
// Inside fn, create one or more table stores with NewTableStore or
// NewSoftDeleteTableStore plus WithTx(tx).
func Transaction(ctx context.Context, fn func(tx *Tx) error, opts ...TransactionOption) error {
	if fn == nil {
		return fmt.Errorf("dbhelper: transaction function is nil")
	}

	options := resolveTransactionOptions(opts)
	mgr := asInternalManager(options.manager)
	if mgr == nil {
		mgr = dbsp.DefaultManager()
	}

	databaseGroupKey := options.databaseGroupKey
	if databaseGroupKey == "" {
		databaseGroupKey = dbspi.DefaultDatabaseGroupKey
	}

	commonFields := options.commonFields.apply(mgr.CommonFieldAutoFillOptions())
	return mgr.Transaction(ctx, databaseGroupKey, options.shardingKey, commonFields, func(txMgr *dbsp.Manager) error {
		return fn(&Tx{
			manager:          txMgr,
			databaseGroupKey: databaseGroupKey,
			commonFields:     commonFields,
		})
	})
}

func newTxTableStore[T dbspi.Entity](entity T, tx *Tx, commonFields commonFieldPatch) dbspi.TableStore[T] {
	if tx == nil || tx.manager == nil {
		return dbsp.NewErrorTableStore[T](fmt.Errorf("dbhelper: transaction is nil"))
	}
	if err := validateTxEntityDatabaseGroupKey(tx, entity); err != nil {
		return dbsp.NewErrorTableStore[T](err)
	}
	resolvedCommonFields := commonFields.apply(tx.commonFields)
	return dbsp.ForWithCommonFieldAutoFill(entity, tx.manager, resolvedCommonFields)
}

func newTxSoftDeleteTableStore[T dbspi.Entity](entity T, tx *Tx, commonFields commonFieldPatch) dbspi.SoftDeleteTableStore[T] {
	if tx == nil || tx.manager == nil {
		return dbsp.NewErrorSoftDeleteTableStore[T](fmt.Errorf("dbhelper: transaction is nil"))
	}
	if err := validateTxEntityDatabaseGroupKey(tx, entity); err != nil {
		return dbsp.NewErrorSoftDeleteTableStore[T](err)
	}

	resolvedCommonFields := commonFields.apply(tx.commonFields)
	return dbsp.ForSoftDeleteWithCommonFieldAutoFill(entity, tx.manager, resolvedCommonFields)
}

func resolveTransactionOptions(opts []TransactionOption) transactionOptions {
	var options transactionOptions
	for _, opt := range opts {
		if opt != nil {
			opt.applyTransactionOption(&options)
		}
	}
	return options
}

func validateTxEntityDatabaseGroupKey[T dbspi.Entity](tx *Tx, entity T) error {
	entityDatabaseGroupKey := dbspi.DefaultDatabaseGroupKey
	if provider, ok := any(entity).(dbspi.DatabaseGroupKeyProvider); ok {
		entityDatabaseGroupKey = provider.DatabaseGroupKey()
	}
	if entityDatabaseGroupKey != tx.databaseGroupKey {
		return fmt.Errorf("dbhelper: transaction is bound to database group %q, but entity %q uses database group %q", tx.databaseGroupKey, entity.TableName(), entityDatabaseGroupKey)
	}
	return nil
}
