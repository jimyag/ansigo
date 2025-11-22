package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

// PlaybookPreprocessor 预处理 Ansible playbook
type PlaybookPreprocessor struct {
	inputFile  string
	outputFile string
	inPlace    bool
	verbose    bool
}

func main() {
	pp := &PlaybookPreprocessor{}

	flag.StringVar(&pp.inputFile, "input", "", "Input playbook file (required)")
	flag.StringVar(&pp.outputFile, "output", "", "Output playbook file (default: input_preprocessed.yml)")
	flag.BoolVar(&pp.inPlace, "in-place", false, "Modify file in place")
	flag.BoolVar(&pp.verbose, "v", false, "Verbose output")
	flag.Parse()

	if pp.inputFile == "" {
		flag.Usage()
		log.Fatal("Error: -input flag is required")
	}

	// 设置输出文件
	if pp.inPlace {
		pp.outputFile = pp.inputFile
	} else if pp.outputFile == "" {
		dir := filepath.Dir(pp.inputFile)
		base := filepath.Base(pp.inputFile)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)
		pp.outputFile = filepath.Join(dir, name+"_preprocessed"+ext)
	}

	if err := pp.process(); err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("✅ Preprocessed playbook written to: %s\n", pp.outputFile)
}

func (pp *PlaybookPreprocessor) process() error {
	// 读取输入文件
	content, err := ioutil.ReadFile(pp.inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	originalContent := string(content)
	processedContent := originalContent

	// 预处理步骤
	processedContent = pp.preprocessTildeOperator(processedContent)
	processedContent = pp.preprocessInlineConditional(processedContent)

	// 如果有变化，显示统计
	if pp.verbose && processedContent != originalContent {
		pp.showChanges(originalContent, processedContent)
	}

	// 写入输出文件
	if err := ioutil.WriteFile(pp.outputFile, []byte(processedContent), 0o644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// preprocessTildeOperator 预处理 Jinja2 的 ~ 连接符
// 将 {{ a ~ b ~ c }} 转换为 {{ a }}{{ b }}{{ c }}
func (pp *PlaybookPreprocessor) preprocessTildeOperator(content string) string {
	// 匹配 {{ ... }} 中包含 ~ 的表达式
	re := regexp.MustCompile(`\{\{([^}]*~[^}]*)\}\}`)

	replacements := 0
	result := re.ReplaceAllStringFunc(content, func(match string) string {
		// 提取 {{ 和 }} 之间的内容
		inner := match[2 : len(match)-2]

		// 如果不包含 ~，直接返回
		if !strings.Contains(inner, "~") {
			return match
		}

		// 检查是否有过滤器（在 | 后面）
		filterIdx := -1
		parenDepth := 0
		for i := len(inner) - 1; i >= 0; i-- {
			ch := inner[i]
			if ch == ')' {
				parenDepth++
			} else if ch == '(' {
				parenDepth--
			} else if ch == '|' && parenDepth == 0 {
				filterIdx = i
				break
			}
		}

		// 如果有过滤器，保持原样（用户需要手动修改）
		if filterIdx != -1 {
			if pp.verbose {
				log.Printf("⚠️  Warning: Found tilde operator with filter, keeping as-is: %s", match)
			}
			return match
		}

		// 移除外层括号
		inner = strings.TrimSpace(inner)
		if strings.HasPrefix(inner, "(") && strings.HasSuffix(inner, ")") {
			inner = strings.TrimSpace(inner[1 : len(inner)-1])
		}

		// 分割 ~ 操作符
		parts := splitTildeExpression(inner)
		if len(parts) <= 1 {
			return match
		}

		// 转换为多个连续的 {{ }} 表达式
		var result strings.Builder
		for _, part := range parts {
			result.WriteString("{{ ")
			result.WriteString(part)
			result.WriteString(" }}")
		}

		replacements++
		if pp.verbose {
			log.Printf("✏️  Converted tilde: %s → %s", match, result.String())
		}

		return result.String()
	})

	if pp.verbose && replacements > 0 {
		log.Printf("📊 Converted %d tilde operators", replacements)
	}

	return result
}

// preprocessInlineConditional 预处理 Jinja2 的内联条件表达式
// 将 {{ 'a' if condition else 'b' }} 转换为 {% if condition %}{{ 'a' }}{% else %}{{ 'b' }}{% endif %}
func (pp *PlaybookPreprocessor) preprocessInlineConditional(content string) string {
	// 匹配 {{ ... if ... else ... }} 模式
	re := regexp.MustCompile(`\{\{([^}]*)\s+if\s+([^}]*)\s+else\s+([^}]*)\}\}`)

	replacements := 0
	result := re.ReplaceAllStringFunc(content, func(match string) string {
		// 提取 {{ 和 }} 之间的内容
		inner := match[2 : len(match)-2]

		// 分割 if 和 else
		ifIdx := strings.Index(inner, " if ")
		if ifIdx == -1 {
			return match
		}

		elseIdx := strings.LastIndex(inner, " else ")
		if elseIdx == -1 || elseIdx <= ifIdx {
			return match
		}

		// 提取三个部分: true_value, condition, false_value
		trueValue := strings.TrimSpace(inner[:ifIdx])
		condition := strings.TrimSpace(inner[ifIdx+4 : elseIdx])
		falseValue := strings.TrimSpace(inner[elseIdx+6:])

		// 转换为 {% if condition %}{{ true_value }}{% else %}{{ false_value }}{% endif %}
		converted := fmt.Sprintf("{%% if %s %%}{{ %s }}{%% else %%}{{ %s }}{%% endif %%}",
			condition, trueValue, falseValue)

		replacements++
		if pp.verbose {
			log.Printf("✏️  Converted conditional: %s → %s", match, converted)
		}

		return converted
	})

	if pp.verbose && replacements > 0 {
		log.Printf("📊 Converted %d inline conditionals", replacements)
	}

	return result
}

// splitTildeExpression 分割包含 ~ 的表达式
// 保留字符串字面量中的 ~
func splitTildeExpression(expr string) []string {
	var parts []string
	var currentPart strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		switch ch {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
			currentPart.WriteByte(ch)
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
			currentPart.WriteByte(ch)
		case '~':
			// 如果在引号内，不作为操作符处理
			if inSingleQuote || inDoubleQuote {
				currentPart.WriteByte(ch)
			} else {
				// 这是一个连接操作符，保存当前部分
				part := strings.TrimSpace(currentPart.String())
				if part != "" {
					parts = append(parts, part)
				}
				currentPart.Reset()
			}
		default:
			currentPart.WriteByte(ch)
		}
	}

	// 添加最后一部分
	part := strings.TrimSpace(currentPart.String())
	if part != "" {
		parts = append(parts, part)
	}

	return parts
}

func (pp *PlaybookPreprocessor) showChanges(original, processed string) {
	originalLines := strings.Split(original, "\n")
	processedLines := strings.Split(processed, "\n")

	changed := 0
	for i := 0; i < len(originalLines) && i < len(processedLines); i++ {
		if originalLines[i] != processedLines[i] {
			changed++
		}
	}

	log.Printf("📊 Summary: %d lines changed out of %d total lines", changed, len(originalLines))
}
