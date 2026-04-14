package dbspi

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
)

var ErrShardingKeyRequired = errors.New("sharding key is required: " +
	"use Shard(key) or pass via WithShardingKey(ctx, key)")

// ================== ShardingValue ==================

type shardingValueKind int

const (
	kindInt64 shardingValueKind = iota
	kindUint64
	kindString
)

// ShardingValue wraps a typed sharding value with support for string
// conversion (used by named/direct rules) and numeric conversion (used
// by hash_mod/range rules).
type ShardingValue struct {
	intVal  int64
	uintVal uint64
	strVal  string
	kind    shardingValueKind
}

func IntVal(v int64) ShardingValue   { return ShardingValue{intVal: v, kind: kindInt64} }
func UintVal(v uint64) ShardingValue { return ShardingValue{uintVal: v, kind: kindUint64} }
func StrVal(v string) ShardingValue  { return ShardingValue{strVal: v, kind: kindString} }

// String returns a canonical string representation.
func (sv ShardingValue) String() string {
	switch sv.kind {
	case kindInt64:
		return strconv.FormatInt(sv.intVal, 10)
	case kindUint64:
		return strconv.FormatUint(sv.uintVal, 10)
	case kindString:
		return sv.strVal
	}
	return ""
}

// ToUint64 converts the value to uint64 for hash-based sharding.
// String values are hashed via FNV-1a.
func (sv ShardingValue) ToUint64() (uint64, error) {
	switch sv.kind {
	case kindInt64:
		return uint64(sv.intVal), nil
	case kindUint64:
		return sv.uintVal, nil
	case kindString:
		h := fnv.New64a()
		_, _ = h.Write([]byte(sv.strVal))
		return h.Sum64(), nil
	}
	return 0, errors.New("unknown ShardingValue kind")
}

// ToInt64 converts the value to int64 for range-based sharding.
func (sv ShardingValue) ToInt64() (int64, error) {
	switch sv.kind {
	case kindInt64:
		return sv.intVal, nil
	case kindUint64:
		return int64(sv.uintVal), nil
	case kindString:
		return 0, fmt.Errorf("cannot convert string %q to int64", sv.strVal)
	}
	return 0, errors.New("unknown ShardingValue kind")
}

// ================== ShardingKey ==================

// ShardingKey is a composite sharding key that maps DB column names to
// ShardingValue instances. The column names must match the key_field
// values configured in DbShardConfig / TableShardConfig.
type ShardingKey struct {
	fields map[string]ShardingValue
}

// NewShardingKey creates an empty ShardingKey.
func NewShardingKey() *ShardingKey {
	return &ShardingKey{fields: make(map[string]ShardingValue)}
}

// Set stores a sharding value under the given Column's name.
func (sk *ShardingKey) Set(col Column, value ShardingValue) *ShardingKey {
	sk.fields[col.Name()] = value
	return sk
}

// Get retrieves the ShardingValue for the given DB column name.
func (sk *ShardingKey) Get(columnName string) (ShardingValue, error) {
	v, ok := sk.fields[columnName]
	if !ok {
		return ShardingValue{}, fmt.Errorf("sharding key field %q not found", columnName)
	}
	return v, nil
}

// ================== DbTarget ==================

// DbTarget binds a routing key to a Db instance.
// The Key is matched against the string returned by DbShardingRule.ResolveDbKey.
type DbTarget struct {
	Key string
	Db  Db
}

// ================== Sharding Rule Interfaces ==================

// DbShardingRule resolves a ShardingValue to a target key string.
// The returned string is matched against DbTarget.Key to determine which Db to use.
type DbShardingRule interface {
	ResolveDbKey(key ShardingValue) (targetKey string, err error)
}

// TableShardingRule resolves the physical table name from the logical
// table name and a ShardingValue.
type TableShardingRule interface {
	ResolveTable(logicalTable string, key ShardingValue) (string, error)
}

// ShardCounter is an optional interface that table sharding rules can
// implement to declare the total number of shards. Used by FindAll /
// CountAll to enumerate all physical tables without Enumerable.
type ShardCounter interface {
	ShardCount() int
}

// ================== Context helpers ==================

type shardingKeyCtxKey struct{}

// WithShardingKey injects a ShardingKey into the context.
func WithShardingKey(ctx context.Context, key *ShardingKey) context.Context {
	return context.WithValue(ctx, shardingKeyCtxKey{}, key)
}

// ShardingKeyFromCtx extracts the ShardingKey from the context.
func ShardingKeyFromCtx(ctx context.Context) (*ShardingKey, bool) {
	key, ok := ctx.Value(shardingKeyCtxKey{}).(*ShardingKey)
	return key, ok && key != nil
}
