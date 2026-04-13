package servererr

// =============================================================================
// 错误码定义
// 格式: HTTPSTATUS + 5位业务码
// =============================================================================

// ErrorCode 业务错误码类型。
type ErrorCode int

const (
	Success ErrorCode = 0 // 成功

	// 4xx 客户端错误
	ErrBadRequest   ErrorCode = 40000000 // 请求参数错误
	ErrUnauthorized ErrorCode = 40100000 // 未认证
	ErrForbidden    ErrorCode = 40300000 // 无权限
	ErrNotFound     ErrorCode = 40400000 // 资源不存在

	// 5xx 服务端错误
	ErrInternal ErrorCode = 50000001 // 内部错误
)

// HTTPStatus 错误码对应的 HTTP 状态码。
func (c ErrorCode) HTTPStatus() int {
	return int(c) / 100000
}
