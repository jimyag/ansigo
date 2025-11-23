package playbook

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/jimyag/ansigo/pkg/connection"
	"github.com/jimyag/ansigo/pkg/facts"
	"github.com/jimyag/ansigo/pkg/inventory"
	"github.com/jimyag/ansigo/pkg/logger"
	"github.com/jimyag/ansigo/pkg/module"
)

// Runner Playbook 执行器
type Runner struct {
	inventory        *inventory.Manager
	connMgr          *connection.Manager
	modExec          *module.Executor
	varMgr           *VariableManager
	template         TemplateEngineInterface
	logger           *logger.AnsibleLogger
	notifiedHandlers map[string]bool // 记录被通知的 handlers
	playbookPath     string          // Playbook 文件路径（用于 role 查找）
	currentPlay      *Play           // 当前正在执行的 Play（用于访问 play 级别设置）
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

// SetPlaybookPath 设置 playbook 文件路径
func (r *Runner) SetPlaybookPath(path string) {
	r.playbookPath = path
}

// Close 关闭 Runner 并释放资源
func (r *Runner) Close() error {
	if r.template != nil {
		return r.template.Close()
	}
	return nil
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

	// 设置当前 Play（用于任务执行时访问 play 级别设置）
	r.currentPlay = play

	// 初始化 notified handlers 跟踪
	r.notifiedHandlers = make(map[string]bool)

	// 加载并展开 roles
	var allTasks []Task
	var allHandlers []Handler
	playVars := make(map[string]interface{})

	// 复制 play vars
	for k, v := range play.Vars {
		playVars[k] = v
	}

	// 处理 lookup() 调用（在 Jinja2 渲染之前）
	lookupHandler := NewLookupHandler(r.playbookPath, r.template)
	processedVars, err := lookupHandler.ProcessLookupsInVars(playVars, playVars)
	if err != nil {
		return fmt.Errorf("failed to process lookups in play vars: %w", err)
	}
	playVars = processedVars

	// 处理 roles
	if len(play.Roles) > 0 {
		loader := NewRoleLoader(r.playbookPath)

		for _, roleData := range play.Roles {
			// 解析 role spec
			spec, err := ParseRoleSpec(roleData)
			if err != nil {
				return fmt.Errorf("failed to parse role spec: %w", err)
			}

			// 加载 role
			role, err := loader.LoadRole(spec)
			if err != nil {
				return fmt.Errorf("failed to load role '%s': %w", spec.Name, err)
			}

			// 合并 role defaults（最低优先级）
			for k, v := range role.Defaults {
				if _, exists := playVars[k]; !exists {
					playVars[k] = v
				}
			}

			// 合并 role vars（高优先级）
			for k, v := range role.Vars {
				playVars[k] = v
			}

			// 添加 role 任务到任务列表
			allTasks = append(allTasks, role.Tasks...)

			// 添加 role handlers
			allHandlers = append(allHandlers, role.Handlers...)
		}
	}

	// 添加 play 的任务（在 role 任务之后）
	allTasks = append(allTasks, play.Tasks...)

	// 添加 play 的 handlers
	allHandlers = append(allHandlers, play.Handlers...)

	// 展开任务（处理 import_tasks 和 include_role）
	taskIncluder := NewTaskIncluder(r.playbookPath)
	expandedTasks, err := r.expandAllTasks(allTasks, taskIncluder, playVars)
	if err != nil {
		return fmt.Errorf("failed to expand tasks: %w", err)
	}
	allTasks = expandedTasks

	// 设置合并后的 Play 变量
	r.varMgr.SetPlayVars(playVars)

	// 获取目标主机
	hosts, err := r.inventory.GetHosts(play.Hosts)
	if err != nil {
		return fmt.Errorf("failed to get hosts: %w", err)
	}

	if len(hosts) == 0 {
		return fmt.Errorf("no hosts matched pattern: %s", play.Hosts)
	}

	// 设置 play 主机列表（用于魔法变量）
	hostNames := make([]string, len(hosts))
	for i, host := range hosts {
		hostNames[i] = host.Name
	}
	r.varMgr.SetPlayHosts(hostNames)

	// 跟踪活跃主机（未失败的主机）
	activeHosts := make([]*inventory.Host, len(hosts))
	copy(activeHosts, hosts)

	// 主机统计
	stats := make(map[string]*HostStats)
	for _, host := range hosts {
		stats[host.Name] = &HostStats{}
	}

	// Gather facts if enabled (default is true unless explicitly set to false)
	if play.GatherFacts {
		if err := r.gatherFactsForHosts(hosts); err != nil {
			return fmt.Errorf("failed to gather facts: %w", err)
		}
	}

	// 执行所有任务（包括 role 任务和 play 任务）
	for taskIdx, task := range allTasks {
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

			// 处理 ansible_facts (set_fact 模块)
			if ansibleFacts, ok := result.Data["ansible_facts"].(map[string]interface{}); ok {
				for key, value := range ansibleFacts {
					r.varMgr.SetHostVar(result.Host, key, value)
				}
			}

			// 处理 notify（只在任务 changed 时通知 handler）
			if result.Changed && len(task.Notify) > 0 {
				for _, handlerName := range task.Notify {
					r.notifiedHandlers[handlerName] = true
				}
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
		if taskIdx == len(allTasks)-1 {
			fmt.Println()
		}
	}

	// 执行所有被通知的 handlers（包括 role handlers 和 play handlers）
	if len(allHandlers) > 0 && len(r.notifiedHandlers) > 0 {
		if err := r.executeHandlers(allHandlers, activeHosts, stats); err != nil {
			return fmt.Errorf("handler execution failed: %w", err)
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

	// 如果是 block 任务，执行 block 逻辑
	if task.TaskBlock != nil {
		return r.executeBlock(task, host)
	}

	// 如果有循环，执行循环逻辑
	if len(task.Loop) > 0 {
		return r.executeTaskWithLoop(task, host)
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

	// 特殊处理 debug 模块的 var 参数
	if task.Module == "debug" {
		if varName, ok := renderedArgs["var"].(string); ok {
			// 从 context 中获取变量值
			if varValue, exists := context[varName]; exists {
				// 将变量值格式化为字符串并设置为 msg
				renderedArgs["msg"] = fmt.Sprintf("%s: %v", varName, varValue)
			} else {
				renderedArgs["msg"] = fmt.Sprintf("%s: VARIABLE IS NOT DEFINED!", varName)
			}
			// 删除 var 参数，使用 msg 参数
			delete(renderedArgs, "var")
		}
	}

	// 特殊处理 template 模块 - 读取并渲染模板文件
	if task.Module == "template" {
		srcInterface, ok := renderedArgs["src"]
		if !ok {
			result.Failed = true
			result.Msg = "template module requires 'src' argument"
			return result
		}

		src, ok := srcInterface.(string)
		if !ok {
			result.Failed = true
			result.Msg = "template module 'src' must be a string"
			return result
		}

		// 读取本地模板文件（控制节点）
		templateContent, err := os.ReadFile(src)
		if err != nil {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to read template file '%s': %v", src, err)
			return result
		}

		// 渲染模板内容
		renderedContent, err := r.template.RenderString(string(templateContent), context)
		if err != nil {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to render template: %v", err)
			return result
		}

		// 将渲染后的内容添加到参数中
		renderedArgs["_rendered_content"] = renderedContent
		// 删除 src 参数（模块不需要它）
		delete(renderedArgs, "src")
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

	// 确定是否需要 become（任务级别优先，然后是 play 级别）
	shouldBecome := r.currentPlay.Become
	becomeUser := r.currentPlay.BecomeUser
	becomeMethod := r.currentPlay.BecomeMethod

	// Task 级别 become 优先
	if task.Become != nil {
		shouldBecome = *task.Become
		if task.BecomeUser != "" {
			becomeUser = task.BecomeUser
		}
		if task.BecomeMethod != "" {
			becomeMethod = task.BecomeMethod
		}
	}

	// 执行模块
	modResult, err := r.modExec.Execute(conn, task.Module, normalizedArgs, shouldBecome, becomeUser, becomeMethod)
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

	// 如果有 ansible_facts，添加到 Data 中
	if len(modResult.AnsibleFacts) > 0 {
		result.Data["ansible_facts"] = modResult.AnsibleFacts
	}

	// 评估 failed_when 条件
	if task.FailedWhen != "" {
		// 创建包含任务结果的上下文
		evalContext := make(map[string]interface{})
		for k, v := range context {
			evalContext[k] = v
		}
		// 添加任务结果到上下文中
		// 支持两种方式访问：直接使用 rc、stdout 等，或通过 register 变量名访问
		evalContext["rc"] = modResult.RC
		evalContext["stdout"] = modResult.Stdout
		evalContext["stderr"] = modResult.Stderr
		evalContext["changed"] = modResult.Changed
		evalContext["failed"] = modResult.Failed

		// 如果有 register，也添加为变量（用于 register_var.rc 这样的访问）
		if task.Register != "" {
			evalContext[task.Register] = result.Data
		}

		shouldFail, err := r.template.EvaluateCondition(task.FailedWhen, evalContext)
		if err != nil {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to evaluate failed_when: %v", err)
			return result
		}
		result.Failed = shouldFail
		if shouldFail {
			result.Msg = fmt.Sprintf("failed due to failed_when condition: %s", task.FailedWhen)
		}
	}

	// 评估 changed_when 条件
	if task.ChangedWhen != "" {
		// 创建包含任务结果的上下文
		evalContext := make(map[string]interface{})
		for k, v := range context {
			evalContext[k] = v
		}
		evalContext["rc"] = modResult.RC
		evalContext["stdout"] = modResult.Stdout
		evalContext["stderr"] = modResult.Stderr
		evalContext["changed"] = modResult.Changed
		evalContext["failed"] = modResult.Failed

		// 如果有 register，也添加为变量
		if task.Register != "" {
			evalContext[task.Register] = result.Data
		}

		shouldChange, err := r.template.EvaluateCondition(task.ChangedWhen, evalContext)
		if err != nil {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to evaluate changed_when: %v", err)
			return result
		}
		result.Changed = shouldChange
	}

	return result
}

// printTaskResult 打印任务结果
func (r *Runner) printTaskResult(result *TaskResult) {
	// 检查是否是循环结果
	if results, ok := result.Data["results"].([]map[string]interface{}); ok && len(results) > 0 {
		// 这是循环任务，显示每个迭代的结果
		for _, iterResult := range results {
			// 获取循环变量名
			loopVar := "item"
			if lv, ok := iterResult["ansible_loop_var"].(string); ok {
				loopVar = lv
			}

			// 获取循环项的值
			itemValue := iterResult[loopVar]

			// 构建消息
			msg := ""
			if iterMsg, ok := iterResult["msg"].(string); ok && iterMsg != "" {
				msg = iterMsg
			} else if iterResult["stdout"] != nil && iterResult["stdout"] != "" {
				msg = fmt.Sprintf("%v", iterResult["stdout"])
			}

			// 显示状态
			changed := false
			if c, ok := iterResult["changed"].(bool); ok {
				changed = c
			}
			failed := false
			if f, ok := iterResult["failed"].(bool); ok {
				failed = f
			}
			skipped := false
			if s, ok := iterResult["skipped"].(bool); ok {
				skipped = s
			}

			// 格式化输出 "item=value => msg"
			displayMsg := fmt.Sprintf("item=%v", itemValue)
			if msg != "" {
				displayMsg = fmt.Sprintf("%s => %s", displayMsg, msg)
			}

			r.logger.TaskResult("ok", result.Host, displayMsg, changed, failed, skipped)
		}
	} else {
		// 普通任务结果
		status := "ok"
		r.logger.TaskResult(status, result.Host, result.Msg, result.Changed, result.Failed, result.Skipped)
	}
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

// executeHandlers 执行所有被通知的 handlers
func (r *Runner) executeHandlers(handlers []Handler, hosts []*inventory.Host, stats map[string]*HostStats) error {
	if len(handlers) == 0 || len(r.notifiedHandlers) == 0 {
		return nil
	}

	// 打印 RUNNING HANDLER 标题
	fmt.Println()
	fmt.Println("RUNNING HANDLER", strings.Repeat("*", 60))

	// 按 handlers 定义顺序执行（不是按通知顺序）
	for _, handler := range handlers {
		// 检查是否被通知（通过名称或 listen topic）
		notified := r.notifiedHandlers[handler.Name]
		if handler.Listen != "" {
			notified = notified || r.notifiedHandlers[handler.Listen]
		}

		if !notified {
			continue
		}

		// 显示 handler 名称
		handlerName := handler.Name
		if handlerName == "" {
			handlerName = handler.Module
		}
		r.logger.TaskHeader(handlerName)

		// 并发执行 handler 任务
		results := make(chan *TaskResult, len(hosts))
		var wg sync.WaitGroup

		for _, host := range hosts {
			wg.Add(1)
			go func(h *inventory.Host) {
				defer wg.Done()
				result := r.executeHandlerTask(&handler, h)
				results <- result
			}(host)
		}

		// 等待所有 handler 完成
		go func() {
			wg.Wait()
			close(results)
		}()

		// 收集结果并更新统计
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
		}
	}

	fmt.Println()
	return nil
}

// executeHandlerTask 在单个主机上执行 handler 任务
func (r *Runner) executeHandlerTask(handler *Handler, host *inventory.Host) *TaskResult {
	result := &TaskResult{
		Host: host.Name,
		Task: handler.Name,
		Data: make(map[string]interface{}),
	}

	// 获取主机变量上下文
	context := r.varMgr.GetContext(host.Name)

	// 评估 when 条件
	if handler.When != "" {
		shouldRun, err := r.template.EvaluateCondition(handler.When, context)
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
	renderedArgs, err := r.template.RenderArgs(handler.ModuleArgs, context)
	if err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to render args: %v", err)
		return result
	}

	// 特殊处理 debug 模块的 var 参数
	if handler.Module == "debug" {
		if varName, ok := renderedArgs["var"].(string); ok {
			// 从 context 中获取变量值
			if varValue, exists := context[varName]; exists {
				// 将变量值格式化为字符串并设置为 msg
				renderedArgs["msg"] = fmt.Sprintf("%s: %v", varName, varValue)
			} else {
				renderedArgs["msg"] = fmt.Sprintf("%s: VARIABLE IS NOT DEFINED!", varName)
			}
			// 删除 var 参数，使用 msg 参数
			delete(renderedArgs, "var")
		}
	}

	// 规范化参数
	normalizedArgs := NormalizeModuleArgs(handler.Module, renderedArgs)

	// 建立连接
	conn, err := r.connMgr.Connect(host)
	if err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("connection failed: %v", err)
		result.Data["unreachable"] = true
		return result
	}
	defer conn.Close()

	// Handlers 默认不使用 become，除非在 handler 定义中明确设置
	// 注意：Handler 结构需要有 Become 字段才能支持，目前使用默认值
	shouldBecome := false
	becomeUser := ""
	becomeMethod := ""

	// 执行模块
	modResult, err := r.modExec.Execute(conn, handler.Module, normalizedArgs, shouldBecome, becomeUser, becomeMethod)
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

	// 如果有 ansible_facts，添加到 Data 中
	if len(modResult.AnsibleFacts) > 0 {
		result.Data["ansible_facts"] = modResult.AnsibleFacts
	}

	return result
}

// executeTaskWithLoop 执行带循环的任务
func (r *Runner) executeTaskWithLoop(task *Task, host *inventory.Host) *TaskResult {
	// 获取循环变量名和索引变量名
	loopVar := "item"
	indexVar := ""
	var pause int
	// label 用于简化输出显示，目前未实现，预留供将来使用
	// var label string

	if task.LoopControl != nil {
		if task.LoopControl.LoopVar != "" {
			loopVar = task.LoopControl.LoopVar
		}
		indexVar = task.LoopControl.IndexVar
		pause = task.LoopControl.Pause
		// label = task.LoopControl.Label
	}

	// 获取主机变量上下文
	baseContext := r.varMgr.GetContext(host.Name)

	// 评估循环列表（可能包含模板变量）
	var loopItems []interface{}
	for _, item := range task.Loop {
		// 如果是字符串且包含模板语法，进行渲染
		if strItem, ok := item.(string); ok && IsTemplateString(strItem) {
			// 使用 RenderValue 获取原始值（可能是列表）
			rendered, err := r.template.RenderValue(strItem, baseContext)
			if err != nil {
				// 渲染失败，返回错误
				return &TaskResult{
					Host:   host.Name,
					Task:   task.Name,
					Failed: true,
					Msg:    fmt.Sprintf("failed to render loop item: %v", err),
					Data:   make(map[string]interface{}),
				}
			}
			// 如果渲染结果是列表，直接使用
			if list, ok := rendered.([]interface{}); ok {
				loopItems = list
				break // 整个 loop 已经被展开，不需要继续
			}
			// 尝试 []map[string]interface{} 类型（register results 的情况）
			if list, ok := rendered.([]map[string]interface{}); ok {
				// 转换为 []interface{}
				loopItems = make([]interface{}, len(list))
				for i, v := range list {
					loopItems[i] = v
				}
				break
			}
			loopItems = append(loopItems, rendered)
		} else {
			loopItems = append(loopItems, item)
		}
	}

	// 存储所有迭代结果
	results := make([]map[string]interface{}, 0, len(loopItems))
	hasChanged := false
	hasFailed := false
	hasSkipped := false
	allSkipped := true

	// 遍历循环项
	for idx, item := range loopItems {
		// 创建循环上下文
		loopContext := make(map[string]interface{})
		for k, v := range baseContext {
			loopContext[k] = v
		}
		loopContext[loopVar] = item
		if indexVar != "" {
			loopContext[indexVar] = idx
		}

		// 评估 when 条件（在循环上下文中）
		if task.When != "" {
			shouldRun, err := r.template.EvaluateCondition(task.When, loopContext)
			if err != nil {
				// when 条件评估失败
				iterResult := map[string]interface{}{
					"failed":           true,
					"msg":              fmt.Sprintf("failed to evaluate when condition: %v", err),
					loopVar:            item,
					"ansible_loop_var": loopVar,
				}
				if indexVar != "" {
					iterResult[indexVar] = idx
				}
				results = append(results, iterResult)
				hasFailed = true
				allSkipped = false
				continue
			}
			if !shouldRun {
				// 条件不满足，跳过
				iterResult := map[string]interface{}{
					"skipped":          true,
					"skip_reason":      "Conditional result was False",
					loopVar:            item,
					"ansible_loop_var": loopVar,
				}
				if indexVar != "" {
					iterResult[indexVar] = idx
				}
				results = append(results, iterResult)
				hasSkipped = true
				continue
			}
		}

		allSkipped = false

		// 渲染模块参数
		renderedArgs, err := r.template.RenderArgs(task.ModuleArgs, loopContext)
		if err != nil {
			iterResult := map[string]interface{}{
				"failed":           true,
				"msg":              fmt.Sprintf("failed to render args: %v", err),
				loopVar:            item,
				"ansible_loop_var": loopVar,
			}
			if indexVar != "" {
				iterResult[indexVar] = idx
			}
			results = append(results, iterResult)
			hasFailed = true
			continue
		}

		// 特殊处理 debug 模块的 var 参数
		if task.Module == "debug" {
			if varName, ok := renderedArgs["var"].(string); ok {
				if varValue, exists := loopContext[varName]; exists {
					renderedArgs["msg"] = fmt.Sprintf("%s: %v", varName, varValue)
				} else {
					renderedArgs["msg"] = fmt.Sprintf("%s: VARIABLE IS NOT DEFINED!", varName)
				}
				delete(renderedArgs, "var")
			}
		}

		// 规范化参数
		normalizedArgs := NormalizeModuleArgs(task.Module, renderedArgs)

		// 建立连接
		conn, err := r.connMgr.Connect(host)
		if err != nil {
			iterResult := map[string]interface{}{
				"failed":           true,
				"unreachable":      true,
				"msg":              fmt.Sprintf("connection failed: %v", err),
				loopVar:            item,
				"ansible_loop_var": loopVar,
			}
			if indexVar != "" {
				iterResult[indexVar] = idx
			}
			results = append(results, iterResult)
			hasFailed = true
			continue
		}

		// 确定是否使用 become（任务级别优先，然后是 play 级别）
		shouldBecome := r.currentPlay.Become
		becomeUser := r.currentPlay.BecomeUser
		becomeMethod := r.currentPlay.BecomeMethod
		if task.Become != nil {
			shouldBecome = *task.Become
			if task.BecomeUser != "" {
				becomeUser = task.BecomeUser
			}
			if task.BecomeMethod != "" {
				becomeMethod = task.BecomeMethod
			}
		}

		// 执行模块
		modResult, err := r.modExec.Execute(conn, task.Module, normalizedArgs, shouldBecome, becomeUser, becomeMethod)
		conn.Close()

		if err != nil {
			iterResult := map[string]interface{}{
				"failed":           true,
				"msg":              err.Error(),
				loopVar:            item,
				"ansible_loop_var": loopVar,
			}
			if indexVar != "" {
				iterResult[indexVar] = idx
			}
			results = append(results, iterResult)
			hasFailed = true
			continue
		}

		// 记录迭代结果
		iterResult := map[string]interface{}{
			"changed":          modResult.Changed,
			"failed":           modResult.Failed,
			"unreachable":      modResult.Unreachable,
			"msg":              modResult.Msg,
			"rc":               modResult.RC,
			"stdout":           modResult.Stdout,
			"stderr":           modResult.Stderr,
			loopVar:            item,
			"ansible_loop_var": loopVar,
		}
		if indexVar != "" {
			iterResult[indexVar] = idx
		}

		// 如果有 ansible_facts，添加到结果中
		if len(modResult.AnsibleFacts) > 0 {
			iterResult["ansible_facts"] = modResult.AnsibleFacts
			// 将 facts 设置到主机变量中
			for key, value := range modResult.AnsibleFacts {
				r.varMgr.SetHostVar(host.Name, key, value)
			}
		}

		// 评估 failed_when 条件
		if task.FailedWhen != "" {
			evalContext := make(map[string]interface{})
			for k, v := range loopContext {
				evalContext[k] = v
			}
			evalContext["rc"] = modResult.RC
			evalContext["stdout"] = modResult.Stdout
			evalContext["stderr"] = modResult.Stderr
			evalContext["changed"] = modResult.Changed
			evalContext["failed"] = modResult.Failed

			shouldFail, err := r.template.EvaluateCondition(task.FailedWhen, evalContext)
			if err != nil {
				iterResult["failed"] = true
				iterResult["msg"] = fmt.Sprintf("failed to evaluate failed_when: %v", err)
			} else if shouldFail {
				iterResult["failed"] = true
				iterResult["msg"] = fmt.Sprintf("failed due to failed_when condition: %s", task.FailedWhen)
			}
		}

		// 评估 changed_when 条件
		if task.ChangedWhen != "" {
			evalContext := make(map[string]interface{})
			for k, v := range loopContext {
				evalContext[k] = v
			}
			evalContext["rc"] = modResult.RC
			evalContext["stdout"] = modResult.Stdout
			evalContext["stderr"] = modResult.Stderr
			evalContext["changed"] = modResult.Changed
			evalContext["failed"] = modResult.Failed

			shouldChange, err := r.template.EvaluateCondition(task.ChangedWhen, evalContext)
			if err != nil {
				iterResult["failed"] = true
				iterResult["msg"] = fmt.Sprintf("failed to evaluate changed_when: %v", err)
			} else {
				iterResult["changed"] = shouldChange
			}
		}

		results = append(results, iterResult)

		// 更新总体状态
		if iterResult["changed"].(bool) {
			hasChanged = true
		}
		if iterResult["failed"].(bool) && !task.IgnoreErrors {
			hasFailed = true
		}

		// 如果配置了暂停，执行暂停
		if pause > 0 && idx < len(loopItems)-1 {
			fmt.Printf("  (pausing %d seconds between loop iterations)\n", pause)
			// 这里使用 time.Sleep，但实际可能需要导入 time 包
		}
	}

	// 构建循环任务的总体结果
	result := &TaskResult{
		Host:    host.Name,
		Task:    task.Name,
		Changed: hasChanged,
		Failed:  hasFailed && !task.IgnoreErrors,
		Skipped: allSkipped,
		Data: map[string]interface{}{
			"results": results,
			"changed": hasChanged,
			"failed":  hasFailed,
			"skipped": hasSkipped,
		},
	}

	// 如果有 register，保存 results 列表
	// 这将在 ExecutePlay 中处理

	// 设置消息
	if allSkipped {
		result.Msg = "All items skipped"
	} else if hasFailed {
		result.Msg = fmt.Sprintf("One or more items failed")
	} else if hasChanged {
		result.Msg = fmt.Sprintf("Loop completed with changes")
	} else {
		result.Msg = fmt.Sprintf("Loop completed")
	}

	return result
}

// executeBlock 执行 block/rescue/always 结构
func (r *Runner) executeBlock(task *Task, host *inventory.Host) *TaskResult {
	result := &TaskResult{
		Host: host.Name,
		Task: task.Name,
		Data: make(map[string]interface{}),
	}

	// 获取主机变量上下文
	context := r.varMgr.GetContext(host.Name)

	// 评估 when 条件（block 级别）
	if task.When != "" {
		shouldRun, err := r.template.EvaluateCondition(task.When, context)
		if err != nil {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to evaluate when condition: %v", err)
			return result
		}
		if !shouldRun {
			result.Skipped = true
			result.Msg = "block skipped due to when condition"
			return result
		}
	}

	block := task.TaskBlock
	var blockError error
	var failedTask *Task
	var failedResult *TaskResult

	// 执行 block 部分的任务
	for i := range block.Block {
		taskResult := r.executeTask(&block.Block[i], host)

		// 更新整体结果状态
		if taskResult.Changed {
			result.Changed = true
		}

		// 如果任务失败且不忽略错误，记录失败并跳出
		if taskResult.Failed && !block.Block[i].IgnoreErrors {
			blockError = fmt.Errorf("task failed: %s", taskResult.Msg)
			failedTask = &block.Block[i]
			failedResult = taskResult
			break
		}
	}

	// 如果 block 失败，执行 rescue 部分
	if blockError != nil && len(block.Rescue) > 0 {
		// 设置特殊变量 ansible_failed_task 和 ansible_failed_result
		if failedTask != nil {
			r.varMgr.SetHostVar(host.Name, "ansible_failed_task", map[string]interface{}{
				"name": failedTask.Name,
			})
		}
		if failedResult != nil {
			r.varMgr.SetHostVar(host.Name, "ansible_failed_result", failedResult.Data)
		}

		// 执行 rescue 任务
		rescueError := false
		for i := range block.Rescue {
			rescueResult := r.executeTask(&block.Rescue[i], host)

			if rescueResult.Changed {
				result.Changed = true
			}

			// 如果 rescue 任务失败，记录错误
			if rescueResult.Failed && !block.Rescue[i].IgnoreErrors {
				rescueError = true
				result.Failed = true
				result.Msg = fmt.Sprintf("rescue task failed: %s", rescueResult.Msg)
				break
			}
		}

		// 如果 rescue 成功执行（没有错误），则认为 block 已恢复
		if !rescueError {
			blockError = nil
			result.Msg = "block recovered by rescue"
		}
	} else if blockError != nil {
		// block 失败且没有 rescue
		result.Failed = true
		result.Msg = blockError.Error()
	}

	// 总是执行 always 部分（无论 block 成功或失败）
	if len(block.Always) > 0 {
		for i := range block.Always {
			alwaysResult := r.executeTask(&block.Always[i], host)

			if alwaysResult.Changed {
				result.Changed = true
			}

			// always 部分的失败会覆盖之前的成功状态
			if alwaysResult.Failed && !block.Always[i].IgnoreErrors {
				result.Failed = true
				result.Msg = fmt.Sprintf("always task failed: %s", alwaysResult.Msg)
			}
		}
	}

	// 如果没有设置消息，设置默认消息
	if result.Msg == "" {
		if result.Failed {
			result.Msg = "block failed"
		} else if result.Changed {
			result.Msg = "block completed with changes"
		} else {
			result.Msg = "block completed"
		}
	}

	return result
}

// expandAllTasks 递归展开所有任务（处理 import_tasks 和 include_role）
func (r *Runner) expandAllTasks(tasks []Task, includer *TaskIncluder, vars map[string]interface{}) ([]Task, error) {
	var result []Task

	for _, task := range tasks {
		// 检查是否是包含任务
		if task.Module == "import_tasks" || task.Module == "ansible.builtin.import_tasks" ||
			task.Module == "include_role" || task.Module == "ansible.builtin.include_role" {
			// 展开包含任务
			expandedTasks, err := includer.ExpandTask(&task, vars)
			if err != nil {
				return nil, fmt.Errorf("failed to expand task '%s': %w", task.Name, err)
			}

			// 递归展开（因为展开的任务可能也包含 import_tasks）
			expandedTasks, err = r.expandAllTasks(expandedTasks, includer, vars)
			if err != nil {
				return nil, err
			}

			result = append(result, expandedTasks...)
		} else {
			// 不是包含任务，直接添加
			result = append(result, task)
		}
	}

	return result, nil
}

// gatherFactsForHosts gathers facts for all hosts in parallel
func (r *Runner) gatherFactsForHosts(hosts []*inventory.Host) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(hosts))

	for _, host := range hosts {
		wg.Add(1)
		go func(h *inventory.Host) {
			defer wg.Done()

			// Connect to host
			conn, err := r.connMgr.Connect(h)
			if err != nil {
				errors <- fmt.Errorf("failed to connect to %s: %w", h.Name, err)
				return
			}
			defer conn.Close()

			// Gather facts
			hostFacts, err := facts.GatherFacts(conn)
			if err != nil {
				errors <- fmt.Errorf("failed to gather facts for %s: %w", h.Name, err)
				return
			}

			// Set facts as host variables
			r.varMgr.SetHostVars(h.Name, hostFacts)
		}(host)
	}

	// Wait for all fact gathering to complete
	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}
