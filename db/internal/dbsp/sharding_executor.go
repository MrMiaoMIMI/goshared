package dbsp

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"golang.org/x/sync/errgroup"
)

// ShardedExecutorConfig is the internal config for creating a sharded executor.
// dbhelper converts its ShardOption list into this config.
type ShardedExecutorConfig struct {
	Dbs            []dbspi.DbTarget
	DbRule         dbspi.DbShardingRule
	TableRule      dbspi.TableShardingRule
	MaxConcurrency int // max concurrent goroutines for FindAll/CountAll, 0 = unlimited
}

type shardedExecutor[T dbspi.Entity] struct {
	entity         T
	dbs            []dbspi.DbTarget
	dbRule         dbspi.DbShardingRule
	tableRule      dbspi.TableShardingRule
	maxConcurrency int
}

func NewShardedExecutor[T dbspi.Entity](entity T, cfg ShardedExecutorConfig) *shardedExecutor[T] {
	if len(cfg.Dbs) == 0 {
		panic("sharded executor requires at least one DbTarget via WithDbs")
	}
	if cfg.TableRule == nil && cfg.DbRule == nil {
		panic("sharded executor requires at least one of WithTableRule or WithDbRule")
	}

	return &shardedExecutor[T]{
		entity:         entity,
		dbs:            cfg.Dbs,
		dbRule:         cfg.DbRule,
		tableRule:      cfg.TableRule,
		maxConcurrency: cfg.MaxConcurrency,
	}
}

// findDb looks up the Db by matching the target key against DbTarget.Key in the list.
func (e *shardedExecutor[T]) findDb(targetKey any) (dbspi.Db, error) {
	for _, dt := range e.dbs {
		if fmt.Sprintf("%v", dt.Key) == fmt.Sprintf("%v", targetKey) {
			return dt.Db, nil
		}
	}
	return nil, fmt.Errorf("no DbTarget found for key: %v", targetKey)
}

// resolve determines the target Db and physical table name for the given sharding key.
func (e *shardedExecutor[T]) resolve(key any) (dbspi.Db, string, error) {
	var db dbspi.Db

	if e.dbRule != nil {
		targetKey, err := e.dbRule.ResolveDbKey(key)
		if err != nil {
			return nil, "", fmt.Errorf("resolve db key failed: %w", err)
		}
		db, err = e.findDb(targetKey)
		if err != nil {
			return nil, "", err
		}
	} else {
		db = e.dbs[0].Db
	}

	tableName := e.entity.TableName()
	if e.tableRule != nil {
		var err error
		tableName, err = e.tableRule.ResolveTable(e.entity.TableName(), key)
		if err != nil {
			return nil, "", fmt.Errorf("resolve table failed: %w", err)
		}
	}

	return db, tableName, nil
}

// resolveExecutor creates a single-table executor for the given sharding key.
func (e *shardedExecutor[T]) resolveExecutor(key any) (dbspi.Executor[T], error) {
	db, tableName, err := e.resolve(key)
	if err != nil {
		return nil, err
	}
	return NewExecutorWithTableName(db, e.entity, tableName), nil
}

// resolveFromCtx extracts the sharding key from context and resolves the executor.
func (e *shardedExecutor[T]) resolveFromCtx(ctx context.Context) (dbspi.Executor[T], error) {
	key, ok := dbspi.ShardingKeyFromCtx(ctx)
	if !ok {
		return nil, dbspi.ErrShardingKeyRequired
	}
	return e.resolveExecutor(key)
}

// Shard routes to a specific shard by the given sharding key.
func (e *shardedExecutor[T]) Shard(key any) (dbspi.Executor[T], error) {
	return e.resolveExecutor(key)
}

// ================== CRUD methods (resolve from ctx) ==================

func (e *shardedExecutor[T]) GetById(ctx context.Context, id any) (T, error) {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	return exec.GetById(ctx, id)
}

func (e *shardedExecutor[T]) ExistsById(ctx context.Context, id any) (bool, T, error) {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		var zero T
		return false, zero, err
	}
	return exec.ExistsById(ctx, id)
}

func (e *shardedExecutor[T]) UpdateById(ctx context.Context, id any, updater dbspi.Updater) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.UpdateById(ctx, id, updater)
}

func (e *shardedExecutor[T]) DeleteById(ctx context.Context, id any) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.DeleteById(ctx, id)
}

func (e *shardedExecutor[T]) Find(ctx context.Context, query dbspi.Query, pagenation dbspi.PaginationConfig) ([]T, error) {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return exec.Find(ctx, query, pagenation)
}

