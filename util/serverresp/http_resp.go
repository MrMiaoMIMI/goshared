package serverresp

import (
	"errors"
	"net/http"

	"github.com/MrMiaoMIMI/goshared/util/servererr"
	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    servererr.ErrorCode `json:"code"`           // 状态码: 0=成功, 非0=错误码
	Message string              `json:"message"`        // 响应消息
	Data    any                 `json:"data,omitempty"` // 业务数据
}

func httpResponse(ctx *gin.Context, data any, httpStatus int, code servererr.ErrorCode, message string) {
	ctx.JSON(httpStatus, Response{Code: code, Message: message, Data: data})
}

func Success(ctx *gin.Context, data any) {
	httpResponse(ctx, data, http.StatusOK, servererr.Success, "success")
}

func BadRequestError(ctx *gin.Context, err error) {
	var bizErr *servererr.BizError
	if errors.As(err, &bizErr) {
		httpResponse(ctx, nil, bizErr.Code.HTTPStatus(), bizErr.Code, bizErr.Message)
		return
	}
	httpResponse(ctx, nil, http.StatusBadRequest, servererr.ErrBadRequest, err.Error())
}

func UnauthorizedError(ctx *gin.Context, err error) {
	httpResponse(ctx, nil, http.StatusUnauthorized, servererr.ErrUnauthorized, err.Error())
}

func ForbiddenError(ctx *gin.Context, err error) {
	httpResponse(ctx, nil, http.StatusForbidden, servererr.ErrForbidden, err.Error())
}

func NotFoundError(ctx *gin.Context, err error) {
	httpResponse(ctx, nil, http.StatusNotFound, servererr.ErrNotFound, err.Error())
}

func InternalServerError(ctx *gin.Context, err error) {
	var bizErr *servererr.BizError
	if errors.As(err, &bizErr) {
		httpResponse(ctx, nil, bizErr.Code.HTTPStatus(), bizErr.Code, bizErr.Message)
		return
	}
	httpResponse(ctx, nil, http.StatusInternalServerError, servererr.ErrInternal, err.Error())
}
