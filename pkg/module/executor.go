package module

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jimyag/ansigo/pkg/connection"
)

// Executor 模块执行器
type Executor struct{}

// NewExecutor 创建一个新的模块执行器
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute 执行模块
func (e *Executor) Execute(conn *connection.Connection, moduleName string, args map[string]interface{}) (*Result, error) {
	switch moduleName {
	case "ping":
		return e.executePing(conn)
	case "raw":
		return e.executeRaw(conn, args)
	case "command":
		return e.executeCommand(conn, args)
	case "shell":
		return e.executeShell(conn, args)
	case "copy":
		return e.executeCopy(conn, args)
	case "debug":
		return e.executeDebug(args)
	default:
		return nil, fmt.Errorf("unsupported module: %s", moduleName)
	}
}

// executePing 执行 ping 模块
func (e *Executor) executePing(conn *connection.Connection) (*Result, error) {
	// Ansible 的 ping 模块只是测试连接性，不需要 Python
	// 我们简单返回 pong
	result := &Result{
		Changed: false,
		Ping:    "pong",
	}

	return result, nil
}

// executeRaw 执行 raw 模块
func (e *Executor) executeRaw(conn *connection.Connection, args map[string]interface{}) (*Result, error) {
	// raw 模块直接执行命令
	cmd, ok := args["_raw_params"].(string)
	if !ok {
		// 尝试其他可能的参数名
		if c, ok := args["cmd"].(string); ok {
			cmd = c
		} else {
			return nil, fmt.Errorf("raw module requires command")
		}
	}

	stdout, stderr, exitCode, err := conn.Exec(cmd)
	if err != nil {
		return &Result{
			Failed: true,
			Msg:    err.Error(),
			RC:     exitCode,
		}, nil
	}

	result := &Result{
		Changed: true, // raw 模块总是 changed
		RC:      exitCode,
		Stdout:  string(stdout),
		Stderr:  string(stderr),
	}

	if exitCode != 0 {
		result.Failed = true
		result.Msg = fmt.Sprintf("non-zero return code: %d", exitCode)
	}

	return result, nil
}

// executeCommand 执行 command 模块
func (e *Executor) executeCommand(conn *connection.Connection, args map[string]interface{}) (*Result, error) {
	// command 模块执行命令，但不使用 shell 解析
	// 获取命令参数
	var cmd string
	if rawCmd, ok := args["_raw_params"].(string); ok {
		cmd = rawCmd
	} else if argvInterface, ok := args["argv"]; ok {
		// 支持 argv 数组格式
		if argv, ok := argvInterface.([]interface{}); ok {
			parts := make([]string, len(argv))
			for i, v := range argv {
				parts[i] = fmt.Sprintf("%v", v)
			}
			cmd = strings.Join(parts, " ")
		}
	} else if cmdArg, ok := args["cmd"].(string); ok {
		cmd = cmdArg
	} else {
		return &Result{
			Failed: true,
			Msg:    "command module requires 'cmd' or '_raw_params' argument",
		}, nil
	}

	// 获取工作目录
	chdir, _ := args["chdir"].(string)
	if chdir != "" {
		cmd = fmt.Sprintf("cd %s && %s", chdir, cmd)
	}

	// 执行命令
	stdout, stderr, exitCode, err := conn.Exec(cmd)
	if err != nil {
		return &Result{
			Failed: true,
			Msg:    err.Error(),
			RC:     exitCode,
		}, nil
	}

	result := &Result{
		Changed: true, // command 模块总是 changed
		RC:      exitCode,
		Stdout:  strings.TrimSpace(string(stdout)),
		Stderr:  strings.TrimSpace(string(stderr)),
	}

	if exitCode != 0 {
		result.Failed = true
		result.Msg = "non-zero return code"
	}

	return result, nil
}

