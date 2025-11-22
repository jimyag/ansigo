package inventory

import (
	"fmt"
	"strings"
)

// Manager 是 Inventory 管理器
type Manager struct {
	inventory *Inventory
}

// NewManager 创建一个新的 Manager
func NewManager() *Manager {
	return &Manager{}
}

// Load 加载 inventory 文件
func (m *Manager) Load(path string) error {
	// 根据文件扩展名选择解析器
	var parser Parser
	if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
		// TODO: 实现 YAML 解析器
		return fmt.Errorf("YAML inventory not yet supported")
	} else {
		parser = NewINIParser()
	}

	inv, err := parser.Parse(path)
	if err != nil {
		return err
	}

	m.inventory = inv
	return nil
}

// GetHost 获取单个主机
func (m *Manager) GetHost(name string) (*Host, error) {
	host, exists := m.inventory.Hosts[name]
	if !exists {
		return nil, fmt.Errorf("host not found: %s", name)
	}
	return host, nil
}

// GetHosts 根据模式获取主机列表
func (m *Manager) GetHosts(pattern string) ([]*Host, error) {
	var hosts []*Host

	// 简单模式匹配
	switch pattern {
	case "all":
		// 返回所有主机
		for _, host := range m.inventory.Hosts {
			hosts = append(hosts, host)
		}
	default:
		// 假设是组名
		group, exists := m.inventory.Groups[pattern]
		if !exists {
			return nil, fmt.Errorf("group not found: %s", pattern)
		}

		// 收集组中的所有主机（包括子组）
		hostnames := m.collectGroupHosts(group)
		for _, hostname := range hostnames {
			if host, exists := m.inventory.Hosts[hostname]; exists {
				hosts = append(hosts, host)
			}
		}
	}

	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts matched pattern: %s", pattern)
	}

	return hosts, nil
}

// GetGroup 获取组
func (m *Manager) GetGroup(name string) (*Group, error) {
	group, exists := m.inventory.Groups[name]
	if !exists {
		return nil, fmt.Errorf("group not found: %s", name)
	}
	return group, nil
}

// collectGroupHosts 递归收集组中的所有主机
func (m *Manager) collectGroupHosts(group *Group) []string {
	hostnames := make([]string, 0)
	seen := make(map[string]bool)

	var collect func(*Group)
	collect = func(g *Group) {
		// 添加直接主机
		for _, hostname := range g.Hosts {
			if !seen[hostname] {
				hostnames = append(hostnames, hostname)
				seen[hostname] = true
			}
		}

		// 递归处理子组
		for _, childName := range g.Children {
			if child, exists := m.inventory.Groups[childName]; exists {
				collect(child)
			}
		}
	}

	collect(group)
	return hostnames
}
