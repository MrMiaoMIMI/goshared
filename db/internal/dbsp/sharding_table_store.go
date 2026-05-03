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

// ShardedTableStoreConfig is the internal config for creating a sharded table store.
type ShardedTableStoreConfig struct {
	Dbs               []DatabaseTarget
	DbRule            DatabaseShardingRule
	TableShardingRule TableShardingRule
	MaxConcurrency    int
	CommonFields      CommonFieldAutoFillOptions
}

// shardingKeyResolver auto-extracts sharding key column values from CRUD parameters.
// Built once at table store construction time using reflection on entity type T.
type shardingKeyResolver struct {
	requiredCols []string       // union of db rule + table rule required columns
	fieldMap     map[string]int // gorm column name -> struct field index
	idColumnName string         // from IdFieldNameProvider or DefaultIdFieldName
}

// buildShardingKeyResolver creates a resolver from the sharding rules and entity type.
// Returns nil if any rule doesn't implement ShardingKeyColumnsProvider.
func buildShardingKeyResolver(entityType reflect.Type, idColumnName string, dbRule DatabaseShardingRule, tableRule TableShardingRule) *shardingKeyResolver {
	seen := make(map[string]bool)
	var requiredCols []string

	if dbRule != nil {
		provider, ok := dbRule.(ShardingKeyColumnsProvider)
		if !ok {
			return nil
		}
		for _, col := range provider.RequiredColumns() {
			if !seen[col] {
				seen[col] = true
				requiredCols = append(requiredCols, col)
			}
		}
	}
	if tableRule != nil {
		provider, ok := tableRule.(ShardingKeyColumnsProvider)
		if !ok {
			return nil
		}
		for _, col := range provider.RequiredColumns() {
			if !seen[col] {
				seen[col] = true
				requiredCols = append(requiredCols, col)
			}
		}
	}

	if len(requiredCols) == 0 {
		return nil
	}

	fieldMap := buildColumnFieldMap(entityType)

	return &shardingKeyResolver{
		requiredCols: requiredCols,
		fieldMap:     fieldMap,
		idColumnName: idColumnName,
	}
}

// buildColumnFieldMap builds a mapping from gorm column names to struct field indices.
func buildColumnFieldMap(t reflect.Type) map[string]int {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	m := make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		colName := ""
		if tag := field.Tag.Get("gorm"); tag != "" {
			for _, part := range strings.Split(tag, ";") {
				kv := strings.SplitN(part, ":", 2)
				if len(kv) == 2 && kv[0] == "column" {
					colName = kv[1]
					break
				}
			}
		}
		if colName == "" {
			colName = toSnakeCase(field.Name)
		}
		m[colName] = i
	}
	return m
}

// fromEntity extracts sharding-relevant column values from an entity struct.
func (r *shardingKeyResolver) fromEntity(entity any) map[string]any {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	result := make(map[string]any)
	for _, col := range r.requiredCols {
		if idx, ok := r.fieldMap[col]; ok {
			result[col] = val.Field(idx).Interface()
		}
	}
	return result
}

// fromId constructs a column map from an ID parameter.
func (r *shardingKeyResolver) fromId(id any) map[string]any {
	return map[string]any{r.idColumnName: id}
}

// fromQuery extracts column-value pairs from Query Eq/IN conditions.
// Returns a multi-value map where each column may have multiple values,
// and a set of columns that have range conditions (for diagnostic errors).
func (r *shardingKeyResolver) fromQuery(query dbspi.Query) (map[string][]any, map[string]bool) {
	if query == nil {
		return make(map[string][]any), make(map[string]bool)
	}
	return ExtractColumnsFromQuery(query)
}

