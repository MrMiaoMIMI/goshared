package dbspi

import (
	"context"
	"errors"
)

var ErrShardingKeyRequired = errors.New("sharding key is required: " +
	"use Shard(key) or pass via WithShardingKey(ctx, key)")

// DbTarget binds a routing key to a Db instance.
// The Key is used by DbShardingRule to match the resolved target key.
type DbTarget struct {
	Key any
	Db  Db
}

// DbShardingRule resolves the sharding key to a target key.
// The returned target key is matched against DbTarget.Key in the DbTarget list
// to determine which Db to use.
type DbShardingRule interface {
	ResolveDbKey(key any) (targetKey any, err error)
}

// TableShardingRule resolves the physical table name by the logical table name and sharding key.
type TableShardingRule interface {
	ResolveTable(logicalTable string, key any) (string, error)
}

// Enumerable is an optional interface that sharding rules can implement
// to support scatter-gather operations (FindAll / CountAll).
// It returns all possible sharding keys for enumeration.
type Enumerable interface {
	AllKeys() []any
}

type shardingKeyCtxKey struct{}

// WithShardingKey injects a sharding key into the context.
func WithShardingKey(ctx context.Context, key any) context.Context {
	return context.WithValue(ctx, shardingKeyCtxKey{}, key)
}

// ShardingKeyFromCtx extracts the sharding key from the context.
func ShardingKeyFromCtx(ctx context.Context) (any, bool) {
	key := ctx.Value(shardingKeyCtxKey{})
	return key, key != nil
}
