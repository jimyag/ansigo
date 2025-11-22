# Claude 开发指南

本文档为 AI 助手（Claude）提供 AnsiGo 项目的开发规范和指南。

## 开发规范

### Ansible 功能兼容性开发

**重要：在开始设计/开发任何兼容 Ansible 的功能之前，必须先查看 Ansible 官方文档的定义。**

1. **查阅官方文档**
   - 所有 Ansible 官方文档位于：`docs/ansible-documentation/`
   - 在实现任何 Ansible 功能之前，先在该目录中查找相关文档
   - 理解 Ansible 官方的功能定义、参数、行为和最佳实践

2. **实现原则**
   - 严格按照 Ansible 官方文档的定义进行实现
   - 保持与 Ansible 的兼容性和一致性
   - 参数名称、行为、返回值等应与 Ansible 保持一致

3. **开发流程**
   ```
   1. 确定要实现的 Ansible 功能
   2. 在 docs/ansible-documentation/ 中查找相关文档
   3. 仔细阅读官方定义和示例
   4. 基于官方文档进行设计和实现
   5. 编写测试用例验证兼容性
   ```

4. **示例**
   - 实现新的模块（如 `copy`, `shell`, `command` 等）时，先查看 `docs/ansible-documentation/` 中对应的模块文档
   - 实现 Playbook 功能（如循环、条件、变量等）时，先查看相关的 Playbook 文档
   - 实现 Jinja2 模板功能时，先查看模板相关文档

## 技术栈

- Go 1.25
- Zerolog（日志库）
- Pongo2（Jinja2 模板引擎）
- YAML v3（配置解析）

## 项目结构

- `cmd/` - 命令行工具
- `pkg/` - 核心包
  - `logger/` - 日志系统
  - `inventory/` - 主机清单管理
  - `playbook/` - Playbook 执行引擎
  - `module/` - Ansible 模块实现
  - `connection/` - SSH 连接管理
- `tests/` - 测试文件
  - `playbooks/` - 测试用 Playbook
  - `scripts/` - 测试脚本
- `docs/` - 文档
  - `ansible-documentation/` - Ansible 官方文档（参考用）

## 代码规范

1. **日志输出**
   - 使用 `pkg/logger` 包进行日志记录
   - Ansible 风格的输出使用 `AnsibleLogger`
   - 保持输出格式与 Ansible 一致（包括颜色、格式等）

2. **错误处理**
   - 使用 `fmt.Errorf` 包装错误信息
   - 保持错误信息清晰、具有上下文

3. **测试**
   - 每个新功能都必须有对应的测试
   - 集成测试用 Playbook 存放在 `tests/playbooks/`
   - 使用 `tests/scripts/run-integration-tests.sh` 运行所有测试

4. **版本控制**
   - `go.mod` 使用 Go 1.25（不带 .0 后缀）
   - 遵循语义化版本规范

## 注意事项

- 不要随意在本地执行二进制文件，所有测试应在 Docker 容器中运行
- 使用 `staticcheck ./...` 检查代码质量
- 保持代码简洁，避免过度设计
