# Jinja2 模板兼容性说明

AnsiGo 使用 [go-jinja2](https://github.com/kluctl/kluctl/tree/main/lib/go-jinja2) 提供 **100% 兼容**的 Jinja2 模板引擎。go-jinja2 通过嵌入完整的 Python 解释器，实现了真正的 Jinja2 支持。

## ✅ 完整支持的功能

### 基本模板语法

- ✅ **变量替换**: `{{ variable }}`
- ✅ **过滤器**: `{{ variable | upper }}`
- ✅ **控制结构**: `{% if %} {% for %} {% block %}`
- ✅ **注释**: `{# comment #}`

### 字符串连接操作符 (~)

完全支持 Jinja2 的波浪号字符串连接操作符，包括与过滤器的组合使用。

**示例**:

```jinja2
{{ "hello" ~ " " ~ "world" }}
# 输出: hello world

{{ app_name ~ '-' ~ app_version }}
# 如果 app_name="myapp", app_version="1.2.3"
# 输出: myapp-1.2.3

# ✅ 支持与过滤器组合（pongo2 不支持此功能）
{{ (app_name ~ '-' ~ app_version) | upper }}
# 输出: MYAPP-1.2.3
```

### 内联条件表达式 (三元运算符)

完全支持 Jinja2 的内联条件表达式语法。

**示例**:

```jinja2
{{ 'enabled' if debug else 'disabled' }}
# 如果 debug=true, 输出: enabled
# 如果 debug=false, 输出: disabled

{{ 'production' if env == 'prod' else 'development' }}

# ✅ 支持与过滤器组合
{{ ('yes' if enabled else 'no') | upper }}
# 输出: YES 或 NO
```

### 标准过滤器

支持所有 Jinja2 标准过滤器：

- **字符串**: `upper`, `lower`, `trim`, `length`, `default`, `join`, `replace`
- **列表**: `first`, `last`, `length`, `join`, `sort`, `reverse`
- **数字**: `abs`, `round`, `int`, `float`
- **格式化**: `to_json`, `to_yaml`, `b64encode`, `b64decode`
- **其他**: `default`, `escape`, `safe`, `regex_replace`, `regex_search`

### 高级功能

- ✅ **宏 (Macros)**: 完整支持
- ✅ **模板继承**: `extends` 和 `block`
- ✅ **包含**: `include`
- ✅ **自定义过滤器**: 可通过 Python 代码添加
- ✅ **复杂的嵌套循环和条件**: 完整支持

## 部署要求

### 二进制大小

使用 go-jinja2 会增加约 **30-35MB** 的二进制大小，因为嵌入了完整的 Python 解释器。

### 平台支持

**✅ 支持的平台**:

- Linux (glibc) - Debian, Ubuntu, RHEL, CentOS, Fedora 等
- macOS (Intel 和 Apple Silicon)
- Windows

**❌ 不支持的平台**:

- **Alpine Linux (musl libc)** - go-embed-python 使用的 Python 发行版是为 glibc 编译的

### 编译要求

**重要**: 必须在目标平台上编译，交叉编译不支持。

```bash
# ✅ 正确：在 Linux 上为 Linux 编译
go build -o ansigo ./cmd/ansigo-playbook

# ✅ 正确：在 Docker 中为 Linux 编译
docker run --rm -v $(pwd):/work -w /work golang:1.25 go build -o ansigo ./cmd/ansigo-playbook

# ❌ 错误：从 macOS 交叉编译到 Linux
GOOS=linux GOARCH=amd64 go build -o ansigo ./cmd/ansigo-playbook
```

## 性能考虑

- **启动时间**: 首次初始化需要启动 Python 进程（约 1-2 秒）
- **渲染性能**: 与原生 Jinja2 性能相当
- **内存占用**: 增加约 50-100MB（Python 解释器）

## 与 Ansible 的兼容性

### ✅ 完全兼容

所有 Ansible 使用的 Jinja2 语法都完全支持，无需任何修改或变通方案。

### 不支持的 Ansible 特定功能

以下是 Ansible 特有的功能，不是 Jinja2 的一部分：

1. **Lookup 插件**: 如 `{{ lookup('file', '/path/to/file') }}`
2. **Ansible 变量**: 如 `hostvars`, `groups`, `inventory_hostname`
   - 这些需要在 AnsiGo 中单独实现

## 迁移说明

从之前的 pongo2 实现迁移到 go-jinja2 是完全透明的：

- ✅ 所有原有的模板无需修改
- ✅ API 接口保持不变
- ✅ 新增了对复杂语法的支持（波浪号 + 过滤器等）

## 测试

我们的测试套件包括：

- 基本模板渲染测试
- 条件表达式测试
- 循环测试
- **波浪号操作符测试**（包括与过滤器的组合）
- **内联条件表达式测试**（包括与过滤器的组合）

运行测试：

```bash
go test -v ./pkg/playbook -run TestJinja2TemplateEngine
```

## 技术细节

### 实现原理

go-jinja2 使用 [go-embed-python](https://github.com/kluctl/go-embed-python) 嵌入完整的 Python 发行版：

1. **嵌入方式**: 使用 Go 的 `//go:embed` 特性嵌入 Python 二进制
2. **进程模型**: Python 运行在独立进程中，通过 stdin/stdout 通信
3. **无需 CGO**: 不使用 CGO 或动态链接
4. **无需系统 Python**: 目标机器不需要安装 Python

### 资源管理

- 使用 `defer engine.Close()` 确保 Python 进程正确关闭
- Runner 在结束时自动清理资源
- 延迟初始化，仅在需要时启动 Python 进程

## 参考资料

- [Jinja2 官方文档](https://jinja.palletsprojects.com/)
- [Ansible 模板文档](https://docs.ansible.com/ansible/latest/user_guide/playbooks_templating.html)
- [go-jinja2 项目](https://github.com/kluctl/kluctl/tree/main/lib/go-jinja2)
- [go-embed-python 项目](https://github.com/kluctl/go-embed-python)
