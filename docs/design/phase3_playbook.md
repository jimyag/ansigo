# 阶段 3 详细设计：剧本 (Playbook) 支持

## 目标
实现解析和执行简单的线性 Ansible Playbook。

## 1. Playbook 解析

### 1.1 YAML 结构
支持以下基础结构：
```yaml
- name: Play 1
  hosts: webservers
  gather_facts: no
  tasks:
    - name: Task 1
      ping:
    
    - name: Task 2
      command: uptime
      register: uptime_result
```

### 1.2 数据模型
```go
type Playbook []Play

type Play struct {
    Name        string
    Hosts       string
    GatherFacts bool `yaml:"gather_facts"`
    Vars        map[string]interface{}
    Tasks       []Task
}

type Task struct {
    Name         string
    Module       string                 // 在 UnmarshalYAML 中动态解析
    ModuleArgs   map[string]interface{} // 模块参数
    Register     string
    When         string
    IgnoreErrors bool `yaml:"ignore_errors"`
}

// Task 需要实现自定义 YAML 解析
// 解析逻辑：
// 1. 先识别标准字段（name, register, when, ignore_errors）
// 2. 遍历剩余字段，识别模块名（与已知模块列表匹配）
// 3. 解析模块参数：
//    - 短格式: "command: uptime" → Module="command", ModuleArgs={"_raw_params": "uptime"}
//    - 长格式: "command: {cmd: uptime}" → Module="command", ModuleArgs={"cmd": "uptime"}
```

## 2. 执行引擎 (Runner)

### 2.1 线性执行策略
Ansible 默认采用线性策略：所有主机并行执行 Task 1，等待全部完成后，再执行 Task 2。

**实现细节：**
```go
type PlaybookRunner struct {
    inventory *inventory.Inventory
    varMgr    *vars.VariableManager
    connMgr   *connection.Manager
}

func (r *PlaybookRunner) ExecutePlay(play *Play) error {
    // 1. 筛选目标主机
    hosts := r.inventory.GetHosts(play.Hosts)
    activeHosts := hosts // 跟踪未失败的主机列表

    // 2. 遍历任务（串行）
    for _, task := range play.Tasks {
        results := make(chan *TaskResult, len(activeHosts))

        // 3. 并发执行任务（所有活跃主机并行）
        for _, host := range activeHosts {
            go func(h *Host) {
                result := r.executeTask(task, h)
                results <- result
            }(host)
        }

        // 4. 收集结果并更新状态
        failedHosts := []string{}
        for i := 0; i < len(activeHosts); i++ {
            result := <-results

            // 处理 register
            if task.Register != "" {
                r.varMgr.SetHostVar(result.Host, task.Register, result.Data)
            }

            // 处理失败
            if result.Failed && !task.IgnoreErrors {
                failedHosts = append(failedHosts, result.Host)
            }
        }

        // 5. 从活跃列表移除失败主机
        activeHosts = removeHosts(activeHosts, failedHosts)

        if len(activeHosts) == 0 {
            return fmt.Errorf("all hosts failed")
        }
    }

    return nil
}
```

**失败主机处理：**
- 失败主机从后续任务中移除（除非 `ignore_errors: yes`）
- 保留失败主机信息用于最终报告

### 2.2 状态管理
*   **HostVars**: 存储每个主机的变量（Inventory 变量 + `register` 变量 + `set_fact`）。
*   **TaskResult**: 记录每个任务的执行结果。

```go
type TaskResult struct {
    Host     string
    Task     string
    Changed  bool
    Failed   bool
    Msg      string
    Data     map[string]interface{} // 完整的模块返回数据
}
```

### 2.3 变量管理器 (Variable Manager)

**职责：** 维护变量作用域和优先级

