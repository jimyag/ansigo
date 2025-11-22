# AnsiGo 功能路线图

本文档记录了 AnsiGo 项目的功能实现计划，基于 Ansible 官方文档分析和实际使用频率排序。

生成时间: 2025-11-22
参考文档: `docs/ansible-documentation/docs/docsite/rst/playbook_guide/`

---

## ✅ 已完成功能

### Phase 1: 基础连接 (已完成)
- ✅ SSH 连接管理
- ✅ Inventory 解析（INI 格式）
- ✅ 主机变量和组变量

### Phase 2: 模块执行 (已完成)
- ✅ ping 模块
- ✅ command 模块
- ✅ shell 模块
- ✅ raw 模块
- ✅ copy 模块（基本功能）
- ✅ debug 模块（含 var 参数）
- ✅ set_fact 模块

### Phase 3: Playbook 基础 (已完成)
- ✅ YAML playbook 解析
- ✅ Play 和 Task 执行
- ✅ 变量系统（Play vars, Inventory vars, Register vars）
- ✅ Jinja2 模板引擎（100% 兼容，使用 go-jinja2）
- ✅ 条件执行 (when)
- ✅ 基本循环 (loop)
- ✅ 任务注册 (register)
- ✅ 错误忽略 (ignore_errors)
- ✅ 自定义失败条件 (failed_when)
- ✅ 自定义变更状态 (changed_when)
- ✅ 魔法变量 (inventory_hostname, ansible_host)
- ✅ 并发执行多主机任务
- ✅ Ansible 风格的彩色输出

### Phase 4: Handlers 和 Notify (已完成 - 2025-11-22)

- ✅ notify 关键字（支持字符串和列表）
- ✅ handlers 部分（Play 级别定义）
- ✅ Handler 去重机制（同一 handler 只执行一次）
- ✅ Handler 执行顺序（按定义顺序，非通知顺序）
- ✅ listen 关键字（支持 topic 监听）
- ✅ 只有 changed 任务才触发 notify
- ✅ Handler 支持 when 条件

### Phase 5: 循环功能增强 (已完成 - 2025-11-22)

- ✅ 基本循环 (loop)
- ✅ 静态列表循环
- ✅ 模板变量循环
- ✅ 字典列表循环
- ✅ 注册变量的 results 列表
- ✅ loop_control.loop_var（自定义循环变量名）
- ✅ loop_control.index_var（循环索引）
- ✅ loop_control.pause（迭代间暂停）
- ✅ 嵌套循环支持（loop over register results）
- ✅ 循环中使用 when/failed_when/changed_when
- ✅ 循环输出优化（显示每次迭代）

### Phase 6: Block/Rescue/Always (已完成 - 2025-11-22)

- ✅ block 关键字（任务分组）
- ✅ rescue 关键字（错误恢复）
- ✅ always 关键字（总是执行）
- ✅ ansible_failed_task 变量
- ✅ ansible_failed_result 变量
- ✅ Block 级别 when 条件
- ✅ Block 中 ignore_errors 支持
- ✅ Rescue 失败传播
- ✅ 递归任务执行支持

### Phase 7: 魔法变量 (已完成 - 2025-11-23)

- ✅ hostvars - 所有主机的变量
- ✅ groups - 所有组及其成员
- ✅ group_names - 当前主机所在的组
- ✅ ansible_play_hosts - 当前 play 的主机列表
- ✅ ansible_play_batch - 当前批次的主机
- ✅ inventory_hostname - 当前主机名
- ✅ ansible_host - 主机地址

### Phase 8: 常用模块 (部分完成 - 2025-11-23)

- ✅ file 模块 - 文件和目录管理
- ✅ template 模块 - Jinja2 模板渲染
- ✅ lineinfile 模块 - 行级文件编辑

---

## 📋 待实现功能

根据实际项目使用需求（基于 homelab 项目分析）和 Ansible 官方文档，按优先级排序：

## 🔴 CRITICAL - Homelab 项目依赖功能

以下功能是运行 homelab 项目的必需功能，缺少任何一项都会导致项目无法运行。

**参考**: 基于 `/Users/jimyag/src/github/homelab` 项目实际分析结果

