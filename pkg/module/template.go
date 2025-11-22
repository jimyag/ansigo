package module

import (
	"fmt"
	"strings"

	"github.com/jimyag/ansigo/pkg/connection"
)

// TemplateModule template 模块实现
// template 模块用于渲染 Jinja2 模板并部署到目标主机
type TemplateModule struct{}

// Execute 执行 template 模块
// 注意：模板渲染由 runner 预处理，这里只负责文件传输和权限设置
func (m *TemplateModule) Execute(conn *connection.Connection, args map[string]interface{}, become bool, becomeUser, becomeMethod string) (*Result, error) {
	result := &Result{}

	// 获取必需参数：dest（目标路径）
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

	// 获取渲染后的内容（由 runner 预处理并设置）
	contentInterface, ok := args["_rendered_content"]
	if !ok {
		result.Failed = true
		result.Msg = "internal error: _rendered_content not provided by runner"
		return result, nil
	}

	content, ok := contentInterface.(string)
	if !ok {
		result.Failed = true
		result.Msg = "internal error: _rendered_content must be a string"
		return result, nil
	}

	// 检查目标文件是否存在
	changed := false
	checkCmd := fmt.Sprintf("test -f %s", dest)
	checkResult, err := executeCommand(conn, checkCmd)

	fileExists := err == nil && checkResult.RC == 0

	if fileExists {
		// 文件存在，读取现有内容并比较
		// 使用 cat 读取，但要注意 executeCommand 会 TrimSpace
		catCmd := fmt.Sprintf("cat %s", dest)
		catResult, err := executeCommand(conn, catCmd)
		if err == nil && catResult.RC == 0 {
			existingContent := catResult.Stdout
			// 比较内容时也 TrimSpace，保持一致
			if strings.TrimSpace(existingContent) != strings.TrimSpace(content) {
				changed = true
			}
		} else {
			// 无法读取文件，假设需要更新
			changed = true
		}
	} else {
		// 文件不存在，需要创建
		changed = true
	}

	// 如果需要备份，先备份原文件
	if backup, ok := args["backup"].(bool); ok && backup && !changed {
		// 只有文件存在且内容不同时才备份
		if checkResult.RC == 0 {
			backupCmd := fmt.Sprintf("cp -p %s %s.bak", dest, dest)
			_, _ = executeCommand(conn, backupCmd)
		}
	}

	// 写入内容到目标文件
	if changed {
		// 使用 heredoc 写入内容，避免特殊字符问题
		writeCmd := fmt.Sprintf("cat > %s << 'ANSIGO_TEMPLATE_EOF'\n%s\nANSIGO_TEMPLATE_EOF", dest, content)
		writeResult, err := executeCommand(conn, writeCmd)
		if err != nil || writeResult.RC != 0 {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to write template to dest: %s", writeResult.Stderr)
			return result, nil
		}
	}

	// 应用权限、所有者、组
	permChanged, err := m.applyPermissions(conn, dest, args)
	if err != nil {
		result.Failed = true
		result.Msg = err.Error()
		return result, nil
	}

	if permChanged {
		changed = true
	}

	// 如果指定了 validate，执行验证命令
	if validateInterface, ok := args["validate"]; ok {
		if validate, ok := validateInterface.(string); ok && validate != "" {
			// %s 会被替换为目标文件路径
			validateCmd := fmt.Sprintf(validate, dest)
			validateResult, err := executeCommand(conn, validateCmd)
			if err != nil || validateResult.RC != 0 {
				result.Failed = true
				result.Msg = fmt.Sprintf("validation failed: %s", validateResult.Stderr)
				return result, nil
			}
		}
	}

	result.Changed = changed
	if changed {
		result.Msg = fmt.Sprintf("template rendered to %s", dest)
	} else {
		result.Msg = fmt.Sprintf("template already up to date at %s", dest)
	}
	result.Data = map[string]interface{}{
		"dest": dest,
	}

	return result, nil
}

// applyPermissions 应用权限、所有者和组（复用 file 模块的逻辑）
func (m *TemplateModule) applyPermissions(conn *connection.Connection, path string, args map[string]interface{}) (bool, error) {
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

	return changed, nil
}
