package dbsp

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

type commonFieldTestEntity struct {
	dbspi.CommonDo
	Name string `gorm:"column:name"`
}

type commonFieldTimeOnlyTestEntity struct {
	dbspi.TimeDo
	Name string `gorm:"column:name"`
}

func (*commonFieldTestEntity) TableName() string {
	return "common_field_test_tab"
}

func testCommonFieldOptions() dbspi.CommonFieldOptions {
	return dbspi.CommonFieldOptions{
		Enabled:          true,
		TimeProvider:     func() uint64 { return 12345 },
		OperatorProvider: dbspi.OperatorFromContext,
	}
}

func TestDefaultCommonFieldOptionsEnableAutofill(t *testing.T) {
	opts := dbspi.DefaultCommonFieldOptions()
	opts.TimeProvider = func() uint64 { return 12345 }
	entity := &commonFieldTimeOnlyTestEntity{}

	applyCreateCommonFields(context.Background(), opts, entity)

	if entity.Ctime != 12345 || entity.Mtime != 12345 {
		t.Fatalf("default common field options should enable autofill: %+v", entity.TimeDo)
	}
}

func TestDisabledCommonFieldOptionsSkipAutofill(t *testing.T) {
	entity := &commonFieldTimeOnlyTestEntity{}

	applyCreateCommonFields(context.Background(), dbspi.DisabledCommonFieldOptions(), entity)

	if entity.Ctime != 0 || entity.Mtime != 0 {
		t.Fatalf("disabled common field options should skip autofill: %+v", entity.TimeDo)
	}
}

func TestApplyCreateCommonFields(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "creator_a")
	entity := &commonFieldTestEntity{}

	applyCreateCommonFields(ctx, testCommonFieldOptions(), entity)

	if entity.Ctime != 12345 {
		t.Fatalf("Ctime = %d, want 12345", entity.Ctime)
	}
	if entity.Mtime != 12345 {
		t.Fatalf("Mtime = %d, want 12345", entity.Mtime)
	}
	if entity.Creator != "creator_a" {
		t.Fatalf("Creator = %q, want creator_a", entity.Creator)
	}
	if entity.Updater != "creator_a" {
		t.Fatalf("Updater = %q, want creator_a", entity.Updater)
	}
}

func TestCommonDoManagedFields(t *testing.T) {
	entity := &commonFieldTestEntity{}
	entity.SetId(9)
	entity.SetDeleted(true)

	if entity.Id != 9 || entity.GetId() != 9 || entity.IdFieldName() != dbspi.DefaultIdFieldName {
		t.Fatalf("id managed fields not synced: %+v", entity.CommonDo)
	}
	if !entity.Deleted || !entity.GetDeleted() || entity.DeletedFieldName() != dbspi.DefaultDeletedFieldName {
		t.Fatalf("deleted managed fields not synced: %+v", entity.CommonDo)
	}
}

func TestApplyCreateCommonFieldsDoesNotOverwriteExplicitValuesByDefault(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "creator_a")
	entity := &commonFieldTestEntity{
		CommonDo: dbspi.CommonDo{
			TimeDo: dbspi.TimeDo{
				Ctime: 1,
				Mtime: 2,
			},
			OperatorDo: dbspi.OperatorDo{
				Creator: "creator_existing",
				Updater: "updater_existing",
			},
		},
	}

	applyCreateCommonFields(ctx, testCommonFieldOptions(), entity)

	if entity.Ctime != 1 || entity.Mtime != 2 || entity.Creator != "creator_existing" || entity.Updater != "updater_existing" {
		t.Fatalf("create common fields should not overwrite explicit values: %+v", entity.CommonDo)
	}
}

func TestApplyCreateCommonFieldsCanOverwriteExplicitValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "creator_a")
	opts := testCommonFieldOptions()
	opts.OverwriteExplicitValues = true
	entity := &commonFieldTestEntity{
		CommonDo: dbspi.CommonDo{
			TimeDo: dbspi.TimeDo{
				Ctime: 1,
				Mtime: 2,
			},
			OperatorDo: dbspi.OperatorDo{
				Creator: "creator_existing",
				Updater: "updater_existing",
			},
		},
	}

	applyCreateCommonFields(ctx, opts, entity)

	if entity.Ctime != 12345 || entity.Mtime != 12345 || entity.Creator != "creator_a" || entity.Updater != "creator_a" {
		t.Fatalf("create common fields should overwrite explicit values when configured: %+v", entity.CommonDo)
	}
}

