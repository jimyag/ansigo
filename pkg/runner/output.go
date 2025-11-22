package runner

import (
	"encoding/json"
	"fmt"
	"sort"
)

// FormatResults 格式化输出结果（类似 Ansible 风格）
func FormatResults(results []TaskResult) string {
	output := ""

	// 按主机名排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Host < results[j].Host
	})

	for _, result := range results {
		status := "SUCCESS"
		color := "\033[32m" // 绿色

		if result.ModuleResult.Unreachable {
			status = "UNREACHABLE"
			color = "\033[31m" // 红色
		} else if result.ModuleResult.Failed {
			status = "FAILED"
			color = "\033[31m" // 红色
		} else if result.ModuleResult.Changed {
			status = "CHANGED"
			color = "\033[33m" // 黄色
		}

		// 序列化结果为 JSON
		jsonData, err := json.Marshal(result.ModuleResult)
		if err != nil {
			jsonData = []byte(fmt.Sprintf(`{"error": "failed to marshal result: %v"}`, err))
		}

		output += fmt.Sprintf("%s | %s%s\033[0m => %s\n",
			result.Host,
			color,
			status,
			string(jsonData))
	}

	return output
}