### 使用统计

从 homelab 项目分析得出的功能使用频率：

**模块使用情况**:
- import_tasks: 14 次
- include_role: 12 次
- systemd: 9 次
- template: 6 次 ✅
- get_url: 6 次
- file: 2 次 ✅
- copy: 2 次 ✅
- command: 2 次 ✅
- unarchive: 2 次
- user: 1 次
- fail: 1 次
- debug: 1 次 ✅

**Playbook 特性使用情况**:
- tags: 21 次
- become: 18 次
- vars: 17 次 ✅
- notify: 10 次 ✅
- roles: 6 次
- when: 5 次 ✅
- strategy: 5 次
- loop: 5 次 ✅
- register: 2 次 ✅
- changed_when: 1 次 ✅

---

### 🔴 优先级 0: Roles 系统 (CRITICAL)

**重要程度**: ⭐⭐⭐⭐⭐ **CRITICAL**
**预计工作量**: 3-5 天
**当前状态**: ❌ 未实现
**Homelab 依赖**: 所有 playbook 都使用 roles

#### 功能描述

Ansible Roles 是代码组织和复用的核心机制，homelab 项目完全基于 roles 架构。

#### 核心特性

- [ ] **Role 目录结构**
  ```
  roles/
    common/
      tasks/main.yaml        # 必需
      handlers/main.yaml     # 可选
      defaults/main.yaml     # 可选
      vars/main.yaml         # 可选
      templates/            # 可选
      files/                # 可选
  ```

- [ ] **Play 级别 roles 关键字**
  ```yaml
  - name: Deploy service
    hosts: servers
    roles:
      - common
      - nginx
  ```

- [ ] **Role 变量优先级**
  - defaults/main.yaml (最低优先级)
  - vars/main.yaml (高优先级)
  - Play vars 覆盖 role defaults

- [ ] **Role 路径解析**
  - 相对于 playbook 的 `./roles/` 目录
  - 相对于当前目录的 `./roles/` 目录
  - 可配置的 roles_path

- [ ] **自动加载 Role 组件**
  - 自动执行 tasks/main.yaml
  - 自动加载 handlers/main.yaml
  - 自动加载 defaults/main.yaml 和 vars/main.yaml
  - template 和 copy 模块自动查找 role 的 templates/ 和 files/ 目录

#### 实现位置

- 数据结构: `pkg/playbook/types.go` (Role, Play.Roles)
- 加载逻辑: `pkg/playbook/role_loader.go` (新建)
- 执行逻辑: `pkg/playbook/runner.go` (扩展 ExecutePlay)
- 路径解析: `pkg/playbook/path.go` (新建)

#### 测试文件

- `tests/playbooks/test-roles-basic.yml`
- `tests/playbooks/test-roles-vars.yml`
- `tests/roles/test_role/` (测试用 role)

#### 实现步骤

1. 定义 Role 数据结构
2. 实现 Role 目录扫描和加载
3. 实现 Role 变量合并
4. 集成到 Play 执行流程
5. 修改 template/copy 模块支持 role 路径
6. 编写测试用例

---

### 🔴 优先级 1: Task 包含机制 (CRITICAL)

**重要程度**: ⭐⭐⭐⭐⭐ **CRITICAL**
**预计工作量**: 2-3 天
**当前状态**: ❌ 未实现
**Homelab 依赖**: 26 次使用（14 次 import_tasks + 12 次 include_role）

#### 功能描述

Task 包含机制允许将任务分散到多个文件中，提高代码复用性和可维护性。

#### 核心特性

- [ ] **import_tasks** (静态包含)
  ```yaml
  - name: Install software
    ansible.builtin.import_tasks: install.yaml
  ```
  - 编译时展开
  - 不支持循环
  - 支持 tags 继承

- [ ] **include_role** (动态包含 role 的部分内容)
  ```yaml
  - name: Ensure directories
    ansible.builtin.include_role:
      name: common
      tasks_from: ensure_directories
    vars:
      directory_list: ["/opt/app"]
  ```
  - 运行时包含
  - 可以只包含 role 的特定任务文件
  - 可以传递变量