// buildShardingKey validates that all required columns are present and builds a ShardingKey.
// rangeCols provides hints about columns that appeared in range conditions (Gt/Lt/Gte/Lte),
// enabling a more actionable error message when those columns are missing.
func (r *shardingKeyResolver) buildShardingKey(columns map[string]any, rangeCols map[string]bool) (*dbspi.ShardingKey, error) {
	var missing []string
	var rangeHints []string
	for _, col := range r.requiredCols {
		if _, ok := columns[col]; !ok {
			missing = append(missing, col)
			if rangeCols[col] {
				rangeHints = append(rangeHints, col)
			}
		}
	}
	if len(missing) > 0 {
		if len(rangeHints) > 0 {
			return nil, fmt.Errorf(
				"sharding columns %v have range conditions (Gt/Lt/Between) which cannot determine a single shard; "+
					"range conditions may cause cross-shard operations. "+
					"Use Eq/In for sharding columns, set WithShardingKey(ctx, key), or use FindAll/CountAll for cross-shard queries",
				rangeHints)
		}
		available := make([]string, 0, len(columns))
		for k := range columns {
			available = append(available, k)
		}
		return nil, fmt.Errorf(
			"sharding key missing required columns: %v (available: %v). "+
				"Provide via WithShardingKey(ctx, key) or ensure values exist in entity/query parameters",
			missing, available)
	}
	sk := dbspi.NewShardingKey()
	for _, col := range r.requiredCols {
		sk.SetValue(col, columns[col])
	}
	return sk, nil
}

// mergeIntoMultiValues merges single-value maps (from entity/ctx) into a multi-value map
// (from query extraction). All values are collected for later same-target validation.
func mergeIntoMultiValues(singleCols map[string]any, multiCols map[string][]any) map[string][]any {
	result := make(map[string][]any)
	for col, val := range singleCols {
		result[col] = append(result[col], val)
	}
	for col, vals := range multiCols {
		result[col] = append(result[col], vals...)
	}
	return result
}

// mergeSingleIntoMulti appends all single-value entries into an existing multi-value map.
func mergeSingleIntoMulti(base map[string][]any, singleCols map[string]any) {
	for col, val := range singleCols {
		base[col] = append(base[col], val)
	}
}

// deduplicateValues returns the distinct values from the input slice,
// preserving order of first occurrence.
func deduplicateValues(values []any) []any {
	var unique []any
	for _, v := range values {
		found := false
		for _, u := range unique {
			if reflect.DeepEqual(u, v) {
				found = true
				break
			}
		}
		if !found {
			unique = append(unique, v)
		}
	}
	return unique
}

// resolveTarget resolves a ShardingKey to the target routing coordinates (db key + table name)
// without looking up the actual Db instance. Used for same-target validation.
func (e *shardedTableStore[T]) resolveTarget(sk *dbspi.ShardingKey) (dbKey string, tableName string, err error) {
	if e.dbRule != nil {
		dbKey, err = e.dbRule.ResolveDatabaseTargetKey(sk)
		if err != nil {
			return "", "", fmt.Errorf("resolve db key failed: %w", err)
		}
	}
	tableName = e.entity.TableName()
	if e.tableRule != nil {
		tableName, err = e.tableRule.ResolveTable(e.entity.TableName(), sk)
		if err != nil {
			return "", "", fmt.Errorf("resolve table failed: %w", err)
		}
	}
	return dbKey, tableName, nil
}

// reduceColumns deduplicates multi-value columns and validates that all distinct values
// for each required sharding column route to the same target (db + table).
// Returns a single-value map suitable for building a ShardingKey.
func (e *shardedTableStore[T]) reduceColumns(multiCols map[string][]any) (map[string]any, error) {
	result := make(map[string]any)

	type multiValEntry struct {
		name   string
		values []any
	}
	var multiValCols []multiValEntry

	for col, values := range multiCols {
		unique := deduplicateValues(values)
		if len(unique) == 0 {
			continue
		}
		result[col] = unique[0]
		if len(unique) > 1 && e.isRequiredColumn(col) {
			multiValCols = append(multiValCols, multiValEntry{name: col, values: unique})
		}
	}

	if len(multiValCols) == 0 {
		return result, nil
	}

	// Check all required columns are present before attempting resolution
	if e.keyResolver != nil {
		for _, reqCol := range e.keyResolver.requiredCols {
			if _, ok := result[reqCol]; !ok {
				return result, nil // missing column; buildShardingKey will report it
			}
		}
	}

	// Build reference ShardingKey with first values and resolve the reference target
	refSk := dbspi.NewShardingKey()
	for _, col := range e.keyResolver.requiredCols {
		refSk.SetValue(col, result[col])
	}
	refDbKey, refTable, err := e.resolveTarget(refSk)
	if err != nil {
		return nil, err
	}

	// Validate each alternative value routes to the same target
	for _, mvc := range multiValCols {
		for _, altVal := range mvc.values[1:] {
			altSk := dbspi.NewShardingKey()
			for _, reqCol := range e.keyResolver.requiredCols {
				if reqCol == mvc.name {
					altSk.SetValue(reqCol, altVal)
				} else {
					altSk.SetValue(reqCol, result[reqCol])
				}
			}
			altDbKey, altTable, err := e.resolveTarget(altSk)
			if err != nil {
				return nil, fmt.Errorf("validate sharding column %q value %v: %w", mvc.name, altVal, err)
			}
			if altDbKey != refDbKey || altTable != refTable {
				return nil, fmt.Errorf(
					"cross-shard query not allowed: column %q values %v route to different targets "+
						"(db=%q table=%q vs db=%q table=%q)",
					mvc.name, mvc.values, refDbKey, refTable, altDbKey, altTable)
			}
		}
	}

	return result, nil
}

