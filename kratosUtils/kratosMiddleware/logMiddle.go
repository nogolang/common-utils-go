package kratosMiddleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/nogolang/common-utils-go/kratosUtils/kratosCodeUtils"
	pkgError "github.com/pkg/errors"
)

func LoggerServerMiddleware(logger *slog.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code                 int32
				reason               string
				message              string
				metadata             map[string]string
				kind                 string // 操作种类，grpc还是http
				method               string // 操作的方法
				kindAndMethodAndCode string // 组合起来作为一个key
			)

			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				method = info.Operation()
			} else {
				newErr := pkgError.New("系统错误,无法从FromServerContext里拿到 kind和method")
				logger.ErrorContext(ctx, "获取服务端请求信息失败", "err", fmt.Sprintf("%+v", newErr))
				return reply, pkgError.Errorf("系统错误，检查后台日志")
			}

			startTime := time.Now()
			reply, err = handler(ctx, req)

			marshal, marshalErr := json.Marshal(req)
			if marshalErr != nil {
				newErr := pkgError.New("系统错误，中间件中 Marshal出错")
				logger.ErrorContext(ctx, "序列化请求参数失败", "err", fmt.Sprintf("%+v", newErr))
				return reply, pkgError.Errorf("系统错误，检查后台日志")
			}

			fields := []any{
				"args", json.RawMessage(marshal),
				"latency", time.Since(startTime),
			}

			myError := kratosCodeUtils.FormError(err)
			if myError != nil {
				code = int32(myError.Status)
				reason = myError.Code
				message = myError.Message
				metadata = myError.Metadata
				kindAndMethodAndCode = fmt.Sprintf("%s  %s  %d", kind, method, code)

				if code != errors.UnknownCode {
					fields = append(fields,
						"status", code,
						"code", reason,
						"message", message,
						"metadata", metadata,
					)
					logger.InfoContext(ctx, kindAndMethodAndCode, fields...)
					return reply, err
				}

				fields = append(fields, "err", fmt.Sprintf("%+v", err))
				logger.ErrorContext(ctx, kindAndMethodAndCode, fields...)
				return reply, errors.New(http.StatusInternalServerError, "InternalServerError", "系统错误，检查后台日志")
			}

			kindAndMethodAndCode = fmt.Sprintf("%s  %s  %d", kind, method, http.StatusOK)
			logger.InfoContext(ctx, kindAndMethodAndCode, fields...)
			return reply, err
		}
	}
}

// client和server中间件是差不多的
func LoggerClientMiddleware(logger *slog.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code                 int32
				reason               string
				message              string
				metadata             map[string]string
				endpoint             string
				kind                 string
				method               string
				kindAndMethodAndCode string
			)
			startTime := time.Now()

			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				endpoint = info.Endpoint()
				method = info.Operation()
			} else {
				newErr := pkgError.New("系统错误,无法从FromClientContext里拿到 kind和method")
				logger.ErrorContext(ctx, "获取客户端请求信息失败", "err", fmt.Sprintf("%+v", newErr))
				return reply, pkgError.Errorf("系统错误，检查后台日志")
			}

			reply, err = handler(ctx, req)
			marshal, marshalErr := json.Marshal(req)
			if marshalErr != nil {
				newErr := pkgError.New("系统错误，中间件中 Marshal出错")
				logger.ErrorContext(ctx, "序列化请求参数失败", "err", fmt.Sprintf("%+v", newErr))
				return reply, pkgError.Errorf("系统错误，检查后台日志")
			}

			fields := []any{
				"args", json.RawMessage(marshal),
				"latency", time.Since(startTime),
			}

			myError := kratosCodeUtils.FormError(err)
			if myError != nil {
				code = int32(myError.Status)
				reason = myError.Code
				message = myError.Message
				metadata = myError.Metadata
				kindAndMethodAndCode = fmt.Sprintf("[client] %s  %s%s  %d", kind, endpoint, method, code)

				if code != errors.UnknownCode && code != 504 {
					fields = append(fields,
						"status", code,
						"code", reason,
						"message", message,
						"metadata", metadata,
					)
					logger.InfoContext(ctx, kindAndMethodAndCode, fields...)
					return reply, err
				}

				fields = append(fields, "err", fmt.Sprintf("%+v", err))
				logger.ErrorContext(ctx, kindAndMethodAndCode, fields...)
				return reply, errors.New(http.StatusInternalServerError, "InternalServerError", "系统错误，检查后台日志")
			}

			kindAndMethodAndCode = fmt.Sprintf("[client] %s  %s  %d", kind, method, http.StatusOK)
			logger.InfoContext(ctx, kindAndMethodAndCode, fields...)
			return reply, err
		}
	}
}
