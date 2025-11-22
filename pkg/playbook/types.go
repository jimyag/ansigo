package playbook

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Playbook 代表一个 Ansible Playbook
type Playbook []Play

// Play 代表 Playbook 中的一个 play
type Play struct {
	Name         string                 `yaml:"name"`
	Hosts        string                 `yaml:"hosts"`
	GatherFacts  bool                   `yaml:"gather_facts"`
	Vars         map[string]interface{} `yaml:"vars"`
	Roles        []interface{}          `yaml:"roles"` // 可以是字符串或字典
	Tasks        []Task                 `yaml:"tasks"`
	Handlers     []Handler              `yaml:"handlers"`
	Become       bool                   `yaml:"become"`        // Play 级别权限提升
	BecomeUser   string                 `yaml:"become_user"`   // 切换到的用户（默认 root）
	BecomeMethod string                 `yaml:"become_method"` // 提权方法（默认 sudo）
}

// Role 代表一个 Ansible Role
type Role struct {
	Name     string                 // Role 名称
	Path     string                 // Role 路径
	Vars     map[string]interface{} // Role 变量
	Defaults map[string]interface{} // 默认变量
	Tasks    []Task                 // 任务列表
	Handlers []Handler              // Handler 列表
}

// RoleSpec 代表 Role 引用（可以是字符串或带参数的字典）
type RoleSpec struct {
	Name string                 // Role 名称
	Vars map[string]interface{} // 传递给 role 的变量
}

// LoopControl 循环控制选项
type LoopControl struct {
	LoopVar  string `yaml:"loop_var"`  // 自定义循环变量名（默认 item）
	IndexVar string `yaml:"index_var"` // 循环索引变量名
	Label    string `yaml:"label"`     // 简化输出显示
	Pause    int    `yaml:"pause"`     // 循环迭代之间暂停（秒）
}

// Block 代表任务块（用于错误处理）
type Block struct {
	Block  []Task // 主任务列表
	Rescue []Task // 错误恢复任务
	Always []Task // 总是执行的任务
}

// Task 代表一个任务
type Task struct {
	Name         string
	Module       string
	ModuleArgs   map[string]interface{}
	Register     string
	When         string
	FailedWhen   string
	ChangedWhen  string
	IgnoreErrors bool
	Notify       []string      // 通知的 handler 名称列表
	Loop         []interface{} // 循环列表
	LoopControl  *LoopControl  // 循环控制选项
	TaskBlock    *Block        // Block 结构（如果是 block 任务）
	Become       *bool         // Task 级别权限提升（指针以区分未设置和 false）
	BecomeUser   string        // 切换到的用户
	BecomeMethod string        // 提权方法
}

// Handler 代表一个 handler（本质是特殊的任务）
type Handler struct {
	Name         string
	Listen       string // 可选，监听的 topic
	Module       string
	ModuleArgs   map[string]interface{}
	When         string
	IgnoreErrors bool
}