// isRequiredColumn checks if a column is required by the sharding rules.
func (e *shardedTableStore[T]) isRequiredColumn(col string) bool {
	if e.keyResolver == nil {
		return false
	}
	for _, c := range e.keyResolver.requiredCols {
		if c == col {
			return true
		}
	}
	return false
}

type shardedTableStore[T dbspi.Entity] struct {
	entity         T
	dbs            []DatabaseTarget
	dbRule         DatabaseShardingRule
	tableRule      TableShardingRule
	maxConcurrency int
	keyResolver    *shardingKeyResolver
	commonFields   CommonFieldAutoFillOptions
}

func NewShardedTableStore[T dbspi.Entity](entity T, cfg ShardedTableStoreConfig) *shardedTableStore[T] {
	if len(cfg.Dbs) == 0 {
		panic("sharded table store requires at least one DatabaseTarget via WithDbs")
	}
	if cfg.TableShardingRule == nil && cfg.DbRule == nil {
		panic("sharded table store requires at least one of WithTableRule or WithDbRule")
	}

	idColumnName := dbspi.DefaultIdFieldName
	if namer, ok := any(entity).(dbspi.IdFieldNameProvider); ok {
		idColumnName = namer.IdFieldName()
	}

	entityType := reflect.TypeOf(entity)
	resolver := buildShardingKeyResolver(entityType, idColumnName, cfg.DbRule, cfg.TableShardingRule)

	return &shardedTableStore[T]{
		entity:         entity,
		dbs:            cfg.Dbs,
		dbRule:         cfg.DbRule,
		tableRule:      cfg.TableShardingRule,
		maxConcurrency: cfg.MaxConcurrency,
		keyResolver:    resolver,
		commonFields:   cfg.CommonFields,
	}
}

func toSoftDeleteTableStore[T dbspi.Entity](store dbspi.TableStore[T]) (dbspi.SoftDeleteTableStore[T], error) {
	softDeleteStore, ok := store.(dbspi.SoftDeleteTableStore[T])
	if !ok {
		return nil, fmt.Errorf("resolved table store does not implement SoftDeleteTableStore")
	}
	return softDeleteStore, nil
}

func toSQLTableStore[T dbspi.Entity](store dbspi.TableStore[T]) (dbspi.SQLTableStore[T], error) {
	sqlStore, ok := store.(dbspi.SQLTableStore[T])
	if !ok {
		return nil, fmt.Errorf("resolved table store does not implement SQLTableStore")
	}
	return sqlStore, nil
}

// findDb looks up the Db by matching the target key string.
func (e *shardedTableStore[T]) findDb(targetKey string) (dbSession, error) {
	for _, dt := range e.dbs {
		if dt.Key == targetKey {
			return dt.Db, nil
		}
	}
	return nil, fmt.Errorf("no DatabaseTarget found for key: %s", targetKey)
}

