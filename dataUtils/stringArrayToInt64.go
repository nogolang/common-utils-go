package dataUtils

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
)

func StringArrayToInt64(arr []string) ([]int64, error) {
	var int64Arr []int64
	for _, v := range arr {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "str转换到int64转换失败")
		}
		int64Arr = append(int64Arr, i)
	}
	return int64Arr, nil
}
func StringArrayToInt32(arr []string) ([]int32, error) {
	var intArr []int32
	for _, v := range arr {
		i, err := strconv.Atoi(v)
		if err != nil {
			return nil, errors.Wrap(err, "str转换到int转换失败")
		}
		intArr = append(intArr, int32(i))
	}
	return intArr, nil
}

func Int64ToStringArray(arr []int64) []string {
	var strArr []string
	for _, v := range arr {
		strArr = append(strArr, fmt.Sprintf("%d", v))
	}
	return strArr
}
