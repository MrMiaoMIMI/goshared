package servererr

// ErrorCode 业务错误码类型。
// 格式: HTTPSTATUS + 5位业务码
type ErrorCode int

const (
	Success ErrorCode = 0

	// 4xx 客户端错误
	ErrBadRequest          ErrorCode = 40000000 // 请求参数错误
	ErrUnauthorized        ErrorCode = 40100000 // 未认证
	ErrPaymentRequired     ErrorCode = 40200000 // 需要付费
	ErrForbidden           ErrorCode = 40300000 // 无权限
	ErrNotFound            ErrorCode = 40400000 // 资源不存在
	ErrMethodNotAllowed    ErrorCode = 40500000 // 方法不允许
	ErrConflict            ErrorCode = 40900000 // 资源冲突
	ErrTooManyRequests     ErrorCode = 42900000 // 请求过多
	ErrRequestTimeout      ErrorCode = 40800000 // 请求超时
	ErrUnprocessableEntity ErrorCode = 42200000 // 参数验证失败

	// 5xx 服务端错误
	ErrInternal        ErrorCode = 50000001 // 内部错误
	ErrNotImplemented  ErrorCode = 50100000 // 未实现
	ErrBadGateway      ErrorCode = 50200000 // 网关错误
	ErrServiceDown     ErrorCode = 50300000 // 服务不可用
	ErrGatewayTimeout  ErrorCode = 50400000 // 网关超时
	ErrDatabaseError   ErrorCode = 50000002 // 数据库错误
	ErrCacheError      ErrorCode = 50000003 // 缓存错误
	ErrExternalService ErrorCode = 50000004 // 外部服务错误
)

// HTTPStatus 错误码对应的 HTTP 状态码。
func (c ErrorCode) HTTPStatus() int {
	return int(c) / 100000
}

// IsSuccess 判断是否成功。
func (c ErrorCode) IsSuccess() bool {
	return c == Success
}

// IsClientError 判断是否为 4xx 客户端错误。
func (c ErrorCode) IsClientError() bool {
	status := c.HTTPStatus()
	return status >= 400 && status < 500
}

// IsServerError 判断是否为 5xx 服务端错误。
func (c ErrorCode) IsServerError() bool {
	status := c.HTTPStatus()
	return status >= 500 && status < 600
}
