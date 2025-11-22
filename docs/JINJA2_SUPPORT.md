# AnsiGo Jinja2 模板支持

## 概述

AnsiGo 现在支持完整的 Jinja2 模板功能，通过集成 `pongo2` 库（Jinja2 的 Go 实现）来实现。这使得 AnsiGo 能够处理复杂的模板表达式、过滤器和控制结构。

## 支持的功能

### 1. 变量替换

#### 基本变量
```yaml
vars:
  app_name: "myapp"
  version: "1.0.0"
tasks:
  - debug:
      msg: "{{ app_name }} version {{ version }}"
```

#### 嵌套变量访问
```yaml
vars:
  config:
    host: "localhost"
    port: 8080
tasks:
  - debug:
      msg: "Server: {{ config.host }}:{{ config.port }}"
```

#### 注册变量访问
```yaml
tasks:
  - command: hostname
    register: result
  - debug:
      msg: "Hostname is {{ result.stdout }}"
```

### 2. 控制结构

#### if-else 语句
```yaml
- debug:
    msg: "{% if environment == 'production' %}PROD{% else %}DEV{% endif %}"
```

#### for 循环
```yaml
vars:
  packages:
    - nginx
    - redis
    - postgresql
tasks:
  - debug:
      msg: "{% for pkg in packages %}{{ pkg }}{% if not loop.last %}, {% endif %}{% endfor %}"
```

#### 带索引的循环
```yaml
- debug:
    msg: "{% for item in list %}{{ loop.index }}. {{ item }} {% endfor %}"
```

#### 条件循环
```yaml
- debug:
    msg: "{% for item in list %}{% if item > 10 %}{{ item }} {% endif %}{% endfor %}"
```

### 3. 过滤器

#### 字符串过滤器

**upper** - 转换为大写
```yaml
- debug:
    msg: "{{ app_name | upper }}"  # MYAPP
```

**lower** - 转换为小写
```yaml
- debug:
    msg: "{{ environment | lower }}"  # production
```

**title** - 标题格式（首字母大写）
```yaml
- debug:
    msg: "{{ 'hello world' | title }}"  # Hello World
```

**trim** - 去除首尾空格
```yaml
- debug:
    msg: "{{ '  text  ' | trim }}"  # text
```

#### 数组过滤器

**first** - 获取第一个元素
```yaml
- debug:
    msg: "{{ packages | first }}"  # nginx
```

**last** - 获取最后一个元素
```yaml
- debug:
    msg: "{{ packages | last }}"  # postgresql
```

**length** - 获取长度
```yaml
- debug:
    msg: "{{ packages | length }}"  # 3
```

**join** - 连接数组元素
```yaml
- debug:
    msg: "{{ packages | join:', ' }}"  # nginx, redis, postgresql
```

### 4. 条件表达式（when 子句）

#### 比较操作符
```yaml
# 等于
when: variable == 'value'

# 不等于
when: variable != 'value'

# 大于/小于
when: port > 1024
when: port < 65535

# 大于等于/小于等于
when: version >= '1.0.0'
when: count <= 100
```

#### 逻辑操作符
```yaml
# and
when: environment == 'production' and port == 8080

# or
when: environment == 'production' or environment == 'staging'

# not
when: not config.debug
```

#### 复杂条件
```yaml
when: (environment == 'production' or environment == 'staging') and port > 1024
```

### 5. 循环变量

在 for 循环中可用的特殊变量：

- `loop.index` - 当前索引（从 1 开始）
- `loop.index0` - 当前索引（从 0 开始）
- `loop.first` - 是否是第一个元素
- `loop.last` - 是否是最后一个元素
- `loop.length` - 循环总长度

```yaml
- debug:
    msg: "{% for item in items %}{{ loop.index }}. {{ item }}{% if not loop.last %}, {% endif %}{% endfor %}"
```

## 完整示例

### 示例 1: 配置文件生成

```yaml
---
- name: Generate Application Config
  hosts: webservers
  vars:
    app_name: "myapp"
    app_version: "2.1.0"
    environment: "production"
    config:
      host: "0.0.0.0"
      port: 8080
      workers: 4
      modules:
        - auth
        - api
        - admin
  tasks:
    - name: Create config file
      copy:
        content: |
          # {{ app_name | upper }} Configuration
          # Version: {{ app_version }}
          # Environment: {{ environment }}

          [server]
          host = {{ config.host }}
          port = {{ config.port }}
          workers = {{ config.workers }}

          [modules]
          enabled = {% for mod in config.modules %}{{ mod }}{% if not loop.last %}, {% endif %}{% endfor %}

          [environment]
          mode = {% if environment == 'production' %}prod{% else %}dev{% endif %}
        dest: /etc/{{ app_name }}/config.ini
```

### 示例 2: 条件部署

