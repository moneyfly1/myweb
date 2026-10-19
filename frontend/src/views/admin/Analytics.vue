<template>
  <div class="an-page">
    <!-- 工具条：一行放完（手机端自动换行），不再占半屏 -->
    <div class="an-bar">
      <div class="an-bar__left">
        <el-radio-group v-model="timeRange" size="small" @change="onRangeChange">
          <el-radio-button label="day">今日</el-radio-button>
          <el-radio-button label="month">本月</el-radio-button>
          <el-radio-button label="year">本年</el-radio-button>
        </el-radio-group>
        <span class="an-hint">{{ timeRangeLabel }} · 更新于 {{ updatedAtText }}</span>
      </div>
      <div class="an-bar__right">
        <el-button size="small" :loading="loading" @click="loadData(); reloadCurrent(1)">
          <el-icon><Refresh /></el-icon><span class="an-btn-text">刷新</span>
        </el-button>
        <el-button size="small" :loading="exporting" @click="exportData">
          <el-icon><Download /></el-icon><span class="an-btn-text">导出</span>
        </el-button>
      </div>
    </div>

    <!-- 指标条：紧凑 4 格（手机 2×2） -->
    <div class="an-metrics">
      <div v-for="m in overviewCards" :key="m.label" class="an-metric">
        <div class="an-metric__icon"><el-icon><component :is="m.icon" /></el-icon></div>
        <div class="an-metric__body">
          <div class="an-metric__value">{{ m.value }}</div>
          <div class="an-metric__label">{{ m.label }}</div>
        </div>
      </div>
    </div>

    <el-tabs v-model="activeTab" class="an-tabs">
      <!-- ============ 概览 ============ -->
      <el-tab-pane label="概览" name="overview">
        <div class="an-grid">
          <el-card shadow="never" class="an-card">
            <template #header><span class="an-card__title">收入</span></template>
            <div class="an-income">
              <div class="an-income__item">
                <div class="an-income__num">¥{{ revenueStats.current || '0.00' }}</div>
                <div class="an-income__label">{{ timeRangeLabel }}收入</div>
                <div class="an-income__trend" :class="revenueTrendClass">
                  <el-icon><component :is="revenueTrendIcon" /></el-icon>{{ revenueTrendText }}
                </div>
              </div>
              <div class="an-income__item">
                <div class="an-income__num">{{ revenueStats.order_count || 0 }}</div>
                <div class="an-income__label">{{ timeRangeLabel }}订单</div>
              </div>
              <div class="an-income__item">
                <div class="an-income__num">¥{{ revenueStats.avg_order || '0.00' }}</div>
                <div class="an-income__label">客单价</div>
              </div>
            </div>
          </el-card>

          <el-card shadow="never" class="an-card">
            <template #header><span class="an-card__title">用户留存</span></template>
            <ResponsiveDataView
              :data="retention"
              :fields="mobileRetentionFields"
              :loading="loading"
              title-field="label"
              empty-title="暂无留存数据"
              empty-description="当前时间范围内还没有可展示的留存记录。"
            >
              <template #table>
                <el-table :data="retention" size="small" v-loading="loading">
                  <el-table-column label="指标" min-width="80">
                    <template #default="{ row }">{{ row.label || `第${row.day}天` }}</template>
                  </el-table-column>
                  <el-table-column prop="total" label="基数" width="70" align="center" />
                  <el-table-column prop="retained" label="达标" width="70" align="center" />
                  <el-table-column label="留存率" min-width="120">
                    <template #default="{ row }">
                      <el-progress :percentage="Math.round(row.rate || 0)" :stroke-width="12"
                        :text-inside="true" :color="getProgressColor(row.rate || 0)" />
                    </template>
                  </el-table-column>
                </el-table>
              </template>
              <template #header="{ item }">
                <span class="an-card__title">{{ item.label || `第${item.day}天` }}</span>
                <el-tag size="small" :type="(item.rate || 0) >= 70 ? 'success' : (item.rate || 0) >= 40 ? 'warning' : 'danger'">
                  {{ Math.round(item.rate || 0) }}%
                </el-tag>
              </template>
              <template #field-rate="{ item }">
                <el-progress :percentage="Math.round(item.rate || 0)" :stroke-width="10"
                  :color="getProgressColor(item.rate || 0)" />
              </template>
            </ResponsiveDataView>
          </el-card>

          <el-card shadow="never" class="an-card">
            <template #header><span class="an-card__title">设备类型 / 系统</span></template>
            <div class="an-mini-two">
              <div>
                <div class="an-sub">设备类型</div>
                <el-table :data="deviceStats" size="small" :show-header="false" empty-text="暂无">
                  <el-table-column min-width="90">
                    <template #default="{ row }">{{ getDeviceTypeName(row.device_type) }}</template>
                  </el-table-column>
                  <el-table-column width="60" align="right">
                    <template #default="{ row }">{{ row.count }}</template>
                  </el-table-column>
                  <el-table-column width="70" align="right">
                    <template #default="{ row }">
                      <el-tag size="small" effect="plain">{{ getPercentage(row.count, totalDevices) }}%</el-tag>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
              <div>
                <div class="an-sub">系统</div>
                <el-table :data="osStats" size="small" :show-header="false" empty-text="暂无">
                  <el-table-column min-width="90">
                    <template #default="{ row }">{{ getDeviceTypeName(row.device_type) }}</template>
                  </el-table-column>
                  <el-table-column width="60" align="right">
                    <template #default="{ row }">{{ row.count }}</template>
                  </el-table-column>
                  <el-table-column width="70" align="right">
                    <template #default="{ row }">
                      <el-tag size="small" type="success" effect="plain">{{ getPercentage(row.count, totalOS) }}%</el-tag>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </div>
          </el-card>
        </div>
      </el-tab-pane>

      <!-- ============ 设备与客户端 ============ -->
      <el-tab-pane label="设备与客户端" name="devices">
        <!-- 视图切换：按用户（谁在用哪个软件/版本）/ 按分组（软件、版本、机型的分布） -->
        <div class="an-filter an-filter--view">
          <el-radio-group v-model="duView" size="small" @change="onViewChange">
            <el-radio-button label="user">按用户</el-radio-button>
            <el-radio-button label="group">按分组</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 筛选条：一行（手机端逐行堆叠） -->
        <div class="an-filter">
          <el-select v-if="duView === 'group'" v-model="du.dim" size="small" class="an-filter__dim"
            @change="loadDeviceUsage(1)">
            <el-option v-for="d in duDims" :key="d.key" :label="d.label" :value="d.key" />
          </el-select>
          <el-input v-model="du.q" size="small" class="an-filter__search" clearable
            placeholder="搜索型号 / 系统 / 客户端 / 版本"
            @keyup.enter="reloadCurrent(1)" @clear="reloadCurrent(1)">
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-radio-group v-model="du.official" size="small" @change="reloadCurrent(1)">
            <el-radio-button label="all">全部</el-radio-button>
            <el-radio-button label="official">官方</el-radio-button>
            <el-radio-button label="third_party">第三方</el-radio-button>
          </el-radio-group>
          <el-select v-model="du.days" size="small" class="an-filter__days" @change="reloadCurrent(1)">
            <el-option label="全部时间" value="0" />
            <el-option label="近 7 天" value="7" />
            <el-option label="近 30 天" value="30" />
          </el-select>
        </div>

        <!-- 汇总行：一行文字，不占卡片 -->
        <div class="an-summary">
          <span>设备 <b>{{ duSummary.devices || 0 }}</b></span>
          <span class="an-dot">·</span>
          <span>用户 <b>{{ duSummary.users || 0 }}</b></span>
          <span class="an-dot">·</span>
          <span class="an-ok">官方客户端 <b>{{ duSummary.official_devices || 0 }}</b></span>
          <span class="an-dot">·</span>
          <span class="an-warn an-clickable" @click="quickThirdParty">
            第三方客户端 <b>{{ duSummary.third_party_devices || 0 }}</b>（点击只看第三方）
          </span>
          <template v-if="duSummary.automation_devices">
            <span class="an-dot">·</span>
            <el-tooltip content="curl / 脚本 / 链接预览爬虫等抓取请求，不是用户设备" placement="top">
              <span class="an-muted">其中自动化请求 <b>{{ duSummary.automation_devices }}</b> 台（非设备）</span>
            </el-tooltip>
          </template>
          <span class="an-summary__spacer"></span>
          <template v-if="duView === 'user'">
            <el-button size="small" type="primary" plain :disabled="!selectedUsers.length"
              @click="notifySelectedUsers()">
              提醒所选用户<span v-if="selectedUsers.length">（{{ selectedUsers.length }}）</span>
            </el-button>
          </template>
          <template v-else>
            <el-button size="small" type="primary" plain :disabled="!selectedGroups.length"
              @click="openNotify({ groups: selectedGroups.map(g => g.key) })">
              提醒所选分组<span v-if="selectedGroups.length">（{{ selectedGroups.length }}）</span>
            </el-button>
          </template>
        </div>

        <!-- ===== 按用户：谁在用哪个软件/版本，每行可直接发升级通知 ===== -->
        <ResponsiveDataView
          v-if="duView === 'user'"
          :data="duUsers"
          :fields="mobileUserFields"
          :loading="loadingUsers"
          :selectable="true"
          :selected-rows="selectedUsers"
          id-field="user_id"
          title-field="username"
          empty-title="暂无用户数据"
          empty-description="换个筛选条件或时间范围试试。"
          @selection-change="onUserSelection"
        >
          <template #table>
            <el-table :data="duUsers" size="small" v-loading="loadingUsers"
              @selection-change="onUserSelection" class="an-table">
              <el-table-column type="selection" width="42" />
              <el-table-column label="用户" min-width="150">
                <template #default="{ row }">
                  <div class="an-user">
                    <span class="an-user__name">
                      {{ row.username || `用户#${row.user_id}` }}
                      <el-tag v-if="!row.user_active" size="small" type="info" effect="plain">已停用</el-tag>
                    </span>
                    <span class="an-sub">{{ row.email || '无邮箱' }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="使用的客户端（软件 + 版本）" min-width="260">
                <template #default="{ row }">
                  <div class="an-clients">
                    <el-tag v-for="c in row.clients" :key="c" size="small" effect="plain"
                      :type="isOfficialClient(c) ? 'success' : 'warning'">{{ c }}</el-tag>
                    <span v-if="!row.clients || !row.clients.length" class="an-sub">—</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="devices" label="设备" width="66" align="center" />
              <el-table-column label="机型/系统" min-width="120">
                <template #default="{ row }">
                  <span class="an-sub">{{ (row.brands || []).join('、') || '—' }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="last_access" label="最近活跃" width="126" />
              <el-table-column label="订阅到期" width="100">
                <template #default="{ row }"><span class="an-sub">{{ row.sub_expire || '—' }}</span></template>
              </el-table-column>
              <el-table-column label="操作" width="118" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" :disabled="!row.email"
                    @click.stop="notifySelectedUsers([row])">
                    发送升级通知
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </template>
          <template #header="{ item }">
            <div class="an-user">
              <span class="an-user__name">{{ item.username || `用户#${item.user_id}` }}</span>
              <span class="an-sub">{{ item.email || '无邮箱' }}</span>
            </div>
          </template>
          <template #field-clients="{ item }">
            <div class="an-clients">
              <el-tag v-for="c in (item.clients || [])" :key="c" size="small" effect="plain"
                :type="isOfficialClient(c) ? 'success' : 'warning'">{{ c }}</el-tag>
            </div>
          </template>
          <template #field-devices="{ value }"><span class="an-kv">设备 <b>{{ value }}</b></span></template>
          <template #field-brands="{ item }">
            <span class="an-sub">{{ (item.brands || []).join('、') || '—' }}</span>
          </template>
          <template #field-last_access="{ value }"><span class="an-sub">{{ value }}</span></template>
          <template #actions="{ item }">
            <el-button size="small" type="primary" plain :disabled="!item.email"
              @click.stop="notifySelectedUsers([item])">发送升级通知</el-button>
          </template>
        </ResponsiveDataView>

        <div v-if="duView === 'user'" class="an-pager">
          <PaginationBar :current-page="duUserPage" :page-size="du.limit" :total="duUserTotal"
            :page-sizes="[20, 50, 100, 200]"
            @update:current-page="loadDeviceUsers" @update:page-size="onLimitChange" />
        </div>

        <!-- ===== 按分组：软件/版本/机型的分布统计 ===== -->
        <ResponsiveDataView
          v-else
          :data="duGroups"
          :fields="mobileGroupFields"
          :loading="loadingGroups"
          :selectable="true"
          :selected-rows="selectedGroups"
          id-field="key"
          title-field="label"
          empty-title="暂无设备数据"
          empty-description="换个筛选条件或时间范围试试。"
          @selection-change="onGroupSelection"
        >
          <template #table>
            <el-table ref="groupTableRef" :data="duGroups" size="small" v-loading="loadingGroups"
              @selection-change="onGroupSelection" @row-click="openDetail" class="an-table">
              <el-table-column type="selection" width="42" />
              <el-table-column :label="duDimLabel" min-width="170">
                <template #default="{ row }">
                  <div class="an-group">
                    <span class="an-group__name">{{ row.label || '未知' }}</span>
                    <el-tag v-if="row.official" size="small" type="success" effect="plain">官方</el-tag>
                    <el-tag v-else-if="isThirdPartyDim" size="small" type="warning" effect="plain">第三方</el-tag>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="devices" label="设备" width="76" align="center" />
              <el-table-column prop="users" label="用户" width="76" align="center" />
              <el-table-column label="占比" min-width="130">
                <template #default="{ row }">
                  <div class="an-share">
                    <el-progress :percentage="Math.min(100, Math.round(row.share || 0))" :stroke-width="8"
                      :show-text="false" />
                    <span>{{ (row.share || 0).toFixed(1) }}%</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="last_access" label="最近活跃" width="126" />
              <el-table-column label="操作" width="78" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click.stop="openDetail(row)">详情</el-button>
                </template>
              </el-table-column>
            </el-table>
          </template>
          <template #header="{ item }">
            <div class="an-group">
              <span class="an-group__name">{{ item.label || '未知' }}</span>
              <el-tag v-if="item.official" size="small" type="success" effect="plain">官方</el-tag>
              <el-tag v-else-if="isThirdPartyDim" size="small" type="warning" effect="plain">第三方</el-tag>
            </div>
          </template>
          <template #field-devices="{ value }"><span class="an-kv">设备 <b>{{ value }}</b></span></template>
          <template #field-users="{ value }"><span class="an-kv">用户 <b>{{ value }}</b></span></template>
          <template #field-share="{ value }">
            <el-progress :percentage="Math.min(100, Math.round(value || 0))" :stroke-width="8" />
          </template>
          <template #actions="{ item }">
            <el-button size="small" type="primary" plain @click.stop="openDetail(item)">查看用户</el-button>
          </template>
        </ResponsiveDataView>

        <div v-if="duView !== 'user'" class="an-pager">
          <PaginationBar :current-page="du.page" :page-size="du.limit" :total="duGroupTotal"
            :page-sizes="[20, 50, 100, 200]"
            @update:current-page="loadDeviceUsage" @update:page-size="onLimitChange" />
        </div>
      </el-tab-pane>

      <!-- ============ 用户预警 ============ -->
      <el-tab-pane :label="churnUsers.length ? `用户预警 (${churnUsers.length})` : '用户预警'" name="churn">
        <ResponsiveDataView
          :data="churnUsers"
          :fields="mobileChurnFields"
          :loading="loading"
          title-field="username"
          empty-title="暂无预警用户"
          empty-description="当前没有符合流失预警条件的用户。"
        >
          <template #table>
            <el-table :data="churnUsers" size="small" v-loading="loading" class="an-table">
              <el-table-column prop="id" label="ID" width="66" />
              <el-table-column prop="username" label="用户名" min-width="110" show-overflow-tooltip />
              <el-table-column prop="email" label="邮箱" min-width="170" show-overflow-tooltip />
              <el-table-column label="最后登录" width="118">
                <template #default="{ row }">
                  <span :class="{ 'an-danger': isLongTimeNoLogin(row.last_login) }">{{ formatDate(row.last_login) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="订阅到期" width="118">
                <template #default="{ row }">
                  <span :class="{ 'an-warn-text': isExpiringSoon(row.expire_time) }">{{ formatDate(row.expire_time) }}</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="78" fixed="right">
                <template #default="{ row }">
                  <el-button size="small" type="primary" plain @click="openContactDialog(row)">联系</el-button>
                </template>
              </el-table-column>
            </el-table>
          </template>
          <template #header="{ item }">
            <div class="an-group">
              <span class="an-group__name">{{ item.username }}</span>
            </div>
          </template>
          <template #field-last_login="{ item }">
            <span :class="{ 'an-danger': isLongTimeNoLogin(item.last_login) }">{{ formatDate(item.last_login) }}</span>
          </template>
          <template #field-expire_time="{ item }">
            <span :class="{ 'an-warn-text': isExpiringSoon(item.expire_time) }">{{ formatDate(item.expire_time) }}</span>
          </template>
          <template #actions="{ item }">
            <el-button size="small" type="primary" plain @click="openContactDialog(item)">联系</el-button>
          </template>
        </ResponsiveDataView>
      </el-tab-pane>
    </el-tabs>

    <!-- ============ 抽屉：某分组下的用户/设备明细 ============ -->
    <AppDrawer v-model="detail.visible" :title="detailTitle" size="780px" mobile-size="100%">
      <div class="an-filter an-filter--drawer">
        <el-input v-model="detail.q" size="small" clearable placeholder="搜索本组内的型号 / 邮箱 / 版本"
          @keyup.enter="loadDetail(1)" @clear="loadDetail(1)">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button size="small" type="primary" plain :disabled="!detail.selected.length"
          @click="openNotify({ userIds: detail.selected.map(r => r.user_id) })">
          提醒所选（{{ detail.selected.length }}）
        </el-button>
      </div>

      <ResponsiveDataView
        :data="detail.items"
        :fields="mobileDetailFields"
        :loading="detail.loading"
        :selectable="true"
        :selected-rows="detail.selected"
        id-field="device_id"
        title-field="username"
        empty-title="暂无数据"
        @selection-change="onDetailSelection"
      >
        <template #table>
          <el-table :data="detail.items" size="small" v-loading="detail.loading"
            max-height="440" @selection-change="onDetailSelection" class="an-table">
            <el-table-column type="selection" width="42" />
            <el-table-column label="用户" min-width="128">
              <template #default="{ row }">
                <div class="an-user">
                  <span class="an-user__name">{{ row.username || `用户#${row.user_id}` }}</span>
                  <span class="an-sub">{{ row.email || '无邮箱' }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="设备" min-width="126">
              <template #default="{ row }">
                <div class="an-user">
                  <span>{{ row.device_model || row.device_brand || '未知设备' }}</span>
                  <span class="an-sub">{{ row.os_name }}{{ row.os_version ? ' ' + row.os_version : '' }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="客户端" min-width="118">
              <template #default="{ row }">
                <el-tag size="small" :type="row.official ? 'success' : 'warning'" effect="plain">
                  {{ row.software_name || '未知' }}
                </el-tag>
                <span class="an-sub"> {{ row.software_version }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="last_access" label="最近活跃" width="118" />
            <el-table-column label="订阅到期" width="100">
              <template #default="{ row }"><span class="an-sub">{{ row.sub_expire || '—' }}</span></template>
            </el-table-column>
          </el-table>
        </template>
        <template #header="{ item }">
          <div class="an-user">
            <span class="an-user__name">{{ item.username || `用户#${item.user_id}` }}</span>
            <span class="an-sub">{{ item.email || '无邮箱' }}</span>
          </div>
        </template>
        <template #field-device_model="{ item }">
          <span>{{ item.device_model || item.device_brand || '未知设备' }}</span>
          <span class="an-sub"> {{ item.os_name }}</span>
        </template>
        <template #field-software_name="{ item }">
          <el-tag size="small" :type="item.official ? 'success' : 'warning'" effect="plain">{{ item.software_name }}</el-tag>
          <span class="an-sub"> {{ item.software_version }}</span>
        </template>
        <template #field-last_access="{ value }"><span class="an-sub">{{ value }}</span></template>
      </ResponsiveDataView>

      <div class="an-pager">
        <PaginationBar :current-page="detail.page" :page-size="detail.limit" :total="detail.total"
          :page-sizes="[20, 50, 100]"
          @update:current-page="loadDetail" @update:page-size="onDetailLimitChange" />
      </div>
      <div class="an-drawer-tip">
        共 {{ detail.total }} 条设备记录 · 去重后 {{ detail.distinctUsers }} 人（同一个人多台设备只发一封）
      </div>
    </AppDrawer>

    <!-- ============ 抽屉：批量升级提醒 ============ -->
    <AppDrawer v-model="notify.visible" title="发送升级提醒邮件" size="680px" mobile-size="100%">
      <el-alert v-if="notify.smtpConfigured === false" type="warning" :closable="false" show-icon
        title="当前未配置 SMTP，邮件只会入队但发不出去"
        description="请先到「系统设置 → 邮件」填写 SMTP 账号密码。" class="an-alert" />

      <div class="an-kv-grid">
        <div><span class="an-sub">目标</span><b>{{ notify.targetLabel }}</b></div>
        <div><span class="an-sub">可发送</span><b class="an-ok">{{ notify.previewInfo?.targets ?? '—' }} 人</b></div>
        <div v-if="notify.previewInfo?.smtp_configured">
          <span class="an-sub">SMTP</span>
          <b>{{ notify.previewInfo.smtp_host }}<span class="an-sub"> · {{ notify.previewInfo.smtp_user }}</span></b>
        </div>
        <div v-if="notify.previewInfo?.skipped_no_email"><span class="an-sub">缺邮箱</span><b>{{ notify.previewInfo.skipped_no_email }}</b></div>
        <div v-if="notify.previewInfo?.skipped_unsubscribed"><span class="an-sub">已退订</span><b>{{ notify.previewInfo.skipped_unsubscribed }}</b></div>
      </div>

      <el-form label-width="62px" label-position="left" class="an-form">
        <el-form-item label="主题">
          <el-input v-model="notify.subject" size="small" placeholder="留空使用内置模板主题" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="notify.note" size="small" type="textarea" :rows="2"
            placeholder="可选：会显示在邮件正文的提示框里" />
        </el-form-item>
      </el-form>

      <div class="an-preview">
        <div class="an-preview__head">
          <span class="an-card__title">邮件预览</span>
          <el-button size="small" :loading="notify.previewing" @click="previewNotify()">生成预览（不发送）</el-button>
        </div>
        <div v-if="notify.preview?.subject" class="an-preview__subject">{{ notify.preview.subject }}</div>
        <div v-if="notify.preview?.content" class="an-preview__body" v-html="notify.preview.content"></div>
        <div v-else class="an-sub">点「生成预览」先看渲染结果（含各平台加速镜像下载地址），确认后再发送。</div>
      </div>

      <template #footer>
        <div class="an-footer">
          <el-button @click="notify.visible = false">取消</el-button>
          <el-button type="primary" :loading="notify.sending" :disabled="!notify.preview" @click="sendNotify">
            确认发送{{ notify.previewInfo?.targets ? `（${notify.previewInfo.targets} 人）` : '' }}
          </el-button>
        </div>
      </template>
    </AppDrawer>

    <!-- ============ 抽屉：联系单个用户（保留原有能力） ============ -->
    <AppDrawer v-model="contactDialogVisible" title="联系用户" size="600px" mobile-size="100%" :loading="sending">
      <el-form :model="contactForm" label-width="80px" label-position="left">
        <el-form-item label="用户"><el-input v-model="contactForm.username" disabled /></el-form-item>
        <el-form-item label="邮箱"><el-input v-model="contactForm.email" disabled /></el-form-item>
        <el-form-item label="邮件类型">
          <el-select v-model="contactForm.templateName" placeholder="请选择邮件模板" @change="onTemplateChange" style="width:100%">
            <el-option v-for="t in emailTemplates" :key="t.name" :label="t.label" :value="t.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="邮件主题"><el-input v-model="contactForm.subject" placeholder="请输入邮件主题" /></el-form-item>
        <el-form-item label="邮件内容">
          <el-input v-model="contactForm.content" type="textarea" :rows="isMobile ? 8 : 7" placeholder="请输入邮件内容" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="an-footer">
          <el-button @click="contactDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="sending" @click="sendEmail">发送邮件</el-button>
        </div>
      </template>
    </AppDrawer>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from '@/utils/elementPlusServices'
import {
  Refresh, User, TrendCharts, DataAnalysis, Monitor, Download, Top, Bottom, Search
} from '@element-plus/icons-vue'
import { api } from '@/utils/api'
import { formatDateTimeSafe } from '@/utils/date'
import AppDrawer from '@/components/AppDrawer.vue'
import ResponsiveDataView from '@/components/ResponsiveDataView.vue'
import { useMobile } from '@/composables/useMobile'

const isMobile = useMobile()

// ---------------- 概览 ----------------
const loading = ref(false)
const exporting = ref(false)
const timeRange = ref('day')
const activeTab = ref('overview')
const userAnalytics = ref({})
const retention = ref([])
const churnUsers = ref([])
const deviceStats = ref([])
const osStats = ref([])
const revenueStats = ref({})
const updatedAt = ref(null)
let loadSeq = 0

const timeRangeLabel = computed(() => ({ day: '今日', month: '本月', year: '本年' }[timeRange.value] || '今日'))
const updatedAtText = computed(() =>
  updatedAt.value ? formatDateTimeSafe(updatedAt.value, 'HH:mm:ss', '—') : '—')

const overviewCards = computed(() => [
  { label: '日活跃', value: userAnalytics.value.dau || 0, icon: User },
  { label: '周活跃', value: userAnalytics.value.wau || 0, icon: TrendCharts },
  { label: '月活跃', value: userAnalytics.value.mau || 0, icon: DataAnalysis },
  { label: '总用户', value: userAnalytics.value.total_users || 0, icon: Monitor }
])

const revenueTrendClass = computed(() => ((revenueStats.value.change_rate || 0) >= 0 ? 'an-up' : 'an-down'))
const revenueTrendIcon = computed(() => ((revenueStats.value.change_rate || 0) >= 0 ? Top : Bottom))
const revenueTrendText = computed(() => {
  const change = revenueStats.value.change_rate || 0
  return `${change >= 0 ? '较上期增长' : '较上期下降'} ${Math.abs(change).toFixed(1)}%`
})

const totalDevices = computed(() => deviceStats.value.reduce((s, i) => s + (i.count || 0), 0))
const totalOS = computed(() => osStats.value.reduce((s, i) => s + (i.count || 0), 0))

const deviceTypeMap = {
  mobile: '移动设备', desktop: '桌面设备', tablet: '平板设备', unknown: '未知设备',
  ios: 'iOS', android: 'Android', windows: 'Windows', macos: 'macOS', linux: 'Linux'
}
const getDeviceTypeName = (t) => deviceTypeMap[(t || '').toLowerCase()] || t || '未知'
const getPercentage = (v, total) => (total ? ((v / total) * 100).toFixed(1) : 0)
const getProgressColor = (rate) => (rate >= 70 ? '#67c23a' : rate >= 40 ? '#e6a23c' : '#f56c6c')
const formatDate = (d) => formatDateTimeSafe(d, 'YYYY-MM-DD', '-')
const isLongTimeNoLogin = (t) => {
  if (!t) return true
  return Math.floor((Date.now() - new Date(t).getTime()) / 86400000) > 7
}
const isExpiringSoon = (t) => {
  if (!t) return false
  const d = Math.floor((new Date(t).getTime() - Date.now()) / 86400000)
  return d <= 7 && d >= 0
}

const mobileRetentionFields = [
  { key: 'total', label: '基数' },
  { key: 'retained', label: '达标' },
  { key: 'rate', label: '留存率', fullWidth: true }
]
const mobileChurnFields = [
  { key: 'last_login', label: '最后登录' },
  { key: 'expire_time', label: '订阅到期' }
]

const onRangeChange = () => loadData()

const loadData = async () => {
  const seq = ++loadSeq
  const range = timeRange.value
  loading.value = true
  try {
    const [revenueRes, uRes, rRes, cRes, dRes] = await Promise.all([
      api.get(`/admin/analytics/revenue?range=${range}`),
      api.get(`/admin/analytics/users?range=${range}`),
      api.get(`/admin/analytics/retention?range=${range}`),
      api.get(`/admin/analytics/churn?range=${range}`),
      api.get(`/admin/analytics/devices?range=${range}`)
    ])
    if (seq !== loadSeq) return
    revenueStats.value = revenueRes.data?.data || {}
    userAnalytics.value = uRes.data?.data || {}
    retention.value = rRes.data?.data || []
    churnUsers.value = cRes.data?.data || []
    const dev = dRes.data?.data || {}
    deviceStats.value = dev.device_types || []
    osStats.value = dev.os_stats || []
    updatedAt.value = new Date()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '加载数据失败')
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

// ---------------- 设备与客户端使用 ----------------
const du = reactive({ dim: 'client', q: '', official: 'all', days: '0', page: 1, limit: 20 })
// 视图：user = 按用户（谁在用哪个软件/版本，可直接发升级通知）；group = 按分组统计
const duView = ref('user')
const duUsers = ref([])
const duUserTotal = ref(0)
const duUserPage = ref(1)
const loadingUsers = ref(false)
const selectedUsers = ref([])
const duDims = ref([
  { key: 'client', label: '客户端软件' },
  { key: 'client_version', label: '客户端版本' },
  { key: 'platform', label: '平台（系统）' },
  { key: 'ios_model', label: 'iOS 设备型号' },
  { key: 'ios_version', label: 'iOS 系统版本' },
  { key: 'os_version', label: '系统版本' },
  { key: 'device_type', label: '设备类型' },
  { key: 'brand', label: '设备品牌' }
])
const duGroups = ref([])
const duSummary = ref({})
const duGroupTotal = ref(0)
const duDimLabel = ref('客户端软件')
const loadingGroups = ref(false)
const groupTableRef = ref(null)
const selectedGroups = ref([])

const isThirdPartyDim = computed(() => ['client', 'client_version'].includes(du.dim))

const loadDeviceUsage = async (page = 1) => {
  du.page = page
  loadingGroups.value = true
  try {
    const res = await api.get('/admin/analytics/device-usage', {
      params: {
        dim: du.dim, q: du.q || undefined, official: du.official,
        days: du.days, page: du.page, limit: du.limit
      }
    })
    const data = res.data?.data || {}
    duGroups.value = data.groups || []
    duSummary.value = data.summary || {}
    duGroupTotal.value = data.group_total || 0
    duDimLabel.value = data.dim_label || '分组'
    if (Array.isArray(data.dims) && data.dims.length) duDims.value = data.dims
    selectedGroups.value = []
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '设备数据加载失败')
  } finally {
    loadingGroups.value = false
  }
}

const onGroupSelection = (rows) => { selectedGroups.value = rows }
const onUserSelection = (rows) => { selectedUsers.value = rows }

/// 视图切换：切到哪个就加载哪个（两个视图共用同一套筛选条件）
const onViewChange = () => {
  if (duView.value === 'user') loadDeviceUsers(1)
  else loadDeviceUsage(1)
}

/// 每页条数变化：改 limit 后回到第 1 页（参考其他列表的交互）
const onLimitChange = (size) => {
  du.limit = size
  reloadCurrent(1)
}
const onDetailLimitChange = (size) => {
  detail.limit = size
  loadDetail(1)
}

/// 筛选条件变化：按当前视图重新加载
const reloadCurrent = (page = 1) => {
  if (duView.value === 'user') loadDeviceUsers(page)
  else loadDeviceUsage(page)
}

/// 按用户列出「谁在用哪个软件/版本」
const loadDeviceUsers = async (page = 1) => {
  duUserPage.value = page
  loadingUsers.value = true
  try {
    const res = await api.get('/admin/analytics/device-usage/users', {
      params: {
        q: du.q || undefined, official: du.official,
        days: du.days, page, limit: du.limit,
      }
    })
    const data = res.data?.data || {}
    duUsers.value = data.items || []
    duUserTotal.value = data.total || 0
    selectedUsers.value = []
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '用户列表加载失败')
  } finally {
    loadingUsers.value = false
  }
}

/// 某条客户端标签是不是官方客户端（MoneyFly）
const isOfficialClient = (label) =>
    (label || '').toLowerCase().startsWith('moneyfly')

/// 给「选中的用户」或「指定用户」发升级通知：直接进预览抽屉，确认后才发送
const notifySelectedUsers = (rows) => {
  const list = (rows && rows.length ? rows : selectedUsers.value) || []
  const ids = [...new Set(list.map(r => r.user_id).filter(Boolean))]
  if (!ids.length) {
    ElMessage.warning('请先勾选要提醒的用户')
    return
  }
  const withoutEmail = list.filter(r => !r.email)
  if (withoutEmail.length) {
    ElMessage.warning(`${withoutEmail.length} 位用户没有邮箱，将自动跳过`)
  }
  openNotify({ userIds: ids })
}

/// 一键切到「第三方客户端」筛选（汇总行里的数字可点）
const quickThirdParty = () => {
  du.official = 'third_party'
  loadDeviceUsage(1)
}

// ---------------- 明细抽屉 ----------------
const detail = reactive({
  visible: false, loading: false, dim: 'client', key: '', label: '',
  q: '', page: 1, limit: 20, items: [], total: 0, distinctUsers: 0, selected: []
})
const detailTitle = computed(() =>
  `${duDimLabel.value}：${detail.label || '全部'}${detail.total ? `（${detail.total} 台设备）` : ''}`)

const openDetail = async (row) => {
  detail.dim = du.dim
  detail.key = row.key
  detail.label = row.label
  detail.q = ''
  detail.selected = []
  detail.visible = true
  await loadDetail(1)
}

const loadDetail = async (page = 1) => {
  detail.page = page
  detail.loading = true
  try {
    const res = await api.get('/admin/analytics/device-usage/detail', {
      params: {
        dim: detail.dim, key: detail.key, q: detail.q || undefined,
        official: du.official, days: du.days, page: detail.page, limit: detail.limit
      }
    })
    const data = res.data?.data || {}
    detail.items = data.items || []
    detail.total = data.total || 0
    detail.distinctUsers = data.distinct_users || 0
    detail.selected = []
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '明细加载失败')
  } finally {
    detail.loading = false
  }
}

const onDetailSelection = (rows) => { detail.selected = rows }

const mobileUserFields = [
  { key: 'clients', label: '使用的客户端', fullWidth: true },
  { key: 'devices', label: '设备' },
  { key: 'brands', label: '机型/系统', fullWidth: true },
  { key: 'last_access', label: '最近活跃' }
]
const mobileGroupFields = [
  { key: 'devices', label: '设备' },
  { key: 'users', label: '用户' },
  { key: 'share', label: '占比', fullWidth: true }
]
const mobileDetailFields = [
  { key: 'device_model', label: '设备' },
  { key: 'software_name', label: '客户端' },
  { key: 'last_access', label: '最近活跃' }
]

// ---------------- 批量升级提醒 ----------------
const notify = reactive({
  visible: false, sending: false, previewing: false,
  targetKind: '', groupKeys: [], userIds: [], targetLabel: '', targetCount: 0,
  subject: '', note: '', preview: null, previewInfo: null, result: null, smtpConfigured: null
})

/// 打开提醒抽屉：
///  - groups: 表格里勾选的分组（后端展开成这些分组下的用户）
///  - userIds: 抽屉里勾选的用户
const openNotify = async ({ groups, userIds } = {}) => {
  notify.groupKeys = groups || []
  notify.userIds = userIds || []
  notify.targetKind = (userIds && userIds.length) ? 'users' : 'groups'
  notify.subject = ''
  notify.note = ''
  notify.preview = null
  notify.previewInfo = null
  notify.result = null
  notify.smtpConfigured = null
  notify.visible = true

  if (notify.targetKind === 'users') {
    notify.targetLabel = `${notify.userIds.length} 位选中用户`
    notify.targetCount = notify.userIds.length
  } else {
    notify.targetLabel = `${notify.groupKeys.length} 个所选分组`
    notify.targetCount = selectedGroups.value.reduce((s, g) => s + (g.users || 0), 0)
  }
  await previewNotify(true)
}

const notifyPayload = (dryRun) => {
  const p = {
    dim: du.dim,
    official: notify.targetKind === 'users' ? 'third_party' : du.official,
    days: du.days,
    q: du.q || undefined,
    subject: notify.subject || undefined,
    note: notify.note || undefined,
    dry_run: dryRun
  }
  if (notify.targetKind === 'users') return { ...p, user_ids: notify.userIds }
  return { ...p, keys: notify.groupKeys }
}

const previewNotify = async (silent = false) => {
  notify.previewing = true
  try {
    const res = await api.post('/admin/analytics/device-usage/notify-upgrade', notifyPayload(true))
    const data = res.data?.data || {}
    notify.previewInfo = data
    notify.smtpConfigured = data.smtp_configured
    if (data.targets) notify.targetCount = data.targets
    if (data.preview) notify.preview = data.preview
    if (!silent) {
      if (data.targets) ElMessage.success(`将发送给 ${data.targets} 人（预览已生成）`)
      else ElMessage.warning('当前条件下没有可发送的用户')
    }
  } catch (e) {
    if (!silent) ElMessage.error(e.response?.data?.message || '生成预览失败')
  } finally {
    notify.previewing = false
  }
}

const sendNotify = async () => {
  try {
    await ElMessageBox.confirm(
      `确认向 ${notify.targetCount} 人发送升级提醒邮件？发送后进入邮件队列，无法撤回。`,
      '确认发送', { type: 'warning', confirmButtonText: '确认发送', cancelButtonText: '取消' })
  } catch { return }
  notify.sending = true
  try {
    const res = await api.post('/admin/analytics/device-usage/notify-upgrade', notifyPayload(false))
    const data = res.data?.data || {}
    notify.result = data
    const skipped = (data.skipped_no_email || 0) + (data.skipped_inactive || 0) + (data.skipped_unsubscribed || 0)
    ElMessage.success(`已入队 ${data.queued || 0} 封${skipped ? `，跳过 ${skipped} 人` : ''}${data.failed ? `，失败 ${data.failed}` : ''}`)
    if (data.failed && data.fail_reasons?.length) ElMessage.warning(data.fail_reasons[0])
    notify.visible = false
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '发送失败')
  } finally {
    notify.sending = false
  }
}

// ---------------- 联系单个用户（保留原有功能） ----------------
const contactDialogVisible = ref(false)
const sending = ref(false)
const contactForm = ref({ userId: null, username: '', email: '', templateName: '', subject: '', content: '' })
const emailTemplates = ref([
  { name: 'subscription_expiring', label: '订阅即将到期提醒' },
  { name: 'subscription_expired', label: '订阅已到期通知' },
  { name: 'user_recall', label: '流失用户召回' },
  { name: 'custom', label: '自定义邮件' }
])

const openContactDialog = (user) => {
  contactForm.value = { userId: user.id, username: user.username, email: user.email, templateName: '', subject: '', content: '' }
  const daysDiff = Math.floor((new Date(user.expire_time).getTime() - Date.now()) / 86400000)
  if (daysDiff < 0) contactForm.value.templateName = 'subscription_expired'
  else if (daysDiff <= 7) contactForm.value.templateName = 'subscription_expiring'
  else if (isLongTimeNoLogin(user.last_login)) contactForm.value.templateName = 'user_recall'
  onTemplateChange()
  contactDialogVisible.value = true
}

const onTemplateChange = async () => {
  const name = contactForm.value.templateName
  if (!name || name === 'custom') return
  try {
    const res = await api.get(`/admin/email-templates/${name}`)
    if (res.data?.data) {
      contactForm.value.subject = res.data.data.subject
      contactForm.value.content = res.data.data.content
    }
  } catch { /* 模板不存在时让管理员手写 */ }
}

const sendEmail = async () => {
  if (!contactForm.value.subject || !contactForm.value.content) {
    ElMessage.warning('请填写邮件主题和内容')
    return
  }
  sending.value = true
  try {
    await api.post('/admin/users/send-email', {
      user_id: contactForm.value.userId, email: contactForm.value.email,
      subject: contactForm.value.subject, content: contactForm.value.content,
      template_name: contactForm.value.templateName
    })
    ElMessage.success('邮件发送成功')
    contactDialogVisible.value = false
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '邮件发送失败')
  } finally {
    sending.value = false
  }
}

// ---------------- 导出 ----------------
const exportData = async () => {
  exporting.value = true
  try {
    let csv = '\uFEFF'
    csv += '用户分析数据导出\n'
    csv += `导出时间,${new Date().toLocaleString('zh-CN')}\n时间范围,${timeRangeLabel.value}\n\n`
    csv += '收入统计\n指标,数值\n'
    csv += `${timeRangeLabel.value}收入,¥${revenueStats.value.current || '0.00'}\n`
    csv += `订单数量,${revenueStats.value.order_count || 0}\n客单价,¥${revenueStats.value.avg_order || '0.00'}\n\n`
    csv += '用户活跃\n指标,数值\n'
    csv += `DAU,${userAnalytics.value.dau || 0}\nWAU,${userAnalytics.value.wau || 0}\nMAU,${userAnalytics.value.mau || 0}\n总用户,${userAnalytics.value.total_users || 0}\n\n`
    csv += '设备类型\n类型,数量,占比\n'
    deviceStats.value.forEach(r => { csv += `${getDeviceTypeName(r.device_type)},${r.count},${getPercentage(r.count, totalDevices.value)}%\n` })
    csv += '\n系统分布\n系统,数量,占比\n'
    osStats.value.forEach(r => { csv += `${getDeviceTypeName(r.device_type)},${r.count},${getPercentage(r.count, totalOS.value)}%\n` })
    csv += `\n${duDimLabel.value}（当前筛选）\n名称,设备数,用户数,占比,最近活跃\n`
    duGroups.value.forEach(g => {
      csv += `"${(g.label || '').replace(/"/g, '""')}",${g.devices},${g.users},${(g.share || 0).toFixed(2)}%,${g.last_access || ''}\n`
    })
    csv += '\n流失预警用户\nID,用户名,邮箱,最后登录,订阅到期\n'
    churnUsers.value.forEach(u => { csv += `${u.id},${u.username},${u.email},${formatDate(u.last_login)},${formatDate(u.expire_time)}\n` })

    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `用户分析_${timeRangeLabel.value}_${Date.now()}.csv`
    a.click()
    URL.revokeObjectURL(url)
    ElMessage.success('已导出 CSV')
  } catch {
    ElMessage.error('导出失败')
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  loadData()
  loadDeviceUsage(1) // 汇总行/分组的数字始终要有
  loadDeviceUsers(1) // 默认展示「按用户」视图
})
</script>

<style scoped>
/* ===== 整体：紧凑，一屏尽量放完 ===== */
.an-page {
  padding: 12px 14px 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.an-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}
.an-bar__left,
.an-bar__right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.an-hint {
  font-size: 12px;
  color: #909399;
  white-space: nowrap;
}

/* 指标条：4 格紧凑卡 */
.an-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}
.an-metric {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  min-width: 0;
}
.an-metric__icon {
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: #f2f6ff;
  color: #409eff;
  font-size: 15px;
}
.an-metric__body { min-width: 0; }
.an-metric__value {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.15;
  color: #303133;
  font-variant-numeric: tabular-nums;
}
.an-metric__label {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.an-tabs { margin-top: 2px; }
.an-tabs :deep(.el-tabs__header) { margin-bottom: 10px; }
.an-tabs :deep(.el-tabs__item) { font-size: 13.5px; height: 38px; line-height: 38px; }

.an-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
.an-grid > .an-card:nth-child(3) { grid-column: 1 / -1; }
.an-card { border-radius: 8px; }
.an-card :deep(.el-card__header) { padding: 8px 12px; }
.an-card :deep(.el-card__body) { padding: 10px 12px; }
.an-card__title { font-size: 13.5px; font-weight: 600; color: #303133; }
.an-sub { font-size: 12px; color: #909399; }

.an-income { display: flex; gap: 18px; flex-wrap: wrap; }
.an-income__item { min-width: 84px; }
.an-income__num { font-size: 20px; font-weight: 700; color: #303133; font-variant-numeric: tabular-nums; }
.an-income__label { font-size: 12px; color: #909399; margin-top: 2px; }
.an-income__trend { font-size: 12px; margin-top: 4px; display: flex; align-items: center; gap: 3px; }
.an-up { color: #67c23a; }
.an-down { color: #f56c6c; }

.an-mini-two { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }

/* 筛选条 + 汇总行 */
.an-filter {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}
.an-filter--drawer { margin-bottom: 10px; }
.an-filter--view { margin-bottom: 6px; }
.an-clients { display: flex; flex-wrap: wrap; gap: 4px; }
.an-clients .el-tag { max-width: 100%; overflow: hidden; text-overflow: ellipsis; }
.an-filter__dim { width: 148px; }
.an-filter__search { width: 240px; flex: 1 1 200px; max-width: 320px; }
.an-filter__days { width: 112px; }

.an-summary {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 12.5px;
  color: #606266;
  margin-bottom: 8px;
}
.an-summary b { color: #303133; font-variant-numeric: tabular-nums; }
.an-summary__spacer { flex: 1 1 auto; }
.an-dot { color: #dcdfe6; }
.an-ok b { color: #67c23a; }
.an-warn b { color: #e6a23c; }
.an-muted { color: #909399; }
.an-muted b { color: #909399; }
.an-clickable { cursor: pointer; }
.an-clickable:hover { text-decoration: underline; }

/* 表格里的紧凑单元格 */
.an-table :deep(.el-table__cell) { padding: 6px 0; }
.an-table { cursor: pointer; }
.an-group { display: flex; align-items: center; gap: 6px; min-width: 0; }
.an-group__name {
  font-weight: 600;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.an-user { display: flex; flex-direction: column; line-height: 1.3; min-width: 0; }
.an-user__name { font-weight: 600; color: #303133; }
.an-kv { font-size: 12.5px; color: #606266; }
.an-share { display: flex; align-items: center; gap: 6px; font-size: 12px; color: #909399; }
.an-share .el-progress { flex: 1 1 auto; }
.an-danger { color: #f56c6c; }
.an-warn-text { color: #e6a23c; }

.an-pager { display: flex; justify-content: flex-end; margin-top: 10px; }
.an-drawer-tip { font-size: 12px; color: #909399; margin-top: 8px; }
.an-alert { margin-bottom: 10px; }
.an-kv-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px 14px;
  font-size: 12.5px;
  color: #606266;
  margin-bottom: 10px;
}
.an-kv-grid b { color: #303133; margin-left: 6px; }
.an-form :deep(.el-form-item) { margin-bottom: 10px; }
.an-preview {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 10px 12px;
  background: #fafbfc;
}
.an-preview__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
}
.an-preview__subject {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 6px;
}
.an-preview__body {
  max-height: 300px;
  overflow: auto;
  font-size: 13px;
  color: #303133;
  background: #fff;
  border-radius: 6px;
  padding: 8px 10px;
}
.an-footer { display: flex; justify-content: flex-end; gap: 8px; }

/* ===== 手机端 ===== */
@media (max-width: 768px) {
  .an-page { padding: 10px 10px 16px; gap: 8px; }
  .an-btn-text { display: none; }
  .an-metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .an-metric { padding: 8px 10px; }
  .an-metric__value { font-size: 16px; }
  .an-grid { grid-template-columns: minmax(0, 1fr); }
  .an-grid > .an-card:nth-child(3) { grid-column: auto; }
  .an-mini-two { grid-template-columns: minmax(0, 1fr); }
  .an-filter { gap: 6px; }
  .an-filter__dim,
  .an-filter__search,
  .an-filter__days { width: 100%; max-width: none; flex: 1 1 100%; }
  .an-filter :deep(.el-radio-group) { width: 100%; display: flex; }
  .an-filter :deep(.el-radio-button) { flex: 1 1 0; }
  .an-filter :deep(.el-radio-button__inner) { width: 100%; padding: 8px 0; }
  .an-summary { font-size: 12px; }
  .an-summary__spacer { flex: 1 1 100%; }
  .an-income { gap: 12px; }
  .an-kv-grid { grid-template-columns: minmax(0, 1fr); }
  .an-pager { justify-content: center; }
  .an-preview__body { max-height: 220px; }
}
</style>