- [ ] **任务文件路径解析**
  - 相对于当前 playbook 文件
  - 相对于 role 的 tasks/ 目录
  - 支持绝对路径

- [ ] **变量作用域**
  - include 时可以传递 vars
  - 被包含的任务可以访问这些变量

#### 实现位置

- 数据结构: `pkg/playbook/types.go` (ImportTasks, IncludeRole)
- 加载逻辑: `pkg/playbook/include_loader.go` (新建)
- 执行逻辑: `pkg/playbook/runner.go` (扩展任务执行)

#### 测试文件

- `tests/playbooks/test-import-tasks.yml`
- `tests/playbooks/test-include-role.yml`
- `tests/playbooks/tasks/subtasks.yml`

---

### 🔴 优先级 2: systemd 模块 (CRITICAL)

**重要程度**: ⭐⭐⭐⭐⭐ **CRITICAL**
**预计工作量**: 2-3 天
**当前状态**: ❌ 未实现
**Homelab 依赖**: 9 次使用

#### 功能描述

systemd 模块用于管理 systemd 服务，是 Linux 系统服务管理的核心。

#### 核心特性

- [ ] **daemon_reload** - 重新加载 systemd 配置
  ```yaml
  - name: Reload systemd
    ansible.builtin.systemd:
      daemon_reload: true
  ```

- [ ] **服务状态管理**
  ```yaml
  - name: Start service
    ansible.builtin.systemd:
      name: nginx
      state: started
  ```
  - state: started/stopped/restarted/reloaded

- [ ] **服务使能管理**
  ```yaml
  - name: Enable service
    ansible.builtin.systemd:
      name: nginx
      enabled: yes
  ```

- [ ] **组合操作**
  ```yaml
  - name: Enable and start
    ansible.builtin.systemd:
      name: nginx
      state: started
      enabled: yes
      daemon_reload: yes
  ```

#### 参数

- `name`: 服务名称（.service 后缀可选）
- `state`: started/stopped/restarted/reloaded
- `enabled`: yes/no
- `daemon_reload`: yes/no
- `masked`: yes/no (可选，高级功能)

#### 实现位置

- `pkg/module/systemd.go`
- SSH 执行 systemctl 命令
- 解析命令输出判断成功/失败

#### 测试文件

- `tests/playbooks/test-systemd.yml`

#### 实现注意

- 需要 sudo 权限（依赖 become）
- 不同 systemd 版本可能有细微差异
- 错误处理要详细（服务不存在、权限不足等）

---

### 🔴 优先级 3: become (权限提升) (CRITICAL)

**重要程度**: ⭐⭐⭐⭐⭐ **CRITICAL**
**预计工作量**: 2-3 天
**当前状态**: ❌ 未实现
**Homelab 依赖**: 18 次使用

#### 功能描述

become 机制用于权限提升（通常是 sudo），是执行系统级操作的必需功能。

#### 核心特性

- [ ] **Play 级别 become**
  ```yaml
  - name: System tasks
    hosts: all
    become: true
    tasks:
      - name: Install package
        ...
  ```

- [ ] **Task 级别 become**
  ```yaml
  - name: Edit system file
    copy:
      dest: /etc/config
      content: "..."
    become: true
  ```

- [ ] **become_user** (可选)
  ```yaml
  - name: Run as postgres
    shell: psql -c "..."
    become: true
    become_user: postgres
  ```

- [ ] **become_method** (可选，默认 sudo)
  - sudo (最常用)
  - su
  - pbrun
  - 等

#### 实现位置

- 数据结构: `pkg/playbook/types.go` (Play.Become, Task.Become)
- SSH 命令包装: `pkg/connection/ssh.go` (修改 Execute 方法)
- 命令前缀: `sudo -S` (接受密码从 stdin)

#### 测试文件

- `tests/playbooks/test-become.yml`
- 测试 play 级别和 task 级别 become

#### 实现注意

- 处理 sudo 密码提示（ansible_become_pass）
- 处理 NOPASSWD sudo
- 错误处理（sudo 失败、权限不足）
- 安全性：不在日志中显示密码

---

### 🟠 优先级 4: get_url 模块 (HIGH)

