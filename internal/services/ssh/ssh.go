package ssh

import (
	"bufio"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// SSHService SSH服务
type SSHService struct {
	DB *gorm.DB
}

// NewSSHService 创建SSH服务
func NewSSHService() *SSHService {
	return &SSHService{
		DB: nil,
	}
}

// GetSSHClient 获取SSH客户端连接
func (s *SSHService) GetSSHClient(server models.Server) (*ssh.Client, error) {
	password := server.Password
	if utils.IsEncrypted(password) {
		decrypted, err := utils.DecryptAES(password)
		if err == nil {
			password = decrypted
		}
	}

	config := &ssh.ClientConfig{
		User: server.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %w", err)
	}

	return client, nil
}

// ExecuteCommand 在远程服务器上执行命令
func (s *SSHService) ExecuteCommand(server models.Server, command string) (string, error) {
	client, err := s.GetSSHClient(server)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("执行失败: %w, 输出: %s", err, string(output))
	}

	return string(output), nil
}

// ExecuteCommandWithCallback 实时执行命令并调用回调函数处理输出
func (s *SSHService) ExecuteCommandWithCallback(server models.Server, command string, callback func(string)) (string, error) {
	client, err := s.GetSSHClient(server)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("获取标准输出管道失败: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("获取标准错误管道失败: %w", err)
	}

	var fullOutput strings.Builder
	done := make(chan bool, 2)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fullOutput.WriteString(line + "\n")
			if callback != nil {
				callback(line)
			}
		}
		done <- true
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fullOutput.WriteString(line + "\n")
			if callback != nil {
				callback(line)
			}
		}
		done <- true
	}()

	if err := session.Start(command); err != nil {
		return "", fmt.Errorf("启动命令失败: %w", err)
	}

	<-done
	<-done

	if err := session.Wait(); err != nil {
		return fullOutput.String(), fmt.Errorf("执行失败: %w", err)
	}

	return fullOutput.String(), nil
}

// UploadFile 上传文件
func (s *SSHService) UploadFile(server models.Server, localPath, remotePath string) error {
	readCmd := fmt.Sprintf("base64 -w 0 %s", localPath)
	localContent, err := s.ExecuteCommand(server, readCmd)
	if err != nil {
		return fmt.Errorf("读取本地文件失败: %w", err)
	}

	remoteDir := filepath.Dir(remotePath)
	s.ExecuteCommand(server, fmt.Sprintf("mkdir -p %s", remoteDir))

	writeCmd := fmt.Sprintf("echo '%s' | base64 -d > %s", localContent, remotePath)
	_, err = s.ExecuteCommand(server, writeCmd)
	if err != nil {
		return fmt.Errorf("上传文件失败: %w", err)
	}

	return nil
}

// InstallV2rayAgent 安装 v2ray-agent，带重试逻辑
func (s *SSHService) InstallV2rayAgent(server models.Server, domain string, logCallback func(string)) ([]string, string, error) {
	maxRetries := 2
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if logCallback != nil {
			logCallback(fmt.Sprintf("开始安装 v2ray-agent (尝试 %d/%d)", attempt, maxRetries))
		}

		links, output, err := s.installV2rayAgentOnce(server, domain, logCallback)
		if err == nil && len(links) > 0 {
			return links, output, nil
		}

		// 检查由于进程占用导致的失败
		if strings.Contains(output, "关闭失败") || strings.Contains(output, "占用") {
			if logCallback != nil {
				logCallback("检测到端口或进程占用，正在清理环境...")
			}
			s.ExecuteCommand(server, "pkill -9 xray; pkill -9 sing-box; pkill -9 nginx; systemctl stop nginx xray sing-box 2>/dev/null || true")
			time.Sleep(3 * time.Second)
		}
	}
	return nil, "", fmt.Errorf("安装完成但未解析到有效节点链接，请手动检查安装结果")
}

