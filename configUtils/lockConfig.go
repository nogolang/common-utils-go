package configUtils

const (
	LockUse_Local = "local"
	LockUse_Redis = "redis"
)

type LockConfig struct {
	Use string `json:"use"`
}
