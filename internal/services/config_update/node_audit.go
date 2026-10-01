package config_update

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// NodeAuditEntry 单条体检结果（脱敏：不含 password/uuid）
type NodeAuditEntry struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Cipher string `json:"cipher,omitempty"`
	Server string `json:"server,omitempty"`
	Port   int    `json:"port,omitempty"`
	Action string `json:"action"` // dropped | corrected | kernel-pruned
	Reason string `json:"reason"`
}

// NodeAuditReport 全库节点体检报告
type NodeAuditReport struct {
	Total       int              `json:"total"`
	Kept        int              `json:"kept"`
	Dropped     int              `json:"dropped"`
	Corrected   int              `json:"corrected"`
	Entries     []NodeAuditEntry `json:"entries"`
	ConfigBytes int              `json:"config_bytes"`

	KernelBin        string           `json:"kernel_bin"`
	KernelRan        bool             `json:"kernel_ran"`
	KernelOK         bool             `json:"kernel_ok"`
	KernelError      string           `json:"kernel_error,omitempty"`
	KernelDurationMS int64            `json:"kernel_duration_ms"`
	KernelCalls      int              `json:"kernel_calls"`
	KernelPruned     []NodeAuditEntry `json:"kernel_pruned"`
}

// AuditActiveNodes 全库活跃节点体检（只读）：
//  1. 逐节点跑第一层静态校验（含 ss 2022 密钥编码修正），列出会被丢弃/修正的节点；
//  2. 用「存活节点」生成一份 Clash 配置，跑真内核 mihomo -t；
//  3. 若整体失败，折半二分定位到具体坏节点（回答"今天用户是否已经在踩雷"）。
//
// 绝不写库、绝不删数据：只做 SELECT 与只读判别。
func AuditActiveNodes(db *gorm.DB, kernelBin string) (*NodeAuditReport, error) {
	svc := &ConfigUpdateService{db: db}
	report := &NodeAuditReport{KernelBin: kernelBin, Entries: []NodeAuditEntry{}, KernelPruned: []NodeAuditEntry{}}

	var nodes []models.Node
	if err := db.Where("is_active = ?", true).Order("order_index ASC, created_at ASC").Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("读取活跃节点失败: %w", err)
	}
	report.Total = len(nodes)

	seen := make(map[string]bool)
	var survivors []*ProxyNode
	for _, n := range nodes {
		if n.Config == nil || *n.Config == "" {
			report.Entries = append(report.Entries, NodeAuditEntry{ID: n.ID, Name: n.Name, Type: n.Type,
				Action: "dropped", Reason: "config 为空"})
			report.Dropped++
			continue
		}
		var p ProxyNode
		if err := json.Unmarshal([]byte(*n.Config), &p); err != nil {
			report.Entries = append(report.Entries, NodeAuditEntry{ID: n.ID, Name: n.Name, Type: n.Type,
				Action: "dropped", Reason: "config JSON 解析失败: " + err.Error()})
			report.Dropped++
			continue
		}
		p.Name = n.Name

		if corrected, detail := NormalizeSS2022Key(&p); corrected {
			report.Corrected++
			report.Entries = append(report.Entries, NodeAuditEntry{ID: n.ID, Name: n.Name, Type: p.Type,
				Cipher: p.Cipher, Server: p.Server, Port: p.Port, Action: "corrected",
				Reason: ReasonKeyURLDecoded + ": " + detail})
		}
		if err := ValidateProxyNode(&p); err != nil {
			ve, ok := err.(*NodeValidationError)
			if !ok {
				ve = &NodeValidationError{Code: ReasonKernelInvalid, Detail: err.Error()}
			}
			report.Entries = append(report.Entries, NodeAuditEntry{ID: n.ID, Name: n.Name, Type: p.Type,
				Cipher: p.Cipher, Server: p.Server, Port: p.Port, Action: "dropped",
				Reason: ve.Code + ": " + ve.Detail})
			report.Dropped++
			continue
		}
		key := fmt.Sprintf("%s:%s:%d:%s", p.Type, p.Server, p.Port, p.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		survivors = append(survivors, &p)
	}
	report.Kept = len(survivors)

	// 生成配置并跑真内核
	ctx := &SubscriptionContext{Status: StatusNormal}
	info := svc.addInfoNodes(nil, ctx)
	cfg := svc.generateClashYAML(append(append([]*ProxyNode{}, info...), survivors...), ctx)
	report.ConfigBytes = len(cfg)

	if kernelBin == "" {
		return report, nil
	}
	if err := os.Setenv("MF_MIHOMO_BIN", kernelBin); err != nil {
		return report, fmt.Errorf("设置内核路径失败: %w", err)
	}
	if mihomoBinaryPath() == "" {
		return report, fmt.Errorf("内核二进制不可用: %s", kernelBin)
	}
	report.KernelRan = true

	start := time.Now()
	b := newKernelBudget(10 * time.Minute)
	ok, err := svc.kernelTestNodes(info, survivors, ctx, b)
	if kernelTestHook != nil {
		// 体检工具不使用测试 hook
		kernelTestHook = nil
	}
	report.KernelDurationMS = time.Since(start).Milliseconds()
	report.KernelCalls = b.tests

	if ok {
		report.KernelOK = true
		return report, nil
	}
	report.KernelError = fmt.Sprint(err)

	// 整体失败 → 折半二分定位坏节点
	good, bad := svc.divideConquer(survivors, info, ctx, b)
	for _, n := range bad {
		report.KernelPruned = append(report.KernelPruned, NodeAuditEntry{
			Name: n.Name, Type: n.Type, Cipher: n.Cipher, Server: n.Server, Port: n.Port,
			Action: "kernel-pruned", Reason: ReasonKernelInvalid + ": 内核 mihomo -t 判定该节点使整份配置失效",
		})
	}
	report.KernelCalls = b.tests
	if finalOK, _ := svc.kernelTestNodes(info, good, ctx, b); finalOK {
		report.KernelOK = true
		report.KernelError = fmt.Sprintf("原始配置失败（%v），二分剔除 %d 个坏节点后通过", err, len(bad))
	}
	report.KernelDurationMS = time.Since(start).Milliseconds()
	return report, nil
}
