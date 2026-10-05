package logUtils

// LogConfig 日志配置
//
// 2026-09-26 配置解耦：从原 configUtils 下沉到本包。
type LogConfig struct {
	Level string `json:"level"`
	//日志实现，默认使用 zap
	Use string `json:"use"`
	//需要脱敏的字段
	HiddenField []string `json:"hiddenField"`
	//编码格式：console（默认，人读） / json（生产日志采集用）
	Encoder string `json:"encoder"`
	//输出位置：console（默认，全部输出到控制台） / file（all+error 分文件落盘，控制台只输出error，方便pod里查看，避免撑爆容器volume）
	Output string `json:"output"`
}
