package module

import (
	"fmt"
	"strings"

	"github.com/jimyag/ansigo/pkg/connection"
)

// ServiceModule service 模块实现
// service 模块用于管理系统服务
type ServiceModule struct{}

// Execute 执行 service 模块
func (m *ServiceModule) Execute(conn *connection.Connection, args map[string]interface{}, become bool, becomeUser, becomeMethod string) (*Result, error) {
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

	// 检测使用 systemd 还是 service 命令
	useSystemd := m.detectSystemd(conn)

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

	// 如果既没有 state 也没有 enabled，报错
	if state == "" && enabled == nil {
		result.Failed = true
		result.Msg = "one of 'state' or 'enabled' is required"
		return result, nil
	}

	changed := false

	// 处理 state
	if state != "" {
		stateChanged, err := m.manageState(conn, name, state, useSystemd)
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
		enabledChanged, err := m.manageEnabled(conn, name, *enabled, useSystemd)
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
		result.Msg = fmt.Sprintf("service %s state changed", name)
	} else {
		result.Msg = fmt.Sprintf("service %s already in desired state", name)
	}

	return result, nil
}

// detectSystemd 检测系统是否使用 systemd
func (m *ServiceModule) detectSystemd(conn *connection.Connection) bool {
	// 检查 systemctl 命令是否存在
	checkCmd := "command -v systemctl"
	checkResult, err := executeCommand(conn, checkCmd)
	return err == nil && checkResult.RC == 0
}

// manageState 管理服务状态
func (m *ServiceModule) manageState(conn *connection.Connection, name string, state string, useSystemd bool) (bool, error) {
	// 获取当前服务状态
	isRunning, err := m.isServiceRunning(conn, name, useSystemd)
	if err != nil {
		return false, fmt.Errorf("failed to check service status: %v", err)
	}

	var cmd string
	needChange := false

	switch state {
	case "started":
		if !isRunning {
			if useSystemd {
				cmd = fmt.Sprintf("systemctl start %s", name)
			} else {
				cmd = fmt.Sprintf("service %s start", name)
			}
			needChange = true
		}

	case "stopped":
		if isRunning {
			if useSystemd {
				cmd = fmt.Sprintf("systemctl stop %s", name)
			} else {
				cmd = fmt.Sprintf("service %s stop", name)
			}
			needChange = true
		}

	case "restarted":
		if useSystemd {
			cmd = fmt.Sprintf("systemctl restart %s", name)
		} else {
			cmd = fmt.Sprintf("service %s restart", name)
		}
		needChange = true

	case "reloaded":
		if useSystemd {
			cmd = fmt.Sprintf("systemctl reload %s", name)
		} else {
			cmd = fmt.Sprintf("service %s reload", name)
		}
		needChange = true

	default:
		return false, fmt.Errorf("invalid state: %s (must be started/stopped/restarted/reloaded)", state)
	}

	if needChange {
		result, err := executeCommand(conn, cmd)
		if err != nil || result.RC != 0 {
			return false, fmt.Errorf("failed to change service state: %s", result.Stderr)
		}
		return true, nil
	}

	return false, nil
}

// manageEnabled 管理服务开机自启
func (m *ServiceModule) manageEnabled(conn *connection.Connection, name string, enabled bool, useSystemd bool) (bool, error) {
	// 检查当前 enabled 状态
	isEnabled, err := m.isServiceEnabled(conn, name, useSystemd)
	if err != nil {
		return false, fmt.Errorf("failed to check service enabled status: %v", err)
	}

	if isEnabled == enabled {
		// 已经是期望状态
		return false, nil
	}

	// 需要修改 enabled 状态
	var cmd string
	if useSystemd {
		if enabled {
			cmd = fmt.Sprintf("systemctl enable %s", name)
		} else {
			cmd = fmt.Sprintf("systemctl disable %s", name)
		}
	} else {
		// 使用 update-rc.d (Debian/Ubuntu) 或 chkconfig (RHEL/CentOS)
		// 先尝试 update-rc.d
		checkUpdateRc := "command -v update-rc.d"
		checkResult, _ := executeCommand(conn, checkUpdateRc)
		if checkResult != nil && checkResult.RC == 0 {
			if enabled {
				cmd = fmt.Sprintf("update-rc.d %s defaults", name)
			} else {
				cmd = fmt.Sprintf("update-rc.d -f %s remove", name)
			}
		} else {
			// 尝试 chkconfig
			if enabled {
				cmd = fmt.Sprintf("chkconfig %s on", name)
			} else {
				cmd = fmt.Sprintf("chkconfig %s off", name)
			}
		}
	}

	result, err := executeCommand(conn, cmd)
	if err != nil || result.RC != 0 {
		return false, fmt.Errorf("failed to change service enabled status: %s", result.Stderr)
	}

	return true, nil
}

// isServiceRunning 检查服务是否正在运行
func (m *ServiceModule) isServiceRunning(conn *connection.Connection, name string, useSystemd bool) (bool, error) {
	var cmd string
	if useSystemd {
		cmd = fmt.Sprintf("systemctl is-active %s", name)
	} else {
		cmd = fmt.Sprintf("service %s status", name)
	}

	result, err := executeCommand(conn, cmd)
	if err != nil {
		return false, err
	}

	if useSystemd {
		// systemctl is-active returns "active" if running
		return result.RC == 0 && strings.TrimSpace(result.Stdout) == "active", nil
	}

	// service status returns 0 if running
	return result.RC == 0, nil
}

// isServiceEnabled 检查服务是否开机自启
func (m *ServiceModule) isServiceEnabled(conn *connection.Connection, name string, useSystemd bool) (bool, error) {
	var cmd string
	if useSystemd {
		cmd = fmt.Sprintf("systemctl is-enabled %s", name)
	} else {
		// 尝试使用 chkconfig 或检查 /etc/rc*.d/
		checkChkconfig := "command -v chkconfig"
		checkResult, _ := executeCommand(conn, checkChkconfig)
		if checkResult != nil && checkResult.RC == 0 {
			cmd = fmt.Sprintf("chkconfig --list %s", name)
		} else {
			// 检查 /etc/rc*.d/ 中是否有启动脚本
			cmd = fmt.Sprintf("ls /etc/rc*.d/S*%s 2>/dev/null | wc -l", name)
		}
	}

	result, err := executeCommand(conn, cmd)
	if err != nil {
		return false, err
	}

	if useSystemd {
		// systemctl is-enabled returns "enabled" if enabled
		return result.RC == 0 && strings.TrimSpace(result.Stdout) == "enabled", nil
	}

	// 对于非 systemd 系统，如果命令成功且输出包含启用信息，则认为已启用
	return result.RC == 0 && result.Stdout != "0", nil
}
