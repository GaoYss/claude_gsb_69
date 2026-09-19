package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger 输出结构化访问日志, 并按状态码分级。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		if query != "" {
			path = path + "?" + query
		}
		attrs := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"request_id", RequestIDFrom(c),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "error", c.Errors.String())
		}

		switch {
		case c.Writer.Status() >= 500:
			slog.Error("请求处理失败", attrs...)
		case c.Writer.Status() >= 400:
			slog.Warn("请求被拒绝", attrs...)
		default:
			slog.Info("请求处理完成", attrs...)
		}
	}
}
