package dbsp

import (
	"fmt"
	"strconv"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ================== Table Sharding Rules ==================

var (
	_ dbspi.TableShardingRule = (*hashModTableRule)(nil)
	_ dbspi.ShardCounter     = (*hashModTableRule)(nil)
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

func (r *hashModTableRule) ResolveTable(logicalTable string, key dbspi.ShardingValue) (string, error) {
	hash, err := key.ToUint64()
	if err != nil {
		return "", fmt.Errorf("hash mod table rule: %w", err)
	}
	idx := int(hash % uint64(r.shardCount))
	return logicalTable + fmt.Sprintf(r.suffixFormat, idx), nil
}

func (r *hashModTableRule) ShardCount() int {
	return r.shardCount
}

type customTableRule struct {
	fn func(logicalTable string, key dbspi.ShardingValue) (string, error)
}

func NewCustomTableRule(fn func(logicalTable string, key dbspi.ShardingValue) (string, error)) *customTableRule {
	return &customTableRule{fn: fn}
}

func (r *customTableRule) ResolveTable(logicalTable string, key dbspi.ShardingValue) (string, error) {
	return r.fn(logicalTable, key)
}

// ================== Db Sharding Rules ==================

var (
	_ dbspi.DbShardingRule = (*hashModDbRule)(nil)
	_ dbspi.DbShardingRule = (*rangeDbRule)(nil)
	_ dbspi.DbShardingRule = (*directDbRule)(nil)
	_ dbspi.DbShardingRule = (*customDbRule)(nil)
)

type hashModDbRule struct {
	dbCount int
}

// NewHashModDbRule creates a hash-mod db sharding rule.
// Returns the target key as a stringified int index ("0", "1", ...).
func NewHashModDbRule(dbCount int) *hashModDbRule {
	if dbCount <= 0 {
		panic("dbCount must be positive")
	}
	return &hashModDbRule{dbCount: dbCount}
}

func (r *hashModDbRule) ResolveDbKey(key dbspi.ShardingValue) (string, error) {
	hash, err := key.ToUint64()
	if err != nil {
		return "", fmt.Errorf("hash mod db rule: %w", err)
	}
	idx := int(hash % uint64(r.dbCount))
	return strconv.Itoa(idx), nil
}

type rangeDbRule struct {
	boundaries []int64
}

// NewRangeDbRule creates a range-based db sharding rule.
// boundaries defines the upper bound (exclusive) for each shard.
// Returns target key as a stringified int index ("0", "1", ...).
func NewRangeDbRule(boundaries []int64) *rangeDbRule {
	return &rangeDbRule{boundaries: boundaries}
}

func (r *rangeDbRule) ResolveDbKey(key dbspi.ShardingValue) (string, error) {
	val, err := key.ToInt64()
	if err != nil {
		return "", fmt.Errorf("range db rule: %w", err)
	}
	for i, bound := range r.boundaries {
		if val < bound {
			return strconv.Itoa(i), nil
		}
	}
	return strconv.Itoa(len(r.boundaries)), nil
}

// directDbRule is a pass-through rule that converts the sharding value
// to its string representation and uses it as the target key.
type directDbRule struct{}

func NewDirectDbRule() *directDbRule {
	return &directDbRule{}
}

func (r *directDbRule) ResolveDbKey(key dbspi.ShardingValue) (string, error) {
	return key.String(), nil
}

type customDbRule struct {
	fn func(key dbspi.ShardingValue) (string, error)
}

// NewCustomDbRule creates a custom db sharding rule.
// The fn should return a target key string that matches DbTarget.Key.
func NewCustomDbRule(fn func(key dbspi.ShardingValue) (string, error)) *customDbRule {
	return &customDbRule{fn: fn}
}

func (r *customDbRule) ResolveDbKey(key dbspi.ShardingValue) (string, error) {
	return r.fn(key)
}
