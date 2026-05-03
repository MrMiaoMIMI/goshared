package dbsp

import "github.com/MrMiaoMIMI/goshared/db/dbspi"

// CommonFieldAutoFillOptions configures automatic maintenance for common fields.
type CommonFieldAutoFillOptions struct {
	// AutoFillEnabled controls whether table stores apply common-field automation at all.
	AutoFillEnabled bool

	// OverwriteExplicitValues controls whether generated common-field values may
	// overwrite values already provided by the caller.
	//
	// When false, common-field automation only fills zero-value ctime/mtime and
	// empty creator/updater. UpdateByQuery also keeps explicit mtime/updater
	// values in the updater. Set this to true to force generated values to
	// replace explicit values.
	OverwriteExplicitValues bool

	// TimeProvider generates values for ctime and mtime. If nil, Normalize uses
	// the default Unix-millisecond provider.
	TimeProvider dbspi.TimeProvider

	// OperatorProvider resolves creator and updater values from ctx. If nil,
	// Normalize uses dbspi.OperatorFromContext.
	OperatorProvider dbspi.OperatorProvider
}

// DefaultCommonFieldAutoFillOptions returns the default common-field behavior.
//
// Common-field automation is enabled by default, uses Unix milliseconds for
// ctime/mtime, and resolves creator/updater from ctx with dbspi.OperatorFromContext.
func DefaultCommonFieldAutoFillOptions() CommonFieldAutoFillOptions {
	return CommonFieldAutoFillOptions{
		AutoFillEnabled:  true,
		TimeProvider:     dbspi.DefaultTimeProvider,
		OperatorProvider: dbspi.OperatorFromContext,
	}
}

// DisabledCommonFieldAutoFillOptions disables common-field behavior.
func DisabledCommonFieldAutoFillOptions() CommonFieldAutoFillOptions {
	return CommonFieldAutoFillOptions{}
}

// Normalize fills missing function hooks with defaults.
func (o CommonFieldAutoFillOptions) Normalize() CommonFieldAutoFillOptions {
	defaults := DefaultCommonFieldAutoFillOptions()
	if o.TimeProvider == nil {
		o.TimeProvider = defaults.TimeProvider
	}
	if o.OperatorProvider == nil {
		o.OperatorProvider = defaults.OperatorProvider
	}
	return o
}
