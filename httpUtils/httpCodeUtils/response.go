package httpCodeUtils

import "fmt"

//
/*
如果代码本身出错，那么应该打印到日志里，而非返回给前台
状态码200的统一返回给用户，比如有要找的用户不存在，同时会有一个错误码，
  错误码用英文表示，还有错误信息，这是口语化的错误
  200的情况下，错误码如果是OK，那么代表是提示给用户的，如果不是，则打印到前台日志里
不是200状态码的，打印到日志里
*/
type Response struct {
	Status int `json:"status"`

	//错误码，比如ParamsInvalid,也可以叫reason,最好不要用数字
	Code string `json:"code"`

	//错误的信息，我们口语化的错误，参数错误
	Message string `json:"message"`

	//返回的数据，在http里可以这样做，但是在grpc里不能这么做
	Data interface{} `json:"data"`

	//附加数据，用于grpc的附加数据
	Metadata map[string]string `json:"metadata"`
}

// 实现error接口，返回给error
func (receiver *Response) Error() string {
	return fmt.Sprintf("status=%d ", receiver.Status) +
		fmt.Sprintf("code=%s ", receiver.Code) +
		fmt.Sprintf("message=%s ", receiver.Message) +
		fmt.Sprintf("metadata=%+v ", receiver.Metadata)
}

func (receiver *Response) WithData(data interface{}) *Response {
	response := NewResponse(receiver.Status, receiver.Code, receiver.Message)
	response.Data = data
	return response
}

func (receiver *Response) WithMessage(newMessage string) *Response {
	response := NewResponse(receiver.Status, receiver.Code, newMessage)
	return response
}

func (receiver *Response) WithMetadata(metadata map[string]string) *Response {
	response := NewResponse(receiver.Status, receiver.Code, receiver.Message)
	response.Metadata = metadata
	return response
}

func NewResponse(status int, code string, message string) *Response {
	return &Response{
		Status:  status,
		Code:    code,
		Message: message,
	}
}
