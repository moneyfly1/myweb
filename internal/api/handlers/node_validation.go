package handlers

import (
	"net/http"
	"strconv"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/services/config_update"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// GetNodeValidationLogs 后台只读接口：最近 N 条节点校验日志（已丢弃 / 已剔除 / 已修正）
//
// GET /api/v1/admin/nodes/validation-logs?limit=100&event=dropped_at_ingest&type=ss&search=xxx
//
// 返回两类事件，供后台展示「已丢弃 / 已剔除」列表：
//   - dropped_at_ingest / corrected_at_ingest / renamed_at_ingest：第一层静态校验
//   - pruned_at_generate：第二层 mihomo -t 内核自检剔除
//
// 安全：本表只记录判定结论，从不写入 password/uuid（见 models.NodeValidationLog 注释），
// 因此可以安全地在后台明文展示。
func GetNodeValidationLogs(c *gin.Context) {
	db := database.GetDB()

	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}

	query := db.Model(&models.NodeValidationLog{})
	if e := c.Query("event"); e != "" && e != "all" {
		query = query.Where("event = ?", e)
	}
	if t := c.Query("type"); t != "" && t != "all" {
		query = query.Where("node_type = ?", t)
	}
	if search := c.Query("search"); search != "" {
		if kw := utils.SanitizeSearchKeyword(search); kw != "" {
			pattern := "%" + utils.EscapeLikePattern(kw) + "%"
			query = query.Where("node_name LIKE ? OR server LIKE ? OR reason LIKE ?", pattern, pattern, pattern)
		}
	}

	var logs []models.NodeValidationLog
	if err := query.Order("id DESC").Limit(limit).Find(&logs).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取节点校验日志失败", err)
		return
	}

	// 按事件类型汇总（供后台顶部概览）
	type eventCount struct {
		Event string `json:"event"`
		Total int64  `json:"total"`
	}
	var rows []eventCount
	if err := db.Model(&models.NodeValidationLog{}).
		Select("event, COUNT(*) as total").Group("event").Scan(&rows).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "统计节点校验日志失败", err)
		return
	}
	summary := make(map[string]int64, len(rows))
	var totalAll int64
	for _, r := range rows {
		summary[r.Event] = r.Total
		totalAll += r.Total
	}

	// 第二层（内核自检）的运行期状态：让后台一眼看出"内核是否在工作"，
	// 避免 mihomo 缺失/开关关闭导致静默降级却无人察觉。
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"list":    logs,
		"total":   len(logs),
		"all":     totalAll,
		"summary": summary,
		"kernel":  config_update.GetKernelSelfCheckStats(),
	})
}
