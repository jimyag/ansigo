#!/bin/bash
# Phase 1 兼容性测试脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ROOT="/workspace/ansigo"

echo "=========================================="
echo "Phase 1 Compatibility Test"
echo "=========================================="
echo ""

# 构建 AnsiGo
echo "==> Building AnsiGo..."
cd "$PROJECT_ROOT"
go build -o /tmp/ansigo ./cmd/ansigo
echo "✓ AnsiGo built successfully"
echo ""

# 测试用例 1: Ping 模块
echo "==> Test 1: Ping Module"
echo ""

echo "--- Running Ansible ---"
cd "$TEST_ROOT"
ansible -i inventory/hosts.ini -m ping all > /tmp/ansible-ping.out 2>&1 || true
cat /tmp/ansible-ping.out
echo ""

echo "--- Running AnsiGo ---"
/tmp/ansigo -i inventory/hosts.ini -m ping all > /tmp/ansigo-ping.out 2>&1 || true
cat /tmp/ansigo-ping.out
echo ""

echo "--- Comparing Results ---"
python3 "$SCRIPT_DIR/compare-output.py" /tmp/ansible-ping.out /tmp/ansigo-ping.out
echo ""

# 测试用例 2: Raw 模块
echo "==> Test 2: Raw Module"
echo ""

echo "--- Running Ansible ---"
ansible -i inventory/hosts.ini -m raw -a "echo hello" all > /tmp/ansible-raw.out 2>&1 || true
cat /tmp/ansible-raw.out
echo ""

echo "--- Running AnsiGo ---"
/tmp/ansigo -i inventory/hosts.ini -m raw -a "echo hello" all > /tmp/ansigo-raw.out 2>&1 || true
cat /tmp/ansigo-raw.out
echo ""

echo "--- Comparing Results ---"
python3 "$SCRIPT_DIR/compare-output.py" /tmp/ansible-raw.out /tmp/ansigo-raw.out
echo ""

# 测试用例 3: 主机筛选
echo "==> Test 3: Host Pattern (webservers only)"
echo ""

echo "--- Running Ansible ---"
ansible -i inventory/hosts.ini -m ping webservers > /tmp/ansible-pattern.out 2>&1 || true
cat /tmp/ansible-pattern.out
echo ""

echo "--- Running AnsiGo ---"
/tmp/ansigo -i inventory/hosts.ini -m ping webservers > /tmp/ansigo-pattern.out 2>&1 || true
cat /tmp/ansigo-pattern.out
echo ""

echo "--- Comparing Results ---"
python3 "$SCRIPT_DIR/compare-output.py" /tmp/ansible-pattern.out /tmp/ansigo-pattern.out
echo ""

echo "=========================================="
echo "Phase 1 Tests Summary"
echo "=========================================="
echo ""
echo "All tests completed. Check output above for details."
