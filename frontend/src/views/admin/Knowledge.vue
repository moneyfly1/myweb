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
            <el-select v-model="articleFilter.category_id" placeholder="筛选分类" clearable class="filter-select" @change="loadArticles">
              <el-option v-for="cat in categories" :key="cat.id" :label="cat.name" :value="cat.id" />
            </el-select>
            <el-input v-model="articleFilter.keyword" placeholder="搜索标题..." clearable class="filter-search" @keyup.enter="loadArticles">
              <template #append>
                <el-button @click="loadArticles"><el-icon><Search /></el-icon></el-button>
              </template>
            </el-input>
            <div class="filter-actions">
              <el-button type="primary" @click="loadArticles">
                <el-icon><Search /></el-icon>
                搜索
              </el-button>
              <el-button @click="resetArticleFilter">重置</el-button>
            </div>
          </div>

          <ResponsiveDataView
            class="knowledge-data-view"
            :data="articles"
            :fields="mobileArticleFields"
            :loading="loading"
            title-field="title"
            empty-title="暂无文章"
            empty-description="可新建文章或调整筛选条件"
          >
            <template #table>
              <el-table :data="articles" v-loading="loading" stripe border class="knowledge-table">
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
            </template>

            <template #header="{ item }">
              <div class="knowledge-mobile-header">
                <div class="knowledge-mobile-title">
                  <span class="item-title">{{ item.title }}</span>
                  <span class="item-id">#{{ item.id }}</span>
                </div>
                <el-tag :type="item.is_active ? 'success' : 'info'" size="small">
                  {{ item.is_active ? '启用' : '禁用' }}
                </el-tag>
              </div>
            </template>

            <template #field-category="{ item }">
              <el-tag v-if="item.category" size="small" effect="plain">
                {{ item.category.name }}
              </el-tag>
              <span v-else>-</span>
            </template>

            <template #field-created_at="{ item }">
              {{ formatDate(item.created_at) }}
            </template>

            <template #actions="{ item }">
              <div class="knowledge-mobile-actions">
                <el-button size="small" @click="showArticleDrawer(item)">编辑</el-button>
                <el-button size="small" type="danger" @click="deleteArticle(item.id)">删除</el-button>
              </div>
            </template>
          </ResponsiveDataView>

          <PaginationBar
            v-model:current-page="articlePagination.page"
            v-model:page-size="articlePagination.page_size"
            :total="articlePagination.total"
            @change="loadArticles"
          />
        </el-tab-pane>

        <el-tab-pane label="分类管理" name="categories">
          <ResponsiveDataView
            class="knowledge-data-view"
            :data="categories"
            :fields="mobileCategoryFields"
            title-field="name"
            empty-title="暂无分类"
            empty-description="新建分类后可用于组织知识库文章"
          >
            <template #table>
              <el-table :data="categories" stripe border class="knowledge-table">
                <el-table-column prop="id" label="ID" width="60" />
                <el-table-column prop="name" label="名称" min-width="150" />
                <el-table-column prop="icon" label="图标" width="150">
                  <template #default="{ row }">
                    <el-icon><component :is="resolveIcon(row.icon)" /></el-icon>
                    <span class="icon-name">{{ row.icon || 'Folder' }}</span>
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
            </template>

            <template #header="{ item }">
              <div class="knowledge-mobile-header">
                <div class="knowledge-mobile-title">
                  <span class="mobile-icon">
                    <el-icon><component :is="resolveIcon(item.icon)" /></el-icon>
                  </span>
                  <span class="item-title">{{ item.name }}</span>
                  <span class="item-id">#{{ item.id }}</span>
                </div>
                <el-tag :type="item.is_active ? 'success' : 'info'" size="small">
                  {{ item.is_active ? '启用' : '禁用' }}
                </el-tag>
              </div>
            </template>

            <template #field-icon="{ item }">
              <span class="mobile-icon-field">
                <el-icon><component :is="resolveIcon(item.icon)" /></el-icon>
                {{ item.icon || 'Folder' }}
              </span>
            </template>

            <template #field-created_at="{ item }">
              {{ formatDate(item.created_at) }}
            </template>

            <template #actions="{ item }">
              <div class="knowledge-mobile-actions">
                <el-button size="small" @click="showCategoryDrawer(item)">编辑</el-button>
                <el-button size="small" type="danger" @click="deleteCategory(item.id)">删除</el-button>
              </div>
            </template>
          </ResponsiveDataView>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 分类抽屉 -->
    <AppDrawer
      v-model="catDrawerVisible"
      :title="catForm.id ? '编辑分类' : '新建分类'"
      size="500px"
      mobile-size="100%"
      :loading="saving"
    >
      <el-form :model="catForm" label-width="80px" :rules="catRules" ref="catFormRef">
        <el-form-item label="名称" prop="name">
          <el-input v-model="catForm.name" placeholder="请输入分类名称" />
        </el-form-item>
        <el-form-item label="图标" prop="icon">
          <el-input v-model="catForm.icon" placeholder="如: Folder, Document, Star" />
          <div class="form-tip">
            常用图标: Folder, Document, Star, Reading, Guide, QuestionFilled
          </div>
        </el-form-item>
        <el-form-item label="排序" prop="sort_order">
          <el-input-number v-model="catForm.sort_order" :min="0" :max="9999" />
          <div class="form-tip">
            数字越小越靠前
          </div>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="catForm.is_active" />
        </el-form-item>
      </el-form>
      <template #footer>
        <FormActionBar
          :loading="saving"
          :sticky="false"
          @cancel="catDrawerVisible = false"
          @submit="saveCategory"
        />
      </template>
    </AppDrawer>

    <!-- 文章抽屉 -->
    <AppDrawer
      v-model="articleDrawerVisible"
      :title="articleForm.id ? '编辑文章' : '新建文章'"
      size="760px"
      mobile-size="100%"
      :loading="saving"
    >
      <el-form
        :model="articleForm"
        label-width="80px"
        :rules="articleRules"
        ref="articleFormRef"
        class="article-drawer-form"
      >
        <el-form-item label="标题" prop="title">
          <el-input v-model="articleForm.title" placeholder="请输入文章标题" />
        </el-form-item>
        <el-form-item label="分类" prop="category_id">
          <el-select v-model="articleForm.category_id" placeholder="请选择分类" class="form-control">
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
          <div class="form-tip">
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
        <FormActionBar
          :loading="saving"
          :sticky="false"
          @cancel="articleDrawerVisible = false"
          @submit="saveArticle"
        />
      </template>
    </AppDrawer>
  </div>
