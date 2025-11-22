package inventory

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/jimyag/ansigo/pkg/errors"
)

// Parser 是 Inventory 解析器接口
type Parser interface {
	Parse(filePath string) (*Inventory, error)
}

// INIParser 解析 INI 格式的 inventory
type INIParser struct{}

// NewINIParser 创建一个新的 INI 解析器
func NewINIParser() *INIParser {
	return &INIParser{}
}

// Parse 解析 INI 格式的 inventory 文件
func (p *INIParser) Parse(filePath string) (*Inventory, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read inventory file: %w", err)
	}

	inv := NewInventory()
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	currentSection := ""
	currentGroup := ""

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// 解析 section header [groupname] 或 [groupname:vars] 或 [groupname:children]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := line[1 : len(line)-1]

			if strings.Contains(section, ":vars") {
				currentGroup = strings.TrimSuffix(section, ":vars")
				currentSection = "vars"
			} else if strings.Contains(section, ":children") {
				currentGroup = strings.TrimSuffix(section, ":children")
				currentSection = "children"
			} else {
				currentGroup = section
				currentSection = "hosts"
			}

			// 确保组存在
			if _, exists := inv.Groups[currentGroup]; !exists {
				inv.Groups[currentGroup] = &Group{
					Name:     currentGroup,
					Hosts:    []string{},
					Children: []string{},
					Vars:     make(map[string]interface{}),
					Parents:  []string{},
				}
			}

			continue
		}

		// 解析内容
		if err := p.parseLine(inv, line, currentSection, currentGroup, filePath, lineNum); err != nil {
			return nil, err
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.NewParseError(filePath, err)
	}

	// 后处理：建立层级关系和合并变量
	if err := p.postProcess(inv); err != nil {
		return nil, err
	}

	return inv, nil
}

// parseLine 解析单行内容
func (p *INIParser) parseLine(inv *Inventory, line, section, group, filePath string, lineNum int) error {
	switch section {
	case "hosts":
		return p.parseHost(inv, line, group)
	case "vars":
		return p.parseGroupVar(inv, line, group)
	case "children":
		return p.parseChild(inv, line, group)
	default:
		// 默认作为主机处理（文件开头的主机）
		return p.parseHost(inv, line, "ungrouped")
	}
}

// parseHost 解析主机行
func (p *INIParser) parseHost(inv *Inventory, line, group string) error {
	// 格式: hostname [key=value key=value ...]
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	hostname := parts[0]
	vars := make(map[string]interface{})

	// 解析行内变量
	for _, part := range parts[1:] {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			vars[kv[0]] = kv[1]
		}
	}

	// 创建或更新主机
	host, exists := inv.Hosts[hostname]
	if !exists {
		host = &Host{
			Name:   hostname,
			Vars:   vars,
			Groups: []string{},
		}
		inv.Hosts[hostname] = host
	} else {
		// 合并变量
		for k, v := range vars {
			host.Vars[k] = v
		}
	}

	// 添加到组
	if group != "" {
		if !contains(host.Groups, group) {
			host.Groups = append(host.Groups, group)
		}
		if !contains(inv.Groups[group].Hosts, hostname) {
			inv.Groups[group].Hosts = append(inv.Groups[group].Hosts, hostname)
		}
	}

	// 添加到 all 组
	if !contains(inv.Groups["all"].Hosts, hostname) {
		inv.Groups["all"].Hosts = append(inv.Groups["all"].Hosts, hostname)
	}

	return nil
}

// parseGroupVar 解析组变量
func (p *INIParser) parseGroupVar(inv *Inventory, line, group string) error {
	kv := strings.SplitN(line, "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("invalid variable line: %s", line)
	}

	key := strings.TrimSpace(kv[0])
	value := strings.TrimSpace(kv[1])

	if g, exists := inv.Groups[group]; exists {
		g.Vars[key] = value
	}

	return nil
}

// parseChild 解析子组
func (p *INIParser) parseChild(inv *Inventory, line, group string) error {
	childName := strings.TrimSpace(line)

	// 确保子组存在
	if _, exists := inv.Groups[childName]; !exists {
		inv.Groups[childName] = &Group{
			Name:     childName,
			Hosts:    []string{},
			Children: []string{},
			Vars:     make(map[string]interface{}),
			Parents:  []string{},
		}
	}

	// 建立父子关系
	if g, exists := inv.Groups[group]; exists {
		if !contains(g.Children, childName) {
			g.Children = append(g.Children, childName)
		}
	}

	if c, exists := inv.Groups[childName]; exists {
		if !contains(c.Parents, group) {
			c.Parents = append(c.Parents, group)
		}
	}

	return nil
}

// postProcess 后处理：合并变量到主机
func (p *INIParser) postProcess(inv *Inventory) error {
	// 为每个主机合并变量（按优先级）
	for _, host := range inv.Hosts {
		mergedVars := p.mergeHostVars(inv, host)
		host.Vars = mergedVars
	}

	return nil
}

// mergeHostVars 合并主机的所有变量
func (p *INIParser) mergeHostVars(inv *Inventory, host *Host) map[string]interface{} {
	result := make(map[string]interface{})

	// 1. all 组变量（最低优先级）
	if allGroup, exists := inv.Groups["all"]; exists {
		for k, v := range allGroup.Vars {
			result[k] = v
		}
	}

	// 2. 父组变量
	// 3. 子组变量
	// 简化版：直接合并所有组变量
	for _, groupName := range host.Groups {
		if group, exists := inv.Groups[groupName]; exists {
			for k, v := range group.Vars {
				result[k] = v
			}
		}
	}

	// 4. 主机变量（最高优先级）
	for k, v := range host.Vars {
		result[k] = v
	}

	return result
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
