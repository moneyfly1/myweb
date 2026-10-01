package config_update

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// NodeAuditEntry 单条体检结果（脱敏：不含 password/uuid）
type NodeAuditEntry struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Cipher      string `json:"cipher,omitempty"`
	Server      string `json:"server,omitempty"`
	Port        int    `json:"port,omitempty"`
	SourceIndex int    `json:"source_index,omitempty"`
	Action      string `json:"action"`      // dropped | corrected | kernel-pruned
	ReasonCode  string `json:"reason_code"` // 机器可读原因码（用于聚合）
	Reason      string `json:"reason"`      // 人类可读说明
}

// ReasonCount 按原因码聚合
type ReasonCount struct {
	ReasonCode string `json:"reason_code"`
	Count      int    `json:"count"`
	Sample     string `json:"sample,omitempty"` // 一个脱敏样例（节点名）
}

// TypeCipherCount 按 协议类型 + cipher 聚合：「合法 type 但不合法 cipher/密钥」有多少
type TypeCipherCount struct {
	Type   string `json:"type"`
	Cipher string `json:"cipher,omitempty"`
	Count  int    `json:"count"`
}

// SourceCount 按来源订阅编号聚合
type SourceCount struct {
	SourceIndex int `json:"source_index"`
	Count       int `json:"count"`
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

	// 聚合视图：回答"这类坏节点现网还有多少个、集中在哪"
	DroppedByReason     []ReasonCount     `json:"dropped_by_reason"`
	DroppedByTypeCipher []TypeCipherCount `json:"dropped_by_type_cipher"`
	DroppedBySource     []SourceCount     `json:"dropped_by_source"`
	CorrectedByReason   []ReasonCount     `json:"corrected_by_reason"`
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
				SourceIndex: n.SourceIndex, Action: "dropped", ReasonCode: "empty-config", Reason: "config 为空"})
			report.Dropped++
			continue
		}
		var p ProxyNode
		if err := json.Unmarshal([]byte(*n.Config), &p); err != nil {
			report.Entries = append(report.Entries, NodeAuditEntry{ID: n.ID, Name: n.Name, Type: n.Type,
				SourceIndex: n.SourceIndex, Action: "dropped", ReasonCode: "invalid-config-json",
				Reason: "config JSON 解析失败: " + err.Error()})
			report.Dropped++
			continue
		}
		p.Name = n.Name

		if corrected, detail := NormalizeSS2022Key(&p); corrected {
			report.Corrected++
			report.Entries = append(report.Entries, NodeAuditEntry{ID: n.ID, Name: n.Name, Type: p.Type,
				Cipher: p.Cipher, Server: p.Server, Port: p.Port, SourceIndex: n.SourceIndex, Action: "corrected",
				ReasonCode: ReasonKeyURLDecoded, Reason: ReasonKeyURLDecoded + ": " + detail})
		}
		if err := ValidateProxyNode(&p); err != nil {
			ve, ok := err.(*NodeValidationError)
			if !ok {
				ve = &NodeValidationError{Code: ReasonKernelInvalid, Detail: err.Error()}
			}
			report.Entries = append(report.Entries, NodeAuditEntry{ID: n.ID, Name: n.Name, Type: p.Type,
				Cipher: p.Cipher, Server: p.Server, Port: p.Port, SourceIndex: n.SourceIndex, Action: "dropped",
				ReasonCode: ve.Code, Reason: ve.Code + ": " + ve.Detail})
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
		report.buildAggregates()
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
		report.buildAggregates()
		return report, nil
	}
	report.KernelError = fmt.Sprint(err)

	// 整体失败 → 折半二分定位坏节点
	good, bad := svc.divideConquer(survivors, info, ctx, b)
	for _, n := range bad {
		report.KernelPruned = append(report.KernelPruned, NodeAuditEntry{
			Name: n.Name, Type: n.Type, Cipher: n.Cipher, Server: n.Server, Port: n.Port,
			Action: "kernel-pruned", ReasonCode: ReasonKernelInvalid,
			Reason: ReasonKernelInvalid + ": 内核 mihomo -t 判定该节点使整份配置失效",
		})
	}
	report.KernelCalls = b.tests
	if finalOK, _ := svc.kernelTestNodes(info, good, ctx, b); finalOK {
		report.KernelOK = true
		report.KernelError = fmt.Sprintf("原始配置失败（%v），二分剔除 %d 个坏节点后通过", err, len(bad))
	}
	report.KernelDurationMS = time.Since(start).Milliseconds()
	report.buildAggregates()
	return report, nil
}

// buildAggregates 生成聚合视图（按原因 / 按 type+cipher / 按来源），供后台与报告直接引用。
// 只统计"被丢弃"与"被修正"的事件；内核剔除项单独成表。
func (r *NodeAuditReport) buildAggregates() {
	reasonCount := map[string]*ReasonCount{}
	typeCipher := map[string]*TypeCipherCount{}
	sourceCount := map[int]*SourceCount{}
	corrReason := map[string]*ReasonCount{}

	for _, e := range r.Entries {
		if e.Action == "dropped" {
			rc := reasonCount[e.ReasonCode]
			if rc == nil {
				rc = &ReasonCount{ReasonCode: e.ReasonCode, Sample: e.Name}
				reasonCount[e.ReasonCode] = rc
			}
			rc.Count++

			key := e.Type + "|" + e.Cipher
			tc := typeCipher[key]
			if tc == nil {
				tc = &TypeCipherCount{Type: e.Type, Cipher: e.Cipher}
				typeCipher[key] = tc
			}
			tc.Count++

			sc := sourceCount[e.SourceIndex]
			if sc == nil {
				sc = &SourceCount{SourceIndex: e.SourceIndex}
				sourceCount[e.SourceIndex] = sc
			}
			sc.Count++
		}
		if e.Action == "corrected" {
			cr := corrReason[e.ReasonCode]
			if cr == nil {
				cr = &ReasonCount{ReasonCode: e.ReasonCode, Sample: e.Name}
				corrReason[e.ReasonCode] = cr
			}
			cr.Count++
		}
	}

	for _, rc := range reasonCount {
		r.DroppedByReason = append(r.DroppedByReason, *rc)
	}
	for _, tc := range typeCipher {
		r.DroppedByTypeCipher = append(r.DroppedByTypeCipher, *tc)
	}
	for _, sc := range sourceCount {
		r.DroppedBySource = append(r.DroppedBySource, *sc)
	}
	for _, cr := range corrReason {
		r.CorrectedByReason = append(r.CorrectedByReason, *cr)
	}
	sort.Slice(r.DroppedByReason, func(i, j int) bool {
		return r.DroppedByReason[i].Count > r.DroppedByReason[j].Count
	})
	sort.Slice(r.DroppedByTypeCipher, func(i, j int) bool {
		return r.DroppedByTypeCipher[i].Count > r.DroppedByTypeCipher[j].Count
	})
	sort.Slice(r.DroppedBySource, func(i, j int) bool {
		return r.DroppedBySource[i].Count > r.DroppedBySource[j].Count
	})
	sort.Slice(r.CorrectedByReason, func(i, j int) bool {
		return r.CorrectedByReason[i].Count > r.CorrectedByReason[j].Count
	})
}
