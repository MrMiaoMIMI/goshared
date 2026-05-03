package dbspi

// Pagination configures query limit, offset, and ordering.
type Pagination interface {
	WithLimit(limit *int) Pagination
	WithOffset(offset *int) Pagination
	AppendOrder(order Order) Pagination
	Limit() *int
	Offset() *int
	Orders() []Order
}

// Order configures one ORDER BY column.
type Order interface {
	Column() Column
	Desc() bool
}
