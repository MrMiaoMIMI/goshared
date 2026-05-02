package dbhelper

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

func TestCommonFieldOptionsCanConfigureManagerAndFor(t *testing.T) {
	opt := WithCommonFields(false)

	managerOptions := resolveManagerOptions([]ManagerOption{opt})
	forOptions := resolveForOptions([]ForOption{opt})

	if managerOptions.commonFields.apply(dbspi.DefaultCommonFieldOptions()).Enabled {
		t.Fatal("manager common fields should be disabled")
	}
	if forOptions.commonFields.apply(dbspi.DefaultCommonFieldOptions()).Enabled {
		t.Fatal("for common fields should be disabled")
	}
}

func TestForCommonFieldOptionsOverlayManagerDefaults(t *testing.T) {
	managerCommonFields := dbspi.DefaultCommonFieldOptions()
	managerCommonFields.TimeProvider = func() uint64 { return 1 }
	managerCommonFields.OperatorProvider = func(context.Context) (string, bool) {
		return "manager", true
	}

	forOptions := resolveForOptions([]ForOption{
		WithCommonFieldTimeProvider(func() uint64 { return 2 }),
		WithCommonFieldOperatorProvider(func(context.Context) (string, bool) {
			return "for", true
		}),
	})
	commonFields := forOptions.commonFields.apply(managerCommonFields)

	if got := commonFields.TimeProvider(); got != 2 {
		t.Fatalf("time provider = %d, want 2", got)
	}
	if got, ok := commonFields.OperatorProvider(context.Background()); !ok || got != "for" {
		t.Fatalf("operator provider = %q, %v; want for, true", got, ok)
	}
}

func TestCommonFieldOverwriteExplicitValuesOptionCanSetFalse(t *testing.T) {
	managerCommonFields := dbspi.DefaultCommonFieldOptions()
	managerCommonFields.OverwriteExplicitValues = true

	forOptions := resolveForOptions([]ForOption{
		WithCommonFieldOverwriteExplicitValues(false),
	})
	commonFields := forOptions.commonFields.apply(managerCommonFields)

	if commonFields.OverwriteExplicitValues {
		t.Fatal("overwrite explicit values should be disabled by For option")
	}
}
