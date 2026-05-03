package dbsp

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ================== Check implementions for all spi ==================

type _tableForCheck struct{}

func (t _tableForCheck) TableName() string {
	return "table_for_check"
}

func (t _tableForCheck) IdFieldName() string {
	return dbspi.DefaultIdFieldName
}

var (
	// Check external interfaces
	_ dbspi.Condition                     = (*GormCondition)(nil)
	_ dbspi.Column                        = (*GormColumn)(nil)
	_ dbspi.Field[any]                    = (*GormField[any])(nil)
	_ dbspi.Query                         = (*GormQuery)(nil)
	_ dbspi.Updater                       = (*GormUpdater)(nil)
	_ dbspi.TableStore[_tableForCheck]    = new(GormTableStore[_tableForCheck])
	_ dbspi.SQLTableStore[_tableForCheck] = new(GormTableStore[_tableForCheck])

	// Check internal interfaces
	_ gormExpression = (*GormCondition)(nil)
	_ gormExpression = (*GormQuery)(nil)
)

// Internal interfaces
type gormExpression interface {
	ToGormExpression() clause.Expression
}

// ================== Condition Implementation ==================

// GormCondition implements dbspi.Condition
type GormCondition struct {
	expr clause.Expression
}

// newCondition creates a new GormCondition from clause.Expression
func newCondition(expr clause.Expression) dbspi.Condition {
	return &GormCondition{expr: expr}
}

func (c *GormCondition) ToGormExpression() clause.Expression {
	return c.expr
}

// ================== Column Implementation ==================

// GormColumn implements dbspi.Column
type GormColumn struct {
	name  string
	table string
	alias string
}

// NewColumn creates a new GormColumn
func NewColumn(name string) *GormColumn {
	return &GormColumn{
		name: name,
	}
}

// Name implements dbspi.Column
func (c *GormColumn) Name() string {
	return c.name
}

// Table implements dbspi.Column
func (c *GormColumn) Table() string {
	return c.table
}

// Alias implements dbspi.Column
func (c *GormColumn) Alias() string {
	return c.alias
}

// WithTable implements dbspi.Column
func (c *GormColumn) WithTable(table string) dbspi.Column {
	return &GormColumn{
		name:  c.name,
		table: table,
	}
}

// WithAlias implements dbspi.Column
func (c *GormColumn) WithAlias(alias string) dbspi.Column {
	return &GormColumn{
		name:  c.name,
		alias: alias,
	}
}

// ================== Field Implementation ==================

// GormField implements dbspi.Field[T]
type GormField[T any] struct {
	dbspi.Column
}

// NewField creates a new GormField
func NewField[T any](name string) *GormField[T] {
	return &GormField[T]{
		Column: NewColumn(name),
	}
}

// columnExpr returns the column expression for queries
func (f *GormField[T]) columnExpr() clause.Column {
	return clause.Column{
		Name: f.Column.Name(),
	}
}

// IsNull implements dbspi.Field
func (f *GormField[T]) IsNull() dbspi.Condition {
	return newCondition(clause.Eq{
		Column: f.columnExpr(),
		Value:  nil,
	})
}

// IsNotNull implements dbspi.Field
func (f *GormField[T]) IsNotNull() dbspi.Condition {
	return newCondition(clause.Neq{
		Column: f.columnExpr(),
		Value:  nil,
	})
}

// Eq implements dbspi.Field
func (f *GormField[T]) Eq(v *T) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Eq{
		Column: f.columnExpr(),
		Value:  *v,
	})
}

// NotEq implements dbspi.Field
func (f *GormField[T]) NotEq(v *T) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Neq{
		Column: f.columnExpr(),
		Value:  *v,
	})
}

// In implements dbspi.Field
func (f *GormField[T]) In(v []T) dbspi.Condition {
	if len(v) == 0 {
		return nil
	}
	values := make([]any, len(v))
	for i, val := range v {
		values[i] = val
	}
	return newCondition(clause.IN{
		Column: f.columnExpr(),
		Values: values,
	})
}

