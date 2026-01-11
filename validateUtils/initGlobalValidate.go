package validateUtils

import (
	"errors"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/go-playground/validator/v10/non-standard/validators"
)

// 自定义全局校验器,它不属于中间件
func InitGlobalValidator() error {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		//NotBlank
		err := v.RegisterValidation("NotBlank", validators.NotBlank)
		if err != nil {
			return errors.New("初始化全局校验器失败")
		}
	} else {
		return errors.New("初始化全局校验器失败")
	}
	return nil
}
