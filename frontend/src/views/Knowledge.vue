<template>
  <div class="list-container knowledge-container">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>知识库</span>
          <el-input
            v-model="keyword"
            placeholder="搜索文章..."
            clearable
            class="search-input"
            @keyup.enter="loadArticles"
          >
            <template #append>
              <el-button @click="loadArticles">
                <el-icon><Search /></el-icon>
              </el-button>
            </template>
          </el-input>
        </div>
      </template>

      <div class="knowledge-layout">
        <!-- 分类侧边栏 -->
        <div class="category-sidebar">
          <div
            class="category-item"
            :class="{ active: !selectedCategory }"
            @click="selectCategory(null)"
          >
            <el-icon><Folder /></el-icon>
            <span>全部分类</span>
          </div>
          <div
            v-for="cat in categories"
            :key="cat.id"
            class="category-item"
            :class="{ active: selectedCategory === cat.id }"
            @click="selectCategory(cat.id)"
          >
            <el-icon><component :is="cat.icon || 'Folder'" /></el-icon>
            <span>{{ cat.name }}</span>
          </div>
        </div>

        <!-- 文章列表 -->
        <div class="article-list-wrapper">
          <div v-if="loading" class="loading-state">
            <el-skeleton :rows="5" animated />
          </div>
          <div v-else-if="articles.length === 0" class="empty-state">
            <el-empty description="暂无文章" />
          </div>
          <div v-else class="article-list">
            <div
              v-for="article in articles"
              :key="article.id"
              class="article-card"
              @click="openArticle(article)"
            >
              <div class="article-header">
                <h3 class="article-title">{{ article.title }}</h3>
                <el-tag v-if="article.category" size="small" type="info">
                  {{ article.category.name }}
                </el-tag>
              </div>
              <p class="article-summary">
                {{ getSummary(article) }}
              </p>
              <div class="article-meta">
                <span><el-icon><View /></el-icon> {{ article.view_count || 0 }}</span>
                <span><el-icon><Clock /></el-icon> {{ formatDate(article.created_at) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 文章详情抽屉 -->
    <el-drawer
      v-model="articleVisible"
      :title="currentArticle?.title"
      :size="isMobile ? '100%' : '60%'"
      direction="rtl"
    >
      <div class="article-detail">
        <div class="article-detail-meta">
          <el-tag v-if="currentArticle?.category" size="small">
            {{ currentArticle.category.name }}
          </el-tag>
          <span class="meta-item">
            <el-icon><View /></el-icon>
            {{ currentArticle?.view_count || 0 }} 次浏览
          </span>
          <span class="meta-item">
            <el-icon><Clock /></el-icon>
            {{ formatDate(currentArticle?.created_at) }}
          </span>
        </div>
        <el-divider />
        <div class="article-content" v-html="sanitizeContent(currentArticle?.content)"></div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, Folder, View, Clock } from '@element-plus/icons-vue'
import { knowledgeAPI } from '@/utils/api'
import DOMPurify from 'dompurify'

const isMobile = ref(window.innerWidth <= 768)
const categories = ref([])
const articles = ref([])
const loading = ref(false)
const keyword = ref('')
const selectedCategory = ref(null)
const articleVisible = ref(false)
const currentArticle = ref(null)

const formatDate = (d) => {
  if (!d) return ''
  const date = new Date(d)
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

const getSummary = (article) => {
  if (article.summary?.String) return article.summary.String
  if (article.summary && typeof article.summary === 'string') return article.summary
  if (article.content) {
    const text = article.content.replace(/<[^>]*>/g, '')
    return text.substring(0, 120) + (text.length > 120 ? '...' : '')
  }
  return '暂无摘要'
}

const sanitizeContent = (html) => {
  if (!html) return ''
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'b', 'i', 'u', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'a', 'div', 'span', 'blockquote', 'pre', 'code', 'img', 'table', 'thead', 'tbody', 'tr', 'th', 'td'],
    ALLOWED_ATTR: ['href', 'target', 'style', 'class', 'id', 'src', 'alt', 'width', 'height'],
    ALLOW_DATA_ATTR: false
  })
}

const loadCategories = async () => {
  try {
    const res = await knowledgeAPI.getCategories()
    categories.value = res.data?.data || []
  } catch (e) {
    console.error('加载分类失败:', e)
  }
}

