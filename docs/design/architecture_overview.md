# Ansigo 架构总览

## 1. 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Layer                            │
│  (ansigo / ansigo-playbook)                                 │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                    Executor Layer                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Ad-hoc      │  │  Playbook    │  │  Task        │      │
│  │  Runner      │  │  Runner      │  │  Executor    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                    Core Components                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Inventory   │  │  Module      │  │  Variable    │      │
│  │  Manager     │  │  Executor    │  │  Manager     │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │  Connection  │  │  Template    │                        │
│  │  Manager     │  │  Engine      │                        │
│  └──────────────┘  └──────────────┘                        │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                  Transport Layer                             │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │  SSH Client  │  │  SFTP Client │                        │
│  └──────────────┘  └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

## 2. 核心组件说明

### 2.1 Inventory Manager
**职责**: 解析和管理主机清单

**关键功能**:
- 解析 INI/YAML 格式的 inventory 文件
- 维护主机和组的层级关系
- 计算变量优先级并扁平化到主机
- 提供主机查询接口（按组、按模式）

**接口**:
```go
type InventoryManager interface {
    Load(path string) error
    GetHost(name string) (*Host, error)
    GetHosts(pattern string) ([]*Host, error)
    GetGroup(name string) (*Group, error)
}
```

### 2.2 Connection Manager
**职责**: 管理到远程主机的连接

**关键功能**:
- 建立 SSH 连接（支持密钥和密码认证）
- 执行远程命令
- 文件传输（SFTP）
- 连接池管理（可选优化）

**接口**:
```go
type Connection interface {
    Connect(host *Host) error
    Exec(cmd string) (stdout, stderr []byte, exitCode int, err error)
    PutFile(localPath, remotePath string) error
    GetFile(remotePath, localPath string) error
    Close() error
}
```

### 2.3 Module Executor
**职责**: 执行 Ansible 模块

**关键功能**:
- 模块发现（本地路径搜索）
- 参数序列化（JSON）
- 模块传输和执行
- 结果解析

**执行流程**:
```
1. 准备参数 → JSON
2. 创建远程临时目录
3. 传输模块文件
4. 传输参数文件
5. 执行模块
6. 捕获输出
7. 解析 JSON 结果
8. 清理临时文件
```

### 2.4 Variable Manager
**职责**: 管理变量作用域和优先级

**变量来源**（优先级从低到高）:
1. `all` 组变量
2. 父组变量
3. 子组变量
4. 主机变量
5. Playbook 变量
6. `register` 变量
7. `set_fact` 变量

**接口**:
```go
type VariableManager interface {
    SetHostVar(host, key string, value interface{})
    GetHostVars(host string) map[string]interface{}
    MergeVars(sources ...map[string]interface{}) map[string]interface{}
}
```

### 2.5 Template Engine
**职责**: 处理 Jinja2 模板语法

**实现方案**:
- **Phase 1-2**: 不支持模板
- **Phase 3**: 集成 `pongo2` 库（Go 的 Jinja2 实现）

**功能范围**:
- 变量替换 `{{ var }}`
- 简单过滤器 `{{ var | default('value') }}`
- 条件表达式（用于 `when`）

## 3. 数据流

### 3.1 Ad-hoc 命令执行流程
```
用户输入: ansigo -i hosts -m ping all
    ↓
1. CLI 解析参数
    ↓
2. Inventory Manager 加载 hosts
    ↓
3. 根据 pattern "all" 筛选主机
    ↓
4. 并发循环每个主机:
    ├─ Connection Manager 建立连接
    ├─ Module Executor 执行 ping
    └─ 收集结果
    ↓
5. 输出结果汇总
```

### 3.2 Playbook 执行流程
```
用户输入: ansigo-playbook site.yml
    ↓
1. 解析 YAML → Playbook 对象
    ↓
2. 加载 Inventory
    ↓
3. 遍历 Play:
    ├─ 根据 hosts 筛选目标
    ├─ 初始化 HostVars
    └─ 遍历 Task:
        ├─ 模板渲染参数
        ├─ 并发执行（所有主机）
        ├─ 收集结果
        └─ 更新 HostVars (register)
    ↓
4. 输出 Play Recap
```

## 4. 目录结构

```
ansigo/
├── cmd/
│   ├── ansigo/           # Ad-hoc 命令入口
│   └── ansigo-playbook/  # Playbook 命令入口
├── pkg/
│   ├── inventory/        # Inventory 解析和管理
│   │   ├── parser.go     # INI/YAML 解析器
│   │   ├── host.go       # Host 数据结构
│   │   └── group.go      # Group 数据结构
│   ├── connection/       # SSH 连接管理
│   │   ├── ssh.go        # SSH 客户端
│   │   └── sftp.go       # SFTP 客户端
│   ├── module/           # 模块执行
│   │   ├── executor.go   # 执行引擎
│   │   ├── finder.go     # 模块查找
│   │   └── result.go     # 结果解析
│   ├── playbook/         # Playbook 解析
│   │   ├── parser.go     # YAML 解析
│   │   └── types.go      # Play/Task 数据结构
│   ├── runner/           # 执行编排
│   │   ├── adhoc.go      # Ad-hoc 执行器
│   │   └── playbook.go   # Playbook 执行器
│   ├── vars/             # 变量管理
│   │   └── manager.go
│   └── template/         # 模板引擎
│       └── engine.go
├── library/              # 内置模块（可选）
│   ├── ping
│   └── command
└── tests/
    ├── integration/      # 集成测试
    └── fixtures/         # 测试数据
```

## 5. 关键设计决策

### 5.1 并发模型
- **Ad-hoc**: 使用 Goroutine 并发执行，WaitGroup 等待
- **Playbook**: 线性策略，Task 级别并发

### 5.2 错误处理
- 连接失败: 标记主机为 UNREACHABLE，继续其他主机
- 模块失败: 根据 `ignore_errors` 决定是否继续
- 致命错误: 立即中止（如 Playbook 解析失败）

### 5.3 兼容性边界
**完全兼容**:
- Inventory 格式（INI/YAML）
- 核心模块协议（WANT_JSON）
- 基础 Playbook 语法

**部分兼容**:
- Jinja2 模板（仅支持常用语法）
- 变量优先级（简化版）

**不兼容**:
- Roles 和 Collections
- 复杂的插件系统
- Ansible Vault（可后续支持）
