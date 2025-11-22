#!/bin/bash

# Phase 2 测试脚本 - 验证 command, shell, copy 模块

# Don't exit on error - we want to see all tests

INVENTORY="/workspace/tests/inventory/hosts.ini"
PATTERN="all"

echo "=========================================="
echo "Phase 2 Module Testing"
echo "=========================================="
echo ""

# 测试 1: command 模块
echo "Test 1: Command Module - uname"
echo "--------------------------------------"
echo "AnsiGo:"
ansigo -i $INVENTORY -m command -a "uname -a" $PATTERN
echo ""
echo "Ansible:"
ansible -i $INVENTORY -m command -a "uname -a" $PATTERN
echo ""

# 测试 2: command 模块 - 带工作目录
echo "Test 2: Command Module - with chdir"
echo "--------------------------------------"
echo "AnsiGo:"
ansigo -i $INVENTORY -m command -a "cmd=pwd chdir=/tmp" $PATTERN || true
echo ""
echo "Ansible:"
ansible -i $INVENTORY -m command -a "pwd chdir=/tmp" $PATTERN || true
echo ""

# 测试 3: shell 模块 - 简单命令
echo "Test 3: Shell Module - simple command"
echo "--------------------------------------"
echo "AnsiGo:"
ansigo -i $INVENTORY -m shell -a "hostname" $PATTERN || true
echo ""
echo "Ansible:"
ansible -i $INVENTORY -m shell -a "hostname" $PATTERN || true
echo ""

# 测试 4: shell 模块 - 带管道
echo "Test 4: Shell Module - with pipe"
echo "--------------------------------------"
echo "AnsiGo:"
ansigo -i $INVENTORY -m shell -a "echo test | wc -c" $PATTERN || true
echo ""
echo "Ansible:"
ansible -i $INVENTORY -m shell -a "echo test | wc -c" $PATTERN || true
echo ""

# 测试 5: shell 模块 - 环境变量
echo "Test 5: Shell Module - environment variable"
echo "--------------------------------------"
echo "AnsiGo:"
ansigo -i $INVENTORY -m shell -a "echo \$USER" $PATTERN || true
echo ""
echo "Ansible:"
ansible -i $INVENTORY -m shell -a "echo \$USER" $PATTERN || true
echo ""

# 测试 6: copy 模块 - content 参数
echo "Test 6: Copy Module - with content"
echo "--------------------------------------"
echo "AnsiGo:"
ansigo -i $INVENTORY -m copy -a "content='Hello from AnsiGo Phase 2' dest=/tmp/phase2-test.txt" $PATTERN || true
echo ""
echo "Verify content:"
ansigo -i $INVENTORY -m command -a "cat /tmp/phase2-test.txt" $PATTERN || true
echo ""
echo "Ansible:"
ansible -i $INVENTORY -m copy -a "content='Hello from Ansible Phase 2' dest=/tmp/ansible-phase2-test.txt" $PATTERN || true
echo ""
echo "Verify Ansible content:"
ansible -i $INVENTORY -m command -a "cat /tmp/ansible-phase2-test.txt" $PATTERN || true
echo ""

# 测试 7: copy 模块 - 多行内容
echo "Test 7: Copy Module - multiline content"
echo "--------------------------------------"
echo "AnsiGo:"
ansigo -i $INVENTORY -m copy -a "content='Line 1
Line 2
Line 3' dest=/tmp/multiline.txt" $PATTERN || true
echo ""
echo "Verify multiline:"
ansigo -i $INVENTORY -m command -a "cat /tmp/multiline.txt" $PATTERN || true
echo ""

echo "=========================================="
echo "Phase 2 Testing Complete!"
echo "=========================================="