**重要程度**: ⭐⭐⭐⭐☆ **HIGH**
**预计工作量**: 1-2 天
**当前状态**: ❌ 未实现
**Homelab 依赖**: 6 次使用

#### 功能描述

从 URL 下载文件到远程主机。

#### 核心特性

- [ ] **基本下载**
  ```yaml
  - name: Download binary
    ansible.builtin.get_url:
      url: https://example.com/file.tar.gz
      dest: /opt/app/file.tar.gz
  ```

- [ ] **设置文件属性**
  ```yaml
  - name: Download with permissions
    ansible.builtin.get_url:
      url: https://example.com/binary
      dest: /usr/local/bin/app
      mode: '0755'
      owner: root
      group: root
  ```

- [ ] **校验和验证**
  ```yaml
  - name: Download with checksum
    ansible.builtin.get_url:
      url: https://example.com/file
      dest: /tmp/file
      checksum: sha256:abc123...
  ```

- [ ] **条件下载**
  ```yaml
  - name: Download if not exists
    ansible.builtin.get_url:
      url: https://example.com/file
      dest: /tmp/file
      force: no  # 不覆盖已存在的文件
  ```

#### 参数

- `url`: 下载地址（必需）
- `dest`: 目标路径（必需）
- `mode`: 文件权限
- `owner`: 所有者
- `group`: 组
- `force`: 是否覆盖（默认 yes）
- `checksum`: 校验和（可选）
- `timeout`: 超时时间
- `headers`: HTTP 头（可选）

#### 实现位置

- `pkg/module/get_url.go`
- 在远程主机上执行 wget 或 curl
- 或者：在控制节点下载后 copy（更可靠）

#### 测试文件

- `tests/playbooks/test-get-url.yml`

---

### 🟡 优先级 5: tags (标签过滤) (MEDIUM-HIGH)

**重要程度**: ⭐⭐⭐⭐☆
**预计工作量**: 2-3 天
**当前状态**: ❌ 未实现
**Homelab 依赖**: 21 次使用

#### 功能描述

Tags 允许选择性执行 playbook 的部分任务。

#### 核心特性

- [ ] **Task 级别 tags**
  ```yaml
  - name: Install app
    shell: install.sh
    tags:
      - install
      - setup
  ```

- [ ] **命令行过滤**
  ```bash
  ansigo-playbook site.yml --tags install
  ansigo-playbook site.yml --skip-tags config
  ```

- [ ] **特殊 tags**
  - `always`: 总是执行
  - `never`: 从不执行（除非明确指定）

- [ ] **Block 级别 tags**
  ```yaml
  - block:
      - name: Task 1
        ...
    tags: [config]
  ```

#### 实现位置

- 数据结构: `pkg/playbook/types.go` (Task.Tags, Block.Tags)
- CLI 参数: `cmd/ansigo-playbook/main.go` (--tags, --skip-tags)
- 过滤逻辑: `pkg/playbook/runner.go`

#### 测试文件

- `tests/playbooks/test-tags.yml`

---

### 🟡 优先级 6: strategy (执行策略) (MEDIUM)

**重要程度**: ⭐⭐⭐☆☆
**预计工作量**: 1-2 天
**当前状态**: ❌ 未实现（当前默认 linear）
**Homelab 依赖**: 5 次使用

#### 功能描述

Strategy 控制任务在多主机间的执行方式。

#### 核心特性

- [ ] **linear** (默认)
  ```yaml
  - name: Deploy
    hosts: all
    strategy: linear
  ```
  - 所有主机完成 Task 1，再执行 Task 2
  - 当前 AnsiGo 默认行为

- [ ] **free**
  ```yaml
  - name: Deploy
    hosts: all
    strategy: free
  ```
  - 每个主机独立执行所有任务
  - 不等待其他主机
  - 更快，但主机间可能不同步

#### 实现位置

- 数据结构: `pkg/playbook/types.go` (Play.Strategy)
- 执行逻辑: `pkg/playbook/runner.go` (ExecutePlay 方法)
- 并发控制: 使用 goroutine 和 channel

#### 测试文件

- `tests/playbooks/test-strategy-linear.yml`
- `tests/playbooks/test-strategy-free.yml`

---

