package handlers

import (
	"fmt"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
)

// 手工导入路径的第一层防御入口。
//
// 背景：采集链路（config_update.processFetchedNodes）已经做了协议白名单 / 必填字段 /
// 取值白名单 / 凭据规范化，但管理员手工导入走的是**另一条路**——「从 Clash 配置导入」、
// 「粘贴链接导入」、「单节点新建」都是直接 buildNodeModel 写库，完全绕过第一层。
//
// 后果（本次上线排查时确认的真实缺口）：
//   - 管理员粘贴进来的 naive 节点、坏 cipher、缺字段、无法解析的 2022 密钥会**静默进库**；
//   - 生成层的内核自检虽然会把它剔除（不会拖垮整份订阅），但管理员**拿不到任何反馈**，
//     只看到"导入 N 个成功"，而那个节点永远不可用；
//   - 凭据也不会被规范化，所以"URL 编码污染的 ss 2022 密钥"这类事故可以经手工导入复现。
//
// 这里让两条路共用同一套判定：先做凭据规范化，再做白名单+必填字段校验；
// 不通过就**不写库**，并把原因回给管理员（同时落进 node_validation_logs 供后台查看）。
//
// 返回 (原因码, 说明)；原因码为空表示通过，且 p 的凭据已被就地规范化。
func validateAndNormalizeImportedNode(p *config_update.ProxyNode, source string, record bool) (string, string) {
	if p == nil {
		return config_update.ReasonUnsupportedType, "空节点"
	}

	corrected, correctDetail := config_update.NormalizeCredentials(p)

	if err := config_update.ValidateProxyNode(p); err != nil {
		code, detail := config_update.ReasonKernelInvalid, err.Error()
		if ve, ok := err.(*config_update.NodeValidationError); ok {
			code, detail = ve.Code, ve.Detail
		}
		if record {
			recordImportedNodeEvent(models.NodeValidationDroppedAtIngest, source, p, code, detail)
		}
		return code, detail
	}

	if corrected && record {
		recordImportedNodeEvent(models.NodeValidationCorrectedAtIngest, source, p,
			config_update.ReasonCredentialURLDecoded, correctDetail)
	}
	return "", ""
}

// recordImportedNodeEvent 记录一条校验事件（失败只打日志，绝不影响导入主流程）
func recordImportedNodeEvent(event, source string, p *config_update.ProxyNode, code, detail string) {
	config_update.RecordNodeValidationEvents(database.GetDB(), []config_update.NodeValidationEvent{
		config_update.NewNodeValidationEvent(event, source, p, code, detail),
	})
}

// importedNodeReject 导入被拒的一条记录（回给管理员的脱敏原因）
type importedNodeReject struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// rejectFromNode 由节点与原因构造拒绝记录
func rejectFromNode(p *config_update.ProxyNode, code, detail string) importedNodeReject {
	r := importedNodeReject{Reason: code + ": " + detail}
	if p != nil {
		r.Name, r.Type = p.Name, p.Type
	}
	return r
}

// formatReject 转为单行文本（兼容旧的 errors []string 返回体）
func formatReject(r importedNodeReject) string {
	return fmt.Sprintf("[%s] %s: %s", r.Type, r.Name, r.Reason)
}

// rejectTexts 把拒绝明细转成单行文本列表（旧返回体兼容 + 前端可直接展示）
func rejectTexts(rs []importedNodeReject) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, formatReject(r))
	}
	return out
}
