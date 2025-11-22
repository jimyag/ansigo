package errors

import (
	"fmt"
	"time"
)

// ErrorType 定义错误类型
type ErrorType int

const (
	// ErrUnreachable 主机不可达（连接失败）
	ErrUnreachable ErrorType = iota
	// ErrFailed 模块执行失败
	ErrFailed
	// ErrTimeout 执行超时
	ErrTimeout
	// ErrParse 解析错误（Inventory、Playbook 等）
	ErrParse
	// ErrInvalidArgs 参数错误
	ErrInvalidArgs
	// ErrModuleNotFound 模块未找到
	ErrModuleNotFound
)

// ExecutionError 统一的执行错误类型
type ExecutionError struct {
	Type      ErrorType              // 错误类型
	Host      string                 // 目标主机（如果适用）
	Task      string                 // 任务名称（如果适用）
	Module    string                 // 模块名称（如果适用）
	Message   string                 // 错误消息
	Cause     error                  // 原始错误
	Retriable bool                   // 是否可重试
	Details   map[string]interface{} // 额外的错误详情
}

func (e *ExecutionError) Error() string {
	if e.Host != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Host, e.Task, e.Message)
	}
	return e.Message
}

func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// NewUnreachableError 创建不可达错误
func NewUnreachableError(host string, cause error) *ExecutionError {
	return &ExecutionError{
		Type:      ErrUnreachable,
		Host:      host,
		Message:   fmt.Sprintf("Failed to connect to host: %v", cause),
		Cause:     cause,
		Retriable: true,
	}
}

// NewModuleFailedError 创建模块失败错误
func NewModuleFailedError(host, task, module, msg string) *ExecutionError {
	return &ExecutionError{
		Type:      ErrFailed,
		Host:      host,
		Task:      task,
		Module:    module,
		Message:   msg,
		Retriable: false,
	}
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(host, task string, duration time.Duration) *ExecutionError {
	return &ExecutionError{
		Type:      ErrTimeout,
		Host:      host,
		Task:      task,
		Message:   fmt.Sprintf("Task timeout after %v", duration),
		Retriable: true,
	}
}

// NewParseError 创建解析错误
func NewParseError(filePath string, cause error) *ExecutionError {
	return &ExecutionError{
		Type:      ErrParse,
		Message:   fmt.Sprintf("Failed to parse %s: %v", filePath, cause),
		Cause:     cause,
		Retriable: false,
	}
}
