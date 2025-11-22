package module

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jimyag/ansigo/pkg/connection"
)

// ModuleTransfer 处理模块传输和执行
type ModuleTransfer struct {
	conn *connection.Connection
}

// NewModuleTransfer 创建模块传输器
func NewModuleTransfer(conn *connection.Connection) *ModuleTransfer {
	return &ModuleTransfer{
		conn: conn,
	}
}

// PrepareRemoteDir 在远程创建临时目录
func (mt *ModuleTransfer) PrepareRemoteDir() (string, error) {
	taskID := uuid.New().String()
	remoteDir := fmt.Sprintf("~/.ansible/tmp/ansigo-%s", taskID)

	_, _, exitCode, err := mt.conn.Exec(fmt.Sprintf("mkdir -p %s", remoteDir))
	if err != nil {
		return "", fmt.Errorf("failed to create remote directory: %w", err)
	}
	if exitCode != 0 {
		return "", fmt.Errorf("failed to create remote directory, exit code: %d", exitCode)
	}

	return remoteDir, nil
}

// TransferArgs 将参数传输到远程
func (mt *ModuleTransfer) TransferArgs(args map[string]interface{}, remoteDir string) (string, error) {
	// 序列化参数为 JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal args: %w", err)
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "ansigo-args-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// 写入参数
	if _, err := tmpFile.Write(argsJSON); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write args: %w", err)
	}
	tmpFile.Close()

	// 传输到远程
	remoteArgsPath := filepath.Join(remoteDir, "args.json")
	if err := mt.conn.PutFile(tmpFile.Name(), remoteArgsPath); err != nil {
		return "", fmt.Errorf("failed to transfer args: %w", err)
	}

	return remoteArgsPath, nil
}

// TransferModule 传输模块文件到远程（如果需要）
func (mt *ModuleTransfer) TransferModule(localModulePath, remoteDir string) (string, error) {
	remoteModulePath := filepath.Join(remoteDir, filepath.Base(localModulePath))

	if err := mt.conn.PutFile(localModulePath, remoteModulePath); err != nil {
		return "", fmt.Errorf("failed to transfer module: %w", err)
	}

	// 设置执行权限
	_, _, exitCode, err := mt.conn.Exec(fmt.Sprintf("chmod +x %s", remoteModulePath))
	if err != nil {
		return "", fmt.Errorf("failed to set execute permission: %w", err)
	}
	if exitCode != 0 {
		return "", fmt.Errorf("failed to set execute permission, exit code: %d", exitCode)
	}

	return remoteModulePath, nil
}

// Cleanup 清理远程临时目录
func (mt *ModuleTransfer) Cleanup(remoteDir string) error {
	_, _, _, err := mt.conn.Exec(fmt.Sprintf("rm -rf %s", remoteDir))
	return err
}
