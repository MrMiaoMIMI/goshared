package dbhelper

import "github.com/MrMiaoMIMI/goshared/db/dbspi"

// WithCommonFieldAutoFill enables or disables common-field autofill.
func WithCommonFieldAutoFill(enabled bool) CommonFieldAutoFillOption {
	return commonFieldOptionFunc(func(p *commonFieldPatch) {
		p.setEnabled = true
		p.enabled = enabled
	})
}

// WithCommonFieldOverwriteExplicitValues controls whether common-field autofill
// may overwrite values explicitly provided by the caller.
//
// By default, explicit values are preserved and only zero-value ctime/mtime or
// empty creator/updater are filled.
func WithCommonFieldOverwriteExplicitValues(overwrite bool) CommonFieldAutoFillOption {
	return commonFieldOptionFunc(func(p *commonFieldPatch) {
		p.setOverwriteExplicitValues = true
		p.overwriteExplicitValues = overwrite
	})
}

// WithCommonFieldTimeProvider sets the timestamp provider for ctime/mtime.
// The default provider returns Unix milliseconds.
func WithCommonFieldTimeProvider(provider dbspi.TimeProvider) CommonFieldAutoFillOption {
	return commonFieldOptionFunc(func(p *commonFieldPatch) {
		p.timeProvider = provider
	})
}

// WithCommonFieldOperatorProvider sets the operator provider for common fields.
func WithCommonFieldOperatorProvider(provider dbspi.OperatorProvider) CommonFieldAutoFillOption {
	return commonFieldOptionFunc(func(p *commonFieldPatch) {
		p.operatorProvider = provider
	})
}

type commonFieldOptionFunc func(*commonFieldPatch)

func (f commonFieldOptionFunc) applyManagerOption(o *managerOptions) {
	f(&o.commonFields)
}

func (f commonFieldOptionFunc) applyExecutorOption(o *executorOptions) {
	f(&o.commonFields)
}

func (f commonFieldOptionFunc) applyTransactionOption(o *transactionOptions) {
	f(&o.commonFields)
}

type commonFieldPatch struct {
	setEnabled                 bool
	enabled                    bool
	setOverwriteExplicitValues bool
	overwriteExplicitValues    bool
	timeProvider               dbspi.TimeProvider
	operatorProvider           dbspi.OperatorProvider
}

func (p commonFieldPatch) apply(base dbspi.CommonFieldAutoFillOptions) dbspi.CommonFieldAutoFillOptions {
	if p.setEnabled {
		base.AutoFillEnabled = p.enabled
	}
	if p.setOverwriteExplicitValues {
		base.OverwriteExplicitValues = p.overwriteExplicitValues
	}
	if p.timeProvider != nil {
		base.TimeProvider = p.timeProvider
	}
	if p.operatorProvider != nil {
		base.OperatorProvider = p.operatorProvider
	}
	return base.Normalize()
}
