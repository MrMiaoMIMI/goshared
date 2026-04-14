package serverresp

import (
	"errors"
	"net/http"

	"github.com/MrMiaoMIMI/goshared/util/servererr"
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    servererr.ErrorCode `json:"code"`
	Message string              `json:"message"`
	Data    any                 `json:"data,omitempty"`
}

// PageData 分页数据结构
type PageData struct {
	List  any   `json:"list"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
}

func httpResponse(ctx *gin.Context, data any, httpStatus int, code servererr.ErrorCode, message string) {
	ctx.JSON(httpStatus, Response{Code: code, Message: message, Data: data})
}

// Success 成功响应（200 + 数据）
func Success(ctx *gin.Context, data any) {
	httpResponse(ctx, data, http.StatusOK, servererr.Success, "success")
}

// SuccessMessage 成功响应（200 + 自定义消息，无数据）
func SuccessMessage(ctx *gin.Context, message string) {
	httpResponse(ctx, nil, http.StatusOK, servererr.Success, message)
}

// SuccessPage 成功分页响应
func SuccessPage(ctx *gin.Context, list any, total int64, page, size int) {
	Success(ctx, PageData{
		List:  list,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// Error 通用错误响应：自动从 BizError 提取状态码，否则使用 500。
func Error(ctx *gin.Context, err error) {
	if err == nil {
		httpResponse(ctx, nil, http.StatusInternalServerError, servererr.ErrInternal, "unknown error")
		return
	}
	var bizErr *servererr.BizError
	if errors.As(err, &bizErr) {
		httpResponse(ctx, nil, bizErr.Code.HTTPStatus(), bizErr.Code, bizErr.Message)
		return
	}
	httpResponse(ctx, nil, http.StatusInternalServerError, servererr.ErrInternal, err.Error())
}

// ErrorWithCode 指定错误码的错误响应
func ErrorWithCode(ctx *gin.Context, code servererr.ErrorCode, message string) {
	httpResponse(ctx, nil, code.HTTPStatus(), code, message)
}

// BadRequestError 400 错误
func BadRequestError(ctx *gin.Context, err error) {
	var bizErr *servererr.BizError
	if errors.As(err, &bizErr) {
		httpResponse(ctx, nil, bizErr.Code.HTTPStatus(), bizErr.Code, bizErr.Message)
		return
	}
	httpResponse(ctx, nil, http.StatusBadRequest, servererr.ErrBadRequest, errMsg(err))
}

// UnauthorizedError 401 错误
func UnauthorizedError(ctx *gin.Context, err error) {
	httpResponse(ctx, nil, http.StatusUnauthorized, servererr.ErrUnauthorized, errMsg(err))
}

// ForbiddenError 403 错误
func ForbiddenError(ctx *gin.Context, err error) {
	httpResponse(ctx, nil, http.StatusForbidden, servererr.ErrForbidden, errMsg(err))
}

// NotFoundError 404 错误
func NotFoundError(ctx *gin.Context, err error) {
	httpResponse(ctx, nil, http.StatusNotFound, servererr.ErrNotFound, errMsg(err))
}

// InternalServerError 500 错误
func InternalServerError(ctx *gin.Context, err error) {
	if err == nil {
		httpResponse(ctx, nil, http.StatusInternalServerError, servererr.ErrInternal, "unknown error")
		return
	}
	var bizErr *servererr.BizError
	if errors.As(err, &bizErr) {
		httpResponse(ctx, nil, bizErr.Code.HTTPStatus(), bizErr.Code, bizErr.Message)
		return
	}
	httpResponse(ctx, nil, http.StatusInternalServerError, servererr.ErrInternal, err.Error())
}

// TooManyRequestsError 429 错误
func TooManyRequestsError(ctx *gin.Context, err error) {
	httpResponse(ctx, nil, http.StatusTooManyRequests, servererr.ErrTooManyRequests, errMsg(err))
}

func errMsg(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}
