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

---

## 📋 待实现功能

根据 Ansible 官方文档和实际使用频率，按优先级排序：

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
