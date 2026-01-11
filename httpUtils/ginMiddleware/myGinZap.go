package ginMiddleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nogolang/common-utils-go/httpUtils/httpCodeUtils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// MyGinZap 自定义中间件
func MyGinZap(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		//取消当前的caller，因为这是放到中间件调用的，没必要用caller
		//即使显示caller，而是显示的中间件调用，没有意义
		newLogger := logger.WithOptions(zap.WithCaller(false))

		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		allRequestStr := method + " " + path
		if query != "" {
			allRequestStr = path + "?" + query
		}

		//取出body作为args
		//然后把内容转换为json
		var bodyArgs []byte
		if c.Request.ContentLength != 0 {
			bodyArgs, _ = io.ReadAll(c.Request.Body)

			//重新赋值
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyArgs))
		}

		//调用下一个中间件
		//如果调用到我们的方法，那么方法结束后，就会走后面的逻辑
		//如果下一个中间件，那么调用下一个中间件即可，如果中间件而返回错误，也会适用于这个log中间件
		//当前的log中间件是第1个放入的
		c.Next()

		latency := time.Since(start)

		//转换为rawJson，这样就不会有\符号
		//但是有些参数不是json，此时需要转换
		//比如支付宝的回调，它把base64编码直接放到body里了，且没有key
		var bodyArgsStr json.RawMessage
		valid := json.Valid(bodyArgs)
		if valid {
			bodyArgsStr = bodyArgs
		} else {
			marshal, _ := json.Marshal(map[string]interface{}{"content": bodyArgs})
			bodyArgsStr = marshal
		}

		fields := []zapcore.Field{
			zap.Int("status", c.Writer.Status()),
			zap.Any("args", bodyArgsStr),
			zap.String("ip", c.ClientIP()),
			//zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
		}

		//这里只当我们使用ctx.Error(err)把错误设置进去了，这里才会有值
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				//转成原始的error
				err := e.Err

				//把error转换到我们的错误，因为我们的Response实现了error接口的
				//如果是我们的错误，则返回我们的信息给前台
				var myResponse *httpCodeUtils.Response
				if errors.As(err, &myResponse) {
					//自定义错误都是返回前台的
					fields = append(fields, zap.String("code", myResponse.Code))
					fields = append(fields, zap.String("message", myResponse.Message))
					newLogger.Info(allRequestStr, fields...)
					//自定义错误都是返回前台的
					c.JSON(myResponse.Status, gin.H{
						"status":  myResponse.Status,
						"code":    myResponse.Code,
						"message": myResponse.Message,
					})
					return
				} else {
					//如果不是我们返回的错误，比如gorm的语句错误，或者其他的中间件的错误
					//  那就返回500，这里的错误信息最好别返回，而是打印日志即可，前台就显示简单的信息
					newLogger.Error(allRequestStr, fields...)
					newLogger.Sugar().Error(fmt.Sprintf("%+v", err))
					c.JSON(http.StatusInternalServerError, gin.H{
						"status":  http.StatusInternalServerError,
						"code":    "UnexpectedError",
						"message": "未知错误，请联系管理员",
					})
					return
				}
			}
		}

		//如果调用过程中没有产生error，则打印info，然后返回即可
		newLogger.Info(allRequestStr, fields...)
		return
	}
}
