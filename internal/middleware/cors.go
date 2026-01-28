package middleware

import (
	"cboard-go/internal/core/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

func CORSMiddleware() gin.HandlerFunc {
	cfg := config.AppConfig
	origins := []string{"*"}
	if cfg != nil && len(cfg.CorsOrigins) > 0 {
		origins = cfg.CorsOrigins
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
