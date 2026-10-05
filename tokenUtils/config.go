package tokenUtils

// JwtConfig token 签发/校验配置
//
// 2026-09-26 两次调整：
//  1. 从业务侧（shop-common/commonConf）下沉到本包——「谁消费谁声明」，配置模型不该
//     寄居在要求所有引入方迁就它的公共库里。
//  2. **取消「secret + adminSecret」双密钥**。原先是"一把用户密钥 + 一把管理端密钥"，
//     但实际只有签发端自己知道该用哪把：每个服务（admin/business/user）各自持有**自己的**
//     `jwt.secret`（字段名相同、值互不相同），只有鉴权网关 shop-auth 需要全量持有用于校验。
//     三端共用一把密钥的真正问题不是"谁在用"，而是**网关分不出 token 是哪端签的**，
//     也就无法断言"这端签的 token 不许冒充别的 role"。
//
// 注意本包的 GenToken/ParseToken 仍是纯函数（secret 由调用方传参），
// 本结构体只是**配置模型**，不改变函数签名。
type JwtConfig struct {
	// Secret **本端**签发用的签名密钥。鉴权网关（shop-auth）不签发 token，此项留空，
	// 只用下面的 VerifyKeys 校验。
	Secret string `json:"secret"`
	// Role 本端签发的 token 声明的角色（admin / business / user）
	Role string `json:"role"`
	// Expired 过期时间（秒）
	Expired int      `json:"expired"`
	PassUrl []string `json:"passUrl"`

	// VerifyKeys 鉴权网关专用：各端签发密钥 + 它们签出来的 token 允许的角色
	VerifyKeys []VerifyKey `json:"verifyKeys"`
}

// VerifyKey 一把校验密钥
type VerifyKey struct {
	// Name 端标识（admin / user / business），仅用于日志与报错文案
	Name string `json:"name"`
	// Secret 该端的签名密钥
	Secret string `json:"secret"`
	// Roles 该密钥签发的 token **允许**的角色；空 = 不做来源限制。
	//
	// 为什么必须限制：GenToken 的 userInfo 是调用方自选的 map，role 只是其中一个字段。
	// 持有 user 密钥的人可以自己签一个 role=admin 的 token —— 若网关只按 URL 前缀判角色
	// （/admin/* 要 admin），这个伪造 token 就能进管理端。绑定「密钥 → 允许角色」才补上这条。
	Roles []string `json:"roles"`
}
