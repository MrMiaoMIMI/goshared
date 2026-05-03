package dbsp

import (
	"context"
	"testing"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

type commonFieldTestEntity struct {
	dbspi.CommonFields
	Name string `gorm:"column:name"`
}

type commonFieldTimeOnlyTestEntity struct {
	dbspi.TimeFields
	Name string `gorm:"column:name"`
}

func (*commonFieldTestEntity) TableName() string {
	return "common_field_test_tab"
}

func testCommonFieldAutoFillOptions() dbspi.CommonFieldAutoFillOptions {
	return dbspi.CommonFieldAutoFillOptions{
		AutoFillEnabled:  true,
		TimeProvider:     func() uint64 { return 12345 },
		OperatorProvider: dbspi.OperatorFromContext,
	}
}

func TestDefaultCommonFieldAutoFillOptionsEnableAutofill(t *testing.T) {
	opts := dbspi.DefaultCommonFieldAutoFillOptions()
	opts.TimeProvider = func() uint64 { return 12345 }
	entity := &commonFieldTimeOnlyTestEntity{}

	applyCreateCommonFields(context.Background(), opts, entity)

	if entity.Ctime != 12345 || entity.Mtime != 12345 {
		t.Fatalf("default common field options should enable autofill: %+v", entity.TimeFields)
	}
}

func TestDisabledCommonFieldAutoFillOptionsSkipAutofill(t *testing.T) {
	entity := &commonFieldTimeOnlyTestEntity{}

	applyCreateCommonFields(context.Background(), dbspi.DisabledCommonFieldAutoFillOptions(), entity)

	if entity.Ctime != 0 || entity.Mtime != 0 {
		t.Fatalf("disabled common field options should skip autofill: %+v", entity.TimeFields)
	}
}

func TestApplyCreateCommonFields(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "creator_a")
	entity := &commonFieldTestEntity{}

	applyCreateCommonFields(ctx, testCommonFieldAutoFillOptions(), entity)

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

func TestCommonFieldsManagedFields(t *testing.T) {
	entity := &commonFieldTestEntity{}
	entity.SetId(9)
	entity.SetDeleted(true)

	if entity.Id != 9 || entity.GetId() != 9 || entity.IdFieldName() != dbspi.DefaultIdFieldName {
		t.Fatalf("id managed fields not synced: %+v", entity.CommonFields)
	}
	if !entity.Deleted || !entity.GetDeleted() || entity.SoftDeleteFieldName() != dbspi.DefaultDeletedFieldName {
		t.Fatalf("deleted managed fields not synced: %+v", entity.CommonFields)
	}
}

func TestApplyCreateCommonFieldsDoesNotOverwriteExplicitValuesByDefault(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "creator_a")
	entity := &commonFieldTestEntity{
		CommonFields: dbspi.CommonFields{
			CreateTimeField: dbspi.CreateTimeField{
				Ctime: 1,
			},
			UpdateTimeField: dbspi.UpdateTimeField{
				Mtime: 2,
			},
			CreatorField: dbspi.CreatorField{
				Creator: "creator_existing",
			},
			UpdaterField: dbspi.UpdaterField{
				Updater: "updater_existing",
			},
		},
	}

	applyCreateCommonFields(ctx, testCommonFieldAutoFillOptions(), entity)

	if entity.Ctime != 1 || entity.Mtime != 2 || entity.Creator != "creator_existing" || entity.Updater != "updater_existing" {
		t.Fatalf("create common fields should not overwrite explicit values: %+v", entity.CommonFields)
	}
}

func TestApplyCreateCommonFieldsCanOverwriteExplicitValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "creator_a")
	opts := testCommonFieldAutoFillOptions()
	opts.OverwriteExplicitValues = true
	entity := &commonFieldTestEntity{
		CommonFields: dbspi.CommonFields{
			CreateTimeField: dbspi.CreateTimeField{
				Ctime: 1,
			},
			UpdateTimeField: dbspi.UpdateTimeField{
				Mtime: 2,
			},
			CreatorField: dbspi.CreatorField{
				Creator: "creator_existing",
			},
			UpdaterField: dbspi.UpdaterField{
				Updater: "updater_existing",
			},
		},
	}

	applyCreateCommonFields(ctx, opts, entity)

	if entity.Ctime != 12345 || entity.Mtime != 12345 || entity.Creator != "creator_a" || entity.Updater != "creator_a" {
		t.Fatalf("create common fields should overwrite explicit values when configured: %+v", entity.CommonFields)
	}
}

func TestApplySaveCommonFieldsDoesNotOverwriteExplicitValuesByDefault(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	entity := &commonFieldTestEntity{
		CommonFields: dbspi.CommonFields{
			CreateTimeField: dbspi.CreateTimeField{
				Ctime: 1,
			},
			UpdateTimeField: dbspi.UpdateTimeField{
				Mtime: 2,
			},
			CreatorField: dbspi.CreatorField{
				Creator: "creator_existing",
			},
			UpdaterField: dbspi.UpdaterField{
				Updater: "updater_existing",
			},
		},
	}

	applySaveCommonFields(ctx, testCommonFieldAutoFillOptions(), entity)

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

	applySaveCommonFields(ctx, testCommonFieldAutoFillOptions(), entity)

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
	opts := testCommonFieldAutoFillOptions()
	opts.OverwriteExplicitValues = true
	entity := &commonFieldTestEntity{
		CommonFields: dbspi.CommonFields{
			CreateTimeField: dbspi.CreateTimeField{
				Ctime: 1,
			},
			UpdateTimeField: dbspi.UpdateTimeField{
				Mtime: 2,
			},
			CreatorField: dbspi.CreatorField{
				Creator: "creator_existing",
			},
			UpdaterField: dbspi.UpdaterField{
				Updater: "updater_existing",
			},
		},
	}

	applySaveCommonFields(ctx, opts, entity)

	if entity.Ctime != 12345 || entity.Mtime != 12345 || entity.Creator != "updater_a" || entity.Updater != "updater_a" {
		t.Fatalf("save common fields should overwrite explicit values when configured: %+v", entity.CommonFields)
	}
}