### 🟢 优先级 7: 其他模块 (MEDIUM-LOW)

#### 7.1 unarchive 模块

**Homelab 依赖**: 2 次使用

```yaml
- name: Extract archive
  ansible.builtin.unarchive:
    src: /tmp/app.tar.gz
    dest: /opt/app
    remote_src: yes
```

参数:
- `src`: 源文件
- `dest`: 解压目录
- `remote_src`: 文件是否在远程主机（yes/no）
- `creates`: 如果存在则跳过

#### 7.2 user 模块

**Homelab 依赖**: 1 次使用

```yaml
- name: Create user
  ansible.builtin.user:
    name: appuser
    state: present
    shell: /bin/bash
    groups: docker
```

参数:
- `name`: 用户名
- `state`: present/absent
- `shell`: 登录 shell
- `groups`: 附加组
- `home`: 家目录

#### 7.3 fail 模块

**Homelab 依赖**: 1 次使用

```yaml
- name: Fail if condition
  ansible.builtin.fail:
    msg: "Required variable is not defined"
  when: required_var is not defined
```

参数:
- `msg`: 失败消息

---

## 📊 Homelab 兼容性实现时间线

| 功能 | 优先级 | 工作量 | 依赖 | 状态 |
|------|--------|--------|------|------|
| Roles 系统 | P0 | 3-5天 | - | ❌ 待实现 |
| import_tasks/include_role | P1 | 2-3天 | Roles | ❌ 待实现 |
| systemd 模块 | P2 | 2-3天 | become | ❌ 待实现 |
| become | P3 | 2-3天 | - | ❌ 待实现 |
| get_url 模块 | P4 | 1-2天 | - | ❌ 待实现 |
| tags | P5 | 2-3天 | - | ❌ 待实现 |
| strategy | P6 | 1-2天 | - | ❌ 待实现 |
| unarchive | P7 | 1天 | - | ❌ 待实现 |
| user | P7 | 1天 | - | ❌ 待实现 |
| fail | P7 | 0.5天 | - | ❌ 待实现 |

**总计**: 约 16-24 天（3-5 周）

---

## 🎯 Homelab 兼容性里程碑

### Milestone 1: 代码组织 (第 1-2 周)
- Roles 系统
- import_tasks/include_role
- **目标**: 能够加载和执行 homelab 的 role 结构

### Milestone 2: 系统管理 (第 3 周)
- become (权限提升)
- systemd 模块
- **目标**: 能够管理系统服务

### Milestone 3: 完整功能 (第 4-5 周)
- get_url 模块
- tags
- strategy
- 其他辅助模块
- **目标**: 完整运行 homelab 项目

---

## 旧功能规划（保留）

### ~~🔴 优先级 1: Handlers 和 Notify~~ ✅ 已完成

**状态**: ✅ 已于 2025-11-22 完成

**实现位置**:

- 数据结构: `pkg/playbook/types.go` (Handler, Task.Notify, Play.Handlers)
- 执行逻辑: `pkg/playbook/runner.go` (executeHandlers, executeHandlerTask)
- 测试文件: `tests/playbooks/test-handlers*.yml`

**详细文档**: [PHASE4_HANDLERS_SUMMARY.md](./PHASE4_HANDLERS_SUMMARY.md)

---

### ~~🟠 优先级 1 (新): 循环功能增强~~ ✅ 已完成

**状态**: ✅ 已于 2025-11-22 完成

**实现位置**:

- 数据结构: `pkg/playbook/types.go` (LoopControl, Task.Loop, Task.LoopControl)
- 执行逻辑: `pkg/playbook/runner.go` (executeTaskWithLoop, printTaskResult)
- 模板引擎: `pkg/playbook/template_jinja2.go` (RenderValue)
- 测试文件: `tests/playbooks/test-loop*.yml`, `test-loops-iteration.yml`

**测试结果**:
- ✅ 基本循环测试通过（静态列表、模板变量、字典列表）
- ✅ Register 与循环配合测试通过
- ✅ 嵌套循环测试通过（loop over results）
- ✅ loop_control 所有选项测试通过
- ✅ 综合循环测试（12 个任务）全部通过

