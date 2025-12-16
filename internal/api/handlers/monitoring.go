package handlers

import (
	"net/http"
	"runtime"

	"cboard-go/internal/core/database"
	"github.com/gin-gonic/gin"
)

// GetSystemInfo 获取系统信息
func GetSystemInfo(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info := map[string]interface{}{
		"memory": map[string]interface{}{
			"allocated":      m.Alloc,
			"total_allocated": m.TotalAlloc,
			"sys":            m.Sys,
			"num_gc":         m.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
		"cpu_count":  runtime.NumCPU(),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    info,
	})
}

// GetDatabaseStats 获取数据库统计
func GetDatabaseStats(c *gin.Context) {
	db := database.GetDB()

	stats := map[string]interface{}{
		"status": "connected",
	}

	// 测试连接
	sqlDB, err := db.DB()
	if err != nil {
		stats["status"] = "error"
		stats["error"] = err.Error()
	} else {
		if err := sqlDB.Ping(); err != nil {
			stats["status"] = "disconnected"
			stats["error"] = err.Error()
		} else {
			stats["max_open_conns"] = sqlDB.Stats().MaxOpenConnections
			stats["open_conns"] = sqlDB.Stats().OpenConnections
			stats["in_use"] = sqlDB.Stats().InUse
			stats["idle"] = sqlDB.Stats().Idle
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