func TestApplySaveCommonFieldsDoesNotOverwriteExplicitValuesByDefault(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	entity := &commonFieldTestEntity{
		CommonDo: dbspi.CommonDo{
			TimeDo: dbspi.TimeDo{
				Ctime: 1,
				Mtime: 2,
			},
			OperatorDo: dbspi.OperatorDo{
				Creator: "creator_existing",
				Updater: "updater_existing",
			},
		},
	}

	applySaveCommonFields(ctx, testCommonFieldOptions(), entity)

	if entity.Ctime != 1 {
		t.Fatalf("Ctime = %d, want unchanged 1", entity.Ctime)
	}
	if entity.Mtime != 2 {
		t.Fatalf("Mtime = %d, want unchanged 2", entity.Mtime)
	}
	if entity.Creator != "creator_existing" {
		t.Fatalf("Creator = %q, want creator_existing", entity.Creator)
	}
	if entity.Updater != "updater_existing" {
		t.Fatalf("Updater = %q, want updater_existing", entity.Updater)
	}
}

func TestApplySaveCommonFieldsFillsMissingMutableFields(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	entity := &commonFieldTestEntity{}

	applySaveCommonFields(ctx, testCommonFieldOptions(), entity)

	if entity.Ctime != 12345 {
		t.Fatalf("Ctime = %d, want 12345", entity.Ctime)
	}
	if entity.Mtime != 12345 {
		t.Fatalf("Mtime = %d, want 12345", entity.Mtime)
	}
	if entity.Creator != "updater_a" {
		t.Fatalf("Creator = %q, want updater_a", entity.Creator)
	}
	if entity.Updater != "updater_a" {
		t.Fatalf("Updater = %q, want updater_a", entity.Updater)
	}
}

func TestApplySaveCommonFieldsCanOverwriteExplicitValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	opts := testCommonFieldOptions()
	opts.OverwriteExplicitValues = true
	entity := &commonFieldTestEntity{
		CommonDo: dbspi.CommonDo{
			TimeDo: dbspi.TimeDo{
				Ctime: 1,
				Mtime: 2,
			},
			OperatorDo: dbspi.OperatorDo{
				Creator: "creator_existing",
				Updater: "updater_existing",
			},
		},
	}

	applySaveCommonFields(ctx, opts, entity)

	if entity.Ctime != 12345 || entity.Mtime != 12345 || entity.Creator != "updater_a" || entity.Updater != "updater_a" {
		t.Fatalf("save common fields should overwrite explicit values when configured: %+v", entity.CommonDo)
	}
}

func TestApplyUpdateCommonFieldsDoesNotOverwriteExplicitValuesByDefault(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	entity := &commonFieldTestEntity{
		CommonDo: dbspi.CommonDo{
			TimeDo: dbspi.TimeDo{
				Mtime: 2,
			},
			OperatorDo: dbspi.OperatorDo{
				Updater: "updater_existing",
			},
		},
	}

	applyUpdateCommonFields(ctx, testCommonFieldOptions(), entity)

	if entity.Mtime != 2 {
		t.Fatalf("Mtime = %d, want unchanged 2", entity.Mtime)
	}
	if entity.Updater != "updater_existing" {
		t.Fatalf("Updater = %q, want updater_existing", entity.Updater)
	}
}

func TestApplyUpdateCommonFieldsFillsMissingMutableFields(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	entity := &commonFieldTestEntity{}

	applyUpdateCommonFields(ctx, testCommonFieldOptions(), entity)

	if entity.Mtime != 12345 {
		t.Fatalf("Mtime = %d, want 12345", entity.Mtime)
	}
	if entity.Updater != "updater_a" {
		t.Fatalf("Updater = %q, want updater_a", entity.Updater)
	}
}

