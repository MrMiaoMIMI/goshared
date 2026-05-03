package dbspi

import (
	"context"
	"time"
)

// DefaultTimeProvider returns the current Unix timestamp in milliseconds.
func DefaultTimeProvider(context.Context) uint64 {
	return uint64(time.Now().UnixMilli())
}

// OperatorProvider resolves the current operator from ctx.
type OperatorProvider func(ctx context.Context) (string, bool)

// TimeProvider returns the timestamp value used by common fields.
//
// The unit is application-defined. The default provider uses Unix milliseconds.
// Implementations should be lightweight and use ctx only for request-scoped
// values, such as a fixed timestamp shared by one operation.
type TimeProvider func(ctx context.Context) uint64
