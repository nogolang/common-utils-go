package encryptUtils

import (
	"crypto/md5"
	"encoding/hex"
)

func Md5(str string) string {
	sum := md5.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}
