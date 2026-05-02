package dbhelper

import (
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

// ManagerOption configures a DbManager created by NewDbManager.
//
// ManagerOption is sealed to this package. Use the WithXxx helpers in dbhelper
// instead of implementing this interface directly.
type ManagerOption interface {
	applyManagerOption(*managerOptions)
}

// ForOption configures an Executor created by For or ForEnhance.
//
// ForOption is sealed to this package. Use the WithXxx helpers in dbhelper
// instead of implementing this interface directly.
type ForOption interface {
	applyForOption(*forOptions)
}

// CommonFieldOption can be used both as a DbManager global option and as a
// per-table For/ForEnhance override.
type CommonFieldOption interface {
	ManagerOption
	ForOption
}

type managerOptions struct {
	commonFields commonFieldPatch
}

type forOptions struct {
	manager      dbspi.DbManager
	commonFields commonFieldPatch
}

type commonFieldPatch struct {
	setEnabled                 bool
	enabled                    bool
	setOverwriteExplicitValues bool
	overwriteExplicitValues    bool
	timeProvider               dbspi.TimeProvider
	operatorProvider           dbspi.OperatorProvider
}

type commonFieldOptionFunc func(*commonFieldPatch)

func (f commonFieldOptionFunc) applyManagerOption(o *managerOptions) {
	f(&o.commonFields)
}

func (f commonFieldOptionFunc) applyForOption(o *forOptions) {
	f(&o.commonFields)
}

type forOptionFunc func(*forOptions)

func (f forOptionFunc) applyForOption(o *forOptions) {
	f(o)
}

// WithCommonFields enables or disables common-field autofill.
func WithCommonFields(enabled bool) CommonFieldOption {
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
func WithCommonFieldOverwriteExplicitValues(overwrite bool) CommonFieldOption {
	return commonFieldOptionFunc(func(p *commonFieldPatch) {
		p.setOverwriteExplicitValues = true
		p.overwriteExplicitValues = overwrite
	})
}

// WithCommonFieldTimeProvider sets the timestamp provider for ctime/mtime.
// The default provider returns Unix milliseconds.
func WithCommonFieldTimeProvider(provider dbspi.TimeProvider) CommonFieldOption {
	return commonFieldOptionFunc(func(p *commonFieldPatch) {
		p.timeProvider = provider
	})
}

// WithCommonFieldOperatorProvider sets the operator provider for common fields.
func WithCommonFieldOperatorProvider(provider dbspi.OperatorProvider) CommonFieldOption {
	return commonFieldOptionFunc(func(p *commonFieldPatch) {
		p.operatorProvider = provider
	})
}

// WithDbManager makes For/ForEnhance use the given DbManager instead of
// the global default manager.
func WithDbManager(mgr dbspi.DbManager) ForOption {
	return forOptionFunc(func(o *forOptions) {
		o.manager = mgr
	})
}

func (p commonFieldPatch) apply(base dbspi.CommonFieldOptions) dbspi.CommonFieldOptions {
	if p.setEnabled {
		base.Enabled = p.enabled
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
