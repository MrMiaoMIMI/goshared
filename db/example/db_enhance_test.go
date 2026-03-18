package example

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
)

func Test_EnhancedExecutor_SoftDelete(t *testing.T) {
	ctx := context.Background()
	db := testNewDb()
	fm := NewUserFieldManager()
	executor := dbhelper.NewEnhancedExecutor(db, &User{})

	// Example 1: SoftDeleteById
	err := executor.SoftDeleteById(ctx, 13)
	t.Logf("Example 1: SoftDeleteById: err: %v", err)

	// Example 2: SoftDeleteByQuery
	email := "alice@example.com"
	query := dbhelper.Q(fm.Email.Eq(ptrString(email)))
	err = executor.SoftDeleteByQuery(ctx, query)
	t.Logf("Example 2: SoftDeleteByQuery: err: %v", err)
}

func Test_EnhancedExecutor_RecoverFromDeleted(t *testing.T) {
	ctx := context.Background()
	db := testNewDb()
	fm := NewUserFieldManager()
	executor := dbhelper.NewEnhancedExecutor(db, &User{})

	// Example 3: RecoverFromDeletedById
	err := executor.RecoverFromDeletedById(ctx, 13)
	t.Logf("Example 3: RecoverFromDeletedById: err: %v", err)

	// Example 4: RecoverFromDeletedByQuery
	email := "alice@example.com"
	query := dbhelper.Q(fm.Email.Eq(ptrString(email)))
	err = executor.RecoverFromDeletedByQuery(ctx, query)
	t.Logf("Example 4: RecoverFromDeletedByQuery: err: %v", err)
}

func Test_EnhancedExecutor_FindWithoutDeleted(t *testing.T) {
	ctx := context.Background()
	db := testNewDb()
	fm := NewUserFieldManager()
	executor := dbhelper.NewEnhancedExecutor(db, &User{})

	// Example 5: FindWithoutDeleted - all non-deleted records
	users, err := executor.FindWithoutDeleted(ctx, nil, nil)
	t.Logf("Example 5: FindWithoutDeleted (all): users: %v, err: %v", users, err)

	// Example 6: FindWithoutDeleted - with query conditions
	activeStatus := "active"
	query := dbhelper.Q(fm.Status.Eq(ptrString(activeStatus)))
	users, err = executor.FindWithoutDeleted(ctx, query, nil)
	t.Logf("Example 6: FindWithoutDeleted (with query): users: %v, err: %v", users, err)

	// Example 7: FindWithoutDeleted - with pagination
	limit := 10
	offset := 0
	paginationConfig := dbhelper.NewPaginationConfig().
		WithLimit(ptrInt(limit)).
		WithOffset(ptrInt(offset)).
		AppendOrder(dbhelper.NewOrderConfig(fm.ID, true))
	users, err = executor.FindWithoutDeleted(ctx, nil, paginationConfig)
	t.Logf("Example 7: FindWithoutDeleted (with pagination): users: %v, err: %v", users, err)

	// Example 8: FindWithoutDeleted - complex query with pagination
	searchTerm := "test"
	complexQuery := dbhelper.Q(
		dbhelper.Or(
			fm.Name.Contains(ptrString(searchTerm)),
			fm.Email.Contains(ptrString(searchTerm)),
		),
		fm.Age.Gt(ptrInt(18)),
	)
	users, err = executor.FindWithoutDeleted(ctx, complexQuery, paginationConfig)
	t.Logf("Example 8: FindWithoutDeleted (complex query + pagination): users: %v, err: %v", users, err)
}

func Test_EnhancedExecutor_CountWithoutDeleted(t *testing.T) {
	ctx := context.Background()
	db := testNewDb()
	fm := NewUserFieldManager()
	executor := dbhelper.NewEnhancedExecutor(db, &User{})

	// Example 9: CountWithoutDeleted - all non-deleted records
	count, err := executor.CountWithoutDeleted(ctx, nil)
	t.Logf("Example 9: CountWithoutDeleted (all): count: %d, err: %v", count, err)

	// Example 10: CountWithoutDeleted - with query conditions
	activeStatus := "active"
	query := dbhelper.Q(fm.Status.Eq(ptrString(activeStatus)))
	count, err = executor.CountWithoutDeleted(ctx, query)
	t.Logf("Example 10: CountWithoutDeleted (with query): count: %d, err: %v", count, err)
}

func Test_EnhancedExecutor_ExistsWithoutDeleted(t *testing.T) {
	ctx := context.Background()
	db := testNewDb()
	fm := NewUserFieldManager()
	executor := dbhelper.NewEnhancedExecutor(db, &User{})

	// Example 11: ExistsByIdWithoutDeleted
	exists, user, err := executor.ExistsByIdWithoutDeleted(ctx, 14)
	t.Logf("Example 11: ExistsByIdWithoutDeleted: exists: %v, user: %v, err: %v", exists, user, err)

	// Example 12: ExistsWithoutDeleted - with query
	email := "alice@example.com"
	query := dbhelper.Q(fm.Email.NotEq(ptrString(email)))
	exists, user, err = executor.ExistsWithoutDeleted(ctx, query)
	t.Logf("Example 12: ExistsWithoutDeleted (with query): exists: %v, user: %v, err: %v", exists, user, err)
}

func Test_EnhancedExecutor_SoftDeleteAndRecover(t *testing.T) {
	ctx := context.Background()
	db := testNewDb()
	executor := dbhelper.NewEnhancedExecutor(db, &User{})

	targetID := 34

	// Step 1: Verify the record exists and is not deleted
	exists, user, err := executor.ExistsByIdWithoutDeleted(ctx, targetID)
	t.Logf("Step 1: Before soft delete: exists: %v, user: %v, err: %v", exists, user, err)

	// Step 2: Soft delete the record
	err = executor.SoftDeleteById(ctx, targetID)
	t.Logf("Step 2: SoftDeleteById: err: %v", err)

	// Step 3: Verify the record is no longer found via WithoutDeleted queries
	exists, user, err = executor.ExistsByIdWithoutDeleted(ctx, targetID)
	t.Logf("Step 3: After soft delete: exists: %v, user: %v, err: %v", exists, user, err)

	// Step 4: But the record still exists in normal queries
	exists, user, err = executor.ExistsById(ctx, targetID)
	t.Logf("Step 4: Normal ExistsById after soft delete: exists: %v, user: %v, err: %v", exists, user, err)

	// Step 5: Recover the record
	err = executor.RecoverFromDeletedById(ctx, targetID)
	t.Logf("Step 5: RecoverFromDeletedById: err: %v", err)

	// Step 6: Verify the record is back
	exists, user, err = executor.ExistsByIdWithoutDeleted(ctx, targetID)
	t.Logf("Step 6: After recover: exists: %v, user: %v, err: %v", exists, user, err)
}
