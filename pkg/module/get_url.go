package module

import (
	"fmt"
	"strings"

	"github.com/jimyag/ansigo/pkg/connection"
)

// GetUrlModule get_url 模块实现
// get_url 模块用于从 HTTP/HTTPS/FTP URL 下载文件到远程节点
type GetUrlModule struct{}

// Execute 执行 get_url 模块
func (m *GetUrlModule) Execute(conn *connection.Connection, args map[string]interface{}, become bool, becomeUser, becomeMethod string) (*Result, error) {
	result := &Result{}

	// 获取必需参数 url
	urlInterface, ok := args["url"]
	if !ok {
		result.Failed = true
		result.Msg = "missing required argument: url"
		return result, nil
	}

	url, ok := urlInterface.(string)
	if !ok {
		result.Failed = true
		result.Msg = "url must be a string"
		return result, nil
	}

	// 获取必需参数 dest
	destInterface, ok := args["dest"]
	if !ok {
		result.Failed = true
		result.Msg = "missing required argument: dest"
		return result, nil
	}

	dest, ok := destInterface.(string)
	if !ok {
		result.Failed = true
		result.Msg = "dest must be a string"
		return result, nil
	}

	// 获取可选参数 mode
	mode := ""
	if modeInterface, ok := args["mode"]; ok {
		if m, ok := modeInterface.(string); ok {
			mode = m
		}
	}

	// 获取可选参数 owner
	owner := ""
	if ownerInterface, ok := args["owner"]; ok {
		if o, ok := ownerInterface.(string); ok {
			owner = o
		}
	}

	// 获取可选参数 group
	group := ""
	if groupInterface, ok := args["group"]; ok {
		if g, ok := groupInterface.(string); ok {
			group = g
		}
	}

	// 获取可选参数 force (default: false)
	force := false
	if forceInterface, ok := args["force"]; ok {
		switch v := forceInterface.(type) {
		case bool:
			force = v
		case string:
			force = v == "yes" || v == "true"
		}
	}

	// 获取可选参数 checksum
	checksum := ""
	if checksumInterface, ok := args["checksum"]; ok {
		if c, ok := checksumInterface.(string); ok {
			checksum = c
		}
	}

	// 检查目标文件是否已存在
	fileExists, err := m.checkFileExists(conn, dest, become, becomeUser, becomeMethod)
	if err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to check if file exists: %v", err)
		return result, nil
	}

	// 如果文件存在且 force=false，跳过下载
	if fileExists && !force {
		// 如果指定了 checksum，验证现有文件
		if checksum != "" {
			valid, err := m.verifyChecksum(conn, dest, checksum, become, becomeUser, becomeMethod)
			if err != nil {
				result.Failed = true
				result.Msg = fmt.Sprintf("failed to verify checksum: %v", err)
				return result, nil
			}
			if valid {
				result.Changed = false
				result.Msg = "file already exists and checksum matches"
				return result, nil
			}
			// checksum 不匹配，需要重新下载
		} else {
			result.Changed = false
			result.Msg = "file already exists"
			return result, nil
		}
	}

	// 创建目标目录（如果不存在）
	if err := m.createDestDir(conn, dest, become, becomeUser, becomeMethod); err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to create destination directory: %v", err)
		return result, nil
	}

	// 下载文件
	if err := m.downloadFile(conn, url, dest, become, becomeUser, becomeMethod); err != nil {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to download file: %v", err)
		return result, nil
	}

	// 验证 checksum（如果指定）
	if checksum != "" {
		valid, err := m.verifyChecksum(conn, dest, checksum, become, becomeUser, becomeMethod)
		if err != nil {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to verify checksum: %v", err)
			return result, nil
		}
		if !valid {
			result.Failed = true
			result.Msg = "checksum verification failed"
			return result, nil
		}
	}

	// 设置文件权限（如果指定）
	if mode != "" {
		if err := m.setFileMode(conn, dest, mode, become, becomeUser, becomeMethod); err != nil {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to set file mode: %v", err)
			return result, nil
		}
	}

	// 设置文件所有者和组（如果指定）
	if owner != "" || group != "" {
		if err := m.setFileOwner(conn, dest, owner, group, become, becomeUser, becomeMethod); err != nil {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to set file owner: %v", err)
			return result, nil
		}
	}

	result.Changed = true
	result.Msg = fmt.Sprintf("file downloaded successfully from %s to %s", url, dest)
	return result, nil
}

