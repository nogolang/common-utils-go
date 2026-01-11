package ginMiddleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DefaultHandleRecovery
// 如果发生了panic，那么就返回500，并打印日志
func DefaultHandleRecovery(logger *zap.Logger) func(*gin.Context, any) {
	return func(c *gin.Context, err any) {
		//这里使用全局的logger，无伤大雅
		logger.Error("系统出现panic", zap.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "发生异常，请联系管理员",
		})
		return
	}
}
