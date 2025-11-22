package playbook

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TaskIncluder 处理任务包含（import_tasks, include_role）
type TaskIncluder struct {
	playbookPath string
	roleLoader   *RoleLoader
}

// NewTaskIncluder 创建任务包含处理器
func NewTaskIncluder(playbookPath string) *TaskIncluder {
	return &TaskIncluder{
		playbookPath: playbookPath,
		roleLoader:   NewRoleLoader(playbookPath),
	}
}

// ExpandTask 展开任务（处理 import_tasks 和 include_role）
// 返回展开后的任务列表
func (ti *TaskIncluder) ExpandTask(task *Task, vars map[string]interface{}) ([]Task, error) {
	// 规范化模块名（移除 ansible.builtin. 前缀）
	moduleName := task.Module
	if moduleName == "ansible.builtin.import_tasks" {
		moduleName = "import_tasks"
	} else if moduleName == "ansible.builtin.include_role" {
		moduleName = "include_role"
	}

	switch moduleName {
	case "import_tasks":
		return ti.expandImportTasks(task, vars)
	case "include_role":
		return ti.expandIncludeRole(task, vars)
	default:
		// 不是包含任务，返回原任务
		return []Task{*task}, nil
	}
}

// expandImportTasks 展开 import_tasks
func (ti *TaskIncluder) expandImportTasks(task *Task, vars map[string]interface{}) ([]Task, error) {
	// 获取要导入的文件路径
	var tasksFile string

	if file, ok := task.ModuleArgs["file"].(string); ok {
		tasksFile = file
	} else if rawParams, ok := task.ModuleArgs["_raw_params"].(string); ok {
		tasksFile = rawParams
	} else {
		return nil, fmt.Errorf("import_tasks requires 'file' parameter")
	}

	// 解析文件路径（相对于 playbook 目录）
	playbookDir := filepath.Dir(ti.playbookPath)
	fullPath := filepath.Join(playbookDir, tasksFile)

	// 尝试 .yaml 和 .yml 扩展名
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		if filepath.Ext(fullPath) == "" {
			// 没有扩展名，尝试添加
			if _, err := os.Stat(fullPath + ".yaml"); err == nil {
				fullPath = fullPath + ".yaml"
			} else if _, err := os.Stat(fullPath + ".yml"); err == nil {
				fullPath = fullPath + ".yml"
			}
		}
	}

	// 读取任务文件
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file %s: %w", tasksFile, err)
	}

	// 解析任务列表
	var tasks []Task
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks file %s: %w", tasksFile, err)
	}

	return tasks, nil
}

// expandIncludeRole 展开 include_role
func (ti *TaskIncluder) expandIncludeRole(task *Task, vars map[string]interface{}) ([]Task, error) {
	// 获取 role 名称
	roleName, ok := task.ModuleArgs["name"].(string)
	if !ok {
		return nil, fmt.Errorf("include_role requires 'name' parameter")
	}

	// 获取可选的 tasks_from 参数
	tasksFrom, _ := task.ModuleArgs["tasks_from"].(string)

	// 构建 RoleSpec
	spec := RoleSpec{
		Name: roleName,
		Vars: make(map[string]interface{}),
	}

	// 提取传递给 role 的变量（从 vars 参数）
	if roleVars, ok := task.ModuleArgs["vars"].(map[string]interface{}); ok {
		spec.Vars = roleVars
	}

	// 如果有 tasks_from，只加载特定的任务文件
	if tasksFrom != "" {
		return ti.loadRoleTasksFrom(roleName, tasksFrom, spec.Vars)
	}

	// 否则加载整个 role
	role, err := ti.roleLoader.LoadRole(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to load role '%s': %w", roleName, err)
	}

	return role.Tasks, nil
}

// loadRoleTasksFrom 加载 role 的特定任务文件
// 注意: 当使用 tasks_from 时，role 的 defaults 和 vars 不会自动加载
// 需要在 playbook 中明确传递变量
func (ti *TaskIncluder) loadRoleTasksFrom(roleName, tasksFrom string, vars map[string]interface{}) ([]Task, error) {
	// 查找 role 目录
	rolePath, err := ti.roleLoader.findRolePath(roleName)
	if err != nil {
		return nil, err
	}

	// 构建任务文件路径
	tasksFile := filepath.Join(rolePath, "tasks", tasksFrom)

	// 尝试 .yaml 和 .yml 扩展名
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		if filepath.Ext(tasksFile) == "" {
			if _, err := os.Stat(tasksFile + ".yaml"); err == nil {
				tasksFile = tasksFile + ".yaml"
			} else if _, err := os.Stat(tasksFile + ".yml"); err == nil {
				tasksFile = tasksFile + ".yml"
			}
		}
	}

	// 读取任务文件
	data, err := os.ReadFile(tasksFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file %s from role %s: %w", tasksFrom, roleName, err)
	}

	// 解析任务列表
	var tasks []Task
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks file %s: %w", tasksFrom, err)
	}

	return tasks, nil
}
