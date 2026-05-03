package dbhelper

import "github.com/MrMiaoMIMI/goshared/db/dbspi"

// WithTxDatabaseGroupKey selects the database group used by a transaction.
//
// If omitted, Transaction uses dbspi.DefaultDatabaseGroupKey.
func WithTxDatabaseGroupKey(databaseGroupKey string) TransactionOption {
	return transactionOptionFunc(func(o *transactionOptions) {
		o.databaseGroupKey = databaseGroupKey
	})
}

// WithTxShardingKey selects the physical database shard used by a transaction.
//
// It is required when the selected database group has database-level sharding.
func WithTxShardingKey(key *dbspi.ShardingKey) TransactionOption {
	return transactionOptionFunc(func(o *transactionOptions) {
		o.shardingKey = key
	})
}

type transactionOptionFunc func(*transactionOptions)

func (f transactionOptionFunc) applyTransactionOption(o *transactionOptions) {
	f(o)
}