// NotIn implements dbspi.Field
func (f *GormField[T]) NotIn(v []T) dbspi.Condition {
	if len(v) == 0 {
		return nil
	}
	values := make([]any, len(v))
	for i, val := range v {
		values[i] = val
	}
	return newCondition(clause.Not(clause.IN{
		Column: f.columnExpr(),
		Values: values,
	}))
}

// Gt implements dbspi.Field
func (f *GormField[T]) Gt(v *T) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Gt{
		Column: f.columnExpr(),
		Value:  *v,
	})
}

// GtEq implements dbspi.Field
func (f *GormField[T]) GtEq(v *T) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Gte{
		Column: f.columnExpr(),
		Value:  *v,
	})
}

// Lt implements dbspi.Field
func (f *GormField[T]) Lt(v *T) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Lt{
		Column: f.columnExpr(),
		Value:  *v,
	})
}

// LtEq implements dbspi.Field
func (f *GormField[T]) LtEq(v *T) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Lte{
		Column: f.columnExpr(),
		Value:  *v,
	})
}

// Between implements dbspi.Field
func (f *GormField[T]) Between(min, max *T) dbspi.Condition {
	if min == nil || max == nil {
		return nil
	}
	return newCondition(clause.And(
		clause.Gte{Column: f.columnExpr(), Value: *min},
		clause.Lte{Column: f.columnExpr(), Value: *max},
	))
}

// Like implements dbspi.Field
func (f *GormField[T]) Like(v *string) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Like{
		Column: f.columnExpr(),
		Value:  *v,
	})
}

// NotLike implements dbspi.Field
func (f *GormField[T]) NotLike(v *string) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Not(clause.Like{
		Column: f.columnExpr(),
		Value:  *v,
	}))
}

// StartsWith implements dbspi.Field
func (f *GormField[T]) StartsWith(v *string) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Like{
		Column: f.columnExpr(),
		Value:  *v + "%",
	})
}

// EndsWith implements dbspi.Field
func (f *GormField[T]) EndsWith(v *string) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Like{
		Column: f.columnExpr(),
		Value:  "%" + *v,
	})
}

// Contains implements dbspi.Field
func (f *GormField[T]) Contains(v *string) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Like{
		Column: f.columnExpr(),
		Value:  "%" + *v + "%",
	})
}

// NotContains implements dbspi.Field
func (f *GormField[T]) NotContains(v *string) dbspi.Condition {
	if v == nil {
		return nil
	}
	return newCondition(clause.Not(clause.Like{
		Column: f.columnExpr(),
		Value:  "%" + *v + "%",
	}))
}

// ================== Query Implementation ==================

type queryKeyword string

const (
	keywordAnd queryKeyword = "AND"
	keywordOr  queryKeyword = "OR"
	keywordNot queryKeyword = "NOT"
)

// GormQuery implements dbspi.Query
type GormQuery struct {
	keyword    queryKeyword
	conditions []dbspi.Condition
}

// NewQuery creates a new GormQuery
func newQuery(keyword queryKeyword, conditions ...dbspi.Condition) dbspi.Query {
	return &GormQuery{keyword: keyword, conditions: conditions}
}

// NewQuery creates a new GormQuery with AND keyword
func NewQuery(conditions ...dbspi.Condition) dbspi.Query {
	return newQuery(keywordAnd, conditions...)
}

type columnSelectionQuery interface {
	dbspi.Query
	Columns() []dbspi.Column
}

// gormColumnSelectionQuery implements columnSelectionQuery.
type gormColumnSelectionQuery struct {
	*GormQuery
	columns []dbspi.Column
}

func (q *gormColumnSelectionQuery) Columns() []dbspi.Column {
	return q.columns
}

// Select wraps a query with specific column selection.
func Select(columns []dbspi.Column, conditions ...dbspi.Condition) dbspi.Query {
	return &gormColumnSelectionQuery{
		GormQuery: &GormQuery{keyword: keywordAnd, conditions: conditions},
		columns:   columns,
	}
}

// And creates a new GormQuery with AND keyword
func And(conditions ...dbspi.Condition) dbspi.Query {
	return newQuery(keywordAnd, conditions...)
}

