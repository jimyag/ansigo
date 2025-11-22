# AnsiGo 集成测试文档

本文档详细说明 AnsiGo 项目的集成测试覆盖情况，包括所有 Ansible 兼容功能的测试。

## 测试概览

### 测试统计

- **测试套件数量**: 13 个
- **测试用例总数**: 200+ 个
- **覆盖的 Ansible 功能**: 核心模块、变量管理、模板引擎、条件执行、循环、错误处理、多主机并发

### 测试文件列表

```
tests/playbooks/
├── test-jinja2-working.yml         # Jinja2 模板引擎基础功能 (20 个测试)
├── test-jinja2-filters.yml         # Jinja2 过滤器 (16 个测试)
├── test-jinja2-loops.yml           # Jinja2 循环结构 (6 个测试)
├── test-jinja2-advanced.yml        # Jinja2 高级特性 (15 个测试)
├── test-modules.yml                # 核心模块执行 (10 个测试) ✨ 新增
├── test-variable-precedence.yml    # 变量优先级 (10 个测试) ✨ 新增
├── test-loops-iteration.yml        # 循环和迭代 (11 个测试) ✨ 新增
├── test-error-handling.yml         # 错误处理 (10 个测试) ✨ 新增
├── test-when-conditions.yml        # 条件执行 (20 个测试) ✨ 新增
├── test-multi-host.yml             # 多主机并发 (11 个测试) ✨ 新增
├── test-module-args.yml            # 模块参数和返回值 (15 个测试) ✨ 新增
├── test-basic.yml                  # 基础功能
└── test-conditionals.yml           # 条件判断基础
```

## 详细测试说明

### Test Suite 1: Jinja2 模板引擎基础功能

**文件**: [test-jinja2-working.yml](playbooks/test-jinja2-working.yml)

**测试用例** (20 个):
1. ✅ 基本变量替换 - `{{ variable }}`
2. ✅ 字符串过滤器 - `upper`, `lower`
3. ✅ 嵌套变量访问 - `{{ config.host }}:{{ config.port }}`
4. ✅ 条件表达式 - `{% if %}...{% else %}...{% endif %}`
5. ✅ 数组访问 - `first`, `last`
6. ✅ 数组长度 - `length`
7. ✅ 数组连接 - `join`
8. ✅ when 条件 - 等于判断
9. ✅ when 条件 - 不等于判断
10. ✅ when 条件 - 布尔值判断
11. ✅ when 条件 - 数字比较
12. ✅ 命令执行和结果注册
13. ✅ 使用注册变量
14. ✅ 字符串长度过滤器
15. ✅ 复杂条件组合
16. ✅ for 循环 - 基本迭代
17. ✅ for 循环 - 带索引
18. ✅ for 循环 - 条件过滤
19. ✅ title 过滤器
20. ✅ 所有测试通过

**验证的 Ansible 功能**:
- Jinja2 变量插值
- 内置过滤器
- 控制结构 (if/for)
- 变量注册和引用

---

### Test Suite 7: 核心模块执行 ✨ 新增

**文件**: [test-modules.yml](playbooks/test-modules.yml)

**测试用例** (10 个):
1. ✅ **ping 模块** - 测试主机连通性
2. ✅ **command 模块** - 简单命令执行 (echo)
3. ✅ **command 模块** - creates 参数
4. ✅ **shell 模块** - 管道命令
5. ✅ **copy 模块** - content 参数
6. ✅ **command 模块** - 获取系统信息 (hostname)
7. ✅ **shell 模块** - 环境变量
8. ✅ **command 模块** - 返回码检查
9. ✅ 文件验证 - cat 命令
10. ✅ 清理测试文件

**验证的 Ansible 功能**:
- `ping` - 连通性测试
- `command` - 命令执行、参数处理
- `shell` - Shell 命令、管道支持
- `copy` - 文件复制、content 参数
- `register` - 结果注册
- 模块返回值: `stdout`, `stderr`, `rc`

---

### Test Suite 8: 变量优先级和作用域 ✨ 新增

**文件**: [test-variable-precedence.yml](playbooks/test-variable-precedence.yml)

**测试用例** (10 个):
1. ✅ Play 级别变量
2. ✅ Inventory 主机名变量
3. ✅ 注册变量 (register)
4. ✅ 变量优先级测试 - registered > play
5. ✅ 嵌套变量访问
6. ✅ 变量在命令参数中的使用
7. ✅ 多个注册变量
8. ✅ 变量在 copy 模块中的展开
9. ✅ 未定义变量的 default 过滤器
10. ✅ 变量作用域验证

**验证的 Ansible 功能**:
- 变量优先级: registered > play > inventory
- 变量作用域
- `register` 关键字
- `default` 过滤器
- 变量在模板中的展开

**变量优先级顺序** (从高到低):
1. Registered variables (注册变量)
2. Play variables (Play 级变量)
3. Inventory variables (Inventory 变量)

---

### Test Suite 9: 循环和迭代 ✨ 新增

**文件**: [test-loops-iteration.yml](playbooks/test-loops-iteration.yml)

