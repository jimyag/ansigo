package playbook

import (
	"fmt"
	"strings"
	"sync"

	gojinja2 "github.com/kluctl/kluctl/lib/go-jinja2"
)

// Jinja2TemplateEngine 完整的 Jinja2 模板引擎（使用 go-jinja2）
type Jinja2TemplateEngine struct {
	j2     *gojinja2.Jinja2
	mu     sync.Mutex
	inited bool
}

// NewJinja2TemplateEngine 创建 Jinja2 模板引擎
func NewJinja2TemplateEngine() *Jinja2TemplateEngine {
	return &Jinja2TemplateEngine{}
}

// init 延迟初始化 Jinja2 引擎
func (te *Jinja2TemplateEngine) init() error {
	te.mu.Lock()
	defer te.mu.Unlock()

	if te.inited {
		return nil
	}

	// 创建 Jinja2 实例，使用单个渲染进程
	j2, err := gojinja2.NewJinja2("ansigo", 1)
	if err != nil {
		return fmt.Errorf("failed to initialize Jinja2 engine: %w", err)
	}

	te.j2 = j2
	te.inited = true
	return nil
}

// Close 关闭 Jinja2 引擎
func (te *Jinja2TemplateEngine) Close() error {
	te.mu.Lock()
	defer te.mu.Unlock()

	if te.j2 != nil {
		te.j2.Close()
		te.j2 = nil
		te.inited = false
	}
	return nil
}

// RenderString 渲染单个字符串
func (te *Jinja2TemplateEngine) RenderString(template string, context map[string]interface{}) (string, error) {
	// 如果没有模板语法，直接返回
	if !strings.Contains(template, "{{") && !strings.Contains(template, "{%") {
		return template, nil
	}

	// 确保引擎已初始化
	if err := te.init(); err != nil {
		return "", err
	}

	// 使用 go-jinja2 渲染
	// 注意：go-jinja2 完全支持 Jinja2 语法，包括波浪号操作符和内联条件表达式
	result, err := te.j2.RenderString(template, gojinja2.WithGlobals(context))
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return result, nil
}

// RenderValue 渲染并返回原始值（可能是列表、字典等）
func (te *Jinja2TemplateEngine) RenderValue(template string, context map[string]interface{}) (interface{}, error) {
	// 如果没有模板语法，直接返回原字符串
	if !strings.Contains(template, "{{") && !strings.Contains(template, "{%") {
		return template, nil
	}

	// 确保引擎已初始化
	if err := te.init(); err != nil {
		return nil, err
	}

	// 对于简单的变量引用（如 "{{ packages }}" 或 "{{ var.field }}"），直接从 context 获取值
	template = strings.TrimSpace(template)
	if strings.HasPrefix(template, "{{") && strings.HasSuffix(template, "}}") {
		// 提取变量表达式
		varExpr := strings.TrimSpace(template[2 : len(template)-2])

		// 处理简单变量名（没有过滤器、括号、运算符等）
		if !strings.ContainsAny(varExpr, "|[]()+-*/%<>=!&") {
			// 尝试解析点号访问（如 var.field.subfield）
			parts := strings.Split(varExpr, ".")
			value := context[parts[0]]

			// 如果有嵌套访问
			for i := 1; i < len(parts) && value != nil; i++ {
				if m, ok := value.(map[string]interface{}); ok {
					value = m[parts[i]]
				} else {
					// 无法继续访问，返回 nil
					value = nil
					break
				}
			}

			if value != nil {
				return value, nil
			}
		}
	}

	// 对于复杂表达式，先渲染为字符串再尝试解析
	// 这种情况下可能丢失类型信息，但至少能工作
	result, err := te.j2.RenderString(template, gojinja2.WithGlobals(context))
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	return result, nil
}

// RenderArgs 渲染模块参数
func (te *Jinja2TemplateEngine) RenderArgs(args map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range args {
		switch v := value.(type) {
		case string:
			rendered, err := te.RenderString(v, context)
			if err != nil {
				return nil, fmt.Errorf("failed to render %s: %w", key, err)
			}
			result[key] = rendered
		case map[string]interface{}:
			// 递归渲染嵌套的 map
			rendered, err := te.RenderArgs(v, context)
			if err != nil {
				return nil, err
			}
			result[key] = rendered
		case []interface{}:
			// 渲染数组中的字符串
			renderedArray := make([]interface{}, len(v))
			for i, item := range v {
				if strItem, ok := item.(string); ok {
					rendered, err := te.RenderString(strItem, context)
					if err != nil {
						return nil, fmt.Errorf("failed to render array item: %w", err)
					}
					renderedArray[i] = rendered
				} else {
					renderedArray[i] = item
				}
			}
			result[key] = renderedArray
		default:
			result[key] = value
		}
	}

	return result, nil
}

// EvaluateCondition 评估 when 条件
func (te *Jinja2TemplateEngine) EvaluateCondition(condition string, context map[string]interface{}) (bool, error) {
	if condition == "" {
		return true, nil
	}

	// 确保引擎已初始化
	if err := te.init(); err != nil {
		return false, err
	}

	// Ansible 的 when 条件不需要 {{ }}
	// 我们需要将其包装为 Jinja2 表达式
	// 例如: "result.rc == 0" -> "{% if result.rc == 0 %}true{% else %}false{% endif %}"

	// 构建一个简单的 if 模板来评估条件
	template := fmt.Sprintf("{%% if %s %%}true{%% else %%}false{%% endif %%}", condition)

	result, err := te.j2.RenderString(template, gojinja2.WithGlobals(context))
	if err != nil {
		return false, fmt.Errorf("failed to evaluate condition: %w", err)
	}

	return strings.TrimSpace(result) == "true", nil
}

// RegisterCustomFilters 注册自定义过滤器
// 注意：go-jinja2 使用真正的 Jinja2，因此自定义过滤器需要通过 Python 代码注册
// 目前这个方法保留为空，因为标准 Jinja2 过滤器已经完全支持
func (te *Jinja2TemplateEngine) RegisterCustomFilters() {
	// go-jinja2 提供了完整的 Jinja2 环境，包括所有标准过滤器：
	// - to_json, to_yaml
	// - b64encode, b64decode
	// - regex_replace, regex_search
	// - default
	// - length
	// - upper, lower
	// - trim
	// - join
	// - first, last
	// - etc.
	//
	// 如果需要自定义过滤器，需要通过 Jinja2Opt 在初始化时提供 Python 代码
}

// EvaluateExpression 评估表达式（用于更复杂的情况）
func (te *Jinja2TemplateEngine) EvaluateExpression(expr string, context map[string]interface{}) (interface{}, error) {
	// 将表达式包装为模板
	template := fmt.Sprintf("{{ %s }}", expr)

	result, err := te.RenderString(template, context)
	if err != nil {
		return nil, err
	}

	return result, nil
}
