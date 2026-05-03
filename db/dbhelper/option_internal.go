package dbhelper

import "github.com/MrMiaoMIMI/goshared/db/dbspi"

type managerOptions struct {
	commonFields commonFieldPatch
}

type executorOptions struct {
	manager      dbspi.Manager
	tx           *Tx
	setTx        bool
	commonFields commonFieldPatch
}

type transactionOptions struct {
	manager          dbspi.Manager
	databaseGroupKey string
	shardingKey      *dbspi.ShardingKey
	commonFields     commonFieldPatch
}
