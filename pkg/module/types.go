package module

// Result 模块执行结果
type Result struct {
	Changed      bool                   `json:"changed"`
	Failed       bool                   `json:"failed,omitempty"`
	Unreachable  bool                   `json:"unreachable,omitempty"`
	Skipped      bool                   `json:"skipped,omitempty"`
	Msg          string                 `json:"msg,omitempty"`
	RC           int                    `json:"rc,omitempty"`
	Stdout       string                 `json:"stdout,omitempty"`
	Stderr       string                 `json:"stderr,omitempty"`
	Ping         string                 `json:"ping,omitempty"`          // ping 模块专用
	Dest         string                 `json:"dest,omitempty"`          // copy 模块目标路径
	Checksum     string                 `json:"checksum,omitempty"`      // copy 模块校验和
	AnsibleFacts map[string]interface{} `json:"ansible_facts,omitempty"` // set_fact 模块设置的 facts
	Data         map[string]interface{} `json:"-"`                       // 其他动态字段
}
