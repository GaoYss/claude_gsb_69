package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"streetlight/internal/apperr"
	"streetlight/internal/response"
)

// Recovery 捕获 panic, 记录堆栈并返回统一的 500 响应, 避免服务整体崩溃。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("请求处理发生 panic",
					"error", recovered,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"request_id", RequestIDFrom(c),
					"stack", string(debug.Stack()),
				)
				response.Fail(c, apperr.Internal("服务器内部错误, 请稍后重试"))
				c.Abort()
			}
		}()
		c.Next()
	}
}
