# AnsiGo Tools

本目录包含用于 AnsiGo 的辅助工具。

## Playbook 预处理器

`playbook-preprocessor` 是一个命令行工具，用于预处理 Ansible playbook 文件，将不支持的 Jinja2 语法转换为 pongo2 兼容的语法。

### 功能

- **波浪号操作符转换**: 将 `{{ a ~ b ~ c }}` 转换为 `{{ a }}{{ b }}{{ c }}`
- **内联条件表达式转换**: 将 `{{ 'a' if cond else 'b' }}` 转换为 `{% if cond %}{{ 'a' }}{% else %}{{ 'b' }}{% endif %}`
- **保留原始语法**: 对于不支持的复杂语法（如 `{{ (a ~ b) | filter }}`）会发出警告但保持原样

### 构建

```bash
go build -o bin/playbook-preprocessor tools/playbook-preprocessor/main.go
```

### 使用方法

#### 基本用法

```bash
# 生成新的预处理文件
./bin/playbook-preprocessor -input playbook.yml

# 输出会保存到 playbook_preprocessed.yml
```

#### 指定输出文件

```bash
./bin/playbook-preprocessor -input playbook.yml -output processed.yml
```

#### 就地修改

```bash
./bin/playbook-preprocessor -input playbook.yml -in-place
```

#### 详细输出

```bash
./bin/playbook-preprocessor -input playbook.yml -v
```

### 示例

```bash
# 预处理 Jinja2 过滤器测试文件
./bin/playbook-preprocessor -input tests/playbooks/test-jinja2-filters.yml -v

# 输出:
# 2025/11/22 21:50:55 ✏️  Converted tilde: {{ app_name ~ '-' ~ app_version }} → {{ app_name }}{{ '-' }}{{ app_version }}
# 2025/11/22 21:50:55 ⚠️  Warning: Found tilde operator with filter, keeping as-is: {{ (app_name ~ '-' ~ app_version) | upper }}
# 2025/11/22 21:50:55 📊 Converted 1 tilde operators
# 2025/11/22 21:50:55 ✏️  Converted conditional: {{ 'enabled' if config.debug else 'disabled' }} → {% if config.debug %}{{ 'enabled' }}{% else %}{{ 'disabled' }}{% endif %}
# 2025/11/22 21:50:55 📊 Converted 2 inline conditionals
# ✅ Preprocessed playbook written to: tests/playbooks/test-jinja2-filters_preprocessed.yml
```

---

## AnsiGo Playbook Wrapper

`ansigo-playbook-wrapper.sh` 是一个 Bash 脚本，自动预处理 playbook 然后执行，对用户透明。

### 功能

- 自动检测并运行 playbook 预处理器
- 创建临时预处理文件
- 执行预处理后的 playbook
- 自动清理临时文件（可选保留）

### 使用方法

#### 基本用法

```bash
# 自动预处理并执行
./tools/ansigo-playbook-wrapper.sh -i hosts.ini playbook.yml
```

#### 跳过预处理

```bash
# 直接执行不预处理
./tools/ansigo-playbook-wrapper.sh --no-preprocess -i hosts.ini playbook.yml
```

#### 保留预处理文件

```bash
# 保留预处理后的文件用于调试
./tools/ansigo-playbook-wrapper.sh --keep-preprocessed -i hosts.ini playbook.yml
```

#### 详细输出

```bash
# 显示详细信息
./tools/ansigo-playbook-wrapper.sh -v -i hosts.ini playbook.yml
```

### 环境变量

- `ANSIGO_PREPROCESSOR`: 预处理器路径（默认: `./bin/playbook-preprocessor`）
- `ANSIGO_PLAYBOOK`: AnsiGo playbook 执行器路径（默认: `./bin/ansigo-playbook`）

### 完整示例

```bash
# 1. 构建所需的二进制文件
go build -o bin/ansigo-playbook cmd/ansigo-playbook/main.go
go build -o bin/playbook-preprocessor tools/playbook-preprocessor/main.go

# 2. 执行 playbook（自动预处理）
./tools/ansigo-playbook-wrapper.sh \
    -v \
    -i tests/inventory/hosts.ini \
    tests/playbooks/test-jinja2-filters.yml

# 输出示例:
# [INFO] Preprocessing playbook: tests/playbooks/test-jinja2-filters.yml
# ✅ Preprocessed playbook written to: /tmp/test-jinja2-filters_preprocessed_abc123.yml
# [SUCCESS] Preprocessing completed
# [INFO] Executing playbook: /tmp/test-jinja2-filters_preprocessed_abc123.yml
#
# PLAY [Jinja2 Filters and Features Test] ********************************************
# ...
# [SUCCESS] Playbook execution completed successfully
```

---

## 工作流程

### 推荐的开发工作流程

1. **使用标准 Ansible playbook**
   ```yaml
   - name: Test tilde operator
     debug:
       msg: "{{ app_name ~ '-' ~ app_version }}"
   ```

2. **使用 wrapper 自动处理**
   ```bash
   ./tools/ansigo-playbook-wrapper.sh -i hosts.ini playbook.yml
   ```

3. **调试时查看预处理结果**
   ```bash
   ./tools/ansigo-playbook-wrapper.sh --keep-preprocessed -v -i hosts.ini playbook.yml
   # 检查生成的临时文件
   ```

### CI/CD 集成

```bash
#!/bin/bash
# 在 CI/CD 管道中使用

# 构建
make build

# 运行测试
./tools/ansigo-playbook-wrapper.sh \
    -i tests/inventory/hosts.ini \
    tests/playbooks/ci-test.yml

# 退出码会传递给 CI 系统
```

---

## 已知限制

### 预处理器限制

1. **波浪号 + 过滤器组合** - 暂不支持
   ```yaml
   # ❌ 不支持 - 会发出警告但保持原样
   msg: "{{ (app_name ~ '-' ~ app_version) | upper }}"

   # ✅ 替代方案
   msg: "{{ app_name | upper }}{{ '-' }}{{ app_version | upper }}"
   ```

2. **复杂嵌套表达式** - 可能无法正确处理
   ```yaml
   # ⚠️ 复杂情况需要测试
   msg: "{{ (a ~ b) if cond else (c ~ d) }}"
   ```

### 解决方案

对于不支持的语法，预处理器会：
1. 发出警告信息
2. 保持原始语法不变
3. 让用户在 playbook 执行时看到实际错误
4. 用户可以根据文档手动修改 playbook

---

## 参考文档

- [Jinja2 兼容性说明](../docs/JINJA2_COMPATIBILITY.md)
- [Ansible 兼容性分析](../docs/ANSIBLE_COMPATIBILITY_ANALYSIS.md)
- [AnsiGo 主文档](../README.md)

---

## 贡献

如果您发现预处理器无法正确处理某些 Jinja2 语法，请：

1. 提交 issue 并附上示例 playbook
2. 在 issue 中说明预期行为和实际结果
3. 如果可能，提供 Pull Request 改进预处理逻辑

---

## 许可证

与 AnsiGo 项目主许可证相同。
