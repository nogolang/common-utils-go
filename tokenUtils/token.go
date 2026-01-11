package tokenUtils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

// Claims声明
// 里面的内容都可以被解析出来，但是无法伪造
type TokenClaims struct {
	//需要在当前结构体里加入，这是固定的字段
	jwt.RegisteredClaims
	UserInfo map[string]interface{} `json:"userInfo" mapstructure:"userInfo"`
}

// 生成token
func GenToken(userInfo map[string]interface{}, secret string, expiredSecond int) (string, error) {
	//设置过期时间，签发人
	claims := &TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiredSecond) * time.Second)),
		},
		UserInfo: userInfo,
	}

	//指定算法和结构
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//再根据秘钥算出签名
	signedString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return signedString, nil
}
func ParseToken(token string, secret string) (*TokenClaims, error) {
	myToken, err := jwt.ParseWithClaims(token,
		&TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
	if err != nil {
		return nil, err
	}
	if claims, ok := myToken.Claims.(*TokenClaims); ok && myToken.Valid {
		return claims, nil
	}
	return nil, errors.Errorf("%s", "无效的token")
}
