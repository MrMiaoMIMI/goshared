package servererr

import (
	"errors"
	"fmt"
)

// BizError 业务错误，包含错误码和消息。
type BizError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	cause   error
}

// Error 实现 error 接口。
func (e *BizError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Is / errors.As 链式解包。
func (e *BizError) Unwrap() error {
	return e.cause
}

// Is 支持 errors.Is 按错误码比较。
func (e *BizError) Is(target error) bool {
	var t *BizError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// WithCause 返回一个包含原因的新 BizError 副本。
func (e *BizError) WithCause(err error) *BizError {
	return &BizError{
		Code:    e.Code,
		Message: e.Message,
		cause:   err,
	}
}

// WithMessage 返回一个替换消息的新 BizError 副本。
func (e *BizError) WithMessage(msg string) *BizError {
	return &BizError{
		Code:    e.Code,
		Message: msg,
		cause:   e.cause,
	}
}

// WithMessagef 返回一个格式化替换消息的新 BizError 副本。
func (e *BizError) WithMessagef(format string, args ...interface{}) *BizError {
	return e.WithMessage(fmt.Sprintf(format, args...))
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

// Wrap 将底层 error 包装为 BizError。
func Wrap(code ErrorCode, message string, err error) *BizError {
	return &BizError{
		Code:    code,
		Message: message,
		cause:   err,
	}
}

// Wrapf 将底层 error 包装为格式化 BizError。
func Wrapf(code ErrorCode, err error, format string, args ...interface{}) *BizError {
	return &BizError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		cause:   err,
	}
}

// IsBizError 判断 err 是否为 BizError。
func IsBizError(err error) bool {
	var bizErr *BizError
	return errors.As(err, &bizErr)
}

// AsBizError 尝试将 err 转为 BizError。
func AsBizError(err error) (*BizError, bool) {
	var bizErr *BizError
	ok := errors.As(err, &bizErr)
	return bizErr, ok
}

// CodeOf 提取 error 的错误码。非 BizError 返回 ErrInternal。
func CodeOf(err error) ErrorCode {
	if bizErr, ok := AsBizError(err); ok {
		return bizErr.Code
	}
	return ErrInternal
}
