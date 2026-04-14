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
	host                   string
	port                   uint
	user                   string
	password               string
	dbName                 string
	maxOpenConns           int
	maxIdleConns           int
	connMaxLifetimeSeconds int
	debugMode              bool
}

// DbConfigOption configures the DbConfig at construction time.
type DbConfigOption func(*DbConfig)

func WithMaxOpenConns(n int) DbConfigOption {
	return func(c *DbConfig) { c.maxOpenConns = n }
}

func WithMaxIdleConns(n int) DbConfigOption {
	return func(c *DbConfig) { c.maxIdleConns = n }
}

func WithConnMaxLifetimeSeconds(s int) DbConfigOption {
	return func(c *DbConfig) { c.connMaxLifetimeSeconds = s }
}

func WithDebugMode(debug bool) DbConfigOption {
	return func(c *DbConfig) { c.debugMode = debug }
}

func NewDbConfig(host string, port uint, user string, password string, dbName string, opts ...DbConfigOption) dbspi.DbConfig {
	cfg := &DbConfig{
		host:                   host,
		port:                   port,
		user:                   user,
		password:               password,
		dbName:                 dbName,
		maxOpenConns:           100,
		maxIdleConns:           10,
		connMaxLifetimeSeconds: 3600,
		debugMode:              false,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func (c *DbConfig) Host() string                { return c.host }
func (c *DbConfig) Port() uint                  { return c.port }
func (c *DbConfig) User() string                { return c.user }
func (c *DbConfig) Password() string            { return c.password }
func (c *DbConfig) DbName() string              { return c.dbName }
func (c *DbConfig) MaxOpenConns() int            { return c.maxOpenConns }
func (c *DbConfig) MaxIdleConns() int            { return c.maxIdleConns }
func (c *DbConfig) ConnMaxLifetimeSeconds() int  { return c.connMaxLifetimeSeconds }
func (c *DbConfig) DebugMode() bool              { return c.debugMode }

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
