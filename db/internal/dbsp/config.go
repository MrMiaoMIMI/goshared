package dbsp

import "github.com/MrMiaoMIMI/goshared/db/dbspi"

var (
	_ dbspi.Pagination = (*Pagination)(nil)
	_ dbspi.Order      = (*Order)(nil)
)

type Pagination struct {
	limit  *int
	offset *int
	orders []dbspi.Order
}

func NewPagination() dbspi.Pagination {
	return &Pagination{}
}

func (c *Pagination) WithLimit(limit *int) dbspi.Pagination {
	c.limit = limit
	return c
}

func (c *Pagination) WithOffset(offset *int) dbspi.Pagination {
	c.offset = offset
	return c
}

func (c *Pagination) AppendOrder(order dbspi.Order) dbspi.Pagination {
	c.orders = append(c.orders, order)
	return c
}

func (c *Pagination) Limit() *int {
	return c.limit
}

func (c *Pagination) Offset() *int {
	return c.offset
}

func (c *Pagination) Orders() []dbspi.Order {
	return c.orders
}

type Order struct {
	column dbspi.Column
	desc   bool
}

func newOrder(column dbspi.Column, desc bool) dbspi.Order {
	return &Order{
		column: column,
		desc:   desc,
	}
}

func Asc(column dbspi.Column) dbspi.Order {
	return newOrder(column, false)
}

func Desc(column dbspi.Column) dbspi.Order {
	return newOrder(column, true)
}

func (c *Order) Column() dbspi.Column {
	return c.column
}

func (c *Order) Desc() bool {
	return c.desc
}
