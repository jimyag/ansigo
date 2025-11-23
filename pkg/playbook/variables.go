package playbook

import (
	"github.com/jimyag/ansigo/pkg/inventory"
)

// VariableManager 管理变量作用域和优先级
type VariableManager struct {
	inventory      *inventory.Manager
	playVars       map[string]interface{}
	registeredVars map[string]map[string]interface{} // hostname -> vars
	playHosts      []string                          // 当前 play 的主机列表
}

// NewVariableManager 创建变量管理器
func NewVariableManager(inv *inventory.Manager) *VariableManager {
	return &VariableManager{
		inventory:      inv,
		playVars:       make(map[string]interface{}),
		registeredVars: make(map[string]map[string]interface{}),
	}
}

// SetPlayVars 设置 Play 级别变量
func (vm *VariableManager) SetPlayVars(vars map[string]interface{}) {
	vm.playVars = vars
}

// SetPlayHosts 设置当前 play 的主机列表
func (vm *VariableManager) SetPlayHosts(hosts []string) {
	vm.playHosts = hosts
}

// SetHostVar 设置主机变量（用于 register）
func (vm *VariableManager) SetHostVar(hostname, key string, value interface{}) {
	if vm.registeredVars[hostname] == nil {
		vm.registeredVars[hostname] = make(map[string]interface{})
	}
	vm.registeredVars[hostname][key] = value
}

// SetHostVars 批量设置主机变量（用于 facts）
func (vm *VariableManager) SetHostVars(hostname string, vars map[string]interface{}) {
	if vm.registeredVars[hostname] == nil {
		vm.registeredVars[hostname] = make(map[string]interface{})
	}
	for k, v := range vars {
		vm.registeredVars[hostname][k] = v
	}
}

// GetHostVar 获取主机的特定变量
func (vm *VariableManager) GetHostVar(hostname, key string) (interface{}, bool) {
	// 先查找 registered 变量
	if hostVars, ok := vm.registeredVars[hostname]; ok {
		if value, exists := hostVars[key]; exists {
			return value, true
		}
	}

	// 查找 play 变量
	if value, ok := vm.playVars[key]; ok {
		return value, true
	}

	// 查找 inventory 变量
	hosts, err := vm.inventory.GetHosts(hostname)
	if err == nil && len(hosts) > 0 {
		if value, ok := hosts[0].Vars[key]; ok {
			return value, true
		}
	}

	return nil, false
}

// GetContext 获取主机的完整变量上下文
// 用于模板渲染
func (vm *VariableManager) GetContext(hostname string) map[string]interface{} {
	context := make(map[string]interface{})

	// 1. 合并 inventory 变量
	host, err := vm.inventory.GetHost(hostname)
	if err == nil {
		for k, v := range host.Vars {
			context[k] = v
		}
	}

	// 2. 合并 play 变量
	for k, v := range vm.playVars {
		context[k] = v
	}

	// 3. 合并 registered 变量（最高优先级）
	if hostVars, ok := vm.registeredVars[hostname]; ok {
		for k, v := range hostVars {
			context[k] = v
		}
	}

	// 4. 添加特殊变量
	context["inventory_hostname"] = hostname

	// 从 host vars 中获取 ansible_host，如果没有则使用 inventory_hostname
	if host != nil {
		if ansibleHost, ok := host.Vars["ansible_host"]; ok {
			context["ansible_host"] = ansibleHost
		} else {
			context["ansible_host"] = hostname
		}
	} else {
		context["ansible_host"] = hostname
	}

	// 5. 添加魔法变量

	// hostvars: 所有主机的变量
	context["hostvars"] = vm.buildHostvars()

	// groups: 所有组及其成员
	context["groups"] = vm.buildGroups()

	// group_names: 当前主机所在的组
	context["group_names"] = vm.getGroupNames(hostname)

	// ansible_play_hosts: 当前 play 的主机列表
	if len(vm.playHosts) > 0 {
		context["ansible_play_hosts"] = vm.playHosts
		context["ansible_play_batch"] = vm.playHosts // 简化实现，与 ansible_play_hosts 相同
	}

	return context
}

// GetAllHostVars 获取所有主机的变量（用于 hostvars）
func (vm *VariableManager) GetAllHostVars() map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})

	// 这里简化实现，实际应该遍历所有主机
	for hostname := range vm.registeredVars {
		result[hostname] = vm.GetContext(hostname)
	}

	return result
}

// ClearRegisteredVars 清除所有 registered 变量
// 通常在新的 Play 开始时调用
func (vm *VariableManager) ClearRegisteredVars() {
	vm.registeredVars = make(map[string]map[string]interface{})
}

// buildHostvars 构建 hostvars 魔法变量
// 返回所有主机的变量字典
func (vm *VariableManager) buildHostvars() map[string]interface{} {
	hostvars := make(map[string]interface{})

	// 获取所有主机
	allHosts, err := vm.inventory.GetHosts("all")
	if err != nil {
		return hostvars
	}

	// 为每个主机构建变量上下文
	for _, host := range allHosts {
		hostContext := make(map[string]interface{})

		// 添加 inventory 变量
		for k, v := range host.Vars {
			hostContext[k] = v
		}

		// 添加 play 变量
		for k, v := range vm.playVars {
			hostContext[k] = v
		}

		// 添加 registered 变量
		if hostVars, ok := vm.registeredVars[host.Name]; ok {
			for k, v := range hostVars {
				hostContext[k] = v
			}
		}

		// 添加基本魔法变量
		hostContext["inventory_hostname"] = host.Name
		if ansibleHost, ok := host.Vars["ansible_host"]; ok {
			hostContext["ansible_host"] = ansibleHost
		} else {
			hostContext["ansible_host"] = host.Name
		}

		hostvars[host.Name] = hostContext
	}

	return hostvars
}

// buildGroups 构建 groups 魔法变量
// 返回所有组及其成员列表
func (vm *VariableManager) buildGroups() map[string]interface{} {
	groups := make(map[string]interface{})

	// 获取所有主机以访问 inventory
	allHosts, err := vm.inventory.GetHosts("all")
	if err != nil || len(allHosts) == 0 {
		return groups
	}

	// 遍历所有组
	// 通过访问 inventory 的内部结构
	// 注意: 这需要 inventory Manager 提供访问组的方法
	// 简化实现: 从主机的 Groups 字段收集
	groupMembers := make(map[string][]string)
	for _, host := range allHosts {
		for _, groupName := range host.Groups {
			if groupMembers[groupName] == nil {
				groupMembers[groupName] = []string{}
			}
			groupMembers[groupName] = append(groupMembers[groupName], host.Name)
		}
	}

	// 添加 "all" 组
	allHostNames := make([]string, len(allHosts))
	for i, host := range allHosts {
		allHostNames[i] = host.Name
	}
	groupMembers["all"] = allHostNames

	// 转换为 interface{} 类型
	for groupName, members := range groupMembers {
		groups[groupName] = members
	}

	return groups
}

// getGroupNames 获取主机所属的所有组名
func (vm *VariableManager) getGroupNames(hostname string) []string {
	host, err := vm.inventory.GetHost(hostname)
	if err != nil {
		return []string{}
	}

	return host.Groups
}