// installV2rayAgentOnce 执行一次安装尝试
func (s *SSHService) installV2rayAgentOnce(server models.Server, domain string, logCallback func(string)) ([]string, string, error) {
	// 1. 环境准备
	preCmd := `apt-get update && apt-get install -y wget expect curl || yum install -y wget expect curl`
	s.ExecuteCommand(server, preCmd)

	downloadCmd := `wget -P /root -N --no-check-certificate "https://raw.githubusercontent.com/mack-a/v2ray-agent/master/install.sh" && chmod 700 /root/install.sh`
	s.ExecuteCommand(server, downloadCmd)

	// 2. 自动化安装
	// 简化流程：第一次选择1，第二次选择1，输入域名，端口输入443，其余全部回车
	expectScript := fmt.Sprintf(`expect <<'EXPECT_EOF'
set timeout 1800
log_user 1
spawn bash /root/install.sh
set select_count 0
set install_complete 0
expect {
    "请选择" {
        incr select_count
        if {$select_count <= 2} {
            send "1\r"
        } else {
            send "\r"
        }
        exp_continue
    }
    "请输入要配置的域名" {
        send "%s\r"
        exp_continue
    }
    "域名:" {
        send "%s\r"
        exp_continue
    }
    "请输入端口" {
        send "443\r"
        exp_continue
    }
    "端口:" {
        send "443\r"
        exp_continue
    }
    "进度 12/12" {
        set install_complete 1
        exp_continue
    }
    "进度.*账号" {
        set install_complete 1
        exp_continue
    }
    "账号" {
        set install_complete 1
        exp_continue
    }
    "关闭失败" {
        exit 2
    }
    timeout {
        if {$install_complete == 0} {
            exit 1
        }
    }
    eof {
        if {$install_complete == 0} {
            exit 1
        }
    }
    default {
        send "\r"
        exp_continue
    }
}
EXPECT_EOF`, domain, domain)

	var output string
	var err error
	if logCallback != nil {
		output, err = s.ExecuteCommandWithCallback(server, expectScript, logCallback)
	} else {
		output, err = s.ExecuteCommand(server, expectScript)
	}

	if err != nil && !strings.Contains(err.Error(), "exit status") {
		return nil, output, err
	}

	// 3. 等待安装完成并提取链接
	if logCallback != nil {
		logCallback("安装脚本执行完成，等待服务启动并提取节点链接...")
	}

	// 等待 account.log 生成
	time.Sleep(30 * time.Second)

	// 多源提取链接：1. 从安装输出 2. 从 account.log 3. 从订阅地址
	var allLinks []string
	linkMap := make(map[string]bool)

	// 1. 从安装输出提取
	outputLinks := s.parseLinksFromText(output)
	for _, link := range outputLinks {
		if !linkMap[link] {
			allLinks = append(allLinks, link)
			linkMap[link] = true
		}
	}

	// 2. 从 account.log 提取（重试多次）
	maxRetries := 10
	for i := 0; i < maxRetries && len(allLinks) == 0; i++ {
		if logCallback != nil && i > 0 {
			logCallback(fmt.Sprintf("尝试从 account.log 提取链接 (第 %d/%d 次)...", i+1, maxRetries))
		}
		logLinks, _ := s.GetV2rayAgentLinks(server)
		for _, link := range logLinks {
			if !linkMap[link] {
				allLinks = append(allLinks, link)
				linkMap[link] = true
			}
		}
		if len(allLinks) > 0 {
			break
		}
		time.Sleep(5 * time.Second)
	}

	// 3. 如果还是没有，尝试从订阅地址提取
	if len(allLinks) == 0 {
		if logCallback != nil {
			logCallback("尝试从订阅地址提取节点链接...")
		}
		subLinks, _ := s.GetV2rayAgentLinksFromSubscription(server)
		for _, link := range subLinks {
			if !linkMap[link] {
				allLinks = append(allLinks, link)
				linkMap[link] = true
			}
		}
	}

	if logCallback != nil {
		logCallback(fmt.Sprintf("成功提取到 %d 个节点链接", len(allLinks)))
	}

	return allLinks, output, nil
}