func (e *shardedExecutor[T]) Exists(ctx context.Context, query dbspi.Query) (bool, T, error) {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		var zero T
		return false, zero, err
	}
	return exec.Exists(ctx, query)
}

func (e *shardedExecutor[T]) Count(ctx context.Context, query dbspi.Query) (uint64, error) {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return 0, err
	}
	return exec.Count(ctx, query)
}

func (e *shardedExecutor[T]) Create(ctx context.Context, entity T) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.Create(ctx, entity)
}

func (e *shardedExecutor[T]) Save(ctx context.Context, entity T) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.Save(ctx, entity)
}

func (e *shardedExecutor[T]) Update(ctx context.Context, entity T) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.Update(ctx, entity)
}

func (e *shardedExecutor[T]) Delete(ctx context.Context, entity T) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.Delete(ctx, entity)
}

func (e *shardedExecutor[T]) BatchCreate(ctx context.Context, entities []T, batchSize int) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.BatchCreate(ctx, entities, batchSize)
}

func (e *shardedExecutor[T]) BatchSave(ctx context.Context, entities []T) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.BatchSave(ctx, entities)
}

func (e *shardedExecutor[T]) UpdateByQuery(ctx context.Context, query dbspi.Query, updater dbspi.Updater) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.UpdateByQuery(ctx, query, updater)
}

func (e *shardedExecutor[T]) DeleteByQuery(ctx context.Context, query dbspi.Query) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.DeleteByQuery(ctx, query)
}

func (e *shardedExecutor[T]) Upsert(ctx context.Context, entity T, updateColumns []dbspi.Column) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.Upsert(ctx, entity, updateColumns)
}

func (e *shardedExecutor[T]) FirstOrCreate(ctx context.Context, entity T, query dbspi.Query) (T, error) {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	return exec.FirstOrCreate(ctx, entity, query)
}

func (e *shardedExecutor[T]) Raw(ctx context.Context, sql string, args ...any) ([]T, error) {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return exec.Raw(ctx, sql, args...)
}

func (e *shardedExecutor[T]) Exec(ctx context.Context, sql string, args ...any) error {
	exec, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	return exec.Exec(ctx, sql, args...)
}

// ================== Scatter-gather methods ==================

// shardTarget represents a resolved (Db, TableName) pair for scatter-gather.
type shardTarget struct {
	db        dbspi.Db
	tableName string
}

// allShardTargets computes the cross-product of all (Db, TableName) combinations.
// When both DbRule and TableRule are Enumerable, it produces the full cross-product.
// When only one is Enumerable, it pairs with the default for the other dimension.
func (e *shardedExecutor[T]) allShardTargets() ([]shardTarget, error) {
	logicalTable := e.entity.TableName()

	var dbKeys []any
	var tableKeys []any

	if e.dbRule != nil {
		if enum, ok := e.dbRule.(dbspi.Enumerable); ok {
			dbKeys = enum.AllKeys()
		}
	}
	if e.tableRule != nil {
		if enum, ok := e.tableRule.(dbspi.Enumerable); ok {
			tableKeys = enum.AllKeys()
		}
	}

	if len(dbKeys) == 0 && len(tableKeys) == 0 {
		return nil, fmt.Errorf("FindAll/CountAll: neither DbShardingRule nor TableShardingRule implements Enumerable")
	}

	var targets []shardTarget

	if len(dbKeys) > 0 && len(tableKeys) > 0 {
		// Cross-product: every Db × every Table
		for _, dk := range dbKeys {
			db, err := e.findDb(dk)
			if err != nil {
				return nil, err
			}
			for _, tk := range tableKeys {
				tableName, err := e.tableRule.ResolveTable(logicalTable, tk)
				if err != nil {
					return nil, fmt.Errorf("resolve table for key %v failed: %w", tk, err)
				}
				targets = append(targets, shardTarget{db: db, tableName: tableName})
			}
		}
	} else if len(tableKeys) > 0 {
		// Table sharding only: single Db × all tables
		db := e.dbs[0].Db
		for _, tk := range tableKeys {
			tableName, err := e.tableRule.ResolveTable(logicalTable, tk)
			if err != nil {
				return nil, fmt.Errorf("resolve table for key %v failed: %w", tk, err)
			}
			targets = append(targets, shardTarget{db: db, tableName: tableName})
		}
	} else {
		// Db sharding only: all Dbs × single table
		for _, dk := range dbKeys {
			db, err := e.findDb(dk)
			if err != nil {
				return nil, err
			}
			targets = append(targets, shardTarget{db: db, tableName: logicalTable})
		}
	}

	return targets, nil
}

