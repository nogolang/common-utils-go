package dataUtils

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func Id32Validate(idStr string) (int32, error) {
	if strings.TrimSpace(idStr) == "" {
		return 0, errors.Errorf("%s", "id不能为空")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return 0, errors.Errorf("%s", "无效的id")
	}
	return int32(id), nil
}

func Id64Validate(idStr string) (int64, error) {
	if strings.TrimSpace(idStr) == "" {
		return 0, errors.Errorf("%s", "id不能为空")
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.Errorf("%s", "无效的id")
	}
	return id, nil
}
