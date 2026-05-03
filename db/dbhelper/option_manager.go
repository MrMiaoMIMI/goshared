package dbhelper

import "github.com/MrMiaoMIMI/goshared/db/dbspi"

// WithManager makes NewTableStore/NewSoftDeleteTableStore/Transaction use the given
// Manager instead of the global default manager.
func WithManager(mgr dbspi.Manager) ManagerSelectionOption {
	return managerSelectionOption{manager: mgr}
}

type managerSelectionOption struct {
	manager dbspi.Manager
}

func (o managerSelectionOption) applyTableStoreOption(opts *tableStoreOptions) {
	opts.manager = o.manager
}

func (o managerSelectionOption) applyTransactionOption(opts *transactionOptions) {
	opts.manager = o.manager
}
