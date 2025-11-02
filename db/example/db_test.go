package example

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// Example demonstrates how to use the GORM implementation of dbspi

// User model example
type User struct {
	ID      int64  `gorm:"primaryKey"`
	Name    string `gorm:"column:name"`
	Email   string `gorm:"column:email"`
	Age     int    `gorm:"column:age"`
	Status  string `gorm:"column:status"`
	Deleted bool   `gorm:"column:deleted"`
}

func (*User) TableName() string {
	return "dbspi_test_user_tab"
}

func (*User) IdFiledName() string {
	return "id"
}

// UserTable represents the user table with type-safe fields
type UserFieldManager struct {
	ID      dbspi.Field[int64]
	Name    dbspi.Field[string]
	Email   dbspi.Field[string]
	Age     dbspi.Field[int]
	Status  dbspi.Field[string]
	Deleted dbspi.Field[bool]
}

// NewUserFieldManager creates a new UserFieldManager with field definitions
func NewUserFieldManager() *UserFieldManager {
	return &UserFieldManager{
		ID:      dbhelper.NewField[int64]("id"),
		Name:    dbhelper.NewField[string]("name"),
		Email:   dbhelper.NewField[string]("email"),
		Age:     dbhelper.NewField[int]("age"),
		Status:  dbhelper.NewField[string]("status"),
		Deleted: dbhelper.NewField[bool]("deleted"),
	}
}

func testNewDb() dbspi.Db {
	dbConfig := dbhelper.NewDbConfig("127.0.0.1", 3306, "root", "123456", "my_test")
	return dbhelper.NewDb(dbConfig)
}

