package module

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/jimyag/ansigo/pkg/connection"
)

// FileModule file 模块实现
type FileModule struct{}

// execResult 包装执行结果
type execResult struct {
	RC     int
	Stdout string
	Stderr string
}

// executeCommand 执行命令并返回包装后的结果
func executeCommand(conn *connection.Connection, cmd string) (*execResult, error) {
	stdout, stderr, exitCode, err := conn.Exec(cmd)
	if err != nil {
		return nil, err
	}
	return &execResult{
		RC:     exitCode,
		Stdout: strings.TrimSpace(string(stdout)),
		Stderr: strings.TrimSpace(string(stderr)),
	}, nil
}

// Execute 执行 file 模块
func (m *FileModule) Execute(conn *connection.Connection, args map[string]interface{}) (*Result, error) {
	result := &Result{}

	// 获取必需参数 path
	pathInterface, ok := args["path"]
	if !ok {
		result.Failed = true
		result.Msg = "missing required argument: path"
		return result, nil
	}

	path, ok := pathInterface.(string)
	if !ok {
		result.Failed = true
		result.Msg = "path must be a string"
		return result, nil
	}

	// 获取 state 参数（默认为 file）
	state := "file"
	if stateInterface, ok := args["state"]; ok {
		if s, ok := stateInterface.(string); ok {
			state = s
		}
	}

	// 根据 state 执行不同操作
	switch state {
	case "file":
		return m.ensureFile(conn, path, args)
	case "directory":
		return m.ensureDirectory(conn, path, args)
	case "absent":
		return m.ensureAbsent(conn, path)
	case "touch":
		return m.touchFile(conn, path, args)
	case "link":
		return m.createLink(conn, path, args)
	default:
		result.Failed = true
		result.Msg = fmt.Sprintf("invalid state: %s", state)
		return result, nil
	}
}

// ensureFile 确保文件存在
func (m *FileModule) ensureFile(conn *connection.Connection, path string, args map[string]interface{}) (*Result, error) {
	result := &Result{}

	// 检查文件是否存在
	checkCmd := fmt.Sprintf("test -f %s", path)
	checkResult, err := executeCommand(conn, checkCmd)
	if err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to check file: %v", err)
		return result, nil
	}

	exists := checkResult.RC == 0

	if !exists {
		result.Failed = true
		result.Msg = fmt.Sprintf("file not found: %s (use state=touch to create)", path)
		return result, nil
	}

	// 文件存在，应用权限/所有者等
	changed, err := m.applyPermissions(conn, path, args)
	if err != nil {
		result.Failed = true
		result.Msg = err.Error()
		return result, nil
	}

	result.Changed = changed
	result.Msg = fmt.Sprintf("file %s is present", path)
	return result, nil
}

// ensureDirectory 确保目录存在
func (m *FileModule) ensureDirectory(conn *connection.Connection, path string, args map[string]interface{}) (*Result, error) {
	result := &Result{}

	// 检查目录是否存在
	checkCmd := fmt.Sprintf("test -d %s", path)
	checkResult, err := executeCommand(conn, checkCmd)
	if err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to check directory: %v", err)
		return result, nil
	}

	exists := checkResult.RC == 0

	if !exists {
		// 创建目录
		mkdirCmd := fmt.Sprintf("mkdir -p %s", path)
		mkdirResult, err := executeCommand(conn, mkdirCmd)
		if err != nil || mkdirResult.RC != 0 {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to create directory: %s", mkdirResult.Stderr)
			return result, nil
		}
		result.Changed = true
	}

	// 应用权限/所有者等
	permChanged, err := m.applyPermissions(conn, path, args)
	if err != nil {
		result.Failed = true
		result.Msg = err.Error()
		return result, nil
	}

	if permChanged {
		result.Changed = true
	}

	if result.Changed {
		result.Msg = fmt.Sprintf("directory %s created", path)
	} else {
		result.Msg = fmt.Sprintf("directory %s already exists", path)
	}

	return result, nil
}

