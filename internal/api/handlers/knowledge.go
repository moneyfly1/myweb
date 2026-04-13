package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// 用户端 - 获取知识库分类列表
func GetKnowledgeCategories(c *gin.Context) {
	db := database.GetDB()
	var categories []models.KnowledgeCategory
	db.Where("is_active = ?", true).Order("sort_order ASC, id ASC").Find(&categories)
	utils.SuccessResponse(c, http.StatusOK, "", categories)
}

// 用户端 - 获取分类下的文章列表
func GetKnowledgeArticles(c *gin.Context) {
	db := database.GetDB()
	categoryID := c.Query("category_id")
	keyword := c.Query("keyword")

	query := db.Model(&models.KnowledgeArticle{}).Where("is_active = ?", true)
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}
	if keyword != "" {
		sanitized := utils.SanitizeSearchKeyword(keyword)
		if sanitized != "" {
			escaped := utils.EscapeLikePattern(sanitized)
			query = query.Where("title LIKE ? OR content LIKE ?", "%"+escaped+"%", "%"+escaped+"%")
		}
	}

	var articles []models.KnowledgeArticle
	query.Preload("Category").Order("sort_order ASC, id DESC").Limit(200).Find(&articles)
	utils.SuccessResponse(c, http.StatusOK, "", articles)
}

// 用户端 - 获取文章详情
func GetKnowledgeArticle(c *gin.Context) {
	db := database.GetDB()
	id := c.Param("id")

	var article models.KnowledgeArticle
	if err := db.Preload("Category").First(&article, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "文章不存在", nil)
		return
	}

	db.Model(&article).Update("view_count", article.ViewCount+1)
	utils.SuccessResponse(c, http.StatusOK, "", article)
}

// 管理端 - 获取所有分类
func GetAdminKnowledgeCategories(c *gin.Context) {
	db := database.GetDB()
	var categories []models.KnowledgeCategory
	db.Order("sort_order ASC, id ASC").Find(&categories)
	utils.SuccessResponse(c, http.StatusOK, "", categories)
}

// 管理端 - 创建分类
func CreateKnowledgeCategory(c *gin.Context) {
	var cat models.KnowledgeCategory
	if err := c.ShouldBindJSON(&cat); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
		return
	}
	db := database.GetDB()
	if err := db.Create(&cat).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建失败", nil)
		return
	}
	utils.CreateAuditLogSimple(c, "create_knowledge_category", "knowledge_category", cat.ID, fmt.Sprintf("创建知识库分类: %s", cat.Name))
	utils.SuccessResponse(c, http.StatusOK, "创建成功", cat)
}

// 管理端 - 更新分类
func UpdateKnowledgeCategory(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var cat models.KnowledgeCategory
	if err := db.First(&cat, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "分类不存在", nil)
		return
	}
	if err := c.ShouldBindJSON(&cat); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
		return
	}
	db.Save(&cat)
	utils.CreateAuditLogSimple(c, "update_knowledge_category", "knowledge_category", cat.ID, fmt.Sprintf("更新知识库分类: %s", cat.Name))
	utils.SuccessResponse(c, http.StatusOK, "更新成功", cat)
}

// 管理端 - 删除分类
func DeleteKnowledgeCategory(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var cat models.KnowledgeCategory
	if err := db.First(&cat, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "分类不存在", nil)
		return
	}
	db.Delete(&models.KnowledgeCategory{}, id)
	db.Where("category_id = ?", id).Delete(&models.KnowledgeArticle{})
	utils.CreateAuditLogSimple(c, "delete_knowledge_category", "knowledge_category", cat.ID, fmt.Sprintf("删除知识库分类: %s (含文章)", cat.Name))
	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}

// 管理端 - 获取所有文章
func GetAdminKnowledgeArticles(c *gin.Context) {
	db := database.GetDB()
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	categoryID := c.Query("category_id")

	query := db.Model(&models.KnowledgeArticle{})
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	var total int64
	query.Count(&total)

	var articles []models.KnowledgeArticle
	query.Preload("Category").Order("sort_order ASC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&articles)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"list":  articles,
		"total": total,
		"page":  page,
	})
}

// 管理端 - 创建文章
func CreateKnowledgeArticle(c *gin.Context) {
	var article models.KnowledgeArticle
	if err := c.ShouldBindJSON(&article); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
		return
	}
	db := database.GetDB()
	if err := db.Create(&article).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建失败", nil)
		return
	}
	utils.CreateAuditLogSimple(c, "create_knowledge_article", "knowledge_article", article.ID, fmt.Sprintf("创建知识库文章: %s", article.Title))
	utils.SuccessResponse(c, http.StatusOK, "创建成功", article)
}

// 管理端 - 更新文章
func UpdateKnowledgeArticle(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var article models.KnowledgeArticle
	if err := db.First(&article, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "文章不存在", nil)
		return
	}
	if err := c.ShouldBindJSON(&article); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "参数错误", nil)
		return
	}
	db.Save(&article)
	utils.CreateAuditLogSimple(c, "update_knowledge_article", "knowledge_article", article.ID, fmt.Sprintf("更新知识库文章: %s", article.Title))
	utils.SuccessResponse(c, http.StatusOK, "更新成功", article)
}

// 管理端 - 删除文章
func DeleteKnowledgeArticle(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var article models.KnowledgeArticle
	if err := db.First(&article, id).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "文章不存在", nil)
		return
	}
	db.Delete(&models.KnowledgeArticle{}, id)
	utils.CreateAuditLogSimple(c, "delete_knowledge_article", "knowledge_article", article.ID, fmt.Sprintf("删除知识库文章: %s", article.Title))
	utils.SuccessResponse(c, http.StatusOK, "删除成功", nil)
}
