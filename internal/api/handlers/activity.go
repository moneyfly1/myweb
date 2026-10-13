package handlers

import (
	"net/http"

	"cboard-go/internal/services/activity"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// GetActivitySummary 后台概览：在线 / 活跃 / 近期用过 的用户数。
//
// 口径说明（务必如实展示给管理员，避免把「订阅拉取」当成客户端心跳）：
//   - online：节点侧按用户上报了连接数（需要节点能区分用户；自建节点当前是单凭据设计，
//     因此该项在节点升级前恒为 0，前端应显示为"未启用"而不是"没人用"）；
//   - active：ActiveWindow（默认 15 分钟）内有过订阅拉取或节点上报；
//   - idle：IdleWindow（默认 24 小时）内有过信号。
func GetActivitySummary(c *gin.Context) {
	counts := activity.Counts()
	utils.SuccessResponse(c, http.StatusOK, "ok", gin.H{
		"online": counts.Online,
		"active": counts.Active,
		"idle":   counts.Idle,
		"windows": gin.H{
			"online_seconds":  int(activity.OnlineTTL.Seconds()),
			"active_minutes":  int(activity.ActiveWindow.Minutes()),
			"idle_hours":      int(activity.IdleWindow.Hours()),
		},
		"basis": gin.H{
			"online": "节点侧按用户上报的活跃连接数 > 0（秒级，需要节点支持按用户区分）",
			"active": "15 分钟内拉取过订阅（第三方客户端唯一可靠的活跃信号）或节点有上报",
			"idle":   "24 小时内有过上述任一信号",
		},
	})
}

// GetUserActivity 单个用户的使用状态（详情抽屉/排障用）
func GetUserActivity(c *gin.Context) {
	userID, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	st := activity.StateFor(userID)
	utils.SuccessResponse(c, http.StatusOK, "ok", st)
}

// parseUintParam 解析路径参数里的正整数 ID
func parseUintParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	var v uint64
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
			return 0, false
		}
		v = v*10 + uint64(ch-'0')
	}
	if raw == "" || v == 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
		return 0, false
	}
	return uint(v), true
}