**测试用例** (11 个):
1. ✅ 简单列表循环 - `loop: {{ list }}`
2. ✅ 字典项循环 - `item.key`, `item.value`
3. ✅ 循环中执行命令
4. ✅ 循环结果访问 - `loop_results.results`
5. ✅ 循环 + when 条件
6. ✅ 循环创建文件
7. ✅ 循环验证文件
8. ✅ 嵌套数据循环
9. ✅ 循环 + copy 模块
10. ✅ 循环 + 变量插值
11. ✅ 循环清理

**验证的 Ansible 功能**:
- `loop` 关键字
- `item` 变量
- 循环结果 `.results` 访问
- 循环 + 条件组合
- 循环 + 模块参数
- 循环中的变量展开

---

### Test Suite 10: 错误处理 ✨ 新增

**文件**: [test-error-handling.yml](playbooks/test-error-handling.yml)

**测试用例** (10 个):
1. ✅ 正常任务执行
2. ✅ `ignore_errors: yes` - 命令失败继续
3. ✅ `ignore_errors: yes` - 不存在的命令
4. ✅ 多个忽略错误的任务
5. ✅ 读取不存在文件 (忽略)
6. ✅ 错误后创建文件
7. ✅ 多主机错误处理
8. ✅ 变量 + 错误忽略
9. ✅ 任务链中的错误
10. ✅ 清理测试文件

**验证的 Ansible 功能**:
- `ignore_errors: yes` - 错误忽略
- 错误后继续执行
- 多主机环境下的错误处理
- 错误返回码 `rc`
- Playbook 执行流程控制

---

### Test Suite 11: 高级条件执行 ✨ 新增

**文件**: [test-when-conditions.yml](playbooks/test-when-conditions.yml)

**测试用例** (20 个):
1. ✅ 简单等于判断 - `when: var == 'value'`
2. ✅ 不等于判断 - `when: var != 'value'`
3. ✅ 大于判断 - `when: num > 1024`
4. ✅ 小于判断 - `when: num < 65535`
5. ✅ 布尔判断 - `when: not debug_mode`
6. ✅ AND 条件 - `when: a and b`
7. ✅ OR 条件 - `when: a or b`
8. ✅ 复杂嵌套条件 - `(a or b) and c`
9. ✅ NOT 条件
10. ✅ 嵌套变量条件 - `when: obj.field == value`
11. ✅ 多个 AND 条件
12. ✅ 注册变量 + 条件
13. ✅ 字符串包含判断
14. ✅ 分组条件
15. ✅ 命令输出 + 条件
16. ✅ 大于等于 `>=`
17. ✅ 小于等于 `<=`
18. ✅ when + loop 组合
19. ✅ NOT + AND 组合
20. ✅ 复杂真实场景条件

**验证的 Ansible 功能**:
- `when` 关键字
- 比较运算符: `==`, `!=`, `>`, `<`, `>=`, `<=`
- 逻辑运算符: `and`, `or`, `not`
- 条件优先级和括号
- 条件 + 循环组合
- 嵌套对象属性访问

---

### Test Suite 12: 多主机并发执行 ✨ 新增

**文件**: [test-multi-host.yml](playbooks/test-multi-host.yml)

**测试用例** (11 个):
1. ✅ 所有主机执行任务
2. ✅ 获取每个主机的 hostname
3. ✅ 创建主机特定文件
4. ✅ 共享变量测试
5. ✅ 每个主机注册变量
6. ✅ 并发命令执行
7. ✅ 主机特定条件
8. ✅ 每个主机不同内容
9. ✅ 循环 + 主机特定数据
10. ✅ 执行顺序独立性
11. ✅ 清理

**验证的 Ansible 功能**:
- `hosts: all` - 多主机目标
- `inventory_hostname` - 主机名变量
- 并发执行
- 主机特定变量
- 主机隔离 (每个主机独立的注册变量)
- 多主机 + 循环组合

---

### Test Suite 13: 模块参数和返回值 ✨ 新增

**文件**: [test-module-args.yml](playbooks/test-module-args.yml)

**测试用例** (15 个):
1. ✅ command 返回值 - `stdout`, `rc`
2. ✅ shell stderr 捕获
3. ✅ 多参数命令
4. ✅ copy 模块参数
5. ✅ shell 管道命令
6. ✅ 工作目录测试
7. ✅ 访问多个返回字段
8. ✅ changed 状态测试
9. ✅ 环境变量
10. ✅ creates 参数
11. ✅ 长输出处理
12. ✅ 特殊字符
13. ✅ 空输出
14. ✅ 多行输出
15. ✅ 清理

**验证的 Ansible 功能**:
- 模块返回值结构:
  - `stdout` - 标准输出
  - `stderr` - 标准错误
  - `rc` - 返回码
  - `changed` - 变更状态
- 模块参数:
  - `content` - copy 模块
  - `dest` - copy 模块
  - `creates` - command 模块
  - `args` - 额外参数
- 返回值在模板中的使用

---

## 运行测试

