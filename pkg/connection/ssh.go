package connection

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jimyag/ansigo/pkg/errors"
	"github.com/jimyag/ansigo/pkg/inventory"
	"golang.org/x/crypto/ssh"
)

// Connection 表示一个 SSH 连接
type Connection struct {
	client *ssh.Client
	host   *inventory.Host
}

// Manager 管理 SSH 连接
type Manager struct {
	timeout time.Duration
}

// NewManager 创建一个新的连接管理器
func NewManager() *Manager {
	return &Manager{
		timeout: 30 * time.Second,
	}
}

// Connect 连接到主机
func (m *Manager) Connect(host *inventory.Host) (*Connection, error) {
	// 从 host.Vars 获取连接参数
	ansibleHost, _ := host.Vars["ansible_host"].(string)
	if ansibleHost == "" {
		ansibleHost = host.Name
	}

	port := 22
	if portStr, ok := host.Vars["ansible_port"].(string); ok {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	user, _ := host.Vars["ansible_user"].(string)
	if user == "" {
		user = "root"
	}

	password, _ := host.Vars["ansible_password"].(string)
	keyFile, _ := host.Vars["ansible_ssh_private_key_file"].(string)

	// 构建 SSH 配置
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 测试用，生产环境应该验证
		Timeout:         m.timeout,
	}

	// 添加认证方式
	if password != "" {
		config.Auth = append(config.Auth, ssh.Password(password))
	}

	if keyFile != "" {
		auth, err := publicKeyAuth(keyFile)
		if err == nil {
			config.Auth = append(config.Auth, auth)
		}
	}

	// 如果没有指定认证方式，尝试默认密钥
	if len(config.Auth) == 0 {
		homeDir, _ := os.UserHomeDir()
		defaultKeys := []string{
			filepath.Join(homeDir, ".ssh", "id_rsa"),
			filepath.Join(homeDir, ".ssh", "id_ed25519"),
		}

		for _, keyPath := range defaultKeys {
			if auth, err := publicKeyAuth(keyPath); err == nil {
				config.Auth = append(config.Auth, auth)
			}
		}
	}

	// 连接
	addr := fmt.Sprintf("%s:%d", ansibleHost, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, errors.NewUnreachableError(host.Name, err)
	}

	return &Connection{
		client: client,
		host:   host,
	}, nil
}

// publicKeyAuth 创建公钥认证
func publicKeyAuth(keyPath string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}

// Exec 执行命令
func (c *Connection) Exec(cmd string) (stdout, stderr []byte, exitCode int, err error) {
	return c.ExecWithTimeout(cmd, 30*time.Second)
}

// ExecWithTimeout 执行命令（带超时）
func (c *Connection) ExecWithTimeout(cmd string, timeout time.Duration) (stdout, stderr []byte, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	session, err := c.client.NewSession()
	if err != nil {
		return nil, nil, -1, err
	}
	defer session.Close()

	// 创建缓冲区
	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// 启动命令
	if err := session.Start(cmd); err != nil {
		return nil, nil, -1, err
	}

	// 等待完成或超时
	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case <-ctx.Done():
		// 超时，尝试终止
		session.Signal(ssh.SIGKILL)
		return nil, nil, -1, errors.NewTimeoutError(c.host.Name, cmd, timeout)
	case err := <-done:
		stdout = stdoutBuf.Bytes()
		stderr = stderrBuf.Bytes()

		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				return stdout, stderr, exitErr.ExitStatus(), nil
			}
			return stdout, stderr, -1, err
		}
		return stdout, stderr, 0, nil
	}
}

// PutFile 上传文件到远程主机
func (c *Connection) PutFile(localPath, remotePath string) error {
	// 读取本地文件
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// 使用 scp 协议上传
	// 简化版：使用 cat > file 命令
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// 创建远程文件
	cmd := fmt.Sprintf("cat > %s", remotePath)
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	if err := session.Start(cmd); err != nil {
		return err
	}

	// 写入数据
	if _, err := io.Copy(stdin, bytes.NewReader(data)); err != nil {
		return err
	}
	stdin.Close()

	// 等待完成
	if err := session.Wait(); err != nil {
		return err
	}

	return nil
}

// GetFile 从远程主机下载文件
func (c *Connection) GetFile(remotePath, localPath string) error {
	// 使用 cat 命令读取远程文件
	stdout, _, exitCode, err := c.Exec(fmt.Sprintf("cat %s", remotePath))
	if err != nil {
		return err
	}

	if exitCode != 0 {
		return fmt.Errorf("failed to read remote file, exit code: %d", exitCode)
	}

	// 写入本地文件
	if err := os.WriteFile(localPath, stdout, 0o644); err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	return nil
}

// Close 关闭连接
func (c *Connection) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// ExecWithBecome 使用权限提升执行命令
func (c *Connection) ExecWithBecome(cmd string, becomeUser, becomeMethod string) (stdout, stderr []byte, exitCode int, err error) {
	// 如果没有指定 become_user，默认为 root
	if becomeUser == "" {
		becomeUser = "root"
	}

	// 如果没有指定 become_method，默认为 sudo
	if becomeMethod == "" {
		becomeMethod = "sudo"
	}

	// 构建 sudo 命令
	// 使用 -n 选项避免密码提示（假设配置了 NOPASSWD）
	// 使用 -u 指定目标用户
	var sudoCmd string
	switch becomeMethod {
	case "sudo":
		if becomeUser == "root" {
			sudoCmd = fmt.Sprintf("sudo -n sh -c %s", shellQuote(cmd))
		} else {
			sudoCmd = fmt.Sprintf("sudo -n -u %s sh -c %s", becomeUser, shellQuote(cmd))
		}
	case "su":
		// su 方式（不太常用）
		sudoCmd = fmt.Sprintf("su - %s -c %s", becomeUser, shellQuote(cmd))
	default:
		return nil, nil, -1, fmt.Errorf("unsupported become method: %s", becomeMethod)
	}

	return c.ExecWithTimeout(sudoCmd, 30*time.Second)
}

// shellQuote 为 shell 命令添加引号
func shellQuote(s string) string {
	// 简单实现：使用单引号，并转义内部的单引号
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// ExecuteCommand 执行命令并返回标准输出（用于 facts 收集）
func (c *Connection) ExecuteCommand(cmd string) ([]byte, error) {
	stdout, _, exitCode, err := c.Exec(cmd)
	if err != nil {
		return nil, err
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("command failed with exit code %d", exitCode)
	}
	return stdout, nil
}
