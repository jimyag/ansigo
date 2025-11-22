package module

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jimyag/ansigo/pkg/connection"
)

// LineinfileModule lineinfile 模块实现
// lineinfile 模块用于确保文件中存在或不存在某一行
type LineinfileModule struct{}

// Execute 执行 lineinfile 模块
func (m *LineinfileModule) Execute(conn *connection.Connection, args map[string]interface{}, become bool, becomeUser, becomeMethod string) (*Result, error) {
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

	// 获取 state 参数（默认为 present）
	state := "present"
	if stateInterface, ok := args["state"]; ok {
		if s, ok := stateInterface.(string); ok {
			state = s
		}
	}

	// 获取 line 参数
	lineInterface, ok := args["line"]
	if !ok && state == "present" {
		result.Failed = true
		result.Msg = "missing required argument: line (required when state=present)"
		return result, nil
	}

	var line string
	if lineInterface != nil {
		if l, ok := lineInterface.(string); ok {
			line = l
		} else {
			result.Failed = true
			result.Msg = "line must be a string"
			return result, nil
		}
	}

	// 获取 regexp 参数
	var regexpStr string
	var regexpCompiled *regexp.Regexp
	if regexpInterface, ok := args["regexp"]; ok {
		if r, ok := regexpInterface.(string); ok {
			regexpStr = r
			var err error
			regexpCompiled, err = regexp.Compile(regexpStr)
			if err != nil {
				result.Failed = true
				result.Msg = fmt.Sprintf("invalid regexp: %v", err)
				return result, nil
			}
		}
	}

	// 检查文件是否存在
	checkCmd := fmt.Sprintf("test -f %s", path)
	checkResult, _ := executeCommand(conn, checkCmd)
	fileExists := checkResult != nil && checkResult.RC == 0

	// 如果文件不存在
	if !fileExists {
		// 检查 create 参数
		create := false
		if createInterface, ok := args["create"]; ok {
			switch v := createInterface.(type) {
			case bool:
				create = v
			case string:
				create = v == "yes" || v == "true"
			}
		}

		if !create {
			result.Failed = true
			result.Msg = fmt.Sprintf("file %s does not exist (use create=yes to create)", path)
			return result, nil
		}

		// 创建文件
		if state == "present" {
			touchCmd := fmt.Sprintf("touch %s", path)
			touchResult, err := executeCommand(conn, touchCmd)
			if err != nil || touchResult.RC != 0 {
				result.Failed = true
				result.Msg = fmt.Sprintf("failed to create file: %s", touchResult.Stderr)
				return result, nil
			}
			fileExists = true
		} else {
			// state=absent 且文件不存在，无需操作
			result.Changed = false
			result.Msg = fmt.Sprintf("file %s does not exist, nothing to do", path)
			return result, nil
		}
	}

	// 读取文件内容
	catCmd := fmt.Sprintf("cat %s", path)
	catResult, err := executeCommand(conn, catCmd)
	if err != nil || catResult.RC != 0 {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to read file: %s", catResult.Stderr)
		return result, nil
	}

	fileContent := catResult.Stdout
	lines := strings.Split(fileContent, "\n")

	// 处理 state=absent
	if state == "absent" {
		return m.ensureAbsent(conn, path, lines, line, regexpCompiled, result)
	}

	// 处理 state=present
	return m.ensurePresent(conn, path, lines, line, regexpCompiled, args, result)
}

