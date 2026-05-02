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

// TransactionOption configures a transaction created by Transaction.
//
// TransactionOption is sealed to this package. Use the WithXxx helpers in
// dbhelper instead of implementing this interface directly.
type TransactionOption interface {
	applyTransactionOption(*transactionOptions)
}

// CommonFieldOption can be used both as a DbManager global option and as a
// per-table For/ForEnhance/ForTx override.
type CommonFieldOption interface {
	ManagerOption
	ForOption
	TransactionOption
}

// DbManagerOption selects the DbManager used by For/ForEnhance or Transaction.
type DbManagerOption interface {
	ForOption
	TransactionOption
}

type managerOptions struct {
	commonFields commonFieldPatch
}

type forOptions struct {
	manager      dbspi.DbManager
	commonFields commonFieldPatch
}

type transactionOptions struct {
	manager      dbspi.DbManager
	dbKey        string
	shardingKey  *dbspi.ShardingKey
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

func (f commonFieldOptionFunc) applyTransactionOption(o *transactionOptions) {
	f(&o.commonFields)
}

type dbManagerOption struct {
	manager dbspi.DbManager
}

func (o dbManagerOption) applyForOption(opts *forOptions) {
	opts.manager = o.manager
}

func (o dbManagerOption) applyTransactionOption(opts *transactionOptions) {
	opts.manager = o.manager
}

type transactionOptionFunc func(*transactionOptions)

func (f transactionOptionFunc) applyTransactionOption(o *transactionOptions) {
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

// WithDbManager makes For/ForEnhance/Transaction use the given DbManager
// instead of the global default manager.
func WithDbManager(mgr dbspi.DbManager) DbManagerOption {
	return dbManagerOption{manager: mgr}
}

// WithTxDbKey selects the database group used by a transaction.
//
// If omitted, Transaction uses dbspi.DefaultDbKey.
func WithTxDbKey(dbKey string) TransactionOption {
	return transactionOptionFunc(func(o *transactionOptions) {
		o.dbKey = dbKey
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