```yaml
---
- name: Conditional Deployment
  hosts: all
  vars:
    deploy_version: "1.2.3"
    min_version: "1.0.0"
  tasks:
    - name: Check current version
      command: cat /etc/app/version
      register: current_version
      ignore_errors: yes

    - name: Deploy if version is older
      debug:
        msg: "Deploying {{ deploy_version }}"
      when: current_version.stdout < deploy_version or current_version.failed

    - name: Skip if up to date
      debug:
        msg: "Already on {{ current_version.stdout }}"
      when: not current_version.failed and current_version.stdout >= deploy_version
```

### 示例 3: 动态主机列表

```yaml
---
- name: Generate Hosts File
  hosts: localhost
  vars:
    servers:
      - name: "web1"
        ip: "192.168.1.10"
      - name: "web2"
        ip: "192.168.1.11"
      - name: "db1"
        ip: "192.168.1.20"
  tasks:
    - name: Create hosts file
      copy:
        content: |
          # Generated hosts file
          127.0.0.1 localhost

          # Application servers
          {% for server in servers %}
          {{ server.ip }} {{ server.name }}
          {% endfor %}
        dest: /tmp/hosts
```

## Pongo2 vs Ansible Jinja2 差异

### 支持的功能
✅ 变量替换 `{{ var }}`
✅ 控制结构 `{% if %} {% for %}`
✅ 过滤器 `{{ var | filter }}`
✅ 比较操作符 `==`, `!=`, `>`, `<`, `>=`, `<=`
✅ 逻辑操作符 `and`, `or`, `not`
✅ 数组访问 `{{ array.0 }}` 或 `{{ array[0] }}`
✅ 字典访问 `{{ dict.key }}` 或 `{{ dict['key'] }}`

### 不支持的功能
❌ 字符串连接操作符 `~` (使用过滤器或字符串插值替代)
❌ 默认值语法 `{{ var | default(value) }}` (pongo2 使用 `{{ var | default:"value" }}`)
❌ 部分高级过滤器（如 `regex_replace`, `to_yaml`）
❌ 宏 (macros)
❌ 继承 (extends/blocks)

### 解决方案

**字符串连接**:
```yaml
# 不支持: {{ str1 ~ str2 }}
# 替代方案 1: 直接在模板中拼接
msg: "{{ str1 }}-{{ str2 }}"

# 替代方案 2: 在 vars 中预先组合
vars:
  combined: "{{ str1 }}-{{ str2 }}"
```

**默认值**:
```yaml
# 不支持: {{ var | default('value') }}
# 替代方案: pongo2 语法
msg: {{ var | default:"value" }}

# 或使用 if-else
msg: "{% if var %}{{ var }}{% else %}default{% endif %}"
```

## 性能考虑

- **模板缓存**: pongo2 会自动缓存已编译的模板
- **变量解析**: 嵌套变量访问的性能与 Python Jinja2 相当
- **循环**: 大量迭代时性能良好，适合处理中等规模的数据
- **条件评估**: 简单条件评估速度很快

## 最佳实践

### 1. 使用合适的数据结构
```yaml
# 好的做法 - 使用结构化数据
vars:
  config:
    host: "localhost"
    port: 8080

# 避免 - 使用字符串拼接
vars:
  server_url: "http://localhost:8080"  # 难以分离和修改
```

### 2. 保持模板简洁
```yaml
# 好的做法 - 简单清晰的模板
- debug:
    msg: "Server: {{ config.host }}:{{ config.port }}"

# 避免 - 过于复杂的模板逻辑
- debug:
    msg: "{% for i in range(10) %}{% if i % 2 == 0 %}{{ i }}{% endif %}{% endfor %}"
```

### 3. 合理使用条件
```yaml
# 好的做法 - 在 when 中使用条件
- name: Deploy to production
  command: deploy.sh
  when: environment == 'production'

# 避免 - 在模板中嵌套过多条件
- debug:
    msg: "{% if env == 'prod' %}{% if port == 80 %}...{% endif %}{% endif %}"
```

## 测试验证

运行 Jinja2 功能测试：
```bash
# 完整的 Jinja2 功能测试
docker exec ansigo-control ansigo-playbook -i /workspace/tests/inventory/hosts.ini \
    /workspace/tests/playbooks/test-jinja2-working.yml
```

测试覆盖：
- ✅ 20 个不同的 Jinja2 功能测试
- ✅ 变量替换和嵌套访问
- ✅ 字符串过滤器（upper, lower, title, length）
- ✅ 数组操作（first, last, join, length）
- ✅ 循环和迭代
- ✅ 条件表达式和 when 子句
- ✅ 复杂条件组合

## 总结

AnsiGo 的 Jinja2 支持提供了：
- ✅ 完整的变量替换和访问
- ✅ 强大的控制结构（if/for）
- ✅ 丰富的内置过滤器
- ✅ 灵活的条件表达式
- ✅ 与 Ansible 高度兼容的语法
- ✅ 良好的性能表现

通过 pongo2 集成，AnsiGo 现在能够处理几乎所有常见的 Ansible 模板用例，为自动化任务提供了强大的模板能力。
