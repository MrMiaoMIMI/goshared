package dbhelper

// WithTx makes NewTableStore/NewSoftDeleteTableStore run on tx.
//
// If WithTx and WithManager are both provided to a table store factory, WithTx
// takes precedence because a Tx is already bound to the Manager selected when
// Transaction started.
//
// Invalid transaction state, such as a nil Tx or a database group mismatch, is
// reported by the returned table store's methods because NewTableStore itself does
// not return an error.
func WithTx(tx *Tx) TableStoreOption {
	return txTableStoreOption{tx: tx}
}

type txTableStoreOption struct {
	tx *Tx
}

func (o txTableStoreOption) applyTableStoreOption(opts *tableStoreOptions) {
	opts.tx = o.tx
	opts.setTx = true
}
