package kratosCodeUtils

import (
	rawError "errors"
	"log"
	"net/http"

	kratosError "github.com/go-kratos/kratos/v2/errors"
	"github.com/nogolang/common-utils-go/httpUtils/httpCodeUtils"
)

//
/*
返回grpc的error，我们自己的错误，return回去的时候，需要转换到grpc的error
  但是grpc的error构建不太友好，我们可以使用kratos的error，它实现了GRPCStatus接口
  当grpc源码里执行grpcStatus := gs.GRPCStatus()这个方法的时候
  会调用实现的接口，然后把kratos的error转换到grpc能识别的error
然后在client，我们再用kratos提供的方法，先转换到kratos的结构（方便取值）
  然后又可以转换到我们自己的错误
*/
func ToGrpcError(err error) error {
	if err == nil {
		return nil
	}

	//如果是自定义的error，则封装的到grpc能识别的error即可
	//  kratos已经给我们写好了，我们直接使用即可
	//  但是候返回给客户端的时候，我们需要用FormGrpcToMy来解析
	var response *httpCodeUtils.Response
	if rawError.As(err, &response) {
		//不能在状态码为200的状况下还返回错误信息
		//  grpc在源码里做了处理，如果状态码是ok，并且还返回错误，那么会直接把error变为nil
		//  在grpc fromError方法的withDetails里有写，一路跟过去就能看到
		//所以我们把400状态统一定义为返回给前台的错误信息
		//  这个错误信息，不会记录到日志里，一般是参数错误，或者用户刻意传错误参数
		if response.Status == http.StatusOK {
			log.Fatal("不能在状态码为200的状况下还返回错误信息")
			return nil
		}
		return kratosError.New(response.Status, response.Code, response.Message).WithMetadata(response.Metadata)
	}

	//如果是kratos的error或者其他500错误，那么直接返回即可
	return err
}

// 解析出自定义的error
func FormError(err error) *httpCodeUtils.Response {
	if err == nil {
		return nil
	}
	//如果error是自定义的error，那么就直接返回
	//  因为可能会把自定义的error传递过来，日志的地方会这样使用
	var response *httpCodeUtils.Response
	if rawError.As(err, &response) {
		return response
	}
	//如果返回的错误是grpc的error
	//  那么还需要转换一下，这里kratos直接提供了，即使不用kratos也可以用这个
	fromError := kratosError.FromError(err)
	response = httpCodeUtils.NewResponse(
		int(fromError.GetCode()),
		fromError.GetReason(),
		fromError.GetMessage(),
	).WithMetadata(fromError.GetMetadata())
	return response
}

func KratosErrorWithMessage(err *kratosError.Error, newMessage string) error {
	clone := kratosError.Clone(err)
	clone.Message = newMessage
	return clone
}
