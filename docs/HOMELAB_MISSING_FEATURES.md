# Homelab 项目缺失功能深度分析

基于对 `/Users/jimyag/src/github/homelab` 项目的深入分析，以下是 AnsiGo 需要实现的缺失功能清单。

## 已实现的核心功能 ✅

1. ✅ **Roles 系统** - 完全支持 role 结构（tasks, handlers, defaults, templates, files）
2. ✅ **import_tasks / include_role** - 支持任务包含和角色包含
3. ✅ **become** - 权限提升支持
4. ✅ **systemd 模块** - 服务管理
5. ✅ **get_url 模块** - 文件下载
6. ✅ **基础模块** - copy, file, template, command, shell, debug, set_fact
7. ✅ **循环 (loop)** - 基本循环支持
8. ✅ **条件 (when)** - 基本条件判断
9. ✅ **handlers + notify** - Handler 通知机制
10. ✅ **变量渲染** - 基本 Jinja2 模板

---

## 缺失的关键功能分析

### 🔴 优先级 1：阻塞性功能（必须立即实现）

#### 1. **Strategy: free** (使用频率：5 次)
**影响范围**：所有主 playbook 都使用了 `strategy: free`

```yaml
- name: Mihomo
  strategy: free  # ← 当前不支持
  hosts: mihomo
  become: true
```

**功能说明**：
- `strategy: linear`（默认）：按顺序在所有主机上执行每个任务
- `strategy: free`：允许每个主机以自己的速度执行任务，不等待其他主机

**实现优先级**：🔴 **极高** - 所有 playbook 都依赖此功能
**实现复杂度**：⭐⭐⭐⭐ 中高
**预计工作量**：2-3 天

---

#### 2. **Tags 系统** (使用频率：21 次)
**影响范围**：所有 roles 的 tasks

```yaml
- name: Download mihomo binary
  ansible.builtin.get_url:
    url: "..."
    dest: "..."
  tags:
    - install_mihomo  # ← 当前不支持
```

**功能说明**：
- 允许选择性执行带特定标签的任务
- 命令行：`ansible-playbook playbook.yml --tags "install,config"`
- 或跳过：`--skip-tags "download"`

**实现优先级**：🔴 **极高** - 用于快速部署和调试
**实现复杂度**：⭐⭐⭐ 中等
**预计工作量**：1-2 天

---

### 🟠 优先级 2：重要功能（严重影响使用）

#### 3. **Ansible Facts 变量** (使用频率：12 次)
**影响范围**：所有下载二进制文件的任务

```yaml
- name: Download binary
  ansible.builtin.get_url:
    url: "{{ base_url }}/{{ ansible_system | lower }}/{{ ansible_architecture | replace('x86_64', 'amd64') }}"
    #                      ^^^^^^^^^^^^^^          ^^^^^^^^^^^^^^^^^^^^
    #                      需要实现                 需要实现
```

**需要实现的 facts**：
- `ansible_system`: Linux, Darwin, Windows
- `ansible_architecture`: x86_64, aarch64, armv7l
- `ansible_os_family`: Debian, RedHat, Arch, etc.
- `ansible_distribution`: Ubuntu, CentOS, Debian
- `ansible_distribution_version`: 22.04, 8, etc.

**实现优先级**：🟠 **高** - 跨平台部署必需
**实现复杂度**：⭐⭐⭐ 中等
**预计工作量**：1-2 天

---

#### 4. **Jinja2 Filters** (使用频率：72 次)
**影响范围**：变量处理和模板渲染

**必须实现的过滤器**：
```yaml
# 1. default - 使用频率：51 次
{{ mihomo | default({}) }}
{{ mihomo.get('home', default_home) }}

# 2. replace - 使用频率：12 次
{{ ansible_architecture | replace('x86_64', 'amd64') | replace('aarch64', 'arm64') }}

# 3. lower - 使用频率：6 次
{{ ansible_system | lower }}

# 4. length - 使用频率：3 次
{{ systemd_unit_name | length > 0 }}
```

