package handlers

import (
	"net/http"

	"cboard-go/internal/services/geoip"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
)

// LookupIPLocation 查询 IP 地理位置（带缓存）
func LookupIPLocation(c *gin.Context) {
	ip := c.Query("ip")
	if ip == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "IP 地址不能为空", nil)
		return
	}

	// 使用带缓存的查询
	location := geoip.GetLocationSimpleWithCache(ip)

	if location == "" {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"ip":       ip,
			"location": nil,
			"message":  "无法获取地理位置信息",
		})
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"ip":       ip,
		"location": location,
	})
}

// BatchLookupIPLocation 批量查询 IP 地理位置
func BatchLookupIPLocation(c *gin.Context) {
	var req struct {
		IPs []string `json:"ips" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	if len(req.IPs) > 100 {
		utils.ErrorResponse(c, http.StatusBadRequest, "一次最多查询 100 个 IP", nil)
		return
	}

	results := make([]gin.H, 0, len(req.IPs))
	for _, ip := range req.IPs {
		location := geoip.GetLocationSimpleWithCache(ip)
		results = append(results, gin.H{
			"ip":       ip,
			"location": location,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"results": results,
		"total":   len(results),
	})
}