// ensurePresent 确保行存在
func (m *LineinfileModule) ensurePresent(conn *connection.Connection, path string, lines []string, line string, regexpCompiled *regexp.Regexp, args map[string]interface{}, result *Result) (*Result, error) {
	// 查找匹配的行
	matchedLineIndex := -1
	if regexpCompiled != nil {
		for i, l := range lines {
			if regexpCompiled.MatchString(l) {
				matchedLineIndex = i
				break
			}
		}
	}

	// 如果找到匹配行
	if matchedLineIndex >= 0 {
		// 检查是否需要替换
		if lines[matchedLineIndex] == line {
			// 行已存在且内容相同，无需修改
			result.Changed = false
			result.Msg = "line already present"
			return result, nil
		}
		// 替换行
		lines[matchedLineIndex] = line
		result.Changed = true
	} else {
		// 没有找到匹配行，需要插入
		// 获取 insertafter 和 insertbefore 参数
		insertAfter, hasInsertAfter := args["insertafter"].(string)
		insertBefore, hasInsertBefore := args["insertbefore"].(string)

		insertIndex := -1

		if hasInsertBefore {
			// 在匹配行之前插入
			if insertBefore == "BOF" {
				// 在文件开头插入
				insertIndex = 0
			} else {
				// 查找匹配行
				beforeRegexp, err := regexp.Compile(insertBefore)
				if err == nil {
					for i, l := range lines {
						if beforeRegexp.MatchString(l) {
							insertIndex = i
							break
						}
					}
				}
			}
		} else if hasInsertAfter {
			// 在匹配行之后插入
			if insertAfter == "EOF" {
				// 在文件末尾插入（默认行为）
				insertIndex = len(lines)
			} else {
				// 查找匹配行
				afterRegexp, err := regexp.Compile(insertAfter)
				if err == nil {
					for i, l := range lines {
						if afterRegexp.MatchString(l) {
							insertIndex = i + 1
							break
						}
					}
				}
			}
		}

		// 如果没有找到插入位置，默认在末尾
		if insertIndex == -1 {
			insertIndex = len(lines)
		}

		// 插入行
		if insertIndex >= len(lines) {
			lines = append(lines, line)
		} else {
			// 在指定位置插入
			lines = append(lines[:insertIndex], append([]string{line}, lines[insertIndex:]...)...)
		}
		result.Changed = true
	}

	// 写回文件
	if result.Changed {
		newContent := strings.Join(lines, "\n")
		writeCmd := fmt.Sprintf("cat > %s << 'ANSIGO_LINEINFILE_EOF'\n%s\nANSIGO_LINEINFILE_EOF", path, newContent)
		writeResult, err := executeCommand(conn, writeCmd)
		if err != nil || writeResult.RC != 0 {
			result.Failed = true
			result.Msg = fmt.Sprintf("failed to write file: %s", writeResult.Stderr)
			return result, nil
		}
		result.Msg = "line added or modified"
	}

	return result, nil
}

// ensureAbsent 确保行不存在
func (m *LineinfileModule) ensureAbsent(conn *connection.Connection, path string, lines []string, line string, regexpCompiled *regexp.Regexp, result *Result) (*Result, error) {
	// 查找并删除匹配的行
	newLines := []string{}
	removed := false

	for _, l := range lines {
		shouldRemove := false

		// 使用 regexp 匹配
		if regexpCompiled != nil && regexpCompiled.MatchString(l) {
			shouldRemove = true
		} else if line != "" && l == line {
			// 使用精确匹配
			shouldRemove = true
		}

		if shouldRemove {
			removed = true
		} else {
			newLines = append(newLines, l)
		}
	}

	if !removed {
		result.Changed = false
		result.Msg = "line not present, nothing to do"
		return result, nil
	}

	// 写回文件
	newContent := strings.Join(newLines, "\n")
	writeCmd := fmt.Sprintf("cat > %s << 'ANSIGO_LINEINFILE_EOF'\n%s\nANSIGO_LINEINFILE_EOF", path, newContent)
	writeResult, err := executeCommand(conn, writeCmd)
	if err != nil || writeResult.RC != 0 {
		result.Failed = true
		result.Msg = fmt.Sprintf("failed to write file: %s", writeResult.Stderr)
		return result, nil
	}

	result.Changed = true
	result.Msg = "line removed"
	return result, nil
}
