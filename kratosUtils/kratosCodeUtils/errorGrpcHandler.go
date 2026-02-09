package kratosCodeUtils

import (
	rawError "errors"
	"log"
	"net/http"

	kratosError "github.com/go-kratos/kratos/v2/errors"
	"github.com/nogolang/common-utils-go/httpUtils/httpCodeUtils"
)

// 返回grpc的error，我们自己的错误，return回去的时候，需要转换到grpc的error
// 这里我们无需再手动处理，因为kratos已经写好了，我们再封装一层即可
func ToGrpcError(err error) error {
	if err == nil {
		return nil
	}

	//如果是自定义的error，则封装的到grpc能识别的error即可
	//  kratos已经给我们写好了，我们直接使用即可
	//  但是但时候返回给客户端的时候，我们需要用FormGrpcToMy来解析
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

	//如果是kratos的error或者其他错误，那么直接返回即可
	return err
}

func KratosErrorWithMessage(err *kratosError.Error, newMessage string) error {
	clone := kratosError.Clone(err)
	clone.Message = newMessage
	return clone
}
