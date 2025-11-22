# 错误处理设计

## 1. 错误分类

AnsiGo 定义了统一的错误类型系统，用于在多层架构中传递错误上下文。

### 1.1 错误类型定义

```go
package errors

import (
    "fmt"
)

// ErrorType 定义错误类型
type ErrorType int

const (
    // ErrUnreachable 主机不可达（连接失败）
    ErrUnreachable ErrorType = iota

    // ErrFailed 模块执行失败
    ErrFailed

    // ErrTimeout 执行超时
    ErrTimeout

    // ErrParse 解析错误（Inventory、Playbook 等）
    ErrParse

    // ErrInvalidArgs 参数错误
    ErrInvalidArgs

    // ErrModuleNotFound 模块未找到
    ErrModuleNotFound
)

// ExecutionError 统一的执行错误类型
type ExecutionError struct {
    Type      ErrorType              // 错误类型
    Host      string                 // 目标主机（如果适用）
    Task      string                 // 任务名称（如果适用）
    Module    string                 // 模块名称（如果适用）
    Message   string                 // 错误消息
    Cause     error                  // 原始错误
    Retriable bool                   // 是否可重试
    Details   map[string]interface{} // 额外的错误详情
}

func (e *ExecutionError) Error() string {
    if e.Host != "" {
        return fmt.Sprintf("[%s] %s: %s", e.Host, e.Task, e.Message)
    }
    return e.Message
}

func (e *ExecutionError) Unwrap() error {
    return e.Cause
}

// 错误构造函数
func NewUnreachableError(host string, cause error) *ExecutionError {
    return &ExecutionError{
        Type:      ErrUnreachable,
        Host:      host,
        Message:   fmt.Sprintf("Failed to connect to host: %v", cause),
        Cause:     cause,
        Retriable: true,
    }
}

func NewModuleFailedError(host, task, module, msg string) *ExecutionError {
    return &ExecutionError{
        Type:      ErrFailed,
        Host:      host,
        Task:      task,
        Module:    module,
        Message:   msg,
        Retriable: false,
    }
}

func NewTimeoutError(host, task string, duration time.Duration) *ExecutionError {
    return &ExecutionError{
        Type:      ErrTimeout,
        Host:      host,
        Task:      task,
        Message:   fmt.Sprintf("Task timeout after %v", duration),
        Retriable: true,
    }
}

func NewParseError(filePath string, cause error) *ExecutionError {
    return &ExecutionError{
        Type:      ErrParse,
        Message:   fmt.Sprintf("Failed to parse %s: %v", filePath, cause),
        Cause:     cause,
        Retriable: false,
    }
}
```

## 2. 错误处理策略

### 2.1 连接错误处理

**场景：** SSH 连接失败、网络超时、认证失败

**处理策略：**
- 标记主机为 `UNREACHABLE`
- 继续执行其他主机的任务
- 在最终报告中显示为 UNREACHABLE
- 不重试（除非用户配置了重试参数）

```go
func (cm *ConnectionManager) Connect(host *Host) (*Connection, error) {
    conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.IP, host.Port), config)
    if err != nil {
        return nil, NewUnreachableError(host.Name, err)
    }
    return &Connection{client: conn}, nil
}

// Ad-hoc Runner 中的处理
func (r *AdhocRunner) executeOnHost(host *Host, module string, args map[string]interface{}) *TaskResult {
    conn, err := r.connMgr.Connect(host)
    if err != nil {
        var execErr *ExecutionError
        if errors.As(err, &execErr) && execErr.Type == ErrUnreachable {
            return &TaskResult{
                Host:        host.Name,
                Unreachable: true,
                Msg:         execErr.Message,
            }
        }
        // 其他错误也标记为 UNREACHABLE
        return &TaskResult{
            Host:        host.Name,
            Unreachable: true,
            Msg:         err.Error(),
        }
    }
    defer conn.Close()

    // 执行模块...
}
```

### 2.2 模块执行失败

**场景：** 模块返回 failed: true、命令返回非零退出码