// UnmarshalYAML 自定义 Task 的 YAML 解析
func (t *Task) UnmarshalYAML(value *yaml.Node) error {
	// 使用辅助结构解析已知字段
	type TaskFields struct {
		Name         string       `yaml:"name"`
		Register     string       `yaml:"register"`
		When         string       `yaml:"when"`
		FailedWhen   string       `yaml:"failed_when"`
		ChangedWhen  string       `yaml:"changed_when"`
		IgnoreErrors bool         `yaml:"ignore_errors"`
		Notify       interface{}  `yaml:"notify"`        // 可以是字符串或列表
		Loop         interface{}  `yaml:"loop"`          // 循环列表（可以是列表或模板字符串）
		LoopControl  *LoopControl `yaml:"loop_control"`  // 循环控制
		Block        []Task       `yaml:"block"`         // Block 任务列表
		Rescue       []Task       `yaml:"rescue"`        // Rescue 任务列表
		Always       []Task       `yaml:"always"`        // Always 任务列表
		Become       *bool        `yaml:"become"`        // 权限提升
		BecomeUser   string       `yaml:"become_user"`   // 切换用户
		BecomeMethod string       `yaml:"become_method"` // 提权方法
	}

	var fields TaskFields
	if err := value.Decode(&fields); err != nil {
		return err
	}

	t.Name = fields.Name
	t.Register = fields.Register
	t.When = fields.When
	t.FailedWhen = fields.FailedWhen
	t.ChangedWhen = fields.ChangedWhen
	t.IgnoreErrors = fields.IgnoreErrors
	t.LoopControl = fields.LoopControl
	t.Become = fields.Become
	t.BecomeUser = fields.BecomeUser
	t.BecomeMethod = fields.BecomeMethod
	t.ModuleArgs = make(map[string]interface{})

	// 检查是否是 block 任务
	if len(fields.Block) > 0 {
		t.TaskBlock = &Block{
			Block:  fields.Block,
			Rescue: fields.Rescue,
			Always: fields.Always,
		}
		// Block 任务不需要 Module
		return nil
	}

	// 解析 loop 字段（可以是列表或模板字符串）
	if fields.Loop != nil {
		switch l := fields.Loop.(type) {
		case []interface{}:
			t.Loop = l
		case string:
			// 如果是字符串（如 "{{ packages }}"），存储为单元素列表
			t.Loop = []interface{}{l}
		default:
			// 其他类型也转为列表
			t.Loop = []interface{}{l}
		}
	}

	// 解析 notify 字段（支持字符串或列表）
	if fields.Notify != nil {
		switch n := fields.Notify.(type) {
		case string:
			t.Notify = []string{n}
		case []interface{}:
			t.Notify = make([]string, len(n))
			for i, v := range n {
				if s, ok := v.(string); ok {
					t.Notify[i] = s
				}
			}
		}
	}

	// 已知的标准字段
	knownFields := map[string]bool{
		"name":          true,
		"register":      true,
		"when":          true,
		"failed_when":   true,
		"changed_when":  true,
		"ignore_errors": true,
		"notify":        true,
		"loop":          true,
		"loop_control":  true,
		"block":         true,
		"rescue":        true,
		"always":        true,
		"become":        true,
		"become_user":   true,
		"become_method": true,
	}

	// 已知的模块列表
	knownModules := map[string]bool{
		"ping":                         true,
		"command":                      true,
		"shell":                        true,
		"raw":                          true,
		"copy":                         true,
		"debug":                        true,
		"set_fact":                     true,
		"file":                         true,
		"template":                     true,
		"lineinfile":                   true,
		"service":                      true,
		"ansible.builtin.import_tasks": true,
		"import_tasks":                 true,
		"ansible.builtin.include_role": true,
		"include_role":                 true,
	}

	// 遍历所有字段，查找模块名
	if value.Kind == yaml.MappingNode {
		for i := 0; i < len(value.Content); i += 2 {
			keyNode := value.Content[i]
			valueNode := value.Content[i+1]

			key := keyNode.Value

			// 跳过已知字段
			if knownFields[key] {
				continue
			}

			// 检查是否是模块
			if knownModules[key] {
				t.Module = key

				// 解析模块参数
				switch valueNode.Kind {
				case yaml.ScalarNode:
					// 短格式: command: uptime
					if valueNode.Value != "" {
						t.ModuleArgs["_raw_params"] = valueNode.Value
					}
				case yaml.MappingNode:
					// 长格式: command: {cmd: uptime}
					var args map[string]interface{}
					if err := valueNode.Decode(&args); err != nil {
						return fmt.Errorf("failed to parse module args: %w", err)
					}
					t.ModuleArgs = args
				default:
					return fmt.Errorf("unsupported module args format for module %s", key)
				}
				break
			}
		}
	}

	if t.Module == "" {
		return fmt.Errorf("no module found in task: %s", t.Name)
	}

	return nil
}

