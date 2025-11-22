# 兼容性验证方案

## 目标
确保 `ansigo` 的行为与原生 Ansible 保持一致，通过自动化测试验证兼容性。

## 1. 验证策略

### 1.1 对比测试法
**原理**: 在相同的测试环境下，分别运行 `ansible` 和 `ansigo`，对比输出结果。

**测试环境**:
- **目标主机**: Docker 容器运行 Ubuntu 24.04 + SSH
- **控制节点**: 本地开发机器
- 统一的 inventory 文件
- 统一的 playbook/命令

#### Docker 测试容器配置

**Dockerfile** (`tests/docker/Dockerfile`):
```dockerfile
FROM ubuntu:24.04

# 安装 SSH 服务器和 Python（Ansible 模块依赖）
RUN apt-get update && \
    apt-get install -y openssh-server python3 python3-pip sudo && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# 配置 SSH
RUN mkdir /var/run/sshd && \
    echo 'root:testpass' | chpasswd && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config

# 创建测试用户
RUN useradd -m -s /bin/bash testuser && \
    echo 'testuser:testpass' | chpasswd && \
    usermod -aG sudo testuser && \
    echo 'testuser ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers

# 配置 SSH 密钥认证
RUN mkdir -p /home/testuser/.ssh && \
    chmod 700 /home/testuser/.ssh && \
    chown testuser:testuser /home/testuser/.ssh

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D"]
```

**启动测试环境**:
```bash
# 构建镜像
docker build -t ansigo-test:ubuntu24 tests/docker/

# 启动容器
docker run -d --name ansigo-test-node \
  -p 2222:22 \
  ansigo-test:ubuntu24

# 配置 SSH 密钥（可选，用于密钥认证测试）
ssh-keygen -t rsa -f tests/ssh_keys/test_key -N ""
docker exec ansigo-test-node bash -c \
  "echo '$(cat tests/ssh_keys/test_key.pub)' >> /home/testuser/.ssh/authorized_keys"
docker exec ansigo-test-node chown testuser:testuser /home/testuser/.ssh/authorized_keys
docker exec ansigo-test-node chmod 600 /home/testuser/.ssh/authorized_keys
```

**测试 Inventory** (`tests/inventory/hosts`):
```ini
[test_nodes]
test-node-1 ansible_host=127.0.0.1 ansible_port=2222 ansible_user=testuser ansible_password=testpass

[test_nodes:vars]
ansible_python_interpreter=/usr/bin/python3
```

### 1.2 验证维度

#### 1.2.1 功能正确性
- **连接**: 能否成功连接到目标主机
- **模块执行**: 模块返回的 JSON 结构是否一致
- **变量解析**: 变量替换结果是否一致
- **条件判断**: `when` 条件的执行逻辑是否一致

#### 1.2.2 输出一致性
- **JSON 结构**: 模块返回的字段名和类型
- **退出码**: 成功/失败的退出码
- **错误信息**: 错误场景下的提示信息

## 2. 测试框架设计

### 2.1 测试脚本结构
```bash
#!/bin/bash
# verify_compat.sh

set -e

# 1. 启动测试容器（Ubuntu 24.04）
echo "Starting test container..."
docker run -d --name ansigo-test-node \
  -p 2222:22 \
  ansigo-test:ubuntu24

# 等待 SSH 服务启动
sleep 3

# 验证 SSH 可用
until ssh -o StrictHostKeyChecking=no -p 2222 testuser@127.0.0.1 -o PasswordAuthentication=yes echo "SSH ready" 2>/dev/null; do
  echo "Waiting for SSH..."
  sleep 1
done

# 2. 运行 Ansible
echo "Running Ansible..."
ansible -i tests/inventory/hosts -m ping all > ansible_output.json 2>&1
ANSIBLE_RC=$?

# 3. 运行 Ansigo
echo "Running Ansigo..."
ansigo -i tests/inventory/hosts -m ping all > ansigo_output.json 2>&1
ANSIGO_RC=$?

# 4. 对比退出码
if [ $ANSIBLE_RC -ne $ANSIGO_RC ]; then
    echo "FAIL: Exit codes differ (Ansible: $ANSIBLE_RC, Ansigo: $ANSIGO_RC)"
    docker stop ansigo-test-node && docker rm ansigo-test-node
    exit 1
fi

# 5. 对比 JSON 输出（忽略时间戳等动态字段）
python3 compare_json.py ansible_output.json ansigo_output.json

# 6. 清理
echo "Cleaning up..."
docker stop ansigo-test-node && docker rm ansigo-test-node

echo "Test completed successfully!"
```

