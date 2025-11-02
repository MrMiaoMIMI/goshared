package dbspi

type DbConfig interface {
	Host() string
	Port() uint
	User() string
	Password() string
	DbName() string
	GetDSN() string
}

type PaginationConfig interface {
	WithLimit(limit *int) PaginationConfig
	WithOffset(offset *int) PaginationConfig
	AppendOrder(order OrderConfig) PaginationConfig
	Limit() *int
	Offset() *int
	Orders() []OrderConfig
}

type OrderConfig interface {
	Column() Column
	Desc() bool
}
