package dbsp

import (
	"fmt"

	"github.com/MrMiaoMIMI/goshared/db/dbspi"
)

var (
	_ dbspi.DbConfig         = (*DbConfig)(nil)
	_ dbspi.PaginationConfig = (*PaginationConfig)(nil)
	_ dbspi.OrderConfig      = (*OrderConfig)(nil)
)

type DbConfig struct {
	host     string
	port     uint
	user     string
	password string
	dbName   string
}

func NewDbConfig(host string, port uint, user string, password string, dbName string) dbspi.DbConfig {
	return &DbConfig{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		dbName:   dbName,
	}
}

func (c *DbConfig) Host() string {
	return c.host
}

func (c *DbConfig) Port() uint {
	return c.port
}

func (c *DbConfig) User() string {
	return c.user
}

func (c *DbConfig) Password() string {
	return c.password
}

func (c *DbConfig) DbName() string {
	return c.dbName
}

func (c *DbConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", c.User(), c.Password(), c.Host(), c.Port(), c.DbName())
}

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