// Or creates a new GormQuery with OR keyword
func Or(conditions ...dbspi.Condition) dbspi.Query {
	return newQuery(keywordOr, conditions...)
}

// Not creates a new GormQuery with NOT keyword
func Not(condition dbspi.Condition) dbspi.Query {
	return newQuery(keywordNot, condition)
}

func (q *GormQuery) ToGormExpression() clause.Expression {
	gormExpressions := make([]clause.Expression, 0, len(q.conditions))
	for _, cond := range q.conditions {
		if cond == nil {
			continue
		}
		if gc, ok := cond.(gormExpression); ok {
			if gc.ToGormExpression() == nil {
				continue
			}
			gormExpressions = append(gormExpressions, gc.ToGormExpression())
		} else {
			// Unknown condition implementations are ignored so custom query
			// builders can opt out by not mapping to GORM expressions.
			continue
		}
	}
	if len(gormExpressions) == 0 {
		return nil
	}
	switch q.keyword {
	case keywordAnd:
		return clause.And(gormExpressions...)
	case keywordOr:
		return clause.Or(gormExpressions...)
	case keywordNot:
		return clause.Not(gormExpressions...)
	}
	return nil
}

// ================== Sharding Key Column Extraction ==================

// extractEqColumns collects column values from Eq and IN conditions.
// Multiple values per column are allowed; deduplication and validation
// happen later in the sharded table store.
// Range conditions (Gt/Gte/Lt/Lte) are recorded in rangeCols for diagnostics.
func (c *GormCondition) extractEqColumns(out map[string][]any, rangeCols map[string]bool) {
	switch expr := c.expr.(type) {
	case clause.Eq:
		col, ok := expr.Column.(clause.Column)
		if !ok {
			return
		}
		out[col.Name] = append(out[col.Name], expr.Value)
	case clause.IN:
		col, ok := expr.Column.(clause.Column)
		if !ok {
			return
		}
		out[col.Name] = append(out[col.Name], expr.Values...)
	case clause.Gt:
		if col, ok := expr.Column.(clause.Column); ok {
			rangeCols[col.Name] = true
		}
	case clause.Gte:
		if col, ok := expr.Column.(clause.Column); ok {
			rangeCols[col.Name] = true
		}
	case clause.Lt:
		if col, ok := expr.Column.(clause.Column); ok {
			rangeCols[col.Name] = true
		}
	case clause.Lte:
		if col, ok := expr.Column.(clause.Column); ok {
			rangeCols[col.Name] = true
		}
	case clause.AndConditions:
		for _, inner := range expr.Exprs {
			wrapped := &GormCondition{expr: inner}
			wrapped.extractEqColumns(out, rangeCols)
		}
	case clause.OrConditions:
		for _, inner := range expr.Exprs {
			wrapped := &GormCondition{expr: inner}
			wrapped.extractEqColumns(out, rangeCols)
		}
	}
}

// extractEqColumns recursively collects column-value pairs from Eq and IN conditions.
// AND and OR branches are both traversed to collect all values.
// NOT branches are skipped (negation doesn't provide usable routing values).
// Range conditions are recorded in rangeCols for diagnostic purposes.
func (q *GormQuery) extractEqColumns(out map[string][]any, rangeCols map[string]bool) {
	if q.keyword == keywordNot {
		return
	}
	for _, cond := range q.conditions {
		if cond == nil {
			continue
		}
		switch c := cond.(type) {
		case *GormCondition:
			c.extractEqColumns(out, rangeCols)
		case *GormQuery:
			c.extractEqColumns(out, rangeCols)
		case *gormColumnSelectionQuery:
			c.GormQuery.extractEqColumns(out, rangeCols)
		}
	}
}

// ExtractColumnsFromQuery extracts all column-value pairs from Eq and IN conditions
// in query trees (AND and OR). Each column may have multiple values.
// Also detects range conditions (Gt/Lt/Gte/Lte) on columns for diagnostic purposes.
// Returns the value map and a set of column names that have range conditions.
func ExtractColumnsFromQuery(query dbspi.Query) (values map[string][]any, rangeCols map[string]bool) {
	values = make(map[string][]any)
	rangeCols = make(map[string]bool)
	if query == nil {
		return values, rangeCols
	}
	switch q := query.(type) {
	case *GormQuery:
		q.extractEqColumns(values, rangeCols)
	case *gormColumnSelectionQuery:
		q.GormQuery.extractEqColumns(values, rangeCols)
	}
	return values, rangeCols
}

