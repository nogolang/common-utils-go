package kratosCodeUtils

import (
	rawError "errors"

	kratosError "github.com/go-kratos/kratos/v2/errors"
	"github.com/nogolang/common-utils-go/httpUtils/httpCodeUtils"
)

// 解析出自定义的error，用于打印日志
func FormErrorMy(err error) *httpCodeUtils.Response {
	if err == nil {
		return nil
	}
	//如果grpc直接返回的自定义code，那么就直接返回
	var response *httpCodeUtils.Response
	if rawError.As(err, &response) {
		return response
	}

	//如果返回的错误是kratos的error，或者grpc的error
	//  那么还需要转换一下，这里kratos直接提供了
	//  但是我们最终要转换到自己的erro
	//  这里暂时没有提供metadata之类的
	fromError := kratosError.FromError(err)
	response = httpCodeUtils.NewResponse(
		int(fromError.GetCode()),
		fromError.GetReason(),
		fromError.GetMessage(),
	).WithMetadata(fromError.GetMetadata())
	return response
}

// 返回grpc的error，我们自己的错误，return回去的时候，需要转换到grpc的error
// 这里我们无需再手动处理，因为kratos已经写好了，我们再封装一层即可
func ToGrpcError(err error) error {
	if err == nil {
		return nil
	}

	//如果是自定义的error，则封装的到grpc能识别的error即可
	//  kratos已经给我们写好了，我们直接使用即可
	var response *httpCodeUtils.Response
	if rawError.As(err, &response) {
		return kratosError.New(response.Status, response.Code, response.Message).WithMetadata(response.Metadata)
	}

	//如果是kratos的error，那么直接返回即可
	//如果是其他错误，那我们的log中间件里也会处理，处理成500，如果不处理，kratos的GRPCStatus也会处理
	return err
}

func KratosErrorWithMessage(err *kratosError.Error, newMessage string) error {
	clone := kratosError.Clone(err)
	clone.Message = newMessage
	return clone
}
