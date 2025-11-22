# AnsiGo

AnsiGo（发音 /ˈæn-si-goʊ/）现代化、轻量、可编译的 Ansible 执行引擎（Go 实现）

AnsiGo = Ansible Compatibility + Golang Re-Engineered

## 项目概述

AnsiGo 旨在使用 Go 语言重新实现 Ansible 核心功能，提供：
- **单一二进制文件**：无需 Python 依赖，易于部署
- **高性能**：利用 Go 的并发特性
- **Ansible 兼容**：兼容现有 Ansible inventory 和 playbook

## 文档

详细设计文档请查看 [docs/design/](docs/design/) 目录：
- [架构总览](docs/design/architecture_overview.md)
- [阶段 1: 核心连接与 Ad-hoc 执行](docs/design/phase1_connectivity.md)
- [阶段 2: 模块执行引擎](docs/design/phase2_module_execution.md)
- [阶段 3: Playbook 支持](docs/design/phase3_playbook.md)
- [兼容性验证方案](docs/design/compatibility_verification.md)

## 开发状态

当前处于 Phase 1 实现阶段。

## 快速开始

```bash
# 构建
go build -o ansigo ./cmd/ansigo

# 运行 ad-hoc 命令
./ansigo -i inventory.ini -m ping all
```
