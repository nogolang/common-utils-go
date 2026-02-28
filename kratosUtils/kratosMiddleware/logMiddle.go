package kratosMiddleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/nogolang/common-utils-go/kratosUtils/kratosCodeUtils"
	pkgError "github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func LoggerServerMiddleware(logger *zap.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code                 int32
				reason               string
				message              string
				metadata             map[string]string
				kind                 string //操作种类，grpc还是http
				method               string //操作的方法
				kindAndMethodAndCode string //组合起来作为一个key
			)

			//从ctx里可以拿到一些基本信息
			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				method = info.Operation()
			} else {
				//下面大概率情况下是不会触发的
				//  FromServerContext只要在服务端使用一定就能拿到
				//  除非我们把这个服务端中间件放到了客户端使用
				newErr := pkgError.New("系统错误,无法从FromServerContext里拿到 kind和method")
				logger.Sugar().Errorf("%+v", newErr)
				//这个error会被返回个前台，需要注意，无需返回堆栈
				return reply, pkgError.Errorf("系统错误，检查后台日志")
			}
			startTime := time.Now()
			//调用下一个中间件
			reply, err = handler(ctx, req)

			marshal, marshalErr := json.Marshal(req)
			if marshalErr != nil {
				newErr := pkgError.New("系统错误，中间件中 Marshal出错")
				logger.Sugar().Errorf("%+v", newErr)
				return reply, pkgError.Errorf("系统错误，检查后台日志")
			}
			//不管是不是打印错误，都需要参数和延迟
			fields := []zapcore.Field{
				zap.Any("args", json.RawMessage(marshal)),
				zap.Duration("latency", time.Since(startTime)),
			}

			//解析出我们的错误，里面会自动判断是自定义的错误还是kratos的错误
			//最后都会转换到自定义的错误
			myError := kratosCodeUtils.FormError(err)
			if myError != nil {
				//这里status对应的是code，也就是http和grpc的状态码
				//而我们自定义的code对应的是reason
				code = int32(myError.Status)
				reason = myError.Code
				message = myError.Message
				metadata = myError.Metadata
				kindAndMethodAndCode = fmt.Sprintf("%s  %s  %d", kind, method, code)

				//如果不是额外的错误，而是我们用status.new（原生grpc）或者errors.new（kratos)创建的错误
				//  那么一定会被转换到kratos的错误
				if code != errors.UnknownCode {
					//一般来说，参数错误，业务错误，都应该直接返回出去，而非打印无意义的error日志
					//如果是那种无法预知的错误，我们才会打印error日志
					//参数错误这种，前台不弹框提示信息，但在前台控制台显示即可，方便人员排查
					//因为有人会恶意填其他参数，那我们就会打印很多无意义的error
					fields = append(fields,
						zap.Int32("status", code),
						zap.String("code", reason),
						zap.String("message", message),
						zap.Any("metadata", metadata),
					)
					logger.Info(kindAndMethodAndCode, fields...)
					//自定义错误直接返回回去，这里返回grpc的error
					//  我们调试，一路跟到grpc自己的fromError，kratos的error是实现了里面的GRPCStatus接口的
					//  所以grpc会自动转换
					return reply, err
				} else {
					//如果是其他错误，没有经过我们处理过的
					//  那就打印出来错误，并且打印堆栈，所以我们的这种第三方错误最用warp包裹出来
					//  这里一个是打印请求和方法，一个是打印堆栈
					//  但是返回给前台的，必须不能包含这些内部错误信息，所以用errorf返回自定义的
					logger.Error(kindAndMethodAndCode, fields...)
					logger.Sugar().Errorf("%+v", err)
					return reply, errors.New(http.StatusInternalServerError, "InternalServerError", "系统错误，检查后台日志")
				}
			}

			//如果没有error，那么打印 方法，参数，延迟即可
			//  这里直接赋予grpc的OK，如果是http调用，则会转换到200
			kindAndMethodAndCode = fmt.Sprintf("%s  %s  %d", kind, method, http.StatusOK)
			logger.Info(kindAndMethodAndCode, fields...)

			//返回值，下一个中间件调用的时候拿到的就是这个值
			return reply, err
		}
	}
}

// client和server中间件是差不多的
func LoggerClientMiddleware(logger *zap.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code                 int32
				reason               string
				message              string
				metadata             map[string]string
				kind                 string
				method               string
				kindAndMethodAndCode string
			)
			startTime := time.Now()

			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				method = info.Operation()
			} else {
				newErr := pkgError.New("系统错误,无法从FromClientContext里拿到 kind和method")
				logger.Sugar().Errorf("%+v", newErr)
				return reply, pkgError.Errorf("系统错误，检查后台日志")
			}

			//一般来说就是调用我们的远程方法了
			reply, err = handler(ctx, req)
			marshal, marshalErr := json.Marshal(req)
			if marshalErr != nil {
				newErr := pkgError.New("系统错误，中间件中 Marshal出错")
				logger.Sugar().Errorf("%+v", newErr)
				return reply, pkgError.Errorf("系统错误，检查后台日志")
			}
			//不管是不是打印错误，都需要参数和延迟
			fields := []zapcore.Field{
				zap.Any("args", json.RawMessage(marshal)),
				zap.Duration("latency", time.Since(startTime)),
			}

			//解析出我们的错误，里面会自动判断是自定义的错误还是kratos的错误
			//最后都会转换到自定义的错误
			myError := kratosCodeUtils.FormError(err)
			if myError != nil {
				//这里status对应的是code，也就是http和grpc的状态码
				//而我们自定义的code对应的是reason
				code = int32(myError.Status)
				reason = myError.Code
				message = myError.Message
				metadata = myError.Metadata
				kindAndMethodAndCode = fmt.Sprintf("[client] %s  %s  %d", kind, method, code)

				//如果不是额外的错误，而是我们用status.new（原生grpc）或者errors.new（kratos)创建的错误
				//  那么一定会被转换到kratos的错误
				if code != errors.UnknownCode {
					//一般来说，参数错误，业务错误，都应该直接返回出去，而非打印无意义的error日志
					//如果是那种无法预知的错误，我们才会打印error日志
					//参数错误这种，前台不弹框提示信息，但在前台控制台显示即可，方便人员排查
					//因为有人会恶意填其他参数，那我们就会打印很多无意义的error
					fields = append(fields,
						zap.Int32("status", code),
						zap.String("code", reason),
						zap.String("message", message),
						zap.Any("metadata", metadata),
					)
					logger.Info(kindAndMethodAndCode, fields...)
					//自定义错误直接返回回去
					return reply, err
				} else {
					logger.Error(kindAndMethodAndCode, fields...)
					logger.Sugar().Errorf("%+v", err)
					return reply, errors.New(http.StatusInternalServerError, "InternalServerError", "系统错误，检查后台日志")
				}
			}

			//如果没有error，那么打印 方法，参数，延迟即可
			//  这里直接赋予grpc的OK，如果是http调用，则会转换到200
			kindAndMethodAndCode = fmt.Sprintf("[client] %s  %s  %d", kind, method, http.StatusOK)
			logger.Info(kindAndMethodAndCode, fields...)

			//返回值，下一个中间件调用的时候拿到的就是这个值
			return reply, err
		}
	}
}
