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
    Tasks       []Task
}

type Task struct {
    Name      string
    Module    string // 需要自定义 UnmarshalYAML 来解析 "ping:", "command:" 等键值对
    Args      map[string]interface{}
    Register  string
    When      string
    IgnoreErrors bool `yaml:"ignore_errors"`
}
```

## 2. 执行引擎 (Runner)

### 2.1 线性执行策略
Ansible 默认采用线性策略：所有主机并行执行 Task 1，等待全部完成后，再执行 Task 2。

### 2.2 状态管理
*   **HostVars**: 存储每个主机的变量（Inventory 变量 + `register` 变量 + `set_fact`）。
*   **TaskResult**: 记录每个任务的执行结果。

### 2.3 变量替换 (Templating)
*   **需求**: 支持 `{{ variable }}` 语法。
*   **实现**:
    *   集成 Go 的模板引擎 (如 `text/template`) 或寻找兼容 Jinja2 的 Go 库 (如 `github.com/flosch/pongo2`)。
    *   在任务执行前，对参数进行模板渲染。

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
*   不支持 `roles`。
*   不支持 `handlers`。
*   不支持复杂的 `include`/`import`。
*   不支持 `become` (sudo)。

## 5. 验证
*   创建一个包含 `ping` 和 `command` 的简单 Playbook。
*   运行 `ansigo-playbook site.yml`。
*   验证所有任务在所有主机上成功执行。
*   验证 `register` 变量可以被后续任务引用 (如果实现了模板引擎)。
