# 阶段 1 详细设计：核心连接与 Ad-hoc 执行

## 目标
建立 `ansigo` 的基础架构，实现读取主机清单，通过 SSH 连接远程主机，并执行简单的命令或模块。

## 1. 架构组件

### 1.1 Inventory Manager (清单管理)
*   **功能**: 解析主机清单文件，提供主机查找和分组功能。
*   **输入**: 
    *   **INI 格式** (MVP 优先): 支持 `[group]`, `[group:vars]`, `[group:children]`, 行内变量 `key=value`。
    *   **YAML 格式**: 支持 `hosts:`, `vars:`, `children:` 结构。
*   **核心概念**:
    *   **默认组**: 自动创建 `all` (包含所有主机) 和 `ungrouped` (未显式分组的主机)。
    *   **嵌套组**: 支持组的继承关系 (Parent/Child)。
    *   **行为参数**: 识别标准连接参数如 `ansible_host`, `ansible_port`, `ansible_user`, `ansible_ssh_private_key_file`。
*   **数据结构**:
    ```go
    type Host struct {
        Name string                 // Inventory hostname (alias)
        Vars map[string]interface{} // 包含 ansible_host, ansible_port 等
        Groups []string             // 所属组名
    }
    type Group struct {
        Name     string
        Hosts    []string           // 主机名列表
        Children []string           // 子组名列表
        Vars     map[string]interface{}
        Parent   []string           // 父组名列表 (用于计算变量优先级)
    }
    type Inventory struct {
        Groups map[string]*Group
        Hosts  map[string]*Host
    }
    ```
*   **变量优先级 (Precedence)**:
    1.  Host variables (最高)
    2.  Child Group variables
    3.  Parent Group variables
    4.  `all` Group variables (最低)
*   **实现细节**:
    *   解析器需处理范围语法 (e.g., `web[01:50].example.com`) - *Phase 1 可选*。
    *   **加载顺序**: 
        1. 读取文件。
        2. 解析主机和组。
        3. 解析 `:children` 建立层级。
        4. 解析 `:vars` 和行内变量。
        5. 扁平化变量到 Host 层面 (计算最终生效的连接参数)。

### 1.2 Connection Manager (连接管理)
*   **功能**: 管理 SSH 连接生命周期。
*   **库**: `golang.org/x/crypto/ssh`
*   **接口**:
    ```go
    type Connection interface {
        Connect(host *Host) error
        Exec(cmd string) (stdout, stderr []byte, err error)
        PutFile(src, dst string) error
        Close()
    }
    ```
*   **实现细节**:
    *   支持 SSH Key (默认 `~/.ssh/id_rsa`) 和 密码认证。
    *   实现 `PutFile` (用于后续传输模块)，可以使用 SFTP (`github.com/pkg/sftp`) 或 `scp` 命令封装。
    *   连接复用（可选，MVP 阶段可每次新建连接）。

### 1.3 Ad-hoc Runner (执行器)
*   **功能**: 串联 Inventory 和 Connection，针对指定主机执行动作。
*   **CLI**: `ansigo -i hosts -m <module> -a <args> <pattern>`
*   **流程**:
    1.  解析 CLI 参数。
    2.  加载 Inventory。
    3.  根据 `<pattern>` (如 `all`, `webservers`) 筛选目标主机列表。
    4.  并发循环主机列表：
        a.  建立连接。
        b.  执行指令。
        c.  收集结果。
    5.  输出结果。

## 2. 模块实现 (MVP)

### 2.1 Raw 模块
*   直接在远程 Shell 执行命令。
*   相当于 `ssh user@host "cmd"`.

### 2.2 Ping 模块 (模拟)
*   **目标**: 验证 `ansigo` 具备模块执行的基本形态（输入->执行->JSON输出）。
*   **实现**:
    *   不依赖 Python。
    *   直接执行远程命令 `echo '{"ping": "pong"}'`。
    *   解析返回的 JSON，如果包含 `ping: pong` 则视为成功。

## 3. 验证计划
1.  **本地测试**: 启动 Docker 容器运行 `sshd`。
2.  **Inventory 测试**: 解析包含多个组和主机的 `hosts.ini`。
3.  **连接测试**: `ansigo -i hosts -m raw -a "hostname" all` 应返回容器主机名。
4.  **Ping 测试**: `ansigo -i hosts -m ping all` 应返回绿色成功信息。