// ensureAbsent 确保文件/目录不存在
func (m *FileModule) ensureAbsent(conn *connection.Connection, path string) (*Result, error) {
	result := &Result{}

	// 检查路径是否存在
	checkCmd := fmt.Sprintf("test -e %s", path)
	checkResult, err := executeCommand(conn, checkCmd)
	if err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to check path: %v", err)
		return result, nil
	}

	if checkResult.RC != 0 {
		// 路径不存在
		result.Changed = false
		result.Msg = fmt.Sprintf("path %s does not exist", path)
		return result, nil
	}

	// 删除文件或目录
	rmCmd := fmt.Sprintf("rm -rf %s", path)
	rmResult, err := executeCommand(conn, rmCmd)
	if err != nil || rmResult.RC != 0 {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to remove: %s", rmResult.Stderr)
		return result, nil
	}

	result.Changed = true
	result.Msg = fmt.Sprintf("removed %s", path)
	return result, nil
}

// touchFile 创建空文件或更新时间戳
func (m *FileModule) touchFile(conn *connection.Connection, path string, args map[string]interface{}) (*Result, error) {
	result := &Result{}

	// 检查文件是否存在
	checkCmd := fmt.Sprintf("test -e %s", path)
	checkResult, err := executeCommand(conn, checkCmd)
	if err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to check file: %v", err)
		return result, nil
	}

	exists := checkResult.RC == 0

	// 执行 touch 命令
	touchCmd := fmt.Sprintf("touch %s", path)
	touchResult, err := executeCommand(conn, touchCmd)
	if err != nil || touchResult.RC != 0 {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to touch file: %s", touchResult.Stderr)
		return result, nil
	}

	// 如果文件之前不存在，标记为 changed
	result.Changed = !exists

	// 应用权限/所有者等
	permChanged, err := m.applyPermissions(conn, path, args)
	if err != nil {
		result.Failed = true
		result.Msg = err.Error()
		return result, nil
	}

	if permChanged {
		result.Changed = true
	}

	if exists {
		result.Msg = fmt.Sprintf("file %s timestamp updated", path)
	} else {
		result.Msg = fmt.Sprintf("file %s created", path)
	}

	return result, nil
}

// createLink 创建符号链接
func (m *FileModule) createLink(conn *connection.Connection, path string, args map[string]interface{}) (*Result, error) {
	result := &Result{}

	// 获取 src 参数
	srcInterface, ok := args["src"]
	if !ok {
		result.Failed = true
		result.Msg = "state=link requires src parameter"
		return result, nil
	}

	src, ok := srcInterface.(string)
	if !ok {
		result.Failed = true
		result.Msg = "src must be a string"
		return result, nil
	}

	// 检查链接是否已存在且正确
	checkCmd := fmt.Sprintf("readlink %s", path)
	checkResult, err := executeCommand(conn, checkCmd)
	if err == nil && checkResult.RC == 0 {
		// 链接存在，检查是否指向正确的目标
		currentTarget := checkResult.Stdout
		if currentTarget == src {
			// 链接已正确
			result.Changed = false
			result.Msg = fmt.Sprintf("link %s -> %s already correct", path, src)
			return result, nil
		}
		// 链接存在但目标不对，需要更新
		rmCmd := fmt.Sprintf("rm -f %s", path)
		_, _ = executeCommand(conn, rmCmd)
	}

	// 创建符号链接
	lnCmd := fmt.Sprintf("ln -s %s %s", src, path)
	lnResult, err := executeCommand(conn, lnCmd)
	if err != nil || lnResult.RC != 0 {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to create link: %s", lnResult.Stderr)
		return result, nil
	}

	result.Changed = true
	result.Msg = fmt.Sprintf("created link %s -> %s", path, src)
	return result, nil
}

