package module

import (
	"fmt"
	"strings"

	"github.com/jimyag/ansigo/pkg/connection"
)

// SystemdModule systemd 模块实现
// systemd 模块用于管理 systemd 服务（更现代的 systemd 原生接口）
type SystemdModule struct{}

// Execute 执行 systemd 模块
func (m *SystemdModule) Execute(conn *connection.Connection, args map[string]interface{}, become bool, becomeUser, becomeMethod string) (*Result, error) {
	result := &Result{}

	// 获取必需参数 name
	nameInterface, ok := args["name"]
	if !ok {
		result.Failed = true
		result.Msg = "missing required argument: name"
		return result, nil
	}

	name, ok := nameInterface.(string)
	if !ok {
		result.Failed = true
		result.Msg = "name must be a string"
		return result, nil
	}

	// 获取 state 参数
	var state string
	if stateInterface, ok := args["state"]; ok {
		if s, ok := stateInterface.(string); ok {
			state = s
		}
	}

	// 获取 enabled 参数
	var enabled *bool
	if enabledInterface, ok := args["enabled"]; ok {
		switch v := enabledInterface.(type) {
		case bool:
			enabled = &v
		case string:
			val := v == "yes" || v == "true"
			enabled = &val
		}
	}

	// 获取 daemon_reload 参数
	daemonReload := false
	if reloadInterface, ok := args["daemon_reload"]; ok {
		switch v := reloadInterface.(type) {
		case bool:
			daemonReload = v
		case string:
			daemonReload = v == "yes" || v == "true"
		}
	}

	// 如果既没有 state 也没有 enabled 也没有 daemon_reload，报错
	if state == "" && enabled == nil && !daemonReload {
		result.Failed = true
		result.Msg = "one of 'state', 'enabled', or 'daemon_reload' is required"
		return result, nil
	}

	changed := false

	// 执行 daemon_reload（如果需要）
	if daemonReload {
		reloadChanged, err := m.reloadDaemon(conn, become, becomeUser, becomeMethod)
		if err != nil {
			result.Failed = true
			result.Msg = err.Error()
			return result, nil
		}
		if reloadChanged {
			changed = true
		}
	}

	// 处理 state
	if state != "" {
		stateChanged, err := m.manageState(conn, name, state, become, becomeUser, becomeMethod)
		if err != nil {
			result.Failed = true
			result.Msg = err.Error()
			return result, nil
		}
		if stateChanged {
			changed = true
		}
	}

	// 处理 enabled
	if enabled != nil {
		enabledChanged, err := m.manageEnabled(conn, name, *enabled, become, becomeUser, becomeMethod)
		if err != nil {
			result.Failed = true
			result.Msg = err.Error()
			return result, nil
		}
		if enabledChanged {
			changed = true
		}
	}

	result.Changed = changed
	if changed {
		result.Msg = fmt.Sprintf("systemd unit %s state changed", name)
	} else {
		result.Msg = fmt.Sprintf("systemd unit %s already in desired state", name)
	}

	return result, nil
}

// reloadDaemon 重新加载 systemd daemon
func (m *SystemdModule) reloadDaemon(conn *connection.Connection, become bool, becomeUser, becomeMethod string) (bool, error) {
	cmd := "systemctl daemon-reload"
	var stderr []byte
	var exitCode int
	var err error

	if become {
		_, stderr, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		_, stderr, exitCode, err = conn.Exec(cmd)
	}

	if err != nil || exitCode != 0 {
		return false, fmt.Errorf("failed to reload daemon: %s", strings.TrimSpace(string(stderr)))
	}

	// daemon-reload 总是标记为 changed
	return true, nil
}

// manageState 管理服务状态
func (m *SystemdModule) manageState(conn *connection.Connection, name string, state string, become bool, becomeUser, becomeMethod string) (bool, error) {
	// 获取当前服务状态
	isRunning, err := m.isServiceRunning(conn, name, become, becomeUser, becomeMethod)
	if err != nil {
		return false, fmt.Errorf("failed to check service status: %v", err)
	}

	var cmd string
	needChange := false

	switch state {
	case "started":
		if !isRunning {
			cmd = fmt.Sprintf("systemctl start %s", name)
			needChange = true
		}

	case "stopped":
		if isRunning {
			cmd = fmt.Sprintf("systemctl stop %s", name)
			needChange = true
		}

	case "restarted":
		cmd = fmt.Sprintf("systemctl restart %s", name)
		needChange = true

	case "reloaded":
		cmd = fmt.Sprintf("systemctl reload %s", name)
		needChange = true

	default:
		return false, fmt.Errorf("invalid state: %s (must be started/stopped/restarted/reloaded)", state)
	}

	if needChange {
		var stderr []byte
		var exitCode int
		var err error

		if become {
			_, stderr, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
		} else {
			_, stderr, exitCode, err = conn.Exec(cmd)
		}

		if err != nil || exitCode != 0 {
			return false, fmt.Errorf("failed to change service state: %s", strings.TrimSpace(string(stderr)))
		}
		return true, nil
	}

	return false, nil
}

// manageEnabled 管理服务开机自启
func (m *SystemdModule) manageEnabled(conn *connection.Connection, name string, enabled bool, become bool, becomeUser, becomeMethod string) (bool, error) {
	// 检查当前 enabled 状态
	isEnabled, err := m.isServiceEnabled(conn, name, become, becomeUser, becomeMethod)
	if err != nil {
		return false, fmt.Errorf("failed to check service enabled status: %v", err)
	}

	if isEnabled == enabled {
		// 已经是期望状态
		return false, nil
	}

	// 需要修改 enabled 状态
	var cmd string
	if enabled {
		cmd = fmt.Sprintf("systemctl enable %s", name)
	} else {
		cmd = fmt.Sprintf("systemctl disable %s", name)
	}

	var stderr []byte
	var exitCode int

	if become {
		_, stderr, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		_, stderr, exitCode, err = conn.Exec(cmd)
	}

	if err != nil || exitCode != 0 {
		return false, fmt.Errorf("failed to change service enabled status: %s", strings.TrimSpace(string(stderr)))
	}

	return true, nil
}

// isServiceRunning 检查服务是否正在运行
func (m *SystemdModule) isServiceRunning(conn *connection.Connection, name string, become bool, becomeUser, becomeMethod string) (bool, error) {
	cmd := fmt.Sprintf("systemctl is-active %s", name)

	var stdout []byte
	var exitCode int
	var err error

	if become {
		stdout, _, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		stdout, _, exitCode, err = conn.Exec(cmd)
	}

	if err != nil {
		return false, err
	}

	// systemctl is-active returns "active" if running
	return exitCode == 0 && strings.TrimSpace(string(stdout)) == "active", nil
}

// isServiceEnabled 检查服务是否开机自启
func (m *SystemdModule) isServiceEnabled(conn *connection.Connection, name string, become bool, becomeUser, becomeMethod string) (bool, error) {
	cmd := fmt.Sprintf("systemctl is-enabled %s", name)

	var stdout []byte
	var exitCode int
	var err error

	if become {
		stdout, _, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		stdout, _, exitCode, err = conn.Exec(cmd)
	}

	if err != nil {
		return false, err
	}

	// systemctl is-enabled returns "enabled" if enabled
	return exitCode == 0 && strings.TrimSpace(string(stdout)) == "enabled", nil
}
