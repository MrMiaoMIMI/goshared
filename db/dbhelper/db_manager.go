package dbhelper

import (
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

// NewDbManager creates a new DbManager from the given configuration.
func NewDbManager(cfg dbspi.DatabaseConfig, opts ...ManagerOption) dbspi.DbManager {
	options := resolveManagerOptions(opts)
	commonFields := options.commonFields.apply(dbspi.DefaultCommonFieldOptions())
	return dbsp.NewDbManager(cfg, commonFields)
}

// SetDefault sets the global default DbManager.
func SetDefault(mgr dbspi.DbManager) {
	dbsp.SetDefaultDbManager(asDbManager(mgr))
}

// Default returns the global default DbManager.
func Default() dbspi.DbManager {
	return dbsp.DefaultDbManager()
}

// For creates an Executor for the given entity using the DbManager.
func For[T dbspi.Entity](entity T, opts ...ForOption) dbspi.Executor[T] {
	options := resolveForOptions(opts)
	mgr := asDbManager(options.manager)
	if mgr == nil {
		mgr = dbsp.DefaultDbManager()
	}
	commonFields := options.commonFields.apply(mgr.CommonFieldOptions())
	return dbsp.ForWithCommonFields(entity, mgr, commonFields)
}

// ForEnhance creates an EnhancedExecutor for the given entity using the DbManager.
func ForEnhance[T dbspi.Entity](entity T, opts ...ForOption) dbspi.EnhancedExecutor[T] {
	options := resolveForOptions(opts)
	mgr := asDbManager(options.manager)
	if mgr == nil {
		mgr = dbsp.DefaultDbManager()
	}
	commonFields := options.commonFields.apply(mgr.CommonFieldOptions())
	return dbsp.ForEnhanceWithCommonFields(entity, mgr, commonFields)
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

func resolveForOptions(opts []ForOption) forOptions {
	var options forOptions
	for _, opt := range opts {
		if opt != nil {
			opt.applyForOption(&options)
		}
	}
	return options
}

func asDbManager(mgr dbspi.DbManager) *dbsp.DbManager {
	if mgr == nil {
		return nil
	}
	internal, ok := mgr.(*dbsp.DbManager)
	if !ok {
		panic("dbhelper: unsupported DbManager implementation")
	}
	return internal
}