**详细文档**: [PHASE5_LOOPS_SUMMARY.md](./PHASE5_LOOPS_SUMMARY.md)

---

### 🟠 优先级 2 (新): 循环功能扩展

**重要程度**: ⭐⭐⭐☆☆
**预计工作量**: 1-2 天
**当前状态**: 待实现

#### 功能描述
扩展循环功能，支持更多 Ansible 循环特性。

#### 核心特性

- [ ] **loop_control.label**: 简化复杂循环的输出显示
- [ ] **loop_control.extended**: 扩展循环变量（ansible_loop）
- [ ] **dict2items filter**: 字典转列表过滤器
  - 支持遍历字典
  - 每个元素包含 `key` 和 `value`
- [ ] **until 循环**: 重试直到成功
  - `retries`: 重试次数
  - `delay`: 重试间隔

---

### ~~🟡 优先级 3: Block/Rescue/Always~~ ✅ 已完成

**状态**: ✅ 已于 2025-11-22 完成

**实现位置**:

- 数据结构: `pkg/playbook/types.go` (Block, Task.TaskBlock)
- 执行逻辑: `pkg/playbook/runner.go` (executeBlock)
- 测试文件: `tests/playbooks/test-block*.yml`, `demo-blocks.yml`

**测试结果**:
- ✅ Block 任务分组测试通过
- ✅ Rescue 错误恢复测试通过
- ✅ Always 总是执行测试通过
- ✅ 特殊变量 (ansible_failed_task/result) 测试通过
- ✅ Block 级别 when 条件测试通过

**详细文档**: [PHASE6_BLOCKS_SUMMARY.md](./PHASE6_BLOCKS_SUMMARY.md)

---

### 🟡 优先级 3 (新): Block 功能扩展

**重要程度**: ⭐⭐⭐☆☆
**预计工作量**: 1-2 天
**当前状态**: 待实现

#### 功能描述
扩展 Block 功能，支持更多 Ansible Block 特性。

#### 核心特性

- [ ] **Block 级别指令支持**
  - become: 提权执行 block 内所有任务
  - tags: 标签过滤
  - 其他 Play/Task 级别指令

- [ ] **嵌套 Block 支持**
  - Block 内部再定义 block
  - 更复杂的错误处理策略

- [ ] **ansible_failed_task 完整字段**
  - 添加模块名称
  - 添加任务参数

---

### 🟢 优先级 4: 常用模块

**重要程度**: ⭐⭐⭐⭐☆
**预计工作量**: 每个模块 1-2 天

#### 4.1 file 模块

**参考**: Ansible builtin.file 文档

##### 功能
- 创建/删除文件和目录
- 修改权限 (mode)
- 修改所有者 (owner/group)
- 创建符号链接
- Touch 文件

##### 参数
- `path`: 文件路径（必需）
- `state`: 状态 (file, directory, link, absent, touch)
- `mode`: 权限（八进制）
- `owner`: 所有者
- `group`: 组
- `recurse`: 递归应用（目录）
- `src`: 链接源（state=link 时）

##### 实现文件
`pkg/module/file.go`

---

#### 4.2 template 模块

**参考**: Ansible builtin.template 文档

##### 功能
- 渲染 Jinja2 模板文件
- 部署到目标主机
- 支持变量替换

##### 参数
- `src`: 模板文件路径（控制节点）
- `dest`: 目标路径（远程主机）
- `mode`: 权限
- `owner`: 所有者
- `group`: 组
- `backup`: 备份原文件
- `validate`: 验证命令

##### 实现文件
`pkg/module/template.go`

##### 特别注意
- 需要读取本地模板文件
- 使用已有的 Jinja2 引擎渲染
- 传输渲染后的内容到远程

---

#### 4.3 lineinfile 模块

**参考**: Ansible builtin.lineinfile 文档

##### 功能
- 确保文件中存在某一行
- 使用正则表达式匹配
- 支持插入位置控制

##### 参数
- `path`: 文件路径
- `line`: 要确保存在的行
- `regexp`: 匹配正则表达式
- `state`: present/absent
- `insertafter`: 在匹配行之后插入
- `insertbefore`: 在匹配行之前插入
- `create`: 文件不存在时创建

---