```go
type VariableManager struct {
    inventory     *inventory.Inventory
    playVars      map[string]interface{}                // Play 级别变量
    hostVars      map[string]map[string]interface{}     // hostname -> vars
    registeredVars map[string]map[string]interface{}    // hostname -> registered vars
}

// 获取主机的完整变量上下文（用于模板渲染）
func (vm *VariableManager) GetContext(hostname string) map[string]interface{} {
    context := make(map[string]interface{})

    // 1. 合并 inventory 变量（已经按优先级合并）
    if host, _ := vm.inventory.GetHost(hostname); host != nil {
        for k, v := range host.Vars {
            context[k] = v
        }
    }

    // 2. 合并 play 变量
    for k, v := range vm.playVars {
        context[k] = v
    }

    // 3. 合并 registered 变量（最高优先级）
    if hostRegs, ok := vm.registeredVars[hostname]; ok {
        for k, v := range hostRegs {
            context[k] = v
        }
    }

    // 4. 添加特殊变量
    context["inventory_hostname"] = hostname
    context["hostvars"] = vm.getAllHostVars()
    context["groups"] = vm.inventory.GetGroups()

    return context
}

func (vm *VariableManager) SetHostVar(hostname, key string, value interface{}) {
    if vm.registeredVars[hostname] == nil {
        vm.registeredVars[hostname] = make(map[string]interface{})
    }
    vm.registeredVars[hostname][key] = value
}
```

### 2.4 变量替换 (Templating)
*   **需求**: 支持 `{{ variable }}` 语法。
*   **实现**:
    *   集成 `github.com/flosch/pongo2` (Jinja2 的 Go 实现)。
    *   在任务执行前，对参数进行模板渲染。

```go
import "github.com/flosch/pongo2/v6"

type TemplateEngine struct {
    // pongo2 引擎
}

// 渲染模块参数
func (te *TemplateEngine) RenderArgs(args map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
    result := make(map[string]interface{})

    for key, value := range args {
        if strVal, ok := value.(string); ok {
            // 检查是否包含模板语法
            if strings.Contains(strVal, "{{") {
                tpl, err := pongo2.FromString(strVal)
                if err != nil {
                    return nil, err
                }
                rendered, err := tpl.Execute(context)
                if err != nil {
                    return nil, err
                }
                result[key] = rendered
            } else {
                result[key] = value
            }
        } else {
            result[key] = value
        }
    }

    return result, nil
}

// 评估 when 条件
func (te *TemplateEngine) EvaluateCondition(condition string, context map[string]interface{}) (bool, error) {
    // 将条件包装为 {{ condition }}
    tpl, err := pongo2.FromString("{{ " + condition + " }}")
    if err != nil {
        return false, err
    }

    result, err := tpl.Execute(context)
    if err != nil {
        return false, err
    }

    // 解析布尔结果
    return result == "true" || result == "True", nil
}
```

## 3. 核心流程
1.  **Load**: 读取 `site.yml`，解析为 `Playbook` 对象。
2.  **Inventory**: 根据 Play 的 `hosts` 字段筛选目标主机。
3.  **Play Loop**: 遍历 Play。
4.  **Task Loop**: 遍历 Play.Tasks。
    *   **Templating**: 使用当前 HostVars 渲染 Task 参数。
    *   **Execution**: 并发调用 Phase 2 的 Module Executor。
    *   **Result Handling**:
        *   如果失败且 `ignore_errors: no`，将主机标记为失败，从后续列表中移除。
        *   如果 `register` 存在，将结果存入 HostVars。
5.  **Report**: 输出回顾 (Play Recap)。

## 4. 限制 (MVP)

### 4.1 不支持的功能
*   不支持 `roles`
*   不支持 `handlers`
*   不支持复杂的 `include`/`import`
*   不支持 `become` (sudo)
*   不支持 `delegate_to`
*   不支持 `run_once`
*   不支持 `block` 和 `rescue`
*   不支持异步任务 (`async`/`poll`)
*   不支持 `serial` (批量执行)
*   不支持 `tags`

### 4.2 模板引擎限制
*   仅支持基本的变量替换 `{{ var }}`
*   支持简单的过滤器：`default`, `upper`, `lower`
*   不支持复杂的 Jinja2 控制结构（for, if 等）

## 5. 验证
*   创建一个包含 `ping` 和 `command` 的简单 Playbook。
*   运行 `ansigo-playbook site.yml`。
*   验证所有任务在所有主机上成功执行。
*   验证 `register` 变量可以被后续任务引用 (如果实现了模板引擎)。