**实现优先级**：🟠 **高** - 所有变量处理依赖
**实现复杂度**：⭐⭐ 简单-中等
**预计工作量**：1 天

---

#### 5. **Jinja2 Tests** (使用频率：8 次)
**影响范围**：条件判断

```yaml
# 1. is defined / is not defined
when: victoriametrics_service_status.status is defined

# 2. is not none
when: systemd_unit_state is not none

# 3. is running (可能需要自定义)
when: service_status is running
```

**实现优先级**：🟠 **高** - 条件判断必需
**实现复杂度**：⭐⭐ 简单-中等
**预计工作量**：半天

---

#### 6. **lookup 插件 - template** (使用频率：5 次)
**影响范围**：动态加载模板内容

```yaml
vars:
  systemd_unit_content: "{{ lookup('template', 'mihomo.service') }}"
  #                         ^^^^^^ 需要实现
```

**实现优先级**：🟠 **高** - systemd 单元部署依赖
**实现复杂度**：⭐⭐ 简单-中等
**预计工作量**：半天

---

#### 7. **Handler listen 关键字** (使用频率：8 次)
**影响范围**：Handler 触发机制

```yaml
# handlers/main.yaml
- name: Reload systemd units
  ansible.builtin.systemd:
    daemon_reload: true
  listen: Reload systemd units  # ← 当前不支持

# tasks
notify:
  - Reload systemd units  # 通过 listen 关键字匹配
```

**功能说明**：多个 handler 可以监听同一个名称，一个 notify 可以触发多个 handler

**实现优先级**：🟠 **高** - Handler 系统增强
**实现复杂度**：⭐⭐ 简单-中等
**预计工作量**：半天

---

### 🟡 优先级 3：增强功能（影响使用体验）

#### 8. **loop_control.label** (使用频率：4 次)
**影响范围**：循环输出简化

```yaml
loop: "{{ mihomo_internal.config_files }}"
loop_control:
  label: "{{ item.dest }}"  # ← 当前不支持，输出时只显示 dest
```

**实现优先级**：🟡 **中** - 改善输出可读性
**实现复杂度**：⭐⭐ 简单
**预计工作量**：半天

---

#### 9. **unarchive 模块** (使用频率：2 次)
**影响范围**：压缩包解压

```yaml
- name: Unarchive victoriametrics binaries
  ansible.builtin.unarchive:
    src: "/tmp/victoriametrics.tar.gz"
    dest: "{{ victoriametrics_home }}"
    remote_src: true  # ← 重要：表示文件在远程主机上
    creates: "{{ victoriametrics_home }}/victoria-metrics"  # ← 幂等性检查
```

**实现优先级**：🟡 **中** - 部署二进制工具需要
**实现复杂度**：⭐⭐⭐ 中等
**预计工作量**：1 天

---

#### 10. **user 模块** (使用频率：1 次)
**影响范围**：用户管理

```yaml
- name: Ensure user jimyag exists
  ansible.builtin.user:
    name: jimyag
    state: present
    shell: /bin/bash
```

**实现优先级**：🟡 **中** - 初始化系统需要
**实现复杂度**：⭐⭐⭐ 中等
**预计工作量**：1 天

---

#### 11. **fail 模块** (使用频率：1 次)
**影响范围**：错误处理

```yaml
- name: Fail if service is not running
  ansible.builtin.fail:
    msg: "victoriametrics service is not running."
  when: service_status.status.ActiveState != 'active'
```

**实现优先级**：🟡 **中** - 错误处理增强
**实现复杂度**：⭐ 简单
**预计工作量**：1 小时

---

### 🟢 优先级 4：完善功能（锦上添花）

#### 12. **copy 模块的 validate 参数**
**影响范围**：配置文件验证

```yaml
- name: Configure passwordless sudo
  ansible.builtin.copy:
    content: "jimyag ALL=(ALL) NOPASSWD: ALL\n"
    dest: /etc/sudoers.d/jimyag
    validate: "visudo -cf %s"  # ← 写入前验证
```