// executeShell 执行 shell 模块
func (e *Executor) executeShell(conn *connection.Connection, args map[string]interface{}) (*Result, error) {
	// shell 模块通过 shell 执行命令，支持管道、重定向等
	var cmd string
	if rawCmd, ok := args["_raw_params"].(string); ok {
		cmd = rawCmd
	} else if cmdArg, ok := args["cmd"].(string); ok {
		cmd = cmdArg
	} else {
		return &Result{
			Failed: true,
			Msg:    "shell module requires 'cmd' or '_raw_params' argument",
		}, nil
	}

	// 获取工作目录
	chdir, _ := args["chdir"].(string)

	// shell 模块默认使用 /bin/sh -c
	executable, _ := args["executable"].(string)
	if executable == "" {
		executable = "/bin/sh"
	}

	// 构建完整命令
	fullCmd := fmt.Sprintf("%s -c %s", executable, shellQuote(cmd))
	if chdir != "" {
		fullCmd = fmt.Sprintf("cd %s && %s", chdir, fullCmd)
	}

	// 执行命令
	stdout, stderr, exitCode, err := conn.Exec(fullCmd)
	if err != nil {
		return &Result{
			Failed: true,
			Msg:    err.Error(),
			RC:     exitCode,
		}, nil
	}

	result := &Result{
		Changed: true, // shell 模块总是 changed
		RC:      exitCode,
		Stdout:  strings.TrimSpace(string(stdout)),
		Stderr:  strings.TrimSpace(string(stderr)),
	}

	if exitCode != 0 {
		result.Failed = true
		result.Msg = "non-zero return code"
	}

	return result, nil
}

// executeCopy 执行 copy 模块
func (e *Executor) executeCopy(conn *connection.Connection, args map[string]interface{}) (*Result, error) {
	// copy 模块用于文件传输
	dest, ok := args["dest"].(string)
	if !ok {
		return &Result{
			Failed: true,
			Msg:    "copy module requires 'dest' argument",
		}, nil
	}

	// 检查是否有 content 参数
	if content, hasContent := args["content"].(string); hasContent {
		// 使用 content 参数直接写入
		writeCmd := fmt.Sprintf("cat > %s << 'ANSIGO_EOF'\n%s\nANSIGO_EOF", dest, content)
		_, stderr, exitCode, err := conn.Exec(writeCmd)
		if err != nil {
			return &Result{
				Failed: true,
				Msg:    fmt.Sprintf("failed to write content: %s", err.Error()),
				RC:     exitCode,
				Stderr: string(stderr),
			}, nil
		}
		if exitCode != 0 {
			return &Result{
				Failed: true,
				Msg:    "failed to write content to destination",
				RC:     exitCode,
				Stderr: string(stderr),
			}, nil
		}

		return &Result{
			Changed: true,
			Dest:    dest,
		}, nil
	}

	// 否则需要 src 参数
	src, ok := args["src"].(string)
	if !ok {
		return &Result{
			Failed: true,
			Msg:    "copy module requires either 'src' or 'content' argument",
		}, nil
	}

	// 传输文件（这里需要本地文件系统访问）
	// 注意：在实际实现中，我们需要从控制节点读取文件
	// 简化版本：假设 src 是控制节点上的路径
	if err := conn.PutFile(src, dest); err != nil {
		return &Result{
			Failed: true,
			Msg:    fmt.Sprintf("failed to copy file: %s", err.Error()),
		}, nil
	}

	// 设置文件权限（如果指定）
	if mode, ok := args["mode"].(string); ok {
		chmodCmd := fmt.Sprintf("chmod %s %s", mode, dest)
		_, _, exitCode, err := conn.Exec(chmodCmd)
		if err != nil || exitCode != 0 {
			return &Result{
				Failed: true,
				Msg:    "failed to set file permissions",
				RC:     exitCode,
			}, nil
		}
	}

	return &Result{
		Changed: true, // 文件已传输，标记为 changed
		Dest:    dest,
	}, nil
}

// executeDebug 执行 debug 模块
func (e *Executor) executeDebug(args map[string]interface{}) (*Result, error) {
	// debug 模块用于输出调试信息，不需要连接
	var msg string

	if msgArg, ok := args["msg"].(string); ok {
		msg = msgArg
	} else if varArg, ok := args["var"].(string); ok {
		// var 参数用于打印变量值
		msg = fmt.Sprintf("%s: %v", varArg, args[varArg])
	} else {
		msg = "Debug output"
	}

	return &Result{
		Changed: false,
		Msg:     msg,
	}, nil
}

// shellQuote 对 shell 命令进行引号转义
func shellQuote(s string) string {
	// 简单实现：使用单引号包裹，并转义内部的单引号
	s = strings.ReplaceAll(s, "'", "'\"'\"'")
	return "'" + s + "'"
}

// ResultToJSON 将结果转换为 JSON
func (r *Result) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
