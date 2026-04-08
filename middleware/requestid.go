package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"goflylivechat/tools"
)

const RequestIDContextKey = "request_id"

// RequestID 输入为空，输出为 Gin 中间件，目的在于为每个请求注入可追踪的 request id。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = tools.Uuid()
		}
		c.Set(RequestIDContextKey, requestID)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), RequestIDContextKey, requestID))
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}