// ================== Updater Implementation ==================

// GormUpdater implements dbspi.Updater
type GormUpdater struct {
	updates map[string]any
}

// NewUpdater creates a new GormUpdater
func NewUpdater() *GormUpdater {
	return &GormUpdater{
		updates: make(map[string]any),
	}
}

// Set implements dbspi.Updater.
func (u *GormUpdater) Set(column dbspi.Column, value any) dbspi.Updater {
	key := column.Name()
	u.updates[key] = value
	return u
}

// SetMap implements dbspi.Updater
func (u *GormUpdater) SetMap(columnMap map[dbspi.Column]any) dbspi.Updater {
	for col, val := range columnMap {
		u.Set(col, val)
	}
	return u
}

// Remove implements dbspi.Updater
func (u *GormUpdater) Remove(column dbspi.Column) dbspi.Updater {
	key := column.Name()
	delete(u.updates, key)
	return u
}

func (u *GormUpdater) Values() map[string]any {
	return u.updates
}

type updaterValueReader interface {
	Values() map[string]any
}

func readUpdaterValues(updater dbspi.Updater) (map[string]any, bool) {
	reader, ok := updater.(updaterValueReader)
	if !ok {
		return nil, false
	}
	return reader.Values(), true
}

func requireUpdaterValues(updater dbspi.Updater) (map[string]any, error) {
	values, ok := readUpdaterValues(updater)
	if !ok {
		return nil, fmt.Errorf("dbhelper: unsupported updater implementation; use dbhelper.NewUpdater")
	}
	return values, nil
}

// ================== TableStore Implementation ==================

// GormTableStore implements dbspi.TableStore[T]
type GormTableStore[T dbspi.Entity] struct {
	db                  dbSession
	emptyEntityInstance T
	commonFields        CommonFieldAutoFillOptions
}

// NewTableStore creates a new GormTableStore with the given entity instance
// Example:
// NewTableStore(db, &User{})
func NewTableStore[T dbspi.Entity](db dbSession, entityInstance T) *GormTableStore[T] {
	return NewTableStoreWithTableName(db, entityInstance, entityInstance.TableName())
}

func NewTableStoreWithCommonFieldAutoFill[T dbspi.Entity](db dbSession, entityInstance T, commonFields CommonFieldAutoFillOptions) *GormTableStore[T] {
	return newTableStoreWithTableName(db, entityInstance, entityInstance.TableName(), commonFields)
}

// Shard is a no-op for a non-sharded table store and returns self.
func (e *GormTableStore[T]) Shard(_ *dbspi.ShardingKey) (dbspi.TableStore[T], error) {
	return e, nil
}

// FindAll is equivalent to Find for a non-sharded table store.
func (e *GormTableStore[T]) FindAll(ctx context.Context, query dbspi.Query, batchSize int) ([]T, error) {
	return e.Find(ctx, query, nil)
}

// CountAll is equivalent to Count for a non-sharded table store.
func (e *GormTableStore[T]) CountAll(ctx context.Context, query dbspi.Query) (uint64, error) {
	return e.Count(ctx, query)
}

// NewTableStoreWithTableName creates a new GormTableStore with the given entity instance and table name
// Example:
// NewTableStoreWithTableName(db, &User{}, "user_tab_00000001")
func NewTableStoreWithTableName[T dbspi.Entity](db dbSession, entityInstance T, tableName string) *GormTableStore[T] {
	return newTableStoreWithTableName(db, entityInstance, tableName, DisabledCommonFieldAutoFillOptions())
}

func NewTableStoreWithTableNameAndCommonFields[T dbspi.Entity](db dbSession, entityInstance T, tableName string, commonFields CommonFieldAutoFillOptions) *GormTableStore[T] {
	return newTableStoreWithTableName(db, entityInstance, tableName, commonFields)
}

