package dbsp

import "github.com/MrMiaoMIMI/goshared/db/dbspi"

// DatabaseShardingRule resolves a ShardingKey to a target key string.
// The returned string is matched against the configured database target key.
type DatabaseShardingRule interface {
	ResolveDatabaseTargetKey(key *dbspi.ShardingKey) (targetKey string, err error)
}

// TableShardingRule resolves the physical table name from the logical table
// name and a ShardingKey.
type TableShardingRule interface {
	ResolveTable(logicalTable string, key *dbspi.ShardingKey) (string, error)
}

// TableShardCounter is an optional interface that table sharding rules can
// implement to declare the total number of shards. Used by FindAll / CountAll
// to enumerate all physical tables.
type TableShardCounter interface {
	ShardCount() int
}

// TableShardEnumerator generates the physical table name for a given shard
// index. Used by scatter-gather (FindAll/CountAll) to enumerate all physical
// tables.
type TableShardEnumerator interface {
	ShardName(logicalTable string, index int) (string, error)
}

// ShardingKeyColumnsProvider is an optional interface that sharding rules can
// implement to declare which @{column} names they require. Used by sharded
// table stores to auto-extract sharding keys from CRUD parameters.
type ShardingKeyColumnsProvider interface {
	RequiredColumns() []string
}
