package dbhelper

import (
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
	"github.com/MrMiaoMIMI/goshared/db/internal/dbsp"
)

// NewDbManager creates a new DbManager from the given configuration.
func NewDbManager(cfg dbspi.DatabaseConfig) dbspi.DbManager {
	return dbsp.NewDbManager(cfg)
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
func For[T dbspi.Entity](entity T, managers ...dbspi.DbManager) dbspi.Executor[T] {
	if len(managers) == 0 {
		return dbsp.For(entity)
	}
	internalManagers := make([]*dbsp.DbManager, len(managers))
	for i, mgr := range managers {
		internalManagers[i] = asDbManager(mgr)
	}
	return dbsp.For(entity, internalManagers...)
}

// ForEnhance creates an EnhancedExecutor for the given entity using the DbManager.
func ForEnhance[T dbspi.Entity](entity T, managers ...dbspi.DbManager) dbspi.EnhancedExecutor[T] {
	if len(managers) == 0 {
		return dbsp.ForEnhance(entity)
	}
	internalManagers := make([]*dbsp.DbManager, len(managers))
	for i, mgr := range managers {
		internalManagers[i] = asDbManager(mgr)
	}
	return dbsp.ForEnhance(entity, internalManagers...)
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
