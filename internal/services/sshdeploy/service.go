// Package sshdeploy 提供通过 SSH 自动部署/管理自建节点（VPS）的能力：
// 使用密码认证连接远程主机，执行安装/管理命令，实现"填 IP+密码 全自动搭建"。
package sshdeploy

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Credentials SSH 连接凭据。
type Credentials struct {
	Host     string
	Port     int
	User     string
	Password string
}

// Client 一次 SSH 会话。
type Client struct {
	conn *ssh.Client
}

// Dial 建立 SSH 连接。
func Dial(cred Credentials, timeout time.Duration) (*Client, error) {
	if cred.Host == "" {
		return nil, fmt.Errorf("主机地址不能为空")
	}
	if cred.Password == "" {
		return nil, fmt.Errorf("SSH 密码不能为空")
	}
	port := cred.Port
	if port <= 0 {
		port = 22
	}
	user := cred.User
	if user == "" {
		user = "root"
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(cred.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 自建节点场景，接受未知主机指纹
		Timeout:         timeout,
		ClientVersion:   "SSH-2.0-CBoardSelfHost",
	}

	addr := fmt.Sprintf("%s:%d", cred.Host, port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	return &Client{conn: conn}, nil
}

// Close 关闭连接。
func (c *Client) Close() error {
	if c != nil && c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Run 执行一条远程命令，返回 stdout（stderr 并入错误信息）。
func (c *Client) Run(ctx context.Context, command string, timeout time.Duration) (string, error) {
	sess, err := c.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建会话失败: %w", err)
	}
	defer sess.Close()

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	done := make(chan error, 1)
	go func() {
		done <- sess.Run(command)
	}()

	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		return "", ctx.Err()
	case err := <-done:
		if err != nil {
			stderr := strings.TrimSpace(errBuf.String())
			if stderr != "" {
				return strings.TrimSpace(outBuf.String()), fmt.Errorf("%w: %s", err, stderr)
			}
			return strings.TrimSpace(outBuf.String()), err
		}
		return strings.TrimSpace(outBuf.String()), nil
	}
}

// RunWithTimeout 带超时执行远程命令。
func (c *Client) RunWithTimeout(command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Run(ctx, command, timeout)
}

// UploadAndRun 将脚本内容上传到远程并执行（用 base64 避免转义问题）。
func (c *Client) UploadAndRun(ctx context.Context, remotePath, content string, timeout time.Duration) (string, error) {
	// base64 编码后写入，避免 heredoc/引号转义问题
	encoded := base64Encode(content)
	writeCmd := fmt.Sprintf("echo '%s' | base64 -d > %s && chmod +x %s", encoded, remotePath, remotePath)
	if _, err := c.RunWithTimeout(writeCmd, 30*time.Second); err != nil {
		return "", fmt.Errorf("上传脚本失败: %w", err)
	}
	return c.Run(ctx, fmt.Sprintf("bash %s", remotePath), timeout)
}

// WriteFile 直接写入文件内容（base64 方式）。
func (c *Client) WriteFile(remotePath, content string) error {
	encoded := base64Encode(content)
	cmd := fmt.Sprintf("echo '%s' | base64 -d > %s && chmod +x %s", encoded, remotePath, remotePath)
	_, err := c.RunWithTimeout(cmd, 30*time.Second)
	return err
}

// ReadFile 读取远程文件内容。
func (c *Client) ReadFile(remotePath string) (string, error) {
	out, err := c.RunWithTimeout(fmt.Sprintf("cat %s 2>/dev/null || echo ''", remotePath), 15*time.Second)
	if err != nil {
		return "", err
	}
	return out, nil
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