func TestApplyUpdateCommonFieldsDoesNotOverwriteExplicitValuesByDefault(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	entity := &commonFieldTestEntity{
		CommonFields: dbspi.CommonFields{
			UpdateTimeField: dbspi.UpdateTimeField{
				Mtime: 2,
			},
			UpdaterField: dbspi.UpdaterField{
				Updater: "updater_existing",
			},
		},
	}

	applyUpdateCommonFields(ctx, testCommonFieldAutoFillOptions(), entity)

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

	applyUpdateCommonFields(ctx, testCommonFieldAutoFillOptions(), entity)

	if entity.Mtime != 12345 {
		t.Fatalf("Mtime = %d, want 12345", entity.Mtime)
	}
	if entity.Updater != "updater_a" {
		t.Fatalf("Updater = %q, want updater_a", entity.Updater)
	}
}

func TestApplyUpdateCommonFieldsCanOverwriteExplicitValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	opts := testCommonFieldAutoFillOptions()
	opts.OverwriteExplicitValues = true
	entity := &commonFieldTestEntity{
		CommonFields: dbspi.CommonFields{
			UpdateTimeField: dbspi.UpdateTimeField{
				Mtime: 2,
			},
			UpdaterField: dbspi.UpdaterField{
				Updater: "updater_existing",
			},
		},
	}

	applyUpdateCommonFields(ctx, opts, entity)

	if entity.Mtime != 12345 || entity.Updater != "updater_a" {
		t.Fatalf("update common fields should overwrite explicit values when configured: %+v", entity.CommonFields)
	}
}

func TestApplyCommonFieldsSupportsTimeFieldsOnly(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "operator_a")
	entity := &commonFieldTimeOnlyTestEntity{}

	applyCreateCommonFields(ctx, testCommonFieldAutoFillOptions(), entity)

	if entity.Ctime != 12345 {
		t.Fatalf("Ctime = %d, want 12345", entity.Ctime)
	}
	if entity.Mtime != 12345 {
		t.Fatalf("Mtime = %d, want 12345", entity.Mtime)
	}

	updater := NewUpdater().Set(NewField[string]("name"), "new name")
	applyUpdateCommonFieldsToUpdater(ctx, testCommonFieldAutoFillOptions(), entity, updater)
	params := updater.Values()
	if params[dbspi.DefaultMtimeFieldName] != uint64(12345) {
		t.Fatalf("mtime = %v, want 12345", params[dbspi.DefaultMtimeFieldName])
	}
	if _, ok := params[dbspi.DefaultUpdaterFieldName]; ok {
		t.Fatalf("updater should not be added for TimeFields-only entity: %v", params)
	}
}

func TestApplyUpdateCommonFieldsToUpdaterKeepsExplicitValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	model := &commonFieldTestEntity{}
	updater := NewUpdater().
		Set(NewField[uint64](dbspi.DefaultMtimeFieldName), uint64(9)).
		Set(NewField[string](dbspi.DefaultUpdaterFieldName), "explicit")

	applyUpdateCommonFieldsToUpdater(ctx, testCommonFieldAutoFillOptions(), model, updater)

	params := updater.Values()
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
	updater := NewUpdater().Set(NewField[string]("name"), "new name")

	applyUpdateCommonFieldsToUpdater(ctx, testCommonFieldAutoFillOptions(), model, updater)

	params := updater.Values()
	if params[dbspi.DefaultMtimeFieldName] != uint64(12345) {
		t.Fatalf("mtime = %v, want 12345", params[dbspi.DefaultMtimeFieldName])
	}
	if params[dbspi.DefaultUpdaterFieldName] != "updater_a" {
		t.Fatalf("updater = %v, want updater_a", params[dbspi.DefaultUpdaterFieldName])
	}
}

func TestApplyUpdateCommonFieldsToUpdaterCanOverwriteExplicitValues(t *testing.T) {
	ctx := dbspi.WithOperator(context.Background(), "updater_a")
	opts := testCommonFieldAutoFillOptions()
	opts.OverwriteExplicitValues = true
	model := &commonFieldTestEntity{}
	updater := NewUpdater().
		Set(NewField[uint64](dbspi.DefaultMtimeFieldName), uint64(9)).
		Set(NewField[string](dbspi.DefaultUpdaterFieldName), "explicit")

	applyUpdateCommonFieldsToUpdater(ctx, opts, model, updater)

	params := updater.Values()
	if params[dbspi.DefaultMtimeFieldName] != uint64(12345) {
		t.Fatalf("mtime = %v, want 12345", params[dbspi.DefaultMtimeFieldName])
	}
	if params[dbspi.DefaultUpdaterFieldName] != "updater_a" {
		t.Fatalf("updater = %v, want updater_a", params[dbspi.DefaultUpdaterFieldName])
	}
}
