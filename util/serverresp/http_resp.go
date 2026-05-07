package serverresp

import (
	"errors"
	"net/http"

	"github.com/MrMiaoMIMI/goshared/util/servererr"
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response[T any] struct {
	Code    servererr.ErrorCode `json:"code"`
	Message string              `json:"message"`
	Data    T                   `json:"data,omitempty"`
}

// PageData 分页数据结构
type PageData[T any] struct {
	List  []T   `json:"list"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
}

func httpResponse[T any](ctx *gin.Context, data T, httpStatus int, code servererr.ErrorCode, message string) {
	ctx.JSON(httpStatus, Response[T]{Code: code, Message: message, Data: data})
}

// Success 成功响应（200 + 数据）
func Success[T any](ctx *gin.Context, data T) {
	httpResponse(ctx, data, http.StatusOK, servererr.Success, "success")
}

// SuccessMessage 成功响应（200 + 自定义消息，无数据）
func SuccessMessage(ctx *gin.Context, message string) {
	httpResponse[any](ctx, nil, http.StatusOK, servererr.Success, message)
}

// SuccessPage 成功分页响应
func SuccessPage[T any](ctx *gin.Context, list []T, total int64, page, size int) {
	Success(ctx, PageData[T]{
		List:  list,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// Error 通用错误响应：自动从 BizError 提取状态码，否则使用 500。
func Error(ctx *gin.Context, err error) {
	if err == nil {
		httpResponse[any](ctx, nil, http.StatusInternalServerError, servererr.ErrInternal, "unknown error")
		return
	}
	var bizErr *servererr.BizError
	if errors.As(err, &bizErr) {
		httpResponse[any](ctx, nil, bizErr.Code.HTTPStatus(), bizErr.Code, bizErr.Message)
		return
	}
	httpResponse[any](ctx, nil, http.StatusInternalServerError, servererr.ErrInternal, err.Error())
}

// ErrorWithCode 指定错误码的错误响应
func ErrorWithCode(ctx *gin.Context, code servererr.ErrorCode, message string) {
	httpResponse[any](ctx, nil, code.HTTPStatus(), code, message)
}

// BadRequestError 400 错误
func BadRequestError(ctx *gin.Context, err error) {
	var bizErr *servererr.BizError
	if errors.As(err, &bizErr) {
		httpResponse[any](ctx, nil, bizErr.Code.HTTPStatus(), bizErr.Code, bizErr.Message)
		return
	}
	httpResponse[any](ctx, nil, http.StatusBadRequest, servererr.ErrBadRequest, errMsg(err))
}

// UnauthorizedError 401 错误
func UnauthorizedError(ctx *gin.Context, err error) {
	httpResponse[any](ctx, nil, http.StatusUnauthorized, servererr.ErrUnauthorized, errMsg(err))
}

// ForbiddenError 403 错误
func ForbiddenError(ctx *gin.Context, err error) {
	httpResponse[any](ctx, nil, http.StatusForbidden, servererr.ErrForbidden, errMsg(err))
}

// NotFoundError 404 错误
func NotFoundError(ctx *gin.Context, err error) {
	httpResponse[any](ctx, nil, http.StatusNotFound, servererr.ErrNotFound, errMsg(err))
}

// InternalServerError 500 错误
func InternalServerError(ctx *gin.Context, err error) {
	if err == nil {
		httpResponse[any](ctx, nil, http.StatusInternalServerError, servererr.ErrInternal, "unknown error")
		return
	}
	var bizErr *servererr.BizError
	if errors.As(err, &bizErr) {
		httpResponse[any](ctx, nil, bizErr.Code.HTTPStatus(), bizErr.Code, bizErr.Message)
		return
	}
	httpResponse[any](ctx, nil, http.StatusInternalServerError, servererr.ErrInternal, err.Error())
}

// TooManyRequestsError 429 错误
func TooManyRequestsError(ctx *gin.Context, err error) {
	httpResponse[any](ctx, nil, http.StatusTooManyRequests, servererr.ErrTooManyRequests, errMsg(err))
}

func errMsg(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}
