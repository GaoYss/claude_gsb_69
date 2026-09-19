package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// 业务错误码, 前后端共用同一套取值。
const (
	CodeInvalidArgument = "INVALID_ARGUMENT"
	CodeNotFound        = "NOT_FOUND"
	CodeConflict        = "CONFLICT"
	CodeInternalError   = "INTERNAL_ERROR"
)

// Error 是可以直接映射为 HTTP 响应的业务错误。
type Error struct {
	Status  int
	Code    string
	Message string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 支持通过 errors.Is / errors.As 追溯底层错误。
func (e *Error) Unwrap() error { return e.cause }

// WithCause 附带底层错误, 仅用于日志排查, 不会返回给客户端。
func (e *Error) WithCause(err error) *Error {
	clone := *e
	clone.cause = err
	return &clone
}

// New 构造业务错误。
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// BadRequest 参数错误(400)。
func BadRequest(format string, args ...any) *Error {
	return New(http.StatusBadRequest, CodeInvalidArgument, fmt.Sprintf(format, args...))
}

// NotFound 资源不存在(404)。
func NotFound(format string, args ...any) *Error {
	return New(http.StatusNotFound, CodeNotFound, fmt.Sprintf(format, args...))
}

// Conflict 业务状态冲突(409)。
func Conflict(format string, args ...any) *Error {
	return New(http.StatusConflict, CodeConflict, fmt.Sprintf(format, args...))
}

// Internal 服务端内部错误(500)。
func Internal(format string, args ...any) *Error {
	return New(http.StatusInternalServerError, CodeInternalError, fmt.Sprintf(format, args...))
}

// As 从任意 error 中提取业务错误。
func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}