// resolve determines the target Db and physical table name for the given ShardingKey.
func (e *shardedTableStore[T]) resolve(sk *dbspi.ShardingKey) (dbSession, string, error) {
	var db dbSession

	if e.dbRule != nil {
		targetKey, err := e.dbRule.ResolveDatabaseTargetKey(sk)
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
		tableName, err = e.tableRule.ResolveTable(e.entity.TableName(), sk)
		if err != nil {
			return nil, "", fmt.Errorf("resolve table failed: %w", err)
		}
	}

	return db, tableName, nil
}

// resolveStore creates a single-table store for the given ShardingKey.
func (e *shardedTableStore[T]) resolveStore(sk *dbspi.ShardingKey) (dbspi.TableStore[T], error) {
	db, tableName, err := e.resolve(sk)
	if err != nil {
		return nil, err
	}
	return NewTableStoreWithTableNameAndCommonFields(db, e.entity, tableName, e.commonFields), nil
}

// resolveFromCtx extracts the ShardingKey from context and resolves the table store.
// This is the fallback for methods where auto-extraction is not possible (Raw, Exec).
func (e *shardedTableStore[T]) resolveFromCtx(ctx context.Context) (dbspi.TableStore[T], error) {
	sk, ok := dbspi.ShardingKeyFromContext(ctx)
	if !ok {
		return nil, dbspi.ErrShardingKeyRequired
	}
	return e.resolveStore(sk)
}

// resolveForEntity resolves by aggregating ctx key + entity struct fields,
// then validating all values route to the same target.
func (e *shardedTableStore[T]) resolveForEntity(ctx context.Context, entity T) (dbspi.TableStore[T], error) {
	ctxSk, hasCtx := dbspi.ShardingKeyFromContext(ctx)
	if hasCtx && e.keyResolver == nil {
		return e.resolveStore(ctxSk)
	}
	if e.keyResolver != nil {
		entityCols := e.keyResolver.fromEntity(entity)
		multiCols := mergeIntoMultiValues(entityCols, nil)
		if hasCtx {
			mergeSingleIntoMulti(multiCols, ctxSk.Fields())
		}
		columns, err := e.reduceColumns(multiCols)
		if err != nil {
			return nil, err
		}
		sk, err := e.keyResolver.buildShardingKey(columns, nil)
		if err != nil {
			return nil, err
		}
		return e.resolveStore(sk)
	}
	return nil, dbspi.ErrShardingKeyRequired
}

// resolveForId resolves by aggregating ctx key + id parameter,
// then validating all values route to the same target.
func (e *shardedTableStore[T]) resolveForId(ctx context.Context, id any) (dbspi.TableStore[T], error) {
	ctxSk, hasCtx := dbspi.ShardingKeyFromContext(ctx)
	if hasCtx && e.keyResolver == nil {
		return e.resolveStore(ctxSk)
	}
	if e.keyResolver != nil {
		idCols := e.keyResolver.fromId(id)
		multiCols := mergeIntoMultiValues(idCols, nil)
		if hasCtx {
			mergeSingleIntoMulti(multiCols, ctxSk.Fields())
		}
		columns, err := e.reduceColumns(multiCols)
		if err != nil {
			return nil, err
		}
		sk, err := e.keyResolver.buildShardingKey(columns, nil)
		if err != nil {
			return nil, err
		}
		return e.resolveStore(sk)
	}
	return nil, dbspi.ErrShardingKeyRequired
}

// resolveForQuery resolves by aggregating ctx key + query conditions,
// then validating all values route to the same target.
func (e *shardedTableStore[T]) resolveForQuery(ctx context.Context, query dbspi.Query) (dbspi.TableStore[T], error) {
	ctxSk, hasCtx := dbspi.ShardingKeyFromContext(ctx)
	if hasCtx && e.keyResolver == nil {
		return e.resolveStore(ctxSk)
	}
	if e.keyResolver != nil {
		multiCols, rangeCols := e.keyResolver.fromQuery(query)
		if hasCtx {
			mergeSingleIntoMulti(multiCols, ctxSk.Fields())
		}
		columns, err := e.reduceColumns(multiCols)
		if err != nil {
			return nil, err
		}
		sk, err := e.keyResolver.buildShardingKey(columns, rangeCols)
		if err != nil {
			return nil, err
		}
		return e.resolveStore(sk)
	}
	return nil, dbspi.ErrShardingKeyRequired
}