**处理策略：**
- 标记任务为 FAILED
- 根据 `ignore_errors` 决定是否继续
- 保存完整的错误信息用于报告

```go
type ModuleResult struct {
    Changed bool                   `json:"changed"`
    Failed  bool                   `json:"failed"`
    Msg     string                 `json:"msg,omitempty"`
    RC      int                    `json:"rc,omitempty"`
    Stdout  string                 `json:"stdout,omitempty"`
    Stderr  string                 `json:"stderr,omitempty"`
    Module  map[string]interface{} `json:"-"` // 其他模块特定字段
}

func (me *ModuleExecutor) Execute(conn *Connection, module string, args map[string]interface{}) (*ModuleResult, error) {
    // 执行模块...

    result := parseModuleOutput(stdout)

    if result.Failed {
        return result, NewModuleFailedError(
            conn.Host.Name,
            module,
            module,
            result.Msg,
        )
    }

    return result, nil
}
```

### 2.3 解析错误

**场景：** Inventory 文件格式错误、Playbook YAML 语法错误

**处理策略：**
- 立即中止执行（致命错误）
- 显示友好的错误提示（包含文件路径和行号）
- 提供修复建议

```go
func (p *InventoryParser) ParseINI(filePath string) (*Inventory, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read inventory file: %w", err)
    }

    inv := &Inventory{
        Hosts:  make(map[string]*Host),
        Groups: make(map[string]*Group),
    }

    scanner := bufio.NewScanner(bytes.NewReader(data))
    lineNum := 0
    currentGroup := ""

    for scanner.Scan() {
        lineNum++
        line := strings.TrimSpace(scanner.Text())

        // 解析逻辑...

        if err := parseLine(line, inv, &currentGroup); err != nil {
            return nil, &ExecutionError{
                Type:    ErrParse,
                Message: fmt.Sprintf("Syntax error at %s:%d: %v", filePath, lineNum, err),
                Cause:   err,
                Details: map[string]interface{}{
                    "file": filePath,
                    "line": lineNum,
                    "content": line,
                },
            }
        }
    }

    return inv, nil
}
```

### 2.4 超时错误

**场景：** 模块执行时间过长

**处理策略：**
- 使用 context.WithTimeout 控制执行时间
- 超时后立即终止连接
- 标记为 TIMEOUT 错误
- 可配置超时时间（默认 30 秒）

```go
func (conn *Connection) ExecWithTimeout(cmd string, timeout time.Duration) (stdout, stderr []byte, exitCode int, err error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    session, err := conn.client.NewSession()
    if err != nil {
        return nil, nil, -1, err
    }
    defer session.Close()

    // 创建管道
    stdoutPipe, _ := session.StdoutPipe()
    stderrPipe, _ := session.StderrPipe()

    // 启动命令
    if err := session.Start(cmd); err != nil {
        return nil, nil, -1, err
    }

    // 等待完成或超时
    done := make(chan error, 1)
    go func() {
        done <- session.Wait()
    }()

    select {
    case <-ctx.Done():
        session.Signal(ssh.SIGKILL)
        return nil, nil, -1, NewTimeoutError(conn.Host.Name, cmd, timeout)
    case err := <-done:
        stdout, _ = io.ReadAll(stdoutPipe)
        stderr, _ = io.ReadAll(stderrPipe)

        if err != nil {
            if exitErr, ok := err.(*ssh.ExitError); ok {
                return stdout, stderr, exitErr.ExitStatus(), nil
            }
            return stdout, stderr, -1, err
        }
        return stdout, stderr, 0, nil
    }
}
```

## 3. 错误报告

### 3.1 终端输出格式

**成功：**
```
ok: [web01] => {"changed": false, "ping": "pong"}
```

**失败：**
```
failed: [web01] => {"msg": "Failed to connect to host", "unreachable": true}
```

**忽略的错误：**
```
failed: [web01] (ignored) => {"msg": "Command exited with code 1", "rc": 1}
```

### 3.2 Play Recap 格式

