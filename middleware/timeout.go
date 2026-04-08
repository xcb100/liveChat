package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout 输入请求超时时长，输出为 Gin 中间件，目的在于为每个请求附带统一的超时上下文。
func Timeout(timeoutDuration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestContext, cancel := context.WithTimeout(c.Request.Context(), timeoutDuration)
		defer cancel()
		c.Request = c.Request.WithContext(requestContext)
		c.Next()
		if requestContext.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"code": http.StatusGatewayTimeout,
				"msg":  "请求处理超时",
			})
		}
	}
}