// applyPermissions 应用权限、所有者和组
func (m *FileModule) applyPermissions(conn *connection.Connection, path string, args map[string]interface{}) (bool, error) {
	changed := false

	// 应用 mode（权限）
	if modeInterface, ok := args["mode"]; ok {
		modeStr := ""
		switch v := modeInterface.(type) {
		case string:
			modeStr = v
		case int:
			modeStr = fmt.Sprintf("%o", v)
		case int64:
			modeStr = fmt.Sprintf("%o", v)
		case float64:
			modeStr = fmt.Sprintf("%o", int(v))
		}

		if modeStr != "" {
			chmodCmd := fmt.Sprintf("chmod %s %s", modeStr, path)
			chmodResult, err := executeCommand(conn, chmodCmd)
			if err != nil || chmodResult.RC != 0 {
				return false, fmt.Errorf("failed to chmod: %s", chmodResult.Stderr)
			}
			changed = true
		}
	}

	// 应用 owner
	if ownerInterface, ok := args["owner"]; ok {
		if owner, ok := ownerInterface.(string); ok && owner != "" {
			chownCmd := fmt.Sprintf("chown %s %s", owner, path)
			chownResult, err := executeCommand(conn, chownCmd)
			if err != nil || chownResult.RC != 0 {
				return false, fmt.Errorf("failed to chown: %s", chownResult.Stderr)
			}
			changed = true
		}
	}

	// 应用 group
	if groupInterface, ok := args["group"]; ok {
		if group, ok := groupInterface.(string); ok && group != "" {
			chgrpCmd := fmt.Sprintf("chgrp %s %s", group, path)
			chgrpResult, err := executeCommand(conn, chgrpCmd)
			if err != nil || chgrpResult.RC != 0 {
				return false, fmt.Errorf("failed to chgrp: %s", chgrpResult.Stderr)
			}
			changed = true
		}
	}

	// 处理 recurse（递归应用权限到目录）
	if recurseInterface, ok := args["recurse"]; ok {
		recurse := false
		switch v := recurseInterface.(type) {
		case bool:
			recurse = v
		case string:
			recurse = v == "yes" || v == "true"
		}

		if recurse {
			// 检查是否是目录
			checkCmd := fmt.Sprintf("test -d %s", path)
			checkResult, _ := executeCommand(conn, checkCmd)
			if checkResult.RC == 0 {
				// 递归应用权限
				if modeInterface, ok := args["mode"]; ok {
					modeStr := fmt.Sprintf("%v", modeInterface)
					chmodCmd := fmt.Sprintf("chmod -R %s %s", modeStr, path)
					executeCommand(conn, chmodCmd)
					changed = true
				}
				if ownerInterface, ok := args["owner"]; ok {
					owner := fmt.Sprintf("%v", ownerInterface)
					chownCmd := fmt.Sprintf("chown -R %s %s", owner, path)
					executeCommand(conn, chownCmd)
					changed = true
				}
				if groupInterface, ok := args["group"]; ok {
					group := fmt.Sprintf("%v", groupInterface)
					chgrpCmd := fmt.Sprintf("chgrp -R %s %s", group, path)
					executeCommand(conn, chgrpCmd)
					changed = true
				}
			}
		}
	}

	return changed, nil
}

// 辅助函数：将八进制字符串转换为权限模式
func parseMode(modeStr string) (os.FileMode, error) {
	mode, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid mode: %s", modeStr)
	}
	return os.FileMode(mode), nil
}

// 辅助函数：获取文件信息
func getFileInfo(conn *connection.Connection, path string) (os.FileMode, int, int, error) {
	statCmd := fmt.Sprintf("stat -c '%%a %%u %%g' %s", path)
	result, err := executeCommand(conn, statCmd)
	if err != nil || result.RC != 0 {
		return 0, 0, 0, fmt.Errorf("failed to stat file")
	}

	var mode uint32
	var uid, gid int
	fmt.Sscanf(result.Stdout, "%o %d %d", &mode, &uid, &gid)
	return os.FileMode(mode), uid, gid, nil
}

// 辅助函数：将用户名转换为 UID
func lookupUser(username string) (int, error) {
	// 这里简化处理，实际应该通过 /etc/passwd 查询
	return -1, fmt.Errorf("user lookup not implemented")
}

// 辅助函数：将组名转换为 GID
func lookupGroup(groupname string) (int, error) {
	// 这里简化处理，实际应该通过 /etc/group 查询
	return -1, fmt.Errorf("group lookup not implemented")
}

// 避免编译错误的占位符
var _ = syscall.Stat_t{}
