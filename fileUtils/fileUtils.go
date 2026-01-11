package fileUtils

import (
	"os"
)

func IsPathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func MakeDir(dirName string) error {
	//755是其它用户无法写入，可以访问，第1个7是本用户，第2个是5是用户所在组的其它用户权限，第3个是5是其它组的其它用户
	err := os.MkdirAll(dirName, 0755)
	return err
}
