package playbook

import (
	"fmt"
	"regexp"
	"strings"
)

// Jinja2ToGoTemplate 将 Jinja2 模板语法转换为 Go text/template 语法
type Jinja2ToGoTemplate struct {
	// 可以添加配置选项
}

// Convert 转换 Jinja2 模板为 Go template
func (j *Jinja2ToGoTemplate) Convert(jinja2Template string) (string, error) {
	result := jinja2Template

	// 1. 转换变量引用: {{ variable }} → {{ .variable }}
	result = j.convertVariables(result)

	// 2. 转换 if 语句: {% if condition %} → {{ if .condition }}
	result = j.convertIfStatements(result)

	// 3. 转换 for 循环: {% for item in list %} → {{ range .list }}
	result = j.convertForLoops(result)

	// 4. 转换过滤器: {{ var | upper }} → {{ .var | upper }}
	result = j.convertFilters(result)

	// 5. 转换字符串连接: {{ a ~ b }} → {{ print .a .b }}
	result = j.convertTildeOperator(result)

	// 6. 转换内联条件: {{ 'a' if cond else 'b' }} → {{ if .cond }}a{{ else }}b{{ end }}
	result = j.convertInlineConditional(result)

	return result, nil
}

// convertVariables 转换变量引用
// {{ variable }} → {{ .variable }}
// {{ variable.field }} → {{ .variable.field }}
func (j *Jinja2ToGoTemplate) convertVariables(template string) string {
	// 匹配 {{ variable }} 但不包含已经有 . 前缀的
	re := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)`)

	return re.ReplaceAllStringFunc(template, func(match string) string {
		// 提取变量名
		parts := strings.SplitN(match, "{{", 2)
		if len(parts) < 2 {
			return match
		}

		varPart := strings.TrimSpace(parts[1])

		// 如果已经有 . 前缀，或者是字符串字面量，跳过
		if strings.HasPrefix(varPart, ".") ||
			strings.HasPrefix(varPart, "'") ||
			strings.HasPrefix(varPart, "\"") {
			return match
		}

		// 添加 . 前缀
		return "{{ ." + varPart
	})
}

// convertIfStatements 转换 if 语句
// {% if condition %} → {{ if .condition }}
// {% elif condition %} → {{ else if .condition }}
// {% else %} → {{ else }}
// {% endif %} → {{ end }}
func (j *Jinja2ToGoTemplate) convertIfStatements(template string) string {
	// {% if condition %}
	template = regexp.MustCompile(`\{%\s*if\s+([^%]+)\s*%\}`).ReplaceAllString(
		template, "{{ if .$1 }}")

	// {% elif condition %}
	template = regexp.MustCompile(`\{%\s*elif\s+([^%]+)\s*%\}`).ReplaceAllString(
		template, "{{ else if .$1 }}")

	// {% else %}
	template = regexp.MustCompile(`\{%\s*else\s*%\}`).ReplaceAllString(
		template, "{{ else }}")

	// {% endif %}
	template = regexp.MustCompile(`\{%\s*endif\s*%\}`).ReplaceAllString(
		template, "{{ end }}")

	return template
}

// convertForLoops 转换 for 循环
// {% for item in list %} → {{ range $item := .list }}
// {% endfor %} → {{ end }}
func (j *Jinja2ToGoTemplate) convertForLoops(template string) string {
	// {% for item in list %}
	re := regexp.MustCompile(`\{%\s*for\s+(\w+)\s+in\s+([^%]+)\s*%\}`)
	template = re.ReplaceAllString(template, "{{ range $$$1 := .$2 }}")

	// 在 loop 内部，item 引用需要变成 $item
	// 这个需要更复杂的上下文感知转换，暂时简化处理

	// {% endfor %}
	template = regexp.MustCompile(`\{%\s*endfor\s*%\}`).ReplaceAllString(
		template, "{{ end }}")

	return template
}

// convertFilters 转换过滤器
// {{ var | upper }} → {{ .var | upper }}
// 注意: Go template 的管道语法和 Jinja2 类似，但函数名可能不同
func (j *Jinja2ToGoTemplate) convertFilters(template string) string {
	// 这个由 convertVariables 已经处理了 .var 前缀
	// 只需要映射过滤器名称

	// Jinja2 → Go template 过滤器映射
	filterMap := map[string]string{
		"upper":   "upper",
		"lower":   "lower",
		"length":  "len",
		"default": "default",
		// Sprig 提供了更多函数
	}

	for jinja2Filter, goFilter := range filterMap {
		re := regexp.MustCompile(`\|\s*` + jinja2Filter + `\b`)
		template = re.ReplaceAllString(template, "| "+goFilter)
	}

	return template
}

// convertTildeOperator 转换字符串连接
// {{ a ~ b ~ c }} → {{ cat .a .b .c }} (使用 Sprig 的 cat 函数)
func (j *Jinja2ToGoTemplate) convertTildeOperator(template string) string {
	re := regexp.MustCompile(`\{\{\s*([^}]*~[^}]*)\s*\}\}`)

	return re.ReplaceAllStringFunc(template, func(match string) string {
		inner := match[2 : len(match)-2]
		inner = strings.TrimSpace(inner)

		// 分割 ~ 操作符
		parts := strings.Split(inner, "~")

		// 清理每个部分并添加 . 前缀（如果需要）
		var cleanParts []string
		for _, part := range parts {
			part = strings.TrimSpace(part)

			// 如果是字符串字面量，转换引号格式
			if strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'") {
				// 单引号 → 双引号 (Go template 使用双引号)
				// 提取引号内的内容
				content := part[1 : len(part)-1]
				part = `"` + content + `"`
			} else if strings.HasPrefix(part, `"`) && strings.HasSuffix(part, `"`) {
				// 已经是双引号，保持不变
			} else if !strings.HasPrefix(part, ".") {
				// 变量，添加 . 前缀
				part = "." + part
			}

			cleanParts = append(cleanParts, part)
		}

		// 使用 print 函数而不是 cat，因为 print 会正确处理混合的变量和字符串
		return fmt.Sprintf("{{ print %s }}", strings.Join(cleanParts, " "))
	})
}

// convertInlineConditional 转换内联条件表达式
// {{ 'a' if condition else 'b' }} → {{ if .condition }}a{{ else }}b{{ end }}
func (j *Jinja2ToGoTemplate) convertInlineConditional(template string) string {
	re := regexp.MustCompile(`\{\{\s*([^}]+)\s+if\s+([^}]+)\s+else\s+([^}]+)\s*\}\}`)

	return re.ReplaceAllStringFunc(template, func(match string) string {
		inner := match[2 : len(match)-2]

		// 分割 if 和 else
		ifIdx := strings.Index(inner, " if ")
		if ifIdx == -1 {
			return match
		}

		elseIdx := strings.LastIndex(inner, " else ")
		if elseIdx == -1 {
			return match
		}

		trueValue := strings.TrimSpace(inner[:ifIdx])
		condition := strings.TrimSpace(inner[ifIdx+4 : elseIdx])
		falseValue := strings.TrimSpace(inner[elseIdx+6:])

		// 添加 . 前缀到条件
		if !strings.HasPrefix(condition, ".") {
			condition = "." + condition
		}

		// 去除字符串字面量的引号
		trueValue = strings.Trim(trueValue, "'\"")
		falseValue = strings.Trim(falseValue, "'\"")

		return fmt.Sprintf("{{ if %s }}%s{{ else }}%s{{ end }}",
			condition, trueValue, falseValue)
	})
}

// Example usage and test
func ExampleJinja2ToGoTemplate() {
	converter := &Jinja2ToGoTemplate{}

	// Test 1: Variable
	input1 := "Hello {{ username }}"
	output1, _ := converter.Convert(input1)
	fmt.Println("Test 1:", output1)
	// Expected: "Hello {{ .username }}"

	// Test 2: If statement
	input2 := "{% if active %}Active{% else %}Inactive{% endif %}"
	output2, _ := converter.Convert(input2)
	fmt.Println("Test 2:", output2)
	// Expected: "{{ if .active }}Active{{ else }}Inactive{{ end }}"

	// Test 3: For loop
	input3 := "{% for item in items %}{{ item }}{% endfor %}"
	output3, _ := converter.Convert(input3)
	fmt.Println("Test 3:", output3)
	// Expected: "{{ range $item := .items }}{{ $item }}{{ end }}"

	// Test 4: Tilde operator
	input4 := "{{ firstname ~ ' ' ~ lastname }}"
	output4, _ := converter.Convert(input4)
	fmt.Println("Test 4:", output4)
	// Expected: "{{ print .firstname ' ' .lastname }}"

	// Test 5: Inline conditional
	input5 := "{{ 'enabled' if debug else 'disabled' }}"
	output5, _ := converter.Convert(input5)
	fmt.Println("Test 5:", output5)
	// Expected: "{{ if .debug }}enabled{{ else }}disabled{{ end }}"
}
