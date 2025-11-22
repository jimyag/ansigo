package logger

import (
	"fmt"
	"os"
	"strings"
)

// AnsibleLogger Ansible 风格的日志输出
type AnsibleLogger struct {
	quiet bool
}

// NewAnsibleLogger 创建 Ansible 风格的日志记录器
func NewAnsibleLogger(quiet bool) *AnsibleLogger {
	return &AnsibleLogger{
		quiet: quiet,
	}
}

// 颜色代码
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

// PlayHeader 打印 Play 头部
func (a *AnsibleLogger) PlayHeader(playName string) {
	if a.quiet {
		return
	}
	header := fmt.Sprintf("\nPLAY [%s] %s\n", playName, strings.Repeat("*", 44))
	fmt.Print(header)
}

// TaskHeader 打印任务头部
func (a *AnsibleLogger) TaskHeader(taskName string) {
	if a.quiet {
		return
	}
	header := fmt.Sprintf("TASK [%s] %s", taskName, strings.Repeat("*", 44))
	fmt.Println(header)
}

// TaskResult 打印任务结果
func (a *AnsibleLogger) TaskResult(status, host, msg string, changed, failed, skipped bool) {
	if a.quiet && !failed {
		return
	}

	var color, statusText string

	if failed {
		statusText = "FAILED"
		color = ColorRed
	} else if skipped {
		statusText = "skipped"
		color = ColorCyan
	} else if changed {
		statusText = "changed"
		color = ColorYellow
	} else {
		statusText = "ok"
		color = ColorGreen
	}

	// 控制台输出 - Ansible 风格
	output := fmt.Sprintf("%s: [%s] => %s%s%s", statusText, host, color, msg, ColorReset)
	fmt.Println(output)
}

// PlayRecap 打印 Play 总结
func (a *AnsibleLogger) PlayRecap(stats map[string]*PlayStats) {
	if a.quiet {
		return
	}

	fmt.Println("PLAY RECAP " + strings.Repeat("*", 44))

	for host, stat := range stats {
		statusColor := ColorGreen
		if !stat.IsSuccess() {
			statusColor = ColorRed
		}

		output := fmt.Sprintf("%s%-20s%s : %s",
			statusColor, host, ColorReset, stat.String())
		fmt.Println(output)
	}

	fmt.Println()
}

// Warning 打印警告信息
func (a *AnsibleLogger) Warning(msg string) {
	if a.quiet {
		return
	}
	fmt.Printf("%s[WARNING]: %s%s\n", ColorYellow, msg, ColorReset)
}

// Error 打印错误信息
func (a *AnsibleLogger) Error(msg string) {
	fmt.Printf("%s[ERROR]: %s%s\n", ColorRed, msg, ColorReset)
}

// Fatal 打印致命错误并退出
func (a *AnsibleLogger) Fatal(msg string) {
	fmt.Printf("%s[FATAL]: %s%s\n", ColorRed, msg, ColorReset)
	os.Exit(1)
}

// Info 打印信息
func (a *AnsibleLogger) Info(msg string) {
	if a.quiet {
		return
	}
	fmt.Println(msg)
}

// Debug 打印调试信息
func (a *AnsibleLogger) Debug(msg string) {
	// Debug messages are typically not shown unless in verbose mode
	// For now, we don't output anything
}

// PlayStats Play 统计信息
type PlayStats struct {
	Ok      int
	Changed int
	Failed  int
	Skipped int
}

// IsSuccess 检查是否成功
func (s *PlayStats) IsSuccess() bool {
	return s.Failed == 0
}

// String 返回统计信息字符串
func (s *PlayStats) String() string {
	return fmt.Sprintf("ok=%d changed=%d unreachable=0 failed=%d skipped=%d rescued=0 ignored=0",
		s.Ok, s.Changed, s.Failed, s.Skipped)
}
