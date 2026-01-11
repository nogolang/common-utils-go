package ginMiddleware

//
/*
如果我们的程序里是返回的kratos的error，那么需要转换到自定义的error
  放到myGinZap中间件的前面，这样就可以无缝从kratos转到gin，从gin转到kratos http
  但是后续我们的程序里统一返回自定义的error，然后返回的时候再转换，所以无需这个了
*/
//func TransKratosError() gin.HandlerFunc {
//	return func(c *gin.Context) {
//		//先执行我们的方法，然后再处理error
//		//然后再交给ginZap中间件
//		c.Next()
//		if len(c.Errors) > 0 {
//			for i := 0; i < len(c.Errors); i++ {
//
//				//先判断是不是自定义的error，如果已经是自定义的了，那么就无需转换到kratos了
//				//因为control里会直接返回自定义的error，而service里可能返回的是kratos的error
//				var response *httpCodeUtils.Response
//				if errors.As(c.Errors[i].Err, &response) {
//					continue
//				}
//				kratosError := errors.FromError(c.Errors[i])
//				if kratosError != nil {
//					tempCode := int(kratosError.GetCode())
//					myError := httpCodeUtils.NewResponse(tempCode, kratosError.Reason, kratosError.Message,kratosError.Metadata)
//					c.Errors[i].Err = myError
//				}
//			}
//		}
//	}
//}
