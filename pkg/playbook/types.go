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
	Name        string                 `yaml:"name"`
	Hosts       string                 `yaml:"hosts"`
	GatherFacts bool                   `yaml:"gather_facts"`
	Vars        map[string]interface{} `yaml:"vars"`
	Tasks       []Task                 `yaml:"tasks"`
}

// Task 代表一个任务
type Task struct {
	Name         string
	Module       string
	ModuleArgs   map[string]interface{}
	Register     string
	When         string
	IgnoreErrors bool
}

// UnmarshalYAML 自定义 Task 的 YAML 解析
func (t *Task) UnmarshalYAML(value *yaml.Node) error {
	// 使用辅助结构解析已知字段
	type TaskFields struct {
		Name         string `yaml:"name"`
		Register     string `yaml:"register"`
		When         string `yaml:"when"`
		IgnoreErrors bool   `yaml:"ignore_errors"`
	}

	var fields TaskFields
	if err := value.Decode(&fields); err != nil {
		return err
	}

	t.Name = fields.Name
	t.Register = fields.Register
	t.When = fields.When
	t.IgnoreErrors = fields.IgnoreErrors
	t.ModuleArgs = make(map[string]interface{})

	// 已知的标准字段
	knownFields := map[string]bool{
		"name":          true,
		"register":      true,
		"when":          true,
		"ignore_errors": true,
	}

	// 已知的模块列表
	knownModules := map[string]bool{
		"ping":     true,
		"command":  true,
		"shell":    true,
		"raw":      true,
		"copy":     true,
		"debug":    true,
		"set_fact": true,
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
