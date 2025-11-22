# 阶段 2 详细设计：模块执行引擎

## 目标
实现兼容 Ansible 协议的模块执行引擎，能够分发并执行标准 Ansible 模块（Python）或自定义模块，并处理参数和返回值。

## 1. 模块工作流分析
根据 Ansible 文档，模块执行的核心流程如下：
1.  **参数处理**: 将用户参数转换为 JSON 格式。
2.  **模块包装**:
    *   **Ansiballz (Python)**: 将模块及其依赖 (`ansible.module_utils`) 打包成 Zip，Base64 编码，嵌入 Python 包装脚本。
    *   **JSONARGS/WANT_JSON**: 简单的参数注入或文件传递。
3.  **传输**: 将生成的 payload 传输到远程临时目录 (`~/.ansible/tmp/...`)。
4.  **执行**: 调用远程解释器（如 `/usr/bin/python`）执行 payload。
5.  **输出**: 读取 stdout 中的 JSON 结果。
6.  **清理**: 删除临时文件。

## 2. Ansigo 实现策略

### 2.1 模块发现 (Module Finder)
*   在本地搜索模块文件。
*   路径: `~/.ansible/plugins/modules`, `/usr/share/ansible/plugins/modules`, 当前目录 `library/`。

### 2.2 模块执行器 (Module Executor)

#### 策略 A: 兼容现有 Python 模块 (Ansiballz) - *高级目标*
*   **难点**: 需要在 Go 中重新实现 `Ansiballz` 的打包逻辑，或者调用本地安装的 `ansible` 库来生成 payload。
*   **方案**: 为了完全兼容，我们需要分析 `ansible-core` 的源码，复刻其打包逻辑。这比较复杂。
*   **替代方案**: 调用本地 `ansible` 命令生成 payload (如果可行)，或者仅支持简单的单文件模块。

#### 策略 B: 独立模块协议 (WANT_JSON) - *MVP 首选*
*   **原理**: 许多非 Python 模块（Binary, Bash 等）使用此协议。
*   **流程**:
    1.  生成参数文件 `args.json`。
    2.  将模块文件 `module` 和 `args.json` 传输到远程。
    3.  执行 `module args.json`。
    4.  模块读取文件，执行，输出 JSON。
*   **兼容性**: 我们可以编写一个通用的 Go 包装器 (Wrapper)，用于执行标准 Python 模块。
    *   Go 包装器负责：设置 `PYTHONPATH`，将参数通过 stdin 或文件传递给 Python 脚本。

### 2.3 核心实现步骤 (MVP)

我们将采用 **"参数文件 + 脚本执行"** 的通用模式，这兼容 `WANT_JSON` 和大多数简单脚本。

1.  **准备阶段**:
    *   生成 UUID 作为任务 ID。
    *   创建远程目录: `mkdir -p ~/.ansible/tmp/ansigo-<uuid>`。

2.  **参数传递**:
    *   将参数序列化为 JSON。
    *   写入临时文件 `args`。

3.  **传输阶段**:
    *   上传模块脚本 (例如 `command.py`) 到远程目录。
    *   上传 `args` 文件到远程目录。

4.  **执行阶段**:
    *   构建执行命令。
    *   对于 Python 模块: `python command.py args` (需要模块支持读取文件参数)。
    *   *注意*: 标准 Ansible Python 模块通常期望 `Ansiballz` 注入。为了在 MVP 中运行它们，我们可能需要一个简化的 Python 包装器 `wrapper.py`，它负责：
        1.  读取 `args` 文件。
        2.  设置 `ansible.module_utils` 路径 (如果远程有) 或者我们需要将 `module_utils` 也上传。
    *   **简化决策**: 阶段 2 初期，我们先支持 **"Binary Modules"** 规范。即：我们自己写简单的 Go/Python 模块，接受一个文件路径参数。
    *   **进阶**: 尝试支持标准 `command` / `shell` 模块。这两个模块依赖较少，较容易移植。

5.  **结果解析**:
    *   捕获 stdout。
    *   寻找 JSON 字符串（Ansible 模块输出可能包含非 JSON 的调试信息，通常需要寻找 `{...}` 块）。
    *   解析为 `ModuleResult` 结构体。

## 3. 数据结构

```go
type ModuleArgs map[string]interface{}

type ModuleResult struct {
    Changed bool            `json:"changed"`
    Msg     string          `json:"msg,omitempty"`
    Failed  bool            `json:"failed,omitempty"`
    RC      int             `json:"rc,omitempty"`
    Stdout  string          `json:"stdout,omitempty"`
    Stderr  string          `json:"stderr,omitempty"`
    // 其他动态字段
    Data    map[string]interface{} `json:"-"` 
}
```

## 4. 验证
*   编写一个符合 `WANT_JSON` 规范的 Shell 脚本模块。
*   使用 `ansigo -m my_shell_mod -a "foo=bar"` 运行。
*   验证参数是否正确传递，结果是否正确解析。
