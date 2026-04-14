package dbsp

import (
	"context"
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

func (t _tableForCheck) IdFiledName() string {
	return "id"
}

var (
	// Check external interfaces
	_ dbspi.Condition                = (*GormCondition)(nil)
	_ dbspi.Column                   = (*GormColumn)(nil)
	_ dbspi.Field[any]               = (*GormField[any])(nil)
	_ dbspi.Query                    = (*GormQuery)(nil)
	_ dbspi.Updater                  = (*GormUpdater)(nil)
	_ dbspi.Executor[_tableForCheck] = new(GormExecutor[_tableForCheck])

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

// GormSelectQuery implements dbspi.SelectQuery
type GormSelectQuery struct {
	*GormQuery
	columns []dbspi.Column
}

func (q *GormSelectQuery) Columns() []dbspi.Column {
	return q.columns
}

// Select wraps a query with specific column selection.
func Select(columns []dbspi.Column, conditions ...dbspi.Condition) dbspi.SelectQuery {
	return &GormSelectQuery{
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
			// TODO: Warning or error log ?
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

// Add implements dbspi.Updater
func (u *GormUpdater) Add(column dbspi.Column, value any) dbspi.Updater {
	key := column.Name()
	u.updates[key] = value
	return u
}

// AddByMap implements dbspi.Updater
func (u *GormUpdater) AddByMap(columnMap map[dbspi.Column]any) dbspi.Updater {
	for col, val := range columnMap {
		u.Add(col, val)
	}
	return u
}

// Remove implements dbspi.Updater
func (u *GormUpdater) Remove(column dbspi.Column) dbspi.Updater {
	key := column.Name()
	delete(u.updates, key)
	return u
}

// Params implements dbspi.Updater
func (u *GormUpdater) Params() map[string]any {
	return u.updates
}

// ================== Executor Implementation ==================

// GormExecutor implements dbspi.Executor[T]
type GormExecutor[T dbspi.Entity] struct {
	db                  dbspi.Db
	emptyEntityInstance T
}

// NewExecutor creates a new GormExecutor with the given entity instance
// Example:
// NewExecutor(db, &User{})
func NewExecutor[T dbspi.Entity](db dbspi.Db, entityInstance T) *GormExecutor[T] {
	return NewExecutorWithTableName(db, entityInstance, entityInstance.TableName())
}

// Shard is a no-op for non-sharded executor, returns self.
func (e *GormExecutor[T]) Shard(_ any) (dbspi.Executor[T], error) {
	return e, nil
}

// FindAll is equivalent to Find for non-sharded executor.
func (e *GormExecutor[T]) FindAll(ctx context.Context, query dbspi.Query, batchSize int) ([]T, error) {
	return e.Find(ctx, query, nil)
}

// CountAll is equivalent to Count for non-sharded executor.
func (e *GormExecutor[T]) CountAll(ctx context.Context, query dbspi.Query) (uint64, error) {
	return e.Count(ctx, query)
}

// NewExecutorWithTableName creates a new GormExecutor with the given entity instance and table name
// Example:
// NewExecutorWithTableName(db, &User{}, "user_tab_00000001")
func NewExecutorWithTableName[T dbspi.Entity](db dbspi.Db, entityInstance T, tableName string) *GormExecutor[T] {
	if any(entityInstance) == nil {
		panic("entityInstance is nil")
	}
	if tableName == "" {
		panic("tableName is empty")
	}

	// New a empty entity instance
	entity := reflect.New(reflect.TypeOf(reflect.ValueOf(entityInstance).Elem().Interface())).Interface()
	db = db.WithModel(entity).WithTableName(tableName)
	return &GormExecutor[T]{
		db:                  db,
		emptyEntityInstance: entity.(T),
	}
}

// GetById implements dbspi.Executor
func (e *GormExecutor[T]) GetById(ctx context.Context, id any) (T, error) {
	_, entity, err := e.ExistsById(ctx, id)
	return entity, err
}

// ExistsById implements dbspi.Executor
func (e *GormExecutor[T]) ExistsById(ctx context.Context, id any) (bool, T, error) {
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

// UpdateById implements dbspi.Executor
func (e *GormExecutor[T]) UpdateById(ctx context.Context, id any, updater dbspi.Updater) error {
	return e.UpdateByQuery(ctx, e.buildQueryById(id), updater)
}

// DeleteById implements dbspi.Executor
func (e *GormExecutor[T]) DeleteById(ctx context.Context, id any) error {
	return e.DeleteByQuery(ctx, e.buildQueryById(id))
}

// Find implements dbspi.Executor
func (e *GormExecutor[T]) Find(ctx context.Context, query dbspi.Query, pagenation dbspi.PaginationConfig) ([]T, error) {
	var results []T
	err := e.db.Find(ctx, &results, query, pagenation)
	return results, err
}

// Exists implements dbspi.Executor
func (e *GormExecutor[T]) Exists(ctx context.Context, query dbspi.Query) (bool, T, error) {
	var entity T
	limit := 1
	paginationConfig := NewPaginationConfig().WithLimit(&limit)
	entities, err := e.Find(ctx, query, paginationConfig)
	if err != nil {
		return false, entity, err
	}
	if len(entities) == 0 {
		return false, entity, nil
	}
	return true, entities[0], nil
}

// Count implements dbspi.Executor
func (e *GormExecutor[T]) Count(ctx context.Context, query dbspi.Query) (uint64, error) {
	return e.db.Count(ctx, query)
}

// Create implements dbspi.Executor
func (e *GormExecutor[T]) Create(ctx context.Context, value T) error {
	return e.db.Create(ctx, value)
}

// Save implements dbspi.Executor
func (e *GormExecutor[T]) Save(ctx context.Context, value T) error {
	return e.db.Save(ctx, value)
}

// Update implements dbspi.Executor
func (e *GormExecutor[T]) Update(ctx context.Context, entity T) error {
	return e.db.Update(ctx, entity)
}

// Delete implements dbspi.Executor
func (e *GormExecutor[T]) Delete(ctx context.Context, entity T) error {
	return e.db.Delete(ctx, entity)
}

// UpdateByQuery implements dbspi.Executor
func (e *GormExecutor[T]) UpdateByQuery(ctx context.Context, query dbspi.Query, updater dbspi.Updater) error {
	return e.db.UpdateByQuery(ctx, query, updater)
}

// DeleteByQuery implements dbspi.Executor
func (e *GormExecutor[T]) DeleteByQuery(ctx context.Context, query dbspi.Query) error {
	return e.db.DeleteByQuery(ctx, e.emptyEntityInstance, query)
}

// BatchCreate implements dbspi.Executor
func (e *GormExecutor[T]) BatchCreate(ctx context.Context, entities []T, batchSize int) error {
	err := e.db.BatchCreate(ctx, entities, batchSize)
	return err
}

// BatchSave implements dbspi.Executor
func (e *GormExecutor[T]) BatchSave(ctx context.Context, entities []T) error {
	err := e.db.BatchSave(ctx, entities)
	return err
}

// Upsert implements dbspi.Executor
func (e *GormExecutor[T]) Upsert(ctx context.Context, entity T, updateColumns []dbspi.Column) error {
	return e.db.Upsert(ctx, entity, updateColumns)
}

// FirstOrCreate implements dbspi.Executor
func (e *GormExecutor[T]) FirstOrCreate(ctx context.Context, entity T, query dbspi.Query) (T, error) {
	err := e.db.FirstOrCreate(ctx, entity, query)
	return entity, err
}

// Raw implements dbspi.Executor
func (e *GormExecutor[T]) Raw(ctx context.Context, sql string, args ...any) ([]T, error) {
	var results []T
	err := e.db.Raw(ctx, &results, sql, args...)
	return results, err
}

// Exec implements dbspi.Executor
func (e *GormExecutor[T]) Exec(ctx context.Context, sql string, args ...any) error {
	return e.db.Exec(ctx, sql, args...)
}

func (e *GormExecutor[T]) buildQueryById(id any) dbspi.Query {
	var entity T
	idFieldName := "id"
	if ider, ok := any(entity).(dbspi.Ider); ok {
		idFieldName = ider.IdFiledName()
	}
	return NewQuery(NewField[any](idFieldName).Eq(&id))
}

type GormDb struct {
	db *gorm.DB
}

// NewGormDb creates a new GormDb
func NewGormDb(dbConfig dbspi.DbConfig) dbspi.Db {
	gormCfg := &gorm.Config{}

	db, err := gorm.Open(mysql.Open(dbConfig.GetDSN()), gormCfg)
	if err != nil {
		panic(err)
	}

	if dbConfig.DebugMode() {
		db = db.Debug()
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	if dbConfig.MaxOpenConns() > 0 {
		sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns())
	}
	if dbConfig.MaxIdleConns() > 0 {
		sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns())
	}
	if dbConfig.ConnMaxLifetimeSeconds() > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(dbConfig.ConnMaxLifetimeSeconds()) * time.Second)
	}

	return &GormDb{
		db: db,
	}
}

// WithModel implements dbspi.Db
func (d *GormDb) WithModel(model any) dbspi.Db {
	return &GormDb{db: d.db.Model(model)}
}

// WithTable implements dbspi.Db
func (d *GormDb) WithTableName(tableName string) dbspi.Db {
	return &GormDb{db: d.db.Table(tableName)}
}

// Find implements dbspi.Db
func (d *GormDb) Find(ctx context.Context, dest any, query dbspi.Query, pagenation dbspi.PaginationConfig) error {
	db := d.db.WithContext(ctx)
	if pagenation != nil {
		if pagenation.Limit() != nil {
			db = db.Limit(*pagenation.Limit())
		}
		if pagenation.Offset() != nil {
			db = db.Offset(*pagenation.Offset())
		}
		if len(pagenation.Orders()) > 0 {
			orders := make([]clause.OrderByColumn, 0, len(pagenation.Orders()))
			for _, order := range pagenation.Orders() {
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

	if sq, ok := query.(dbspi.SelectQuery); ok && len(sq.Columns()) > 0 {
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

// Count implements dbspi.Db
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

// Create implements dbspi.Db
func (d *GormDb) Create(ctx context.Context, entity dbspi.Entity) error {
	err := d.db.WithContext(ctx).Create(entity).Error
	return err
}

// Save implements dbspi.Db
func (d *GormDb) Save(ctx context.Context, entity dbspi.Entity) error {
	err := d.db.WithContext(ctx).Save(entity).Error
	return err
}

// Update implements dbspi.Db
func (d *GormDb) Update(ctx context.Context, entity dbspi.Entity) error {
	err := d.db.WithContext(ctx).Updates(entity).Error
	return err
}

// Delete implements dbspi.Db
func (d *GormDb) Delete(ctx context.Context, entity dbspi.Entity) error {
	err := d.db.WithContext(ctx).Delete(entity).Error
	return err
}

// UpdateByQuery implements dbspi.Db
func (d *GormDb) UpdateByQuery(ctx context.Context, query dbspi.Query, updater dbspi.Updater) error {
	db := d.db.WithContext(ctx)
	gormClause := queryToGormClause(query)
	if gormClause != nil {
		db = db.Clauses(gormClause)
	}
	err := db.Updates(updater.Params()).Error
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

// BatchCreate implements dbspi.Db
func (d *GormDb) BatchCreate(ctx context.Context, entities any, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1000
	}
	err := d.db.WithContext(ctx).CreateInBatches(entities, batchSize).Error
	return err
}

// BatchSave implements dbspi.Db
func (d *GormDb) BatchSave(ctx context.Context, entities any) error {
	err := d.db.WithContext(ctx).Save(entities).Error
	return err
}

// Raw implements dbspi.Db
func (d *GormDb) Raw(ctx context.Context, dest any, sql string, args ...any) error {
	err := d.db.WithContext(ctx).Raw(sql, args...).Scan(dest).Error
	return err
}

// Exec implements dbspi.Db
func (d *GormDb) Exec(ctx context.Context, sql string, args ...any) error {
	err := d.db.WithContext(ctx).Exec(sql, args...).Error
	return err
}

// Upsert implements dbspi.Db
func (d *GormDb) Upsert(ctx context.Context, entity dbspi.Entity, updateColumns []dbspi.Column) error {
	if len(updateColumns) == 0 {
		return d.db.WithContext(ctx).Save(entity).Error
	}
	cols := make([]clause.Column, len(updateColumns))
	for i, c := range updateColumns {
		cols[i] = clause.Column{Name: c.Name()}
	}
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns(columnNames(updateColumns)),
	}).Create(entity).Error
}

// FirstOrCreate implements dbspi.Db
func (d *GormDb) FirstOrCreate(ctx context.Context, entity dbspi.Entity, query dbspi.Query) error {
	db := d.db.WithContext(ctx)
	gormClause := queryToGormClause(query)
	if gormClause != nil {
		db = db.Clauses(gormClause)
	}
	return db.FirstOrCreate(entity).Error
}

func columnNames(columns []dbspi.Column) []string {
	names := make([]string, len(columns))
	for i, c := range columns {
		names[i] = c.Name()
	}
	return names
}

// Transaction implements dbspi.Db
func (d *GormDb) Transaction(ctx context.Context, fn dbspi.TxFn) error {
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