### 运行所有集成测试

```bash
./tests/scripts/run-integration-tests.sh
```

### 运行特定测试套件

```bash
# 运行模块测试
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-modules.yml

# 运行变量测试
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-variable-precedence.yml

# 运行循环测试
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-loops-iteration.yml
```

### 运行单元测试 + 集成测试

```bash
./tests/scripts/run-all-tests.sh
```

## Ansible 功能覆盖矩阵

| 功能分类 | 功能点 | 测试覆盖 | 测试文件 |
|---------|--------|---------|---------|
| **核心模块** | ping | ✅ | test-modules.yml |
| | command | ✅ | test-modules.yml, test-module-args.yml |
| | shell | ✅ | test-modules.yml, test-module-args.yml |
| | copy | ✅ | test-modules.yml, test-module-args.yml |
| | debug | ✅ | 所有测试文件 |
| **变量系统** | play vars | ✅ | test-variable-precedence.yml |
| | inventory vars | ✅ | test-variable-precedence.yml |
| | registered vars | ✅ | test-variable-precedence.yml |
| | 变量优先级 | ✅ | test-variable-precedence.yml |
| | 嵌套变量 | ✅ | test-variable-precedence.yml |
| **模板引擎** | 变量插值 | ✅ | test-jinja2-*.yml |
| | 过滤器 | ✅ | test-jinja2-filters.yml |
| | 控制结构 | ✅ | test-jinja2-working.yml |
| | 条件表达式 | ✅ | test-jinja2-advanced.yml |
| **循环** | loop 关键字 | ✅ | test-loops-iteration.yml |
| | item 变量 | ✅ | test-loops-iteration.yml |
| | 循环结果 | ✅ | test-loops-iteration.yml |
| | loop + when | ✅ | test-loops-iteration.yml |
| **条件执行** | when 关键字 | ✅ | test-when-conditions.yml |
| | 比较运算符 | ✅ | test-when-conditions.yml |
| | 逻辑运算符 | ✅ | test-when-conditions.yml |
| | 复杂条件 | ✅ | test-when-conditions.yml |
| **错误处理** | ignore_errors | ✅ | test-error-handling.yml |
| | 错误后继续 | ✅ | test-error-handling.yml |
| | 返回码检查 | ✅ | test-module-args.yml |
| **多主机** | 并发执行 | ✅ | test-multi-host.yml |
| | 主机变量 | ✅ | test-multi-host.yml |
| | inventory_hostname | ✅ | test-multi-host.yml |
| **模块返回** | stdout/stderr | ✅ | test-module-args.yml |
| | rc | ✅ | test-module-args.yml |
| | changed | ✅ | test-module-args.yml |

## 测试环境

### 本地测试
- Go 版本: 1.21+
- 操作系统: macOS, Linux

### Docker 测试环境
```bash
./tests/scripts/setup-test-env.sh
```

创建的容器:
- `ansigo-control`: 控制节点
- `ansigo-target-1`: 目标节点 1
- `ansigo-target-2`: 目标节点 2

## 测试结果示例

```
==================================================
AnsiGo Integration Test Suite
==================================================

Test Suite 7: Core Modules (ping, command, shell, copy)
--------------------------------------------------------
Running: Core Modules... ✓ PASS

Test Suite 8: Variable Precedence and Scoping
----------------------------------------------
Running: Variable Precedence... ✓ PASS

Test Suite 9: Loops and Iteration
----------------------------------
Running: Loops and Iteration... ✓ PASS

Test Suite 10: Error Handling (ignore_errors)
----------------------------------------------
Running: Error Handling... ✓ PASS

Test Suite 11: Advanced When Conditions
----------------------------------------
Running: When Conditions... ✓ PASS

Test Suite 12: Multi-Host Concurrent Execution
-----------------------------------------------
Running: Multi-Host... ✓ PASS

Test Suite 13: Module Arguments and Return Values
--------------------------------------------------
Running: Module Args... ✓ PASS

==================================================
Test Summary
==================================================
Total Tests:  13
Passed:       13
Failed:       0

All tests passed!
```

## 待实现功能

以下 Ansible 功能尚未实现测试:

- [ ] handlers 和 notify
- [ ] facts gathering
- [ ] set_fact 模块
- [ ] include 和 import
- [ ] roles
- [ ] tags
- [ ] 更多内置模块 (file, service, package 等)
- [ ] vault 加密
- [ ] 动态 inventory

## 贡献指南

添加新的集成测试:

1. 在 `tests/playbooks/` 创建新的 YAML 文件
2. 遵循现有测试的命名规范: `test-<feature>.yml`
3. 在文件顶部添加测试套件说明注释
4. 每个测试任务使用描述性的 `name`
5. 在 `tests/scripts/run-integration-tests.sh` 中添加测试
6. 更新本文档的测试覆盖矩阵

## 参考资料

- [Ansible Documentation](https://docs.ansible.com/)
- [Jinja2 Template Designer Documentation](https://jinja.palletsprojects.com/)
- [AnsiGo Design Documents](../docs/)
