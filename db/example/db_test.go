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
	ID       int64  `gorm:"primaryKey"`
	Name     string `gorm:"column:name"`
	Email    string `gorm:"column:email"`
	Age      int    `gorm:"column:age"`
	Status   string `gorm:"column:status"`
	Deleted  bool   `gorm:"column:deleted"`
}

func (User) Table() string {
	return "dbspi_test_user_tab"
}

// UserTable represents the user table with type-safe fields
type UserFieldManager struct {
	ID     dbspi.Field[int64]
	Name   dbspi.Field[string]
	Email  dbspi.Field[string]
	Age    dbspi.Field[int]
	Status dbspi.Field[string]
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
	executor := dbhelper.NewExecutor[User](db)

	// Example 1: Find all users
	// query1 := dbhelper.NewQuery()
	users, err := executor.Find(ctx, nil, nil)
	if err != nil {
		t.Errorf("find error: %v", err)
	}
	t.Logf("users: %v", users)

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
	// Example 3: Count
	count, err := executor.Count(ctx, query2)
	if err != nil {
		t.Errorf("count error: %v", err)
		_ = count
	}
	t.Logf("count: %d", count)
	
	// Example 4: Create
	newUser := User{
		Name:   "Alice",
		Email:  "alice@example.com",
		Age:    25,
		Status: "active",
	}
	err = executor.Create(ctx, &newUser)
	if err != nil {
		t.Errorf("create error: %v", err)
	}
	
	// Example 5: Update with conditions
	updater := dbhelper.NewUpdater().
		Add(fm.Status, "inactive").
		Add(fm.Age, 30)
	
	query3 := dbhelper.Q(fm.Email.Eq(ptrString(email)))
	err = executor.Update(ctx, query3, updater)
	if err != nil {
		t.Errorf("update error: %v", err)
	}
	
	// Example 6: Delete
	query4 := dbhelper.Q(fm.ID.Eq(ptrInt64(100)))
	err = executor.Delete(ctx, query4)
	if err != nil {
		t.Errorf("delete error: %v", err)
	}
	
	// Example 7: String operations
	searchTerm := "test"
	query5 := dbhelper.Q(
		fm.Name.Contains(ptrString(searchTerm)),
	)
	users, err = executor.Find(ctx, query5, nil)
	if err != nil {
		t.Errorf("find error: %v", err)
	}
	t.Logf("users: %v", users)
	
	// Example 8: Range queries
	query6 := dbhelper.Q(fm.Age.Gt(ptrInt(18)), fm.Age.Lt(ptrInt(65)))
	users, _ = executor.Find(ctx, query6, nil)
	t.Logf("users: %v", users)
	
	// Example 9: IN queries
	statusList := []string{"active", "pending"}
	query7 := dbhelper.Q(
		fm.Status.In(statusList),
	)
	users, _ = executor.Find(ctx, query7, nil)
	t.Logf("users: %v", users)
	
	// Example 10: NULL checks
	query8 := dbhelper.Q(
		fm.Email.IsNotNull(),
	)
	users, _ = executor.Find(ctx, query8, nil)
	t.Logf("users: %v", users)
	
	// Example 11: NOT condition
	activeStatus := "active"
	query9 := dbhelper.Not(
		fm.Status.Eq(ptrString(activeStatus)),
	)
	users, _ = executor.Find(ctx, query9, nil)
	t.Logf("users: %v", users)
	
	// Example 12: Pagination
	limit := 10
	offset := 20
	query10 := dbhelper.Q()

	paginationConfig := dbhelper.NewPaginationConfig().
	WithLimit(ptrInt(limit)).
	WithOffset(ptrInt(offset)).
	AppendOrder(dbhelper.NewOrderConfig(fm.ID, true))
	users, _ = executor.Find(ctx, query10, paginationConfig)
	t.Logf("users: %v", users)

	// Example 13: Raw SQL expression
	rawCond := dbhelper.Q(fm.Age.Gt(ptrInt(18)), fm.Status.Eq(ptrString(activeStatus)))
	query11 := dbhelper.Q(rawCond)
	users, _ = executor.Find(ctx, query11, nil)
	t.Logf("users: %v", users)
	
	// Example 14: Combine multiple conditions
	cond1 := fm.Age.Gt(ptrInt(18))
	cond2 := fm.Status.Eq(ptrString(activeStatus))
	cond3 := fm.Deleted.Eq(ptrBool(false))
	combinedCond := dbhelper.Q(cond1, cond2, cond3)
	query12 := dbhelper.Q(combinedCond)
	users, _ = executor.Find(ctx, query12, nil)
	t.Logf("users: %v", users)

	// Example 15: Combine multiple queries
	// Expect: age > 18 or (status = 'active' and deleted = false and (name like '%test%' or name like '%test%'))
	q1 := dbhelper.Q(fm.Age.Gt(ptrInt(18)))
	q2 := dbhelper.Q(fm.Status.Eq(ptrString(activeStatus)))
	q3 := dbhelper.Q(fm.Deleted.Eq(ptrBool(false)), q2)
	q4 := dbhelper.Or(fm.Name.StartsWith(ptrString(searchTerm)), fm.Name.EndsWith(ptrString(searchTerm)))
	q5 := dbhelper.Q(q3, q4)

	combinedQ := dbhelper.Or(q1, q5)
	users, _ = executor.Find(ctx, combinedQ, nil)
	t.Logf("users: %v", users)
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
	executor := dbhelper.NewExecutor[User](db)
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
	_ = executor.Update(ctx, query, updater)
}

