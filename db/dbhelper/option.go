package dbhelper

// ManagerOption configures a Manager created by NewManager.
//
// ManagerOption is sealed to this package. Use the WithXxx helpers in dbhelper
// instead of implementing this interface directly.
type ManagerOption interface {
	applyManagerOption(*managerOptions)
}

// ExecutorOption configures an Executor created by NewExecutor or NewEnhancedExecutor.
//
// ExecutorOption is sealed to this package. Use the WithXxx helpers in dbhelper
// instead of implementing this interface directly.
type ExecutorOption interface {
	applyExecutorOption(*executorOptions)
}

// TransactionOption configures a transaction created by Transaction.
//
// TransactionOption is sealed to this package. Use the WithXxx helpers in
// dbhelper instead of implementing this interface directly.
type TransactionOption interface {
	applyTransactionOption(*transactionOptions)
}

// CommonFieldOption can be used both as a Manager global option and as a
// per-table NewExecutor/NewEnhancedExecutor override.
type CommonFieldOption interface {
	ManagerOption
	ExecutorOption
	TransactionOption
}

// UseManagerOption selects the Manager used by NewExecutor, NewEnhancedExecutor,
// or Transaction.
type UseManagerOption interface {
	ExecutorOption
	TransactionOption
}
