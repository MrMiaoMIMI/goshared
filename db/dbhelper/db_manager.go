package dbhelper

import (
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

// NewManager creates a new Manager from the given configuration.
func NewManager(cfg dbspi.DatabaseConfig, opts ...ManagerOption) dbspi.Manager {
	options := resolveManagerOptions(opts)
	commonFields := options.commonFields.apply(dbsp.DefaultCommonFieldAutoFillOptions())
	return dbsp.NewManager(cfg, commonFields)
}

// SetDefaultManager sets the global default Manager.
func SetDefaultManager(mgr dbspi.Manager) {
	dbsp.SetDefaultManager(asInternalManager(mgr))
}

// DefaultManager returns the global default Manager.
func DefaultManager() dbspi.Manager {
	return dbsp.DefaultManager()
}

// NewExecutor creates an Executor for the given entity using the Manager.
func NewExecutor[T dbspi.Entity](entity T, opts ...ExecutorOption) dbspi.Executor[T] {
	options := resolveExecutorOptions(opts)
	if options.setTx {
		return newTxExecutor(entity, options.tx, options.commonFields)
	}

	mgr := asInternalManager(options.manager)
	if mgr == nil {
		mgr = dbsp.DefaultManager()
	}
	commonFields := options.commonFields.apply(mgr.CommonFieldAutoFillOptions())
	return dbsp.ForWithCommonFieldAutoFill(entity, mgr, commonFields)
}

// NewEnhancedExecutor creates an EnhancedExecutor for the given entity using the Manager.
func NewEnhancedExecutor[T dbspi.Entity](entity T, opts ...ExecutorOption) dbspi.EnhancedExecutor[T] {
	options := resolveExecutorOptions(opts)
	if options.setTx {
		return newTxEnhancedExecutor(entity, options.tx, options.commonFields)
	}

	mgr := asInternalManager(options.manager)
	if mgr == nil {
		mgr = dbsp.DefaultManager()
	}
	commonFields := options.commonFields.apply(mgr.CommonFieldAutoFillOptions())
	return dbsp.ForEnhanceWithCommonFieldAutoFill(entity, mgr, commonFields)
}

func resolveManagerOptions(opts []ManagerOption) managerOptions {
	var options managerOptions
	for _, opt := range opts {
		if opt != nil {
			opt.applyManagerOption(&options)
		}
	}
	return options
}

func resolveExecutorOptions(opts []ExecutorOption) executorOptions {
	var options executorOptions
	for _, opt := range opts {
		if opt != nil {
			opt.applyExecutorOption(&options)
		}
	}
	return options
}

func asInternalManager(mgr dbspi.Manager) *dbsp.Manager {
	if mgr == nil {
		return nil
	}
	internal, ok := mgr.(*dbsp.Manager)
	if !ok {
		panic("dbhelper: unsupported dbspi.Manager implementation; use dbhelper.NewManager")
	}
	return internal
}