// newErrGroup creates an errgroup with concurrency limit if configured.
func (e *shardedExecutor[T]) newErrGroup(ctx context.Context) (*errgroup.Group, context.Context) {
	g, gCtx := errgroup.WithContext(ctx)
	if e.maxConcurrency > 0 {
		g.SetLimit(e.maxConcurrency)
	}
	return g, gCtx
}

func (e *shardedExecutor[T]) FindAll(ctx context.Context, query dbspi.Query, batchSize int) ([]T, error) {
	targets, err := e.allShardTargets()
	if err != nil {
		return nil, err
	}

	g, gCtx := e.newErrGroup(ctx)
	var mu sync.Mutex
	var results []T

	for _, target := range targets {
		target := target
		g.Go(func() error {
			exec := NewExecutorWithTableName(target.db, e.entity, target.tableName)
			rows, err := e.fetchAllFromShard(gCtx, exec, query, batchSize)
			if err != nil {
				return err
			}
			mu.Lock()
			results = append(results, rows...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// fetchAllFromShard fetches all matching rows from a single shard.
// If batchSize > 0, cursor-based pagination is used to avoid deep pagination issues.
// If batchSize <= 0, all rows are fetched in a single query.
func (e *shardedExecutor[T]) fetchAllFromShard(ctx context.Context, exec dbspi.Executor[T], query dbspi.Query, batchSize int) ([]T, error) {
	if batchSize <= 0 {
		return exec.Find(ctx, query, nil)
	}

	idFieldName := e.getIdFieldName()
	idColumn := NewColumn(idFieldName)

	var allRows []T
	var lastCursor any

	for {
		batchQuery := query
		if lastCursor != nil {
			cursorCond := NewField[any](idFieldName).Gt(&lastCursor)
			if query != nil {
				batchQuery = And(query, cursorCond)
			} else {
				batchQuery = NewQuery(cursorCond)
			}
		}

		pagination := NewPaginationConfig().
			WithLimit(&batchSize).
			AppendOrder(NewOrderConfig(idColumn, false)) // ORDER BY id ASC

		rows, err := exec.Find(ctx, batchQuery, pagination)
		if err != nil {
			return nil, err
		}
		allRows = append(allRows, rows...)
		if len(rows) < batchSize {
			break
		}

		lastRow := rows[len(rows)-1]
		lastCursor = extractFieldValue(lastRow, idFieldName)
	}
	return allRows, nil
}

// getIdFieldName returns the ID field name from the entity.
func (e *shardedExecutor[T]) getIdFieldName() string {
	if ider, ok := any(e.entity).(dbspi.Ider); ok {
		return ider.IdFiledName()
	}
	return "id"
}

// extractFieldValue extracts the value of a column from an entity using reflection.
// It checks GORM column tags first, then falls back to matching Go field names.
func extractFieldValue(entity any, columnName string) any {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Check gorm column tag
		if tag := field.Tag.Get("gorm"); tag != "" {
			for _, part := range strings.Split(tag, ";") {
				kv := strings.SplitN(part, ":", 2)
				if len(kv) == 2 && kv[0] == "column" && kv[1] == columnName {
					return val.Field(i).Interface()
				}
			}
			if strings.EqualFold(tag, "primaryKey") || strings.Contains(tag, "primaryKey") {
				if strings.EqualFold(field.Name, columnName) || strings.EqualFold(toSnakeCase(field.Name), columnName) {
					return val.Field(i).Interface()
				}
			}
		}

		// Fallback: match Go field name (case-insensitive) or snake_case
		if strings.EqualFold(field.Name, columnName) || strings.EqualFold(toSnakeCase(field.Name), columnName) {
			return val.Field(i).Interface()
		}
	}
	return nil
}

func toSnakeCase(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+'a'-'A'))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

func (e *shardedExecutor[T]) CountAll(ctx context.Context, query dbspi.Query) (uint64, error) {
	targets, err := e.allShardTargets()
	if err != nil {
		return 0, err
	}

	g, gCtx := e.newErrGroup(ctx)
	var mu sync.Mutex
	var total uint64

	for _, target := range targets {
		target := target
		g.Go(func() error {
			exec := NewExecutorWithTableName(target.db, e.entity, target.tableName)
			count, err := exec.Count(gCtx, query)
			if err != nil {
				return err
			}
			mu.Lock()
			total += count
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return 0, err
	}
	return total, nil
}
