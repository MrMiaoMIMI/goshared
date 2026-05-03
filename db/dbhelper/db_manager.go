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

// NewTableStore creates a TableStore for the given entity using the Manager.
func NewTableStore[T dbspi.Entity](entity T, opts ...TableStoreOption) dbspi.TableStore[T] {
	options := resolveTableStoreOptions(opts)
	if options.setTx {
		return newTxTableStore(entity, options.tx, options.commonFields)
	}

	mgr := asInternalManager(options.manager)
	if mgr == nil {
		mgr = dbsp.DefaultManager()
	}
	commonFields := options.commonFields.apply(mgr.CommonFieldAutoFillOptions())
	return dbsp.ForWithCommonFieldAutoFill(entity, mgr, commonFields)
}

// NewSoftDeleteTableStore creates a SoftDeleteTableStore for the given entity using the Manager.
func NewSoftDeleteTableStore[T dbspi.Entity](entity T, opts ...TableStoreOption) dbspi.SoftDeleteTableStore[T] {
	options := resolveTableStoreOptions(opts)
	if options.setTx {
		return newTxSoftDeleteTableStore(entity, options.tx, options.commonFields)
	}

	mgr := asInternalManager(options.manager)
	if mgr == nil {
		mgr = dbsp.DefaultManager()
	}
	commonFields := options.commonFields.apply(mgr.CommonFieldAutoFillOptions())
	return dbsp.ForSoftDeleteWithCommonFieldAutoFill(entity, mgr, commonFields)
}

// AsSQLTableStore exposes advanced raw SQL support when store supports it.
//
// Prefer TableStore methods for regular business reads and writes. For sharded
// tables, route raw SQL explicitly with Shard or pass a dbspi.ShardingKey in ctx.
func AsSQLTableStore[T dbspi.Entity](store dbspi.TableStore[T]) (dbspi.SQLTableStore[T], bool) {
	sqlStore, ok := store.(dbspi.SQLTableStore[T])
	return sqlStore, ok
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

func resolveTableStoreOptions(opts []TableStoreOption) tableStoreOptions {
	var options tableStoreOptions
	for _, opt := range opts {
		if opt != nil {
			opt.applyTableStoreOption(&options)
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
