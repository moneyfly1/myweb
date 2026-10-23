// Package dl 提供运维脚本与启动流程共用的下载/体积格式化实现。
//
// 为什么收成一处：GeoLite2（download_geoip）、DB-IP（download_dbip）与启动时的
// GeoIP 自动下载（cmd/server）此前各写一份 http.Get → 状态码 → os.Create → io.Copy。
// 其中两份脚本版**没有超时、也没有大小上限**，一个挂起的 CDN 或错误的 URL
// 就能让脚本永久卡住或把磁盘写满；server 版则额外做了路径校验与 200MB 上限。
// 统一到这里后，所有调用方都获得同一套超时/上限/临时文件语义。
package dl

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Options 下载参数。零值使用默认：30s 超时、200MB 上限、不处理 gzip。
type Options struct {
	Timeout  time.Duration // 整体超时（默认 30s）
	MaxBytes int64         // 最大字节数（默认 200MB）
	Gzip     bool          // 目标内容是 .gz 时自动解压
	Label    string        // 进度提示文案（可空）
}

const (
	defaultTimeout  = 30 * time.Second
	defaultMaxBytes = int64(200 << 20)
)

// Download 下载 url 到 dst，返回写入字节数。
//
// 语义要点：
//   - 目标路径先 Clean 并拒绝 ".."，避免脚本参数把文件写到意外位置；
//   - 先写 dst+".part"，成功后再 rename（避免中断留下半截文件被后续流程误用）；
//   - 超过 MaxBytes 立即中止并删除临时文件。
func Download(ctx context.Context, url, dst string, opts Options) (int64, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = defaultMaxBytes
	}
	cleanDst := filepath.Clean(dst)
	if cleanDst == "" || cleanDst == "." || strings.Contains(cleanDst, "..") {
		return 0, fmt.Errorf("不安全的输出路径: %s", dst)
	}

	if opts.Label != "" {
		fmt.Printf("%s... ", opts.Label)
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("构造请求失败: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	var src io.Reader = io.LimitReader(resp.Body, opts.MaxBytes+1)
	if opts.Gzip {
		gz, err := gzip.NewReader(src)
		if err != nil {
			return 0, fmt.Errorf("解压失败: %w", err)
		}
		defer gz.Close()
		src = gz
	}

	tmp := cleanDst + ".part"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("创建文件失败: %w", err)
	}
	written, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return 0, fmt.Errorf("保存文件失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return 0, fmt.Errorf("关闭文件失败: %w", closeErr)
	}
	if written > opts.MaxBytes {
		_ = os.Remove(tmp)
		return 0, fmt.Errorf("下载内容超过 %d 字节上限，已中止", opts.MaxBytes)
	}
	if err := os.Rename(tmp, cleanDst); err != nil {
		_ = os.Remove(tmp)
		return 0, fmt.Errorf("重命名临时文件失败: %w", err)
	}
	if opts.Label != "" {
		fmt.Printf("✅ %s\n", HumanSize(written))
	}
	return written, nil
}

// HumanSize 把字节数格式化成 B/KB/MB/GB（脚本输出与日志共用）。
func HumanSize(size int64) string {
	f := float64(size)
	switch {
	case f < 1024:
		return fmt.Sprintf("%.0f B", f)
	case f < 1024*1024:
		return fmt.Sprintf("%.2f KB", f/1024)
	case f < 1024*1024*1024:
		return fmt.Sprintf("%.2f MB", f/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", f/(1024*1024*1024))
	}
}
