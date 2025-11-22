package playbook

import (
	"fmt"
	"sync"

	"github.com/jimyag/ansigo/pkg/connection"
	"github.com/jimyag/ansigo/pkg/inventory"
	"github.com/jimyag/ansigo/pkg/logger"
	"github.com/jimyag/ansigo/pkg/module"
)

// Runner Playbook 执行器
type Runner struct {
	inventory *inventory.Manager
	connMgr   *connection.Manager
	modExec   *module.Executor
	varMgr    *VariableManager
	template  TemplateEngineInterface
	logger    *logger.AnsibleLogger
}

// NewRunner 创建 Playbook Runner
func NewRunner(inv *inventory.Manager) *Runner {
	return &Runner{
		inventory: inv,
		connMgr:   connection.NewManager(),
		modExec:   module.NewExecutor(),
		varMgr:    NewVariableManager(inv),
		template:  NewDefaultTemplateEngine(), // 使用 Jinja2 引擎
		logger:    logger.NewAnsibleLogger(false),
	}
}

// Run 执行整个 Playbook
func (r *Runner) Run(playbook Playbook) error {
	for _, play := range playbook {
		if err := r.ExecutePlay(&play); err != nil {
			return fmt.Errorf("play '%s' failed: %w", play.Name, err)
		}
	}
	return nil
}

// ExecutePlay 执行单个 Play
func (r *Runner) ExecutePlay(play *Play) error {
	r.logger.PlayHeader(play.Name)

	// 设置 Play 变量
	r.varMgr.SetPlayVars(play.Vars)

	// 获取目标主机
	hosts, err := r.inventory.GetHosts(play.Hosts)
	if err != nil {
		return fmt.Errorf("failed to get hosts: %w", err)
	}

	if len(hosts) == 0 {
		return fmt.Errorf("no hosts matched pattern: %s", play.Hosts)
	}

	// 跟踪活跃主机（未失败的主机）
	activeHosts := make([]*inventory.Host, len(hosts))
	copy(activeHosts, hosts)

	// 主机统计
	stats := make(map[string]*HostStats)
	for _, host := range hosts {
		stats[host.Name] = &HostStats{}
	}

	// 执行所有任务
	for taskIdx, task := range play.Tasks {
		if len(activeHosts) == 0 {
			r.logger.Warning("No more hosts available, stopping play")
			break
		}

		// 显示任务名称
		taskName := task.Name
		if taskName == "" {
			taskName = task.Module
		}
		r.logger.TaskHeader(taskName)

		// 并发执行任务
		results := make(chan *TaskResult, len(activeHosts))
		var wg sync.WaitGroup

		for _, host := range activeHosts {
			wg.Add(1)
			go func(h *inventory.Host) {
				defer wg.Done()
				result := r.executeTask(&task, h)
				results <- result
			}(host)
		}

		// 等待所有任务完成
		go func() {
			wg.Wait()
			close(results)
		}()

		// 收集结果
		failedHosts := []string{}
		newActiveHosts := []*inventory.Host{}

		for result := range results {
			// 显示结果
			r.printTaskResult(result)

			// 更新统计
			hostStat := stats[result.Host]
			if result.Failed {
				hostStat.Failed++
			} else if result.Skipped {
				hostStat.Skipped++
			} else {
				hostStat.Ok++
				if result.Changed {
					hostStat.Changed++
				}
			}

			// 处理 register
			if task.Register != "" && !result.Failed {
				r.varMgr.SetHostVar(result.Host, task.Register, result.Data)
			}

			// 处理失败
			if result.Failed && !task.IgnoreErrors {
				failedHosts = append(failedHosts, result.Host)
			} else if !result.Failed {
				// 保留成功的主机
				for _, h := range activeHosts {
					if h.Name == result.Host {
						newActiveHosts = append(newActiveHosts, h)
						break
					}
				}
			}
		}

		// 更新活跃主机列表
		if len(failedHosts) > 0 {
			activeHosts = newActiveHosts
		}

		// 如果是最后一个任务，打印空行
		if taskIdx == len(play.Tasks)-1 {
			fmt.Println()
		}
	}

	// 打印 Play Recap
	r.printPlayRecap(play.Name, stats)

	// 检查是否有失败
	for _, stat := range stats {
		if !stat.IsSuccess() {
			return fmt.Errorf("play had failures")
		}
	}

	return nil
}

// executeTask 在单个主机上执行任务
func (r *Runner) executeTask(task *Task, host *inventory.Host) *TaskResult {
	result := &TaskResult{
		Host: host.Name,
		Task: task.Name,
		Data: make(map[string]interface{}),
	}

	// 获取主机变量上下文
	context := r.varMgr.GetContext(host.Name)

	// 评估 when 条件
	if task.When != "" {
		shouldRun, err := r.template.EvaluateCondition(task.When, context)
		if err != nil {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to evaluate when condition: %v", err)
			return result
		}
		if !shouldRun {
			result.Skipped = true
			result.Msg = "skipped due to when condition"
			return result
		}
	}

	// 渲染模块参数
	renderedArgs, err := r.template.RenderArgs(task.ModuleArgs, context)
	if err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to render args: %v", err)
		return result
	}

	// 规范化参数
	normalizedArgs := NormalizeModuleArgs(task.Module, renderedArgs)

	// 建立连接
	conn, err := r.connMgr.Connect(host)
	if err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("connection failed: %v", err)
		result.Data["unreachable"] = true
		return result
	}
	defer conn.Close()

	// 执行模块
	modResult, err := r.modExec.Execute(conn, task.Module, normalizedArgs)
	if err != nil {
		result.Failed = true
		result.Msg = err.Error()
		return result
	}

	// 转换结果
	result.Changed = modResult.Changed
	result.Failed = modResult.Failed || modResult.Unreachable
	result.Msg = modResult.Msg

	// 将模块结果转换为 map
	result.Data = map[string]interface{}{
		"changed":     modResult.Changed,
		"failed":      modResult.Failed,
		"unreachable": modResult.Unreachable,
		"msg":         modResult.Msg,
		"rc":          modResult.RC,
		"stdout":      modResult.Stdout,
		"stderr":      modResult.Stderr,
	}

	return result
}

// printTaskResult 打印任务结果
func (r *Runner) printTaskResult(result *TaskResult) {
	status := "ok"
	r.logger.TaskResult(status, result.Host, result.Msg, result.Changed, result.Failed, result.Skipped)
}

// printPlayRecap 打印 Play 总结
func (r *Runner) printPlayRecap(playName string, stats map[string]*HostStats) {
	// 转换为 logger.PlayStats
	loggerStats := make(map[string]*logger.PlayStats)
	for host, stat := range stats {
		loggerStats[host] = &logger.PlayStats{
			Ok:      stat.Ok,
			Changed: stat.Changed,
			Failed:  stat.Failed,
			Skipped: stat.Skipped,
		}
	}
	r.logger.PlayRecap(loggerStats)
}
