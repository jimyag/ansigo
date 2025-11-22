package playbook

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TemplateEngine 简单的模板引擎
// 支持基本的 {{ variable }} 替换
type TemplateEngine struct{}

// NewTemplateEngine 创建模板引擎
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{}
}

// RenderString 渲染单个字符串
func (te *TemplateEngine) RenderString(template string, context map[string]interface{}) (string, error) {
	if !IsTemplateString(template) {
		return template, nil
	}

	// 匹配 {{ variable }} 模式
	re := regexp.MustCompile(`\{\{\s*([^}]+)\s*\}\}`)

	result := re.ReplaceAllStringFunc(template, func(match string) string {
		// 提取变量名
		varExpr := strings.TrimSpace(match[2 : len(match)-2])

		// 解析变量表达式
		value, err := te.evaluateExpression(varExpr, context)
		if err != nil {
			return match // 保留原始内容
		}

		return fmt.Sprintf("%v", value)
	})

	return result, nil
}

// RenderArgs 渲染模块参数
func (te *TemplateEngine) RenderArgs(args map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
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
		default:
			result[key] = value
		}
	}

	return result, nil
}

// EvaluateCondition 评估 when 条件
func (te *TemplateEngine) EvaluateCondition(condition string, context map[string]interface{}) (bool, error) {
	if condition == "" {
		return true, nil
	}

	// 简单的条件评估
	// 支持的格式：
	// - variable (检查是否为真值)
	// - variable == value
	// - variable != value
	// - not variable
	// - condition1 or condition2
	// - condition1 and condition2

	condition = strings.TrimSpace(condition)

	// 处理 or 操作符（优先级最低）
	if strings.Contains(condition, " or ") {
		parts := strings.Split(condition, " or ")
		for _, part := range parts {
			result, err := te.EvaluateCondition(strings.TrimSpace(part), context)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// 处理 and 操作符
	if strings.Contains(condition, " and ") {
		parts := strings.Split(condition, " and ")
		for _, part := range parts {
			result, err := te.EvaluateCondition(strings.TrimSpace(part), context)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// 处理 not 操作符
	if strings.HasPrefix(condition, "not ") {
		innerCondition := strings.TrimSpace(condition[4:])
		result, err := te.EvaluateCondition(innerCondition, context)
		if err != nil {
			return false, err
		}
		return !result, nil
	}

	// 处理比较操作符
	for _, op := range []string{"==", "!="} {
		if strings.Contains(condition, op) {
			parts := strings.SplitN(condition, op, 2)
			if len(parts) != 2 {
				return false, fmt.Errorf("invalid condition: %s", condition)
			}

			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])

			// 评估左侧
			leftVal, err := te.evaluateExpression(left, context)
			if err != nil {
				return false, err
			}

			// 评估右侧
			rightVal, err := te.evaluateExpression(right, context)
			if err != nil {
				return false, err
			}

			// 比较
			equal := fmt.Sprintf("%v", leftVal) == fmt.Sprintf("%v", rightVal)
			if op == "==" {
				return equal, nil
			}
			return !equal, nil
		}
	}

	// 简单的真值检查
	value, err := te.evaluateExpression(condition, context)
	if err != nil {
		return false, err
	}

	return isTruthy(value), nil
}

// evaluateExpression 评估表达式
func (te *TemplateEngine) evaluateExpression(expr string, context map[string]interface{}) (interface{}, error) {
	expr = strings.TrimSpace(expr)

	// 去掉引号（如果有）
	if (strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) ||
		(strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) {
		return expr[1 : len(expr)-1], nil
	}

	// 检查是否是数字
	if num, err := strconv.Atoi(expr); err == nil {
		return num, nil
	}
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return num, nil
	}

	// 检查是否是布尔值
	if expr == "true" || expr == "True" {
		return true, nil
	}
	if expr == "false" || expr == "False" {
		return false, nil
	}

	// 处理属性访问 (e.g., result.stdout)
	if strings.Contains(expr, ".") {
		parts := strings.SplitN(expr, ".", 2)
		if value, ok := context[parts[0]]; ok {
			// 尝试访问嵌套属性
			if m, ok := value.(map[string]interface{}); ok {
				if nestedValue, exists := m[parts[1]]; exists {
					return nestedValue, nil
				}
			}
		}
		return nil, fmt.Errorf("undefined variable: %s", expr)
	}

	// 查找变量
	if value, ok := context[expr]; ok {
		return value, nil
	}

	// 变量未定义，返回空字符串
	return "", nil
}

// isTruthy 判断值是否为真
func isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "false" && v != "False"
	case int, int64, float64:
		return fmt.Sprintf("%v", v) != "0"
	case map[string]interface{}:
		return len(v) > 0
	case []interface{}:
		return len(v) > 0
	default:
		return true
	}
}
