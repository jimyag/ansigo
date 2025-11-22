package playbook

import (
	"github.com/jimyag/ansigo/pkg/inventory"
)

// VariableManager 管理变量作用域和优先级
type VariableManager struct {
	inventory      *inventory.Manager
	playVars       map[string]interface{}
	registeredVars map[string]map[string]interface{} // hostname -> vars
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

// SetHostVar 设置主机变量（用于 register）
func (vm *VariableManager) SetHostVar(hostname, key string, value interface{}) {
	if vm.registeredVars[hostname] == nil {
		vm.registeredVars[hostname] = make(map[string]interface{})
	}
	vm.registeredVars[hostname][key] = value
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
	hosts, err := vm.inventory.GetHosts(hostname)
	if err == nil && len(hosts) > 0 {
		for k, v := range hosts[0].Vars {
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
	context["ansible_host"] = hostname

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
