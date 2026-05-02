package dbsp

import (
	"context"
	"reflect"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

func applyCreateCommonFields(ctx context.Context, opts dbspi.CommonFieldOptions, entity any) {
	if shouldSkipCommonFields(opts, entity) {
		return
	}
	opts = opts.Normalize()
	now := opts.TimeProvider()

	if managed, ok := entity.(dbspi.CreateTimeManaged); ok && (opts.OverwriteExplicitValues || managed.GetCtime() == 0) {
		managed.SetCtime(now)
	}
	if managed, ok := entity.(dbspi.UpdateTimeManaged); ok && (opts.OverwriteExplicitValues || managed.GetMtime() == 0) {
		managed.SetMtime(now)
	}

	operator, hasOperator := opts.OperatorProvider(ctx)
	if !hasOperator {
		return
	}
	if managed, ok := entity.(dbspi.CreatorManaged); ok && (opts.OverwriteExplicitValues || managed.GetCreator() == "") {
		managed.SetCreator(operator)
	}
	if managed, ok := entity.(dbspi.UpdaterManaged); ok && (opts.OverwriteExplicitValues || managed.GetUpdater() == "") {
		managed.SetUpdater(operator)
	}
}

func applySaveCommonFields(ctx context.Context, opts dbspi.CommonFieldOptions, entity any) {
	if shouldSkipCommonFields(opts, entity) {
		return
	}
	opts = opts.Normalize()
	now := opts.TimeProvider()

	if managed, ok := entity.(dbspi.CreateTimeManaged); ok && (opts.OverwriteExplicitValues || managed.GetCtime() == 0) {
		managed.SetCtime(now)
	}
	if managed, ok := entity.(dbspi.UpdateTimeManaged); ok && (opts.OverwriteExplicitValues || managed.GetMtime() == 0) {
		managed.SetMtime(now)
	}

	operator, hasOperator := opts.OperatorProvider(ctx)
	if !hasOperator {
		return
	}
	if managed, ok := entity.(dbspi.CreatorManaged); ok && (opts.OverwriteExplicitValues || managed.GetCreator() == "") {
		managed.SetCreator(operator)
	}
	if managed, ok := entity.(dbspi.UpdaterManaged); ok && (opts.OverwriteExplicitValues || managed.GetUpdater() == "") {
		managed.SetUpdater(operator)
	}
}

func applyUpdateCommonFields(ctx context.Context, opts dbspi.CommonFieldOptions, entity any) {
	if shouldSkipCommonFields(opts, entity) {
		return
	}
	opts = opts.Normalize()

	if managed, ok := entity.(dbspi.UpdateTimeManaged); ok && (opts.OverwriteExplicitValues || managed.GetMtime() == 0) {
		managed.SetMtime(opts.TimeProvider())
	}
	if managed, ok := entity.(dbspi.UpdaterManaged); ok {
		if operator, ok := opts.OperatorProvider(ctx); ok && (opts.OverwriteExplicitValues || managed.GetUpdater() == "") {
			managed.SetUpdater(operator)
		}
	}
}

func applyCreateCommonFieldsToSlice[T any](ctx context.Context, opts dbspi.CommonFieldOptions, entities []T) {
	for _, entity := range entities {
		applyCreateCommonFields(ctx, opts, entity)
	}
}

func applySaveCommonFieldsToSlice[T any](ctx context.Context, opts dbspi.CommonFieldOptions, entities []T) {
	for _, entity := range entities {
		applySaveCommonFields(ctx, opts, entity)
	}
}

func applyUpdateCommonFieldsToUpdater(ctx context.Context, opts dbspi.CommonFieldOptions, model any, updater dbspi.Updater) {
	if shouldSkipCommonFields(opts, model) || updater == nil {
		return
	}
	opts = opts.Normalize()
	params := updater.Params()

	if managed, ok := model.(dbspi.UpdateTimeManaged); ok {
		fieldName := managed.MtimeFieldName()
		if opts.OverwriteExplicitValues || !hasUpdateParam(params, fieldName) {
			updater.Add(NewColumn(fieldName), opts.TimeProvider())
		}
	}

	operator, hasOperator := opts.OperatorProvider(ctx)
	if !hasOperator {
		return
	}
	if managed, ok := model.(dbspi.UpdaterManaged); ok {
		fieldName := managed.UpdaterFieldName()
		if opts.OverwriteExplicitValues || !hasUpdateParam(params, fieldName) {
			updater.Add(NewColumn(fieldName), operator)
		}
	}
}

func shouldSkipCommonFields(opts dbspi.CommonFieldOptions, entity any) bool {
	return !opts.Enabled || isNilEntity(entity)
}

func hasUpdateParam(params map[string]any, fieldName string) bool {
	_, ok := params[fieldName]
	return ok
}

func isNilEntity(entity any) bool {
	if entity == nil {
		return true
	}
	v := reflect.ValueOf(entity)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