// ExampleUsage demonstrates various usage patterns
func Test_ExampleUsage(t *testing.T) {
	ctx := context.Background()

	db := testNewDb()

	fm := NewUserFieldManager()
	executor := dbhelper.NewExecutor(db, &User{})
	// executor := dbhelper.NewExecutorWithTableName(db, &User{ID: 10}, "dbspi_test_user_tab_00000001")

	// Example 1: Find all users
	// query1 := dbhelper.NewQuery()
	users, err := executor.Find(ctx, nil, nil)
	t.Logf("Example 1: Find all users: users: %v, err: %v", users, err)

	// Example 2: Complex query with OR and ordering
	email := "alice@example.com"
	name := "Alice"
	query2 := dbhelper.Q(
		dbhelper.Or(
			fm.Email.Eq(ptrString(email)),
			fm.Name.Like(ptrString(name)),
		),
		dbhelper.And(
			fm.Deleted.Eq(ptrBool(false)),
		),
	)
	users, err = executor.Find(ctx, query2, nil)
	t.Logf("Example 2: Complex query with OR and ordering: users: %v, err: %v", users, err)

	// Example 3: Count
	count, err := executor.Count(ctx, query2)
	t.Logf("Example 3: Count users: count: %d, err: %v", count, err)

	// Example 4: Create
	newUser := User{
		Name:   "Alice",
		Email:  "alice@example.com",
		Age:    25,
		Status: "active",
	}
	err = executor.Create(ctx, &newUser)
	t.Logf("Example 4: Create user: err: %v", err)

	// Example 5: Update with conditions
	updater := dbhelper.NewUpdater().
		Add(fm.Status, "inactive").
		Add(fm.Age, 30)

	query3 := dbhelper.Q(fm.Email.Eq(ptrString(email)))
	err = executor.UpdateByQuery(ctx, query3, updater)
	t.Logf("Example 5: Update user: err: %v", err)

	// Example 6: Delete
	query4 := dbhelper.Q(fm.ID.Eq(ptrInt64(100)))
	err = executor.DeleteByQuery(ctx, query4)
	t.Logf("Example 6: Delete user: err: %v", err)

	// Example 7: String operations
	searchTerm := "test"
	query5 := dbhelper.Q(
		fm.Name.Contains(ptrString(searchTerm)),
	)
	users, err = executor.Find(ctx, query5, nil)
	t.Logf("Example 7: String operations: users: %v, err: %v", users, err)

	// Example 8: Range queries
	query6 := dbhelper.Q(fm.Age.Gt(ptrInt(18)), fm.Age.Lt(ptrInt(65)))
	users, err = executor.Find(ctx, query6, nil)
	t.Logf("Example 8: Range queries: users: %v, err: %v", users, err)

	// Example 9: IN queries
	statusList := []string{"active", "pending"}
	query7 := dbhelper.Q(
		fm.Status.In(statusList),
	)
	users, _ = executor.Find(ctx, query7, nil)
	t.Logf("Example 9: IN queries: users: %v, err: %v", users, err)

	// Example 10: NULL checks
	query8 := dbhelper.Q(
		fm.Email.IsNotNull(),
	)
	users, _ = executor.Find(ctx, query8, nil)
	t.Logf("Example 10: NULL checks: users: %v, err: %v", users, err)

	// Example 11: NOT condition
	activeStatus := "active"
	query9 := dbhelper.Not(
		fm.Status.Eq(ptrString(activeStatus)),
	)
	users, _ = executor.Find(ctx, query9, nil)
	t.Logf("Example 11: NOT condition: users: %v, err: %v", users, err)

	// Example 12: Pagination
	limit := 10
	offset := 20
	query10 := dbhelper.Q()

	paginationConfig := dbhelper.NewPaginationConfig().
		WithLimit(ptrInt(limit)).
		WithOffset(ptrInt(offset)).
		AppendOrder(dbhelper.NewOrderConfig(fm.ID, true))
	users, _ = executor.Find(ctx, query10, paginationConfig)
	t.Logf("Example 12: Pagination: users: %v, err: %v", users, err)

	// Example 13:  warpped query expression
	query11 := dbhelper.Q(fm.Age.Gt(ptrInt(18)), fm.Status.Eq(ptrString(activeStatus)))
	query11 = dbhelper.Q(query11)
	users, _ = executor.Find(ctx, query11, nil)
	t.Logf("Example 13: Warpped query expression: users: %v, err: %v", users, err)

	// Example 14: Combine multiple conditions
	cond1 := fm.Age.Gt(ptrInt(18))
	cond2 := fm.Status.Eq(ptrString(activeStatus))
	cond3 := fm.Deleted.Eq(ptrBool(false))
	combinedCond := dbhelper.Q(cond1, cond2, cond3)
	query12 := dbhelper.Q(combinedCond)
	users, _ = executor.Find(ctx, query12, nil)
	t.Logf("Example 14: Combine multiple conditions: users: %v, err: %v", users, err)

	// Example 15: Combine multiple queries
	// Expect: age > 18 or (status = 'active' and deleted = false and (name like '%test%' or name like '%test%'))
	q1 := dbhelper.Q(fm.Age.Gt(ptrInt(18)))
	q2 := dbhelper.Q(fm.Status.Eq(ptrString(activeStatus)))
	q3 := dbhelper.Q(fm.Deleted.Eq(ptrBool(false)), q2)
	q4 := dbhelper.Or(fm.Name.StartsWith(ptrString(searchTerm)), fm.Name.EndsWith(ptrString(searchTerm)))
	q5 := dbhelper.Q(q3, q4)

	combinedQ := dbhelper.Or(q1, q5)
	users, _ = executor.Find(ctx, combinedQ, nil)
	t.Logf("Example 15: Combine multiple queries: users: %v, err: %v", users, err)

	// Example 16: Get by id
	id := 34
	user, err := executor.GetById(ctx, id)
	t.Logf("Example 16: Get by id: user: %v, err: %v", user, err)

	// Example 17: Exists by id
	exists, user, err := executor.ExistsById(ctx, id)
	t.Logf("Example 17: Exists by id: exists: %v, user: %v, err: %v", exists, user, err)

	// Example 18: Update by id
	updater = dbhelper.NewUpdater().
		Add(fm.Status, "inactive").
		Add(fm.Age, 30)
	err = executor.UpdateById(ctx, id, updater)
	t.Logf("Example 18: Update by id: user: %v, err: %v", user, err)

	// Example 19: Delete by id
	err = executor.DeleteById(ctx, 1)
	t.Logf("Example 19: Delete by id: err: %v", err)

	// Example 20: Batch create
	users = []*User{
		{Name: "Alice", Email: "alice@example.com", Age: 25, Status: "active"},
		{Name: "Bob", Email: "bob@example.com", Age: 30, Status: "inactive"},
	}
	err = executor.BatchCreate(ctx, users, 1000)
	t.Logf("Example 20: Batch create: users: %v, err: %v", users, err)

	// Example 21: Batch save
	users = []*User{
		{Name: "Alice111", Email: "alice111@example.com", Age: 25, Status: "active"},
		{ID: 34, Name: "Bob", Email: "bob@example.com", Age: 30, Status: "inactive"},
	}
	err = executor.BatchSave(ctx, users)
	t.Logf("Example 21: Batch save: users: %v, err: %v", users, err)

	// Example 23: Update by query
	query := dbhelper.Q(fm.Email.Eq(ptrString(email)))
	updater = dbhelper.NewUpdater().
		Add(fm.Status, "inactive").
		Add(fm.Age, 30)
	err = executor.UpdateByQuery(ctx, query, updater)
	t.Logf("Example 23: Update by query: err: %v", err)

	// Example 24: Delete by query
	query = dbhelper.Q(fm.Email.Eq(ptrString("abc@example.com")))
	err = executor.DeleteByQuery(ctx, query)
	t.Logf("Example 24: Delete by query: err: %v", err)

	// Example 25: Raw
	users, err = executor.Raw(ctx, "SELECT * FROM users WHERE email = ?", email)
	t.Logf("Example 25: Raw: users: %v, err: %v", users, err)

	// Example 26: Exec
	err = executor.Exec(ctx, "UPDATE dbspi_test_user_tab SET status = ? WHERE email = ?", "inactive", email)
	t.Logf("Example 26: Exec: err: %v", err)
}

// Helper functions for pointer creation
func ptrInt(i int) *int {
	return &i
}

func ptrInt64(i int64) *int64 {
	return &i
}

func ptrBool(b bool) *bool {
	return &b
}

func ptrString(s string) *string {
	return &s
}

// ExampleUpdaterWithMap demonstrates using updater with map
func Test_ExampleUpdaterWithMap(t *testing.T) {
	ctx := context.Background()
	db := testNewDb()
	executor := dbhelper.NewExecutor(db, &User{})
	fm := NewUserFieldManager()

	// Create updater with map
	updater := dbhelper.NewUpdater()
	columnMap := map[dbspi.Column]any{
		fm.Name:   "New Name",
		fm.Status: "inactive",
		fm.Age:    35,
	}
	updater.AddByMap(columnMap)

	// Can also remove columns from update
	updater.Remove(fm.Age)

	// Apply update
	email := "test@example.com"
	query := dbhelper.Q(fm.Email.Eq(ptrString(email)))
	_ = executor.UpdateByQuery(ctx, query, updater)
}
