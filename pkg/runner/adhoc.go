package runner

import (
	"sync"

	"github.com/jimyag/ansigo/pkg/connection"
	"github.com/jimyag/ansigo/pkg/errors"
	"github.com/jimyag/ansigo/pkg/inventory"
	"github.com/jimyag/ansigo/pkg/module"
)

// TaskResult 任务执行结果
type TaskResult struct {
	Host         string
	ModuleResult *module.Result
	Error        error
}

// AdhocRunner Ad-hoc 命令执行器
type AdhocRunner struct {
	inventory *inventory.Manager
	connMgr   *connection.Manager
	modExec   *module.Executor
}

// NewAdhocRunner 创建一个新的 Ad-hoc Runner
func NewAdhocRunner(inv *inventory.Manager) *AdhocRunner {
	return &AdhocRunner{
		inventory: inv,
		connMgr:   connection.NewManager(),
		modExec:   module.NewExecutor(),
	}
}

// Run 运行 ad-hoc 命令
func (r *AdhocRunner) Run(pattern, moduleName string, moduleArgs map[string]interface{}) ([]TaskResult, error) {
	// 获取目标主机
	hosts, err := r.inventory.GetHosts(pattern)
	if err != nil {
		return nil, err
	}

	// 并发执行
	results := make(chan TaskResult, len(hosts))
	var wg sync.WaitGroup

	for _, host := range hosts {
		wg.Add(1)
		go func(h *inventory.Host) {
			defer wg.Done()
			result := r.executeOnHost(h, moduleName, moduleArgs)
			results <- result
		}(host)
	}

	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	taskResults := []TaskResult{}
	for result := range results {
		taskResults = append(taskResults, result)
	}

	return taskResults, nil
}

// executeOnHost 在单个主机上执行模块
func (r *AdhocRunner) executeOnHost(host *inventory.Host, moduleName string, moduleArgs map[string]interface{}) TaskResult {
	// 建立连接
	conn, err := r.connMgr.Connect(host)
	if err != nil {
		// 连接失败
		var execErr *errors.ExecutionError
		if e, ok := err.(*errors.ExecutionError); ok {
			execErr = e
		}

		return TaskResult{
			Host: host.Name,
			ModuleResult: &module.Result{
				Unreachable: true,
				Msg:         err.Error(),
			},
			Error: execErr,
		}
	}
	defer conn.Close()

	// 执行模块（ad-hoc 命令默认不使用 become）
	modResult, err := r.modExec.Execute(conn, moduleName, moduleArgs, false, "", "")
	if err != nil {
		return TaskResult{
			Host: host.Name,
			ModuleResult: &module.Result{
				Failed: true,
				Msg:    err.Error(),
			},
			Error: err,
		}
	}

	return TaskResult{
		Host:         host.Name,
		ModuleResult: modResult,
		Error:        nil,
	}
}