// resolveForEntityAndQuery resolves by aggregating ctx key + entity + query,
// then validating all values route to the same target.
func (e *shardedTableStore[T]) resolveForEntityAndQuery(ctx context.Context, entity T, query dbspi.Query) (dbspi.TableStore[T], error) {
	ctxSk, hasCtx := dbspi.ShardingKeyFromContext(ctx)
	if hasCtx && e.keyResolver == nil {
		return e.resolveStore(ctxSk)
	}
	if e.keyResolver != nil {
		entityCols := e.keyResolver.fromEntity(entity)
		queryCols, rangeCols := e.keyResolver.fromQuery(query)
		merged := mergeIntoMultiValues(entityCols, queryCols)
		if hasCtx {
			mergeSingleIntoMulti(merged, ctxSk.Fields())
		}
		columns, err := e.reduceColumns(merged)
		if err != nil {
			return nil, err
		}
		sk, err := e.keyResolver.buildShardingKey(columns, rangeCols)
		if err != nil {
			return nil, err
		}
		return e.resolveStore(sk)
	}
	return nil, dbspi.ErrShardingKeyRequired
}

// Shard explicitly binds subsequent operations to one physical shard.
func (e *shardedTableStore[T]) Shard(key *dbspi.ShardingKey) (dbspi.TableStore[T], error) {
	if key == nil {
		return nil, dbspi.ErrShardingKeyRequired
	}
	return e.resolveStore(key)
}

// ================== CRUD methods ==================

// -- ID-based methods (resolve from ctx > id) --

func (e *shardedTableStore[T]) GetById(ctx context.Context, id any) (T, error) {
	store, err := e.resolveForId(ctx, id)
	if err != nil {
		var zero T
		return zero, err
	}
	return store.GetById(ctx, id)
}

func (e *shardedTableStore[T]) ExistsById(ctx context.Context, id any) (bool, T, error) {
	store, err := e.resolveForId(ctx, id)
	if err != nil {
		var zero T
		return false, zero, err
	}
	return store.ExistsById(ctx, id)
}

func (e *shardedTableStore[T]) UpdateById(ctx context.Context, id any, updater dbspi.Updater) error {
	store, err := e.resolveForId(ctx, id)
	if err != nil {
		return err
	}
	return store.UpdateById(ctx, id, updater)
}

func (e *shardedTableStore[T]) DeleteById(ctx context.Context, id any) error {
	store, err := e.resolveForId(ctx, id)
	if err != nil {
		return err
	}
	return store.DeleteById(ctx, id)
}

func (e *shardedTableStore[T]) SoftDeleteById(ctx context.Context, id any) error {
	store, err := e.resolveForId(ctx, id)
	if err != nil {
		return err
	}
	softDeleteStore, err := toSoftDeleteTableStore(store)
	if err != nil {
		return err
	}
	return softDeleteStore.SoftDeleteById(ctx, id)
}

func (e *shardedTableStore[T]) RestoreById(ctx context.Context, id any) error {
	store, err := e.resolveForId(ctx, id)
	if err != nil {
		return err
	}
	softDeleteStore, err := toSoftDeleteTableStore(store)
	if err != nil {
		return err
	}
	return softDeleteStore.RestoreById(ctx, id)
}

func (e *shardedTableStore[T]) ExistsByIdNotDeleted(ctx context.Context, id any) (bool, T, error) {
	store, err := e.resolveForId(ctx, id)
	if err != nil {
		var zero T
		return false, zero, err
	}
	softDeleteStore, err := toSoftDeleteTableStore(store)
	if err != nil {
		var zero T
		return false, zero, err
	}
	return softDeleteStore.ExistsByIdNotDeleted(ctx, id)
}

// -- Query-based methods (resolve from ctx > query) --

func (e *shardedTableStore[T]) Find(ctx context.Context, query dbspi.Query, pagination dbspi.Pagination) ([]T, error) {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	return store.Find(ctx, query, pagination)
}