### 2.2 JSON 对比工具
```python
# compare_json.py
import json
import sys

def normalize(data):
    """移除时间戳、路径等动态字段"""
    if isinstance(data, dict):
        return {k: normalize(v) for k, v in data.items() 
                if k not in ['_ansible_parsed', 'invocation', 'delta']}
    elif isinstance(data, list):
        return [normalize(item) for item in data]
    return data

def compare(file1, file2):
    with open(file1) as f1, open(file2) as f2:
        data1 = normalize(json.load(f1))
        data2 = normalize(json.load(f2))
    
    if data1 == data2:
        print("PASS: Outputs match")
        return 0
    else:
        print("FAIL: Outputs differ")
        print(f"Ansible: {data1}")
        print(f"Ansigo:  {data2}")
        return 1

if __name__ == '__main__':
    sys.exit(compare(sys.argv[1], sys.argv[2]))
```

## 3. 测试用例集

### 3.1 Phase 1 测试用例
| 用例     | Ansible 命令                                            | 预期结果             |
| -------- | ------------------------------------------------------- | -------------------- |
| 基础连接 | `ansible -i hosts -m ping all`                          | 所有主机返回 `pong`  |
| 主机筛选 | `ansible -i hosts -m ping webservers`                   | 仅 webservers 组响应 |
| 变量使用 | `ansible -i hosts -m debug -a "msg={{ ansible_host }}"` | 输出各主机的 IP      |

### 3.2 Phase 2 测试用例
| 用例    | 模块      | 参数                  | 验证点                  |
| ------- | --------- | --------------------- | ----------------------- |
| Command | `command` | `uptime`              | stdout 包含系统运行时间 |
| Shell   | `shell`   | `echo $HOME`          | 支持 shell 变量展开     |
| Copy    | `copy`    | `src=file dest=/tmp/` | 文件成功传输            |

### 3.3 Phase 3 测试用例
```yaml
# test_playbook.yml
- hosts: all
  gather_facts: no
  tasks:
    - name: Test ping
      ping:
      register: ping_result
    
    - name: Use registered var
      debug:
        msg: "Ping status: {{ ping_result.ping }}"
    
    - name: Conditional task
      debug:
        msg: "This should run"
      when: ping_result.ping == "pong"
```

**验证**: 运行 `ansible-playbook` 和 `ansigo-playbook`，对比 Play Recap 输出。

## 4. 持续集成

### 4.1 CI 流程
```yaml
# .github/workflows/compat-test.yml
name: Compatibility Test
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Install Ansible
        run: pip install ansible-core
      
      - name: Build Ansigo
        run: go build -o ansigo ./cmd/ansigo
      
      - name: Run compatibility tests
        run: ./tests/verify_compat.sh
```

### 4.2 测试覆盖率目标
- **Phase 1**: 覆盖 Inventory 解析、SSH 连接、基础模块执行
- **Phase 2**: 覆盖至少 5 个常用模块 (ping, command, shell, copy, file)
- **Phase 3**: 覆盖线性 Playbook、变量、条件判断

## 5. 差异记录

### 5.1 已知差异
维护一个 `DIFFERENCES.md` 文档，记录有意为之的差异：

```markdown
# Ansigo 与 Ansible 的差异

## 不支持的功能
- [ ] Jinja2 高级过滤器（Phase 1-3 不支持）
- [ ] Roles 和 Collections
- [ ] 动态 Inventory

## 行为差异
- 错误信息格式可能略有不同（保留核心信息）
- 性能特征（Go vs Python）
```

## 6. 验证里程碑

### Phase 1 验证通过标准
- ✅ 成功解析 INI 和 YAML inventory
- ✅ 连接到 SSH 主机
- ✅ 执行 `ping` 模块，输出与 Ansible 一致

### Phase 2 验证通过标准
- ✅ 至少 3 个模块的输出完全一致
- ✅ 参数传递正确
- ✅ 错误处理行为一致

### Phase 3 验证通过标准
- ✅ Playbook 执行顺序正确
- ✅ 变量替换结果一致
- ✅ Play Recap 统计准确
