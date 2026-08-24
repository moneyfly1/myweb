<template>
  <div class="list-container knowledge-container">
    <div class="breadcrumb">首页 / 知识库</div>
    <div class="page-header">
      <div class="page-title">
        <h1>知识库</h1>
      </div>
      <div class="actions">
        <el-button @click="selectCategory(null)">
          全部分类
        </el-button>
      </div>
    </div>
    <div class="card list-filter-card knowledge-filter-card">
      <div class="card-body knowledge-filter-body">
        <el-form :inline="true" class="knowledge-filter-form list-filter-form">
          <el-form-item label="搜索">
            <el-input
              v-model="keyword"
              placeholder="搜索文章标题或关键词"
              clearable
              class="knowledge-search-input"
              @keyup.enter="searchArticles"
              @clear="searchArticles"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item label="分类筛选">
            <el-select v-model="selectedCategory" placeholder="全部分类" clearable class="knowledge-category-select" @change="searchArticles">
              <el-option label="全部分类" :value="null" />
              <el-option
                v-for="cat in categories"
                :key="cat.id"
                :label="cat.name"
                :value="cat.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item class="knowledge-filter-actions">
            <el-button type="primary" @click="searchArticles">搜索</el-button>
            <el-button :disabled="!keyword && !selectedCategory" @click="resetFilters">重置</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
    <div class="card list-card">
      <div class="card-body">
        <LoadingState
          v-if="loading"
          text="正在加载知识库文章..."
          :size="32"
          class="knowledge-loading-state"
        />
        <EmptyState
          v-else-if="articles.length === 0"
          title="暂无文章"
          description="当前分类或搜索条件下没有可展示的知识库文章。"
          :icon-size="56"
          class="knowledge-empty-state"
        />
        <div v-else class="article-list">
          <div
            v-for="article in articles"
            :key="article.id"
            class="article-card knowledge-item"
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
              <span><el-icon><Clock /></el-icon> 更新于 {{ formatDate(article.created_at) }}</span>
            </div>
          </div>
        </div>
        <div v-if="total > 0" class="knowledge-pagination">
          <PaginationBar
            v-model:current-page="page"
            v-model:page-size="pageSize"
            :total="total"
            :page-sizes="[12, 24, 48, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            mobile-layout="sizes, prev, pager, next, jumper"
            @current-change="handlePageChange"
            @size-change="handleSizeChange"
          />
        </div>
      </div>
    </div>

    <!-- 文章详情抽屉 -->
    <AppDrawer
      v-model="articleVisible"
      :title="currentArticle?.title"
      size="720px"
      mobile-size="100%"
      class="knowledge-article-drawer"
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
    </AppDrawer>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { Search, Folder, View, Clock, Document, Reading, Files, Setting, Star, InfoFilled, QuestionFilled, Notebook } from '@element-plus/icons-vue'
import AppDrawer from '@/components/AppDrawer.vue'
import EmptyState from '@/components/EmptyState.vue'
import LoadingState from '@/components/LoadingState.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import { knowledgeAPI } from '@/utils/api'
import { sanitizeArticleHtml } from '@/utils/sanitizeHtml'

const iconMap = { Search, Folder, View, Clock, Document, Reading, Files, Setting, Star, InfoFilled, QuestionFilled, Notebook }
const resolveIcon = (name) => iconMap[name] || Folder

const categories = ref([])
const articles = ref([])
const loading = ref(false)
const keyword = ref('')
const selectedCategory = ref(null)
const articleVisible = ref(false)
const currentArticle = ref(null)
const page = ref(1)
const pageSize = ref(12)
const total = ref(0)

const formatDate = (d) => {
  if (!d) return ''
  const date = new Date(String(d).replace(/-/g, '/'))
  if (isNaN(date.getTime())) return ''
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

const sanitizeContent = sanitizeArticleHtml

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
    const params = { page: page.value, page_size: pageSize.value }
    if (selectedCategory.value) params.category_id = selectedCategory.value
    if (keyword.value) params.keyword = keyword.value
    const res = await knowledgeAPI.getArticles(params)
    const data = res.data?.data || {}
    articles.value = data.items || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error('加载文章失败')
  } finally {
    loading.value = false
  }
}