func (e *shardedTableStore[T]) Exists(ctx context.Context, query dbspi.Query) (bool, T, error) {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		var zero T
		return false, zero, err
	}
	return store.Exists(ctx, query)
}

func (e *shardedTableStore[T]) Count(ctx context.Context, query dbspi.Query) (uint64, error) {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		return 0, err
	}
	return store.Count(ctx, query)
}

func (e *shardedTableStore[T]) UpdateByQuery(ctx context.Context, query dbspi.Query, updater dbspi.Updater) error {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		return err
	}
	return store.UpdateByQuery(ctx, query, updater)
}

func (e *shardedTableStore[T]) DeleteByQuery(ctx context.Context, query dbspi.Query) error {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		return err
	}
	return store.DeleteByQuery(ctx, query)
}

func (e *shardedTableStore[T]) SoftDeleteByQuery(ctx context.Context, query dbspi.Query) error {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		return err
	}
	softDeleteStore, err := toSoftDeleteTableStore(store)
	if err != nil {
		return err
	}
	return softDeleteStore.SoftDeleteByQuery(ctx, query)
}

func (e *shardedTableStore[T]) RestoreByQuery(ctx context.Context, query dbspi.Query) error {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		return err
	}
	softDeleteStore, err := toSoftDeleteTableStore(store)
	if err != nil {
		return err
	}
	return softDeleteStore.RestoreByQuery(ctx, query)
}

func (e *shardedTableStore[T]) FindNotDeleted(ctx context.Context, query dbspi.Query, pagination dbspi.Pagination) ([]T, error) {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	softDeleteStore, err := toSoftDeleteTableStore(store)
	if err != nil {
		return nil, err
	}
	return softDeleteStore.FindNotDeleted(ctx, query, pagination)
}

func (e *shardedTableStore[T]) CountNotDeleted(ctx context.Context, query dbspi.Query) (uint64, error) {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		return 0, err
	}
	softDeleteStore, err := toSoftDeleteTableStore(store)
	if err != nil {
		return 0, err
	}
	return softDeleteStore.CountNotDeleted(ctx, query)
}

func (e *shardedTableStore[T]) ExistsNotDeleted(ctx context.Context, query dbspi.Query) (bool, T, error) {
	store, err := e.resolveForQuery(ctx, query)
	if err != nil {
		var zero T
		return false, zero, err
	}
	softDeleteStore, err := toSoftDeleteTableStore(store)
	if err != nil {
		var zero T
		return false, zero, err
	}
	return softDeleteStore.ExistsNotDeleted(ctx, query)
}

// -- Entity-based methods (resolve from ctx > entity) --

func (e *shardedTableStore[T]) Create(ctx context.Context, entity T) error {
	store, err := e.resolveForEntity(ctx, entity)
	if err != nil {
		return err
	}
	return store.Create(ctx, entity)
}

func (e *shardedTableStore[T]) Save(ctx context.Context, entity T) error {
	store, err := e.resolveForEntity(ctx, entity)
	if err != nil {
		return err
	}
	return store.Save(ctx, entity)
}

func (e *shardedTableStore[T]) Update(ctx context.Context, entity T) error {
	store, err := e.resolveForEntity(ctx, entity)
	if err != nil {
		return err
	}
	return store.Update(ctx, entity)
}

func (e *shardedTableStore[T]) Delete(ctx context.Context, entity T) error {
	store, err := e.resolveForEntity(ctx, entity)
	if err != nil {
		return err
	}
	return store.Delete(ctx, entity)
}

func (e *shardedTableStore[T]) BatchCreate(ctx context.Context, entities []T, batchSize int) error {
	if len(entities) == 0 {
		return nil
	}
	store, err := e.resolveForEntity(ctx, entities[0])
	if err != nil {
		return err
	}
	return store.BatchCreate(ctx, entities, batchSize)
}

func (e *shardedTableStore[T]) BatchSave(ctx context.Context, entities []T) error {
	if len(entities) == 0 {
		return nil
	}
	store, err := e.resolveForEntity(ctx, entities[0])
	if err != nil {
		return err
	}
	return store.BatchSave(ctx, entities)
}

// -- Multi-source method (resolve from ctx > entity + query) --

