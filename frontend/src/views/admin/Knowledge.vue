<template>
  <div class="list-container">
    <el-card class="list-card">
      <template #header>
        <div class="card-header">
          <span>知识库管理</span>
          <div class="header-actions">
            <el-button type="primary" @click="showCategoryDrawer()">
              <el-icon><FolderAdd /></el-icon>
              新建分类
            </el-button>
            <el-button type="success" @click="showArticleDrawer()">
              <el-icon><DocumentAdd /></el-icon>
              新建文章
            </el-button>
          </div>
        </div>
      </template>

      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <el-tab-pane label="文章管理" name="articles">
          <div class="filter-bar">
            <el-select v-model="articleFilter.category_id" placeholder="筛选分类" clearable style="width: 200px" @change="loadArticles">
              <el-option v-for="cat in categories" :key="cat.id" :label="cat.name" :value="cat.id" />
            </el-select>
            <el-input v-model="articleFilter.keyword" placeholder="搜索标题..." clearable style="width: 240px" @keyup.enter="loadArticles">
              <template #append>
                <el-button @click="loadArticles"><el-icon><Search /></el-icon></el-button>
              </template>
            </el-input>
          </div>

          <el-table :data="articles" v-loading="loading" stripe border>
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip />
            <el-table-column label="分类" width="120">
              <template #default="{ row }">
                <el-tag v-if="row.category" size="small">{{ row.category.name }}</el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="view_count" label="浏览" width="80" align="center" />
            <el-table-column prop="sort_order" label="排序" width="80" align="center" />
            <el-table-column label="状态" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="row.is_active ? 'success' : 'info'" size="small">
                  {{ row.is_active ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="160">
              <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="showArticleDrawer(row)">编辑</el-button>
                <el-button size="small" type="danger" @click="deleteArticle(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="pagination-wrapper">
            <el-pagination
              v-model:current-page="articlePagination.page"
              v-model:page-size="articlePagination.page_size"
              :total="articlePagination.total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="loadArticles"
              @current-change="loadArticles"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane label="分类管理" name="categories">
          <el-table :data="categories" stripe border>
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="icon" label="图标" width="150">
              <template #default="{ row }">
                <el-icon><component :is="row.icon || 'Folder'" /></el-icon>
                <span style="margin-left: 8px">{{ row.icon || 'Folder' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="sort_order" label="排序" width="100" align="center" />
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="row.is_active ? 'success' : 'info'" size="small">
                  {{ row.is_active ? '启用' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="160">
              <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }">
                <el-button size="small" @click="showCategoryDrawer(row)">编辑</el-button>
                <el-button size="small" type="danger" @click="deleteCategory(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 分类抽屉 -->
    <el-drawer
      v-model="catDrawerVisible"
      :title="catForm.id ? '编辑分类' : '新建分类'"
      :size="isMobile ? '100%' : '500px'"
      direction="rtl"
      :lock-scroll="false"
    >
      <el-form :model="catForm" label-width="80px" :rules="catRules" ref="catFormRef">
        <el-form-item label="名称" prop="name">
          <el-input v-model="catForm.name" placeholder="请输入分类名称" />
        </el-form-item>
        <el-form-item label="图标" prop="icon">
          <el-input v-model="catForm.icon" placeholder="如: Folder, Document, Star" />
          <div style="margin-top: 8px; font-size: 12px; color: #909399">
            常用图标: Folder, Document, Star, Reading, Guide, QuestionFilled
          </div>
        </el-form-item>
        <el-form-item label="排序" prop="sort_order">
          <el-input-number v-model="catForm.sort_order" :min="0" :max="9999" />
          <div style="margin-top: 8px; font-size: 12px; color: #909399">
            数字越小越靠前
          </div>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="catForm.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="drawer-footer">
          <el-button @click="catDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveCategory">保存</el-button>
        </div>
      </template>
    </el-drawer>

    <!-- 文章抽屉 -->
    <el-drawer
      v-model="articleDrawerVisible"
      :title="articleForm.id ? '编辑文章' : '新建文章'"
      :size="isMobile ? '100%' : '70%'"
      direction="rtl"
      :lock-scroll="false"
    >
      <el-form :model="articleForm" label-width="80px" :rules="articleRules" ref="articleFormRef">
        <el-form-item label="标题" prop="title">
          <el-input v-model="articleForm.title" placeholder="请输入文章标题" />
        </el-form-item>
        <el-form-item label="分类" prop="category_id">
          <el-select v-model="articleForm.category_id" placeholder="请选择分类" style="width: 100%">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="摘要">
          <el-input
            v-model="articleForm.summary"
            type="textarea"
            :rows="3"
            placeholder="请输入文章摘要（选填）"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input
            v-model="articleForm.content"
            type="textarea"
            :rows="15"
            placeholder="请输入文章内容，支持HTML格式"
          />
          <div style="margin-top: 8px; font-size: 12px; color: #909399">
            支持HTML标签，如 &lt;h2&gt;、&lt;p&gt;、&lt;ul&gt;、&lt;li&gt;、&lt;strong&gt; 等
          </div>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="articleForm.sort_order" :min="0" :max="9999" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="articleForm.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="drawer-footer">
          <el-button @click="articleDrawerVisible = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveArticle">保存</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { FolderAdd, DocumentAdd, Search, Folder } from '@element-plus/icons-vue'
import { knowledgeAPI } from '@/utils/api'

const isMobile = ref(window.innerWidth <= 768)
const activeTab = ref('articles')
const loading = ref(false)
const saving = ref(false)

const categories = ref([])
const articles = ref([])

const articleFilter = reactive({
  category_id: null,
  keyword: ''
})

const articlePagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

const catDrawerVisible = ref(false)
const catFormRef = ref()
const catForm = ref({
  name: '',
  icon: '',
  sort_order: 0,
  is_active: true
})

const catRules = {
  name: [{ required: true, message: '请输入分类名称', trigger: 'blur' }]
}

const articleDrawerVisible = ref(false)
const articleFormRef = ref()
const articleForm = ref({
  title: '',
  category_id: null,
  content: '',
  summary: '',
  sort_order: 0,
  is_active: true
})

const articleRules = {
  title: [{ required: true, message: '请输入文章标题', trigger: 'blur' }],
  category_id: [{ required: true, message: '请选择分类', trigger: 'change' }],
  content: [{ required: true, message: '请输入文章内容', trigger: 'blur' }]
}

const formatDate = (d) => {
  if (!d) return ''
  const date = new Date(d)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const loadCategories = async () => {
  try {
    const res = await knowledgeAPI.getAdminCategories()
    categories.value = res.data?.data || []
  } catch (e) {
    console.error('加载分类失败:', e)
  }
}

const loadArticles = async () => {
  loading.value = true
  try {
    const params = {
      page: articlePagination.page,
      page_size: articlePagination.page_size
    }
    if (articleFilter.category_id) params.category_id = articleFilter.category_id
    if (articleFilter.keyword) params.keyword = articleFilter.keyword

    const res = await knowledgeAPI.getAdminArticles(params)
    const data = res.data?.data || {}
    articles.value = data.list || []
    articlePagination.total = data.total || 0
  } catch (e) {
    ElMessage.error('加载文章失败')
  } finally {
    loading.value = false
  }
}

const handleTabChange = (tab) => {
  if (tab === 'categories') {
    loadCategories()
  } else {
    loadArticles()
  }
}

const showCategoryDrawer = (row) => {
  if (row) {
    catForm.value = { ...row }
  } else {
    catForm.value = {
      name: '',
      icon: 'Folder',
      sort_order: 0,
      is_active: true
    }
  }
  catDrawerVisible.value = true
}

const saveCategory = async () => {
  if (!catFormRef.value) return
  await catFormRef.value.validate()

  saving.value = true
  try {
    if (catForm.value.id) {
      await knowledgeAPI.updateCategory(catForm.value.id, catForm.value)
      ElMessage.success('更新成功')
    } else {
      await knowledgeAPI.createCategory(catForm.value)
      ElMessage.success('创建成功')
    }
    catDrawerVisible.value = false
    loadCategories()
    if (activeTab.value === 'articles') {
      loadArticles()
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const deleteCategory = async (id) => {
  await ElMessageBox.confirm('删除分类将同时删除该分类下的所有文章，确定删除？', '确认删除', {
    type: 'warning'
  })
  try {
    await knowledgeAPI.deleteCategory(id)
    ElMessage.success('删除成功')
    loadCategories()
    if (activeTab.value === 'articles') {
      loadArticles()
    }
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '删除失败')
  }
}

const showArticleDrawer = (row) => {
  if (row) {
    articleForm.value = {
      ...row,
      summary: row.summary?.String || row.summary || ''
    }
  } else {
    articleForm.value = {
      title: '',
      category_id: categories.value[0]?.id || null,
      content: '',
      summary: '',
      sort_order: 0,
      is_active: true
    }
  }
  articleDrawerVisible.value = true
}

const saveArticle = async () => {
  if (!articleFormRef.value) return
  await articleFormRef.value.validate()

  saving.value = true
  try {
    const data = { ...articleForm.value }
    if (typeof data.summary === 'string') {
      data.summary = { String: data.summary, Valid: !!data.summary }
    }

    if (data.id) {
      await knowledgeAPI.updateArticle(data.id, data)
      ElMessage.success('更新成功')
    } else {
      await knowledgeAPI.createArticle(data)
      ElMessage.success('创建成功')
    }
    articleDrawerVisible.value = false
    loadArticles()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const deleteArticle = async (id) => {
  await ElMessageBox.confirm('确定删除该文章？', '确认删除', {
    type: 'warning'
  })
  try {
    await knowledgeAPI.deleteArticle(id)
    ElMessage.success('删除成功')
    loadArticles()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '删除失败')
  }
}

const handleResize = () => {
  isMobile.value = window.innerWidth <= 768
}

onMounted(() => {
  loadCategories()
  loadArticles()
  window.addEventListener('resize', handleResize)
})
</script>

<style scoped>
.list-container {
  padding: 0;
}

.list-card {
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.card-header span {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.filter-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.pagination-wrapper {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .card-header {
    flex-direction: column;
    align-items: stretch;
  }

  .header-actions {
    width: 100%;
  }

  .header-actions .el-button {
    flex: 1;
  }

  .filter-bar {
    flex-direction: column;
  }

  .filter-bar .el-select,
  .filter-bar .el-input {
    width: 100% !important;
  }

  :deep(.el-table) {
    font-size: 12px;
  }

  :deep(.el-table .el-button) {
    padding: 5px 8px;
    font-size: 12px;
  }

  .pagination-wrapper {
    justify-content: center;
  }

  :deep(.el-pagination) {
    flex-wrap: wrap;
    justify-content: center;
  }
}
</style>