**实现优先级**：🟢 **低** - 安全性增强
**实现复杂度**：⭐⭐ 简单-中等
**预计工作量**：半天

---

#### 13. **register 返回值的结构化字段**
**影响范围**：复杂条件判断

```yaml
- name: Check service status
  ansible.builtin.systemd:
    name: victoriametrics
    state: started
  register: service_status

- name: Use registered result
  debug:
    msg: "{{ service_status.status.ActiveState }}"
    #        ^^^^^^^^^^^^^^^^^^^^^^^ 需要完善返回值结构
```

**实现优先级**：🟢 **低** - 增强 register 功能
**实现复杂度**：⭐⭐ 简单-中等（需要各模块配合）
**预计工作量**：1 天

---

#### 14. **多项 when 条件（列表形式）**
**影响范围**：条件判断

```yaml
when:
  - systemd_unit_name | length > 0
  - systemd_unit_state is not none
  # 列表中的所有条件都必须为 true（AND 关系）
```

**实现优先级**：🟢 **低** - 已支持单行 when
**实现复杂度**：⭐ 简单
**预计工作量**：1 小时

---

#### 15. **group_vars 和 host_vars 自动加载**
**影响范围**：变量管理

**目录结构**：
```
hosts/
  ├── group_vars/
  │   └── main.yaml
  └── host_vars/
      ├── ipc.yaml
      ├── jp.yaml
      └── ...
```

**实现优先级**：🟢 **低** - 变量管理增强
**实现复杂度**：⭐⭐ 简单-中等
**预计工作量**：半天

---

## 功能实现优先级总结

### 第一阶段（阻塞性）- 预计 3-5 天
1. 🔴 **strategy: free** - 并发执行策略
2. 🔴 **tags** - 任务标签系统

### 第二阶段（重要性）- 预计 4-6 天
3. 🟠 **Ansible Facts** - ansible_system, ansible_architecture 等
4. 🟠 **Jinja2 Filters** - default, replace, lower, length
5. 🟠 **Jinja2 Tests** - is defined, is not defined, is not none
6. 🟠 **lookup('template')** - 模板查找插件
7. 🟠 **handler listen** - Handler 监听机制

### 第三阶段（增强性）- 预计 3-4 天
8. 🟡 **loop_control.label** - 循环输出优化
9. 🟡 **unarchive 模块** - 解压缩支持
10. 🟡 **user 模块** - 用户管理
11. 🟡 **fail 模块** - 显式失败

### 第四阶段（完善性）- 预计 2-3 天
12. 🟢 **copy validate** - 文件验证
13. 🟢 **register 字段** - 返回值结构化
14. 🟢 **多项 when** - 列表条件
15. 🟢 **group_vars/host_vars** - 变量文件加载

---

## 总预计工作量

- **最小可用版本（第一阶段）**：3-5 天
- **基本可用版本（第一+第二阶段）**：7-11 天
- **完整可用版本（所有阶段）**：15-22 天

---

## 当前 AnsiGo vs Homelab 兼容性

| 功能类别 | 已实现 | 缺失 | 兼容率 |
|---------|--------|------|--------|
| 核心模块 | 10 | 3 | 77% |
| Playbook 特性 | 7 | 8 | 47% |
| Jinja2 功能 | 基础渲染 | 过滤器+测试 | 30% |
| 变量系统 | 基础变量 | Facts+文件加载 | 50% |
| **总体兼容性** | **-** | **-** | **≈55%** |

---

## 建议实施策略

### 快速可用（MVP）
仅实现第一阶段（strategy + tags），可以基本运行 homelab playbooks，但功能受限。

### 生产可用（Recommended）
实现第一+第二阶段，达到 80%+ 兼容性，可以完整运行 homelab 项目。

### 完全兼容（Full）
实现所有四个阶段，达到 95%+ 兼容性，支持所有 homelab 高级特性。