func (e *shardedTableStore[T]) FirstOrCreate(ctx context.Context, entity T, query dbspi.Query) (T, error) {
	store, err := e.resolveForEntityAndQuery(ctx, entity, query)
	if err != nil {
		var zero T
		return zero, err
	}
	return store.FirstOrCreate(ctx, entity, query)
}

// -- Raw SQL methods (resolve from ctx only, no auto-extraction) --

func (e *shardedTableStore[T]) Raw(ctx context.Context, sql string, args ...any) ([]T, error) {
	store, err := e.resolveFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	sqlStore, err := toSQLTableStore(store)
	if err != nil {
		return nil, err
	}
	return sqlStore.Raw(ctx, sql, args...)
}

func (e *shardedTableStore[T]) Exec(ctx context.Context, sql string, args ...any) error {
	store, err := e.resolveFromCtx(ctx)
	if err != nil {
		return err
	}
	sqlStore, err := toSQLTableStore(store)
	if err != nil {
		return err
	}
	return sqlStore.Exec(ctx, sql, args...)
}

// ================== Scatter-gather methods ==================

// shardTarget represents a resolved (Db, TableName) pair for scatter-gather.
type shardTarget struct {
	db        dbSession
	tableName string
}

// allShardTargets computes all (Db, TableName) combinations for scatter-gather.
func (e *shardedTableStore[T]) allShardTargets() ([]shardTarget, error) {
	logicalTable := e.entity.TableName()
	var targets []shardTarget

	tableCount := 0
	if e.tableRule != nil {
		if counter, ok := e.tableRule.(TableShardCounter); ok {
			tableCount = counter.ShardCount()
		}
	}

	for _, dt := range e.dbs {
		if tableCount > 0 {
			enumerator, ok := e.tableRule.(TableShardEnumerator)
			if !ok {
				return nil, fmt.Errorf("table rule implements TableShardCounter but not TableShardEnumerator")
			}
			for i := 0; i < tableCount; i++ {
				tableName, err := enumerator.ShardName(logicalTable, i)
				if err != nil {
					return nil, fmt.Errorf("enumerate table shard %d failed: %w", i, err)
				}
				targets = append(targets, shardTarget{db: dt.Db, tableName: tableName})
			}
		} else {
			targets = append(targets, shardTarget{db: dt.Db, tableName: logicalTable})
		}
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("FindAll/CountAll: no shard targets available")
	}

	return targets, nil
}

// newErrGroup creates an errgroup with concurrency limit if configured.
func (e *shardedTableStore[T]) newErrGroup(ctx context.Context) (*errgroup.Group, context.Context) {
	g, gCtx := errgroup.WithContext(ctx)
	if e.maxConcurrency > 0 {
		g.SetLimit(e.maxConcurrency)
	}
	return g, gCtx
}

func (e *shardedTableStore[T]) FindAll(ctx context.Context, query dbspi.Query, batchSize int) ([]T, error) {
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
			store := NewTableStoreWithTableNameAndCommonFields(target.db, e.entity, target.tableName, e.commonFields)
			rows, err := e.fetchAllFromShard(gCtx, store, query, batchSize)
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
func (e *shardedTableStore[T]) fetchAllFromShard(ctx context.Context, store dbspi.TableStore[T], query dbspi.Query, batchSize int) ([]T, error) {
	if batchSize <= 0 {
		return store.Find(ctx, query, nil)
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

		pagination := NewPagination().
			WithLimit(&batchSize).
			AppendOrder(newOrder(idColumn, false))

		rows, err := store.Find(ctx, batchQuery, pagination)
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
func (e *shardedTableStore[T]) getIdFieldName() string {
	if namer, ok := any(e.entity).(dbspi.IdFieldNameProvider); ok {
		return namer.IdFieldName()
	}
	return dbspi.DefaultIdFieldName
}

// extractFieldValue extracts the value of a column from an entity using reflection.
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

func (e *shardedTableStore[T]) CountAll(ctx context.Context, query dbspi.Query) (uint64, error) {
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
			store := NewTableStoreWithTableNameAndCommonFields(target.db, e.entity, target.tableName, e.commonFields)
			count, err := store.Count(gCtx, query)
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
