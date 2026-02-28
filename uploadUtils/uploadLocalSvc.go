package uploadUtils

import (
	"os"
	"path"

	"github.com/nogolang/common-utils-go/fileUtils"
	"go.uber.org/zap"
)

type UploadLocalHandler struct {
	Logger *zap.Logger
}

func (receiver *UploadLocalHandler) save(pathName string, data []byte) error {
	//看看目录存不存在,不存在则要创建
	nowDir, err := os.Getwd()
	nowDir = path.Join(nowDir, "upload", pathName)
	dir := path.Dir(nowDir)
	exist, err := fileUtils.IsPathExist(dir)
	if err != nil {
		return err
	}
	if !exist {
		err := fileUtils.MakeDir(dir)
		if err != nil {
			return err
		}
	}
	file, _ := os.OpenFile(nowDir, os.O_CREATE, 0666)
	_, err = file.Write(data)
	defer file.Close()
	return err
}

func (receiver *UploadLocalHandler) delete(pathName string) error {
	err := os.Remove(pathName)
	if err != nil {
		return err
	}
	return nil
}

func (receiver *UploadLocalHandler) UploadLocal(fileName string, data []byte) (string, error) {
	fullPath := GetRandomFileName(fileName, ".jpg")
	err := receiver.save(fullPath, data)

	//返回给前台，则是url路径，不能用path处理
	//这里涉及到域名问题，暂时不处理，后续可以添加一个主域名选项
	//另外还有分布式问题，如果要分布式，则不能直接保存到本地，应该保存到其他服务器
	//host := "http://localhost:8001"
	//fullPath = host + "/upload" + "/" + fullPath
	return fullPath, err
}
