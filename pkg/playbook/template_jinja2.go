package playbook

import (
	"fmt"
	"strings"

	"github.com/flosch/pongo2/v6"
)

// Jinja2TemplateEngine 完整的 Jinja2 模板引擎（使用 pongo2）
type Jinja2TemplateEngine struct {
	// pongo2 不需要保持状态，每次渲染都是独立的
}

// NewJinja2TemplateEngine 创建 Jinja2 模板引擎
func NewJinja2TemplateEngine() *Jinja2TemplateEngine {
	return &Jinja2TemplateEngine{}
}

// RenderString 渲染单个字符串
func (te *Jinja2TemplateEngine) RenderString(template string, context map[string]interface{}) (string, error) {
	// 如果没有模板语法，直接返回
	if !strings.Contains(template, "{{") && !strings.Contains(template, "{%") {
		return template, nil
	}

	// 使用 pongo2 渲染
	tpl, err := pongo2.FromString(template)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// 转换上下文为 pongo2.Context
	pongoCtx := pongo2.Context{}
	for k, v := range context {
		pongoCtx[k] = v
	}

	result, err := tpl.Execute(pongoCtx)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
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

	// Ansible 的 when 条件不需要 {{ }}
	// 我们需要将其包装为 Jinja2 表达式
	// 例如: "result.rc == 0" -> "{% if result.rc == 0 %}true{% else %}false{% endif %}"

	// 构建一个简单的 if 模板来评估条件
	template := fmt.Sprintf("{%% if %s %%}true{%% else %%}false{%% endif %%}", condition)

	tpl, err := pongo2.FromString(template)
	if err != nil {
		return false, fmt.Errorf("failed to parse condition: %w", err)
	}

	// 转换上下文
	pongoCtx := pongo2.Context{}
	for k, v := range context {
		pongoCtx[k] = v
	}

	result, err := tpl.Execute(pongoCtx)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate condition: %w", err)
	}

	return strings.TrimSpace(result) == "true", nil
}

// RegisterCustomFilters 注册自定义过滤器
func (te *Jinja2TemplateEngine) RegisterCustomFilters() {
	// Ansible 常用过滤器

	// to_json - 转换为 JSON
	pongo2.RegisterFilter("to_json", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		// pongo2 内置了 jsonencode，我们可以直接使用
		return in, nil
	})

	// to_yaml - 转换为 YAML（简化版）
	pongo2.RegisterFilter("to_yaml", func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		return pongo2.AsValue(fmt.Sprintf("%v", in.Interface())), nil
	})

	// b64encode - Base64 编码（需要时可以实现）
	// b64decode - Base64 解码

	// regex_replace - 正则表达式替换（需要时可以实现）

	// Note: pongo2 已经支持很多标准的 Jinja2 过滤器:
	// - default
	// - length
	// - upper, lower
	// - trim
	// - join
	// - first, last
	// - etc.
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
