package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jimyag/ansigo/pkg/inventory"
	"github.com/jimyag/ansigo/pkg/logger"
	"github.com/jimyag/ansigo/pkg/playbook"
)

func main() {
	// 定义命令行参数
	inventoryPath := flag.String("i", "inventory.ini", "Path to inventory file")
	verbose := flag.Bool("v", false, "Verbose mode")
	flag.Parse()

	// 初始化日志系统
	logLevel := logger.InfoLevel
	if *verbose {
		logLevel = logger.DebugLevel
	}
	logger.Init(&logger.Config{
		Level:  logLevel,
		Output: os.Stdout,
		Pretty: true,
	})

	// 获取 playbook 文件路径
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: ansigo-playbook -i <inventory> <playbook.yml>")
		fmt.Println("Example: ansigo-playbook -i hosts.ini site.yml")
		os.Exit(1)
	}
	playbookPath := args[0]

	// 加载 inventory
	invMgr := inventory.NewManager()
	if err := invMgr.Load(*inventoryPath); err != nil {
		logger.Errorf("Failed to load inventory: %v", err)
		os.Exit(1)
	}

	logger.Debugf("Loaded inventory from %s", *inventoryPath)

	// 读取 playbook 文件
	playbookData, err := os.ReadFile(playbookPath)
	if err != nil {
		logger.Errorf("Failed to read playbook: %v", err)
		os.Exit(1)
	}

	// 解析 playbook
	pb, err := playbook.ParsePlaybook(playbookData)
	if err != nil {
		logger.Errorf("Failed to parse playbook: %v", err)
		os.Exit(1)
	}

	logger.Debugf("Parsed playbook from %s", playbookPath)

	// 创建 runner 并执行
	runner := playbook.NewRunner(invMgr)
	defer runner.Close() // 确保释放模板引擎资源

	// 设置 playbook 路径（用于 role 查找）
	runner.SetPlaybookPath(playbookPath)

	if err := runner.Run(pb); err != nil {
		logger.Errorf("Playbook execution failed: %v", err)
		os.Exit(2)
	}
}
