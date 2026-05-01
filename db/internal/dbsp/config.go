package dbsp

import "github.com/MrMiaoMIMI/goshared/db/dbspi"

var (
	_ dbspi.PaginationConfig = (*PaginationConfig)(nil)
	_ dbspi.OrderConfig      = (*OrderConfig)(nil)
)

type PaginationConfig struct {
	limit  *int
	offset *int
	orders []dbspi.OrderConfig
}

func NewPaginationConfig() dbspi.PaginationConfig {
	return &PaginationConfig{}
}

func (c *PaginationConfig) WithLimit(limit *int) dbspi.PaginationConfig {
	c.limit = limit
	return c
}

func (c *PaginationConfig) WithOffset(offset *int) dbspi.PaginationConfig {
	c.offset = offset
	return c
}

func (c *PaginationConfig) AppendOrder(order dbspi.OrderConfig) dbspi.PaginationConfig {
	c.orders = append(c.orders, order)
	return c
}

func (c *PaginationConfig) Limit() *int {
	return c.limit
}

func (c *PaginationConfig) Offset() *int {
	return c.offset
}

func (c *PaginationConfig) Orders() []dbspi.OrderConfig {
	return c.orders
}

type OrderConfig struct {
	column dbspi.Column
	desc   bool
}

func NewOrderConfig(column dbspi.Column, desc bool) dbspi.OrderConfig {
	return &OrderConfig{
		column: column,
		desc:   desc,
	}
}

func (c *OrderConfig) Column() dbspi.Column {
	return c.column
}

func (c *OrderConfig) Desc() bool {
	return c.desc
}
