package dbhelper

import (
	"context"
	"fmt"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

// Tx is a transaction-scoped database manager.
//
// Use ForTx or ForEnhanceTx to create table executors that run on the same
// underlying database transaction.
type Tx struct {
	manager      *dbsp.DbManager
	dbKey        string
	commonFields dbspi.CommonFieldOptions
}

// Transaction runs fn in a single physical database transaction.
//
// The transaction is committed if fn returns nil and rolled back otherwise.
// Use WithTxDbKey to select the database group. For db-sharded groups,
// WithTxShardingKey is required to select one physical database shard. Local
// transactions do not span multiple database groups or database shards.
func Transaction(ctx context.Context, fn func(tx *Tx) error, opts ...TransactionOption) error {
	if fn == nil {
		return fmt.Errorf("dbhelper: transaction function is nil")
	}

	options := resolveTransactionOptions(opts)
	mgr := asDbManager(options.manager)
	if mgr == nil {
		mgr = dbsp.DefaultDbManager()
	}

	dbKey := options.dbKey
	if dbKey == "" {
		dbKey = dbspi.DefaultDbKey
	}

	commonFields := options.commonFields.apply(mgr.CommonFieldOptions())
	return mgr.Transaction(ctx, dbKey, options.shardingKey, commonFields, func(txMgr *dbsp.DbManager) error {
		return fn(&Tx{
			manager:      txMgr,
			dbKey:        dbKey,
			commonFields: commonFields,
		})
	})
}

// ForTx creates an Executor for entity within tx.
//
// Multiple ForTx calls on the same Tx may bind different tables, but all of
// them must belong to the database group selected when the transaction started.
func ForTx[T dbspi.Entity](tx *Tx, entity T, opts ...CommonFieldOption) (dbspi.Executor[T], error) {
	if tx == nil || tx.manager == nil {
		return nil, fmt.Errorf("dbhelper: transaction is nil")
	}
	if err := validateTxEntityDbKey(tx, entity); err != nil {
		return nil, err
	}

	var options transactionOptions
	for _, opt := range opts {
		if opt != nil {
			opt.applyTransactionOption(&options)
		}
	}
	commonFields := options.commonFields.apply(tx.commonFields)
	return dbsp.ForWithCommonFields(entity, tx.manager, commonFields), nil
}

// ForEnhanceTx creates an EnhancedExecutor for entity within tx.
func ForEnhanceTx[T dbspi.Entity](tx *Tx, entity T, opts ...CommonFieldOption) (dbspi.EnhancedExecutor[T], error) {
	exec, err := ForTx(tx, entity, opts...)
	if err != nil {
		return nil, err
	}
	enhanced, ok := exec.(dbspi.EnhancedExecutor[T])
	if !ok {
		return nil, fmt.Errorf("dbhelper: transaction executor does not implement EnhancedExecutor")
	}
	return enhanced, nil
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

func validateTxEntityDbKey[T dbspi.Entity](tx *Tx, entity T) error {
	entityDbKey := dbspi.DefaultDbKey
	if provider, ok := any(entity).(dbspi.DbKeyProvider); ok {
		entityDbKey = provider.DbKey()
	}
	if entityDbKey != tx.dbKey {
		return fmt.Errorf("dbhelper: transaction is bound to db %q, but entity %q uses db %q", tx.dbKey, entity.TableName(), entityDbKey)
	}
	return nil
}
