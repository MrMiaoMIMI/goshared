package dbspi

import (
	"context"
	"errors"
	"fmt"
)

var ErrShardingKeyRequired = errors.New("sharding key is required: " +
	"use Shard(key) or pass via WithShardingKey(ctx, key)")

// ================== ShardingKey ==================

// ShardingKey is a composite sharding key that maps column names to values.
// Values can be int64, int, uint64, or string.
type ShardingKey struct {
	fields map[string]any
}

// NewShardingKey creates an empty ShardingKey.
func NewShardingKey() *ShardingKey {
	return &ShardingKey{fields: make(map[string]any)}
}

// Set stores a value under the given Column's name.
func (sk *ShardingKey) Set(col Column, value any) *ShardingKey {
	sk.fields[col.Name()] = value
	return sk
}

// SetValue stores a value under the given column name.
func (sk *ShardingKey) SetValue(name string, value any) *ShardingKey {
	sk.fields[name] = value
	return sk
}

// Get retrieves the value for the given column name.
func (sk *ShardingKey) Get(name string) (any, error) {
	v, ok := sk.fields[name]
	if !ok {
		return nil, fmt.Errorf("sharding key field %q not found", name)
	}
	return v, nil
}

// Fields returns the underlying map (read-only usage intended).
func (sk *ShardingKey) Fields() map[string]any {
	return sk.fields
}

// ================== Sharding Rule Interfaces ==================

// DatabaseShardingRule resolves a ShardingKey to a target key string.
// The returned string is matched against the configured database target key.
type DatabaseShardingRule interface {
	ResolveDatabaseTargetKey(key *ShardingKey) (targetKey string, err error)
}

// TableShardingRule resolves the physical table name from the logical
// table name and a ShardingKey.
type TableShardingRule interface {
	ResolveTable(logicalTable string, key *ShardingKey) (string, error)
}

// ShardCounter is an optional interface that table sharding rules can
// implement to declare the total number of shards. Used by FindAll /
// CountAll to enumerate all physical tables.
type ShardCounter interface {
	ShardCount() int
}

// ShardEnumerator generates the physical table name for a given shard index.
// Used by scatter-gather (FindAll/CountAll) to enumerate all physical tables.
type ShardEnumerator interface {
	ShardName(logicalTable string, index int) (string, error)
}

// ShardingKeyColumnsProvider is an optional interface that sharding rules
// can implement to declare which @{column} names they require.
// Used by sharded executors to auto-extract sharding keys from CRUD parameters.
type ShardingKeyColumnsProvider interface {
	RequiredColumns() []string
}

// ================== Context helpers ==================

type shardingKeyCtxKey struct{}

// WithShardingKey injects a ShardingKey into the context.
func WithShardingKey(ctx context.Context, key *ShardingKey) context.Context {
	return context.WithValue(ctx, shardingKeyCtxKey{}, key)
}

// ShardingKeyFromContext extracts the ShardingKey from the context.
func ShardingKeyFromContext(ctx context.Context) (*ShardingKey, bool) {
	key, ok := ctx.Value(shardingKeyCtxKey{}).(*ShardingKey)
	return key, ok && key != nil
}
