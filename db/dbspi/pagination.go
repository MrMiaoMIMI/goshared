package dbspi

type Pagination interface {
	WithLimit(limit *int) Pagination
	WithOffset(offset *int) Pagination
	AppendOrder(order Order) Pagination
	Limit() *int
	Offset() *int
	Orders() []Order
}

type Order interface {
	Column() Column
	Desc() bool
}
