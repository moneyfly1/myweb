<template>
  <div class="list-container admin-invites">
    <!--
      邀请管理（单页版）
      ------------------------------------------------------------------
      旧版把「邀请码列表 / 邀请关系 / 邀请统计」拆成三个页签，管理员要来回切换
      才能把「谁邀请了多少人、转化多少、奖励发了多少」拼起来看。
      现在改成：顶部统计 + 一张邀请人主表 + 点开详情抽屉（邀请码 / 邀请关系）。
    -->
    <div class="invite-overview" v-loading="statsLoading">
      <div
        v-for="item in overviewCards"
        :key="item.label"
        class="invite-overview__item"
        :class="`is-${item.tone}`"
      >
        <div class="invite-overview__label">{{ item.label }}</div>
        <div class="invite-overview__value">{{ item.value }}</div>
        <div class="invite-overview__hint">{{ item.hint }}</div>
      </div>
    </div>

    <el-card shadow="never" class="list-card">
      <template #header>
        <div class="card-header-wrapper">
          <span>邀请人列表<template v-if="total > 0">（{{ total }}）</template></span>
          <div class="header-buttons">
            <el-button type="default" @click="openSettings">
              <el-icon><Setting /></el-icon>
              <span class="desktop-only">邀请设置</span>
            </el-button>
            <el-button type="primary" @click="refreshAll">
              <el-icon><Refresh /></el-icon>
              <span class="desktop-only">刷新</span>
            </el-button>
          </div>
        </div>
      </template>

      <div class="invite-toolbar">
        <el-input
          v-model="keyword"
          class="invite-search"
          placeholder="搜索邀请人用户名 / 邮箱 / ID"
          clearable
          @input="debouncedSearch"
          @keyup.enter="searchNow"
          @clear="searchNow"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <ResponsiveDataView
        :data="inviters"
        :fields="inviterFields"
        id-field="inviter_id"
        title-field="username"
        :loading="loading"
        :error="error || false"
        empty-title="暂无邀请数据"
        empty-description="还没有用户通过邀请码成功邀请他人"
        @retry="loadInviters"
      >
        <template #table>
          <el-table :data="inviters" v-loading="loading" class="data-table" style="width: 100%">
            <el-table-column label="邀请人" min-width="220">
              <template #default="{ row }">
                <div class="cell-user">
                  <span class="cell-user__name">{{ row.username || '未知用户' }}</span>
                  <span class="cell-user__meta">
                    ID {{ row.inviter_id }}<template v-if="row.email"> · {{ row.email }}</template>
                  </span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="现存邀请码" width="110" align="center">
              <template #default="{ row }">{{ row.code_count || 0 }} 个</template>
            </el-table-column>
            <el-table-column label="已邀请" width="100" align="center">
              <template #default="{ row }">
                <span class="cell-strong">{{ row.invited_count || 0 }}</span> 人
              </template>
            </el-table-column>
            <el-table-column label="已消费" width="110" align="center">
              <template #default="{ row }">
                <span class="cell-strong">{{ row.purchased_count || 0 }}</span> 人
              </template>
            </el-table-column>
            <el-table-column label="累计消费" width="130" align="right">
              <template #default="{ row }">
                <span class="money-value">¥{{ money(row.consumption) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="奖励" width="150" align="right">
              <template #default="{ row }">
                <div class="cell-reward">
                  <span class="reward-given">已发 ¥{{ money(row.reward_given) }}</span>
                  <span class="reward-pending" :class="{ 'is-zero': !(row.reward_pending > 0) }">
                    待发 ¥{{ money(row.reward_pending) }}
                  </span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="最近邀请" width="170" align="center">
              <template #default="{ row }">{{ formatDateTimeSafe(row.last_invited_at) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="90" align="center" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openDetail(row)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </template>

        <template #actions="{ item }">
          <el-button size="small" type="primary" plain @click="openDetail(item)">查看详情</el-button>
        </template>
      </ResponsiveDataView>

      <PaginationBar
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        @current-change="handlePageChange"
        @size-change="handleSizeChange"
      />
    </el-card>

    <!-- 邀请人详情：邀请码 + 邀请关系（列表点「详情」后展开） -->
    <AppDrawer
      v-model="detailVisible"
      :title="detailTitle"
      size="820px"
      mobile-size="100%"
      :loading="detailLoading"
    >
      <div v-loading="detailLoading" class="inviter-detail">
        <template v-if="detail">
          <div class="detail-summary">
            <div v-for="item in detailSummaryCards" :key="item.label" class="detail-summary__item">
              <div class="detail-summary__value">{{ item.value }}</div>
              <div class="detail-summary__label">{{ item.label }}</div>
            </div>
          </div>

          <section class="detail-section">
            <h4 class="detail-section__title">基本信息</h4>
            <div class="detail-info">
              <div class="detail-info__row">
                <span class="detail-info__label">用户名</span>
                <span class="detail-info__value">{{ detail.user.username || '-' }}</span>
              </div>
              <div class="detail-info__row">
                <span class="detail-info__label">邮箱</span>
                <span class="detail-info__value">{{ detail.user.email || '-' }}</span>
              </div>
              <div class="detail-info__row">
                <span class="detail-info__label">用户 ID</span>
                <span class="detail-info__value">{{ detail.user.id }}</span>
              </div>
              <div class="detail-info__row">
                <span class="detail-info__label">账户余额</span>
                <span class="detail-info__value money-value">¥{{ money(detail.user.balance) }}</span>
              </div>
              <div class="detail-info__row">
                <span class="detail-info__label">注册时间</span>
                <span class="detail-info__value">{{ formatDateTimeSafe(detail.user.created_at) }}</span>
              </div>
            </div>
          </section>

          <section class="detail-section">
            <h4 class="detail-section__title">
              邀请码
              <span class="detail-section__count">{{ detail.codes.length }}</span>
            </h4>
            <ResponsiveDataView
              :data="detail.codes"
              :fields="codeFields"
              id-field="id"
              title-field="code"
              :loading="detailLoading"
              empty-title="暂无邀请码"
              empty-description="该用户名下没有有效邀请码（已删除的邀请码不在列表中，但邀请关系会保留）"
            >
              <template #table>
                <el-table :data="detail.codes" class="data-table" style="width: 100%">
                  <el-table-column label="邀请码" min-width="140">
                    <template #default="{ row }">
                      <span class="code-text">{{ row.code }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="已使用" width="90" align="center">
                    <template #default="{ row }">{{ row.used_count || 0 }} 次</template>
                  </el-table-column>
                  <el-table-column label="状态" width="90" align="center">
                    <template #default="{ row }">
                      <el-tag :type="row.is_active ? 'success' : 'info'" size="small">
                        {{ row.is_active ? '启用' : '已禁用' }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="邀请人奖励" width="110" align="right">
                    <template #default="{ row }">¥{{ money(row.inviter_reward) }}</template>
                  </el-table-column>
                  <el-table-column label="被邀请人奖励" width="120" align="right">
                    <template #default="{ row }">¥{{ money(row.invitee_reward) }}</template>
                  </el-table-column>
                  <el-table-column label="创建时间" width="170" align="center">
                    <template #default="{ row }">{{ formatDateTimeSafe(row.created_at) }}</template>
                  </el-table-column>
                  <el-table-column label="操作" width="80" align="center" fixed="right">
                    <template #default="{ row }">
                      <el-button link type="danger" @click="removeCode(row)">删除</el-button>
                    </template>
                  </el-table-column>
                </el-table>
              </template>
              <template #actions="{ item }">
                <el-button size="small" type="danger" plain @click="removeCode(item)">删除邀请码</el-button>
              </template>
            </ResponsiveDataView>
          </section>

          <section class="detail-section">
            <h4 class="detail-section__title">
              邀请关系
              <span class="detail-section__count">{{ detail.relations.length }}</span>
            </h4>
            <ResponsiveDataView
              :data="detail.relations"
              :fields="relationFields"
              id-field="invitee_id"
              title-field="invitee_username"
              :loading="detailLoading"
              empty-title="暂无邀请关系"
              empty-description="该用户还没有成功邀请他人"
            >
              <template #table>
                <el-table :data="detail.relations" class="data-table" style="width: 100%">
                  <el-table-column label="被邀请人" min-width="180">
                    <template #default="{ row }">
                      <div class="cell-user">
                        <span class="cell-user__name">
                          {{ row.invitee_username || `用户 ${row.invitee_id}` }}
                        </span>
                        <span class="cell-user__meta">
                          ID {{ row.invitee_id }}<template v-if="row.invitee_email"> · {{ row.invitee_email }}</template>
                        </span>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="注册时间" width="170" align="center">
                    <template #default="{ row }">{{ formatDateTimeSafe(row.created_at) }}</template>
                  </el-table-column>
                  <el-table-column label="是否消费" width="100" align="center">
                    <template #default="{ row }">
                      <el-tag :type="row.has_purchased ? 'success' : 'info'" size="small">
                        {{ row.has_purchased ? '已消费' : '未消费' }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="消费金额" width="120" align="right">
                    <template #default="{ row }">
                      <span class="money-value">¥{{ money(row.total_consumption) }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="奖励" width="150" align="center">
                    <template #default="{ row }">
                      <el-tag :type="row.reward_given ? 'success' : 'warning'" size="small">
                        {{ row.status_text || (row.reward_given ? '已发放' : '未发放') }}
                      </el-tag>
                    </template>
                  </el-table-column>
                </el-table>
              </template>
            </ResponsiveDataView>
          </section>
        </template>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button type="primary" @click="openUserDetail">查看完整用户详情</el-button>
      </template>
    </AppDrawer>

    <!-- 邀请设置（保留原有入口，避免能力丢失） -->
    <AppDrawer
      v-model="showSettingsDialog"
      title="邀请设置"
      size="500px"
      mobile-size="100%"
      :loading="savingSettings"
    >
      <div class="settings-dialog-content">
        <el-alert
          title="邀请奖励配置说明"
          type="info"
          :closable="false"
          class="settings-alert"
        >
          <template #default>
            <div class="alert-content">
              <p><strong>邀请人奖励：</strong>被邀请人首次购买套餐后，邀请人获得的奖励金额（元）</p>
              <p><strong>被邀请人奖励：</strong>新用户使用邀请码注册后，立即获得的奖励金额（元）</p>
              <p class="alert-note">注意：此设置应用于之后新生成的邀请码，已生成的邀请码不受影响</p>
            </div>
          </template>
        </el-alert>
        <el-form :model="inviteSettings" label-width="0" class="invite-settings-form">
          <el-form-item prop="inviter_reward" class="settings-form-item">
            <div class="form-item-wrapper">
              <div class="form-item-label">邀请人奖励（元）</div>
              <el-input-number
                v-model="inviteSettings.inviter_reward"
                :min="0"
                :max="10000"
                :precision="2"
                :step="1"
                controls-position="right"
                class="settings-input-number"
              />
            </div>
          </el-form-item>
          <el-form-item prop="invitee_reward" class="settings-form-item">
            <div class="form-item-wrapper">
              <div class="form-item-label">被邀请人奖励（元）</div>
              <el-input-number
                v-model="inviteSettings.invitee_reward"
                :min="0"
                :max="10000"
                :precision="2"
                :step="1"
                controls-position="right"
                class="settings-input-number"
              />
            </div>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <FormActionBar
          :loading="savingSettings"
          submit-text="保存设置"
          :sticky="false"
          @cancel="showSettingsDialog = false"
          @submit="saveInviteSettings"
        />
      </template>
    </AppDrawer>
  </div>
</template>

<script setup>
defineOptions({ name: 'AdminInvites' })

import { computed, reactive, ref, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from '@/utils/elementPlusServices'
import { Search, Refresh, Setting } from '@element-plus/icons-vue'
import { inviteAPI } from '@/utils/api'
import { useApi } from '@/utils/api'
import { formatDateTimeSafe } from '@/utils/date'
import { debounce } from '@/composables/useDebounce'
import { confirmDelete } from '@/utils/confirmAction'
import AppDrawer from '@/components/AppDrawer.vue'
import FormActionBar from '@/components/FormActionBar.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'

const api = useApi()
const router = useRouter()

const money = (value) => Number(value || 0).toFixed(2)

// ---------------------------------------------------------------------------
// 顶部统计
// ---------------------------------------------------------------------------
const statsLoading = ref(false)
const stats = reactive({
  total_codes: 0,
  active_codes: 0,
  total_relations: 0,
  total_inviters: 0,
  purchased_count: 0,
  total_consumption: 0,
  reward_given_amount: 0,
  reward_pending_amount: 0,
})

const loadStats = async () => {
  statsLoading.value = true
  try {
    const response = await inviteAPI.getAdminInviteStatistics()
    const data = response?.data?.data || response?.data || {}
    Object.assign(stats, {
      total_codes: Number(data.total_codes ?? data.total_invite_codes ?? 0),
      active_codes: Number(data.active_codes ?? data.active_invite_codes ?? 0),
      total_relations: Number(data.total_relations ?? data.total_invite_relations ?? 0),
      total_inviters: Number(data.total_inviters ?? 0),
      purchased_count: Number(data.purchased_count ?? 0),
      total_consumption: Number(data.total_consumption ?? 0),
      reward_given_amount: Number(data.reward_given_amount ?? data.total_invite_reward ?? 0),
      reward_pending_amount: Number(data.reward_pending_amount ?? 0),
    })
  } catch (error) {
    console.error('加载邀请统计失败:', error)
    ElMessage.error('加载邀请统计失败: ' + (error.response?.data?.message || error.message || '未知错误'))
  } finally {
    statsLoading.value = false
  }
}

const purchasedRate = computed(() => {
  if (!stats.total_relations) return 0
  return Math.round((stats.purchased_count / stats.total_relations) * 100)
})

const overviewCards = computed(() => [
  {
    label: '邀请码',
    value: stats.total_codes,
    hint: `生效中 ${stats.active_codes}`,
    tone: 'default',
  },
  {
    label: '邀请人',
    value: stats.total_inviters,
    hint: `共邀请 ${stats.total_relations} 人`,
    tone: 'default',
  },
  {
    label: '被邀请人已消费',
    value: stats.purchased_count,
    hint: `转化率 ${purchasedRate.value}%`,
    tone: 'success',
  },
  {
    label: '被邀请人累计消费',
    value: `¥${money(stats.total_consumption)}`,
    hint: '邀请带来的订单金额',
    tone: 'success',
  },
  {
    label: '已发放奖励',
    value: `¥${money(stats.reward_given_amount)}`,
    hint: '邀请人奖励已到账',
    tone: 'primary',
  },
  {
    label: '待发放奖励',
    value: `¥${money(stats.reward_pending_amount)}`,
    hint: stats.reward_pending_amount > 0 ? '达标但尚未到账' : '没有待发放奖励',
    tone: stats.reward_pending_amount > 0 ? 'warning' : 'default',
  },
])

// ---------------------------------------------------------------------------
// 邀请人列表（唯一列表）
// ---------------------------------------------------------------------------
const loading = ref(false)
const error = ref('')
const inviters = ref([])
const keyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 并发保护：keep-alive 首次进入时 onMounted 与 onActivated 会先后触发同一份请求，
// 用序号保证只有最后一次请求的结果（含失败）能写回状态，避免旧请求把新数据清空。
let loadSeq = 0

const loadInviters = async () => {
  const seq = ++loadSeq
  loading.value = true
  error.value = ''
  try {
    const response = await inviteAPI.getAdminInviteInviters({
      page: page.value,
      size: pageSize.value,
      keyword: keyword.value.trim(),
    })
    if (seq !== loadSeq) return
    const data = response?.data?.data || response?.data || {}
    inviters.value = data.list || data.items || []
    total.value = Number(data.total || 0)
  } catch (err) {
    if (seq !== loadSeq) return
    console.error('加载邀请人列表失败:', err)
    error.value = err.response?.data?.message || err.message || '加载邀请人列表失败'
    inviters.value = []
    total.value = 0
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

const debouncedSearch = debounce(() => {
  page.value = 1
  loadInviters()
}, 350)

const searchNow = () => {
  page.value = 1
  loadInviters()
}

const handlePageChange = (value) => {
  page.value = value
  loadInviters()
}

const handleSizeChange = (value) => {
  pageSize.value = value
  page.value = 1
  loadInviters()
}

const refreshAll = () => {
  loadStats()
  loadInviters()
  ElMessage.success('已刷新')
}

const inviterFields = [
  // 移动端卡片标题已经是用户名（titleField），这里不再重复一行，直接给联系方式与聚合数据
  { key: 'email', label: '邮箱', fullWidth: true, formatter: v => v || '-' },
  { key: 'code_count', label: '现存邀请码', formatter: v => `${v || 0} 个` },
  { key: 'invited_count', label: '已邀请人数', formatter: v => `${v || 0} 人` },
  { key: 'purchased_count', label: '已消费人数', formatter: v => `${v || 0} 人` },
  { key: 'consumption', label: '累计消费', type: 'money' },
  { key: 'reward_given', label: '已发奖励', type: 'money' },
  { key: 'reward_pending', label: '待发奖励', type: 'money' },
  { key: 'last_invited_at', label: '最近邀请', type: 'date' },
]

// ---------------------------------------------------------------------------
// 详情抽屉
// ---------------------------------------------------------------------------
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref(null)

const detailTitle = computed(() => {
  const name = detail.value?.user?.username
  return name ? `${name} 的邀请详情` : '邀请详情'
})

const detailSummaryCards = computed(() => {
  const summary = detail.value?.summary || {}
  return [
    { label: '邀请人数', value: summary.registered || 0 },
    { label: '已消费', value: summary.purchased || 0 },
    { label: '累计消费', value: `¥${money(summary.total_consumption)}` },
    { label: '已发奖励', value: `¥${money(summary.total_reward)}` },
    { label: '待发奖励', value: `¥${money(summary.pending_reward)}` },
  ]
})

const openDetail = async (row) => {
  const inviterId = row?.inviter_id || row?.id
  if (!inviterId) return
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const response = await inviteAPI.getAdminInviterDetail(inviterId)
    const data = response?.data?.data || response?.data || {}
    detail.value = {
      user: data.user || {},
      codes: data.codes || [],
      relations: data.relations || [],
      summary: data.summary || {},
    }
  } catch (err) {
    ElMessage.error('加载邀请详情失败: ' + (err.response?.data?.message || err.message || '未知错误'))
    detailVisible.value = false
  } finally {
    detailLoading.value = false
  }
}

const openUserDetail = () => {
  const userId = detail.value?.user?.id
  if (!userId) return
  detailVisible.value = false
  router.push({ path: '/admin/users', query: { user_id: String(userId) } })
}

const removeCode = async (row) => {
  try {
    await confirmDelete('邀请码', 1, {
      message: row.used_count > 0
        ? `邀请码「${row.code}」已被使用 ${row.used_count} 次，删除后将被禁用（保留已产生的邀请关系）。确认继续？`
        : `确定删除邀请码「${row.code}」吗？删除后不可恢复。`,
    })
  } catch (err) {
    return
  }
  try {
    const response = await inviteAPI.batchDeleteInviteCodes([row.id])
    const data = response?.data?.data || {}
    const deleted = data.deleted_count || 0
    const disabled = data.disabled_count || 0
    let message = `已删除 ${deleted} 个邀请码`
    if (disabled > 0) message += `，已禁用 ${disabled} 个已使用的邀请码`
    ElMessage.success(message)
    // 就地更新，避免整页刷新导致抽屉关闭
    if (detail.value) {
      detail.value.codes = detail.value.codes.filter(item => item.id !== row.id)
    }
    loadStats()
    loadInviters()
  } catch (err) {
    ElMessage.error('删除邀请码失败: ' + (err.response?.data?.message || err.message || '未知错误'))
  }
}

const codeFields = [
  { key: 'used_count', label: '已使用', formatter: v => `${v || 0} 次` },
  {
    key: 'is_active',
    label: '状态',
    type: 'tag',
    tagType: v => (v ? 'success' : 'info'),
    formatter: v => (v ? '启用' : '已禁用'),
  },
  { key: 'inviter_reward', label: '邀请人奖励', type: 'money' },
  { key: 'invitee_reward', label: '被邀请人奖励', type: 'money' },
  { key: 'created_at', label: '创建时间', type: 'date' },
]

const relationFields = [
  // 移动端卡片标题即被邀请人用户名（titleField），这里只列明细字段
  { key: 'invitee_email', label: '邮箱', fullWidth: true, formatter: v => v || '-' },
  {
    key: 'has_purchased',
    label: '是否消费',
    type: 'tag',
    tagType: v => (v ? 'success' : 'info'),
    formatter: v => (v ? '已消费' : '未消费'),
  },
  { key: 'total_consumption', label: '消费金额', type: 'money' },
  { key: 'created_at', label: '注册时间', type: 'date' },
  { key: 'status_text', label: '奖励状态' },
]

// ---------------------------------------------------------------------------
// 邀请设置
// ---------------------------------------------------------------------------
const showSettingsDialog = ref(false)
const savingSettings = ref(false)
const inviteSettings = reactive({
  inviter_reward: 0,
  invitee_reward: 0,
})

const openSettings = async () => {
  showSettingsDialog.value = true
  try {
    const response = await api.get('/admin/settings')
    const settings = response?.data?.data || response?.data || {}
    if (settings.invite) {
      Object.assign(inviteSettings, settings.invite)
    }
  } catch (error) {
    console.error('加载邀请设置失败:', error)
    ElMessage.error('加载邀请设置失败: ' + (error.response?.data?.message || error.message || '未知错误'))
  }
}

const saveInviteSettings = async () => {
  savingSettings.value = true
  try {
    await api.put('/admin/settings/invite', inviteSettings)
    ElMessage.success('邀请设置保存成功')
    showSettingsDialog.value = false
  } catch (error) {
    console.error('保存邀请设置失败:', error)
    ElMessage.error('保存邀请设置失败: ' + (error.response?.data?.message || error.message || '未知错误'))
  } finally {
    savingSettings.value = false
  }
}

onMounted(() => {
  loadStats()
  loadInviters()
})

// AdminLayout 用 keep-alive 缓存后台页面，重新进入本页时 onMounted 不会再触发
onActivated(() => {
  loadStats()
  loadInviters()
})
</script>

<style scoped lang="scss">
.admin-invites {
  padding: 20px;
  width: 100%;
  box-sizing: border-box;
  overflow-x: clip;
}

/* 顶部统计 */
.invite-overview {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;

  @media (max-width: 1280px) {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  @media (max-width: 768px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }
}

.invite-overview__item {
  min-width: 0;
  padding: 12px 14px;
  background: var(--card-bg, #fff);
  border: 1px solid #ebeef5;
  border-left: 3px solid #dcdfe6;
  border-radius: 8px;
  box-sizing: border-box;

  &.is-success {
    border-left-color: #67c23a;
  }

  &.is-primary {
    border-left-color: var(--el-color-primary, #409eff);
  }

  &.is-warning {
    border-left-color: #e6a23c;
  }

  @media (max-width: 768px) {
    padding: 10px 12px;
  }
}

.invite-overview__label {
  color: #909399;
  font-size: 13px;
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.invite-overview__value {
  margin: 4px 0 2px;
  color: #303133;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.15;
  word-break: break-all;

  @media (max-width: 768px) {
    font-size: 18px;
  }
}

.invite-overview__hint {
  color: #909399;
  font-size: 12px;
  line-height: 1.3;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 表头 / 工具条 */
.card-header-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  flex-wrap: wrap;
  gap: 10px;
}

.header-buttons {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.header-buttons :deep(.el-button) {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.desktop-only {
  @media (max-width: 768px) {
    display: none !important;
  }
}

.invite-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;

  @media (max-width: 768px) {
    margin-bottom: 10px;
  }
}

.invite-search {
  width: 320px;
  max-width: 100%;
  min-width: 0;

  @media (max-width: 768px) {
    width: 100%;
  }
}

/* 表格单元格 */
.data-table {
  width: 100%;
}

.cell-user {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.35;
}

.cell-user__name {
  color: #303133;
  font-weight: 600;
  word-break: break-all;
}

.cell-user__meta {
  color: #909399;
  font-size: 12px;
  word-break: break-all;
}

.cell-strong {
  color: var(--el-color-primary, #409eff);
  font-weight: 700;
}

.money-value {
  color: #303133;
  font-weight: 600;
}

.cell-reward {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  line-height: 1.35;
}

.reward-given {
  color: #67c23a;
  font-weight: 600;
}

.reward-pending {
  color: #e6a23c;
  font-size: 12px;

  &.is-zero {
    color: #c0c4cc;
  }
}

.code-text {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-weight: 600;
  letter-spacing: 0.5px;
}

/* 详情抽屉 */
.inviter-detail {
  min-width: 0;
}

.detail-summary {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 16px;

  @media (max-width: 768px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.detail-summary__item {
  min-width: 0;
  padding: 10px 8px;
  text-align: center;
  background: #f8fafc;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  box-sizing: border-box;
}

.detail-summary__value {
  color: var(--el-color-primary, #409eff);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.2;
  word-break: break-all;
}

.detail-summary__label {
  margin-top: 2px;
  color: #909399;
  font-size: 12px;
}

.detail-section {
  margin-bottom: 18px;
}

.detail-section__title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 10px;
  padding-left: 8px;
  border-left: 3px solid var(--el-color-primary, #409eff);
  color: #303133;
  font-size: 14px;
  font-weight: 600;
}

.detail-section__count {
  color: #909399;
  font-size: 12px;
  font-weight: 500;
}

.detail-info {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px 16px;

  @media (max-width: 768px) {
    grid-template-columns: minmax(0, 1fr);
  }
}

.detail-info__row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
  padding-bottom: 6px;
  border-bottom: 1px dashed #ebeef5;
}

.detail-info__label {
  flex: 0 0 auto;
  color: #909399;
  font-size: 13px;
}

.detail-info__value {
  min-width: 0;
  color: #303133;
  font-size: 13px;
  text-align: right;
  word-break: break-all;
}

/* 邀请设置 */
.settings-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.settings-alert {
  :deep(.alert-content p) {
    margin: 0 0 6px;
    line-height: 1.5;
  }

  :deep(.alert-note) {
    margin-bottom: 0;
    color: #e6a23c;
  }
}

.invite-settings-form :deep(.el-form-item) {
  margin-bottom: 16px;
}

.form-item-label {
  margin-bottom: 6px;
  color: #303133;
  font-size: 14px;
  font-weight: 500;
}

.settings-input-number {
  width: 100%;
}
</style>
