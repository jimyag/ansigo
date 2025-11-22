# AnsiGo 测试快速入门

快速运行 AnsiGo 测试套件的指南。

## 🚀 快速开始

### 1. 运行所有测试（推荐）

```bash
./tests/scripts/run-all-tests.sh
```

这会运行:
- ✅ 单元测试 (所有 pkg 包)
- ✅ 集成测试 (所有 playbook)
- ✅ 代码检查 (go vet)

### 2. 仅运行单元测试

```bash
go test ./pkg/... -v
```

带覆盖率:
```bash
go test ./pkg/... -v -cover
```

### 3. 仅运行集成测试

```bash
./tests/scripts/run-integration-tests.sh
```

## 📋 测试分类

### 单元测试 (Unit Tests)

快速、隔离的组件测试:

```bash
# 测试 inventory 解析
go test ./pkg/inventory -v

# 测试 Jinja2 模板引擎
go test ./pkg/playbook -v -run TestJinja2

# 测试变量管理器
go test ./pkg/playbook -v -run TestVariable

# 测试模块执行器
go test ./pkg/module -v
```

### 集成测试 (Integration Tests)

端到端功能测试:

```bash
# 测试核心模块
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-modules.yml

# 测试变量优先级
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-variable-precedence.yml

# 测试循环
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-loops-iteration.yml

# 测试错误处理
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-error-handling.yml
```

## 🎯 测试特定功能

### Jinja2 模板

```bash
# 基础功能
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-jinja2-working.yml

# 过滤器
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-jinja2-filters.yml

# 循环
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-jinja2-loops.yml
```

### 条件执行

```bash
# 基础条件
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-conditionals.yml

# 高级条件
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-when-conditions.yml
```

### 多主机功能

```bash
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-multi-host.yml
```

## 🔍 调试测试

### 查看详细输出

```bash
# 单元测试详细输出
go test ./pkg/playbook -v -run TestJinja2TemplateEngine_RenderString

# 集成测试详细输出 (已包含在 playbook 输出中)
./bin/ansigo-playbook -i tests/inventory/test_hosts tests/playbooks/test-modules.yml
```

### 运行单个测试用例

```bash
# 运行特定单元测试
go test ./pkg/inventory -v -run TestParseINI/simple_host

# 运行特定模板测试
go test ./pkg/playbook -v -run TestJinja2TemplateEngine_RenderString/simple_variable
```

### 测试失败时

1. 查看测试输出中的错误信息
2. 检查 `got` vs `want` 值
3. 检查相关的源代码
4. 运行单个失败的测试以隔离问题

## 📊 测试覆盖率

### 生成覆盖率报告

```bash
# 生成覆盖率文件
go test ./pkg/... -coverprofile=coverage.out

# 查看覆盖率报告
go tool cover -html=coverage.out
```

### 当前覆盖率

- `pkg/inventory`: **54.1%**
- `pkg/playbook`: **17.3%**
- `pkg/module`: **5.8%**

## ✅ 测试检查清单

在提交代码前:

- [ ] 运行 `go test ./pkg/...` - 所有单元测试通过
- [ ] 运行 `./tests/scripts/run-integration-tests.sh` - 所有集成测试通过
- [ ] 运行 `go vet ./...` - 无代码问题
- [ ] 运行 `go mod tidy` - 依赖整理

或者简单运行:

```bash
./tests/scripts/run-all-tests.sh
```

## 🐛 常见问题

### 问题: 找不到 ansigo-playbook

**解决**:
```bash
cd /Users/jimyag/src/github/ansigo
go build -o bin/ansigo-playbook ./cmd/ansigo-playbook
```

### 问题: 测试失败 "connection refused"

**解决**: Docker 测试环境未运行
```bash
./tests/scripts/setup-test-env.sh
docker ps | grep ansigo  # 验证容器运行
```

### 问题: 模块依赖错误

**解决**:
```bash
go mod download
go mod tidy
```

### 问题: 集成测试超时

**原因**: 可能是 Docker 容器响应慢

**解决**:
1. 检查 Docker 资源限制
2. 增加测试脚本中的超时时间
3. 查看 Docker 日志: `docker logs ansigo-target-1`

## 📚 测试文档

- **详细测试说明**: [INTEGRATION_TESTS.md](INTEGRATION_TESTS.md)
- **测试目录说明**: [README.md](README.md)

## 🔧 测试开发

### 添加单元测试

1. 创建 `*_test.go` 文件
2. 使用表驱动测试模式:

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name string
        input string
        want string
    }{
        {"case1", "input1", "output1"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := MyFunction(tt.input)
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 添加集成测试

1. 在 `tests/playbooks/` 创建 YAML 文件
2. 在 `tests/scripts/run-integration-tests.sh` 添加:

```bash
if [ -f "${PLAYBOOKS_DIR}/test-my-feature.yml" ]; then
    run_test "My Feature" "${PLAYBOOKS_DIR}/test-my-feature.yml" "true"
fi
```

## 💡 最佳实践

1. **每次提交前运行测试** - 确保不破坏现有功能
2. **编写新功能时先写测试** - TDD 方法
3. **保持测试简单明了** - 每个测试只验证一个功能点
4. **使用描述性的测试名称** - 易于理解测试意图
5. **测试失败场景** - 不仅测试成功路径

## 🎓 测试示例

### 简单单元测试

```go
func TestSimpleVariable(t *testing.T) {
    engine := NewJinja2TemplateEngine()
    result, err := engine.RenderString("Hello {{ name }}", map[string]interface{}{
        "name": "World",
    })

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if result != "Hello World" {
        t.Errorf("got %q, want %q", result, "Hello World")
    }
}
```

### 简单集成测试

```yaml
---
- name: Simple Test
  hosts: all
  tasks:
    - name: Test ping
      ping:

    - name: Test command
      command: echo "test"
      register: result

    - name: Verify result
      debug:
        msg: "Output: {{ result.stdout }}"
```

## 🚦 CI/CD 集成

### GitHub Actions 示例

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: ./tests/scripts/run-all-tests.sh
```

---

**提示**: 保持测试快速、独立、可重复！ 🎉
