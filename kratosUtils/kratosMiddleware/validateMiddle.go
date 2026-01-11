package kratosMiddleware

import (
	"context"
	"net/http"

	"buf.build/go/protovalidate"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/nogolang/common-utils-go/httpUtils/httpCodeUtils"
	"github.com/nogolang/common-utils-go/kratosUtils/kratosCodeUtils"
	"google.golang.org/protobuf/proto"
)

var (
	ParamInvalid = httpCodeUtils.NewResponse(http.StatusBadRequest, "ParamInvalid", "")
)

// 我们使用了buf的validate进行了校验，那么就写一个中间件，专门去校验
func BufValidateParamMiddle() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			message, ok := req.(proto.Message)
			if !ok {
				return nil, kratosCodeUtils.ToGrpcError(ParamInvalid.WithMessage("传递了非proto类型的参数"))
			}
			err := protovalidate.Validate(message)
			var e *protovalidate.ValidationError
			if errors.As(err, &e) {
				//这里一次返回一个错误信息回去即可
				v := e.ToProto().Violations[0]
				//并且一次只返回字段的的一个错误信息
				field := v.Field.Elements[0]
				return nil, kratosCodeUtils.ToGrpcError(ParamInvalid.WithMessage(*field.FieldName + ":" + *v.Message))
			}
			//校验成功，才可以调用我们的方法，这个应该放到最后一层去
			return handler(ctx, req)
		}
	}
}