// checkFileExists 检查文件是否存在
func (m *GetUrlModule) checkFileExists(conn *connection.Connection, path string, become bool, becomeUser, becomeMethod string) (bool, error) {
	cmd := fmt.Sprintf("test -f %s", path)

	var exitCode int
	var err error

	if become {
		_, _, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		_, _, exitCode, err = conn.Exec(cmd)
	}

	if err != nil {
		return false, err
	}

	return exitCode == 0, nil
}

// createDestDir 创建目标目录
func (m *GetUrlModule) createDestDir(conn *connection.Connection, dest string, become bool, becomeUser, becomeMethod string) error {
	// 提取目录路径
	cmd := fmt.Sprintf("mkdir -p $(dirname %s)", dest)

	var stderr []byte
	var exitCode int
	var err error

	if become {
		_, stderr, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		_, stderr, exitCode, err = conn.Exec(cmd)
	}

	if err != nil || exitCode != 0 {
		return fmt.Errorf("failed to create directory: %s", strings.TrimSpace(string(stderr)))
	}

	return nil
}

// downloadFile 下载文件
func (m *GetUrlModule) downloadFile(conn *connection.Connection, url, dest string, become bool, becomeUser, becomeMethod string) error {
	// 使用 curl 或 wget 下载文件
	// 优先使用 curl，如果不存在则使用 wget
	cmd := fmt.Sprintf("if command -v curl >/dev/null 2>&1; then curl -fsSL -o %s %s; elif command -v wget >/dev/null 2>&1; then wget -q -O %s %s; else echo 'neither curl nor wget found' >&2; exit 1; fi", dest, url, dest, url)

	var stderr []byte
	var exitCode int
	var err error

	if become {
		_, stderr, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		_, stderr, exitCode, err = conn.Exec(cmd)
	}

	if err != nil || exitCode != 0 {
		return fmt.Errorf("download failed: %s", strings.TrimSpace(string(stderr)))
	}

	return nil
}

// verifyChecksum 验证文件 checksum
func (m *GetUrlModule) verifyChecksum(conn *connection.Connection, path, checksum string, become bool, becomeUser, becomeMethod string) (bool, error) {
	// checksum 格式: "sha256:abc123..." 或 "md5:def456..."
	parts := strings.SplitN(checksum, ":", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid checksum format, expected 'algorithm:hash'")
	}

	algorithm := parts[0]
	expectedHash := parts[1]

	// 根据算法选择命令
	var cmd string
	switch algorithm {
	case "sha256":
		cmd = fmt.Sprintf("sha256sum %s | awk '{print $1}'", path)
	case "sha1":
		cmd = fmt.Sprintf("sha1sum %s | awk '{print $1}'", path)
	case "md5":
		cmd = fmt.Sprintf("md5sum %s | awk '{print $1}'", path)
	default:
		return false, fmt.Errorf("unsupported checksum algorithm: %s", algorithm)
	}

	var stdout []byte
	var exitCode int
	var err error

	if become {
		stdout, _, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		stdout, _, exitCode, err = conn.Exec(cmd)
	}

	if err != nil || exitCode != 0 {
		return false, fmt.Errorf("failed to calculate checksum")
	}

	actualHash := strings.TrimSpace(string(stdout))
	return actualHash == expectedHash, nil
}

// setFileMode 设置文件权限
func (m *GetUrlModule) setFileMode(conn *connection.Connection, path, mode string, become bool, becomeUser, becomeMethod string) error {
	cmd := fmt.Sprintf("chmod %s %s", mode, path)

	var stderr []byte
	var exitCode int
	var err error

	if become {
		_, stderr, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		_, stderr, exitCode, err = conn.Exec(cmd)
	}

	if err != nil || exitCode != 0 {
		return fmt.Errorf("failed to set mode: %s", strings.TrimSpace(string(stderr)))
	}

	return nil
}

// setFileOwner 设置文件所有者和组
func (m *GetUrlModule) setFileOwner(conn *connection.Connection, path, owner, group string, become bool, becomeUser, becomeMethod string) error {
	ownerGroup := owner
	if group != "" {
		if owner != "" {
			ownerGroup = fmt.Sprintf("%s:%s", owner, group)
		} else {
			ownerGroup = fmt.Sprintf(":%s", group)
		}
	}

	cmd := fmt.Sprintf("chown %s %s", ownerGroup, path)

	var stderr []byte
	var exitCode int
	var err error

	if become {
		_, stderr, exitCode, err = conn.ExecWithBecome(cmd, becomeUser, becomeMethod)
	} else {
		_, stderr, exitCode, err = conn.Exec(cmd)
	}

	if err != nil || exitCode != 0 {
		return fmt.Errorf("failed to set owner: %s", strings.TrimSpace(string(stderr)))
	}

	return nil
}