// parseLinksFromText 使用正则表达式从文本中精准提取链接
func (s *SSHService) parseLinksFromText(text string) []string {
	// 匹配所有协议类型：vless://, vmess://, trojan://, ss://, hysteria2://, tuic://, naive+https://, anytls://
	// 使用更精确的正则表达式，匹配完整的链接（包括参数和锚点）
	patterns := []*regexp.Regexp{
		// vless:// 链接
		regexp.MustCompile(`vless://[^\s'"\(\)#]+#[^\s'"\(\)]*`),
		// vmess:// 链接（base64编码）
		regexp.MustCompile(`vmess://[A-Za-z0-9+/=]+`),
		// trojan:// 链接
		regexp.MustCompile(`trojan://[^\s'"\(\)#]+#[^\s'"\(\)]*`),
		// ss:// 链接
		regexp.MustCompile(`ss://[^\s'"\(\)#]+#[^\s'"\(\)]*`),
		// hysteria2:// 链接
		regexp.MustCompile(`hysteria2://[^\s'"\(\)#]+#[^\s'"\(\)]*`),
		// tuic:// 链接
		regexp.MustCompile(`tuic://[^\s'"\(\)#]+#[^\s'"\(\)]*`),
		// naive+https:// 链接
		regexp.MustCompile(`naive\+https://[^\s'"\(\)#]+#[^\s'"\(\)]*`),
		// anytls:// 链接
		regexp.MustCompile(`anytls://[^\s'"\(\)#]+#[^\s'"\(\)]*`),
	}

	uniqueLinks := make(map[string]bool)
	var result []string

	for _, re := range patterns {
		matches := re.FindAllString(text, -1)
		for _, link := range matches {
			link = strings.TrimSpace(link)
			// 过滤掉包含 api.qrserver.com 的二维码链接，只保留原始节点链接
			if !strings.Contains(link, "api.qrserver.com") &&
				!strings.Contains(link, "qrserver.com") &&
				!uniqueLinks[link] &&
				len(link) > 10 { // 确保链接长度合理
				uniqueLinks[link] = true
				result = append(result, link)
			}
		}
	}
	return result
}

// GetV2rayAgentLinks 从远程服务器的 account.log 提取
func (s *SSHService) GetV2rayAgentLinks(server models.Server) ([]string, error) {
	cmd := "cat /etc/v2ray-agent/account.log 2>/dev/null || true"
	out, _ := s.ExecuteCommand(server, cmd)
	return s.parseLinksFromText(out), nil
}

func (s *SSHService) GetV2rayAgentLinksFromSubscription(server models.Server) ([]string, error) {
	subExpect := `expect <<'EXPECT_EOF'
set timeout 60
spawn bash /root/install.sh
expect {
    "请选择" { send "7\r"; exp_continue }
    "订阅" { send "\r"; exp_continue }
    eof
}
EXPECT_EOF`
	out, _ := s.ExecuteCommand(server, subExpect)

	// 提取订阅链接
	re := regexp.MustCompile(`https?://[^\s]+`)
	subURL := re.FindString(out)

	if subURL == "" {
		return nil, fmt.Errorf("未找到订阅地址")
	}

	// 获取订阅内容
	content, err := s.ExecuteCommand(server, fmt.Sprintf("curl -s -L '%s'", subURL))
	if err != nil {
		return nil, err
	}

	// Base64 解码订阅内容
	decoded, _ := s.decodeBase64(strings.TrimSpace(content))
	return s.parseLinksFromText(decoded), nil
}

func (s *SSHService) decodeBase64(encoded string) (string, error) {
	encoded = strings.TrimSpace(encoded)
	// 自动补齐 Base64 填充符
	if len(encoded)%4 != 0 {
		encoded += strings.Repeat("=", 4-len(encoded)%4)
	}
	d, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		d, err = base64.URLEncoding.DecodeString(encoded)
	}
	return string(d), err
}