func newTableStoreWithTableName[T dbspi.Entity](db dbSession, entityInstance T, tableName string, commonFields CommonFieldAutoFillOptions) *GormTableStore[T] {
	if any(entityInstance) == nil {
		panic("entityInstance is nil")
	}
	if tableName == "" {
		panic("tableName is empty")
	}

	// New a empty entity instance
	entity := reflect.New(reflect.TypeOf(reflect.ValueOf(entityInstance).Elem().Interface())).Interface()
	db = db.WithModel(entity).WithTableName(tableName)
	return &GormTableStore[T]{
		db:                  db,
		emptyEntityInstance: entity.(T),
		commonFields:        commonFields,
	}
}

// GetById implements dbspi.TableStore
func (e *GormTableStore[T]) GetById(ctx context.Context, id any) (T, error) {
	_, entity, err := e.ExistsById(ctx, id)
	return entity, err
}

// ExistsById implements dbspi.TableStore
func (e *GormTableStore[T]) ExistsById(ctx context.Context, id any) (bool, T, error) {
	var entity T
	if id == nil {
		return false, entity, nil
	}
	entities, err := e.Find(ctx, e.buildQueryById(id), nil)
	if err != nil {
		return false, entity, err
	}
	if len(entities) == 0 {
		return false, entity, nil
	}
	return true, entities[0], nil
}

// UpdateById implements dbspi.TableStore
func (e *GormTableStore[T]) UpdateById(ctx context.Context, id any, updater dbspi.Updater) error {
	return e.UpdateByQuery(ctx, e.buildQueryById(id), updater)
}

// DeleteById implements dbspi.TableStore
func (e *GormTableStore[T]) DeleteById(ctx context.Context, id any) error {
	return e.DeleteByQuery(ctx, e.buildQueryById(id))
}

// Find implements dbspi.TableStore
func (e *GormTableStore[T]) Find(ctx context.Context, query dbspi.Query, pagination dbspi.Pagination) ([]T, error) {
	var results []T
	err := e.db.Find(ctx, &results, query, pagination)
	return results, err
}

// Exists implements dbspi.TableStore
func (e *GormTableStore[T]) Exists(ctx context.Context, query dbspi.Query) (bool, T, error) {
	var entity T
	limit := 1
	paginationConfig := NewPagination().WithLimit(&limit)
	entities, err := e.Find(ctx, query, paginationConfig)
	if err != nil {
		return false, entity, err
	}
	if len(entities) == 0 {
		return false, entity, nil
	}
	return true, entities[0], nil
}

// Count implements dbspi.TableStore
func (e *GormTableStore[T]) Count(ctx context.Context, query dbspi.Query) (uint64, error) {
	return e.db.Count(ctx, query)
}

// Create implements dbspi.TableStore
func (e *GormTableStore[T]) Create(ctx context.Context, value T) error {
	applyCreateCommonFields(ctx, e.commonFields, value)
	return e.db.Create(ctx, value)
}

// Save implements dbspi.TableStore
func (e *GormTableStore[T]) Save(ctx context.Context, value T) error {
	applySaveCommonFields(ctx, e.commonFields, value)
	return e.db.Save(ctx, value)
}

// Update implements dbspi.TableStore
func (e *GormTableStore[T]) Update(ctx context.Context, entity T) error {
	applyUpdateCommonFields(ctx, e.commonFields, entity)
	return e.db.Update(ctx, entity)
}

// Delete implements dbspi.TableStore
func (e *GormTableStore[T]) Delete(ctx context.Context, entity T) error {
	return e.db.Delete(ctx, entity)
}

// UpdateByQuery implements dbspi.TableStore
func (e *GormTableStore[T]) UpdateByQuery(ctx context.Context, query dbspi.Query, updater dbspi.Updater) error {
	applyUpdateCommonFieldsToUpdater(ctx, e.commonFields, e.emptyEntityInstance, updater)
	return e.db.UpdateByQuery(ctx, query, updater)
}

// DeleteByQuery implements dbspi.TableStore
func (e *GormTableStore[T]) DeleteByQuery(ctx context.Context, query dbspi.Query) error {
	return e.db.DeleteByQuery(ctx, e.emptyEntityInstance, query)
}