// UnmarshalYAML 自定义 Handler 的 YAML 解析
func (h *Handler) UnmarshalYAML(value *yaml.Node) error {
	// 使用辅助结构解析已知字段
	type HandlerFields struct {
		Name         string `yaml:"name"`
		Listen       string `yaml:"listen"`
		When         string `yaml:"when"`
		IgnoreErrors bool   `yaml:"ignore_errors"`
	}

	var fields HandlerFields
	if err := value.Decode(&fields); err != nil {
		return err
	}

	h.Name = fields.Name
	h.Listen = fields.Listen
	h.When = fields.When
	h.IgnoreErrors = fields.IgnoreErrors
	h.ModuleArgs = make(map[string]interface{})

	// 已知的标准字段
	knownFields := map[string]bool{
		"name":          true,
		"listen":        true,
		"when":          true,
		"ignore_errors": true,
	}

	// 已知的模块列表（与 Task 相同）
	knownModules := map[string]bool{
		"ping":                         true,
		"command":                      true,
		"shell":                        true,
		"raw":                          true,
		"copy":                         true,
		"debug":                        true,
		"set_fact":                     true,
		"file":                         true,
		"template":                     true,
		"lineinfile":                   true,
		"service":                      true,
		"ansible.builtin.import_tasks": true,
		"import_tasks":                 true,
		"ansible.builtin.include_role": true,
		"include_role":                 true,
	}

	// 遍历所有字段，查找模块名
	if value.Kind == yaml.MappingNode {
		for i := 0; i < len(value.Content); i += 2 {
			keyNode := value.Content[i]
			valueNode := value.Content[i+1]

			key := keyNode.Value

			// 跳过已知字段
			if knownFields[key] {
				continue
			}

			// 检查是否是模块
			if knownModules[key] {
				h.Module = key

				// 解析模块参数
				switch valueNode.Kind {
				case yaml.ScalarNode:
					// 短格式: command: uptime
					if valueNode.Value != "" {
						h.ModuleArgs["_raw_params"] = valueNode.Value
					}
				case yaml.MappingNode:
					// 长格式: command: {cmd: uptime}
					var args map[string]interface{}
					if err := valueNode.Decode(&args); err != nil {
						return fmt.Errorf("failed to parse module args: %w", err)
					}
					h.ModuleArgs = args
				default:
					return fmt.Errorf("unsupported module args format for module %s", key)
				}
				break
			}
		}
	}

	if h.Module == "" {
		return fmt.Errorf("no module found in handler: %s", h.Name)
	}

	return nil
}

// TaskResult 任务执行结果
type TaskResult struct {
	Host    string
	Task    string
	Changed bool
	Failed  bool
	Skipped bool
	Msg     string
	Data    map[string]interface{}
}

// PlayRecap Play 执行总结
type PlayRecap struct {
	PlayName string
	Stats    map[string]*HostStats
}

// HostStats 主机统计信息
type HostStats struct {
	Ok          int
	Changed     int
	Failed      int
	Skipped     int
	Unreachable int
}

// String 返回格式化的统计信息
func (s *HostStats) String() string {
	return fmt.Sprintf("ok=%d changed=%d unreachable=%d failed=%d skipped=%d",
		s.Ok, s.Changed, s.Unreachable, s.Failed, s.Skipped)
}

// IsSuccess 检查主机是否成功
func (s *HostStats) IsSuccess() bool {
	return s.Failed == 0 && s.Unreachable == 0
}

// ParsePlaybook 解析 Playbook YAML 文件
func ParsePlaybook(data []byte) (Playbook, error) {
	var playbook Playbook
	if err := yaml.Unmarshal(data, &playbook); err != nil {
		return nil, fmt.Errorf("failed to parse playbook: %w", err)
	}

	// 设置默认值
	for i := range playbook {
		if playbook[i].Vars == nil {
			playbook[i].Vars = make(map[string]interface{})
		}
		// gather_facts 默认为 true，但我们简化为 false
		// 如果 YAML 中没有显式设置，保持解析后的零值 false
	}

	return playbook, nil
}

// FormatTaskName 格式化任务名称用于显示
func FormatTaskName(playName, taskName string) string {
	if taskName == "" {
		return playName
	}
	if playName == "" {
		return taskName
	}
	return fmt.Sprintf("%s : %s", playName, taskName)
}

// NormalizeModuleArgs 规范化模块参数
// 将短格式转换为标准格式
func NormalizeModuleArgs(moduleName string, args map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 复制所有参数
	for k, v := range args {
		result[k] = v
	}

	// 如果有 _raw_params，根据模块类型转换
	if rawParams, ok := result["_raw_params"].(string); ok && rawParams != "" {
		switch moduleName {
		case "command", "shell", "raw":
			// 这些模块支持 _raw_params
			// 保持不变
		case "debug":
			// debug 模块通常使用 msg 参数
			if _, hasMsg := result["msg"]; !hasMsg {
				result["msg"] = rawParams
				delete(result, "_raw_params")
			}
		}
	}

	return result
}

// IsTemplateString 检查字符串是否包含模板语法
func IsTemplateString(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}
