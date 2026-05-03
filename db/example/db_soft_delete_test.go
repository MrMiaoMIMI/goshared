package example

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbhelper"
)

func Test_SoftDeleteTableStore_SoftDeleteAndRestore(t *testing.T) {
	ctx := context.Background()
	store := dbhelper.NewSoftDeleteTableStore(&User{}, dbhelper.WithManager(testManager(testDatabaseName)))
	targetID := 34

	requireNoError(t, store.RestoreById(ctx, targetID))

	exists, user, err := store.ExistsByIdNotDeleted(ctx, targetID)
	requireNoError(t, err)
	if !exists || user == nil || user.Deleted {
		t.Fatalf("before soft delete: exists=%v, user=%+v", exists, user)
	}

	requireNoError(t, store.SoftDeleteById(ctx, targetID))

	exists, user, err = store.ExistsByIdNotDeleted(ctx, targetID)
	requireNoError(t, err)
	if exists || user != nil {
		t.Fatalf("after soft delete, without-deleted query should not return row: exists=%v, user=%+v", exists, user)
	}

	exists, user, err = store.ExistsById(ctx, targetID)
	requireNoError(t, err)
	if !exists || user == nil || !user.Deleted {
		t.Fatalf("normal query should return soft-deleted row: exists=%v, user=%+v", exists, user)
	}

	requireNoError(t, store.RestoreById(ctx, targetID))

	exists, user, err = store.ExistsByIdNotDeleted(ctx, targetID)
	requireNoError(t, err)
	if !exists || user == nil || user.Deleted {
		t.Fatalf("after recover: exists=%v, user=%+v", exists, user)
	}
}
