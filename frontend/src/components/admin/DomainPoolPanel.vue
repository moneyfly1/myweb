<!--
  订阅域名池管理面板（嵌入「系统设置 → 站点设置」）

  用途：官网域名在某些地区被屏蔽时，管理员需要不断更换订阅域名。手工要做的
  「改 DNS → 写 nginx 站点 → 签证书 → 重载 → 改面板配置」这里收敛成一次点击，
  并逐步显示执行结果；每个域名还带实时体检（DNS / 站点 / 证书 / 自动续期 / HTTPS 自检）。

  域名不生效时的排查顺序与列表顺序一致：DNS → 站点 → 证书 → HTTPS 自检 → 面板配置。
-->
<template>
  <div class="domain-pool">
    <div class="dp-head">
      <div>
        <div class="dp-title">订阅域名池</div>
        <div class="dp-sub">
          客户订阅（拉节点）走这里的「主域名」，其它域名作为备用地址一并下发给客户端；
          官网域名继续用于登录/支付，不受影响。换域名<strong>不会</strong>让已发出的订阅地址失效
          （同一后端、token 通用，任何域名都能取到同一份订阅）。
          「一键配置」会自动完成：DNS 校验 → 建站点 → 签证书 → 重载 Nginx → 写入订阅配置；
          证书到期前由 certbot 定时任务自动续期（下表显示剩余天数）。
        </div>
      </div>
      <div class="dp-actions">
        <el-button size="small" :loading="loading" @click="load">刷新体检</el-button>
        <el-button size="small" type="primary" plain :loading="renewing" @click="renewCerts(false)">立即续期</el-button>
      </div>
    </div>

    <el-alert v-if="!available" type="warning" :closable="false" show-icon class="dp-alert">
      <template #title>本服务器未开启「一键配置」</template>
      <div class="dp-alert-body">
        {{ unavailable || '面板进程需要以 root 运行，且服务器上要有 nginx 与 certbot。' }}
        <div class="dp-manual">
          手工步骤：DNS 添加 A 记录 → 在 nginx 里加反代 <code>127.0.0.1:8000</code> 的站点 →
          <code>certbot certonly --webroot -w /www/wwwroot/cboard -d 域名</code> →
          重载 nginx → 回到这里点「加入域名池」。
        </div>
      </div>
    </el-alert>

    <div class="dp-add">
      <el-input v-model="newDomain" size="small" placeholder="输入新域名，例如 sub2.example.com" class="dp-add-input"
        @keyup.enter="configure" />
      <el-checkbox v-model="makePrimary" size="small">同时设为订阅主域名</el-checkbox>
      <el-button type="primary" size="small" :loading="configuring" @click="configure">
        一键配置并加入域名池
      </el-button>
      <el-button size="small" :loading="adding" @click="addOnly">只加入域名池（站点已配好时用）</el-button>
    </div>

    <ResponsiveDataView
      :data="items"
      :fields="mobileFields"
      :loading="loading"
      id-field="domain"
      title-field="domain"
      empty-title="域名池为空"
      empty-description="在上面输入一个已解析到本机的域名，点「一键配置并加入域名池」。"
    >
      <template #table>
        <el-table :data="items" size="small" class="dp-table" :row-class-name="rowClass">

      <el-table-column label="域名" min-width="170">
        <template #default="{ row }">
          <span class="dp-domain">{{ row.domain }}</span>
          <el-tag v-if="row.is_primary" size="small" type="success" class="dp-tag">订阅主域名</el-tag>
          <el-tag v-else-if="row.is_site_domain" size="small" type="info" class="dp-tag">网站域名</el-tag>
          <el-tag v-else size="small" class="dp-tag">备用</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="DNS" width="76" align="center">
        <template #default="{ row }">
          <el-tag :type="row.dns_resolved ? 'success' : 'danger'" size="small" effect="plain">
            {{ row.dns_resolved ? '已解析' : '未解析' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="站点" width="76" align="center">
        <template #default="{ row }">
          <el-tag :type="row.vhost_exists ? 'success' : 'danger'" size="small" effect="plain">
            {{ row.vhost_exists ? '已配置' : '缺失' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="证书" width="116">
        <template #default="{ row }">
          <template v-if="row.cert_exists">
            <el-tag :type="certType(row.cert_days_left)" size="small" effect="plain">
              {{ row.cert_days_left }} 天
            </el-tag>
            <span v-if="row.auto_renew" class="dp-note">自动续期 ✓</span>
            <span v-else class="dp-note dp-warn">无续期配置</span>
          </template>
          <el-tag v-else type="danger" size="small" effect="plain">未签发</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="自检" width="88" align="center">
        <template #default="{ row }">
          <el-tooltip v-if="!row.https_ok && row.error" :content="row.error" placement="top">
            <el-tag type="danger" size="small" effect="plain">失败</el-tag>
          </el-tooltip>
          <el-tag v-else :type="row.https_ok ? 'success' : 'danger'" size="small" effect="plain">
            {{ row.https_ok ? '正常' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="196" align="right" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" :disabled="row.is_primary"
            @click="setPrimary(row.domain)">设为主域名</el-button>
          <el-button link type="primary" size="small" @click="repair(row.domain)">一键修复</el-button>
          <el-button link type="danger" size="small"
            :disabled="row.is_site_domain" @click="remove(row.domain)">移除</el-button>
        </template>
      </el-table-column>
        </el-table>
      </template>
      <template #header="{ item }">
        <div class="dp-mobile-head">
          <div class="dp-mobile-title">
            <span class="dp-domain">{ item.domain }</span>
            <el-tag v-if="item.is_primary" size="small" type="success">订阅主域名</el-tag>
            <el-tag v-else-if="item.is_site_domain" size="small" type="info">网站域名</el-tag>
            <el-tag v-else size="small">备用</el-tag>
          </div>
          <el-tag :type="item.https_ok ? 'success' : 'danger'" size="small" effect="plain">
            { item.https_ok ? '自检正常' : '自检失败' }
          </el-tag>
        </div>
      </template>
      <template #field-cert="{ item }">
        <template v-if="item.cert_exists">
          <el-tag :type="certType(item.cert_days_left)" size="small" effect="plain">{ item.cert_days_left } 天</el-tag>
          <span v-if="item.auto_renew" class="dp-note">自动续期 ✓</span>
          <span v-else class="dp-note dp-warn">无续期配置</span>
        </template>
        <el-tag v-else type="danger" size="small" effect="plain">未签发</el-tag>
      </template>
      <template #field-actions="{ item }">
        <div class="dp-mobile-actions">
          <el-button link type="primary" size="small" :disabled="item.is_primary" @click="setPrimary(item.domain)">设为主域名</el-button>
          <el-button link type="primary" size="small" @click="repair(item.domain)">一键修复</el-button>
          <el-button link type="danger" size="small" :disabled="item.is_site_domain" @click="remove(item.domain)">移除</el-button>
        </div>
      </template>
    </ResponsiveDataView>

    <div v-if="steps.length" class="dp-steps">
      <div class="dp-steps-title">最近一次配置过程</div>
      <div v-for="(s, i) in steps" :key="i" class="dp-step" :class="{ bad: !s.ok }">
        <span class="dp-step-icon">{{ s.ok ? '✓' : '✗' }}</span>
        <span class="dp-step-name">{{ s.name }}</span>
        <span class="dp-step-detail">{{ s.detail }}</span>
      </div>
    </div>

    <div class="dp-foot-actions">
      <el-button size="small" link type="warning" :loading="renewing" @click="renewCerts(true)">
        证书异常？强制续期
      </el-button>
      <span class="dp-note">证书到期前 30 天由服务器上的 certbot 定时任务自动续期并重载 Nginx，通常无需手动操作。</span>
    </div>

    <div class="dp-foot">
      备用地址会随订阅接口一起下发给客户端（<code>subscribe_urls</code>），App 会逐个尝试；
      用户面板「我的订阅」里也会显示备用地址。新增域名后如需 App 内置该域名做故障切换，
      需要发一个新版本（把域名加进 App 的域名池）。
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminAPI } from '@/utils/api'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'

const loading = ref(false)
const configuring = ref(false)
const adding = ref(false)
const available = ref(true)
const unavailable = ref('')
const primary = ref('')
const backups = ref([])
const items = ref([])
const newDomain = ref('')
const makePrimary = ref(false)
const steps = ref([])

// 窄屏卡片字段：与表格列一一对应（ResponsiveDataView 在窄屏自动切换为卡片）
const mobileFields = [
  { key: 'dns_resolved', label: 'DNS', format: (v) => (v ? '已解析' : '未解析') },
  { key: 'vhost_exists', label: '站点配置', format: (v) => (v ? '已配置' : '缺失') },
  { key: 'cert', label: '证书' },
  { key: 'actions', label: '操作' }
]

const certType = (days) => (days > 30 ? 'success' : (days >= 7 ? 'warning' : 'danger'))
const rowClass = ({ row }) => (row.https_ok && row.dns_resolved ? '' : 'dp-row-bad')

// 手动续期：证书未到期时 certbot 会跳过（后端据此提示「无需续期」）；
// force=true 才会强制重签，而 Let's Encrypt 对同一组域名重复签发有每周次数限制，
// 所以强制续期走二次确认，并说明代价。
const renewing = ref(false)
const renewCerts = async (force) => {
  if (force) {
    try {
      await ElMessageBox.confirm(
        "强制续期会重新向 Let's Encrypt 申请一次证书。同一组域名每周签发次数有限（超限会被暂时拒绝），仅在证书确实异常时才需要。确认继续？",
        '强制续期证书',
        { type: 'warning', confirmButtonText: '强制续期', cancelButtonText: '取消' }
      )
    } catch { return }
  }
  renewing.value = true
  try {
    const res = await adminAPI.renewDomainPool({ force })
    const d = res.data?.data || {}
    steps.value = d.steps || []
    ElMessage.success(res.data?.message || '续期完成')
    await load()
  } catch (e) {
    const d = e.response?.data?.data || {}
    if (Array.isArray(d.steps) && d.steps.length) steps.value = d.steps
    const failed = d.result?.failed?.length ? `（失败：${d.result.failed.join('、')}）` : ''
    ElMessage.error((e.response?.data?.message || '续期失败') + failed)
    await load()
  } finally {
    renewing.value = false
  }
}

const load = async () => {
  loading.value = true
  try {
    const res = await adminAPI.getDomainPool()
    const d = res.data?.data || {}
    available.value = d.available !== false
    unavailable.value = d.unavailable || ''
    primary.value = d.primary || ''
    backups.value = d.backups || []
    items.value = d.items || []
  } catch (e) {
    ElMessage.error('加载订阅域名池失败：' + (e.response?.data?.message || e.message))
  } finally {
    loading.value = false
  }
}

const configure = async () => {
  const domain = newDomain.value.trim()
  if (!domain) { ElMessage.warning('请输入域名'); return }
  configuring.value = true
  steps.value = []
  try {
    const res = await adminAPI.configureDomainPool({
      domain,
      make_primary: makePrimary.value
    })
    steps.value = res.data?.data?.steps || []
    const failed = steps.value.find(s => !s.ok)
    if (failed) {
      ElMessage.warning(`配置未完全成功：${failed.name} —— ${failed.detail}`)
    } else {
      ElMessage.success('配置完成，该域名已加入订阅域名池')
      newDomain.value = ''
    }
    await load()
  } catch (e) {
    steps.value = e.response?.data?.data?.steps || []
    const failed = steps.value.find(s => !s.ok)
    ElMessage.error('配置失败：' + (failed ? `${failed.name} —— ${failed.detail}` : (e.response?.data?.message || e.message)))
  } finally {
    configuring.value = false
  }
}

// 站点已在服务器上配好（或手工配好）时，只把它写进域名池
const addOnly = async () => {
  const domain = newDomain.value.trim()
  if (!domain) { ElMessage.warning('请输入域名'); return }
  adding.value = true
  try {
    await adminAPI.setDomainPoolPrimary(domain) // 先确保进池
    ElMessage.success('已加入订阅域名池')
    newDomain.value = ''
    await load()
  } catch (e) {
    ElMessage.error('加入失败：' + (e.response?.data?.message || e.message))
  } finally {
    adding.value = false
  }
}

const repair = (domain) => {
  newDomain.value = domain
  return configure()
}

const setPrimary = async (domain) => {
  try {
    const res = await adminAPI.setDomainPoolPrimary(domain)
    ElMessage.success(res.data?.message || '已切换订阅主域名')
    await load()
  } catch (e) {
    ElMessage.error('切换失败：' + (e.response?.data?.message || e.message))
  }
}

const remove = async (domain) => {
  try {
    await ElMessageBox.confirm(
      `确认从订阅域名池移除 ${domain}？\n\n· 会同时移除该域名的 nginx 站点配置（自动备份）；\n· 证书保留，随时可以一键加回来；\n· 客户已拿到的地址如果正好是这个域名，需要换到剩下的地址。`,
      '移除订阅域名',
      { type: 'warning', confirmButtonText: '确认移除', cancelButtonText: '取消' }
    )
  } catch { return }
  try {
    const res = await adminAPI.removeDomainFromPool(domain)
    steps.value = res.data?.data?.steps || []
    ElMessage.success(res.data?.data?.warn || '已移除')
    await load()
  } catch (e) {
    ElMessage.error('移除失败：' + (e.response?.data?.message || e.message))
  }
}

onMounted(load)
</script>

<style scoped>
.dp-foot-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--el-border-color-lighter);
}
.dp-foot-actions .dp-note { margin-left: 0; }

.domain-pool { margin-top: 8px; }
.dp-head { display: flex; justify-content: space-between; gap: 12px; align-items: flex-start; }
.dp-title { font-size: 14px; font-weight: 600; }
.dp-sub { margin-top: 4px; font-size: 12.5px; line-height: 1.7; color: #909399; max-width: 760px; }
.dp-alert { margin: 10px 0; }
.dp-alert-body { font-size: 12.5px; line-height: 1.7; }
.dp-manual { margin-top: 6px; color: #909399; }
.dp-manual code, .dp-foot code { background: #f5f7fa; padding: 1px 4px; border-radius: 3px; }
.dp-add { display: flex; align-items: center; gap: 10px; margin: 12px 0; flex-wrap: wrap; }
.dp-add-input { max-width: 300px; }
.dp-table { margin-top: 4px; }
.dp-domain { font-weight: 500; }
.dp-tag { margin-left: 6px; }
.dp-note { margin-left: 6px; font-size: 12px; color: #909399; }
.dp-warn { color: #e6a23c; }
.dp-steps { margin-top: 12px; border: 1px solid #ebeef5; border-radius: 6px; padding: 10px 12px; background: #fafafa; }
.dp-steps-title { font-size: 12.5px; font-weight: 600; margin-bottom: 6px; }
.dp-step { display: flex; gap: 8px; font-size: 12.5px; line-height: 1.9; }
.dp-step-icon { width: 14px; color: #67c23a; }
.dp-step.bad .dp-step-icon { color: #f56c6c; }
.dp-step-name { min-width: 190px; }
.dp-step-detail { color: #909399; word-break: break-all; }
.dp-foot { margin-top: 12px; font-size: 12.5px; line-height: 1.7; color: #909399; }
</style>
