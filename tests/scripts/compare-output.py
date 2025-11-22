#!/usr/bin/env python3
"""
对比 Ansible 和 AnsiGo 的输出

忽略动态字段（时间戳、路径等），对比核心功能是否一致
"""

import sys
import re
import json


def normalize_output(content):
    """
    规范化输出内容，提取关键信息
    """
    hosts = {}

    # 解析每个主机的结果
    # 格式: hostname | SUCCESS => {...} 或 hostname | FAILED => {...}
    pattern = r'(\S+)\s+\|\s+(\w+)\s+=>\s+(\{.+?\})'

    for match in re.finditer(pattern, content, re.DOTALL):
        hostname = match.group(1)
        status = match.group(2)
        json_data = match.group(3)

        try:
            data = json.loads(json_data)
            # 移除动态字段
            dynamic_fields = [
                '_ansible_parsed', '_ansible_no_log', 'invocation',
                'delta', 'start', 'end', 'stderr_lines', 'stdout_lines'
            ]
            for field in dynamic_fields:
                data.pop(field, None)

            hosts[hostname] = {
                'status': status,
                'data': data
            }
        except json.JSONDecodeError as e:
            print(f"Warning: Failed to parse JSON for {hostname}: {e}", file=sys.stderr)
            hosts[hostname] = {
                'status': status,
                'data': {'raw': json_data}
            }

    return hosts


def compare_hosts(ansible_hosts, ansigo_hosts):
    """
    对比两组主机结果
    """
    all_hosts = set(ansible_hosts.keys()) | set(ansigo_hosts.keys())

    differences = []

    for host in sorted(all_hosts):
        if host not in ansible_hosts:
            differences.append(f"  ✗ {host}: Missing in Ansible output")
            continue

        if host not in ansigo_hosts:
            differences.append(f"  ✗ {host}: Missing in AnsiGo output")
            continue

        ansible_result = ansible_hosts[host]
        ansigo_result = ansigo_hosts[host]

        # 对比状态
        if ansible_result['status'] != ansigo_result['status']:
            differences.append(
                f"  ✗ {host}: Status mismatch "
                f"(Ansible: {ansible_result['status']}, AnsiGo: {ansigo_result['status']})"
            )
            continue

        # 对比关键字段
        ansible_data = ansible_result['data']
        ansigo_data = ansigo_result['data']

        # 检查关键字段
        key_fields = ['changed', 'failed', 'ping', 'rc', 'msg']
        for field in key_fields:
            if field in ansible_data:
                if field not in ansigo_data:
                    differences.append(f"  ✗ {host}: Missing field '{field}' in AnsiGo")
                elif ansible_data[field] != ansigo_data[field]:
                    differences.append(
                        f"  ✗ {host}: Field '{field}' mismatch "
                        f"(Ansible: {ansible_data[field]}, AnsiGo: {ansigo_data[field]})"
                    )

    if differences:
        print("❌ Differences found:")
        for diff in differences:
            print(diff)
        return False
    else:
        print("✅ Outputs match!")
        return True


def main():
    if len(sys.argv) != 3:
        print("Usage: compare-output.py <ansible-output> <ansigo-output>")
        sys.exit(1)

    ansible_file = sys.argv[1]
    ansigo_file = sys.argv[2]

    try:
        with open(ansible_file, 'r') as f:
            ansible_content = f.read()

        with open(ansigo_file, 'r') as f:
            ansigo_content = f.read()
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    ansible_hosts = normalize_output(ansible_content)
    ansigo_hosts = normalize_output(ansigo_content)

    if not ansible_hosts:
        print("Warning: No hosts found in Ansible output", file=sys.stderr)

    if not ansigo_hosts:
        print("Warning: No hosts found in AnsiGo output", file=sys.stderr)

    success = compare_hosts(ansible_hosts, ansigo_hosts)

    sys.exit(0 if success else 1)


if __name__ == '__main__':
    main()
