package playbook

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RoleLoader 负责加载和解析 Roles
type RoleLoader struct {
	playbookDir string   // Playbook 文件所在目录
	rolePaths   []string // Role 搜索路径
}

// NewRoleLoader 创建 Role 加载器
func NewRoleLoader(playbookPath string) *RoleLoader {
	playbookDir := filepath.Dir(playbookPath)

	return &RoleLoader{
		playbookDir: playbookDir,
		rolePaths: []string{
			filepath.Join(playbookDir, "roles"), // playbook 目录下的 roles/
			"./roles",                           // 当前目录的 roles/
		},
	}
}

// LoadRole 加载指定的 Role
func (rl *RoleLoader) LoadRole(spec RoleSpec) (*Role, error) {
	// 查找 role 目录
	rolePath, err := rl.findRolePath(spec.Name)
	if err != nil {
		return nil, err
	}

	role := &Role{
		Name:     spec.Name,
		Path:     rolePath,
		Vars:     make(map[string]interface{}),
		Defaults: make(map[string]interface{}),
	}

	// 加载 defaults/main.yaml
	if err := rl.loadRoleDefaults(role); err != nil {
		// defaults 是可选的，不存在不报错
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load role defaults: %w", err)
		}
	}

	// 加载 vars/main.yaml
	if err := rl.loadRoleVars(role); err != nil {
		// vars 是可选的，不存在不报错
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load role vars: %w", err)
		}
	}

	// 合并 RoleSpec 中的变量（最高优先级）
	for k, v := range spec.Vars {
		role.Vars[k] = v
	}

	// 加载 tasks/main.yaml
	if err := rl.loadRoleTasks(role); err != nil {
		return nil, fmt.Errorf("failed to load role tasks: %w", err)
	}

	// 加载 handlers/main.yaml
	if err := rl.loadRoleHandlers(role); err != nil {
		// handlers 是可选的，不存在不报错
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load role handlers: %w", err)
		}
	}

	return role, nil
}

// findRolePath 查找 role 目录
func (rl *RoleLoader) findRolePath(roleName string) (string, error) {
	for _, basePath := range rl.rolePaths {
		rolePath := filepath.Join(basePath, roleName)
		if info, err := os.Stat(rolePath); err == nil && info.IsDir() {
			return rolePath, nil
		}
	}
	return "", fmt.Errorf("role not found: %s (searched: %v)", roleName, rl.rolePaths)
}

// loadRoleDefaults 加载 role 的 defaults 变量
func (rl *RoleLoader) loadRoleDefaults(role *Role) error {
	defaultsFile := filepath.Join(role.Path, "defaults", "main.yaml")

	// 尝试 .yaml 和 .yml 两种扩展名
	if _, err := os.Stat(defaultsFile); os.IsNotExist(err) {
		defaultsFile = filepath.Join(role.Path, "defaults", "main.yml")
	}

	data, err := os.ReadFile(defaultsFile)
	if err != nil {
		return err
	}

	var defaults map[string]interface{}
	if err := yaml.Unmarshal(data, &defaults); err != nil {
		return fmt.Errorf("failed to parse defaults file: %w", err)
	}

	role.Defaults = defaults
	return nil
}

// loadRoleVars 加载 role 的 vars 变量
func (rl *RoleLoader) loadRoleVars(role *Role) error {
	varsFile := filepath.Join(role.Path, "vars", "main.yaml")

	// 尝试 .yaml 和 .yml 两种扩展名
	if _, err := os.Stat(varsFile); os.IsNotExist(err) {
		varsFile = filepath.Join(role.Path, "vars", "main.yml")
	}

	data, err := os.ReadFile(varsFile)
	if err != nil {
		return err
	}

	var vars map[string]interface{}
	if err := yaml.Unmarshal(data, &vars); err != nil {
		return fmt.Errorf("failed to parse vars file: %w", err)
	}

	// 合并到 role.Vars
	for k, v := range vars {
		role.Vars[k] = v
	}

	return nil
}

// loadRoleTasks 加载 role 的 tasks
func (rl *RoleLoader) loadRoleTasks(role *Role) error {
	tasksFile := filepath.Join(role.Path, "tasks", "main.yaml")

	// 尝试 .yaml 和 .yml 两种扩展名
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		tasksFile = filepath.Join(role.Path, "tasks", "main.yml")
	}

	data, err := os.ReadFile(tasksFile)
	if err != nil {
		return err
	}

	var tasks []Task
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("failed to parse tasks file: %w", err)
	}

	role.Tasks = tasks
	return nil
}

// loadRoleHandlers 加载 role 的 handlers
func (rl *RoleLoader) loadRoleHandlers(role *Role) error {
	handlersFile := filepath.Join(role.Path, "handlers", "main.yaml")

	// 尝试 .yaml 和 .yml 两种扩展名
	if _, err := os.Stat(handlersFile); os.IsNotExist(err) {
		handlersFile = filepath.Join(role.Path, "handlers", "main.yml")
	}

	data, err := os.ReadFile(handlersFile)
	if err != nil {
		return err
	}

	var handlers []Handler
	if err := yaml.Unmarshal(data, &handlers); err != nil {
		return fmt.Errorf("failed to parse handlers file: %w", err)
	}

	role.Handlers = handlers
	return nil
}

// ParseRoleSpec 解析 role 规格（支持字符串或字典）
func ParseRoleSpec(roleData interface{}) (RoleSpec, error) {
	spec := RoleSpec{
		Vars: make(map[string]interface{}),
	}

	switch v := roleData.(type) {
	case string:
		// 简单格式: roles: [common, nginx]
		spec.Name = v
	case map[string]interface{}:
		// 字典格式: roles: [{role: common, vars: {...}}]
		if name, ok := v["role"].(string); ok {
			spec.Name = name
		} else if name, ok := v["name"].(string); ok {
			spec.Name = name
		} else {
			return spec, fmt.Errorf("role spec must have 'role' or 'name' field")
		}

		// 提取其他字段作为变量
		for k, val := range v {
			if k != "role" && k != "name" {
				spec.Vars[k] = val
			}
		}
	default:
		return spec, fmt.Errorf("unsupported role format: %T", roleData)
	}

	if spec.Name == "" {
		return spec, fmt.Errorf("role name cannot be empty")
	}

	return spec, nil
}
