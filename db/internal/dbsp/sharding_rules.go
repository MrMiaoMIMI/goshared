package dbsp

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ================== Table Sharding Rules ==================

var (
	_ dbspi.TableShardingRule = (*hashModTableRule)(nil)
	_ dbspi.Enumerable        = (*hashModTableRule)(nil)
	_ dbspi.TableShardingRule = (*customTableRule)(nil)
)

type hashModTableRule struct {
	shardCount   int
	suffixFormat string
}

func NewHashModTableRule(shardCount int) *hashModTableRule {
	return NewHashModTableRuleWithFormat(shardCount, "_%08d")
}

func NewHashModTableRuleWithFormat(shardCount int, suffixFormat string) *hashModTableRule {
	if shardCount <= 0 {
		panic("shardCount must be positive")
	}
	return &hashModTableRule{
		shardCount:   shardCount,
		suffixFormat: suffixFormat,
	}
}

func (r *hashModTableRule) ResolveTable(logicalTable string, key any) (string, error) {
	hash, err := toUint64(key)
	if err != nil {
		return "", fmt.Errorf("hash mod table rule: %w", err)
	}
	idx := int(hash % uint64(r.shardCount))
	return logicalTable + fmt.Sprintf(r.suffixFormat, idx), nil
}

func (r *hashModTableRule) AllKeys() []any {
	keys := make([]any, r.shardCount)
	for i := 0; i < r.shardCount; i++ {
		keys[i] = i
	}
	return keys
}

func (r *hashModTableRule) ShardCount() int {
	return r.shardCount
}

type customTableRule struct {
	fn func(logicalTable string, key any) (string, error)
}

func NewCustomTableRule(fn func(logicalTable string, key any) (string, error)) *customTableRule {
	return &customTableRule{fn: fn}
}

func (r *customTableRule) ResolveTable(logicalTable string, key any) (string, error) {
	return r.fn(logicalTable, key)
}

// ================== Db Sharding Rules ==================

var (
	_ dbspi.DbShardingRule = (*hashModDbRule)(nil)
	_ dbspi.Enumerable     = (*hashModDbRule)(nil)
	_ dbspi.DbShardingRule = (*rangeDbRule)(nil)
	_ dbspi.DbShardingRule = (*customDbRule)(nil)
)

type hashModDbRule struct {
	dbCount int
}

// NewHashModDbRule creates a hash-mod db sharding rule.
// It returns the target key as int (0-based index), which should match
// against DbTarget.Key in the DbTarget list.
func NewHashModDbRule(dbCount int) *hashModDbRule {
	if dbCount <= 0 {
		panic("dbCount must be positive")
	}
	return &hashModDbRule{dbCount: dbCount}
}

func (r *hashModDbRule) ResolveDbKey(key any) (any, error) {
	hash, err := toUint64(key)
	if err != nil {
		return nil, fmt.Errorf("hash mod db rule: %w", err)
	}
	idx := int(hash % uint64(r.dbCount))
	return idx, nil
}

func (r *hashModDbRule) AllKeys() []any {
	keys := make([]any, r.dbCount)
	for i := 0; i < r.dbCount; i++ {
		keys[i] = i
	}
	return keys
}

type rangeDbRule struct {
	boundaries []int64
}

// NewRangeDbRule creates a range-based db sharding rule.
// boundaries defines the upper bound (exclusive) for each shard.
// Returns target key as int (0-based index).
// Example: boundaries=[1000, 2000]
//
//	key < 1000 → target key 0, 1000 <= key < 2000 → target key 1, key >= 2000 → target key 2
func NewRangeDbRule(boundaries []int64) *rangeDbRule {
	return &rangeDbRule{boundaries: boundaries}
}

func (r *rangeDbRule) ResolveDbKey(key any) (any, error) {
	val, err := toInt64(key)
	if err != nil {
		return nil, fmt.Errorf("range db rule: %w", err)
	}
	for i, bound := range r.boundaries {
		if val < bound {
			return i, nil
		}
	}
	return len(r.boundaries), nil
}

type customDbRule struct {
	fn func(key any) (any, error)
}

// NewCustomDbRule creates a custom db sharding rule.
// The fn should return a target key that matches against DbTarget.Key in the DbTarget list.
func NewCustomDbRule(fn func(key any) (any, error)) *customDbRule {
	return &customDbRule{fn: fn}
}

func (r *customDbRule) ResolveDbKey(key any) (any, error) {
	return r.fn(key)
}

// ================== Helper functions ==================

func toUint64(key any) (uint64, error) {
	switch v := key.(type) {
	case int:
		return uint64(v), nil
	case int8:
		return uint64(v), nil
	case int16:
		return uint64(v), nil
	case int32:
		return uint64(v), nil
	case int64:
		return uint64(v), nil
	case uint:
		return uint64(v), nil
	case uint8:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case string:
		h := fnv.New64a()
		_, _ = h.Write([]byte(v))
		return h.Sum64(), nil
	case []byte:
		h := fnv.New64a()
		_, _ = h.Write(v)
		return h.Sum64(), nil
	default:
		b, err := marshalKey(key)
		if err != nil {
			return 0, fmt.Errorf("unsupported sharding key type: %T", key)
		}
		h := fnv.New64a()
		_, _ = h.Write(b)
		return h.Sum64(), nil
	}
}

func toInt64(key any) (int64, error) {
	switch v := key.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unsupported sharding key type for range: %T", key)
	}
}

func marshalKey(key any) ([]byte, error) {
	buf := make([]byte, 8)
	switch v := key.(type) {
	case float32:
		binary.LittleEndian.PutUint32(buf, uint32(v))
		return buf[:4], nil
	case float64:
		binary.LittleEndian.PutUint64(buf, uint64(v))
		return buf, nil
	default:
		return nil, fmt.Errorf("cannot marshal key of type %T", v)
	}
}