// BatchCreate implements dbspi.TableStore
func (e *GormTableStore[T]) BatchCreate(ctx context.Context, entities []T, batchSize int) error {
	applyCreateCommonFieldsToSlice(ctx, e.commonFields, entities)
	err := e.db.BatchCreate(ctx, entities, batchSize)
	return err
}

// BatchSave implements dbspi.TableStore
func (e *GormTableStore[T]) BatchSave(ctx context.Context, entities []T) error {
	applySaveCommonFieldsToSlice(ctx, e.commonFields, entities)
	err := e.db.BatchSave(ctx, entities)
	return err
}

// FirstOrCreate implements dbspi.TableStore
func (e *GormTableStore[T]) FirstOrCreate(ctx context.Context, entity T, query dbspi.Query) (T, error) {
	applyCreateCommonFields(ctx, e.commonFields, entity)
	err := e.db.FirstOrCreate(ctx, entity, query)
	return entity, err
}

// Raw implements dbspi.SQLTableStore.
func (e *GormTableStore[T]) Raw(ctx context.Context, sql string, args ...any) ([]T, error) {
	var results []T
	err := e.db.Raw(ctx, &results, sql, args...)
	return results, err
}

// Exec implements dbspi.SQLTableStore.
func (e *GormTableStore[T]) Exec(ctx context.Context, sql string, args ...any) error {
	return e.db.Exec(ctx, sql, args...)
}

func (e *GormTableStore[T]) buildQueryById(id any) dbspi.Query {
	idFieldName := dbspi.DefaultIdFieldName
	if namer, ok := any(e.emptyEntityInstance).(dbspi.IdFieldNameProvider); ok {
		idFieldName = namer.IdFieldName()
	}
	return NewQuery(NewField[any](idFieldName).Eq(&id))
}

type GormDb struct {
	db *gorm.DB
}

// NewGormDb creates a new GormDb.
func NewGormDb(dbConfig dbspi.ServerConfig) dbSession {
	dbConfig = normalizeServerConfig(dbConfig)
	gormCfg := &gorm.Config{}

	db, err := gorm.Open(mysql.Open(dbServerDSN(dbConfig)), gormCfg)
	if err != nil {
		panic(err)
	}

	if dbConfig.Debug {
		db = db.Debug()
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	if dbConfig.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
	}
	if dbConfig.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
	}
	if dbConfig.ConnMaxLifetimeSeconds > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(dbConfig.ConnMaxLifetimeSeconds) * time.Second)
	}

	return &GormDb{
		db: db,
	}
}

func normalizeServerConfig(cfg dbspi.ServerConfig) dbspi.ServerConfig {
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = dbspi.DefaultMaxOpenConns
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = dbspi.DefaultMaxIdleConns
	}
	if cfg.ConnMaxLifetimeSeconds == 0 {
		cfg.ConnMaxLifetimeSeconds = dbspi.DefaultConnMaxLifetimeSeconds
	}
	return cfg
}

func dbServerDSN(cfg dbspi.ServerConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DatabaseName)
}

// WithModel implements dbSession
func (d *GormDb) WithModel(model any) dbSession {
	return &GormDb{db: d.db.Model(model)}
}

// WithTable implements dbSession
func (d *GormDb) WithTableName(tableName string) dbSession {
	return &GormDb{db: d.db.Table(tableName)}
}

// Find implements dbSession
func (d *GormDb) Find(ctx context.Context, dest any, query dbspi.Query, pagination dbspi.Pagination) error {
	db := d.db.WithContext(ctx)
	if pagination != nil {
		if pagination.Limit() != nil {
			db = db.Limit(*pagination.Limit())
		}
		if pagination.Offset() != nil {
			db = db.Offset(*pagination.Offset())
		}
		if len(pagination.Orders()) > 0 {
			orders := make([]clause.OrderByColumn, 0, len(pagination.Orders()))
			for _, order := range pagination.Orders() {
				orders = append(orders, clause.OrderByColumn{
					Column: clause.Column{Name: order.Column().Name(), Raw: true},
					Desc:   order.Desc(),
				})
			}
			db = db.Clauses(clause.OrderBy{
				Columns: orders,
			})
		}
	}

	if sq, ok := query.(columnSelectionQuery); ok && len(sq.Columns()) > 0 {
		colNames := make([]string, len(sq.Columns()))
		for i, c := range sq.Columns() {
			colNames[i] = c.Name()
		}
		db = db.Select(colNames)
	}

	gormClause := queryToGormClause(query)
	if gormClause != nil {
		db = db.Clauses(gormClause)
	}

	return db.Find(dest).Error
}