// 搜索/切换分类时回到第一页
const searchArticles = () => {
  page.value = 1
  loadArticles()
}

const handlePageChange = (p) => {
  page.value = p
  loadArticles()
}

const handleSizeChange = (s) => {
  pageSize.value = s
  page.value = 1
  loadArticles()
}

const selectCategory = (categoryId) => {
  selectedCategory.value = categoryId
  page.value = 1
  loadArticles()
}

const resetFilters = () => {
  keyword.value = ''
  selectedCategory.value = null
  page.value = 1
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

onMounted(() => {
  Promise.all([
    loadCategories(),
    loadArticles()
  ])
})
</script>

<style scoped>
.knowledge-container {
  padding: 0;
}

.breadcrumb {
  margin-bottom: 12px;
  color: #606266;
  font-size: 13px;
}

.card {
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  overflow: hidden;
}

.card-body {
  padding: 16px;
}

.knowledge-filter-card {
  margin-bottom: 14px;
}

.knowledge-filter-body {
  padding: 16px;
}

.knowledge-pagination {
  display: flex;
  justify-content: center;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid #ebeef5;
}

.knowledge-filter-form {
  display: grid;
  grid-template-columns: minmax(240px, 1.4fr) minmax(180px, 0.8fr) minmax(150px, max-content);
  align-items: end;
  gap: 12px;
  width: 100%;
}

.knowledge-filter-form :deep(.el-form-item) {
  margin: 0;
  min-width: 0;
}

.knowledge-filter-form :deep(.el-form-item__label) {
  color: #606266;
  font-weight: 600;
}

.knowledge-search-input {
  width: 100%;
  min-width: 0;
}

.knowledge-category-select {
  width: 100%;
  min-width: 0;
}

.knowledge-filter-actions {
  justify-self: end;
}

.knowledge-filter-actions :deep(.el-form-item__content) {
  display: flex;
  flex-wrap: nowrap;
  gap: 8px;
}

@media (max-width: 1100px) {
  .knowledge-filter-form {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .knowledge-filter-actions {
    justify-self: start;
  }
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  padding: 16px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 8px;
}

.page-title h1 {
  margin: 0;
  color: #303133;
  font-size: 22px;
  line-height: 1.25;
}

.page-title p {
  margin: 6px 0 0;
  color: #606266;
  line-height: 1.5;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.knowledge-empty-state {
  min-height: 360px;
  padding: 48px 16px;
}

.article-list {
  display: grid;
  gap: 0;
}

.article-card {
  padding: 14px 0;
  border-bottom: 1px solid #ebeef5;
  border-radius: 0;
  cursor: pointer;
  transition: border-color 0.16s ease, background-color 0.16s ease;
  background: #fff;
}

.article-card:last-child {
  border-bottom: 0;
}

.article-card:hover {
  background: #fbfdff;
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
  line-height: 1.55;
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
  flex-wrap: wrap;
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
  line-height: 1.75;
  color: #303133;
  overflow-wrap: anywhere;
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
  border: 1px solid #dcdfe6;
  text-align: left;
}

.article-content :deep(th) {
  background: #f5f7fa;
  font-weight: 600;
}

.knowledge-article-drawer :deep(.app-drawer__body) {
  background: #fff;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .knowledge-filter-form {
    display: grid;
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .knowledge-search-input,
  .knowledge-category-select,
  .knowledge-filter-actions,
  .knowledge-filter-actions :deep(.el-form-item__content),
  .knowledge-filter-actions :deep(.el-button) {
    width: 100%;
  }

  .card-header {
    flex-direction: column;
    align-items: stretch;
  }

  .article-card {
    padding: 14px 0;
  }

  .article-header {
    flex-direction: column;
    gap: 8px;
  }

  .article-title {
    font-size: 15px;
  }

  .article-summary {
    font-size: 13px;
  }

  .article-meta {
    font-size: 12px;
    gap: 10px;
  }

  .article-content {
    font-size: 14px;
  }

  .article-content :deep(pre) {
    padding: 12px;
  }

  .article-content :deep(th),
  .article-content :deep(td) {
    padding: 10px;
  }
}
</style>