func TestApplyUpdateCommonFieldsCanOverwriteExplicitValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	opts := testCommonFieldOptions()
	opts.OverwriteExplicitValues = true
	entity := &commonFieldTestEntity{
		CommonDo: dbspi.CommonDo{
			TimeDo: dbspi.TimeDo{
				Mtime: 2,
			},
			OperatorDo: dbspi.OperatorDo{
				Updater: "updater_existing",
			},
		},
	}

	applyUpdateCommonFields(ctx, opts, entity)

	if entity.Mtime != 12345 || entity.Updater != "updater_a" {
		t.Fatalf("update common fields should overwrite explicit values when configured: %+v", entity.CommonDo)
	}
}

func TestApplyCommonFieldsSupportsTimeDoOnly(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "operator_a")
	entity := &commonFieldTimeOnlyTestEntity{}

	applyCreateCommonFields(ctx, testCommonFieldOptions(), entity)

	if entity.Ctime != 12345 {
		t.Fatalf("Ctime = %d, want 12345", entity.Ctime)
	}
	if entity.Mtime != 12345 {
		t.Fatalf("Mtime = %d, want 12345", entity.Mtime)
	}

	updater := NewUpdater().Add(NewField[string]("name"), "new name")
	applyUpdateCommonFieldsToUpdater(ctx, testCommonFieldOptions(), entity, updater)
	params := updater.Params()
	if params[dbspi.DefaultMtimeFieldName] != uint64(12345) {
		t.Fatalf("mtime = %v, want 12345", params[dbspi.DefaultMtimeFieldName])
	}
	if _, ok := params[dbspi.DefaultUpdaterFieldName]; ok {
		t.Fatalf("updater should not be added for TimeDo-only entity: %v", params)
	}
}

func TestApplyUpdateCommonFieldsToUpdaterKeepsExplicitValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	model := &commonFieldTestEntity{}
	updater := NewUpdater().
		Add(NewField[uint64](dbspi.DefaultMtimeFieldName), uint64(9)).
		Add(NewField[string](dbspi.DefaultUpdaterFieldName), "explicit")

	applyUpdateCommonFieldsToUpdater(ctx, testCommonFieldOptions(), model, updater)

	params := updater.Params()
	if params[dbspi.DefaultMtimeFieldName] != uint64(9) {
		t.Fatalf("mtime = %v, want explicit 9", params[dbspi.DefaultMtimeFieldName])
	}
	if params[dbspi.DefaultUpdaterFieldName] != "explicit" {
		t.Fatalf("updater = %v, want explicit", params[dbspi.DefaultUpdaterFieldName])
	}
}

func TestApplyUpdateCommonFieldsToUpdaterAddsMissingValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	model := &commonFieldTestEntity{}
	updater := NewUpdater().Add(NewField[string]("name"), "new name")

	applyUpdateCommonFieldsToUpdater(ctx, testCommonFieldOptions(), model, updater)

	params := updater.Params()
	if params[dbspi.DefaultMtimeFieldName] != uint64(12345) {
		t.Fatalf("mtime = %v, want 12345", params[dbspi.DefaultMtimeFieldName])
	}
	if params[dbspi.DefaultUpdaterFieldName] != "updater_a" {
		t.Fatalf("updater = %v, want updater_a", params[dbspi.DefaultUpdaterFieldName])
	}
}

func TestApplyUpdateCommonFieldsToUpdaterCanOverwriteExplicitValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	opts := testCommonFieldOptions()
	opts.OverwriteExplicitValues = true
	model := &commonFieldTestEntity{}
	updater := NewUpdater().
		Add(NewField[uint64](dbspi.DefaultMtimeFieldName), uint64(9)).
		Add(NewField[string](dbspi.DefaultUpdaterFieldName), "explicit")

	applyUpdateCommonFieldsToUpdater(ctx, opts, model, updater)

	params := updater.Params()
	if params[dbspi.DefaultMtimeFieldName] != uint64(12345) {
		t.Fatalf("mtime = %v, want 12345", params[dbspi.DefaultMtimeFieldName])
	}
	if params[dbspi.DefaultUpdaterFieldName] != "updater_a" {
		t.Fatalf("updater = %v, want updater_a", params[dbspi.DefaultUpdaterFieldName])
	}
}
