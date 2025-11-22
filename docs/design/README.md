# Ansigo 设计文档索引

本目录包含 Ansigo 项目的详细设计文档。

## 文档列表

### 1. [架构总览](./architecture_overview.md)
整体架构设计，包括：
- 系统架构图
- 核心组件说明
- 数据流分析
- 目录结构
- 关键设计决策

### 2. 分阶段详细设计

#### [阶段 1: 核心连接与 Ad-hoc 执行](./phase1_connectivity.md)
- Inventory Manager（清单管理）
  - INI/YAML 格式支持
  - 默认组和嵌套组
  - 变量优先级
- Connection Manager（SSH 连接）
- Ad-hoc Runner（执行器）
- 基础模块（ping, raw）

#### [阶段 2: 模块执行引擎](./phase2_module_execution.md)
- 模块工作流分析
- Ansiballz vs WANT_JSON 协议
- 模块发现和执行
- 参数传递和结果解析

#### [阶段 3: 剧本支持](./phase3_playbook.md)
- Playbook YAML 解析
- 线性执行策略
- 变量管理和模板引擎
- 条件判断（when）

### 3. [兼容性验证方案](./compatibility_verification.md)
- 对比测试方法
- 测试框架设计
- 测试用例集
- CI/CD 集成
- 验证里程碑

## 阅读顺序建议

1. **初次了解**: 先阅读[架构总览](./architecture_overview.md)，理解整体设计
2. **实施参考**: 按阶段顺序阅读 Phase 1 → Phase 2 → Phase 3
3. **质量保证**: 阅读[兼容性验证方案](./compatibility_verification.md)了解测试策略

## 设计原则

1. **兼容优先**: 与 Ansible 核心功能保持兼容
2. **渐进实现**: 分阶段实现，每个阶段可独立验证
3. **简洁高效**: 使用 Go 语言特性，避免过度设计
4. **可测试性**: 每个组件都有明确的接口和测试策略
