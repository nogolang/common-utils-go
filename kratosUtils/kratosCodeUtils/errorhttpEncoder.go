package kratosCodeUtils

import (
	"context"
	"encoding/json"
	rawHttp "net/http"

	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/nogolang/common-utils-go/httpUtils/httpCodeUtils"
)

// 定义kratos http返回给前台时候的数值，我们要包装一层data
// 这样做是为了契合gin版本，这样前台无需任何变动
var HttpResponseEncoder = func(w rawHttp.ResponseWriter, r *rawHttp.Request, v interface{}) error {
	// 通过Request Header的Accept中提取出对应的编码器
	// 如果找不到则忽略报错，并使用默认json编码器
	codec, _ := kratosHttp.CodecForRequest(r, "Accept")
	data, err := codec.Marshal(v)
	if err != nil {
		return err
	}
	//封装一层data
	body, err := json.Marshal(httpCodeUtils.NewResponse(rawHttp.StatusOK, "OK", "").WithData(json.RawMessage(data)))
	if err != nil {
		return err
	}

	// 在Response Header中写入编码器的scheme
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
	return nil
}

// 定义kratos http的错误响应编码器
var ServerHttpErrorEncoder = func(w rawHttp.ResponseWriter, r *rawHttp.Request, err error) {
	//这里要转换到我们自己的错误，因为是grpc转换到http
	myError := FormError(err)
	w.Header().Set("Content-Type", "application/json")
	// 设置HTTP Status Code
	w.WriteHeader(myError.Status)

	body, err := json.Marshal(myError)
	if err != nil {
		return
	}
	w.Write(body)
}

// kratos http作为客户端请求的时候，会从body里解析出错误
// 但我们是自定义的错误，所以要自己解析
var ClintErrorEncoder = func(ctx context.Context, res *rawHttp.Response) error {
	return nil
}