const loadArticles = async () => {
  loading.value = true
  try {
    const params = {}
    if (selectedCategory.value) params.category_id = selectedCategory.value
    if (keyword.value) params.keyword = keyword.value
    const res = await knowledgeAPI.getArticles(params)
    articles.value = res.data?.data || []
  } catch (e) {
    ElMessage.error('加载文章失败')
  } finally {
    loading.value = false
  }
}

const selectCategory = (categoryId) => {
  selectedCategory.value = categoryId
  loadArticles()
}

const openArticle = async (article) => {
  try {
    const res = await knowledgeAPI.getArticle(article.id)
    currentArticle.value = res.data?.data || article
    articleVisible.value = true
  } catch (e) {
    currentArticle.value = article
    articleVisible.value = true
  }
}

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}

onMounted(() => {
  // 并发加载分类和文章，提高页面加载速度
  Promise.all([
    loadCategories(),
    loadArticles()
  ])
  window.addEventListener('resize', handleResize)
})
</script>

<style scoped>
.knowledge-container {
  padding: 0;
}

.list-card {
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
}

.search-input {
  width: 280px;
}

.knowledge-layout {
  display: flex;
  gap: 20px;
  min-height: 500px;
}

.category-sidebar {
  width: 200px;
  flex-shrink: 0;
  border-right: 1px solid #ebeef5;
  padding-right: 20px;
}

.category-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  margin-bottom: 4px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.3s;
  color: #606266;
  font-size: 14px;
}

.category-item:hover {
  background-color: #f5f7fa;
  color: #409eff;
}

.category-item.active {
  background-color: #ecf5ff;
  color: #409eff;
  font-weight: 500;
}

.article-list-wrapper {
  flex: 1;
  min-width: 0;
}

.article-list {
  display: grid;
  gap: 16px;
}

.article-card {
  padding: 20px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s;
  background: #fff;
}

.article-card:hover {
  border-color: #409eff;
  box-shadow: 0 4px 12px rgba(64, 158, 255, 0.15);
  transform: translateY(-2px);
}

.article-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 12px;
}

.article-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  line-height: 1.5;
  flex: 1;
}

.article-summary {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #606266;
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.article-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #909399;
}

.article-meta span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.article-detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  margin-bottom: 16px;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 14px;
  color: #909399;
}

.article-content {
  font-size: 15px;
  line-height: 1.8;
  color: #303133;
}

.article-content :deep(h1),
.article-content :deep(h2),
.article-content :deep(h3),
.article-content :deep(h4) {
  margin: 24px 0 16px;
  font-weight: 600;
  color: #303133;
}

.article-content :deep(h1) { font-size: 24px; }
.article-content :deep(h2) { font-size: 20px; }
.article-content :deep(h3) { font-size: 18px; }
.article-content :deep(h4) { font-size: 16px; }

.article-content :deep(p) {
  margin: 12px 0;
}

.article-content :deep(ul),
.article-content :deep(ol) {
  margin: 12px 0;
  padding-left: 24px;
}

.article-content :deep(li) {
  margin: 8px 0;
}

.article-content :deep(code) {
  padding: 2px 6px;
  background: #f5f7fa;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 14px;
}

.article-content :deep(pre) {
  padding: 16px;
  background: #f5f7fa;
  border-radius: 6px;
  overflow-x: auto;
  margin: 16px 0;
}

.article-content :deep(blockquote) {
  margin: 16px 0;
  padding: 12px 16px;
  border-left: 4px solid #409eff;
  background: #ecf5ff;
  color: #606266;
}

.article-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 6px;
  margin: 16px 0;
}

.article-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 16px 0;
}

.article-content :deep(th),
.article-content :deep(td) {
  padding: 12px;
  border: 1px solid #ebeef5;
  text-align: left;
}

.article-content :deep(th) {
  background: #f5f7fa;
  font-weight: 600;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
  }

  .search-input {
    width: 100%;
  }

  .knowledge-layout {
    flex-direction: column;
    gap: 16px;
  }

  .category-sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid #ebeef5;
    padding-right: 0;
    padding-bottom: 16px;
    display: flex;
    overflow-x: auto;
    gap: 8px;
  }

  .category-item {
    flex-shrink: 0;
    white-space: nowrap;
    padding: 8px 12px;
    margin-bottom: 0;
  }

  .article-card {
    padding: 16px;
  }

  .article-title {
    font-size: 15px;
  }

  .article-summary {
    font-size: 13px;
  }

  .article-meta {
    font-size: 12px;
    gap: 12px;
  }

  .article-content {
    font-size: 14px;
  }
}
</style>
