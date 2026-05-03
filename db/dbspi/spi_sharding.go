package dbspi

import (
	"context"
	"errors"
	"fmt"
)

// ErrShardingKeyRequired is returned when a sharded table operation cannot infer
// or receive the required sharding key.
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

// Fields returns a copy of the sharding key values.
func (sk *ShardingKey) Fields() map[string]any {
	fields := make(map[string]any, len(sk.fields))
	for k, v := range sk.fields {
		fields[k] = v
	}
	return fields
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
