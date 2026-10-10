// Package money 金额换算。
//
// 场景：本地账本和微信一律用「分」，支付宝接口一律用「元」字符串。
// 后果：直接 ParseFloat 再乘 100 会掉精度（0.29 * 100 = 28.999999...），退款金额差一分就对不上账。
// 修法：按小数点切开，整数部分和小数部分分别用 ParseInt，全程不碰浮点。
package money

import (
	"fmt"
	"strconv"
	"strings"
)

// YuanToCents 把最多两位小数的「元」字符串换成分。
// 支付宝查单、退款响应里的 total_amount / refund_amount 都是这种字符串。
func YuanToCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parts := strings.SplitN(value, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	if len(parts) == 1 {
		return whole * 100, nil
	}
	fraction := parts[1]
	if len(fraction) > 2 {
		return 0, fmt.Errorf("最多支持两位小数: %q", value)
	}
	if len(fraction) == 1 {
		fraction += "0"
	}
	decimal, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	return whole*100 + decimal, nil
}

// YuanFloatToCents 把 HTTP 入参里的 float 元换成分。
//
// 场景：/alipay/refund 的 refundAmount 是 float 元（支付宝口径），内部要转成分再走统一退款。
// 后果：悄悄四舍五入会退错钱，用户申请退 0.005 元结果退了 0.01 元。
// 修法：先按最短表示还原字符串，超过两位小数直接报错，让调用方返回 400。
func YuanFloatToCents(value float64) (int64, error) {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if index := strings.IndexByte(text, '.'); index >= 0 && len(text)-index-1 > 2 {
		return 0, fmt.Errorf("金额最多支持两位小数: %v", value)
	}
	return YuanToCents(text)
}

// CentsToYuan 分 → 支付宝要的元字符串，固定两位小数。
func CentsToYuan(amountCents int64) string {
	return fmt.Sprintf("%d.%02d", amountCents/100, amountCents%100)
}