```
PLAY RECAP *********************************************************************
web01                      : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
web02                      : ok=1    changed=0    unreachable=1    failed=0    skipped=0    rescued=0    ignored=0
```

### 3.3 详细错误日志

用户可以通过 `-v` 参数增加详细程度：

- `-v`: 显示任务执行细节
- `-vv`: 显示模块参数和返回值
- `-vvv`: 显示 SSH 连接详情
- `-vvvv`: 显示完整的调试信息

```go
type Logger struct {
    level int
}

func (l *Logger) Debug(format string, args ...interface{}) {
    if l.level >= 4 {
        log.Printf("[DEBUG] "+format, args...)
    }
}

func (l *Logger) Info(format string, args ...interface{}) {
    if l.level >= 1 {
        log.Printf("[INFO] "+format, args...)
    }
}

func (l *Logger) Error(format string, args ...interface{}) {
    log.Printf("[ERROR] "+format, args...)
}
```

## 4. 错误恢复机制

### 4.1 重试逻辑（未来支持）

```go
type RetryConfig struct {
    MaxRetries int
    Delay      time.Duration
    Backoff    float64 // 指数退避因子
}

func (r *TaskExecutor) ExecuteWithRetry(task *Task, host *Host, config RetryConfig) (*TaskResult, error) {
    var lastErr error

    for attempt := 0; attempt <= config.MaxRetries; attempt++ {
        if attempt > 0 {
            delay := time.Duration(float64(config.Delay) * math.Pow(config.Backoff, float64(attempt-1)))
            time.Sleep(delay)
            log.Printf("Retry %d/%d for task '%s' on host '%s'", attempt, config.MaxRetries, task.Name, host.Name)
        }

        result, err := r.Execute(task, host)
        if err == nil && !result.Failed {
            return result, nil
        }

        lastErr = err

        // 检查是否可重试
        var execErr *ExecutionError
        if errors.As(err, &execErr) && !execErr.Retriable {
            break
        }
    }

    return nil, lastErr
}
```

### 4.2 优雅降级

当某些功能不可用时，提供降级方案：

- **Inventory 变量解析失败**: 使用默认值继续
- **模板渲染失败**: 使用原始字符串
- **连接失败**: 跳过该主机，继续其他主机

## 5. 用户友好的错误提示

### 5.1 常见错误及解决方案

```go
var errorSuggestions = map[ErrorType]string{
    ErrUnreachable: `
Possible solutions:
1. Check if the host is reachable: ping <host>
2. Verify SSH service is running: systemctl status sshd
3. Check SSH port and firewall rules
4. Verify authentication credentials (key or password)
`,
    ErrModuleNotFound: `
Possible solutions:
1. Check if the module name is correct
2. Verify module path in configuration
3. Install required Ansible collections
`,
    ErrParse: `
Possible solutions:
1. Validate YAML syntax: yamllint <file>
2. Check for proper indentation
3. Ensure all quotes and brackets are balanced
`,
}

func (e *ExecutionError) Suggestion() string {
    if suggestion, ok := errorSuggestions[e.Type]; ok {
        return suggestion
    }
    return ""
}
```

### 5.2 错误输出示例

```
ERROR: Failed to parse inventory file

  File: /path/to/inventory.ini
  Line: 15
  Content: [webservers:vars
           ^
  Error: Unclosed section header

Possible solutions:
1. Ensure all section headers have closing brackets: [webservers:vars]
2. Check for typos in section names
3. Validate INI syntax
```

## 6. 验证

### 6.1 错误处理测试用例

- **连接失败测试**: 连接不存在的主机
- **超时测试**: 执行长时间运行的命令
- **解析错误测试**: 使用格式错误的 inventory 文件
- **模块失败测试**: 执行返回失败的命令
- **部分失败测试**: 多主机环境中部分主机失败

### 6.2 验收标准

- ✅ 所有错误类型都有明确的错误消息
- ✅ 错误不会导致整个程序崩溃（除非是致命错误）
- ✅ 错误日志包含足够的上下文信息用于调试
- ✅ 用户能够理解错误原因并知道如何修复
