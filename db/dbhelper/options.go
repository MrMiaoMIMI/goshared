package dbhelper

import (
	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

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

type managerOptions struct {
	commonFields commonFieldPatch
}

type executorOptions struct {
	manager      dbspi.Manager
	tx           *Tx
	setTx        bool
	commonFields commonFieldPatch
}

type transactionOptions struct {
	manager          dbspi.Manager
	databaseGroupKey string
	shardingKey      *dbspi.ShardingKey
	commonFields     commonFieldPatch
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

func (f commonFieldOptionFunc) applyExecutorOption(o *executorOptions) {
	f(&o.commonFields)
}

func (f commonFieldOptionFunc) applyTransactionOption(o *transactionOptions) {
	f(&o.commonFields)
}

type managerSelectionOption struct {
	manager dbspi.Manager
}

func (o managerSelectionOption) applyExecutorOption(opts *executorOptions) {
	opts.manager = o.manager
}

func (o managerSelectionOption) applyTransactionOption(opts *transactionOptions) {
	opts.manager = o.manager
}

type txExecutorOption struct {
	tx *Tx
}

func (o txExecutorOption) applyExecutorOption(opts *executorOptions) {
	opts.tx = o.tx
	opts.setTx = true
}

type transactionOptionFunc func(*transactionOptions)

func (f transactionOptionFunc) applyTransactionOption(o *transactionOptions) {
	f(o)
}

// WithCommonFieldAutoFill enables or disables common-field autofill.
func WithCommonFieldAutoFill(enabled bool) CommonFieldOption {
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

// WithManager makes NewExecutor/NewEnhancedExecutor/Transaction use the given
// Manager instead of the global default manager.
func WithManager(mgr dbspi.Manager) UseManagerOption {
	return managerSelectionOption{manager: mgr}
}

// WithTx makes NewExecutor/NewEnhancedExecutor run on tx.
//
// If WithTx and WithManager are both provided to an executor factory, WithTx
// takes precedence because a Tx is already bound to the Manager selected when
// Transaction started.
//
// Invalid transaction state, such as a nil Tx or a database group mismatch, is
// reported by the returned executor's methods because NewExecutor itself does
// not return an error.
func WithTx(tx *Tx) ExecutorOption {
	return txExecutorOption{tx: tx}
}

// WithTxDatabaseGroupKey selects the database group used by a transaction.
//
// If omitted, Transaction uses dbspi.DefaultDatabaseGroupKey.
func WithTxDatabaseGroupKey(databaseGroupKey string) TransactionOption {
	return transactionOptionFunc(func(o *transactionOptions) {
		o.databaseGroupKey = databaseGroupKey
	})
}

// WithTxShardingKey selects the physical database shard used by a transaction.
//
// It is required when the selected database group has database-level sharding.
func WithTxShardingKey(key *dbspi.ShardingKey) TransactionOption {
	return transactionOptionFunc(func(o *transactionOptions) {
		o.shardingKey = key
	})
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