// Count implements dbSession
func (d *GormDb) Count(ctx context.Context, query dbspi.Query) (uint64, error) {
	var count int64
	db := d.db.WithContext(ctx)
	gormClause := queryToGormClause(query)
	if gormClause != nil {
		db = db.Clauses(gormClause)
	}
	err := db.Count(&count).Error
	return uint64(count), err
}

// Create implements dbSession
func (d *GormDb) Create(ctx context.Context, entity dbspi.Entity) error {
	err := d.db.WithContext(ctx).Create(entity).Error
	return err
}

// Save implements dbSession
func (d *GormDb) Save(ctx context.Context, entity dbspi.Entity) error {
	err := d.db.WithContext(ctx).Save(entity).Error
	return err
}

// Update implements dbSession
func (d *GormDb) Update(ctx context.Context, entity dbspi.Entity) error {
	err := d.db.WithContext(ctx).Model(entity).Updates(entity).Error
	return err
}

// Delete implements dbSession
func (d *GormDb) Delete(ctx context.Context, entity dbspi.Entity) error {
	err := d.db.WithContext(ctx).Delete(entity).Error
	return err
}

// UpdateByQuery implements dbSession
func (d *GormDb) UpdateByQuery(ctx context.Context, query dbspi.Query, updater dbspi.Updater) error {
	updates, err := requireUpdaterValues(updater)
	if err != nil {
		return err
	}
	db := d.db.WithContext(ctx)
	gormClause := queryToGormClause(query)
	if gormClause != nil {
		db = db.Clauses(gormClause)
	}
	err = db.Updates(updates).Error
	return err
}

func (d *GormDb) DeleteByQuery(ctx context.Context, entity dbspi.Entity, query dbspi.Query) error {
	db := d.db.WithContext(ctx)
	gormClause := queryToGormClause(query)
	if gormClause != nil {
		db = db.Clauses(gormClause)
	}
	err := db.Delete(entity).Error
	return err
}

// BatchCreate implements dbSession
func (d *GormDb) BatchCreate(ctx context.Context, entities any, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1000
	}
	err := d.db.WithContext(ctx).CreateInBatches(entities, batchSize).Error
	return err
}

// BatchSave implements dbSession
func (d *GormDb) BatchSave(ctx context.Context, entities any) error {
	err := d.db.WithContext(ctx).Save(entities).Error
	return err
}

// Raw implements dbSession
func (d *GormDb) Raw(ctx context.Context, dest any, sql string, args ...any) error {
	err := d.db.WithContext(ctx).Raw(sql, args...).Scan(dest).Error
	return err
}

// Exec implements dbSession
func (d *GormDb) Exec(ctx context.Context, sql string, args ...any) error {
	err := d.db.WithContext(ctx).Exec(sql, args...).Error
	return err
}

// FirstOrCreate implements dbSession
func (d *GormDb) FirstOrCreate(ctx context.Context, entity dbspi.Entity, query dbspi.Query) error {
	db := d.db.WithContext(ctx)
	gormClause := queryToGormClause(query)
	if gormClause != nil {
		db = db.Clauses(gormClause)
	}
	return db.FirstOrCreate(entity).Error
}

// Transaction implements dbSession
func (d *GormDb) Transaction(ctx context.Context, fn transactionFunc) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDB := &GormDb{db: tx}
		return fn(txDB)
	})
}

func queryToGormClause(query dbspi.Query) clause.Expression {
	if query == nil {
		return nil
	}
	if gq, ok := query.(gormExpression); ok {
		return gq.ToGormExpression()
	}
	return nil
}
