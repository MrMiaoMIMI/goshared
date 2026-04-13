package servererr

import (
	"fmt"
)

// =============================================================================
// BizError 业务错误
// =============================================================================

// BizError 业务错误，包含错误码和消息。
type BizError struct {
	Code    ErrorCode `json:"code"`    // 业务错误码
	Message string    `json:"message"` // 错误描述
}

// Error 实现 error 接口。
func (e *BizError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewBizError 创建业务错误。
func NewBizError(code ErrorCode, message string) *BizError {
	return &BizError{
		Code:    code,
		Message: message,
	}
}

// NewBizErrorf 创建格式化业务错误。
func NewBizErrorf(code ErrorCode, format string, args ...interface{}) *BizError {
	return &BizError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}
