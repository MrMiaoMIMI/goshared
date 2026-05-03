package dbspi

import "context"

type operatorCtxKey struct{}

// WithOperator injects the current operator into ctx for common-field autofill.
func WithOperator(ctx context.Context, operator string) context.Context {
	return context.WithValue(ctx, operatorCtxKey{}, operator)
}

// OperatorFromContext extracts the current operator from ctx.
func OperatorFromContext(ctx context.Context) (string, bool) {
	operator, ok := ctx.Value(operatorCtxKey{}).(string)
	return operator, ok
}
