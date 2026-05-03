package dbhelper

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

type txExecutorOption struct {
	tx *Tx
}

func (o txExecutorOption) applyExecutorOption(opts *executorOptions) {
	opts.tx = o.tx
	opts.setTx = true
}
