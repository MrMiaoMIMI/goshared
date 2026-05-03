package example

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
)

func Test_EnhancedExecutor_SoftDeleteAndRecover(t *testing.T) {
	ctx := context.Background()
	executor := dbhelper.NewEnhancedExecutor(&User{}, dbhelper.WithManager(testManager(testDatabaseName)))
	targetID := 34

	requireNoError(t, executor.RestoreById(ctx, targetID))

	exists, user, err := executor.ExistsByIdNotDeleted(ctx, targetID)
	requireNoError(t, err)
	if !exists || user == nil || user.Deleted {
		t.Fatalf("before soft delete: exists=%v, user=%+v", exists, user)
	}

	requireNoError(t, executor.SoftDeleteById(ctx, targetID))

	exists, user, err = executor.ExistsByIdNotDeleted(ctx, targetID)
	requireNoError(t, err)
	if exists || user != nil {
		t.Fatalf("after soft delete, without-deleted query should not return row: exists=%v, user=%+v", exists, user)
	}

	exists, user, err = executor.ExistsById(ctx, targetID)
	requireNoError(t, err)
	if !exists || user == nil || !user.Deleted {
		t.Fatalf("normal query should return soft-deleted row: exists=%v, user=%+v", exists, user)
	}

	requireNoError(t, executor.RestoreById(ctx, targetID))

	exists, user, err = executor.ExistsByIdNotDeleted(ctx, targetID)
	requireNoError(t, err)
	if !exists || user == nil || user.Deleted {
		t.Fatalf("after recover: exists=%v, user=%+v", exists, user)
	}
}
