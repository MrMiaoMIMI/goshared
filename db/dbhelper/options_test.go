package dbhelper

import (
	"context"
	"strings"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

type customManager struct{}

func (customManager) ManagerHandle() {}

type optionEntity struct{}

func (*optionEntity) TableName() string { return "option_tab" }

func TestCommonFieldAutoFillOptionsCanConfigureManagerAndExecutor(t *testing.T) {
	opt := WithCommonFieldAutoFill(false)

	managerOptions := resolveManagerOptions([]ManagerOption{opt})
	executorOptions := resolveExecutorOptions([]ExecutorOption{opt})

	if managerOptions.commonFields.apply(dbspi.DefaultCommonFieldAutoFillOptions()).AutoFillEnabled {
		t.Fatal("manager common fields should be disabled")
	}
	if executorOptions.commonFields.apply(dbspi.DefaultCommonFieldAutoFillOptions()).AutoFillEnabled {
		t.Fatal("executor common fields should be disabled")
	}
}

func TestExecutorCommonFieldAutoFillOptionsOverlayManagerDefaults(t *testing.T) {
	managerCommonFields := dbspi.DefaultCommonFieldAutoFillOptions()
	managerCommonFields.TimeProvider = func() uint64 { return 1 }
	managerCommonFields.OperatorProvider = func(context.Context) (string, bool) {
		return "manager", true
	}

	executorOptions := resolveExecutorOptions([]ExecutorOption{
		WithCommonFieldTimeProvider(func() uint64 { return 2 }),
		WithCommonFieldOperatorProvider(func(context.Context) (string, bool) {
			return "executor", true
		}),
	})
	commonFields := executorOptions.commonFields.apply(managerCommonFields)

	if got := commonFields.TimeProvider(); got != 2 {
		t.Fatalf("time provider = %d, want 2", got)
	}
	if got, ok := commonFields.OperatorProvider(context.Background()); !ok || got != "executor" {
		t.Fatalf("operator provider = %q, %v; want executor, true", got, ok)
	}
}

func TestCommonFieldOverwriteExplicitValuesOptionCanSetFalse(t *testing.T) {
	managerCommonFields := dbspi.DefaultCommonFieldAutoFillOptions()
	managerCommonFields.OverwriteExplicitValues = true

	executorOptions := resolveExecutorOptions([]ExecutorOption{
		WithCommonFieldOverwriteExplicitValues(false),
	})
	commonFields := executorOptions.commonFields.apply(managerCommonFields)

	if commonFields.OverwriteExplicitValues {
		t.Fatal("overwrite explicit values should be disabled by NewExecutor option")
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
	exec := NewExecutor(&optionEntity{}, WithManager(customManager{}), WithTx(nil))

	err := exec.Create(context.Background(), &optionEntity{})
	if err == nil || !strings.Contains(err.Error(), "transaction is nil") {
		t.Fatalf("Create error = %v, want transaction nil error", err)
	}
}

func TestNewEnhancedExecutorWithNilTxReturnsMethodError(t *testing.T) {
	exec := NewEnhancedExecutor(&optionEntity{}, WithTx(nil))

	_, err := exec.CountNotDeleted(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "transaction is nil") {
		t.Fatalf("CountNotDeleted error = %v, want transaction nil error", err)
	}
}
