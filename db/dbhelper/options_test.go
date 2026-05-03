package dbhelper

import (
	"context"
	"strings"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

type customManager struct{}

func (customManager) ManagerHandle() {}

type optionEntity struct{}

func (*optionEntity) TableName() string { return "option_tab" }

func TestCommonFieldAutoFillOptionsCanConfigureManagerAndTableStore(t *testing.T) {
	opt := WithCommonFieldAutoFill(false)

	managerOptions := resolveManagerOptions([]ManagerOption{opt})
	tableStoreOptions := resolveTableStoreOptions([]TableStoreOption{opt})

	if managerOptions.commonFields.apply(dbsp.DefaultCommonFieldAutoFillOptions()).AutoFillEnabled {
		t.Fatal("manager common fields should be disabled")
	}
	if tableStoreOptions.commonFields.apply(dbsp.DefaultCommonFieldAutoFillOptions()).AutoFillEnabled {
		t.Fatal("table store common fields should be disabled")
	}
}

func TestTableStoreCommonFieldAutoFillOptionsOverlayManagerDefaults(t *testing.T) {
	managerCommonFields := dbsp.DefaultCommonFieldAutoFillOptions()
	managerCommonFields.TimeProvider = func(context.Context) uint64 { return 1 }
	managerCommonFields.OperatorProvider = func(context.Context) (string, bool) {
		return "manager", true
	}

	tableStoreOptions := resolveTableStoreOptions([]TableStoreOption{
		WithCommonFieldTimeProvider(func(context.Context) uint64 { return 2 }),
		WithCommonFieldOperatorProvider(func(context.Context) (string, bool) {
			return "table_store", true
		}),
	})
	commonFields := tableStoreOptions.commonFields.apply(managerCommonFields)

	if got := commonFields.TimeProvider(context.Background()); got != 2 {
		t.Fatalf("time provider = %d, want 2", got)
	}
	if got, ok := commonFields.OperatorProvider(context.Background()); !ok || got != "table_store" {
		t.Fatalf("operator provider = %q, %v; want table_store, true", got, ok)
	}
}

func TestCommonFieldOverwriteExplicitValuesOptionCanSetFalse(t *testing.T) {
	managerCommonFields := dbsp.DefaultCommonFieldAutoFillOptions()
	managerCommonFields.OverwriteExplicitValues = true

	tableStoreOptions := resolveTableStoreOptions([]TableStoreOption{
		WithCommonFieldOverwriteExplicitValues(false),
	})
	commonFields := tableStoreOptions.commonFields.apply(managerCommonFields)

	if commonFields.OverwriteExplicitValues {
		t.Fatal("overwrite explicit values should be disabled by NewTableStore option")
	}
}

func TestUnsupportedManagerImplementationPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if !strings.Contains(r.(string), "use dbhelper.NewManager") {
			t.Fatalf("panic = %q, want NewManager hint", r)
		}
	}()

	_ = asInternalManager(customManager{})
}

func TestWithTxOptionTakesPrecedenceOverManager(t *testing.T) {
	store := NewTableStore(&optionEntity{}, WithManager(customManager{}), WithTx(nil))

	err := store.Create(context.Background(), &optionEntity{})
	if err == nil || !strings.Contains(err.Error(), "transaction is nil") {
		t.Fatalf("Create error = %v, want transaction nil error", err)
	}
}

func TestNewSoftDeleteTableStoreWithNilTxReturnsMethodError(t *testing.T) {
	store := NewSoftDeleteTableStore(&optionEntity{}, WithTx(nil))

	_, err := store.CountNotDeleted(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "transaction is nil") {
		t.Fatalf("CountNotDeleted error = %v, want transaction nil error", err)
	}
}
