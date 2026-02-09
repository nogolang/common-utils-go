package grpcUtils

import (
	rawError "errors"

	kratosError "github.com/go-kratos/kratos/v2/errors"
	"github.com/nogolang/common-utils-go/httpUtils/httpCodeUtils"
)

// 解析出自定义的error
func FormGrpcErrorToMy(err error) *httpCodeUtils.Response {
	if err == nil {
		return nil
	}
	//如果error是自定义的error，那么就直接返回
	//  因为可能会把自定义的error传递过来，日志的地方会这样使用
	var response *httpCodeUtils.Response
	if rawError.As(err, &response) {
		return response
	}

	//如果返回的错误是kratos的error，或者grpc的error
	//  那么还需要转换一下，这里kratos直接提供了，即使不用kratos也可以用这个
	//  因为这是对于grpc的转换，转换到kratos自己的error，但是我们最终是要转换到我们的error的
	//  所以不用管中间层的转换
	//但是我们最终要转换到自己的error
	fromError := kratosError.FromError(err)
	response = httpCodeUtils.NewResponse(
		int(fromError.GetCode()),
		fromError.GetReason(),
		fromError.GetMessage(),
	).WithMetadata(fromError.GetMetadata())
	return response
}