#### 4.4 service 模块

**参考**: Ansible builtin.service 文档

##### 功能
- 管理系统服务
- 启动/停止/重启服务
- 设置开机自启

##### 参数
- `name`: 服务名称
- `state`: started/stopped/restarted/reloaded
- `enabled`: 开机自启 (yes/no)

##### 实现注意
- 支持 systemd
- 支持 service 命令（兼容性）

---

### 🔵 优先级 5: 魔法变量

**重要程度**: ⭐⭐⭐☆☆
**预计工作量**: 2-3 天
**参考文档**: `playbook_guide/playbooks_vars_facts.rst`

#### 功能描述
实现 Ansible 特殊变量（魔法变量）。

#### 核心变量

- [ ] **hostvars**: 所有主机的变量
  ```yaml
  {{ hostvars['web01']['ansible_host'] }}
  ```

- [ ] **groups**: 所有组及其成员
  ```yaml
  {{ groups['webservers'] }}  # ['web01', 'web02']
  ```

- [ ] **group_names**: 当前主机所在的组
  ```yaml
  {{ group_names }}  # ['webservers', 'production']
  ```

- [ ] **ansible_play_hosts**: 当前 play 的主机列表

- [ ] **ansible_play_batch**: 当前批次的主机

#### 实现位置
`pkg/playbook/variables.go` 中的 `GetContext` 方法

---

## 📊 实现时间线（估算）

| 功能 | 优先级 | 工作量 | 预计开始 | 预计完成 |
|------|--------|--------|----------|----------|
| Handlers & Notify | P1 | 2-3天 | 第1周 | 第1周 |
| Loop 增强 | P2 | 2-3天 | 第2周 | 第2周 |
| file 模块 | P4 | 1-2天 | 第3周 | 第3周 |
| template 模块 | P4 | 1-2天 | 第3周 | 第3周 |
| Block/Rescue/Always | P3 | 3-4天 | 第4周 | 第4周 |
| 魔法变量 | P5 | 2-3天 | 第5周 | 第5周 |
| lineinfile 模块 | P4 | 1-2天 | 第6周 | 第6周 |
| service 模块 | P4 | 1-2天 | 第6周 | 第6周 |

**总计**: 约 6-8 周

---

## 🎯 里程碑

### Milestone 1: 配置管理核心 (第 1-2 周)
- ✅ Handlers & Notify
- ✅ Loop 增强

### Milestone 2: 文件管理 (第 3 周)
- ✅ file 模块
- ✅ template 模块

### Milestone 3: 错误处理 (第 4 周)
- ✅ Block/Rescue/Always

### Milestone 4: 进阶功能 (第 5-6 周)
- ✅ 魔法变量
- ✅ lineinfile 模块
- ✅ service 模块

---

## 📝 实现规范

### 代码质量要求
1. **每个功能必须有单元测试**
2. **每个功能必须有集成测试（playbook）**
3. **代码需要添加适当的注释**
4. **遵循 Go 编码规范**
5. **错误信息清晰易懂**

### 文档要求
1. **功能实现后更新兼容性分析文档**
2. **添加使用示例到 tests/playbooks/**
3. **更新 README.md 功能列表**

### 测试要求
1. **单元测试覆盖率 > 80%**
2. **集成测试覆盖核心场景**
3. **性能测试（针对循环等高频操作）**

---

## 🔗 相关文档

- [Ansible 兼容性分析](./ANSIBLE_COMPATIBILITY_ANALYSIS.md)
- [Jinja2 兼容性文档](./JINJA2_COMPATIBILITY.md)
- [Phase 3 总结](./PHASE3_SUMMARY.md)
- [架构设计](./design/architecture_overview.md)
- [官方文档](./ansible-documentation/docs/docsite/rst/playbook_guide/)

---

## 📌 注意事项

1. **优先级可能调整**: 根据实际开发过程和用户反馈调整
2. **工作量为估算**: 实际可能有偏差
3. **保持向后兼容**: 新功能不应破坏已有功能
4. **渐进式实现**: 每个功能先实现核心，再完善细节
5. **及时更新文档**: 功能实现后立即更新相关文档

---

最后更新: 2025-11-22
