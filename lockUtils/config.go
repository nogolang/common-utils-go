package lockUtils

const (
	LockUse_Local = "local"
	LockUse_Redis = "redis"
)

// LockConfig 锁实现选择
type LockConfig struct {
	//取值 LockUse_Local / LockUse_Redis
	Use string `json:"use"`
}