</template>

<script setup>
import { ref, onMounted, reactive } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { FolderAdd, DocumentAdd, Search, Folder, Document, Reading, Files, Setting, Star, InfoFilled, QuestionFilled, Notebook, Clock, View } from '@element-plus/icons-vue'
import { knowledgeAPI } from '@/utils/api'
import AppDrawer from '@/components/AppDrawer.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import { confirmDelete } from '@/utils/confirmAction'

const iconMap = { FolderAdd, DocumentAdd, Search, Folder, Document, Reading, Files, Setting, Star, InfoFilled, QuestionFilled, Notebook, Clock, View }
const resolveIcon = (name) => iconMap[name] || Folder

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
  page_size: 10,
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

const mobileArticleFields = [
  { key: 'category', label: '分类' },
  { key: 'view_count', label: '浏览' },
  { key: 'sort_order', label: '排序' },
  { key: 'created_at', label: '创建时间' }
]

const mobileCategoryFields = [
  { key: 'icon', label: '图标' },
  { key: 'sort_order', label: '排序' },
  { key: 'created_at', label: '创建时间' }
]

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

const resetArticleFilter = () => {
  articleFilter.category_id = null
  articleFilter.keyword = ''
  articlePagination.page = 1
  loadArticles()
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
  await confirmDelete('分类', 1, {
    message: '删除分类将同时删除该分类下的所有文章，删除后不可恢复。',
    title: '确认删除分类'
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
  await confirmDelete('文章', 1, {
    message: '确定删除该文章吗？删除后不可恢复。',
    title: '确认删除文章'
  })
  try {
    await knowledgeAPI.deleteArticle(id)
    ElMessage.success('删除成功')
    loadArticles()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '删除失败')
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
.list-container {
  padding: 0;
}

.list-card {
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

.filter-bar {
  display: grid;
  grid-template-columns: minmax(200px, 0.8fr) minmax(240px, 1.2fr) minmax(150px, max-content);
  align-items: end;
  gap: 12px;
  margin-bottom: 16px;
}

.filter-actions {
  display: flex;
  justify-self: end;
  align-items: center;
  gap: 8px;

  .el-button {
    margin-left: 0;
  }
}

.filter-select {
  width: 100%;
  min-width: 0;
}

.filter-search {
  width: 100%;
  min-width: 0;
}

.icon-name {
  margin-left: 8px;
}

.form-control {
  width: 100%;
}

.form-tip {
  margin-top: 8px;
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}

.article-drawer-form :deep(.el-textarea__inner) {
  resize: vertical;
}

.article-drawer-form :deep(.el-input-number) {
  width: 160px;
}

.knowledge-data-view {
  margin-top: 12px;
  min-width: 0;
}

.knowledge-table {
  width: 100%;
}

.knowledge-mobile-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.knowledge-mobile-title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.item-title {
  min-width: 0;
  color: #303133;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
  word-break: break-word;
}

.item-id {
  flex-shrink: 0;
  color: #909399;
  font-size: 12px;
  line-height: 1.4;
}

.mobile-icon,
.mobile-icon-field {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.mobile-icon {
  flex-shrink: 0;
  color: #606266;
}

.knowledge-mobile-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  width: 100%;
}

.knowledge-mobile-actions .el-button {
  min-height: 44px;
  margin: 0;
  touch-action: manipulation;
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
    grid-template-columns: 1fr;
  }

  .filter-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    justify-self: stretch;
    width: 100%;

    .el-button {
      width: 100%;
    }
  }

  .filter-bar .el-select,
  .filter-bar .el-input {
    width: 100%;
  }

  :deep(.el-pagination) {
    flex-wrap: wrap;
  }

  .article-drawer-form :deep(.el-form-item) {
    display: block;
    margin-bottom: 18px;
  }

  .article-drawer-form :deep(.el-form-item__label) {
    display: block;
    width: 100% !important;
    padding: 0 0 6px;
    text-align: left;
    line-height: 1.4;
  }

  .article-drawer-form :deep(.el-form-item__content) {
    margin-left: 0 !important;
  }

  .article-drawer-form :deep(.el-input-number) {
    width: 100%;
  }

  .article-drawer-form :deep(.el-textarea__inner) {
    min-height: 260px !important;
  }
}
</style>
