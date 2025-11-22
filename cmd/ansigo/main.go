package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jimyag/ansigo/pkg/inventory"
	"github.com/jimyag/ansigo/pkg/runner"
)

func main() {
	// 定义命令行参数
	inventoryPath := flag.String("i", "inventory.ini", "Path to inventory file")
	moduleName := flag.String("m", "ping", "Module name to execute")
	moduleArgs := flag.String("a", "", "Module arguments")
	flag.Parse()

	// 获取主机模式
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: ansigo -i <inventory> -m <module> -a <args> <pattern>")
		fmt.Println("Example: ansigo -i hosts.ini -m ping all")
		os.Exit(1)
	}
	pattern := args[0]

	// 加载 inventory
	invMgr := inventory.NewManager()
	if err := invMgr.Load(*inventoryPath); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to load inventory: %v\n", err)
		os.Exit(1)
	}

	// 解析模块参数
	modArgs := parseModuleArgs(*moduleArgs)

	// 创建 runner 并执行
	adhocRunner := runner.NewAdhocRunner(invMgr)
	results, err := adhocRunner.Run(pattern, *moduleName, modArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	output := runner.FormatResults(results)
	fmt.Print(output)

	// 检查是否有失败
	hasFailure := false
	for _, result := range results {
		if result.ModuleResult.Failed || result.ModuleResult.Unreachable {
			hasFailure = true
			break
		}
	}

	if hasFailure {
		os.Exit(2)
	}
}

// parseModuleArgs 解析模块参数字符串
func parseModuleArgs(argsStr string) map[string]interface{} {
	args := make(map[string]interface{})

	if argsStr == "" {
		return args
	}

	// 简单解析 key=value 格式
	// 对于没有 = 的字符串，作为 _raw_params
	if !strings.Contains(argsStr, "=") {
		args["_raw_params"] = argsStr
		return args
	}

	// 解析 key=value key2=value2 格式，支持引号内的空格
	i := 0
	for i < len(argsStr) {
		// 跳过空格
		for i < len(argsStr) && argsStr[i] == ' ' {
			i++
		}
		if i >= len(argsStr) {
			break
		}

		// 读取 key
		keyStart := i
		for i < len(argsStr) && argsStr[i] != '=' {
			i++
		}
		if i >= len(argsStr) {
			break
		}
		key := argsStr[keyStart:i]
		i++ // 跳过 '='

		// 读取 value（可能带引号）
		if i >= len(argsStr) {
			break
		}

		var value string
		if argsStr[i] == '"' || argsStr[i] == '\'' {
			// 带引号的值
			quote := argsStr[i]
			i++
			valueStart := i
			for i < len(argsStr) && argsStr[i] != quote {
				i++
			}
			value = argsStr[valueStart:i]
			if i < len(argsStr) {
				i++ // 跳过结束引号
			}
		} else {
			// 不带引号的值（到空格或结束）
			valueStart := i
			for i < len(argsStr) && argsStr[i] != ' ' {
				i++
			}
			value = argsStr[valueStart:i]
		}

		args[key] = value
	}

	return args
}
