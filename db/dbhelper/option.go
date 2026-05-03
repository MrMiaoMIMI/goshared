package dbhelper

// ManagerOption configures a Manager created by NewManager.
//
// ManagerOption is sealed to this package. Use the WithXxx helpers in dbhelper
// instead of implementing this interface directly.
type ManagerOption interface {
	applyManagerOption(*managerOptions)
}

// TableStoreOption configures a TableStore created by NewTableStore or NewSoftDeleteTableStore.
//
// TableStoreOption is sealed to this package. Use the WithXxx helpers in dbhelper
// instead of implementing this interface directly.
type TableStoreOption interface {
	applyTableStoreOption(*tableStoreOptions)
}

// TransactionOption configures a transaction created by Transaction.
//
// TransactionOption is sealed to this package. Use the WithXxx helpers in
// dbhelper instead of implementing this interface directly.
type TransactionOption interface {
	applyTransactionOption(*transactionOptions)
}

// CommonFieldAutoFillOption can be used both as a Manager global option and as a
// per-table NewTableStore/NewSoftDeleteTableStore override.
type CommonFieldAutoFillOption interface {
	ManagerOption
	TableStoreOption
	TransactionOption
}

// ManagerSelectionOption selects the Manager used by NewTableStore, NewSoftDeleteTableStore,
// or Transaction.
type ManagerSelectionOption interface {
	TableStoreOption
	TransactionOption
}
