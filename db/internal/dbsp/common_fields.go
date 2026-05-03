package dbsp

import (
	"context"
	"reflect"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

func applyCreateCommonFields(ctx context.Context, opts dbspi.CommonFieldAutoFillOptions, entity any) {
	if shouldSkipCommonFields(opts, entity) {
		return
	}
	opts = opts.Normalize()
	now := opts.TimeProvider()

	if managed, ok := entity.(dbspi.CreateTimeAccessor); ok && (opts.OverwriteExplicitValues || managed.GetCtime() == 0) {
		managed.SetCtime(now)
	}
	if managed, ok := entity.(dbspi.UpdateTimeAccessor); ok && (opts.OverwriteExplicitValues || managed.GetMtime() == 0) {
		managed.SetMtime(now)
	}

	operator, hasOperator := opts.OperatorProvider(ctx)
	if !hasOperator {
		return
	}
	if managed, ok := entity.(dbspi.CreatorAccessor); ok && (opts.OverwriteExplicitValues || managed.GetCreator() == "") {
		managed.SetCreator(operator)
	}
	if managed, ok := entity.(dbspi.UpdaterAccessor); ok && (opts.OverwriteExplicitValues || managed.GetUpdater() == "") {
		managed.SetUpdater(operator)
	}
}

func applySaveCommonFields(ctx context.Context, opts dbspi.CommonFieldAutoFillOptions, entity any) {
	if shouldSkipCommonFields(opts, entity) {
		return
	}
	opts = opts.Normalize()
	now := opts.TimeProvider()

	if managed, ok := entity.(dbspi.CreateTimeAccessor); ok && (opts.OverwriteExplicitValues || managed.GetCtime() == 0) {
		managed.SetCtime(now)
	}
	if managed, ok := entity.(dbspi.UpdateTimeAccessor); ok && (opts.OverwriteExplicitValues || managed.GetMtime() == 0) {
		managed.SetMtime(now)
	}

	operator, hasOperator := opts.OperatorProvider(ctx)
	if !hasOperator {
		return
	}
	if managed, ok := entity.(dbspi.CreatorAccessor); ok && (opts.OverwriteExplicitValues || managed.GetCreator() == "") {
		managed.SetCreator(operator)
	}
	if managed, ok := entity.(dbspi.UpdaterAccessor); ok && (opts.OverwriteExplicitValues || managed.GetUpdater() == "") {
		managed.SetUpdater(operator)
	}
}

func applyUpdateCommonFields(ctx context.Context, opts dbspi.CommonFieldAutoFillOptions, entity any) {
	if shouldSkipCommonFields(opts, entity) {
		return
	}
	opts = opts.Normalize()

	if managed, ok := entity.(dbspi.UpdateTimeAccessor); ok && (opts.OverwriteExplicitValues || managed.GetMtime() == 0) {
		managed.SetMtime(opts.TimeProvider())
	}
	if managed, ok := entity.(dbspi.UpdaterAccessor); ok {
		if operator, ok := opts.OperatorProvider(ctx); ok && (opts.OverwriteExplicitValues || managed.GetUpdater() == "") {
			managed.SetUpdater(operator)
		}
	}
}

func applyCreateCommonFieldsToSlice[T any](ctx context.Context, opts dbspi.CommonFieldAutoFillOptions, entities []T) {
	for _, entity := range entities {
		applyCreateCommonFields(ctx, opts, entity)
	}
}

func applySaveCommonFieldsToSlice[T any](ctx context.Context, opts dbspi.CommonFieldAutoFillOptions, entities []T) {
	for _, entity := range entities {
		applySaveCommonFields(ctx, opts, entity)
	}
}

func applyUpdateCommonFieldsToUpdater(ctx context.Context, opts dbspi.CommonFieldAutoFillOptions, model any, updater dbspi.Updater) {
	if shouldSkipCommonFields(opts, model) || updater == nil {
		return
	}
	opts = opts.Normalize()
	params := updater.Values()

	if managed, ok := model.(dbspi.UpdateTimeAccessor); ok {
		fieldName := managed.MtimeFieldName()
		if opts.OverwriteExplicitValues || !hasUpdateParam(params, fieldName) {
			updater.Set(NewColumn(fieldName), opts.TimeProvider())
		}
	}

	operator, hasOperator := opts.OperatorProvider(ctx)
	if !hasOperator {
		return
	}
	if managed, ok := model.(dbspi.UpdaterAccessor); ok {
		fieldName := managed.UpdaterFieldName()
		if opts.OverwriteExplicitValues || !hasUpdateParam(params, fieldName) {
			updater.Set(NewColumn(fieldName), operator)
		}
	}
}

func shouldSkipCommonFields(opts dbspi.CommonFieldAutoFillOptions, entity any) bool {
	return !opts.AutoFillEnabled || isNilEntity(entity)
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
