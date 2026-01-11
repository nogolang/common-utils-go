package uploadUtils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// GetRandomFileName type主要是类型，比如brand品牌上传的，比如user用户上传的
// 获取到随机名称即可,确保不会重复，不然会覆盖掉
func GetRandomFileName(prefix string, ext string) string {
	//后缀名称
	hashName := getHashName(fmt.Sprintf("%d", rand.Int32N(100000)))
	newUUID, _ := uuid.NewUUID()
	name := md5.Sum([]byte(newUUID.String()))
	fileName := prefix + "/" + hashName + "/" + hex.EncodeToString(name[:]) + ext
	return fileName
}

func getHashName(filename string) string {
	// 第一层哈希
	hash1 := md5.Sum([]byte(filename))
	layer1 := fmt.Sprintf("%x", hash1)[:3]

	// 第二层哈希（对第一层结果再哈希）
	//hash2 := md5.Sum([]byte(layer1))
	//layer2 := fmt.Sprintf("%x", hash2)[:3]

	//return layer1 + "/" + layer2
	return layer1
}

// mb字符串转换到字节数
func MbStrToByteInt64(mbStr string) int64 {
	index := strings.Index(mbStr, "MB")
	num, err := strconv.Atoi(mbStr[:index])
	if err != nil {
		log.Println("转换失败", err)
		return 0
	}
	return int64(num) * 1024 * 1024
}

func KbStrToByteInt64(kbStr string) int64 {
	index := strings.Index(kbStr, "KB")
	num, err := strconv.Atoi(kbStr[:index])
	if err != nil {
		log.Println("转换失败", err)
		return 0
	}
	return int64(num) * 1024
}
