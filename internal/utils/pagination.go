package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// PaginationParams 分页参数
type PaginationParams struct {
	Page int
	Size int
}

// ParsePagination 解析分页参数（支持 page/size 和 skip/limit）
func ParsePagination(c *gin.Context) PaginationParams {
	page := 1
	size := 20

	// 解析 page/size
	if pageStr := c.Query("page"); pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if sizeStr := c.Query("size"); sizeStr != "" {
		fmt.Sscanf(sizeStr, "%d", &size)
	}

	if skipStr := c.Query("skip"); skipStr != "" {
		var skip int
		fmt.Sscanf(skipStr, "%d", &skip)
		if page == 1 && size == 20 {
			page = (skip / size) + 1
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		fmt.Sscanf(limitStr, "%d", &limit)
		if size == 20 {
			size = limit
		}
	}

	// 验证和限制
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	return PaginationParams{Page: page, Size: size}
}

// GetOffset 计算偏移量
func (p PaginationParams) GetOffset() int {
	return (p.Page - 1) * p.Size
}
