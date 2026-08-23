# CBoard 全量代码审查发现清单（972 条）

> 本文件为自动化审查的完整逐条发现，按文件组织。主报告见 `docs/AUDIT_REPORT.md`。

共 972 条发现：critical 9 / high 103 / medium 383 / low 443 / info 34，覆盖 253 个文件。

## 目录
1. [.env.example](#envexample)
2. [.env.redis.example](#envredisexample)
3. [.gitignore](#gitignore)
4. [.goreleaser.yaml](#goreleaseryaml)
5. [Dockerfile](#Dockerfile)
6. [Makefile](#Makefile)
7. [bt-deploy.sh](#btdeploysh)
8. [cmd/server/main.go](#cmdservermaingo)
9. [docker-compose.yml](#dockercomposeyml)
10. [frontend/.eslintrc.cjs](#frontendeslintrccjs)
11. [frontend/auto-imports.d.ts](#frontendautoimportsdts)
12. [frontend/components.d.ts](#frontendcomponentsdts)
13. [frontend/index.html](#frontendindexhtml)
14. [frontend/package.json](#frontendpackagejson)
15. [frontend/scripts/audit-click-handlers.mjs](#frontendscriptsauditclickhandlersmjs)
16. [frontend/scripts/audit-style-scope.mjs](#frontendscriptsauditstylescopemjs)
17. [frontend/src/App.vue](#frontendsrcAppvue)
18. [frontend/src/assets/mobile-optimizations.css](#frontendsrcassetsmobileoptimizationscss)
19. [frontend/src/components/AppDialog.vue](#frontendsrccomponentsAppDialogvue)
20. [frontend/src/components/AppDrawer.vue](#frontendsrccomponentsAppDrawervue)
21. [frontend/src/components/CopyableField.vue](#frontendsrccomponentsCopyableFieldvue)
22. [frontend/src/components/DataPage.vue](#frontendsrccomponentsDataPagevue)
23. [frontend/src/components/EmptyState.vue](#frontendsrccomponentsEmptyStatevue)
24. [frontend/src/components/ErrorState.vue](#frontendsrccomponentsErrorStatevue)
25. [frontend/src/components/FilterPanel.vue](#frontendsrccomponentsFilterPanelvue)
26. [frontend/src/components/FormActionBar.vue](#frontendsrccomponentsFormActionBarvue)
27. [frontend/src/components/IconButton.vue](#frontendsrccomponentsIconButtonvue)
28. [frontend/src/components/InlineEditableText.vue](#frontendsrccomponentsInlineEditableTextvue)
29. [frontend/src/components/LoadingState.vue](#frontendsrccomponentsLoadingStatevue)
30. [frontend/src/components/MobileCardList.vue](#frontendsrccomponentsMobileCardListvue)
31. [frontend/src/components/MobileLogFields.vue](#frontendsrccomponentsMobileLogFieldsvue)
32. [frontend/src/components/PaginationBar.vue](#frontendsrccomponentsPaginationBarvue)
33. [frontend/src/components/ResponsiveDataView.vue](#frontendsrccomponentsResponsiveDataViewvue)
34. [frontend/src/components/StatusTag.vue](#frontendsrccomponentsStatusTagvue)
35. [frontend/src/components/TipBlock.vue](#frontendsrccomponentsTipBlockvue)
36. [frontend/src/components/UpgradeDevicesDrawer.vue](#frontendsrccomponentsUpgradeDevicesDrawervue)
37. [frontend/src/components/layout/AdminLayout.vue](#frontendsrccomponentslayoutAdminLayoutvue)
38. [frontend/src/components/layout/UserLayout.vue](#frontendsrccomponentslayoutUserLayoutvue)
39. [frontend/src/components/tutorials/AndroidTutorials.vue](#frontendsrccomponentstutorialsAndroidTutorialsvue)
40. [frontend/src/components/tutorials/MacOSTutorials.vue](#frontendsrccomponentstutorialsMacOSTutorialsvue)
41. [frontend/src/components/tutorials/SoftwareTutorials.vue](#frontendsrccomponentstutorialsSoftwareTutorialsvue)
42. [frontend/src/components/tutorials/WindowsTutorials.vue](#frontendsrccomponentstutorialsWindowsTutorialsvue)
43. [frontend/src/components/tutorials/iOSTutorials.vue](#frontendsrccomponentstutorialsiOSTutorialsvue)
44. [frontend/src/composables/useDebounce.js](#frontendsrccomposablesuseDebouncejs)
45. [frontend/src/composables/useMobile.js](#frontendsrccomposablesuseMobilejs)
46. [frontend/src/composables/usePaymentStatusPolling.js](#frontendsrccomposablesusePaymentStatusPollingjs)
47. [frontend/src/composables/usePersistentTableColumns.js](#frontendsrccomposablesusePersistentTableColumnsjs)
48. [frontend/src/main.js](#frontendsrcmainjs)
49. [frontend/src/router/index.js](#frontendsrcrouterindexjs)
50. [frontend/src/store/auth.js](#frontendsrcstoreauthjs)
51. [frontend/src/store/settings.js](#frontendsrcstoresettingsjs)
52. [frontend/src/store/theme.js](#frontendsrcstorethemejs)
53. [frontend/src/styles/button-common.scss](#frontendsrcstylesbuttoncommonscss)
54. [frontend/src/styles/dialog-common.scss](#frontendsrcstylesdialogcommonscss)
55. [frontend/src/styles/global.scss](#frontendsrcstylesglobalscss)
56. [frontend/src/styles/list-common.scss](#frontendsrcstyleslistcommonscss)
57. [frontend/src/styles/list-unified.scss](#frontendsrcstyleslistunifiedscss)
58. [frontend/src/styles/mobile-buttons.scss](#frontendsrcstylesmobilebuttonsscss)
59. [frontend/src/styles/text-selection.css](#frontendsrcstylestextselectioncss)
60. [frontend/src/styles/user-client-polish.scss](#frontendsrcstylesuserclientpolishscss)
61. [frontend/src/utils/api.js](#frontendsrcutilsapijs)
62. [frontend/src/utils/apiCache.js](#frontendsrcutilsapiCachejs)
63. [frontend/src/utils/confirmAction.js](#frontendsrcutilsconfirmActionjs)
64. [frontend/src/utils/date.js](#frontendsrcutilsdatejs)
65. [frontend/src/utils/elementPlusServices.js](#frontendsrcutilselementPlusServicesjs)
66. [frontend/src/utils/format.js](#frontendsrcutilsformatjs)
67. [frontend/src/utils/githubDownload.js](#frontendsrcutilsgithubDownloadjs)
68. [frontend/src/utils/qrcode.js](#frontendsrcutilsqrcodejs)
69. [frontend/src/utils/safeOpen.js](#frontendsrcutilssafeOpenjs)
70. [frontend/src/utils/sanitizeHtml.js](#frontendsrcutilssanitizeHtmljs)
71. [frontend/src/utils/statusMaps.js](#frontendsrcutilsstatusMapsjs)
72. [frontend/src/utils/textSelection.js](#frontendsrcutilstextSelectionjs)
73. [frontend/src/views/Dashboard.vue](#frontendsrcviewsDashboardvue)
74. [frontend/src/views/Devices.vue](#frontendsrcviewsDevicesvue)
75. [frontend/src/views/Help.vue](#frontendsrcviewsHelpvue)
76. [frontend/src/views/Invites.vue](#frontendsrcviewsInvitesvue)
77. [frontend/src/views/Knowledge.vue](#frontendsrcviewsKnowledgevue)
78. [frontend/src/views/LoginHistory.vue](#frontendsrcviewsLoginHistoryvue)
79. [frontend/src/views/Nodes.vue](#frontendsrcviewsNodesvue)
80. [frontend/src/views/NotFound.vue](#frontendsrcviewsNotFoundvue)
81. [frontend/src/views/Orders.vue](#frontendsrcviewsOrdersvue)
82. [frontend/src/views/Packages.vue](#frontendsrcviewsPackagesvue)
83. [frontend/src/views/PaymentReturn.vue](#frontendsrcviewsPaymentReturnvue)
84. [frontend/src/views/Profile.vue](#frontendsrcviewsProfilevue)
85. [frontend/src/views/Subscription.vue](#frontendsrcviewsSubscriptionvue)
86. [frontend/src/views/Tickets.vue](#frontendsrcviewsTicketsvue)
87. [frontend/src/views/UnifiedAuth.vue](#frontendsrcviewsUnifiedAuthvue)
88. [frontend/src/views/UserSettings.vue](#frontendsrcviewsUserSettingsvue)
89. [frontend/src/views/admin/AbnormalUsers.vue](#frontendsrcviewsadminAbnormalUsersvue)
90. [frontend/src/views/admin/AbnormalUsers.vue, Tickets.vue, Knowledge.vue, Promotions.vue, Coupons.vue](#frontendsrcviewsadminAbnormalUsersvue, Ticketsvue, Knowledgevue, Promotionsvue, Couponsvue)
91. [frontend/src/views/admin/Analytics.vue](#frontendsrcviewsadminAnalyticsvue)
92. [frontend/src/views/admin/Config.vue](#frontendsrcviewsadminConfigvue)
93. [frontend/src/views/admin/ConfigUpdate.vue](#frontendsrcviewsadminConfigUpdatevue)
94. [frontend/src/views/admin/Coupons.vue](#frontendsrcviewsadminCouponsvue)
95. [frontend/src/views/admin/CustomNodes.vue](#frontendsrcviewsadminCustomNodesvue)
96. [frontend/src/views/admin/Dashboard.vue](#frontendsrcviewsadminDashboardvue)
97. [frontend/src/views/admin/Dashboard.vue, Nodes.vue](#frontendsrcviewsadminDashboardvue, Nodesvue)
98. [frontend/src/views/admin/EmailDetail.vue](#frontendsrcviewsadminEmailDetailvue)
99. [frontend/src/views/admin/EmailQueue.vue](#frontendsrcviewsadminEmailQueuevue)
100. [frontend/src/views/admin/Invites.vue](#frontendsrcviewsadminInvitesvue)
101. [frontend/src/views/admin/Knowledge.vue](#frontendsrcviewsadminKnowledgevue)
102. [frontend/src/views/admin/Nodes.vue](#frontendsrcviewsadminNodesvue)
103. [frontend/src/views/admin/Orders.vue](#frontendsrcviewsadminOrdersvue)
104. [frontend/src/views/admin/Packages.vue](#frontendsrcviewsadminPackagesvue)
105. [frontend/src/views/admin/PaymentConfig.vue](#frontendsrcviewsadminPaymentConfigvue)
106. [frontend/src/views/admin/Profile.vue](#frontendsrcviewsadminProfilevue)
107. [frontend/src/views/admin/Promotions.vue](#frontendsrcviewsadminPromotionsvue)
108. [frontend/src/views/admin/Settings.vue](#frontendsrcviewsadminSettingsvue)
109. [frontend/src/views/admin/Statistics.vue](#frontendsrcviewsadminStatisticsvue)
110. [frontend/src/views/admin/Subscriptions.vue](#frontendsrcviewsadminSubscriptionsvue)
111. [frontend/src/views/admin/SystemLogs.vue](#frontendsrcviewsadminSystemLogsvue)
112. [frontend/src/views/admin/Tickets.vue](#frontendsrcviewsadminTicketsvue)
113. [frontend/src/views/admin/UserLevels.vue](#frontendsrcviewsadminUserLevelsvue)
114. [frontend/src/views/admin/Users.vue](#frontendsrcviewsadminUsersvue)
115. [frontend/src/views/admin/components/UserDetailDialog.vue](#frontendsrcviewsadmincomponentsUserDetailDialogvue)
116. [frontend/src/views/admin/logs/*.vue（BalanceLogs/EmailLogs/CommissionLogs/SubscriptionLogs/SubscriptionResetLogs/RegistrationLogs/AuditLogs）](#frontendsrcviewsadminlogs*vue（BalanceLogsEmailLogsCommissionLogsSubscriptionLogsSubscriptionResetLogsRegistrationLogsAuditLogs）)
117. [frontend/src/views/admin/logs/AuditLogs.vue](#frontendsrcviewsadminlogsAuditLogsvue)
118. [frontend/src/views/admin/logs/BalanceLogs.vue](#frontendsrcviewsadminlogsBalanceLogsvue)
119. [frontend/src/views/admin/logs/CommissionLogs.vue](#frontendsrcviewsadminlogsCommissionLogsvue)
120. [frontend/src/views/admin/logs/EmailLogs.vue](#frontendsrcviewsadminlogsEmailLogsvue)
121. [frontend/src/views/admin/logs/RegistrationLogs.vue](#frontendsrcviewsadminlogsRegistrationLogsvue)
122. [frontend/src/views/admin/logs/SubscriptionLogs.vue](#frontendsrcviewsadminlogsSubscriptionLogsvue)
123. [frontend/src/views/admin/logs/SubscriptionResetLogs.vue](#frontendsrcviewsadminlogsSubscriptionResetLogsvue)
124. [frontend/src/views/admin/（跨文件）](#frontendsrcviewsadmin（跨文件）)
125. [frontend/tsconfig.json](#frontendtsconfigjson)
126. [frontend/vite.config.js](#frontendviteconfigjs)
127. [go.mod](#gomod)
128. [go.sum](#gosum)
129. [install-vps.sh](#installvpssh)
130. [install.sh](#installsh)
131. [internal/api/handlers/admin.go](#internalapihandlersadmingo)
132. [internal/api/handlers/analytics.go](#internalapihandlersanalyticsgo)
133. [internal/api/handlers/auth.go](#internalapihandlersauthgo)
134. [internal/api/handlers/backup.go](#internalapihandlersbackupgo)
135. [internal/api/handlers/checkin.go](#internalapihandlerscheckingo)
136. [internal/api/handlers/config.go](#internalapihandlersconfiggo)
137. [internal/api/handlers/coupon.go](#internalapihandlerscoupongo)
138. [internal/api/handlers/custom_node.go](#internalapihandlerscustom_nodego)
139. [internal/api/handlers/dashboard.go](#internalapihandlersdashboardgo)
140. [internal/api/handlers/device.go](#internalapihandlersdevicego)
141. [internal/api/handlers/download.go](#internalapihandlersdownloadgo)
142. [internal/api/handlers/email_template.go](#internalapihandlersemail_templatego)
143. [internal/api/handlers/geoip.go](#internalapihandlersgeoipgo)
144. [internal/api/handlers/invite.go](#internalapihandlersinvitego)
145. [internal/api/handlers/knowledge.go](#internalapihandlersknowledgego)
146. [internal/api/handlers/logs.go](#internalapihandlerslogsgo)
147. [internal/api/handlers/monitoring.go](#internalapihandlersmonitoringgo)
148. [internal/api/handlers/node.go](#internalapihandlersnodego)
149. [internal/api/handlers/notification.go](#internalapihandlersnotificationgo)
150. [internal/api/handlers/order.go](#internalapihandlersordergo)
151. [internal/api/handlers/package.go](#internalapihandlerspackagego)
152. [internal/api/handlers/payment.go](#internalapihandlerspaymentgo)
153. [internal/api/handlers/promotion.go](#internalapihandlerspromotiongo)
154. [internal/api/handlers/recharge.go](#internalapihandlersrechargego)
155. [internal/api/handlers/repo_sync.go](#internalapihandlersrepo_syncgo)
156. [internal/api/handlers/statistics.go](#internalapihandlersstatisticsgo)
157. [internal/api/handlers/subscription.go](#internalapihandlerssubscriptiongo)
158. [internal/api/handlers/subscription_access.go](#internalapihandlerssubscription_accessgo)
159. [internal/api/handlers/ticket.go](#internalapihandlersticketgo)
160. [internal/api/handlers/user.go](#internalapihandlersusergo)
161. [internal/api/handlers/xboard_compat.go](#internalapihandlersxboard_compatgo)
162. [internal/api/router/router.go](#internalapirouterroutergo)
163. [internal/core/auth/auth.go](#internalcoreauthauthgo)
164. [internal/core/cache/redis.go](#internalcorecacheredisgo)
165. [internal/core/cache/user_cache.go](#internalcorecacheuser_cachego)
166. [internal/core/config/config.go](#internalcoreconfigconfiggo)
167. [internal/core/database/database.go](#internalcoredatabasedatabasego)
168. [internal/middleware/auth.go](#internalmiddlewareauthgo)
169. [internal/middleware/brotli.go](#internalmiddlewarebrotligo)
170. [internal/middleware/csrf.go](#internalmiddlewarecsrfgo)
171. [internal/middleware/maintenance.go](#internalmiddlewaremaintenancego)
172. [internal/middleware/ratelimit.go](#internalmiddlewareratelimitgo)
173. [internal/middleware/security.go](#internalmiddlewaresecuritygo)
174. [internal/models/activity.go](#internalmodelsactivitygo)
175. [internal/models/audit_log.go](#internalmodelsaudit_loggo)
176. [internal/models/checkin.go](#internalmodelscheckingo)
177. [internal/models/config.go](#internalmodelsconfiggo)
178. [internal/models/coupon.go](#internalmodelscoupongo)
179. [internal/models/custom_node.go](#internalmodelscustom_nodego)
180. [internal/models/device.go](#internalmodelsdevicego)
181. [internal/models/invite.go](#internalmodelsinvitego)
182. [internal/models/knowledge.go](#internalmodelsknowledgego)
183. [internal/models/logs.go](#internalmodelslogsgo)
184. [internal/models/node.go](#internalmodelsnodego)
185. [internal/models/notification.go](#internalmodelsnotificationgo)
186. [internal/models/order.go](#internalmodelsordergo)
187. [internal/models/package.go](#internalmodelspackagego)
188. [internal/models/payment.go](#internalmodelspaymentgo)
189. [internal/models/payment_config.go](#internalmodelspayment_configgo)
190. [internal/models/promotion.go](#internalmodelspromotiongo)
191. [internal/models/promotion_participation.go](#internalmodelspromotion_participationgo)
192. [internal/models/recharge.go](#internalmodelsrechargego)
193. [internal/models/security.go](#internalmodelssecuritygo)
194. [internal/models/subscription.go](#internalmodelssubscriptiongo)
195. [internal/models/ticket.go](#internalmodelsticketgo)
196. [internal/models/token_blacklist.go](#internalmodelstoken_blacklistgo)
197. [internal/models/user.go](#internalmodelsusergo)
198. [internal/models/user_level.go](#internalmodelsuser_levelgo)
199. [internal/services/backup_service/backup_service.go](#internalservicesbackup_servicebackup_servicego)
200. [internal/services/cache_service/cache_service.go](#internalservicescache_servicecache_servicego)
201. [internal/services/cache_service/flush.go](#internalservicescache_serviceflushgo)
202. [internal/services/cache_service/warmup.go](#internalservicescache_servicewarmupgo)
203. [internal/services/config_update/cache.go](#internalservicesconfig_updatecachego)
204. [internal/services/config_update/config_update.go](#internalservicesconfig_updateconfig_updatego)
205. [internal/services/config_update/node_parser.go](#internalservicesconfig_updatenode_parsergo)
206. [internal/services/config_update/parser.go](#internalservicesconfig_updateparsergo)
207. [internal/services/config_update/region.go](#internalservicesconfig_updateregiongo)
208. [internal/services/config_update/region_maps.go](#internalservicesconfig_updateregion_mapsgo)
209. [internal/services/config_update/sse_manager.go](#internalservicesconfig_updatesse_managergo)
210. [internal/services/config_update/transport_opts.go](#internalservicesconfig_updatetransport_optsgo)
211. [internal/services/device/device_manager.go](#internalservicesdevicedevice_managergo)
212. [internal/services/discount/coupon.go](#internalservicesdiscountcoupongo)
213. [internal/services/email/email.go](#internalservicesemailemailgo)
214. [internal/services/email/template.go](#internalservicesemailtemplatego)
215. [internal/services/email/templates.go](#internalservicesemailtemplatesgo)
216. [internal/services/geoip/cache.go](#internalservicesgeoipcachego)
217. [internal/services/geoip/geoip.go](#internalservicesgeoipgeoipgo)
218. [internal/services/git/git.go](#internalservicesgitgitgo)
219. [internal/services/node_health/node_health.go](#internalservicesnode_healthnode_healthgo)
220. [internal/services/notification/notification.go](#internalservicesnotificationnotificationgo)
221. [internal/services/notification/template.go](#internalservicesnotificationtemplatego)
222. [internal/services/order/order.go](#internalservicesorderordergo)
223. [internal/services/payment/alipay.go](#internalservicespaymentalipaygo)
224. [internal/services/payment/applepay.go](#internalservicespaymentapplepaygo)
225. [internal/services/payment/codepay.go](#internalservicespaymentcodepaygo)
226. [internal/services/payment/query.go](#internalservicespaymentquerygo)
227. [internal/services/payment/wechat.go](#internalservicespaymentwechatgo)
228. [internal/services/payment/yipay.go](#internalservicespaymentyipaygo)
229. [internal/services/payment/yipay_adapter.go](#internalservicespaymentyipay_adaptergo)
230. [internal/services/promotion/promotion.go](#internalservicespromotionpromotiongo)
231. [internal/services/repo_sync/repo_sync.go](#internalservicesrepo_syncrepo_syncgo)
232. [internal/services/scheduler/scheduler.go](#internalservicesschedulerschedulergo)
233. [internal/utils/audit.go](#internalutilsauditgo)
234. [internal/utils/common.go](#internalutilscommongo)
235. [internal/utils/crypto.go](#internalutilscryptogo)
236. [internal/utils/logs.go](#internalutilslogsgo)
237. [internal/utils/network.go](#internalutilsnetworkgo)
238. [internal/utils/response.go](#internalutilsresponsego)
239. [internal/utils/safe_convert.go](#internalutilssafe_convertgo)
240. [internal/utils/validator.go](#internalutilsvalidatorgo)
241. [scripts/admin_tool.go](#scriptsadmin_toolgo)
242. [scripts/configure_payment.sh](#scriptsconfigure_paymentsh)
243. [scripts/configure_payment.sql](#scriptsconfigure_paymentsql)
244. [scripts/download_dbip.go](#scriptsdownload_dbipgo)
245. [scripts/download_geoip.go](#scriptsdownload_geoipgo)
246. [scripts/flush_cache.go](#scriptsflush_cachego)
247. [scripts/init_knowledge.sql](#scriptsinit_knowledgesql)
248. [scripts/migrate_new_features.sh](#scriptsmigrate_new_featuressh)
249. [scripts/migrations/add_promotion_participations.sql](#scriptsmigrationsadd_promotion_participationssql)
250. [scripts/unlock_user.go](#scriptsunlock_usergo)
251. [scripts/update_device_locations.go](#scriptsupdate_device_locationsgo)
252. [scripts/update_knowledge_tutorials.sql](#scriptsupdate_knowledge_tutorialssql)
253. [start.sh](#startsh)

## .env.example

### [MEDIUM] 根目录缺少 .env.example，新部署者没有配置模板
- **位置**: 1  |  **类别**: maintainability  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 任务清单要求的根目录 .env.example 实际不存在（仅 frontend/.env.example 存在）；根 .env 已被 .gitignore 排除，克隆仓库后无任何配置样例可参考，只能从 start.sh/bt-deploy.sh 里反推变量（HOST/PORT/DATABASE_URL/SECRET_KEY/SMTP_*/REDIS_* 等）。
- **建议**: 新增根 .env.example，覆盖 .env 全部键（含 Redis/SMTP/支付/上传配置）并标注必填项与示例值生成方法。

## .env.redis.example

### [MEDIUM] 示例文档推荐 docker run -p 6379:6379 无密码启动 Redis
- **位置**: 14  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 文档建议 docker run -d --name redis -p 6379:6379 redis:alpine（无 requirepass），与 install.sh 的同类命令形成呼应；照此部署的服务器若 6379 可达公网即未授权 Redis，且面板 FLUSHDB 脚本会作用于该实例。
- **建议**: 示例改为 docker run -d --name redis -p 127.0.0.1:6379:6379 redis:alpine --requirepass <随机密码>，并提示同时配置 REDIS_PASSWORD。

## .gitignore

### [LOW] 覆盖较完整但缺 .dockerignore 与证书类忽略项
- **位置**: 27-93  |  **类别**: maintainability  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: .env、*.db、*.log、uploads/、bin/、server、GeoLite2-City.mmdb、dbip-city-lite.mmdb、frontend/.env 均正确忽略，且 !frontend/src/views/admin/logs/ 反选保留日志管理组件；但未忽略 *.pem/*.key（部署后证书备份可能入库）、无 .dockerignore 配合 Dockerfile 的 COPY . .。
- **建议**: 追加 *.pem、*.key、*.crt、backups/ 忽略项；新增 .dockerignore 防止构建上下文携带敏感/大文件。

## .goreleaser.yaml

### [HIGH] goreleaser 配置是 PocketBase 官方示例的整段复制，构建目标 ./examples/base 在本仓库不存在
- **位置**: 3-16  |  **类别**: other  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: project_name: pocketbase、main: ./examples/base、binary: pocketbase、ldflags 注入 github.com/pocketbase/pocketbase.Version——本仓库既无 examples/base 目录也无该模块，运行 goreleaser release 必然失败；整份配置与 CBoard 项目无关，属于模板残留。
- **建议**: 重写为 cboard-go 配置：main: ./cmd/server/main.go、binary: server、注入 cboard-go 版本变量、goos/goarch 按实际发布目标裁剪。

### [LOW] archives 引用仓库中不存在的 LICENSE.md/CHANGELOG.md
- **位置**: 48-54  |  **类别**: other  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: archives.files 声明 LICENSE.md 与 CHANGELOG.md，但仓库根目录没有这两个文件（仅有 README.md/README_zh.md），打包时会告警或遗漏许可文件。
- **建议**: files 改为 README.md/README_zh.md，或补充 LICENSE.md 后再发布。

## Dockerfile

### [CRITICAL] CGO_ENABLED=0 编译包含 mattn/go-sqlite3 的项目必然失败
- **位置**: 14  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: go.mod 通过 gorm.io/driver/sqlite 引入 github.com/mattn/go-sqlite3（go.sum L86-87），该驱动是 cgo 实现；Dockerfile 使用 CGO_ENABLED=0 GOOS=linux go build，编译阶段即报 "cgo: C files required but cgo is disabled"，镜像构建不可用。
- **建议**: 改为 CGO_ENABLED=1 并安装 gcc/musl-dev（alpine）或改用纯 Go 的 modernc.org/sqlite 驱动；本地实测 Docker build 通过后再提交。

### [HIGH] 基础镜像 golang:1.21-alpine 与 go.mod 的 go 1.24.0 不匹配
- **位置**: 2, 14  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: go.mod 要求 go 1.24.0，镜像用 Go 1.21；构建依赖 GOTOOLCHAIN=auto 联网拉取 1.24 工具链，离线/受限网络直接失败，且 -a -installsuffix cgo 是旧式冗余参数。
- **建议**: 升级到 golang:1.24-alpine（与 go.mod 对齐），移除过时参数，使用 go mod download 缓存层优化。

### [MEDIUM] COPY . . 复制整个仓库（含 .git、53MB server 二进制、65MB GeoLite2 库）进构建上下文
- **位置**: 11  |  **类别**: performance  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 无 .dockerignore，构建上下文包含 .git、server/cboard 二进制、GeoLite2-City.mmdb、cboard.db 等，镜像体积与构建耗时显著增大，且把本地数据库快照打进镜像层（潜在数据泄露到镜像仓库）。
- **建议**: 新增 .dockerignore（.git、*.db、*.mmdb、server、cboard、frontend/node_modules、frontend/dist、*.log），运行时通过挂载提供 GeoIP 库。

### [LOW] 运行镜像无 HEALTHCHECK、非 root 用户、时区/证书已配置但缺健康探针
- **位置**: 17-31  |  **类别**: style  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: alpine 运行阶段未定义 USER，容器内以 root 运行；无 HEALTHCHECK，compose 的 restart 无法感知进程僵死（仅能感知退出）。
- **建议**: 添加 USER cboard 与 WORKDIR 权限、HEALTHCHECK CMD wget -qO- http://127.0.0.1:8000/health。

## Makefile

### [HIGH] make clean 的 rm -f *.db *.log 会删除生产数据库与全部日志
- **位置**: 17-19  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: clean 目标 rm -rf bin/ 后执行 rm -f *.db *.log，在项目根目录运行即删除 cboard.db（SQLite 数据库）与 server.log 等运维数据；对已部署站点执行 make clean 等于删库。
- **建议**: clean 仅删除 bin/ 与构建产物，数据库/日志移除改为显式目标（clean-data）并加确认。

### [MEDIUM] build 依赖 geoip 目标，文件存在时会交互式等待输入导致 CI 挂起；产物命名与部署脚本不一致
- **位置**: 27-33  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: make build → geoip → go run scripts/download_geoip.go .：数据库文件已存在且无 CI/BUILD_MODE 时卡在"是否覆盖?"；且 build 输出 bin/cboard-go，而 install.sh/bt-deploy.sh 构建 ./server，部署与本地命名两套。
- **建议**: build 去除 geoip 依赖或加 --skip-if-exists；统一输出名（server 或 bin/cboard-go）并同步到部署脚本。

## bt-deploy.sh

### [HIGH] 生成 .env 中 HOST=0.0.0.0，后端 API 直连公网 8000 端口，绕过 nginx 全部防护
- **位置**: 208-221  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: create_env_file 写 HOST=0.0.0.0，服务监听所有网卡；nginx 只代理 /api 与 /repo-sync，其余路径 Go 自身也可达。公网可直接访问 8000 明文 HTTP 接口（登录、支付、管理 API），绕过 nginx 的 TLS、代理层限流与 CDN，且配合 X-Forwarded-For 盲信可伪造来源 IP。
- **建议**: HOST 改为 127.0.0.1（nginx 反代已足够），或使用 iptables/安全组仅放行 80/443；显式注释为什么不能监听 0.0.0.0。

### [HIGH] force_restart 无差别 pkill -9 server / node，宝塔多站点主机上会杀死全部无关进程
- **位置**: 539-540  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: pkill -9 -f "${PROJECT_DIR}/server"; pkill -9 server; pkill -9 node 三条命令会命中同机其他 Go/Node 服务（宝塔面板本身、其他站点），造成生产事故；无 PID 白名单或 systemd 精确控制。
- **建议**: 删除泛化 pkill，仅 systemctl stop/start cboard；确需杀进程时按 systemd MainPID 或精确路径匹配。

### [MEDIUM] Node/Go 版本号写死（18.20.4 / 1.21.5），与 go.mod 1.24.0 及前端依赖要求脱节
- **位置**: 146, 23  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: install_nodejs_binary 固定 18.20.4；GO_VERSION 默认 1.21.5（同 install-vps.sh）。go.mod 为 go 1.24.0，前端 vite 等依赖也对 Node 有最低版本要求，写死版本导致升级维护成本与隐性不兼容。
- **建议**: 版本号收敛到单一变量并统一升级；Go 至少 1.24.x，Node 20 LTS；构建前校验 go.mod 最低版本。

### [MEDIUM] 默认 PROJECT_DIR 硬编码部署者真实域名路径 /www/wwwroot/dy.moneyfly.top
- **位置**: 21  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: PROJECT_DIR="${PROJECT_DIR:-/www/wwwroot/dy.moneyfly.top}" 把原部署者站点路径写进脚本并提交仓库：泄露运营者域名信息，且他人直接运行脚本时会往陌生路径部署。
- **建议**: 默认值改为中性占位符（如 /www/wwwroot/cboard），域名一律从参数/交互输入获取。

### [LOW] SSL 证书目录 find 模糊匹配可能选错域名
- **位置**: 337  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: cert_root=$(find /etc/letsencrypt/live -name "*${DOMAIN}*" -type d | head -n 1) 对前缀相同的域名（example.com 与 example.com.cn）可能取错证书目录。
- **建议**: 改用精确路径 /etc/letsencrypt/live/${DOMAIN}/ 并检查存在性。

### [LOW] 卸载流程引用从未创建过的 /tmp/cboard_nginx_*.conf 临时文件
- **位置**: 786-792  |  **类别**: other  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: delete_all_configs 遍历 /tmp/cboard_nginx_${DOMAIN}.conf 与 /tmp/cboard_proxy_${DOMAIN}.conf，但本脚本任何路径都不会创建这些文件，属于旧版本遗留的死逻辑。
- **建议**: 删除该段或补上对应的临时文件生成逻辑。

## cmd/server/main.go

### [HIGH] GeoIP 自动下载无超时与大小限制，可挂起启动或耗尽磁盘
- **位置**: 141-171  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: downloadGeoIPDatabase 使用裸 http.Get(url)（line 156）且 io.Copy 无大小上限（line 166）。若 GitHub 不可达或网络黑洞，TCP 默认无超时，服务器启动会无限阻塞；若响应体异常巨大，会写满磁盘。resp.StatusCode 只检查了 200，未检查 Content-Length。
- **建议**: 改用 http.Client{Timeout: 60*time.Second}，并用 io.LimitReader(resp.Body, 200<<20) 限流，超过上限即报错并删除半成品文件；顺带校验 Content-Length。

### [MEDIUM] 默认管理员初始密码明文打印到标准输出/日志
- **位置**: 217-225  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: ensureDefaultAdmin 生成的随机密码直接 log.Printf 到 stdout（line 221）。若部署环境收集 stdout 日志（Docker/Journald 常见），初始密码会永久留在日志中；日志被第三方看到即等于管理员凭据泄露。
- **建议**: 改为仅输出到 TTY（检测 os.Stderr 是否为终端），或增加 must_change_password 标志并在 AuthMiddleware 强制首次登录改密，或把密码写入仅 0600 权限的一次性文件。

### [LOW] 路径安全校验重复实现且含误报
- **位置**: 318-347  |  **类别**: duplication  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: safePathJoin 用 strings.Contains(cleaned, "..") 拒绝任何含 ".." 子串的路径（如合法文件名 "a..b" 被误拒），且与 downloadGeoIPDatabase 里 line 143-146 的 strings.Contains(cleanPath, "..") 重复。utils.IsWithinBaseDir / JoinWithinBaseDir（validator.go:61-84）已提供更严谨的实现。
- **建议**: 统一复用 utils.JoinWithinBaseDir；误报场景可改用 filepath.Rel 判定越界（保留文件存在性检查）。

### [LOW] ensureDefaultEmailTemplates 吞掉非 NotFound 错误
- **位置**: 302-315  |  **类别**: error-handling  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: db.Where(...).First(&existing) 的 err 只区分 ErrRecordNotFound（line 305-313），其他错误（连接断、权限）被静默跳过，排障困难。
- **建议**: err != nil 且非 ErrRecordNotFound 时 log.Printf 记录错误后 continue。

### [LOW] randomInt 在 crypto/rand 失败时回退到时间戳取模
- **位置**: 256-262  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: randomInt 的 fallback 为 time.Now().UnixNano() % max（line 259），预测性强；generateRandomPassword 依赖它生成初始管理员密码。虽然仅在 rand.Read 失败时触发，但密码生成属安全关键路径。
- **建议**: crypto/rand 失败时直接 log.Fatal 或重试，不要用可预测时间源兜底；至少混合 crypto/rand 与多源熵。

## docker-compose.yml

### [HIGH] SECRET_KEY 默认值 your-secret-key-here 公开且被任何未设置环境者使用
- **位置**: 10  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: environment: SECRET_KEY=${SECRET_KEY:-your-secret-key-here}——不导出 SECRET_KEY 的部署将使用公开密钥签发 JWT，攻击者可伪造任意用户/管理员身份（配合管理 API 直达 8000 端口即可完全接管）。
- **建议**: 去掉默认值改为 ${SECRET_KEY:?必须设置 SECRET_KEY} 强制注入，或首次启动自动生成并持久化到 .env。

### [MEDIUM] 8000 端口直接映射到宿主机所有网卡，无 nginx/TLS 前置
- **位置**: 6-7, 11-12  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: ports: "8000:8000" 将后端明文 HTTP 暴露公网，未部署 nginx 场景下登录/管理/支付接口无 TLS 保护；HOST=0.0.0.0 强化了该暴露面。
- **建议**: 映射改为 "127.0.0.1:8000:8000" 并前置 nginx/caddy 反代；或 compose 内加入 caddy 服务统一 TLS。

### [LOW] 仅挂载 cboard.db 单文件，WAL/SHM 文件留在容器层，强制删除容器有数据丢失窗口
- **位置**: 15  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: SQLite 在 WAL 模式下会产生 cboard.db-wal/-shm；只挂载 .db 文件时 -wal 位于可写容器层，docker compose down -v / 意外 kill 后未 checkpoint 的提交可能丢失或文件损坏。
- **建议**: 挂载整个数据目录（如 ./data:/root/data 并把 DATABASE_URL 指向其中），同时覆盖 -wal/-shm。

### [LOW] version: '3.8' 已废弃且无 MySQL/Redis 服务编排
- **位置**: 1  |  **类别**: style  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 新版 Compose 忽略 version 字段；文件注释掉的 MySQL 块暗示可选架构，但未提供 redis 服务，与 README 宣称的 Redis 缓存加速能力不一致。
- **建议**: 删除 version 字段；如宣传 Redis/MySQL 可选，补上对应服务配置或链接到独立部署说明。

## frontend/.eslintrc.cjs

### [MEDIUM] lint 覆盖 .ts/.tsx 但未配置 TS parser，@typescript-eslint 依赖未使用且 peer 版本冲突
- **位置**: 1-17  |  **类别**: architecture  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: package.json lint 脚本含 .ts/.tsx/.cts/.mts，但 .eslintrc 未设置 `parser: '@typescript-eslint/parser'`，TS 语法用 espree 解析必然报错或整文件跳过；同时 devDependencies 里 @typescript-eslint/parser & plugin ^8.20 完全未被引用（死依赖），且 v8 系列要求 eslint ^8.57.0，与当前 eslint ^8.54.0 peer 冲突，npm install 会告警。
- **建议**: 二选一：移除 TS 扩展与 @typescript-eslint 依赖；或补 parser + 相关 rules 并升级 eslint 到兼容版本。

### [MEDIUM] 核心规则设 warn + lint 脚本带 --quiet，等于静默关闭
- **位置**: 30-53  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: no-unused-vars/no-empty/no-useless-escape 等规则级别为 warn，而 `eslint --quiet` 只报 error——这些规则实际从不生效；overrides 仅对 src/utils、src/router、src/store 提升为 error，src/views、src/components（全库最大的代码面）不在强制范围，审查覆盖严重不均。
- **建议**: 去掉 --quiet，将关键规则（no-unused-vars、vue/no-use-v-if-with-v-for 等）统一为 error，或把 overrides 扩展到全 src。

## frontend/auto-imports.d.ts

### [MEDIUM] 空壳残留文件 + unplugin-auto-import 死依赖
- **位置**: 1-10  |  **类别**: maintainability  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 该文件是 unplugin-auto-import 的生成物，但 vite.config.js 只启用了 Components 插件，从未配置 AutoImport——`declare global {}` 是空壳；devDependencies 的 unplugin-auto-import ^21.0.0 无人使用。文件与依赖都属于“曾想用、后放弃”的残留。
- **建议**: 删除 auto-imports.d.ts 与 unplugin-auto-import 依赖；若未来启用 auto-import，再让插件重新生成。

## frontend/components.d.ts

### [LOW] ElIcon 图标组件（Search/Plus 等）未纳入 GlobalComponents 类型
- **位置**: 13-94  |  **类别**: architecture  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: dts 声明了 ElIcon 但没有任何 `@element-plus/icons-vue` 的图标条目；模板里 `<Search />` 等图标在 vue-tsc 严格模式下会报“找不到组件”，类型覆盖不完整。
- **建议**: 引入 unplugin-icons 或在 GlobalComponents 中为 icons-vue 生成全局组件声明。

### [LOW] 自动生成文件提交入库，已与 vite.config 的 resolver 漂移
- **位置**: 1-98  |  **类别**: maintainability  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: dts 由 unplugin-vue-components 生成并提交；vite.config.js 的 elementPlusComponentDirs map 里声明了 ElSplitterPanel（39 行），但 dts 中并无 ElSplitterPanel 条目——resolver map 与生成的 dts 已不同步，说明生成物过期或 map 冗余，组件增删后不重新 dev/build 就会残留过期类型。
- **建议**: dts 加入 .gitignore 由构建生成，或在 CI 增加“dts 是否新鲜”校验；同时删除 resolver map 中未使用条目。

## frontend/index.html

### [MEDIUM] 令牌过期清理只覆盖 admin token，用户 token 从未被检查
- **位置**: 88-110  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 清理脚本仅读取 `cboard_secure_admin_token`（localStorage）并校验 JSON 内 expiry；`cboard_secure_user_token`（sessionStorage）完全没有过期清理逻辑，且整个清理只在页面加载时执行一次——会话中期过期完全依赖后端 401。另外该脚本与 axios 拦截器的清理逻辑职责重叠，存在两份“谁清理 token”的代码。
- **建议**: 把两类 token 的过期校验统一收口到前端 token 工具函数（加载时 + 401 时双触发），index.html 里只保留首屏兜底清理。

### [MEDIUM] nosniff 用 meta 标签声明无效，且全站无 CSP
- **位置**: 16-19  |  **类别**: security  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `<meta http-equiv="X-Content-Type-Options" content="nosniff">` 不会被浏览器识别（X-Content-Type-Options 只认真实响应头，meta 形式被忽略）；同样 X-Frame-Options 缺失（点击劫持面），也没有任何 CSP。所谓 “Security Headers” 注释块实际只有 Referrer-Policy/Permissions-Policy 部分生效。
- **建议**: 在 Go 后端中间件统一下发 nosniff、X-Frame-Options: DENY、CSP 等真实响应头；前端删除无效 meta 或仅保留有效的 referrer/permissions meta。

### [LOW] 默认 Vite 图标与 theme-color 品牌不一致
- **位置**: 5, 11  |  **类别**: style  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `<link rel="icon" href="/vite.svg">` 仍是脚手架默认图标；theme-color #1677ff 与主色 --theme-primary #409EFF 不一致。
- **建议**: 替换为 CBoard 品牌图标；theme-color 与主色统一。

### [LOW] hideLoading 双重触发 + 无条件 2 秒强制隐藏，可能先于 Vue 挂载出现白屏
- **位置**: 111-129  |  **类别**: ux  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: hideLoading 既在 load 事件（124-126 行）又无条件 setTimeout 2000ms（128 行）触发；若应用 chunk 仍在加载，2 秒后 loading 消失而 #app 还是空的，用户看到白屏。且 #loading 深色背景（#0a0a0a）在浅色主题下是明显的暗色闪烁。
- **建议**: 由 Vue 挂载完成（或首个路由 ready）后再隐藏 loading，移除无条件 2s 兜底；loading 背景跟随主题色。

## frontend/package.json

### [MEDIUM] 无 engines 声明、@typescript-eslint 与 unplugin-auto-import 死依赖、eslint peer 版本冲突
- **位置**: 15-45  |  **类别**: architecture  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: Vite 7 要求 Node ≥20.19/22.12 但未声明 engines；@typescript-eslint/* ^8.20 要求 eslint ^8.57 与当前 ^8.54 冲突且配置未使用；unplugin-auto-import ^21 无人引用（见 auto-imports.d.ts 发现）；element-plus ^2.4.4（2023）与 vite ^7.3.1/plugin-vue ^6（2025）代差明显，且 2.4.4 的按需路径体系老旧。
- **建议**: 删除未使用 devDeps；补齐 engines 并锁定 Node 版本；评估升级 element-plus 到 2.9+（自定义 resolver 需回归）；统一 lint/type-check 工具链。

### [LOW] lint 脚本 --ignore-path ../.gitignore 依赖运行目录，且 type-check 无实际约束力
- **位置**: 5-14  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `eslint . --ignore-path ../.gitignore` 从非 frontend/ 目录运行时路径即失效；`type-check` 依赖的 tsconfig strict:false（见 tsconfig 发现）使脚本基本空转。
- **建议**: 改用 eslint 配置文件内 ignorePatterns；type-check 目标与 tsconfig 严格度对齐后再保留脚本。

### [LOW] 缺少 private:true，存在误发布风险
- **位置**: 2-4  |  **类别**: maintainability  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 包未声明 "private": true，若被 CI 误 publish 会污染 npm registry（name 为 cboard-modern-frontend）。
- **建议**: 添加 "private": true。

## frontend/scripts/audit-click-handlers.mjs

### [MEDIUM] 控制流关键字污染 handler 集合导致漏报，且只匹配双引号 @click
- **位置**: 62-79, 26-43  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 第 76 行正则 `\b([A-Za-z_$][\w$]*)\s*\([^)]*\)\s*{` 会把 `if (x) {`、`for (...) {`、`while`、`switch`、`catch` 全部收进 handlers 集合——若模板里真有 `@click="if"` 这类笔误反而被放过；parseTemplateClicks 的 `@click(?:\.\w+)*\s*=\s*"([^"]+)"` 只认双引号，Vue 模板常见的单引号 `@click='fn'` 与换行属性写法全部漏检，审计覆盖面不全。
- **建议**: 排除 if/for/while/switch/catch/return 等关键字（白名单过滤）；模板正则改为 `["']` 同时支持单双引号，并处理跨行属性。

### [LOW] extractSection 只取第一个 <script>，SFC 多 script 块时解析不全
- **位置**: 102-106  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `content.match(/<script[^>]*>([\s\S]*?)<\/script>/)` 无 /g 且取首个匹配：普通 script + `<script setup>` 并存的 SFC 只解析第一个，setup 里定义的处理函数可能全部漏进 handlers 集合造成假阳性误报；模板/脚本字符串里出现 `</template>`、`</script>` 字样也会提前截断。
- **建议**: 遍历所有 <script> 块合并解析；解析前用引号/注释状态机或先剥掉字符串字面量。

## frontend/scripts/audit-style-scope.mjs

### [MEDIUM] 报告行号 off-by-one：所有报错位置比实际多 1 行
- **位置**: 45-53, 95-97  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `const startLine = before.split('\n').length + 1`（96 行）：before 以不含换行符的半行（`<style scoped>` 所在行）结尾，split 会把它算成一个完整元素，startLine 已多算 1；而 auditStyle 内部 `lineNo = styleStartLine + i`（53 行）又从 i=0 对应 `<style>` 所在行开始，双重叠加使每条报错都偏后 1 行（例如第 3 行的选择器被报成第 4 行）。
- **建议**: 改用完整行计数：`const startLine = before.match(/\n/g) ? before.match(/\n/g).length + 1 : 1`，并补一条含多行 style 的用例测试。

### [LOW] 硬编码白名单与 [attr] 前缀误判
- **位置**: 32-34, 36-43  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: isAllowedGlobal 只匹配 `:global(.user-layout) .xxx-container` 这一种形态（33 行），是特例补丁；hasLocalPrefix 对 `[data-x]:deep(...)` 这类以属性选择器开头的行返回 false（视为无前缀），会误报合法写法；同时注释/字符串里的 `:deep(` 也会被当真实规则（51-66 行未过滤注释）。
- **建议**: 把允许的全局规则改成配置化列表；hasLocalPrefix 支持 `[` 开头的属性前缀；解析前先剥离 // 与 /* */ 注释。

## frontend/src/App.vue

### [LOW] 空 <style> 块属死代码
- **位置**: 11-12  |  **类别**: maintainability  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: <style> 标签内无任何样式，且无 scoped，属残留空壳。整文件仅 12 行，无其他问题。
- **建议**: 删除空 <style> 块；如需全局样式统一走 main.js 引入的 styles/global.scss。

## frontend/src/assets/mobile-optimizations.css

### [HIGH] 整个文件是死 CSS：全库无任何入口 import 它
- **位置**: 1-182  |  **类别**: maintainability  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: grep 'mobile-optimizations' 在整个 frontend 无匹配——main.js、App.vue、各布局都没有引用。且其内容与生效样式直接冲突：.el-dialog 宽度 95% !important（110 行）vs global.scss 88%、.el-drawer width:100% !important（6 行）vs global.scss 85%（1090 行）；一旦有人按文件头注释“添加到 main.css”导入，界面会按导入顺序突变。
- **建议**: 删除该文件；其中仍有价值的规则（如 @media (hover:none) 触控优化）合并进 global.scss 的移动端断点块后删除。

### [LOW] 遗留开发注释
- **位置**: 1  |  **类别**: style  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 首行注释“添加到 main.css 或 App.vue”是开发期 TODO 残留，说明该文件从未完成接线。
- **建议**: 随文件删除或替换为实际引用说明。

## frontend/src/components/AppDialog.vue

### [MEDIUM] el-dialog 生命周期事件未透传，与 AppDrawer 的 API 不一致
- **位置**: 2-19, 53  |  **类别**: maintainability  |  **来源组**: F2-components (通用组件)
- **问题**: AppDrawer 透传了 @open/@opened/@closed（AppDrawer.vue 13-15 行），而 AppDialog 对 el-dialog 的 open/opened/close/closed 事件完全不转发，defineEmits 只声明 update:modelValue 和 close-blocked。消费者无法统一以事件方式感知弹窗开合，只能 watch modelValue，两个成对组件的对外契约不一致。
- **建议**: 与 AppDrawer 对齐，在 el-dialog 上补 @open="$emit('open')" @opened="$emit('opened')" @close="$emit('close')" @closed="$emit('closed')"，并加入 defineEmits 声明。

### [LOW] 关闭守卫逻辑与 AppDrawer 完全重复
- **位置**: 63-69  |  **类别**: duplication  |  **来源组**: F2-components (通用组件)
- **问题**: handleBeforeClose（loading 时 emit('close-blocked') 并拒绝关闭，否则 done()）与 AppDrawer.vue 75-81 行逐字重复；连同 loading/closeOnClickModal/close-on-press-escape 的组合在 6 个组件里反复出现。
- **建议**: 抽取 composable useCloseGuard({ loading, emit }) 返回 handleBeforeClose，两个组件共用，消除复制粘贴。

### [LOW] 硬编码边框色 #ebeef5 与主题变量混用
- **位置**: 85-112  |  **类别**: style  |  **来源组**: F2-components (通用组件)
- **问题**: header/footer 的 border-bottom/border-top 使用字面量 #ebeef5，而项目其他组件（DataPage、FormActionBar 等）统一用 var(--theme-border, ...)，深色主题下此处不会跟随主题。
- **建议**: 统一改为 var(--theme-border, #ebeef5)。

## frontend/src/components/AppDrawer.vue

### [LOW] direction prop 缺少 validator
- **位置**: 47-50  |  **类别**: style  |  **来源组**: F2-components (通用组件)
- **问题**: direction 仅声明 type: String，未限制取值；el-drawer 只接受 rtl/ltr/ttb/btt，传错值会静默产生异常布局。
- **建议**: 加 validator: (v) => ['rtl','ltr','ttb','btt'].includes(v)。

## frontend/src/components/CopyableField.vue

### [MEDIUM] 非安全上下文下 navigator.clipboard 不可用，无降级
- **位置**: 37-47  |  **类别**: error-handling  |  **来源组**: F2-components (通用组件)
- **问题**: navigator.clipboard.writeText 仅在 HTTPS/localhost 的 secure context 存在；站点走 http 部署时 undefined.writeText 抛 TypeError，直接进入'复制失败'分支，功能完全不可用。
- **建议**: 先检测 navigator.clipboard?.writeText，不可用时降级 document.execCommand('copy')（textarea + select + execCommand），再不行才报错提示手动复制。

## frontend/src/components/DataPage.vue

### [LOW] 每个 DataPage 实例都渲染 h1，同页多实例会产生多个 h1
- **位置**: 5  |  **类别**: ux  |  **来源组**: F2-components (通用组件)
- **问题**: 页面若有多个 DataPage（如分 tab 的多个数据区块），会出现多个 <h1>，破坏文档大纲语义。
- **建议**: h1 改为可通过 prop 配置的标题级别（如 headingLevel），或默认降级为 h2 由页面根标题承担 h1。

## frontend/src/components/EmptyState.vue

### [LOW] type=error/loading 与 ErrorState/LoadingState 职责重叠
- **位置**: 29-65  |  **类别**: duplication  |  **来源组**: F2-components (通用组件)
- **问题**: EmptyState 支持 error/loading/noPermission 三种扩展类型（含各自图标与默认文案），与独立组件 ErrorState.vue、LoadingState.vue 功能重叠，两套入口并存会导致用法分裂（有的页面用 EmptyState type=error，有的用 ErrorState）。
- **建议**: 收敛为单一职责：EmptyState 只保留 empty/noPermission，错误/加载态统一用 ErrorState/LoadingState，或反之废弃独立组件。

### [LOW] 所有类型统一 aria-live="polite"
- **位置**: 2  |  **类别**: ux  |  **来源组**: F2-components (通用组件)
- **问题**: error 类型应使用 assertive 及时播报（无权限/加载失败），polite 可能被延迟读出。
- **建议**: 按 type 动态设置：error → assertive，empty/loading → polite。

## frontend/src/components/ErrorState.vue

### [MEDIUM] await emit('retry') 不生效，500ms 固定重置与父组件实际重试时长脱节
- **位置**: 66-78  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: Vue 3 的 emit() 不返回 handler 结果（返回 undefined），`await emit('retry')` 立即 resolve；retrying 在 500ms 后无条件复位。若父组件重试请求耗时 3s，按钮 500ms 后即可再次点击，'重试中...' 与实际加载完全脱节，且用户可并发触发多次重试。
- **建议**: 改为接收 onRetry 函数 prop（返回 Promise），await 该函数；或在 emits 之外用 provide/inject 或让父组件通过 prop 传入 loading 状态控制按钮。

### [LOW] router.back() 无历史栈时无反馈
- **位置**: 81-83  |  **类别**: ux  |  **来源组**: F2-components (通用组件)
- **问题**: showBack 为 true 且直接打开页面（无 history）时 router.back() 静默无效，用户以为按钮坏了。
- **建议**: 先判断 window.history.length > 1 再 back()，否则 router.push 到默认首页或隐藏返回按钮。

## frontend/src/components/FilterPanel.vue

### [LOW] 大量 :has() 选择器依赖较新浏览器
- **位置**: 117-163  |  **类别**: style  |  **来源组**: F2-components (通用组件)
- **问题**: :has() 需要 Chrome 105+ / Safari 15.4+ / Firefox 121+，且与 :deep() 组合使用时对编译器版本敏感；360px/768px 两段媒体查询里 :has 规则叠加（117-130 与 156-164 互相覆盖 grid-column: span 2 / auto），可读性差。
- **建议**: 确认目标浏览器版本；或用 JS（组件内根据 children 数量加 class）替代 :has()，降低维护成本。

### [LOW] 2 个操作按钮时 3 列网格留空格且不居中
- **位置**: 121-125  |  **类别**: ux  |  **来源组**: F2-components (通用组件)
- **问题**: :has(> :nth-child(2):last-child) 设 grid-template-columns: repeat(2, minmax(0,1fr))，但容器是 3 列模板，两个按钮占前两列、第三列空白，视觉左偏。
- **建议**: 2 个子元素时改为 repeat(2, minmax(0,1fr)) 并配合 justify-content: center，或直接给 actions 容器加 inline-grid + margin auto 处理。

## frontend/src/components/FormActionBar.vue

### [LOW] el-button 重置样式在 5+ 个组件重复声明
- **位置**: 97-102  |  **类别**: duplication  |  **来源组**: F2-components (通用组件)
- **问题**: margin-left:0 / min-height:36px / touch-action:manipulation / white-space:normal 这套 reset 在 DataPage.vue(90-95)、FilterPanel.vue(82-88)、FormActionBar、ErrorState、EmptyState 里逐文件复制；任一处改（如统一 44px 触控高度）都容易漏改。
- **建议**: 在全局样式（如 styles 目录）定义 .app-action-button 工具类或全局 :deep(.el-button) 基线，组件只引用类名。

## frontend/src/components/IconButton.vue

### [MEDIUM] 纯图标按钮缺无障碍名称时不提供警告/兜底
- **位置**: 1-39  |  **类别**: ux  |  **来源组**: F2-components (通用组件)
- **问题**: aria-label 取 ariaLabel || tooltip，两者都为空（circle 图标按钮常见）时按钮无 accessible name，屏幕阅读器只能读'按钮'；且 handleClick 中 disabled/loading 判断与 el-button 自带禁用冗余。
- **建议**: icon 存在且无默认 slot 文本时强制要求 ariaLabel（模板里计算 fallback = ariaLabel || tooltip || '操作'），避免空名称。

### [LOW] tooltip 分支与无 tooltip 分支整段复制 el-button
- **位置**: 2-39  |  **类别**: duplication  |  **来源组**: F2-components (通用组件)
- **问题**: 两个 v-if 分支各含一份完整的 el-button + el-icon + slot 结构（9-22 与 24-38 行），仅外层多了 el-tooltip，改一处属性要同步两处。
- **建议**: 始终渲染 el-button，用 <el-tooltip v-if="tooltip"><template #default>...</template></el-tooltip> 包裹同一份按钮模板，或抽内部 ButtonInner 子组件。

## frontend/src/components/InlineEditableText.vue

### [MEDIUM] loading 期间 blur 导致编辑态卡死无法退出
- **位置**: 78-82, 13  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: @blur="save" 时若 props.loading 为 true，save() 直接 return（不置 editing=false），输入框已失焦却仍渲染在编辑态；用户既不能保存也不能取消（Esc 的 cancel 不检查 loading，但输入框已失焦、焦点不再触发），界面卡在'假编辑'状态。
- **建议**: loading 为 true 时也退出编辑态（editing=false 但暂不 emit，等父组件 loading 结束后由 watch 同步 draft），或 blur 时若 loading 则先 cancel 视觉态。

### [LOW] 编辑中外部值更新被静默丢弃
- **位置**: 66-68  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: watch 仅在 !editing 时同步 draft；若用户在编辑期间父组件异步刷新了 value，保存会用旧 draft 覆盖新值，无任何提示。
- **建议**: 保存前用最新 props.value 做 diff 提示（值已变化需二次确认），或编辑开始时记录基线版本号，保存时校验。

## frontend/src/components/LoadingState.vue

### [LOW] fullscreen 用 position:fixed 但无 portal，受 transform 祖先影响
- **位置**: 52-61  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: 若组件挂载在 transform/filter/perspective 祖先内（本组件常被用于卡片/弹层内部），position:fixed 会相对该祖先定位，遮罩无法覆盖全屏且可能错位。
- **建议**: fullscreen 时用 Teleport to body，或改用 Element Plus 的 v-loading/fullscreen 指令。

## frontend/src/components/MobileCardList.vue

### [MEDIUM] 本地 formatMoney/formatDate 与 utils 重复实现
- **位置**: 135-147  |  **类别**: duplication  |  **来源组**: F2-components (通用组件)
- **问题**: formatMoney（142-147）与 utils/format.js 的 formatMoney（1-7 行）逻辑完全一致；formatDate（135-140）与 utils/date.js 的 formatDateTime 语义重复但实现不同（此处直接 dayjs(date) 无上海时区处理），同一数据两处格式化结果可能不一致（UTC vs Asia/Shanghai 差 8 小时）。
- **建议**: 删除本地实现，统一 import { formatMoney } from '@/utils/format' 与 { formatDateTime } from '@/utils/date'，保证全站时区/金额口径一致。

### [MEDIUM] v-else 与 v-for 混用在同一元素上
- **位置**: 9-15  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: <div v-else v-for="(item,index) in normalizedData" :key="item[idField] || index">：Vue 3 中 v-if 优先级高于 v-for，此写法虽能工作但语义隐晦，eslint-plugin-vue 的 no-use-v-if-with-v-for 会告警，且后续有人改成 v-else-if 极易踩坑；key 用 `item[idField] || index`，id 为 0 时静默退化为 index。
- **建议**: 改成 <template v-else><div v-for=... :key="item[idField] ?? index"></div></template>，key 用 `item[idField] ?? index` 保留 0 值。

### [LOW] formatValue 对对象/数组值直接插值渲染
- **位置**: 149-153  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: field.formatter 缺省时 `return value`，若字段值是对象（如节点列表、配置 JSON）会渲染成 '[object Object]'，且 getFieldTitle 里再 String() 一次。
- **建议**: 非 string/number 类型时转 JSON 字符串或提示 '-'，并统一走 toDisplayText。

### [LOW] :key="field.key" 无唯一性保障
- **位置**: 26  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: fields 由父组件传入，若两个 field 配置了相同 key（或 key 缺失为 undefined），v-for key 冲突导致渲染错乱且仅控制台告警。
- **建议**: 渲染前对 fields 做 key 归一（field.key ?? field.label ?? index）或文档约束 key 唯一。

## frontend/src/components/MobileLogFields.vue

### [LOW] 通过 :deep() 约定外部类名的隐式契约
- **位置**: 16-71  |  **类别**: architecture  |  **来源组**: F2-components (通用组件)
- **问题**: 组件用 :deep(.mobile-log-title/.mobile-log-field/.mobile-log-wrap) 等命名类给消费者内容打样式，但消费者（其他页面）必须恰好知道这些类名，否则样式失效，属于隐藏依赖，IDE/重构都追不到引用。
- **建议**: 改为命名插槽（title/subtitle/field）由组件自己渲染类名，或把样式类导出为全局工具类并写文档。

### [LOW] 空的 <script setup> 块
- **位置**: 7-8  |  **类别**: other  |  **来源组**: F2-components (通用组件)
- **问题**: 组件无任何逻辑，<script setup></script> 为空，属于死代码。
- **建议**: 删除空的 script 块。

## frontend/src/components/PaginationBar.vue

### [MEDIUM] size-change 时 change 事件携带过期页码
- **位置**: 71-74  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: handleSizeChange 里 emit('change', { page: props.currentPage, ... }) 同步读取的是父组件旧值；el-pagination 切换 pageSize 后内部会重算并钳制 current-page，父组件若仅监听 @change 刷新数据，会用旧页码（如第 5 页）配新 pageSize 请求，可能越界返回空数据。
- **建议**: 切换 size 时按 el-pagination 语义发出 page: 1（或在 nextTick 后读取最新 props.currentPage），并注明 change 事件与 update:currentPage/current-change 的职责划分。

### [LOW] 单次翻页触发三个事件，父组件易重复请求
- **位置**: 18-21, 76-79  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: 一次页码点击同时触发 update:currentPage、current-change、change 三个事件；父组件若同时绑定 v-model:current-page 与 @current-change/@change 去拉数据，会重复发起请求（无去重）。
- **建议**: 收敛事件面：保留 update:* 与 change（或 current-change），在文档/父组件约定只监听其一；或 change 改为内部合并（v-model 同步后统一派发一次）。

## frontend/src/components/ResponsiveDataView.vue

### [LOW] 桌面/移动两套视图始终同时渲染
- **位置**: 6-25  |  **类别**: performance  |  **来源组**: F2-components (通用组件)
- **问题**: table 插槽与 MobileCardList 都无条件渲染，仅靠 CSS display:none 切换；大列表（几百行）时桌面端仍要创建全部卡片 DOM、移动端仍要渲染整张表格，双份开销。
- **建议**: 用 matchMedia('(max-width: 768px)') + 响应式 ref（复用 useMobile）配合 v-if 只渲染当前视口一侧，或对卡片侧使用虚拟列表。

## frontend/src/components/StatusTag.vue

### [MEDIUM] value 为 0 / false 时无法映射，被短路成 '-'
- **位置**: 40  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: getStatusConfig（statusMaps.js 136-139 行）开头 `if (!status) return { text: '-', type: 'info' }`，而组件 value 类型允许 Boolean/Number，因此 value=false 或 0 这类合法状态永远显示 '-'，即使 map 中定义了 false/0 的键也不会命中。
- **建议**: 判断改为 `status === undefined || status === null || status === ''`，让 0/false 可走 map 查找。

## frontend/src/components/TipBlock.vue

### [LOW] closable 时关闭事件未透传
- **位置**: 2-11  |  **类别**: maintainability  |  **来源组**: F2-components (通用组件)
- **问题**: closable 默认 false 但暴露了开关；置 true 后 el-alert 关闭按钮可点，但组件未转发 close 事件，父组件无法感知/联动（如记录'已读提示'）。
- **建议**: 加 emit('close') 并在 el-alert 上 @close="$emit('close')"，同时默认 closable 语义文档化。

## frontend/src/components/UpgradeDevicesDrawer.vue

### [HIGH] 支付方式选择存在竞态：嵌套 setTimeout 会覆盖用户选择并可提交空 payment_method
- **位置**: 445-464, 570-575  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: handleUpgradeDialogOpen 先置 paymentMethod=''，再嵌套 setTimeout(500)+setTimeout(300) 决定最终支付方式：用户在打开后 800ms 内手动选择了'余额支付'或某在线方式，会被定时器按 availableUpgradePaymentMethods[0] 覆盖；期间（前 800ms）'确认升级'按钮可用，confirmUpgrade 会用空字符串 payment_method 调 orderAPI.upgradeDevices({ payment_method: '' })，要么后端报错要么按错误方式创建订单；快速开关抽屉还会残留上一轮的定时器链，用过期数据覆盖新一轮选择。
- **建议**: 去掉嵌套 setTimeout：Promise.all 完成后一次性计算；支付方式默认取余额可支付性决定，用户手动选择后不再被任何异步结果覆盖（加 userTouched 标记）；确认按钮在 paymentMethod 为空或计算中时禁用。

### [MEDIUM] 手写 isMobile + resize 监听，重复实现 useMobile composable
- **位置**: 342, 404-410, 674-689  |  **类别**: duplication  |  **来源组**: F2-components (通用组件)
- **问题**: 组件自建 isMobile ref + handleResize（rAF 防抖）+ onMounted/onUnmounted 监听（674-689），与项目已有 composables/useMobile.js（被 AppDialog/AppDrawer/PaginationBar 等使用）完全同构，属重复实现；两处断点常量也各自维护。
- **建议**: 删除本地实现，直接 const isMobile = useMobile()，统一断点与生命周期管理。

### [MEDIUM] calculateUpgradeCost 无请求序列化，快速操作产生乱序覆盖
- **位置**: 466-488, 666-672  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: changeDeviceCount/selectAdditionalDays 每次变更都无 await 地并发调用 calculateUpgradeCost（preview_only 请求）；慢响应后到会覆盖快响应，最终展示的价格/折扣可能对应旧输入（如先点 +10 再点 -1，+10 的响应最后返回，界面显示 10 个设备的价格但 stepper 显示 1）。
- **建议**: 为预览请求加序号/AbortController（仅采纳最新一次结果），或对 stepper 输入做 300ms debounce 后再请求。

### [MEDIUM] 样式尾部整段 !important 覆盖，且含不属于本组件的死 CSS
- **位置**: 1592-1644  |  **类别**: maintainability  |  **来源组**: F2-components (通用组件)
- **问题**: 1592 行起用 .upgrade-hero,...{ border:...!important; background:#fff!important } 批量覆盖前面已定义的同一批类（722/759/1336 等已设 #111827/#2563eb 色系），说明样式从旧版本复制后冲突未清理，靠 !important 硬压；其中 .footer-price、.month-card-price（1632-1633）在本组件模板不存在，.recharge-qr-section .qr-code-wrapper（1640）属于其他组件，均为死规则。
- **建议**: 删除 1592-1644 覆盖段，把颜色统一到 CSS 变量（var(--el-color-primary) 等），删除 .footer-price/.month-card-price/.recharge-qr-section 死选择器。

### [MEDIUM] iframe 内嵌支付页 URL 无协议白名单与 sandbox
- **位置**: 251-258, 348-354  |  **类别**: security  |  **来源组**: F2-components (通用组件)
- **问题**: paymentUrl 直接来自订单响应（payment_url/payment_qr_code），isPaymentPageUrl 仅靠字符串启发式（含 payapi/pay/payment、submit.php，或任意 http(s) 且不含 'qrcode'）判定后嵌入 <iframe :src="paymentUrl">：http 明文支付页会被嵌入（混合内容被浏览器拦截反而白屏报错）；iframe 无 sandbox 属性，支付页脚本与本应用同上下文权限；后端若返回被篡改/第三方 URL 会直接内嵌展示（钓鱼/点击劫持面）。
- **建议**: 后端返回支付 URL 时按网关域名白名单校验并强制 https；iframe 加 sandbox="allow-scripts allow-forms allow-popups allow-same-origin"（按支付页实际需求裁剪），并在展示前用 normalizeSafeUrl/isSafeWebUrl 复核。

### [LOW] 支付成功后的表单重置值不一致
- **位置**: 541, 446, 578-585  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: 轮询路径成功时 upgradeForm 重置为 { additionalDevices: 5, additionalDays: 0 }（541 行），而每次打开抽屉重置为 { additionalDevices: 1 }（446 行），confirmUpgrade 的 paid 分支（578-585）则完全不重置表单——三个路径行为不一致，用户重开抽屉会看到不同的默认升级数量。
- **建议**: 统一默认值常量（如 DEFAULT_UPGRADE_FORM = { additionalDevices: 1, additionalDays: 0 }），三个路径共用 reset 函数。

### [LOW] 到期日预览用 toISOString()(UTC) 再本地格式化，可能差一天
- **位置**: 369-376  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: newExpireDate 计算 current.getTime() + days*86400000 后 toISOString()（UTC 字符串），formatDate 再按本地时区格式化；项目其余日期统一走 utils/date.js 的 Asia/Shanghai 时区工具（createShanghaiDayjs），此处在边界（跨时区、夏令时、纯日期存储）会出现差一天的预览。
- **建议**: 改用 utils/date.js 的 dayjs 时区封装计算新到期日，保持全站日期口径一致。

### [LOW] payment_qr_code 为图片 URL 时会被再次编码成二维码
- **位置**: 494-524  |  **类别**: logic  |  **来源组**: F2-components (通用组件)
- **问题**: showPaymentQRCode 对非支付页 URL 一律 createQRCodeDataURL(url)：若网关返回的是二维码图片地址（如 payjs/qpay 常见返回 image URL 而非支付链接），用户扫出的二维码内容是图片 URL，支付流程断裂；isPaymentPageUrl 的启发式只能覆盖含 'qrcode'/'qr.alipay' 的 URL。
- **建议**: 后端在订单响应里区分 payment_url（链接）与 payment_qr_code（图片），前端对图片 URL 直接 <img> 展示，对链接才生成二维码。

### [LOW] monthOptions 未使用、getMyLevel 空 catch 无用途、onImageLoad 空实现
- **位置**: 330, 437-439, 649  |  **类别**: other  |  **来源组**: F2-components (通用组件)
- **问题**: monthOptions ref（330 行）定义后从未被引用（实际用 monthCardOptions computed）；fetchUserInfo 内 userLevelAPI.getMyLevel() 结果丢弃且 catch (e) {} 静默吞错；onImageLoad（649 行）为空函数，仅占位。
- **建议**: 删除 monthOptions 与 onImageLoad；getMyLevel 如需预热缓存应调用 cachedAPI.getMyLevel() 并在失败时 console.warn，而非裸调用吞错。

### [LOW] el-radio 的 label 用法在新版 Element Plus 已弃用
- **位置**: 176-191  |  **类别**: style  |  **来源组**: F2-components (通用组件)
- **问题**: payment-radio-card 用 label="balance"/:label="method.key" 作为值（EP 2.6+ 推荐 value prop，label 语义将回归纯文案），后续升级 EP 时选中逻辑可能失效。
- **建议**: 改用 :value="method.key"（EP ≥2.6），或锁定组件版本并在升级时回归测试。

### [LOW] 月份卡片是单选控件但无 aria-pressed/role 语义
- **位置**: 112-121  |  **类别**: ux  |  **来源组**: F2-components (通用组件)
- **问题**: month-card 为 button 实现的单选卡片（选中的 30/90/180 天），仅靠 .is-active 视觉区分，无 aria-pressed 或 role="radio"/aria-checked，屏幕阅读器读不出选中态；stepper 按钮有 aria-label 而月份卡片无。
- **建议**: 给 month-card 加 :aria-pressed="upgradeForm.additionalDays === opt.days"，或改用 el-radio-group 卡片化样式。

### [INFO] 费用明细由服务端 preview 计算，前端仅展示（合理）
- **位置**: 142-161  |  **类别**: other  |  **来源组**: F2-components (通用组件)
- **问题**: 金额/折扣来自 orderAPI.upgradeDevices(preview_only:true) 的响应并解析为浮点展示，未在前端做任何金额计算，余额校验也只在 UI 层（按钮 disabled），服务端结算仍以真实订单为准——设计合理，无发现。
- **建议**: 无需修改；建议确认后端对 preview_only 请求同样做余额/权限校验，避免前端只读绕过。

## frontend/src/components/layout/AdminLayout.vue

### [HIGH] isRouteActive 前缀匹配导致导航双重高亮
- **位置**: 359  |  **类别**: logic  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `const isRouteActive = (path) => route.path === path || (path !== '/admin/dashboard' && route.path.startsWith(path))`。菜单里 /admin/config（配置管理，line 325）是 /admin/config-update（节点更新，line 318）的字符串前缀，因此当路由为 /admin/config-update 时，两个菜单项同时获得 .active 高亮；未来任何兄弟前缀路由（如 /admin/orders 与 /admin/orders-detail）都会复现。
- **建议**: 改为分段前缀匹配：`route.path === path || route.path.startsWith(path + '/')`，并对 /admin/dashboard 保留特例。

### [MEDIUM] 与 UserLayout 约 70% 重复：头部/侧边栏/移动导航/主题/未读逻辑/样式
- **位置**: 1-434  |  **类别**: architecture  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: AdminLayout 与 UserLayout 的 header、sidebar、mobile-nav-bar、mobile-tabbar、mobile-overlay、slide-down 过渡、theme dropdown、未读轮询与去重逻辑、以及几乎逐字相同的 SCSS（如 .header/.sidebar/.main-content/.mobile-tabbar/.mobile-overlay）全部复制。两处改一处忘是常态（本组已出现行为分叉，见 UserLayout sidebarCollapsed 持久化）。
- **建议**: 抽取共享组件：AppHeader / AppSidebar / AppMobileNav / AppMobileTabbar，与 composable `useUnreadTickets()`（含轮询+去重+事件监听）；公共样式并入 styles 下的 layout partial。

### [MEDIUM] 对 route.path 注册了两个重复 watcher
- **位置**: 408-425  |  **类别**: duplication  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: setup 作用域注册 `watch(() => route.path, ...)`（408-413 关闭移动端导航），onMounted 内又注册了第二个 `watch(() => route.path, ...)`（421-425，进入 /admin/tickets 时刷新未读数）。同一 source 两个 watcher 逻辑割裂，第二个的职责完全可并入第一个。
- **建议**: 合并为一个 watcher：路径变化时先关移动端菜单，再判断 `newPath === '/admin/tickets'` 调 loadUnreadTicketCount()。

### [MEDIUM] handleThemeChange 无 try/catch，setTheme 异常导致未处理 Promise 拒绝
- **位置**: 368-371  |  **类别**: error-handling  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `const result = await themeStore.setTheme(themeName); result.success ? ...` 直接解引用 result.success。若 setTheme 网络失败 reject，这里抛出未处理异常，用户无任何反馈，主题可能部分生效。
- **建议**: 包一层 try/catch：`try { const result = await themeStore.setTheme(...) } catch { ElMessage.error('主题保存失败') }`，并把 result?.success 判空。

### [MEDIUM] flushCache 响应解引用缺少可选链
- **位置**: 389-392  |  **类别**: error-handling  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `const res = await api.post('/admin/settings/cache/flush'); if (res.data.success)` 直接访问 res.data.success；若响应体为空或后端返回非 JSON（网络层 204/异常），这里抛 TypeError 且错误分支拿不到真正原因。
- **建议**: 改为 `res.data?.success` 与 `res.data?.message`，并在成功/失败判空后兜底提示。

### [LOW] flushCache 确认弹窗期间按钮无 loading，且依赖 'cancel' 字符串哨兵
- **位置**: 381-396  |  **类别**: logic  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `cacheClearing.value = true` 在 confirmClear 之后才置位，弹窗打开期间按钮不显示加载态；`catch (e) { if (e !== 'cancel') ... }` 与 confirmAction 抛 'cancel' 字符串的约定强耦合，一旦 confirmAction 改抛 Error 对象或 Promise 变为 resolve(false)，误报『清除失败』。
- **建议**: 将 cacheClearing 置位提前到点击时；对取消判断改为 confirmAction 返回布尔（resolve(false) 表示取消），避免字符串哨兵。

### [LOW] 未读工单 30 秒轮询无条件常驻
- **位置**: 414-426  |  **类别**: performance  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: onMounted 即启动 `setInterval(..., 30000)` 轮询 /tickets/unread-count，即使标签页隐藏或管理员从不使用工单模块也在请求；与 window 'ticket-viewed' 事件 + 路由 watcher 三重触发（虽有 in-flight 去重，但无节流）。
- **建议**: 用 `document.visibilitychange` 暂停/恢复轮询，或仅在 /admin/tickets 相关页面驻留轮询；至少将间隔提升到 60s。

## frontend/src/components/layout/UserLayout.vue

### [MEDIUM] handleThemeChange 同样无 try/catch 直接解引用 res.success
- **位置**: 337-340  |  **类别**: error-handling  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `const res = await themeStore.setTheme(name); res.success ? ...` 与 AdminLayout 同款问题：setTheme reject 时未处理异常；且错误提示文案『本地生效』与实现语义（失败时本地主题其实也可能未应用）不符。
- **建议**: 与 AdminLayout 统一：try/catch + `res?.success`，失败提示『主题保存失败，已回退默认』。

### [MEDIUM] userSidebarCollapsed 只写不读，桌面端折叠偏好每次刷新即丢失
- **位置**: 282, 346  |  **类别**: logic  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: toggleSidebar 写入 `localStorage.setItem('userSidebarCollapsed', ...)`（282），checkMobile 桌面分支又强制写 'false' 并置 sidebarCollapsed=false（346），全项目 grep 无任何读取该 key 的地方。与 AdminLayout（checkMobile 桌面分支读 'sidebarCollapsed' 恢复）行为不一致：用户折叠侧栏后刷新页面必然展开。
- **建议**: 与 AdminLayout 对齐：checkMobile 桌面分支读取 localStorage 恢复，删除强行写 'false'；若无意支持持久化则删掉 282/346 两处写入。

### [MEDIUM] returnToAdmin 先注销用户会话，管理员令牌失效时两头落空
- **位置**: 323-336  |  **类别**: logic  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `authStore.logout('user')` 先销毁当前用户会话，再 `setAuth(token, user, false)` 切管理员。若 admin_token 已过期/被后端撤销，路由守卫会把用户踢到 /admin/login，此时用户会话也已注销，用户既回不去管理后台也丢了原会话，只能重新登录。
- **建议**: 切换前先校验 admin_token（如调 /admin/profile 或解析过期时间），失败则提示『管理员登录已过期，请重新登录』且保留用户会话；切换失败时回滚恢复用户会话。

### [LOW] hasAdminAccess 向普通用户暴露『存在管理员会话』且 admin_user 明文存于 secureStorage
- **位置**: 205, 265-272  |  **类别**: security  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `hasAdminAccess = !!(secureStorage.get('admin_token') && secureStorage.get('admin_user'))` 直接把同浏览器里管理员会话的存在性渲染成『返回管理后台』菜单项；admin_user 含账号信息常驻存储，与用户会话同源共存，是信息暴露面与凭据混杂的隐患（虽依赖后端鉴权兜底，非越权）。
- **建议**: 后端提供 /auth/status 判定当前可切换角色并校验令牌有效性，前端据此渲染；存储上对 admin_user 加密或仅存非敏感字段。

### [LOW] 『返回管理后台』是裸 div + click，键盘不可达
- **位置**: 64-69, 109-114  |  **类别**: ux  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `<div class="nav-item admin-back" @click="returnToAdmin()">` 既无 role/tabindex 也无 keydown 处理，而旁边所有 router-link 天然可聚焦；桌面与移动两处均如此，键盘/读屏用户无法触发该操作。
- **建议**: 改用 router-link（v-if 分支复用）或补 `role="button" tabindex="0" @keydown.enter.prevent="returnToAdmin()"`。

## frontend/src/components/tutorials/AndroidTutorials.vue

### [HIGH] 与其余教程组件同样的全量同构复制
- **位置**: 4-153  |  **类别**: duplication  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: Clash Meta / V2rayNG / Hiddify 三面板重复『下载→安装→导入订阅(方法一/二)→使用→tips』结构，样式块与 MacOS/iOS/Windows 组件逐字重复；含 AndroidTutorials 在内 5 个文件合计约 1400 行、其中可数据化约 1000 行。
- **建议**: 统一收敛为 TutorialPanel + 各 OS 数据文件（同 MacOSTutorials 建议）。

### [MEDIUM] Android 标签页无任何下载按钮，教程却指引点击『立即下载』
- **位置**: 7, 57, 107  |  **类别**: ux  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: SoftwareTutorials 的 Android tab 仅渲染 `<AndroidTutorials />`（60 行），三个客户端开头均写『点击上方"立即下载"按钮下载 Clash Meta / V2rayNG / Hiddify 最新版本』，页面上不存在该按钮；且 githubDownload.js clientMap 中无 'clash-meta' 配置（仅 v2rayNG/FlClash/hiddify-app 覆盖 Android），而 Dashboard.vue line 793 存在 clientId 'clash-meta'，同一客户端在不同页面能力不一致。
- **建议**: 在 Android 标签页补下载行（v2rayNG/hiddify 可直接用现成 githubKey；Clash Meta 需在 githubDownload.js 补充配置或移除『立即下载』文案），统一与 Dashboard 客户端清单。

### [INFO] 组件本身无运行时逻辑，除重复与文案问题外无 bug
- **位置**: 161-207  |  **类别**: other  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: 仅 ref + el-collapse 静态内容，无 API 调用、无生命周期副作用；模板插值均走 Vue 转义，无 XSS 面。安全性整体良好（safeOpen 协议白名单、confirmAction 关闭 HTML 字符串、无 v-html）。
- **建议**: 并入统一 TutorialPanel 重构即可，无需单独修复。

## frontend/src/components/tutorials/MacOSTutorials.vue

### [HIGH] 三个客户端面板结构 95% 相同，纯复制粘贴静态模板
- **位置**: 4-168  |  **类别**: duplication  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: FlClash / Clash Part / Clash Verge 三面板（各 ~55 行）的『1. 软件下载 / 2. 安装步骤 / 3. 导入订阅(方法一/二) / 4. 使用方法 / 使用技巧』层级、列表结构与提示文案完全同构，仅软件名与个别步骤措辞不同；与 WindowsTutorials(6 面板)、AndroidTutorials(3 面板)、iOSTutorials 合计约 1400 行同类重复。
- **建议**: 改为数据驱动：定义 `clients: [{ name, collapseKey, steps: { download, install[], importMethods[], usage[], tips[] } }]`，用一个 `TutorialPanel.vue` 渲染，五个 OS 组件瘦身为纯数据文件，预计减重 70%+。

### [LOW] 样式块与四个兄弟教程组件逐字重复
- **位置**: 177-223  |  **类别**: duplication  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: .tutorial-content/.subscription-methods/.tips 等约 45 行 SCSS 在 MacOS/iOS/Windows/Android 四个组件中完全一致（仅 h3 左边框色值不同：#f39c12/#9b59b6/#3498db/#e74c3c），任何样式调整需五处同步。
- **建议**: 抽公共样式文件（如 styles/tutorial-common.scss 或 TutorialPanel 组件内 scoped 样式），各组件仅覆盖主题色变量。

### [LOW] 产品名『Clash Part』与代码库命名『Clash Party』不一致
- **位置**: 59-113  |  **类别**: style  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: 本文件及 WindowsTutorials 均写『Clash Part』，而 githubDownload.js 中配置为 `'clash-party'`、repo `mihomo-party-org/clash-party`，官方产品名为 Clash Party（原 Clash for Windows 继任者）；文案拼写错误会误导用户搜索。
- **建议**: 统一为『Clash Party』，并校对 Dashboard 相关 clientId 文案。

## frontend/src/components/tutorials/SoftwareTutorials.vue

### [HIGH] 教程文案引用不存在的『立即下载』按钮：macOS/Android 标签页无任何下载按钮
- **位置**: 12-61  |  **类别**: ux  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: Windows 标签页有 Clash Verge 下载行（19-30），iOS 有 Shadowrocket『打开商店』行（44-54），但 macOS（34-36 仅 `<MacOSTutorials />`）与 Android（59-61 仅 `<AndroidTutorials />`）标签页没有任何下载按钮；而 MacOSTutorials/AndroidTutorials 每个客户端开头都写『点击上方"立即下载"按钮下载 xxx 最新版本』（如 MacOSTutorials line 7/62/117，AndroidTutorials line 7/57/107），用户按图索骥找不到按钮。Windows 标签页也只有 Clash Verge 一个客户端有下载按钮，其余 5 个客户端同样指向不存在的按钮。
- **建议**: 把下载按钮改为按客户端配置数据驱动（每个 tab 渲染对应 client-grid），或把教程文案中的『点击上方"立即下载"按钮』改为『在用户仪表盘/本页下载入口获取安装包』，避免死引用。

### [MEDIUM] copySubscription 对空订阅链接无守卫，复制 'undefined' 且无反馈
- **位置**: 145-150  |  **类别**: error-handling  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `const url = type === 'clash' ? subscription.value?.clash_url || subscription.value?.universal_url : ...; await copyText(url, '订阅链接已复制')`。loadRuntimeData 用 allSettled，订阅接口失败时 subscription 为空对象，url 为 undefined；copyText(undefined) 很可能写入 'undefined' 文本或抛错，async 处理器无 try/catch → 未处理拒绝，用户以为复制成功。
- **建议**: `if (!url) { ElMessage.error('订阅链接暂不可用，请稍后重试'); return }`，并对 copyText 包 try/catch。

### [LOW] getResponseData 对 falsy 的 data 字段回退到整个响应对象
- **位置**: 84-88  |  **类别**: logic  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `return response.data.data || response.data || {}`：若后端 data 为 0/''/false（如计数类字段），会误把整个响应体当 data 返回，下游字段读取错位。
- **建议**: 改为 `response.data.data !== undefined ? response.data.data : (response.data || {})`。

### [LOW] 下载按钮配置键内嵌平台，但 GitHub 回退按 detectSystem 检测，行为与按钮语境可能不符
- **位置**: 103-143  |  **类别**: logic  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: Windows 标签页 Clash Verge 按钮调用 `downloadClient('clash_verge_windows_url', 'clash-verge')`：配置键含 _windows_ 平台后缀，但未配置时回退 `getClientDownloadUrl` 走 `detectSystem()`，Mac 用户点『Windows 标签下的下载』拿到的是 macOS 包；且 macOS/Android 若配独立键（如 clash_verge_macos_url）当前页面完全没有入口，隐藏依赖后端键命名约定。
- **建议**: 配置键与按钮解耦：client-grid 按 { clientKey, configKey, os } 数据驱动，展示与 detectSystem 匹配的下载项，或明确提示『当前为自动检测架构的通用下载』。

### [INFO] 订阅链接走 5 分钟缓存，套餐变更后复制到的可能是旧 URL
- **位置**: 90-101, 157  |  **类别**: performance  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: `cachedAPI.getUserSubscription()` 缓存 5 分钟（api.js line 799-803），loadRuntimeData 在 onMounted 取一次；用户在订阅重置/变更后 5 分钟内点『复制订阅』会复制过期链接。
- **建议**: 复制订阅时改为直连 subscriptionAPI.getUserSubscription() 绕过缓存，或监听订阅变更事件主动 clearUserCache。

## frontend/src/components/tutorials/WindowsTutorials.vue

### [HIGH] 6 个客户端面板近 340 行同构模板，复制粘贴最严重文件
- **位置**: 4-288  |  **类别**: duplication  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: Clash for Windows / V2rayN / Clash Part / Clash Verge / Hiddify / FlClash 六面板结构完全同构（下载→安装→导入订阅方法一二→使用→tips），其中 Clash Part/Clash Verge 与 MacOSTutorials 对应面板内容几乎逐字重复；下载/安装步骤段落 90% 雷同。
- **建议**: 同 MacOSTutorials：数据驱动 + TutorialPanel 组件；并将各客户端『订阅地址类型』（Clash 订阅地址 vs 通用订阅地址）提取为数据字段，避免手抄错误。

### [MEDIUM] 6 个客户端中 5 个的『点击上方立即下载按钮』无对应按钮
- **位置**: 7-288  |  **类别**: ux  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: Windows 标签页只有 Clash Verge 有下载行（SoftwareTutorials 19-30），Clash for Windows/V2rayN/Clash Part/Hiddify/FlClash 各自开头的『点击上方"立即下载"按钮下载 xxx 最新版本』指向不存在的控件；V2rayN/Hiddify/FlClash 均已在 githubDownload.js 配置（v2rayN/hiddify-app/FlClash），具备接入下载按钮的数据条件却未渲染。
- **建议**: 为每个客户端补数据驱动的下载行（configKey + githubKey 均已在 githubDownload.js 具备），或统一改写文案。

## frontend/src/components/tutorials/iOSTutorials.vue

### [INFO] 平台覆盖不对称：iOS 仅 Shadowrocket 单面板，且无逻辑问题
- **位置**: 1-188  |  **类别**: other  |  **来源组**: F3-layout-tutorials (布局 + 教程组件)
- **问题**: 本文件逻辑上无 bug：仅一个 el-collapse-item（Shadowrocket），导入步骤/警告/高级设置/排障结构完整，activeNames 默认展开正确，样式与兄弟组件重复（见 MacOSTutorials 重复项）。与 Windows(6)/macOS(3)/Android(3) 相比覆盖明显不足，若 iOS 是刻意只支持 Shadowrocket 建议在页面加说明，否则应补齐 Quantumult X / Stash 等面板。
- **建议**: 确认 iOS 客户端覆盖范围；如补充客户端，复用 MacOSTutorials 建议的数据驱动 TutorialPanel。

## frontend/src/composables/useDebounce.js

### [MEDIUM] 防抖搜索无过期请求保护，快速输入会产生乱序结果覆盖
- **位置**: 14-32  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: debouncedSearch 只延迟调用 searchFn，不追踪请求序号：连续输入时先发的慢请求可能后返回，用旧结果覆盖新结果（第 20-26 行 await searchFn(searchValue.value) 无任何次序校验）；另外 watch 无 immediate，初始值不会触发搜索。
- **建议**: 维护 requestId/seq 计数器，回调返回后比对 seq 才写结果；或改用 AbortController 取消旧请求；需要时给 watch 加 { immediate: true }。

### [LOW] searchFn 抛错时产生未处理的 Promise rejection
- **位置**: 20-26  |  **类别**: error-handling  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: setTimeout 内 `await searchFn(...)` 只在 finally 里重置 isSearching，没有 catch，searchFn 失败会变成 unhandledrejection，组件无法感知错误状态。
- **建议**: 加 catch 并暴露错误状态（或至少 .catch(() => {}) 吞掉并置位 isSearching=false）。

## frontend/src/composables/useMobile.js

### [INFO] 无明显问题（rAF 合并 resize 处理正确）
- **位置**: 1-39  |  **类别**: other  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: checkMobile/scheduleCheckMobile 用 requestAnimationFrame 合并高频 resize，onUnmounted 正确移除监听并取消 rAF，被动监听 passive:true，实现干净。仅可考虑用 matchMedia 替代 innerWidth 判断以更贴近 CSS 断点。
- **建议**: 可选：window.matchMedia(`(max-width: ${breakpoint}px)`) + change 事件，减少 resize 全量触发。

## frontend/src/composables/usePaymentStatusPolling.js

### [MEDIUM] poll() 的 rejection 无人处理，setInterval 内产生未捕获异常且失败后继续轮询
- **位置**: 35-41  |  **类别**: error-handling  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: runPoll 里 `await poll()` 无 try/catch，poll 抛错（如网络中断）时：interval 回调里抛出 → unhandledrejection；同时轮询不停止、不降频、无退避，失败期间每秒都在重复打接口。
- **建议**: runPoll 内 catch poll 错误并记录失败计数，连续失败 N 次自动 clearPolling（或指数退避）；至少 .catch(() => {}) 防止未处理 rejection。

### [LOW] startPolling 重复调用会触发多次 onCleanup；超时结束无法通知调用方；可见性/焦点与 interval 无并发去重
- **位置**: 43-60  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: startPolling 先 clearPolling()（内部调用 onCleanup?.()），连续两次 startPolling 会让 onCleanup 执行两次（若 onCleanup 里清支付状态可能出错）；timeoutId 到点只清轮询不回调，调用方无法区分“轮询结束”与“支付超时”；visibilitychange/focus 触发的 runPoll 与 interval 的 runPoll 可能并发执行同一 poll。
- **建议**: clearPolling 增加 inProgress 标记避免重复执行 onCleanup；增加 onTimeout 回调；runPoll 内加 inFlight 互斥锁。

## frontend/src/composables/usePersistentTableColumns.js

### [LOW] 列宽存储键回退到 column.label，表头改名/重名会丢失或互相覆盖宽度
- **位置**: 32-37  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: handleColumnResize 取 `column?.property || column?.columnKey || column?.label`——el-table 的 property 通常是字段名，label 是表头文案；依赖 label 意味着改表头即失效，且两列同名 label 会互相覆盖宽度；同时每次 resize 事件都同步写 localStorage（拖拽过程中高频触发）。
- **建议**: 强制调用方传入稳定的列 key（如 prop 或唯一 columnKey），label 仅作最后兜底并 warning；对 saveColumnWidths 做 200-300ms 防抖。

### [LOW] resetColumnWidths 后仍会把默认宽度写回 localStorage，'重置' 并未真正清除持久化
- **位置**: 39-45  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 43-44 行 Object.assign(columnWidths, defaultWidths) 后立即 saveColumnWidths()，localStorage 里仍是 {columnWidths: {...}} 记录，只是值回到默认；用户预期“重置”应清除存储并恢复初始状态。
- **建议**: resetColumnWidths 改为 localStorage.removeItem(storageKey) 并仅重置内存对象，不再主动写回。

## frontend/src/main.js

### [MEDIUM] 生产环境错误全部静默，$settings 初始化占位为 null 有 NPE 风险
- **位置**: 35-41  |  **类别**: error-handling  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: app.config.errorHandler 只在 development 下 console.error，生产环境任何 Vue 渲染/生命周期错误被完全吞掉，无任何上报通道（第 35-39 行）。第 41 行 app.config.globalProperties.$settings = null，在 loadSettings 完成前访问 this.$settings 的组件会拿到 null 直接报错。
- **建议**: 生产环境至少把 errorHandler 接一个远程错误上报（或统一 console.error 保留现场）；不要用 null 占位，改为懒加载 getter 或让组件统一从 useSettingsStore() 取。

### [MEDIUM] Promise.all 只包了单个任务，主题在 mount 之后异步应用造成首屏主题闪烁
- **位置**: 45-66  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: Promise.all 数组里只有一个 async 任务（第 47-66 行），结构是死结构；且主题应用在 app.mount('#app')（第 42 行）之后才执行，用户首帧渲染的是未应用主题的默认样式，随后再切换，出现 FOUC/闪烁。若 loadSettings 抛错，catch 仅 console.error，主题完全不应用。
- **建议**: 删除 Promise.all 外层；把主题初始化提前到 mount 之前（先用 localStorage 'user-theme' 同步 applyTheme 一次再 mount），网络加载的 defaultTheme 仅在成功返回后二次应用，减少闪烁。

## frontend/src/router/index.js

### [HIGH] 路由守卫 catch 里直接 next() 放行，任何异常都会绕过鉴权检查
- **位置**: 263-268  |  **类别**: security  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 守卫主体 try 块内对 storedUser 的 JSON.parse（第 242 行）、secureStorage 操作等若抛异常，会落入 catch 直接 next() 放行——用户被放进受保护页面且 authStore 未设置，前端路由级鉴权整体失效（仅剩后端兜底）。
- **建议**: catch 里按 to.meta.requiresAuth / requiresAdmin 决定重定向到 /login 或 /admin/login，而不是无条件下放；把 JSON.parse 拆成安全解析，避免损坏数据拖垮整个守卫。

### [MEDIUM] restoreAuthFromRefresh 用裸 axios 刷新令牌，未带 CSRF 头/凭据，与 api.js 的刷新路径不一致
- **位置**: 127-153  |  **类别**: security  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 136-140 行 axios.post('/api/v1/auth/refresh', ...) 没有 withCredentials、没有 X-CSRF-Token 头；而 api.js 拦截器刷新路径（api.js 第 420-426 行）会附加 X-CSRF-Token。若后端对 POST /auth/refresh 做 CSRF 校验，页面加载时的静默刷新必然 403 失败，用户被强制重新登录，且失败后第 170/182 行还会把 refresh token 删掉造成令牌丢失。
- **建议**: 复用 api.js 暴露的刷新函数（或同样的 CSRF 头 + withCredentials 逻辑），保证两处刷新行为一致；删除失败时不要立即 remove refresh token，先走 logout 标记。

### [LOW] 刷新令牌写入重复执行两次
- **位置**: 147-151  |  **类别**: maintainability  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 147 行和第 151 行是逐字相同的 `if (newRefresh) secureStorage.set(refreshKey, newRefresh, useSessionStorage, ADMIN_USER_TTL)`，第 151 行为重复语句。
- **建议**: 删除第 151 行重复代码。

### [LOW] 登录交接凭据经 URL query 传递，存在泄露面且可被任意伪造触发跳转
- **位置**: 108-125, 188-205  |  **类别**: security  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: sessionKey 随 URL 传输（浏览器历史、服务端日志、外链 referrer 均可能留存），虽然数据 5 分钟过期且读取即删除，但同一源下任何脚本都可构造 `?sessionKey=随机值` 让已登录用户访问任何受保护页时被 ElMessage('登录信息已过期') 并踢到 /login（第 203-204 行），可被用作骚扰性重定向。
- **建议**: 改用 sessionStorage 同名键 + 短随机 key 的 channel 模式并校验来源，或限定 sessionKey 只能作用于登录/回调专用路由；对无有效数据的 sessionKey 静默忽略而不是跳登录。

## frontend/src/store/auth.js

### [MEDIUM] login() 只校验 userData 不校验 access_token，异常响应会返回 success:true 的假登录成功
- **位置**: 143-150  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 解构出 { access_token, refresh_token, user } 后仅 `if (!userData)` 拦截；若后端返回缺 token 的畸形响应，token.value = undefined、saveToken(undefined) 会把 undefined 值序列化进存储，而函数仍返回 { success: true, isAdmin }，页面显示登录成功但实际无令牌，后续请求全部 401。
- **建议**: 增加 `if (!access_token || !userData) return { success: false, message: '登录响应格式错误' }`，并把 responseData 为 null/undefined 的情况一并拦截。

### [MEDIUM] login() 硬编码 remember = true，忽略登录表单的记住我选项，令牌必然持久化到 localStorage
- **位置**: 173-177  |  **类别**: security  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 173 行 `const remember = true` 且登录参数只取 username/password（第 139-142 行），用户无论是否勾选“记住我”，access_token 与 refresh_token（30 天 TTL）都会被写入 localStorage，XSS 窃取窗口被固定拉长，“记住我”复选框形同虚设。
- **建议**: 从 credentials 读取 remember 布尔值（默认 false，仅显式勾选时持久化），并按该值决定 saveToken/saveUser/saveRefreshToken 的 useSession。

### [LOW] refreshToken() 不校验响应是否含 access_token，且与 api.js 拦截器存在双份刷新实现
- **位置**: 261-281  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 270-276 行直接 `token.value = access_token`，未检查 access_token 是否存在；同时 api.js 响应拦截器（api.js 第 423-441 行）已经实现了完整的刷新+队列逻辑，store 里再写一份行为不同（不带 CSRF、不挂队列）的刷新，两套逻辑容易漂移。
- **建议**: store.refreshToken 改为只调用 api.js 暴露的统一刷新函数并校验返回值；若保留现状，至少补 `if (!access_token) throw`。

### [LOW] handleApiError 对预期内失败（密码错误等）也 console.error 打日志
- **位置**: 120-135  |  **类别**: style  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 130 行对用户输错密码这类可预期 4xx 也输出 console.error('登录错误:', ...)，生产环境会刷大量无意义错误日志，且错误信息提取链 detail/message/error 与后端实际响应结构需保持一致。
- **建议**: 仅对网络异常/5xx 打 error 日志，业务 4xx 静默返回 message 即可。

## frontend/src/store/settings.js

### [MEDIUM] 默认主题值前后不一致：'default' vs 'light'，resetSettings 与初始 state 互相矛盾
- **位置**: 18-20, 78, 108, 206-208  |  **类别**: architecture  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 初始 state.defaultTheme='default'（第 20 行），但 loadSettings 拉不到时回退 'light'（第 78 行），resetSettings 又重置回 'default'（第 208 行）；applyTheme 的 themeClasses 含 'default' 而 updateThemeVariables 的 themeColors（第 115-164 行）只有 default/dark/blue/green，没有 'light' 键（'light' 会落到 default 配色）。三个来源三种值，主题行为不可预测。
- **建议**: 统一一个默认值（建议 'light' 或明确支持 'default' 并在 theme store 中注册），resetSettings 与初始 state 保持同一常量，themeColors 与 theme store 键集对齐。

### [LOW] allowQqEmailOnly 默认值前后矛盾（初始 true vs resetSettings false），且 API 未返回时默认为 true
- **位置**: 18, 71, 206  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: state 初始 allowQqEmailOnly: true（第 18 行），第 71 行 `settings.allow_qq_email_only !== false` 使后端未下发该字段时也强制 true（QQ 邮箱限制开启），而 resetSettings 却重置为 false（第 206 行）——同一字段存在两种默认语义，可能意外拦住非 QQ 邮箱注册。
- **建议**: 明确产品默认（建议关闭），state 初始值与 resetSettings 统一；loadSettings 对缺失字段回退到与 state 一致的默认值。

### [LOW] currentTheme getter 直接读 localStorage，且 loadSettings 设置 document.title 与路由守卫竞争
- **位置**: 34-40, 87  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: getter 内 `localStorage.getItem('user-theme')` 在隐私模式/存储禁用时直接抛异常会击穿 getter；第 87 行 document.title = siteName 与路由守卫（router/index.js 第 156 行）的 `${title} - CBoard` 相互覆盖，谁后执行谁生效，结果随机。
- **建议**: getter 内 try/catch 包裹存储读取；title 的赋值收敛到单一职责（路由守卫统一处理，settings 不再覆盖）。

## frontend/src/store/theme.js

### [MEDIUM] themeConfigs 无 'default' 键，而 settings store 的默认主题是 'default'，两套主题引擎键集不一致
- **位置**: 80-241, 249  |  **类别**: architecture  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: theme.js 的 themeConfigs 只有 light/dark/blue/green/purple/orange/red/cyan/luck/aurora；settings.js 初始 defaultTheme='default'、availableThemes 也含 'default'。main.js 第 58-61 行 `themeStore.applyTheme(settingsStore.defaultTheme)` 传 'default' 时静默落到 `themeConfigs.light`（第 249 行），而 settings.js 自己的 applyTheme/themeColors 却有 'default' 键——同一主题在两套引擎下渲染出不同颜色，且 'default' 与 'light' 语义混乱。
- **建议**: 统一主题键集：删掉 'default' 或让 themeConfigs 显式定义 'default'，两 store 共用一份 themeConfigs（抽到 utils 或单一 store 暴露），settings store 不再维护自己的 themeColors 副本。

### [MEDIUM] setTheme 的嵌套 try/catch 控制流混乱：localStorage 失败会连带跳过云端保存
- **位置**: 41-79  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 外层 try 先 applyThemeLocally（含 localStorage.setItem），内层 try 才做 API 同步；一旦 localStorage 抛异常（隐私模式/配额满），内层 API 保存整个被跳过，主题永远无法同步到云端；外层 catch 里还重复调用 applyThemeLocally 再次可能抛错。三个 return 分支可读性差。
- **建议**: 重构为：先独立 try 本地应用（catch 记 warning），再独立 try API 同步（catch 返回 localApplied:true），两个环节互不阻塞；避免双层嵌套。

### [LOW] matchMedia.addEventListener 兼容性及监听器生命周期
- **位置**: 335-344  |  **类别**: style  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: initTheme 用 mediaQuery.addEventListener('change')，Safari ≤13 需 addListener；且监听器注册后永不注销（store 单例尚可接受，但若被多实例/测试环境调用会重复注册）。
- **建议**: 加 `mediaQuery.addEventListener ? addEventListener : addListener` 兼容写法，并考虑导出注销函数。

## frontend/src/styles/button-common.scss

### [HIGH] 同一套 :has() 按钮网格规则在本文件复制 6 次
- **位置**: 11-279  |  **类别**: duplication  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: .action-button-group/.el-button-group/.action-buttons-grid/.filter-button-group/.page-header-actions/.toolbar-buttons 六个类的移动端块几乎逐字相同（`grid-template-columns: repeat(3,…)` + `:has(> :nth-child(1):last-child)`…`:nth-child(3):last-child > :nth-child(3)` 一组变体 + `.el-button` 44px 全宽）。与 list-common/mobile-buttons/user-client-polish 合计约 40 处重复，是该库最大的重复源。
- **建议**: 定义 `@mixin mobile-action-grid($cols: 3)` 并全文件复用；长期可收敛为单一 .actions-grid 工具类。

### [MEDIUM] @extend Element Plus 组件类在独立编译单元下是 no-op 死代码
- **位置**: 319-346  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `.btn-primary-action { @extend .el-button--primary !optional; }` 等 6 个语义类依赖 @extend 命中 Element Plus 的类；但 EP 样式是通过按需 sideEffects CSS（vite.config.js 的 resolver 引入 '.../style/css'）在 JS 侧加载的，与本 SCSS 文件不属同一 Sass 编译单元，@extend 编译期根本找不到 .el-button--primary，!optional 静默吞掉错误——这些类实际不产生任何样式，属于无声失效的 API。
- **建议**: 删除这些类，模板直接使用 el-button 原生 type；若确需语义类，改为在组件内显式 `@use` EP 的 theme-chalk 源码后 extend，并加编译期断言。

### [LOW] .btn-loading 与 Element Plus 原生 loading 状态重复造轮子
- **位置**: 352-366  |  **类别**: duplication  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: EP 的 el-button 已内置 loading 属性（含 spinner 与禁用）；自定义 .btn-loading 用 ::after 半透明遮罩 + pointer-events:none 实现类似效果，遮罩还会盖住按钮文字，与 EP 方案行为不一致。
- **建议**: 删除 .btn-loading，统一使用 `:loading` 属性。

## frontend/src/styles/dialog-common.scss

### [MEDIUM] 弹窗尺寸类 .dialog-xs~xl 在移动端被 global.scss 的 88% !important 覆盖而失效
- **位置**: 53-57, 60-87  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: .dialog-sm 等尺寸类用 `width: 480px !important`（54 行），移动端块又统一改 92% !important（80-86 行）；但 global.scss 的 `@media 768 .el-dialog { width: 88% !important }`（520-521、1180-1184 两处）与尺寸类同为 !important、同特异性，按源码顺序（dialog-common 先 @use）后者的 88% 胜出——尺寸类在移动端实际全部失效，且 92%/88%/94vw/min(420px,100dvw-20px) 四处宽度策略并存。
- **建议**: 把移动端弹窗宽度收敛为单一来源（建议 CSS 变量 --dialog-mobile-width），尺寸类仅负责桌面端，删除重复的 92%/88% 声明。

### [LOW] 关闭按钮颜色 fallback 与变量语义不符
- **位置**: 29-36  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `.el-dialog__close { color: var(--theme-text, #909399) }`：--theme-text 语义是正文文本色（默认 #303133），把正文色当图标灰用，fallback #909399 才是想要的灰色——变量与回退值语义错位，浅色下图标会偏深。
- **建议**: 改为独立变量或直接 var(--theme-info, #909399)。

## frontend/src/styles/global.scss

### [MEDIUM] 移动端断点样式三块重复定义且数值互相冲突
- **位置**: 439-891 vs 1088-1215 vs 1216-1241  |  **类别**: duplication  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `@include respond-to(sm)`（439-891）、`@media (max-width:768px)`（1088-1215）和第二个 768px 块（1216-1241）重复设置同一批规则：`.el-dialog` 宽度 88% !important 出现两处、`.el-input__inner` font-size 16px 出现两处、`.el-button` min-height 多处；且 dialog-common.scss 里 92% !important 与这里的 88% !important 同特异性，最终胜负完全由 @use 顺序决定。同一断点下样式来源不可预测。
- **建议**: 合并为单一移动端断点块；弹窗/按钮尺寸收敛为唯一 token（如 --dialog-mobile-width），删除重复声明。

### [MEDIUM] global.scss 被编译进 3 个位置，整份全局 CSS 重复打包约 3 倍
- **位置**: 3-7, UserLayout.vue:380, AdminLayout.vue:437, main.js:19  |  **类别**: performance  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: main.js:19 `import './styles/global.scss'` 是入口级导入；而 UserLayout.vue:380 与 AdminLayout.vue:437 各自在 `<style scoped lang="scss">` 里写 `@use '@/styles/global.scss' as *;`。Vite/Sass 对每个 SFC style 块独立编译，@use 会在该编译单元内发射模块全部 CSS（含 dialog-common/list-common/list-unified/mobile-buttons/button-common 五个 partial，合计约 3700 行）。结果是这整份样式被发射 3 次：一次全局 + 两次带 scoped 属性选择器的副本，后者既是纯重复，又会因 `[data-v-xxx]` 前缀产生特异性干扰。
- **建议**: 把 mixin/变量（respond-to、$breakpoints 等）抽到不含 CSS 输出的纯 partial（如 _mixins.scss），两个布局组件改为 `@use '@/styles/_mixins.scss' as *;`；global.scss 的 CSS 只保留 main.js 一个入口。

### [LOW] .theme-auto 只是 @extend .theme-light，并未实现“跟随系统”语义
- **位置**: 109-111  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `.theme-auto { @extend .theme-light; }` 使 auto 与 light 完全等价，没有任何 `@media (prefers-color-scheme: dark)` 逻辑；选择 auto 主题的暗色系统用户仍得到浅色界面。
- **建议**: 用 `@media (prefers-color-scheme: dark) { .theme-auto { ...dark 变量覆盖... } }` 或在前端 store 里按 matchMedia 解析后再落到 theme-light/theme-dark。

### [LOW] .el-button/.el-card 使用 transition: all 并带 hover 位移
- **位置**: 892-917  |  **类别**: performance  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `.el-button { transition: all 0.3s ease; &:hover { transform: translateY(-1px) } }` 与 `.el-card { transition: all 0.3s ease }` 会对全部属性做过渡；在移动端（@media hover:none 下）hover 位移无意义且引发重绘，过渡属性过宽。
- **建议**: 改为显式属性列表（transition: background-color .2s, box-shadow .2s, transform .2s），并考虑在 touch 设备跳过 transform。

### [LOW] 主题变量三套命名并行 + 十六进制大小写混用 + 断点死配置
- **位置**: 9-30, 425-438  |  **类别**: style  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 全局存在 --theme-primary（global.scss）、--primary-color（list-common.scss:2）与 Element Plus 的 --el-color-primary 三套并行变量；同文件内 #409EFF 与 #409eff 混用（10 行 vs 950 行）。$breakpoints 定义了 xs/sm/md/lg/xl 五个键，但 respond-to 全项目只用到 sm，其余为死配置。
- **建议**: 统一 token 命名（推荐全部收敛到 --el-* 或单一 --theme-* 前缀），hex 统一小写，删除未用断点或补全使用。

## frontend/src/styles/list-common.scss

### [HIGH] :has(:nth-child) 移动端按钮网格模式在 5 个类上整段复制
- **位置**: 609-649  |  **类别**: duplication  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `.mobile-card-actions/.card-actions/.action-buttons/.button-row/.batch-buttons` 五个类各自完整复制同一套 `:has(> :nth-child(1):last-child)`…`:nth-child(5):last-child > :nth-child(4)` 网格规则（609-649）；同一模式又在 button-common.scss（约 6 处）、mobile-buttons.scss、user-client-polish.scss（12 类 × 4 变体）重复，全库合计约 40 处相同代码块。任何一处漏改都会造成不同容器按钮布局漂移。
- **建议**: 抽一个 Sass 占位符 `%mobile-actions-grid`（含 :has 变体），各容器类统一 `@extend %mobile-actions-grid;`，或收敛为单一工具类 .actions-grid 由模板统一使用。

### [MEDIUM] 与 list-unified.scss 重复定义同一批类且数值冲突
- **位置**: 1044-1056, 1163-1185, 1195-1205  |  **类别**: duplication  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: .pagination-wrapper/.records-tabs/.empty-state/.loading-wrapper/.mobile-only/.mobile-card-optimized 在 list-common.scss 与 list-unified.scss 各定义一份：.empty-state padding 48px vs 60px、.loading-wrapper min-height 360px vs 400px、.mobile-card-optimized 有 1px 边框 vs 无边框、圆角 var(--border-radius) vs 12px。两文件都被 global.scss @use，胜负仅由第 4/5 行顺序决定，谁改谁错。
- **建议**: 合并两文件为单一列表样式 partial，或明确职责（common 管结构、unified 只补工具类）并删除重复定义。

### [MEDIUM] `.list-container[class*="admin"]` 子串属性选择器会误伤无关页面
- **位置**: 118-238  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `[class*="admin"]` 匹配任何 class 属性中包含 "admin" 的元素——如 user-admin-xxx、admin-tools 等命名都会命中；且该规则用 !important 强制 .stat-card padding:0、min-height:0，会把非管理页的统计卡样式也改掉，属于脆弱的“按名字猜语义”选择器。
- **建议**: 改用明确作用域类名 `.list-container.admin`（需模板配合），或整体限定在 `.admin-layout .list-container` 下。

### [LOW] 按钮圆角/换行策略与全局 token 不一致
- **位置**: 1243-1258  |  **类别**: style  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `.el-button { border-radius: 7px }`（1244）与 global.scss 的 `--border-radius: 8px`、button-common 输入组 6px、user-client-polish 的 4px !important 各自为政；1257 行移动端 `.el-button { white-space: nowrap }` 又与 678 行 `.card-actions .el-button { white-space: normal }` 语义相反，靠特异性掩盖冲突。
- **建议**: 按钮圆角统一使用 var(--border-radius)，移动端按钮换行策略收敛为单一规则。

## frontend/src/styles/list-unified.scss

### [MEDIUM] 文件头注释宣称“避免两个全局文件覆盖同一批类名”，行为恰好相反
- **位置**: 1-4  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 注释明确说“基础列表结构统一由 list-common.scss 负责，避免两个全局样式文件覆盖同一批类名”，但本文件 6-46 行、48-132 行、134-165 行仍在重复定义 .pagination-wrapper/.mobile-card-optimized/.records-tabs/.empty-state/.loading-wrapper/.mobile-only（均为 list-common.scss 已有类），注释与实现自相矛盾，是“先立规后破规”的典型。
- **建议**: 要么删除本文件中与 list-common 重复的类定义只留新增工具类，要么把注释改为说明二者合并后的职责边界。

### [LOW] 无前缀工具类（.text-xs/.gap-2 等）有全局污染风险
- **位置**: 173-215  |  **类别**: style  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: .text-*、.gap-*、.amount、.status-tag 等均为无命名空间全局类，若后续引入 Tailwind/其他 UI 库或第三方组件同名单类会互相覆盖；.text-primary/.text-success 等也与 Element Plus 语义色体系重复。
- **建议**: 加业务前缀（如 .cb-text-xs）或限定在 .list-container/.user-layout 作用域内。

## frontend/src/styles/mobile-buttons.scss

### [MEDIUM] 移动端 .form-actions 布局与 button-common.scss 方向相反，仅靠 @use 顺序裁决
- **位置**: 141-157  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 本文件 `.form-actions,.action-buttons { flex-direction: column }`（141-157），而 button-common.scss `.form-actions { flex-direction: column-reverse }`（180）；global.scss @use 顺序是 mobile-buttons(第6行) 先、button-common(第7行) 后，因此 column-reverse 胜出——但如果有人调整 @use 顺序或单独 import 某文件，提交/取消按钮顺序会整体反转，属于“顺序决定行为”的脆弱设计。.dialog-footer 也在此文件内重复两遍（98-121 与 159-174）。
- **建议**: 合并两个文件、收敛为单一规则，并在注释中显式说明设计意图（column-reverse 是移动端习惯的“主按钮在下”）。

### [LOW] .mobile-action-btn 样式在 3 个文件中重复定义
- **位置**: 86-96  |  **类别**: duplication  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: .mobile-action-btn（min-height:44px、border-radius:6px、white-space:normal 等）在 mobile-buttons.scss、user-client-polish.scss（.admin-layout .logs-page .mobile-action-btn，1645-1648）以及 list-common 的按钮网格块中重复声明，改动需同步多处。
- **建议**: 将 .mobile-action-btn 收敛为一个全局定义（或按钮 mixin），其余处复用。

## frontend/src/styles/text-selection.css

### [MEDIUM] 全局 `* { user-select: auto !important }` 会破坏依赖 user-select:none 的拖拽组件
- **位置**: 1-7  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 第 1-7 行对所有元素强制 user-select:auto !important，会覆盖 Element Plus 内部（el-slider 拖拽、el-rate、el-carousel、日期面板拖动等）以及 sortablejs 拖拽排序对 user-select:none 的依赖，导致拖拽时文本被选中、拖拽卡顿；白名单只枚举了按钮/标签类元素，漏掉了滑块、拖拽手柄等。
- **建议**: 反转策略：去掉全局 * 的 !important，只对需要禁选的交互元素（按钮、标签、滑块、拖拽句柄）显式设 user-select:none，其余交给浏览器默认。

### [LOW] 复选框/单选框标签文本也被禁选，可读性受影响
- **位置**: 8-31  |  **类别**: ux  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: .el-checkbox/.el-radio 整体 user-select:none，用户无法选中复制选项文字（如“同意服务条款”）；虽然按钮类禁选合理，但 checkbox 标签属于信息文本，禁选不符合无障碍习惯。
- **建议**: 仅对 .el-checkbox__inner/.el-radio__inner 等控件图形禁选，保留标签文本可选。

## frontend/src/styles/user-client-polish.scss

### [HIGH] 2235 行 god 文件 + 数百处 !important 的特异性军备竞赛，前段规则被后段静默覆盖成死代码
- **位置**: 1-2235  |  **类别**: architecture  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 文件自述“final pass”（1696 行注释）叠在 legacy overrides 之上：同一选择器在文件内多次出现且后段用 !important 覆盖前段，如 `.user-layout .stat-number` 先是 `color: var(--theme-primary,#409eff)`（466-474）后是 `color: #303133 !important`（1261-1268）；`.user-layout .el-dialog` 先 `width: 94vw !important`（1009-1013）后被 `width: min(420px, calc(100dvw - 20px)) !important`（2095-2099）覆盖；`.card-header` font-weight 700（538-544）vs 800 !important（1199-1209）。前段声明全部是无效死代码，任何修改都无法判断哪一条在生效。
- **建议**: 重构为按页面/模块拆分的 scoped 样式；删除被后段覆盖的前段规则；!important 收敛到个位数（优先用更高特异性选择器）；为文件建立“规则唯一性”检查。

### [MEDIUM] .admin-layout .logs-page 样式块放进了名为 user-client-polish 的文件
- **位置**: 1602-1653  |  **类别**: architecture  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 1602-1653 行是管理端日志页样式（.admin-layout .logs-page 的 filter-bar/表格），与文件的 .user-layout 客户端主题职责无关，属于放置错位；后续维护者会在错误文件里找管理端样式。
- **建议**: 把该块迁到对应页面组件或 admin 专用样式文件。

### [MEDIUM] 12 个操作容器类 × 4 组 :has() 变体的网格规则再次整段复制
- **位置**: 1786-1865  |  **类别**: duplication  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: `.page-header .actions/.header-actions/.button-row/.actions/.action-buttons/.card-actions/.client-actions/.app-actions/.payment-actions-container/.coupon-buttons/.filter-actions/.orders-filter-actions` 12 个类分别重复 `:has(> :nth-child(1):last-child)`、`:nth-child(2/3/4):last-child`、`:nth-child(3):last-child > :nth-child(3)`、`:nth-child(5):last-child > :nth-child(4)` 四组规则，约 80 行 × 12 的纯复制，是全库 :has 重复模式的最大单块。
- **建议**: 合并选择器列表为 `%mobile-actions-grid` 占位符一次定义；若容器 DOM 结构一致，直接给模板统一加 .actions-grid 类。

### [LOW] 使用已废弃的 -webkit-overflow-scrolling: touch 且 dvh/vw 混用
- **位置**: 2042, 2092, 2148  |  **类别**: style  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 2042/2092/2148 行仍写 `-webkit-overflow-scrolling: touch`（Safari 已废弃，无效果）；同时移动端宽度既有 calc(100dvw - 20px)（2096）也有 94vw/100vw（1010 行等），视口单位策略不统一。
- **建议**: 删除废弃属性；统一移动端弹窗宽度为 dvw 或 vw 单一策略。

## frontend/src/utils/api.js

### [HIGH] nodeAPI 与 adminAPI 重复定义 9 个相同的节点管理方法
- **位置**: 528-545 vs 608-616  |  **类别**: duplication  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: nodeAPI（528-545）与 adminAPI（608-616）各自定义 getAdminNodes/createNode/importNodeLinks/updateNode/getNodeLink/deleteNode/testNode/batchTestNodes/batchDeleteNodes 且端点一致；Nodes.vue 用 adminAPI 副本，改一处漏一处即契约漂移。
- **建议**: 删除 nodeAPI 中的 admin 方法，admin 侧统一收敛到 adminAPI。

### [MEDIUM] 同一业务概念存在多条不一致的 API 路径（设备删除、订单列表、套餐列表）
- **位置**: 481, 497, 514-515  |  **类别**: architecture  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: userAPI.deleteDevice → DELETE '/devices/:id'（第 481 行）vs subscriptionAPI.removeDevice → DELETE '/subscriptions/devices/:id'（第 497 行）；orderAPI.getOrderList → GET '/orders'（第 514 行）vs getUserOrders → GET '/orders/'（第 515 行，尾斜杠不同）；packageAPI.getPackages('/packages/') 与 orderAPI.getPackages('/packages/')（第 504/519 行）重复。路径风格不统一，前后端契约易错。
- **建议**: 梳理后端实际路由，统一为一套规范路径（无尾斜杠、单一路由语义），删除别名，建立 api 层单一事实来源。

### [MEDIUM] config-update 端点组在 paymentAPI 与 configUpdateAPI 中各定义一遍，且职责错位
- **位置**: 699-706, 724-734  |  **类别**: duplication  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: paymentAPI 里塞了 getConfigUpdateStatus/startConfigUpdate/stopConfigUpdate/testConfigUpdate/getConfigUpdateLogs/getConfigUpdateConfig/updateConfigUpdateConfig/clearConfigUpdateLogs（第 699-706 行），configUpdateAPI 又逐字重复这些端点（第 724-734 行）——同一 URL 两份定义，后续改一处漏一处；节点更新与支付配置明显无关，属职责错位。
- **建议**: 删除 paymentAPI 中的 config-update 方法，只保留 configUpdateAPI 一份定义；paymentAPI 只留支付相关端点。

### [MEDIUM] adminAPI 逐字重复 nodeAPI 的管理端方法（getAdminNodes/createNode/updateNode 等 9 个）
- **位置**: 534-544, 608-616  |  **类别**: duplication  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: nodeAPI 已定义 getAdminNodes/getAdminNode/createNode/importNodeLinks/updateNode/getNodeLink/deleteNode/testNode/batchTestNodes/batchDeleteNodes（第 534-544 行），adminAPI 又原样复制一遍（第 608-616 行），两处 URL 完全一致，属复制粘贴，后续路径调整极易不同步。
- **建议**: adminAPI 直接引用 nodeAPI 的对应方法（`getAdminNodes: nodeAPI.getAdminNodes`）或合并为一个 API 模块按域拆分。

### [MEDIUM] PUBLIC_APIS 用 startsWith 前缀匹配，私有端点可能被误判为公开而跳过 Authorization
- **位置**: 173-183, 300-303  |  **类别**: security  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 301 行 `PUBLIC_APIS.some(api => config.url.startsWith(api))` 是前缀匹配：任何以公开路径为前缀的私有端点（如未来新增的 '/auth/login-as/1'、'/coupons/verify-admin'、'/auth/login-history'）都会命中，请求不带 Authorization 头直接发出，安全边界随路径命名变化而漂移。
- **建议**: 改为按路径段精确匹配：`PUBLIC_APIS.some(p => config.url === p || config.url.startsWith(p + '/'))`，或维护显式路由表。

### [LOW] 存在两套登录端点：authAPI.login('/auth/login') 与 store login('/auth/login-json')
- **位置**: 462-470  |  **类别**: duplication  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: authAPI.login 指向 POST /auth/login（第 463 行），auth store 的 login() 却用 POST /auth/login-json（auth.js 第 139 行），两个端点并存，至少一个是死代码或与后端契约不一致。
- **建议**: 确认后端保留哪个登录端点，删掉另一个及对应包装；对整个 api.js 做一次未引用导出清理。

### [LOW] 刷新成功后令牌被写两次，且 setToken 会无条件替换内存 token 并按 URL 路径决定角色存储
- **位置**: 430-437  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 435 行已按 isAdminAPI 用 shouldRememberRole 精确写入对应角色存储，随后第 437 行又调 `_useAuthStore().setToken(access_token)`——setToken（auth.js 第 323-327 行）按 window.location.pathname 判断角色并用 getRememberPreference 决定存储策略，与刚才的写入策略可能不一致（双写同一令牌、策略不同），且 setToken 无条件覆盖内存 token.value，即使内存中当前是另一个角色的 token。
- **建议**: 去掉第 437 行的 setToken 调用，刷新成功后只更新对应角色存储，必要时用按角色更新的方法同步内存态。

### [LOW] _retry 标记同时被超时重试与 401 刷新共用，超时重试过的请求再遇 401 会被直接登出
- **位置**: 338-342, 403-412  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 超时重试在第 341 行置 `error.config._retry = true`；该请求重试后若返回 401，第 403 行 `!error.config._retry` 为 false，跳过令牌刷新直接落入第 453-456 行 clearRoleTokens + handleLogout——一次超时+一次 401 就把用户登出，而本该走刷新。
- **建议**: 拆分标记：超时用 _timeoutRetried，401 用 _retry，互不干扰；或对已超时重试的请求在 401 时仍允许一次刷新。

## frontend/src/utils/apiCache.js

### [LOW] 缓存直接保存 axios 响应对象，消费方就地修改 response.data 会污染后续所有命中缓存者
- **位置**: 86-110  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: wrap 成功后将原始 response（含可变的 data 对象）set 进 cache，组件拿到缓存后若 push/sort 修改 data 数组，后续组件命中的是同一对象引用，读到被污染的数据；此外 `cached !== null`（第 89 行）把合法 null 值当成未命中反复请求。
- **建议**: 缓存前对 response 做一次浅拷贝（或缓存 data 而非整个 response）；用 `cached !== undefined` 判断命中。

### [LOW] generateKey 从未被调用（死代码）
- **位置**: 16-19  |  **类别**: maintainability  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: generateKey(url, params) 定义后全文件无任何调用点，wrap 一律使用调用方传入的显式 key，该方法可删除。
- **建议**: 删除 generateKey，或让 wrap 支持自动生成 key 以体现其价值。

## frontend/src/utils/confirmAction.js

### [INFO] 无明显问题（options 展开覆盖顺序正确，dangerouslyUseHTMLString 默认 false 防 XSS）
- **位置**: 11-22  |  **类别**: other  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: defaultDangerOptions 展开在前、...options 在最后，调用方可完全覆盖；默认关闭 dangerouslyUseHTMLString 避免 HTML 注入；confirmDelete/confirmReset/confirmClear 文案与默认值合理。第 14-19 行显式列出的字段与 ...options 有冗余，但不影响行为。
- **建议**: 可精简为 {...defaultDangerOptions, ...options} 一行，减少重复字段。

## frontend/src/utils/date.js

### [MEDIUM] 位置解析工具（parseLocation/formatLocation/getLocationTag）寄生在 date.js 里
- **位置**: 216-272, 187-215  |  **类别**: architecture  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 216-272 行是位置字符串解析/格式化，与日期毫无关系，却被塞进 date.js 并挂进默认导出对象（第 212-214 行）——日期模块承载了 GeoIP/登录历史展示逻辑，违反单一职责，调用方 import date 会隐式依赖位置函数。
- **建议**: 拆出 utils/location.js（或并入现有 GeoIP 工具），从 date.js 默认导出中移除这三个函数。

### [LOW] 时间字符串解析靠猜测：含 'T' 一律按 UTC 解析、空格分隔按上海本地解析，契约脆弱
- **位置**: 15-30  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: createShanghaiDayjs 对 '2024-05-01T10:00:00'（无 Z 无时区）走 `dayjs.utc(date)`（第 22-23 行）当作 UTC 再转上海（+8 小时）；若后端实际返回上海本地时间的 ISO 串，会被整体平移 8 小时显示错误。解析策略依赖后端约定但无显式契约。
- **建议**: 与后端约定统一时间格式（如一律 UTC ISO 带 Z，或一律 'YYYY-MM-DD HH:mm:ss' 本地时间），删除猜测分支，只保留一种解析路径并注释契约。

### [LOW] getRemainingDays 返回小数天数，消费方可能显示 '0.5天'
- **位置**: 63-69  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: `Math.max(0, diffDays)` 未取整，距到期 12 小时会返回 0.5，若组件直接展示会得到 '0.5天' 这类怪异文案。
- **建议**: 明确语义：向上取整（剩余不足一天算 1 天）或向下取整并在文档注明。

## frontend/src/utils/elementPlusServices.js

### [LOW] 服务层 shim 被 main.js 绕过，深路径导入分散两处
- **位置**: 1-3  |  **类别**: style  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 该文件统一 re-export ElMessage/ElMessageBox/ElNotification（解耦 utils 对 element-plus 深路径的依赖），但 main.js 第 3-6 行仍直接从 'element-plus/es/components/...' 深路径导入同一批组件，绕过了 shim，导入风格不一致且深路径在版本升级时易碎。
- **建议**: main.js 改为从 @/utils/elementPlusServices 导入，收敛所有深路径引用到一处。

## frontend/src/utils/format.js

### [LOW] formatPercent 的 ratio 参数使同一数值语义分裂（0.15 vs 15），调用方易传错
- **位置**: 17-24  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: `const normalized = ratio ? number * 100 : number` 意味着同一函数对 0.15 和 15 两种传参约定，调用方必须知道每个数据源是小数还是百分数，传错会输出 1500% 或 0.15%，属于隐式契约。
- **建议**: 统一约定（建议一律传小数），删除 ratio 开关；或在函数名区分 formatRatio/formatPercent 两个显式函数。

## frontend/src/utils/githubDownload.js

### [HIGH] 系统检测顺序错误：Android UA 含 'Linux' 被先匹配成 linux/x64，安卓用户永远走错下载分支
- **位置**: 133-143  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: detectSystem 判断顺序为 windows→macos→linux→android（第 133-143 行），而 Android UA（'Mozilla/5.0 (Linux; Android 13...)'）lowercase 后包含 'linux'，在第 137-138 行就被命中为 os='linux'、arch='x64'；第 348 行的 `if (os === 'android')` APK 分支成为不可达死代码，安卓手机最终尝试下载 Linux x64 二进制。
- **建议**: 把 android/ios 判断提到 linux 之前（或先检测 'android' 再检测 'linux'），并给 detectSystem 补单元测试覆盖 Android UA。

### [MEDIUM] 客户端仓库映射表被定义三份，极易漂移
- **位置**: 6-127, 336-343, 370-377  |  **类别**: duplication  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: CLIENT_CONFIGS（第 6-127 行）、getClientDownloadUrl 内部 clientMap（第 336-343 行）、getClientReleasesUrl 内部 clientMap（第 370-377 行）三处维护同一批 repo/name/configKey 映射，新增客户端需改三处，漏改即返回错误仓库链接。
- **建议**: 从 CLIENT_CONFIGS 派生 name/repo（getClientDownloadUrl 已通过 configKey 找到 config 就复用其 repo），getClientReleasesUrl 复用同一数据源，删除两份重复 map。

### [MEDIUM] Apple Silicon Mac 会被误判为 intel（UA 里恒有 'Intel Mac'，'Apple' 又匹配 AppleWebKit）
- **位置**: 152-164  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 154 行 `userAgent.includes('Intel') && !userAgent.includes('Apple')`：Apple 为兼容性在所有 M 系 Mac UA 中保留 'Intel Mac OS X'，因此几乎恒为 true 判成 intel；而 'Apple' 匹配的是 'AppleWebKit'（Intel 机也有），两个信号都不可靠，第 159-163 行 hardwareConcurrency 兜底分支基本不可达。
- **建议**: 用 navigator.userAgentData?.getHighEntropyValues(['architecture']) 或更可靠的 UA 特征判断，优先 Apple Silicon 关键词，并考虑让用户手动选择架构。

### [LOW] 浏览器直连 api.github.com 查 release（匿名 60 次/小时/IP 限流），面板多用户会集体 403
- **位置**: 291-292, 256-275  |  **类别**: performance  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 每次点击下载都从浏览器 fetch api.github.com/repos/.../releases/latest（第 291-292 行），未认证 GitHub API 限流 60 req/h/IP；同一面板大量用户访问教程页会集体触发限流，随后退化为跳 releases 页面（第 331 行）体验降级。此外下载走 /api/v1/download/resolve 后端代理，后端需对 target 做 SSRF 白名单校验（前端传入的 browser_download_url/代理 URL 不可全信）。
- **建议**: release 元数据改由后端代理缓存（服务端 token + 缓存 10 分钟）再下发前端，减少客户端直连 GitHub；后端 resolve 端点对 target 做协议/域名白名单。

## frontend/src/utils/qrcode.js

### [INFO] 无明显问题（动态 import 懒加载合理）
- **位置**: 1-30  |  **类别**: other  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: getQRCodeModule 动态 import 'qrcode' 并兼容 default/CJS 导出，空值抛错明确，createQRCodeDataURL/drawQRCodeToCanvas 职责单一。仅提示：调用方需自行 try/catch 这两个抛错函数。
- **建议**: 无需修改；可在调用处统一封装错误提示。

## frontend/src/utils/safeOpen.js

### [LOW] 协议相对 URL（//evil.com）可通过 safeOpenInternal 打开外部站点且只带 noopener 无 noreferrer
- **位置**: 17-46, 103-105  |  **类别**: security  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: normalizeSafeUrl 中 '//evil.com' 被判定为 relative（isRelative 为 true）且解析后协议为 http/https 放行，safeOpenInternal（第 103-105 行）以 'noopener' 打开——会跳转外部域并泄漏 referrer（无 noreferrer）；若调用方误把用户可控字符串当内部路径传入即构成开放跳转面。
- **建议**: safeOpenInternal 强制同源校验（比较 new URL(url, origin).origin === location.origin），不同源拒绝；内部跳转也补 noreferrer 或改用 router 跳转。

### [INFO] 协议白名单与 tabnabbing 防护整体正确
- **位置**: 63-83  |  **类别**: other  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: NAVIGATION_PROTOCOLS + APP_PROTOCOLS 白名单拒绝 javascript:/data: 等危险协议，window.open 带 noopener,noreferrer，relative 输出剥掉 origin；try/catch 兜底。仅提示：features 里 noopener 生效时返回的 newWindow 恒为 null，第 75-77 行的 opener 清理是死分支（无副作用）。
- **建议**: 可保留现状；如需用返回窗口做后续操作，改用不带 noopener 但手动置 opener=null 的方式。

## frontend/src/utils/sanitizeHtml.js

### [LOW] afterSanitizeAttributes 钩子在模块加载时全局注册，影响所有 DOMPurify 调用点（隐藏耦合）
- **位置**: 41-58  |  **类别**: architecture  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: import 本文件即给全局 DOMPurify 挂上 a/img 的强制钩子，项目中任何其他地方直接调用 DOMPurify.sanitize 都会静默继承此策略；若未来某处需要不同的 a 标签策略将直接冲突且难以排查。
- **建议**: 把钩子移入 sanitize() 内部按需注册/注销，或导出独立的 DOMPurify 实例（createDOMPurify）供本项目专用，避免污染全局。

### [LOW] 缓存键用 长度+前120字符 截断，长文命中率低且仅靠 original 比对纠错
- **位置**: 60-62  |  **类别**: performance  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: cacheKey 为 `${mode}:${html.length}:${html.slice(0,120)}`：两个前 120 字符相同、长度相同的不同 HTML（长文章常见）会碰撞，虽然第 68 行 `cached?.original === html` 校验避免了错误结果，但碰撞即缓存失效，等于没缓存长文，浪费内存与计算。
- **建议**: 改用完整内容的哈希（如简易 FNV 或直接以 html 为 key 并限制条目数），消除截断碰撞。

## frontend/src/utils/statusMaps.js

### [LOW] 14 个 getXxxText/getXxxType 样板包装函数可被泛化替代
- **位置**: 114-257  |  **类别**: style  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: getUserStatusText/Type、getSubscriptionStatusText/Type、getOrderStatusText/Type 等 14 个函数全部只是 `getStatusText(status, MAP)` 的一行转发（第 156-257 行），模板里直接 `getStatusConfig(status, MAP).text/.type` 或 `map[status]` 即可；包装函数层增加了无信息量的 API 面。
- **建议**: 保留 3 个通用函数（getStatusText/Type/Config）+ 导出映射表，删除逐状态包装函数；EMAIL_TYPE_MAP 的 'expiry_reminder'（第 77 行）标注为遗留键，确认无用后删除。

## frontend/src/utils/textSelection.js

### [MEDIUM] 模块顶层直接读 navigator.userAgent，非浏览器环境（SSR/测试/部分工具链）import 即崩溃
- **位置**: 3  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 第 3 行 `const isMobileDevice = /Android|.../.test(navigator.userAgent)` 在模块加载时执行，navigator 不存在时抛 ReferenceError，导致任何 import 该工具的文件在 SSR/Node 环境直接报错；第 29 行又重复一次 UA 正则（'iPhone|iPad|iPod'），检测逻辑散落。
- **建议**: 将检测推迟到函数调用时（typeof navigator === 'undefined' 短路），并把 UA 正则收敛为一个 isIOS 常量。

### [MEDIUM] MutationObserver 监听整个 body 子树，每次 DOM 变化都全文档扫描 .el-table td .cell
- **位置**: 53-73  |  **类别**: performance  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: observer 以 childList+subtree 观察 document.body，任何节点增删（通知、加载态等）都触发 500ms 防抖的全文档 querySelectorAll('.el-table td .cell')（第 54 行）；管理后台表格页 DOM 频繁变动时，该扫描可能成为持续开销；且第 75 行 popstate 监听与 observer 功能重叠（SPA pushState 导航本就不触发 popstate）。
- **建议**: 把 observer 限定到各表格容器（或在表格渲染处按需 init），扫描结果做缓存/增量；删除冗余的 popstate 监听。

### [LOW] 菜单项点击关闭后 document 级 close 监听器残留到下一次点击才被清理
- **位置**: 110-120  |  **类别**: logic  |  **来源组**: F1-frontend-base (前端基础 main/router/store/composables/utils)
- **问题**: 点击菜单项时 `ev.stopPropagation()` 阻止了 document 的 click 触发 close()（第 99 行），因此第 117-120 行挂的 document click/contextmenu 监听要等用户下一次点击任意位置才被 close() 移除；高频右键操作会积累临时监听器（虽最终收敛，属轻微泄漏）。
- **建议**: 菜单项点击时主动调用 close 逻辑（或统一走一个 closeMenu() 并把监听器注销与菜单移除绑定在同一处）。

## frontend/src/views/Dashboard.vue

### [HIGH] createRecharge 对 response.data.data 无空值防护，成功判定过松
- **位置**: 1158-1175  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `if (response.data && response.data.success !== false)` 之后直接 `const data = response.data.data`，随后 `data.payment_error`、`data.payment_url` 均在 data 为 null/undefined 时抛 TypeError。虽然被外层 catch 兜住，但用户只会看到误导性的“创建充值订单失败”，且 `success !== false` 会把 `{success: undefined}` 之类的异常响应也当作成功进入分支。
- **建议**: 改为严格判定：`if (response.data?.success === true && response.data.data)`，data 缺失时直接返回明确错误信息，避免依赖 catch 兜底。

### [MEDIUM] 四个 Clash 系列下拉与四个复制函数几乎逐行重复
- **位置**: 168-228, 1456-1486  |  **类别**: duplication  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 模板中 Clash/Flash/Clash Part/Clash Verge 四个 el-dropdown 块结构完全一致（仅按钮 class 与 command 前缀不同）；copyFlashSubscription/copyClashPartySubscription/copyClashVergeSubscription/copyShadowrocketSubscription 都只是复制同一个 clashUrl/universalUrl 换不同提示文案。
- **建议**: 将客户端配置抽象为数组（key/名称/图标/复制目标），用 v-for 渲染下拉，复制函数合并为一个 `copySubscription(target, message)`。

### [MEDIUM] openAlipayAppForRecharge 与 Packages.vue 的 openAlipayApp 完全重复
- **位置**: 1109-1138  |  **类别**: duplication  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 两处都构造 `alipays://platformapi/startapp?saId=10000007&qrcode=...` 并注册 visibilitychange/focus 手动检查监听器（约 30 行），仅回调函数名不同；此外与 Packages.vue 的 isYipay 判定（1177-1182 vs Packages 1294-1299）也重复。
- **建议**: 提取公共 composable `useAlipayJump(paymentUrlRef, onReturn)` 与工具函数 `isYipayMethod(method)`，两文件共用，消除跨文件复制。

### [MEDIUM] 设备数使用 || 而非 ??，在线设备数为 0 时被 currentDevices 覆盖
- **位置**: 133, 137, 371-372, 861-871  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 模板多处 `userInfo.online_devices || subscriptionInfo.currentDevices || 0`：当后端真实返回 online_devices=0（无在线设备）时，会落回 subscriptionInfo.currentDevices；而 loadUserInfo 又把 currentDevices 设为 `dashboardData.subscription?.currentDevices ?? 0`，两个字段语义可能不一致（一个实时在线数、一个订阅配额口径），导致 0 与 N 的展示错位。
- **建议**: 统一用 `??` 空值合并（区分 null 与 0），并指定单一权威字段（如订阅接口的 currentDevices），把设备数计算收敛为单个 computed。

### [MEDIUM] upgradeProgressStyle computed 与 .upgrade-progress CSS 整套未挂到模板
- **位置**: 904-913, 1783-1852  |  **类别**: maintainability  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `upgradeProgressStyle` 计算属性（904-913）在模板中没有任何 `:style` 绑定，模板里 level-card 只渲染了名称与折扣标签，`.upgrade-progress/.progress-bar/.progress-fill/.max-level-tip` 等约 70 行 CSS 无对应 DOM，属于被移除或从未接线的功能残留。
- **建议**: 要么在 level-card 内补上升级进度 UI 并绑定 upgradeProgressStyle，要么删除该 computed 与全部相关 CSS。

### [MEDIUM] 文件尾部 300 余行 !important 覆盖瀑布，且与前面规则重复冲突
- **位置**: 3425-3745  |  **类别**: style  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `.dashboard-container .stat-card`、`.page-header`、`.stats-grid` 等在 1598-2000 行已定义，3425 行之后又用 `!important` 全套重写一遍（还有重复的 @media 块 2903 与 3702）；`.qr-code img`(2775/2841)、`.qr-tip`(2791/2846)、`.qr-code-section`(2645/2752)、`.qr-code-container`(2652/2755) 各自定义了两次。说明本文件样式在与全局 user-client-polish.scss 打补丁式对抗。
- **建议**: 梳理全局样式与本文件职责边界，删除重复定义与 !important 覆盖，收敛为单一来源。

### [MEDIUM] 二维码生成失败时把裸 URL 塞进 img src，用户看到裂图
- **位置**: 1206-1213  |  **类别**: ux  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `catch (qrError) { rechargeQRCode.value = paymentUrl; ... }` 将支付 URL 直接赋给 rechargeQRCode，模板 `<img :src="rechargeQRCode">` 会渲染成破图，用户既扫不了码也读不到链接。
- **建议**: 失败分支应显示可复制的 URL 文本（或回退到后端返回的二维码图片 URL），而不是把文本 URL 当图片源。

### [LOW] 绕开 parsePaymentMethods 自造三分支响应解析
- **位置**: 699-719  |  **类别**: architecture  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: loadRechargePaymentMethods 手写 `success&&data` / 数组 / `data.data` 三形态解析，而 api.js 已导出共享工具 parsePaymentMethods（Packages.vue 1117 行正在使用），且这里用了裸 `api.get('/payment-methods/active')` 而非统一的 API 封装。
- **建议**: 改为 `rechargePaymentMethods.value = parsePaymentMethods(response)`，与 Packages.vue 保持一致。

### [LOW] 在 onMounted 内嵌套注册 onUnmounted 的反模式
- **位置**: 1584-1587  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 事件监听器的卸载逻辑写在了 onMounted 回调内部，若组件在 onMounted 回调执行前被卸载，卸载钩子不会被注册导致监听器泄漏；且本文件同时存在两个顶层 onUnmounted（1589、1594），结构混乱。
- **建议**: 把 addEventListener 与对应的 removeEventListener 都放到顶层生命周期中成对注册。

### [LOW] 多处未使用导入与死变量：CopyDocument/InfoFilled/Trophy/formatDateUtil/isExpiredUtil/sanitizeHtml、resizeRafId、旧轮询变量
- **位置**: 589-628, 697, 1225-1228, 1591-1594  |  **类别**: maintainability  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: grep 确认：`CopyDocument`(595)、`InfoFilled`(597)、`Trophy`(606)、`formatDateUtil`(617，被本地 formatDate 遮蔽)、`isExpiredUtil`(617)、`sanitizeHtml = sanitizeBasicHtml`(628) 均无任何使用；`resizeRafId`(697) 从未被赋值，onUnmounted 里的 cancelAnimationFrame 是死代码；`rechargeStatusInterval/rechargeVisibilityHandler/rechargeFocusHandler/rechargeStatusTimeoutId`(1225-1228) 是旧手写轮询的残留，现由 usePaymentStatusPolling 接管。
- **建议**: 删除这些导入与变量；resizeRafId 若无用途一并移除，避免维护者误以为存在动画帧逻辑。

### [LOW] 静默空 catch 与缩进错误
- **位置**: 678, 1310, 1556, 1081  |  **类别**: style  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: loadCheckinStatus(678)、loadSoftwareConfig(1310)、checkAndShowAnnouncement(1556) 的 catch 为空（无任何日志），排查问题时无法定位失败来源；`loadSubscriptionInfo` 的 `} else {`(1081) 缩进错位。
- **建议**: 空 catch 至少加 console.warn 或失败计数；修正缩进并统一错误处理风格。

## frontend/src/views/Devices.vue

### [HIGH] 客户端筛选只作用于当前服务端分页页，跨页搜索/筛选不可用且分页总数错乱
- **位置**: 390-428, 441-493, 526-528  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: fetchDevices 只按 page/size 请求当前页（subscriptionAPI.getDevices），filteredDevices 却对 devices.value（仅当前页）做 keyword/device_type/online_status 过滤，displayTotal = filteredDevices.length 变成'当前页过滤后数量'。设备总数超过一页时，搜索第 2 页才存在的设备永远搜不到；applyDeviceFilters 只重置 currentPage 不重新取数。
- **建议**: 把 keyword/device_type/online_status 作为查询参数传给后端（需后端在 /subscriptions/devices 支持过滤），或一次性拉全量设备再客户端分页（参照 Nodes.vue 模式）并保证过滤与分页同源。

### [MEDIUM] fetchDevices 无并发去重/顺序守卫，快速翻页时响应乱序覆盖
- **位置**: 441-493, 509-525  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: 挂载、两处刷新按钮、分页变更、removeDevice 成功回调、handleUpgradeSuccess 都会触发 fetchDevices，且没有类似 Orders.vue 的 orderLoadPromise 去重或请求序号守卫；慢响应后返回会覆盖新数据，出现'翻到第 3 页却显示第 1 页内容'的竞态。
- **建议**: 为 fetchDevices 增加 AbortController 或递增 requestId，仅接受最后一次请求的结果；对同一页请求做 in-flight 去重。

### [MEDIUM] getChartFillStyle 函数及整套 chart/summary CSS 无模板对应，约 200 行死代码
- **位置**: 643-651, 814-838, 990-1005, 1106-1157  |  **类别**: maintainability  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: getChartFillStyle（643-651）被定义并在 setup 返回（709 行）但模板从未引用；.chart-item/.chart-bar/.chart-fill/.chart-count/.chart-card/.chart-container/.summary-list/.summary-row/.button-row/.pagination.mobile-pagination 等样式（814-838、990-1005、1106-1157）均无对应 DOM，疑似从旧版仪表盘残留。
- **建议**: 删除 getChartFillStyle 返回项与上述未使用样式块；如需保留图表功能请补齐模板或迁移到仪表盘页。

### [LOW] 筛选激活时改变 pageSize/页码不触发取数，与无筛选时行为不一致
- **位置**: 509-521  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: handleSizeChange/handleCurrentChange 在 hasActiveFilters 时不调 fetchDevices，只改客户端切片；一旦用户清空筛选回到服务端分页，仍显示旧的 pageSize 数据（例如筛选时选了 100/页，清空后服务端仍按 10/页取数但客户端以为 100）。
- **建议**: 清空筛选（resetDeviceFilters）后显式 fetchDevices() 一次，保证 pageSize 与服务端同步。

### [LOW] isOnline 每行每次过滤都新建 dayjs 实例，now 可提至循环外
- **位置**: 632-642  |  **类别**: performance  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: filteredDevices 对每个设备调用 isOnline，内部每次执行 dayjs(lastAccess).tz('Asia/Shanghai') 和 dayjs().tz('Asia/Shanghai') 两次构造；大分页 + 输入关键词逐字符过滤时重复计算。
- **建议**: 把 now 提升为模块/组件级常量（每分钟刷新一次即可），isOnline 只解析 lastAccess。

### [LOW] 页头与卡片头两个'刷新'按钮重复；组件内 dayjs.extend 全局副作用；同模块重复 import
- **位置**: 9-14, 95-101, 318-319, 327-329  |  **类别**: style  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: 模板 9-14 行与 95-101 行是功能完全相同的刷新按钮；dayjs.extend(timezone)（327-329）在组件顶层执行全局插件注册，属隐式副作用；318-319 行对 '@/utils/date' 写了两条独立 import 语句（formatDateTime as formatTimeUtil 与 formatLocation）。
- **建议**: 去掉卡片头重复按钮或改为仅移动端显示；dayjs 插件统一在 main.js/独立 dayjs.js 中注册一次；合并两条 import 为一行。

### [INFO] removeDevice/updateDeviceRemark 直传设备 id，后端已做归属校验，无需前端加固
- **位置**: 570-588, 652-678  |  **类别**: security  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: DELETE /subscriptions/devices/:id 与 PUT .../remark 直接使用行内 id；核对后端 DeleteDevice（internal/api/handlers/device.go:123-141）通过 JOIN subscriptions 校验 user_id 归属，updateDeviceRemark 同理，IDOR 面在后端已关闭。前端继续按 id 直传即可。
- **建议**: 保持现状；建议在代码注释中说明依赖后端归属校验，避免后续重构时误删该校验。

## frontend/src/views/Help.vue

### [MEDIUM] 内容被净化两次：computed 已 sanitize，模板 v-html 又 sanitize 一遍；且输入本就是静态常量
- **位置**: 82, 99, 122, 684-695  |  **类别**: duplication  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: sanitizedGuides/sanitizedFaqs computed（684-695）已对 guide.content/faq.answer 执行 sanitizeHtml，模板 v-html="sanitizeHtml(guide.content)"（82/99/122 行）再次净化；由于 guides/faqs/clients.guide 全是文件内硬编码静态字符串，净化本身当前无实际收益（内容不可能含恶意输入），双份调用只是增加理解负担。
- **建议**: 模板直接 v-html="guide.content"（依赖 computed 已净化），删除模板内 sanitizeHtml 调用；若未来内容来自后端 CMS，再统一在数据层净化一次即可。

### [MEDIUM] 样式表三代布局叠加：同一选择器重复定义 2-3 次并大量使用 !important 覆盖，约 1130 行 CSS
- **位置**: 715-1845  |  **类别**: maintainability  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: .help-container 定义 3 次（716-721、722-731 及 1125-1128）、.page-header 2 次（737、1129）、.help-content 2 次（770、1655）、卡片选择器 .guide-card 等 3 次（817-842、1144-1161、1472-1492）、.client-actions 3 次（1066、1552、1742）、.client-row 2 次（1511、1682）、.contact-info 2 次（1076、1775），后段大量 !important（1477-1522、1663-1770）压前段规则，注释还标注'Final help center layout'暗示此前版本未清理。
- **建议**: 保留最后一代布局（1617 行起的 .help-layout/.help-client-grid/.client-row 体系），删除 715-1615 行中所有被覆盖的旧规则，把 !important 收敛到必要处；建议抽出 help.scss 并删除未使用的 .quick-grid/.nav-links/.client-item/.client-info 等残片。

### [LOW] flclash 的 downloadKeys 写作 'flash_windows_url'，疑似 'flclash_windows_url' 拼写错误
- **位置**: 450  |  **类别**: architecture  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: flclash 客户端 downloadKeys: ['flash_windows_url', 'flash_macos_url']（450 行），其余客户端键名均以自身 id 前缀命名（clash_verge_windows_url、hiddify_windows_url 等）；若后端 software-config 配置项实际命名为 flclash_windows_url，getConfiguredDownloadUrl 将永远匹配不到，静默回退到 GitHub 下载。
- **建议**: 与后端 software-config 的键名对齐（确认是 flash_ 还是 flclash_），并加一条 console.warn 在配置键全部缺失时提示，便于排查。

### [LOW] safeOpen 返回 null（URL 被协议白名单拦截）时仍提示'已打开下载页面'
- **位置**: 591-601  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: downloadClient 中 const configuredUrl = getConfiguredDownloadUrl(client); if (configuredUrl) { safeOpen(configuredUrl); ElMessage.success('已打开下载页面') } —— safeOpen 内部 normalizeSafeUrl 会拦截非 http(s)/mailto/tel/sms 协议并返回 null，但调用方不检查返回值，配置了 javascript: 等危险链接时会提示成功但什么都没发生。
- **建议**: 检查 safeOpen 返回值：为 null 时 ElMessage.warning('下载链接无效，已拦截')，并继续回退到 GitHub 下载分支。

### [LOW] Menu 图标导入未使用；每个 client 的 guideUrl 字段从未被读取
- **位置**: 171, 334-527  |  **类别**: maintainability  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: Menu 从 @element-plus/icons-vue 导入（171 行）但模板与脚本均无引用；baseClients 9 个客户端对象都定义了 guideUrl: '#'（346、366、388 等行），downloadClient/openClientGuide 均不读该字段，属于死数据。
- **建议**: 删除 Menu 导入；删除 guideUrl 字段（或在'教程'按钮未配置教程时用它做兜底跳转）。

### [LOW] loadContactInfo 动态 import('@/utils/api')，与文件顶部静态 import cachedAPI 风格不一致
- **位置**: 193-196  |  **类别**: style  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: 顶部已静态引入 cachedAPI（180 行），loadContactInfo 却用 await import('@/utils/api') 拿 default（195 行）；同模块两种引入方式并存，且该页面完全可以直接静态引入 api 实例。
- **建议**: 统一为静态 import { api, cachedAPI } from '@/utils/api'，删除运行时动态导入（还能减少一个异步边界）。

### [LOW] hash 深链 '#client-guide-xxx' 无法命中：normalizeClientId 只认裸客户端 id
- **位置**: 544, 569-574  |  **类别**: ux  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: applyRouteClientGuide 把 route.hash 去掉 '#' 后当 clientId 传给 openClientGuide，而 normalizeClientId 只匹配 'clash-windows' 这类裸 id（544 行）；用户分享 #client-guide-clash-windows 或 #clash-windows 时，前者被静默丢弃，后者能工作但与页面元素 id（client-guide-xxx）命名不一致，易混淆。
- **建议**: 统一 deep-link 约定：只支持 ?client= 查询参数，或在 hash 匹配前剥离 'client-guide-' 前缀。

### [INFO] v-html 走 DOMPurify 白名单 + URI 协议校验，当前无 XSS 暴露面（内容为静态常量）
- **位置**: 82-122  |  **类别**: security  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: sanitizeHtml 基于 DOMPurify，FORBID style/class/id，URI 仅允许 http(s)/mailto/tel/sms/cid（sanitizeHtml.js:3,41-58,71-79）；Help.vue 的 v-html 内容全部为文件内硬编码常量，唯一的动态数据 contactEmail/contactQQ 走 {{}} 插值自动转义。安全现状良好。
- **建议**: 保持现状；若未来 guide/faq 改由后端 CMS 下发，继续走 sanitizeBasicHtml 并在数据层缓存一次即可，不要在模板重复调用。

## frontend/src/views/Invites.vue

### [HIGH] "已购买人数"恒为 0、"最近邀请记录"恒为空：loadStats 硬编码占位
- **位置**: 424-455,29-35,154-207  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: loadStats 中 `purchased_invites: 0`、`recent_invites: []`（434、437、444、447 行）并注释"后端未提供此字段"。但 UI 上"已购买人数"统计卡（29-35 行）一直显示 0，"最近邀请记录"卡片（154-207 行）永远落到空状态（201-206 行）——对用户是误导性业务指标，对维护者是隐蔽的死功能。
- **建议**: 二选一：后端在 /invites/stats 补齐 purchased 统计与 recent_invites（关联订单/邀请关系表），前端去掉硬编码；或后端补齐前从模板移除这两块 UI（连同 stats.total_consumption 等未用字段）。

### [HIGH] 客户端可操纵邀请奖励金额：前端把 inviter_reward/invitee_reward 作为用户可改字段提交
- **位置**: 459-466  |  **类别**: security  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: generateCode 的 requestData 直接携带 `inviter_reward: Number(inviteRewardSettings.value.inviter_reward)`、`invitee_reward`、`min_order_amount: 0`、`new_user_only: true`（459-466 行）。后端 invite.go 只在值为 0 时才回退系统配置，非零值完全信任客户端（handlers/invite.go:39-101：`if req.InviterReward == 0 || req.InviteeReward == 0 { 回退配置 }`，否则 `InviterReward: req.InviterReward`），且奖励随后按此金额真实发放（handlers/auth.go:802-856）。恶意用户可绕过本页面直接 POST /invites 构造任意奖励金额的邀请码，配合自邀注册形成刷奖励/经济漏洞。
- **建议**: 前端彻底移除 reward_type/inviter_reward/invitee_reward/min_order_amount/new_user_only 字段（这些应由服务端按系统配置生成）；后端改为忽略客户端传入的奖励金额，一律从系统配置读取（后端侧修复是必要条件）。

### [MEDIUM] "留空表示无限制/永不过期"不可达：el-input-number 设了 min=1 无法清空
- **位置**: 228-253,459-471  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: max_uses 与 expires_days 的 el-input-number 均 `:min="1"`（229-231、244-246 行），输入框无法清空为 null；而表单提示与 placeholder 写"留空表示无限制/永不过期"（235-237、250-252 行），generateCode 里也确有 `Number(x) || 0`、`expires_days > 0` 才设 expires_at 的无限制分支（460、467-471 行）——该分支通过 UI 永远走不到。
- **建议**: 提供"不限"开关（如 checkbox 切换 null 值）或允许清空输入，使"无限制/永不过期"真正可选；否则删掉相关提示文案。

### [LOW] 金额格式化两套写法：桌面端内联 toFixed(2)，移动端用 formatMoney
- **位置**: 190,366  |  **类别**: duplication  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 桌面表格 `(scope.row.total_consumption || 0).toFixed(2)`（190 行）与移动端字段 `formatMoney(value)`（366 行）并存，前者对字符串金额/科学计数有边界问题，风格也不一致。
- **建议**: 统一改用 `formatMoney(scope.row.total_consumption)`（utils/format.js 已有 NaN/空值兜底）。

### [LOW] 三处空 `if (process.env.NODE_ENV === 'development') {}` 死分支
- **位置**: 385-387,473-474,488-489  |  **类别**: maintainability  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: loadInviteRewardSettings（385-387 行）、generateCode（473-474、488-489 行）内残留空的 development 分支，疑似删除日志时留下的空壳，无任何作用。
- **建议**: 直接删除这三个空 if 块。

## frontend/src/views/Knowledge.vue

### [MEDIUM] iconMap/resolveIcon 及 10 个未使用图标导入
- **位置**: 121,128-129  |  **类别**: maintainability  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `const iconMap = { Search, Folder, View, ... }` 与 `resolveIcon`（128-129 行）从未在模板/脚本中被调用；第 121 行导入的 Folder、Document、Reading、Files、Setting、Star、InfoFilled、QuestionFilled、Notebook 全部未使用（模板直接写死 Search/View/Clock）。
- **建议**: 删除 iconMap/resolveIcon 与未使用导入，仅保留模板实际用到的 Search/View/Clock。

### [MEDIUM] 文章列表无分页，全量拉取并渲染
- **位置**: 166-179  |  **类别**: performance  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: loadArticles 只传 category_id/keyword（169-171 行），无 page/size 参数，模板一次性 v-for 全部渲染（64-85 行）；知识库文章较多时首屏负载与 DOM 量都会膨胀，且知识库 API（knowledgeAPI.getArticles 支持 params）已具备扩展分页的空间。
- **建议**: 后端 /knowledge/articles 支持分页，前端配合 PaginationBar 或"加载更多"；或至少按分类限制返回条数。

### [LOW] article.category.name 假定内嵌对象，无归一化兜底
- **位置**: 73-75,99-101  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 模板直接访问 `article.category.name`（73-75、99-101 行），若后端列表接口返回 category_id 或字符串，`article.category` 为真但 `.name` 为 undefined，会渲染空 el-tag；openArticle 出错时回退列表项（197-199 行），抽屉里同样缺 content。
- **建议**: 在 loadArticles/getArticle 结果上归一化 `article.category = { id, name }`（id 与 name 都兜底），并给 category.name 加 `|| ''`。

### [INFO] XSS 防护到位：v-html 内容经 DOMPurify + URI 白名单清洗
- **位置**: 112,155  |  **类别**: security  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `<div class="article-content" v-html="sanitizeContent(currentArticle?.content)">` 使用 utils/sanitizeHtml.js 的 sanitizeArticleHtml：DOMPurify 配置 FORBID_TAGS(script/style/iframe/object/embed/form/input/button)、FORBID_ATTR(style/class/id)、ALLOWED_URI_REGEXP 仅放行 http(s)/mailto/tel/cid，且对 a[target=_blank] 强制 rel=noopener noreferrer、img 仅允许白名单协议（sanitizeHtml.js:41-58,71-79）。此路径无 XSS 风险。
- **建议**: 无需修改；仅提示后续新增富文本入口（如评论/工单回复）必须复用同一 sanitize 工具链。

## frontend/src/views/LoginHistory.vue

### [MEDIUM] fetchLoginHistory 响应解析含死分支与冗余判断
- **位置**: 273-318  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `else if (response.data.data && Array.isArray(response.data.data))`（282-283 行）与上一分支 `Array.isArray(response.data.data)`（280-281 行）条件完全重复，永不独立命中；`else if (response && Array.isArray(response))`（288-289 行）——axios response 不可能是 Array，恒为死代码；最后的 `data = response.data.data` 兜底又把对象赋给 data 后再走 `data.logins` 分支，逻辑绕。
- **建议**: 收敛为：`const payload = response?.data; const list = payload?.success !== false ? payload?.data : null; if (Array.isArray(list)) ... else if (list?.logins) ...`，删除死分支。

### [MEDIUM] 全量拉取登录历史后客户端过滤/分页，无服务端分页上限
- **位置**: 238-272  |  **类别**: performance  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `userAPI.getLoginHistory()` 不带任何分页参数（276 行），过滤（238-267 行）与分页（268-272 行）全部在内存完成；账户登录记录累积较多时每次进入页面都拉全量，且该接口被 Profile.vue 弹窗重复使用。
- **建议**: 后端 /users/login-history 支持 page/size/status/date 参数，前端改为服务端分页（PaginationBar 已具备 v-model 契约）；或至少限制返回最近 N 条并提示。

### [LOW] formatTime 无有效性校验、lastLoginDays 依赖排序假设
- **位置**: 319-322,364-369  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `formatTime` 直接 `dayjs(time).format(...)`（319-322 行），脏数据会渲染 'Invalid Date'（Profile.vue 的副本有 isValid 兜底，见该文件 553-569 行，两份已分叉）；`lastLoginDays` 取 `loginHistory.value[0]` 当作最近一次登录（364-369 行），若后端未按时间倒序返回则统计错误。
- **建议**: formatTime 加 `dayjs(time).isValid()` 校验；lastLoginDays 改为取所有记录的最大登录时间，或明确依赖/校验后端倒序契约。

### [LOW] total ref 与 Clock 图标未使用
- **位置**: 231,302-309,198,211  |  **类别**: maintainability  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `total` ref（231 行）在 fetch 中赋值（302-309 行）但模板从未展示（PaginationBar 用的是 filteredLoginHistory.length，187 行）；`Clock` 图标导入并注册（198、211 行）模板未使用（统计卡是文字 L/IP/C/D）。
- **建议**: 删除 total 与 Clock 相关代码。

## frontend/src/views/Nodes.vue

### [MEDIUM] 后端 /nodes/ 已支持 region/type/status 查询过滤，前端却全量拉取后客户端过滤
- **位置**: 228-261, 325-371  |  **类别**: architecture  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: nodeAPI.getNodes() 无任何参数，一次拉取全部可用节点；而后端 GetNodes（internal/api/handlers/node.go:216-220）本身就支持 region/type/status 查询参数。当前实现与后端能力重复，节点数量大时（含专线节点、去重逻辑）payload 膨胀，区域/类型下拉还依赖已拉取数据首屏才有值。
- **建议**: 直接复用后端过滤：把 filterRegion/filterType/filterStatus 作为查询参数传给 getNodes，删除客户端过滤逻辑；下拉选项改由静态配置或 /nodes/stats 提供。

### [MEDIUM] nodeStats/updateNodeStats 计算并返回但模板从未渲染；testing 字段注入未读；约 150 行统计/详情 CSS 无对应 DOM
- **位置**: 222-227, 332-353, 372-380, 481-530, 586-618, 639-669  |  **类别**: maintainability  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: nodeStats（222-227）被 updateNodeStats 更新并 return（450 行），但模板没有 stats-row 标记；fetchNodes 给每个节点注入 testing:false（332-353）从未被读取；.stats-row/.stat-card/.stat-icon/.stat-value/.stats-card/.speed-status-*/.status-item/.node-detail/.detail-item/.recommended-tag 等样式（481-530、586-618、639-669）无模板引用，疑似从旧版节点页残片。
- **建议**: 要么在模板补上统计卡片（total/online/regions/types）使 nodeStats 生效，要么删除 nodeStats、updateNodeStats、testing 注入及全部死样式，二选一避免误导维护者。

### [LOW] 'collect' 筛选把 is_manual === undefined 的节点也当作采集节点
- **位置**: 248-254  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: filterSource==='collect' 时条件为 node.is_manual === false || node.is_manual === undefined（252 行）；后端 Node 模型固定序列化 is_manual（node.go:21 json:"is_manual"），undefined 分支实际是死代码，且掩盖了数据缺失问题。
- **建议**: 收紧为 node.is_manual === false，缺失字段显式视为未知并给出提示，避免静默归类。

### [LOW] NODE_COLUMN_KEYS 含 'source' 并配置了列宽，但表格没有该列
- **位置**: 200-211  |  **类别**: maintainability  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: NODE_COLUMN_KEYS = ['name','region','type','source','status']（200 行）且默认列宽有 source:100（207 行），而 el-table 只有 name/region/type/status/说明 五列，无 source 列；'来源筛选'用的是独立下拉而非表格列，列宽配置纯属死配置。
- **建议**: 从 NODE_COLUMN_KEYS 与默认列宽中移除 'source'，避免 usePersistentTableColumns 把无意义的列宽写进 localStorage。

### [LOW] usePersistentTableColumns 默认列宽对象缩进错乱
- **位置**: 204-209  |  **类别**: style  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: 204-209 行对象属性（name/region/type/source/status）相对外层缩进不一致（4 空格 vs 6 空格），与该文件其余 2 空格风格冲突。
- **建议**: 统一缩进并交给 prettier 处理。

### [LOW] 关键词需手动点'搜索'才生效，而下拉选择自动重置页码，交互不一致
- **位置**: 290-312  |  **类别**: ux  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: watch([filterRegion, filterType, filterSource]) 自动 pagination.page=1（290-292），但 filterKeyword 只在 @keyup.enter/@clear 或点'搜索'按钮时经 applyFilters 重置页码；用户输入关键词后再改下拉，页码被重置但关键词仍处于未提交状态，结果与直觉不符。
- **建议**: 把 filterKeyword 也纳入 watch 自动生效（配合防抖），或统一改为显式'搜索'提交模式（下拉同样不自动触发）。

## frontend/src/views/NotFound.vue

### [INFO] 无明显问题（$router.go(-1) 的边界行为提示）
- **位置**: 13  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 页面简洁无逻辑缺陷。唯一边界：`$router.go(-1)`（13 行）当应用内无历史记录（用户直接键入 404 URL 打开）时可能退出 SPA 回到浏览器外部历史页，体验略突兀。
- **建议**: 可改用 `router.back()` 前判断 `window.history.length > 1`，否则回 /dashboard；属可选优化。

## frontend/src/views/Orders.vue

### [HIGH] 充值 tab 分页完全失效：翻页/改页大小总是加载订单，且分页 total 恒为订单总数
- **位置**: 805-813, 495-528, 656-674  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: handleCurrentChange/handleSizeChange 无条件调用 loadOrders()，在 activeTab==='recharges' 时点第 2 页实际重新拉取订单、充值列表停留在第 1 页。同时 loadRecharges() 从不更新 pagination.total——后端 GET /recharge 的 total 只放在 X-Total-Count 响应头（internal/api/handlers/recharge.go:254），前端从未读取该头，因此充值 tab 的 PaginationBar 显示的是订单总数（onMounted 中 loadOrders 先写入）。
- **建议**: 按 activeTab 分支：'recharges' 时翻页调用 loadRecharges() 并读取 response.headers['x-total-count']（或让后端把 total 放进 body）；'all' 时同时重载两边并重新 mergeRecords。

### [HIGH] 订单关键字搜索是假搜索：keyword 参数被后端 GetOrders 完全忽略
- **位置**: 685-691  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: loadOrdersInternal 把 filters.keyword 放进 params 发送，但后端 GetOrders（internal/api/handlers/order.go:855-940）只读取 status/payment_method/start_date/end_date，从不读 keyword；loadRecharges 则根本不带 keyword。搜索框输入'订单号、套餐、类型'后列表不会有任何变化，属于误导性 UI。
- **建议**: 二选一：在后端 GetOrders/GetRechargeRecords 增加 keyword 过滤（LIKE order_no/package_name），或在搜索结果为空时明确提示'前端暂不支持搜索'并移除该输入框。

### [MEDIUM] GORM NullString 支付方式解包逻辑在 4 处复制粘贴
- **位置**: 540-551, 591-618, 828-841, 1303-1317  |  **类别**: duplication  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: mergeRecords、formatRechargeRecord、normalizePaymentMethodValue、getPaymentMethodText 各自实现了一遍'payment_method 是对象时取 .String/.payment_method/首个字符串值'的兼容逻辑，行为细节略有出入（缺省值 alipay vs ''），后续改一处漏四处。
- **建议**: 在 utils/format.js 或 utils/api.js 提取 export function normalizePaymentMethod(method, fallback='')，4 处统一调用并单测覆盖 NullString/纯字符串/空值三种形态。

### [MEDIUM] 支付状态监听双轨重复：usePaymentStatusPolling 自带 visibility/focus 监听，openAlipayApp 又手写一套
- **位置**: 915-921, 986-1015  |  **类别**: duplication  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: usePaymentStatusPolling（composables/usePaymentStatusPolling.js:47-58）已注册 visibilitychange/focus 回调，openAlipayApp 又自行 addEventListener 注册 paymentManualVisibilityHandler/paymentManualFocusHandler 并手动清理（902-914），两套监听触发同一 checkPaymentStatus，仅靠 paymentStatusRequest 去重掩盖；2s 间隔 × 30min 超时最多产生 900 次状态请求。
- **建议**: 删掉手写监听，仅保留 composable 的 startPolling（其 onCleanup 回调已足够）；如确需跳转支付宝后的即时检测，只在 composable 的 visibilityHandler 里加一次立即 poll。

### [MEDIUM] 支付状态轮询 catch 为空块，网络错误与响应结构异常被静默吞掉
- **位置**: 1091-1172  |  **类别**: error-handling  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: checkPaymentStatus 内 rechargeData = response.data.data（1100 行）与 orderData = response.data.data（1129 行）均无空值守卫，若接口返回错误结构或网络中断，TypeError 直接落入 `catch (error) { }`（1166-1167 空块），用户 2 秒一次轮询却看不到任何失败反馈，轮询继续空转直到 30 分钟超时。
- **建议**: catch 中至少记录日志并在连续 N 次失败后停止轮询 + ElMessage.error；对 response.data.data 做可选链判空后 return。

### [MEDIUM] '全部记录' tab 下筛选只作用于订单，充值记录无筛选且按页码错位合并
- **位置**: 495-528, 529-569, 682-719  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: loadRecharges 只传 start_date/end_date，status/payment_method/keyword 不生效；'all' tab 中 mergeRecords 把第 N 页订单与第 N 页充值拼在一起，但两者 total 不同源（订单 total 在 body，充值 total 在 header），导致合并列表与分页总数自相矛盾，且筛选结果中混入未筛选的充值记录。
- **建议**: 统一筛选语义：充值记录也透传 status/payment_method/keyword；'all' tab 建议改为两个独立列表或由后端提供合并接口（后端已有 admin 合并查询 buildAdminMergedOrderWhere，可参考做成用户版）。

### [MEDIUM] mergeRecords 排序依赖 new Date('YYYY-MM-DD HH:mm:ss')，Safari 下返回 Invalid Date 导致排序失效
- **位置**: 563-567  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: created_at/paid_at 是后端格式化后的北京时区字符串（如 '2025-01-01 10:00:00'），new Date() 解析该格式在 Safari/iOS 返回 Invalid Date → getTime() 为 NaN → Array.sort 比较器返回 NaN 按 0 处理，合并顺序变得不确定；订单与充值两个分页的先后合并还会造成排序抖动。
- **建议**: 用 dayjs(created_at, 'YYYY-MM-DD HH:mm:ss') 或把 ' ' 替换为 'T' 后再 new Date()，并统一使用同一时间函数解析两端记录。

### [LOW] 模板内 Date.now() 缓存戳在每次渲染时重新求值
- **位置**: 365  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: :src="paymentQRCode.startsWith('data:') ? paymentQRCode : (paymentQRCode + '?t=' + Date.now())" —— 当 paymentQRCode 是非 data: 的远程 URL 时，任何响应式更新（isCheckingPayment 每 2 秒轮询翻转）都会生成新 src 导致二维码图片反复重新加载；当前 generateQRCode 恒返回 data: URL 使该分支实际是死代码，但很脆弱。
- **建议**: 在生成二维码时一次性附加时间戳存到 paymentQRCode，模板只读变量，不要在模板里调用 Date.now()。

### [LOW] filters.date_range 数组残留进请求参数
- **位置**: 685-696  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: params = { page, size, ...filters } 后仅 delete keyword，date_range（[start, end] 数组）未删除，axios 会序列化成 date_range[]=2025-01-01&date_range[]=2025-02-01 的垃圾参数，后端忽略但污染请求，且空数组清空筛选后仍带 date_range[]=。
- **建议**: 解构时排除 date_range：const { date_range, ...rest } = filters，再按需补 start_date/end_date。

### [LOW] viewOrderDetail 的 try/catch 永不触发；cancelOrder 错误详情被丢弃
- **位置**: 1180-1195, 1254-1278  |  **类别**: logic  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: viewOrderDetail 内部只有提前 return 和赋值，不会抛异常，catch 分支（'查看订单详情失败'）不可达；cancelOrder 的 catch 固定提示'取消订单失败'（1193 行），未透出 error.response 中的具体原因，排障困难。
- **建议**: 删除 viewOrderDetail 的无用 try/catch；cancelOrder 复用 payOrder 的三级取错模式 error.response?.data?.detail || message。

### [LOW] createPaymentQRCode 与 generateQRCode 完全重复（前者未被调用）；onImageLoad 空实现；getPaymentMethodName 未使用
- **位置**: 843-853, 922-936, 1070-1071, 1062-1069  |  **类别**: maintainability  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: createPaymentQRCode（843-853）与 generateQRCode（922-936）参数、配置、实现逐行相同，前者从未被调用；onImageLoad 是空函数却仍绑定在 <img @load> 上（367 行）；getPaymentMethodName（1062-1069）定义了并返回（1429 行）但模板中无任何引用。
- **建议**: 删除 createPaymentQRCode、getPaymentMethodName 与 onImageLoad（或让 onImageLoad 承担'轮询已开始'的提示职责）。

### [LOW] GET /orders/ 带尾斜杠，每次列表请求多一次 301 重定向
- **位置**: 697  |  **类别**: performance  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: 前端调用 api.get('/orders/')，而后端路由注册为 orders.GET("")（router.go:140），Gin 默认 RedirectTrailingSlash 对 /api/v1/orders/ 返回 301 → 跳 /api/v1/orders，axios 跟随重定向导致每个列表请求双倍往返。
- **建议**: 前端改为 '/orders'（api.js 中 getUserOrders 同改），或后端补注册 GET "/" 别名。

### [LOW] order_no 直接拼接进 URL path 未做 encodeURIComponent
- **位置**: 956, 1125, 1186  |  **类别**: security  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: api.post(`/orders/${orderNo}/pay`)、api.get(`/orders/${selectedOrder.value.order_no}/status`)、api.post(`/orders/${order.order_no}/cancel`) 将订单号裸拼进路径。订单号当前由后端生成（字母数字），风险低，但一旦订单号可含特殊字符（如第三方支付回调生成的单号）即可能造成路径解析错误或注入。
- **建议**: 统一用 encodeURIComponent(orderNo) 包裹路径参数，或改用查询参数传递。

### [LOW] 同一函数内缩进混乱（2/4/6 空格混用），finally/catch 块对齐错误
- **位置**: 720-724, 1166-1170, 1251, 1275  |  **类别**: style  |  **来源组**: F5-user-views-mid (用户端 Orders/Devices/Nodes/Help)
- **问题**: 720-724 行 `} finally {` 与 722 行 `isLoading.value = false` 缩进层级错误；1166-1170、1251、1275 行类似，formatOrderRecord/checkPaymentStatus 等函数体出现 8 空格深层缩进，影响可读性。
- **建议**: 统一为 2 空格缩进，用 prettier 格式化该文件（建议项目加入 eslint + prettier 提交钩子）。

## frontend/src/views/Packages.vue

### [HIGH] 关闭扫码对话框不停止轮询：currentOrder 未清导致 3 秒一次的空转请求
- **位置**: 486, 1582-1588, 1602-1652  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 支付二维码弹窗取消只做 `paymentQRVisible = false`，currentOrder 仍持有 order_no；usePaymentStatusPolling 的 shouldPoll 为 `!!currentOrder.value?.order_no`，因此对话框关闭后仍每 3 秒轮询 `/orders/{no}/status` 直到 30 分钟超时，若订单其后被标记 paid 还会在用户干别的事时突然弹出成功框。
- **建议**: 取消/关闭对话框时调用 closePaymentStatusWatchers() 并清空 currentOrder（或把 shouldPoll 与 paymentQRVisible 绑定）。

### [MEDIUM] 余额支付请求契约不一致：常规订单传 use_balance，自定义订单只传 payment_method=balance
- **位置**: 1180-1187, 1224, 1232-1234  |  **类别**: architecture  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 常规流程 POST /orders/ 传 `use_balance: true`；自定义订单支付分支 POST /orders/{no}/pay 只传 `payment_method: 'balance'`（且未补 use_balance 字段）。若后端 pay 接口依赖 use_balance 字段区分余额抵扣，自定义套餐的余额支付会失败或行为不一致。
- **建议**: 统一两个流程的余额支付载荷（都传 use_balance 或都由后端按 payment_method 判断），并抽公共函数组装 payData。

### [MEDIUM] paymentStatusTimeoutId 从不清理：重复支付或组件卸载后旧定时器仍触发
- **位置**: 698, 1597-1600, 1719-1730  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: markOrderPaid 每次 `paymentStatusTimeoutId = setTimeout(...)` 覆盖旧 id 而不 clearTimeout；onUnmounted 也没有清理。若短时间内连续两次 markOrderPaid（如轮询+手动检查并发命中 paid），先前的定时器仍会在 3 秒后把新弹窗关闭并触发一次 loadPackages 请求（可能发生在组件已卸载后）。
- **建议**: 赋值前 `clearTimeout(paymentStatusTimeoutId)`，并在 onUnmounted 中一并清理。

### [MEDIUM] 页面跳转型支付（微信内/支付宝页面支付）与轮询生命周期冲突
- **位置**: 1309-1313, 1492-1497  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: safeNavigate 实现为 `window.location.href = url`（safeOpen.js 122 行），会整体离开 SPA：微信分支(1309-1313)跳转前未 startPaymentStatusCheck，支付宝页面支付分支(1492-1497)先启动轮询再跳转——轮询随页面销毁；用户返回后组件全新挂载、currentOrder 为 null，不会恢复轮询，只能依赖 pendingPaymentStorage 的全局恢复机制（未见）。
- **建议**: 统一约定：页面跳转型支付不依赖组件内轮询，由全局 pending-payment 恢复逻辑兜底；并在跳转前把订单写入 pendingPaymentStorage（当前微信分支路径未走 setCurrentOrderForPayment）。

### [MEDIUM] isPaymentPageUrl 判定含不可达分支，且与 showPaymentQRCode 内两套 URL 分类规则不一致
- **位置**: 690-696, 1484-1490  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `url.startsWith('http') && !includes('qrcode') && !startsWith('weixin://') && !startsWith('wxp://')` 中 weixin/wxp 分支永远不可达（它们不以 http 开头）；且任何不含 'qrcode' 的 http 链接都会被塞进 iframe，第三方站点常因 X-Frame-Options 拒绝嵌套而显示白屏。同时 showPaymentQRCode 用另一套 isAlipayPagePay/isYipayPaymentPage 规则，两套规则可漂移。
- **建议**: 收敛为单一 URL 分类函数（显式匹配 alipay 页面支付/易支付 submit.php/payapi 路径），非白名单一律生成二维码而非 iframe。

### [MEDIUM] loadPackages 用客户端拼装的通用文案覆盖后端 features，且 is_popular 混入魔法数字
- **位置**: 1049-1059  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `packages.value = packagesList.map(pkg => ({ ...pkg, features: [四条写死的通用文案] }))` 直接丢弃后端返回的 pkg.features 真实内容；`is_popular: ... || pkg.sort_order === 2` 把排序号 2 硬编码为“热门”，后端调整排序即导致标签错乱。
- **建议**: 优先透传后端 features，为空才兜底默认文案；is_popular/is_recommended 统一由后端布尔字段驱动，删除 sort_order 魔法判断。

### [MEDIUM] loadUserBalance 的 force 参数名不副实：走 cachedAPI 仍是 5 分钟缓存
- **位置**: 1082-1113  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `force ? cachedAPI.getUserInfo() : userAPI.getUserInfo()` —— cachedAPI.getUserInfo 在 api.js 785-789 是 apiCache.wrap 的 5 分钟缓存，force 只是换了个 API 对象，并不会绕过缓存；handleUserInfoUpdate/handleSubscriptionUpdate(1696-1702) 用 force:true 期望拿最新余额，实际可能读到陈旧数据（事件方未先 clearUserCache）。
- **建议**: force 分支直接调 `cachedAPI.clearUserCache()` 后再请求，或改调 userAPI.getUserInfo() 原语接口。

### [MEDIUM] iframe 内每 2 秒读 contentWindow.location，跨域页面必然抛错空转，且与 3 秒轮询双份叠加
- **位置**: 1526-1571  |  **类别**: performance  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: onIframeLoad 启动 2s 间隔的 setInterval 读 `iframe.contentWindow.location.href`——易支付/支付宝页面为跨域，读取必抛 SecurityError 被 catch 吞掉，10 秒内做 5 次无效尝试；同时 1570 行又无条件 startPaymentStatusCheck()，与 showPaymentQRCode 1518 行已启动的轮询叠加（虽 startPolling 会重置，但逻辑上双份启动冗余）。
- **建议**: 仅对同源页面启用 interval（比较 iframe.src 的 origin），或删除 interval 只保留 startPaymentStatusCheck，避免双重轮询。

### [LOW] Options API + setup() 巨型组件，与 Dashboard.vue 的 <script setup> 风格不一致
- **位置**: 642-674, 1900-1972  |  **类别**: architecture  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 本文件用 export default + setup() 返回约 60 个绑定，而 Dashboard.vue 用 <script setup>；组件总长 3517 行，同时承载套餐展示、两级优惠券、支付二维码/iframe、折算流程、自定义套餐五块业务，职责过重。
- **建议**: 统一改为 <script setup>；把支付状态机（二维码/iframe/轮询）与自定义套餐购买拆成独立组件或 composable。

### [LOW] getPaymentMethodDisplayName 分支重复且未知方法回退“支付宝”误导
- **位置**: 841-856  |  **类别**: duplication  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: alipay 在第二个与第五个分支重复匹配、wxpay/wechat 同理；完全未知的支付方式最终返回“支付宝”，用户看到的名称可能与实际支付渠道不符。
- **建议**: 改为显式 key 映射表（wxpay→微信、alipay→支付宝、qqpay→QQ钱包）+ 未知回退 `method 原文`。

### [LOW] checkPaymentStatus 对 404/终态订单静默吞错，轮询继续空转到超时
- **位置**: 1602-1652  |  **类别**: error-handling  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: catch 里仅 dev 环境 console.error，未像 Dashboard.vue 1293-1295 那样对 404 做终态处理；订单被删除或服务端不识别时，轮询将一直空转到 30 分钟上限。
- **建议**: 在 catch 中判断 `error.response?.status === 404` 时调用 closePaymentStatusWatchers 并提示订单失效。

### [LOW] 空实现处理函数与未使用 computed：handleCouponFocus/handlePaymentMethodChange/onImageLoad/onPaymentSuccess 等
- **位置**: 787-788, 1125-1126, 1653-1654, 1686-1691, 955-958  |  **类别**: maintainability  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: handleCouponFocus(787)、handlePaymentMethodChange(1125)、onImageLoad(1653) 为空函数却绑定在模板事件上；onPaymentSuccess/onPaymentCancel/onPaymentError(1686-1691) 空实现且模板从未引用，仅随 return 对象暴露；totalDurationDays computed(955-958) 定义并 return(1971) 但模板从未使用。
- **建议**: 删除这些空函数与 totalDurationDays（含 return 项），保留模板真实需要的事件处理。

## frontend/src/views/PaymentReturn.vue

### [MEDIUM] 轮询循环不随组件卸载取消，可劫持用户后续导航
- **位置**: 157-188,229-234  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: pollOrderStatus 最长约 35s（15 次 × 2s）的 for 循环没有任何取消标志；onUnmounted（229-234 行）只清 redirectTimer。用户在这期间离开页面（如手动跳转），循环仍在跑，成功后执行 handlePaymentSuccess → `router.push('/orders')`（147 行）把用户从当前位置强行带走，且对已卸载组件的 ref 赋值/ElMessage 无意义。
- **建议**: 引入 `let cancelled = false`，onUnmounted 置 true；循环每轮检查 `if (cancelled) return`，成功回调前同样检查。

### [MEDIUM] 订单号直接取自 URL query 并轮询其状态，无归属校验（IDOR 面）
- **位置**: 99-156  |  **类别**: security  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `extractOrderNoFromUrl`（99-104 行）信任 out_trade_no/order_no query，`fetchOrderData` 用它对 `/orders/${no}/status` 轮询（149-156 行），仅当 query 缺失时才回退 pendingPaymentStorage/最近订单。若后端该状态接口未按当前用户隔离订单归属，攻击者可构造含他人订单号的返回 URL 探测支付状态与金额；前端也无从确认该订单属于自己。
- **建议**: 轮询前先用自己最近订单（fetchRecentOrderNo 结果）校验 query 订单号归属，不匹配则丢弃并回退到自己的 pending 订单；后端侧务必在 status 接口按 user_id 强制过滤（后端修复为主）。

### [LOW] 金额 NaN 时会渲染 "¥NaN"
- **位置**: 32,135,164  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `parseFloat(data.amount || 0)`（135、164 行）对非数字串得 NaN，模板 `¥{{ amount }}`（32 行）直接输出 ¥NaN；失败视图也没有重置 amount。
- **建议**: 金额统一经 `formatMoney`（utils/format.js 已兜底 NaN/空）或在 parseFloat 后 `Number.isFinite()` 校验。

### [LOW] 订单类型靠单号前缀（RCH/UPG）推断，custom_package 无前缀永远不可达
- **位置**: 94-98  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: getOrderType（94-98 行）只识别 RCH→recharge、UPG→device_upgrade，其余一律 order；normalizeOrderType 虽接受 custom_package（80-84 行），但该类型只能由 pendingPaymentStorage.type 带入，URL 直达时必然错标——前缀约定是隐式契约，后端改前缀即静默错乱。
- **建议**: 类型应以订单状态接口返回的 type 字段为准，前缀推断仅作兜底；并把前缀常量集中定义并加注释说明与后端约定。

### [LOW] 静态与动态导入混用
- **位置**: 64,71,108,124  |  **类别**: style  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `import { useApi, pendingPaymentStorage } from '@/utils/api'`（64 行）静态导入，却又在 fetchRecentOrderNo（108 行）与 refreshUserState（124 行）里 `await import('@/utils/api')` 动态导入 orderAPI/cachedAPI，同一模块两种拿法。
- **建议**: 统一顶部静态导入 orderAPI/cachedAPI，删除动态 import。

## frontend/src/views/Profile.vue

### [MEDIUM] 同一份用户数据映射逻辑复制了 4 遍
- **位置**: 299-311,377-406  |  **类别**: duplication  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: initUserInfo（299-311）、fetchUserInfo 成功分支（377-380）、响应不可解析回退 authStore（382-391）、catch 回退 authStore（397-406）四处几乎逐字重复 `userInfo.value = {...}; profileForm.xxx = ...` 的赋值块。任何字段变更要同步改 4 处，是典型复制粘贴维护坑。
- **建议**: 抽取 `applyUserInfo(data)` 统一完成 userInfo/profileForm 映射与字段兜底（含 avatar/avatar_url、last_login/lastLogin/last_login_time 别名归一），四个分支只调用它。

### [MEDIUM] 登录历史弹窗整块重复实现 LoginHistory.vue 页面，且两份实现行为已分叉
- **位置**: 145-243,526-552  |  **类别**: duplication  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: Profile 的登录历史弹窗（模板 145-243 + getDeviceInfo/getLocationText/formatTime 526-552）与独立页面 LoginHistory.vue（模板 90-191 + 323-336/370-383）逻辑重复。两份已出现分叉：Profile 的 getLocationText 对未知 IP 返回静态"解析中..."（549 行），LoginHistory 返回''或"本地"/"内网"（375-382 行）；Profile 的 formatTime 有 dayjs isValid 校验（553-569），LoginHistory 的没有（319-322，脏数据会渲染 'Invalid Date'）。
- **建议**: 抽公共 `useLoginHistory` composable（fetch/映射/设备解析/位置解析/时间格式化）或直接复用 LoginHistory 页面组件，两份调用同一实现。

### [MEDIUM] router、profileRules、profileFormRef、emailLoading、isMobile 等一批死代码
- **位置**: 264-320,631-632  |  **类别**: maintainability  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `const router = useRouter()`（264 行）声明后从未使用；`profileRules`（312-320）定义后模板没有任何表单绑定它（信息展示用 table，密码表单用 passwordRules）；`profileFormRef`（269 行）、`emailLoading`（268 行，从未赋值）只声明并 return；`authAPI` 导入（251 行）未使用；`isMobile`（632 行 return）模板中从未引用。
- **建议**: 删除未使用的 router/profileRules/profileFormRef/emailLoading/authAPI/isMobile，避免误导后续维护者以为这些功能存在。

### [LOW] getAccountStatusType/Text 与共享 USER_STATUS_MAP 语义冲突
- **位置**: 570-597  |  **类别**: duplication  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 本文件把 inactive 映射为 danger/"已禁用"（575-577、589-591），而共享 utils/statusMaps.js 的 USER_STATUS_MAP（7-12 行）把 inactive 映射为 info/"待激活"。同一状态在两处展示不同含义。
- **建议**: 改用共享 `getUserStatusText/getUserStatusType`，如语义确实不同（禁用 vs 待激活）应修正数据源或统一 map 定义。

### [LOW] Options API + setup() 混写，与其余视图的 `<script setup>` 风格不一致
- **位置**: 246-648  |  **类别**: style  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 本文件用 `export default { setup() {...} }` + 大段 return 清单（620-646），而 Tickets/Invites/Knowledge/UnifiedAuth 均用 `<script setup>`。return 列表还导致死代码不易被发现（见上一条）。
- **建议**: 改为 `<script setup>`，删除 return 清单，与全站风格统一。

## frontend/src/views/Subscription.vue

### [MEDIUM] fetchSubscription 用 userAPI 数据无条件覆盖订阅接口的 clashUrl/qrcodeUrl
- **位置**: 565-569  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `if (userData.clashUrl) subscription.value.clash_url = userData.clashUrl` 等两行：当订阅接口返回新 URL（如刚重置订阅）而用户接口返回旧值（或反之）时，后者覆盖前者，两个数据源谁权威不明确，重置订阅后可能短暂展示旧链接。
- **建议**: 明确单一权威源（订阅接口优先），用户接口仅作缺失值兜底；或在合并前比较并仅在订阅值为空时回填。

### [MEDIUM] 约 250 行未使用 scoped CSS：payment-qr-dialog/upgrade-dialog/subscription-status/masked-url 等
- **位置**: 1184-1303, 930-1144  |  **类别**: maintainability  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 模板（1-393 行）从未使用 `.payment-qr-dialog`(1184-1247)、`.upgrade-dialog`(1248-1303)、`.masked-url`(947)、`.subscription-status`(1009-1030)、`.url-list/.url-item`(1107-1123)、`.qr-code-section`(1124-1144)、`.chip/.chip.active`(930-946)、`.subscription-card`(796)——支付二维码已迁到 Packages.vue、升级抽屉已迁到 UpgradeDevicesDrawer，这些是迁移遗留的死样式。
- **建议**: 删除以上未命中模板的 CSS 块，缩小组件体积约 1/3。

### [MEDIUM] 模板内联重复计算设备比例与三色进度：同一表达式求值 6 次
- **位置**: 94-98  |  **类别**: performance  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `(subscription.onlineDevices || subscription.current_devices || 0)` 与 `(subscription.device_limit || subscription.maxDevices || 1)` 在 94-98 行重复出现 6 次，颜色用嵌套三元链判断 >=0.9 / >=0.7，可读性差且每次渲染重复计算；36-37 行 getRemainingDays(subscription) 也调用 3 次。
- **建议**: 提取 `deviceUsage`（ratio/percent/color）与 `remainingDays` computed，模板只引用计算结果。

### [LOW] stat-card 依赖全局样式（本文件未定义），且图标为神秘字母 A/E/T/D
- **位置**: 67, 76, 84, 90  |  **类别**: architecture  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 模板使用 .stat-card/.stat-icon/.stat-value/.stat-label，但这些类定义在全局 styles/user-client-polish.scss 与 list-common.scss（本组件 scoped 样式无定义），组件不自包含、隐式依赖全局；字母图标对屏幕阅读器会朗读无意义字符，可访问性差。
- **建议**: 把 stat 卡片抽成带自身样式的共享组件（如 StatCard.vue），用语义图标替代字母。

### [LOW] getStatusType/getStatusText/isSubscriptionActive 重复同一过期判定，且与 Dashboard 的专线/设备逻辑重复
- **位置**: 721-740, 754-758  |  **类别**: duplication  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 三个函数各自实现 isExpiredUtil 判断（721-740）；getSpecialNodeModeText(754-758) 与 Dashboard.vue 的 specialNodeModeText computed(872-876) 完全同构，isDeviceFull(741-746) 与 Dashboard 的 isDeviceOverlimit 同构。
- **建议**: 抽取 `useSubscriptionStatus` composable（status/statusType/isActive/isDeviceFull/specialNodeText），三处视图共用。

### [LOW] sendSubscriptionToEmail 双布尔状态 + 2 秒 setTimeout 防重放，状态机脆弱
- **位置**: 696-715  |  **类别**: logic  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: sendEmailLoading 与 sendEmailRequesting 两个标志职责重叠；`setTimeout(() => sendEmailRequesting.value = false, 2000)` 是拍脑袋的防连点 hack，且该定时器未登记清理（卸载后仍会改已卸载组件的 ref）。
- **建议**: 只保留一个 loading 标志（按钮 :loading 本身即防连点），删除 2 秒延迟 hack。

### [LOW] watch 排除协议无防抖，快速点击触发并发二维码生成；且用已废弃的 unescape
- **位置**: 513-515, 607-646  |  **类别**: performance  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: `watch(selectedExcludedProtocols, () => generateQRCodes())` 每次变更立即异步重绘 canvas，无去抖/竞态保护（多次并发生成后写覆盖先写）；618 行 `btoa(unescape(encodeURIComponent(url)))` 使用已废弃的 unescape。
- **建议**: 对 watch 加 debounce（如 150ms）并在 generateQRCodes 内用令牌/最新值守卫；用 `btoa(String.fromCharCode(...new TextEncoder().encode(url)))` 或库函数替代 unescape。

### [LOW] onMounted 内嵌套 onUnmounted 反模式（同 Dashboard），?? 与 || 混用
- **位置**: 508-511, 94, 542-553  |  **类别**: style  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: 508-511 在 onMounted 回调内注册 onUnmounted 卸载监听器，与 Dashboard.vue 1584-1587 同样的问题；94 行用 `||` 而 542-553 用 `??` 做设备数回退，风格不统一；716 行本地 formatDate 包装又遮蔽了工具导入。
- **建议**: 生命周期成对置于顶层；设备数回退统一为 `??`；直接使用 utils/date 的 formatDate。

### [INFO] 安全姿态总体良好：exclude 参数来自硬编码白名单，无注入面
- **位置**: 653-658, 647-652  |  **类别**: security  |  **来源组**: F4-user-views-big (用户端 Dashboard/Packages/Subscription)
- **问题**: buildSubscriptionUrl 追加的 `exclude=` 值取自 availableProtocolOptions 硬编码 value 列表（426-436），不存在参数注入；订阅 URL 均为用户本人资源，无 IDOR；模板插值均由 Vue 转义，无 XSS 面。
- **建议**: 保持现状；若未来协议列表改为后端下发，需对 exclude 值做白名单校验。

## frontend/src/views/Tickets.vue

### [MEDIUM] cancelled 状态被本文件特判，而共享 TICKET_STATUS_MAP 与筛选下拉都没有它
- **位置**: 508-515  |  **类别**: duplication  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `getStatusText`/`getStatusTagType` 在此对 `cancelled` 单独处理，但共享工具 `utils/statusMaps.js` 的 TICKET_STATUS_MAP（32-37 行）没有 cancelled 条目，筛选下拉（23-28 行）也没有"已取消"选项——即数据可能出现的状态无法筛选，且映射散落两处。
- **建议**: 把 `cancelled: { text: '已取消', type: 'danger' }` 加进 TICKET_STATUS_MAP，删掉本文件的本地特判 wrapper，筛选下拉补充"已取消"选项。

### [MEDIUM] 管理员回复判定用字符串比较 `is_admin === 'true'`，依赖后端布尔字段的兼容字段兜底
- **位置**: 253-275  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 模板用 `reply.is_admin === 'true' || reply.is_admin_reply` 区分管理员/用户回复。后端实际返回布尔值（internal/models/ticket.go:66 `IsAdmin bool json:"is_admin"`；handlers/ticket.go:481-482 同时返回 `is_admin` 与 `is_admin_reply` 两个布尔字段）。当前能正常工作完全依赖 `is_admin_reply` 兼容字段；一旦某个响应（如列表接口内嵌回复）缺少该字段，`true !== 'true'` 会把管理员回复误判为"我的回复"（class 与文案都会错）。
- **建议**: 统一归一化：在 loadTickets/viewTicket 收到数据后 `reply.is_admin = reply.is_admin === true || reply.is_admin === 'true' || reply.is_admin_reply`，模板只判断布尔值；或后端直接去掉字符串形态，前端做一次类型收窄。

### [MEDIUM] viewTicket 每次打开详情后都再拉一次完整工单列表，本地已读标记被覆盖
- **位置**: 444-466  |  **类别**: performance  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `viewTicket` 成功回调里先 `markTicketReadLocally(ticketId)` 本地置未读为 0，紧接着 `await loadTickets()` 重新请求整页列表——若后端 GetTicket 已服务端标记已读，本地标记是白做且被刷新结果覆盖；若后端未标记，本地标记又会被服务端旧数据冲掉。同时每次查看详情都多一次列表请求，且与对话框打开无依赖，属于多余往返。
- **建议**: 二选一：依赖服务端已读状态（去掉 markTicketReadLocally，只保留 loadTickets），或保留本地标记并在 `tickets.value` 上直接 patch 该行（不重新拉列表）。

### [LOW] 工单类型选项两处复制（筛选下拉与创建表单）
- **位置**: 31-36,184-188  |  **类别**: duplication  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 类型枚举 technical/billing/account/other 在筛选下拉（31-36 行）和创建表单（184-188 行）重复定义，新增类型需改两处。
- **建议**: 抽成模块级常量 `TICKET_TYPE_OPTIONS`（label/value 数组）供两处 v-for 渲染。

### [LOW] createTicket 的 catch 吞掉后端错误详情，与本文件其他分支不一致
- **位置**: 436-438  |  **类别**: error-handling  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `catch { ElMessage.error('创建工单失败') }` 不展示 `error.response?.data?.detail/message`，而 viewTicket（464 行）与 addReply（503 行）都用 `error.response?.data?.detail || .message`。后端返回的具体原因（如标题超长、工单上限）用户完全看不到。
- **建议**: 统一为 `const msg = error.response?.data?.detail || error.response?.data?.message || '创建工单失败'`。

### [LOW] ticketTableRef 声明并绑定但脚本从未使用
- **位置**: 323  |  **类别**: maintainability  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `const ticketTableRef = ref(null)`（323 行）只绑定在 el-table 的 ref 上，脚本内无任何读取。
- **建议**: 删除该 ref（表格列宽持久化由 usePersistentTableColumns 的 header-dragend 处理，不需要表格实例）。

## frontend/src/views/UnifiedAuth.vue

### [MEDIUM] notification/showNotification 与三个 isXxxFocused 全是未消费的死状态
- **位置**: 265-281,210-219  |  **类别**: maintainability  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `notification` reactive 与 `showNotification`（265-281 行）定义后模板从未引用（提示全走 ElMessage）；`isPasswordFocused`/`isRegPasswordFocused`/`isForgotPasswordFocused`（210-219 行）只在 focus/blur 事件里赋值（41、95、109、145、159 行），从未被读取——疑似为样式切换预留但从未接线。
- **建议**: 删除这些死状态；若密码框聚焦样式确实需要，再通过 class 绑定真正消费它们。

### [MEDIUM] 进入页面重复拉取公共设置：settingsStore 走缓存、checkRegistrationSettings 裸请求
- **位置**: 722-727,681-720  |  **类别**: performance  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: onMounted 里 `Promise.all([checkRegistrationSettings(), settingsStore.loadSettings()])` 并发发起两个 GET /settings/public-settings：前者用 settingsAPI.getPublicSettings（不走缓存），后者用 cachedAPI.getPublicSettings（1h 缓存，api.js:791-797）。同时 registrationEnabled/inviteCodeRequired/emailVerificationRequired/minPasswordLength（221-224 行）与 settingsStore 的 allowRegistration/requireEmailVerification/minPasswordLength（settings.js:16-19）是同一份配置的两份拷贝。
- **建议**: 统一只调 settingsStore.loadSettings()，其余值从 store 派生；或让 checkRegistrationSettings 复用 cachedAPI.getPublicSettings 并删掉重复状态。

### [MEDIUM] "保持登录 30 天"复选框 disabled 且恒真，是误导性死控件
- **位置**: 56,207  |  **类别**: ux  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `<el-checkbox v-model="rememberMe" disabled>保持登录 30 天</el-checkbox>`（56 行）永远不可操作；rememberMe 默认 true（207 行），且 authStore.login 内部硬编码 `const remember = true`（store/auth.js:173-177）——无论复选框如何，登录后 token 都持久化到 localStorage。用户无法关闭持久登录，控件纯摆设。
- **建议**: 要么把 `remember: rememberMe.value` 真正传入 store 并让 store 用该值决定 sessionStorage/localStorage，要么删除该复选框并给出固定文案说明登录状态会保持 30 天。

### [LOW] 密码校验器与 UserSettings 策略不一致，且登录/注册同页两套规则
- **位置**: 228-236,303-311  |  **类别**: duplication  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: passwordValidator（228-236 行）要求"必须含字母和数字 + 4 类中至少 3 类"，与 UserSettings.vue 的 validatePasswordStrength（8 位 + 3/4 类 + 弱口令黑名单）不同——两页注册/改密体验不一致；本页登录规则 password 只要求 min 6（308-310 行），注册却要求 minPasswordLength(默认 8)+组合规则（324-328 行），同页两套标准。
- **建议**: 密码强度规则收敛到公共 utils（与 UserSettings 共用一份），登录的 min 长度与注册对齐（minPasswordLength），避免后端报错与前端校验不一致。

### [LOW] 后端错误文案用 includes() 字符串匹配，脆弱且易漂移
- **位置**: 480-528  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: handleLogin 用 `errorMessage.includes('锁定15分钟')`/`includes('账户已被禁用')`/`includes('系统维护')`（480、506、518、525 行）分支，后端一旦改措辞（如改"账号已锁定"）这些分支全部失效退回默认提示。
- **建议**: 改用后端结构化错误码（如 error.code === 'ACCOUNT_LOCKED'/'MAINTENANCE'）或统一映射表，替代中文字面量匹配。

### [LOW] 邀请码被校验两次：query 手动触发 + watch 防抖触发
- **位置**: 651-679,734-737  |  **类别**: performance  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: route.query.invite 时先 `await validateInviteCode(route.query.invite)`（736 行），同时 registerForm.inviteCode 的 watch（668-679 行）在 500ms 后再次发起相同的 GET /invites/validate/{code}，产生重复请求。
- **建议**: watch 内跳过初始赋值（如赋值前先置空一次或加标志位），或在 onMounted 里只赋值不手动校验，交给 watch 统一处理。

### [INFO] 可访问性总体良好，表单输入仍以 placeholder 为唯一提示
- **位置**: 36-177  |  **类别**: ux  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 密码显隐按钮带 aria-label（46、100、114、150、164 行），按钮 :disabled 有 :focus-visible 样式（942-945 行）；但三张表单的 el-input 均无显式 label 关联（el-form-item 未设置 label，仅 placeholder 提示），读屏用户无法获知字段含义。
- **建议**: 为各 el-form-item 补充 label（可配合 sr-only 样式隐藏视觉呈现）或用 aria-label 绑定输入。

## frontend/src/views/UserSettings.vue

### [HIGH] 邮箱修改是假成功：PUT /users/me 后端忽略 email 与 verification_code，前端仍提示"邮箱修改成功"
- **位置**: 671-696  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `confirmEmailChange` 调用 `userAPI.updateProfile({..., email, verification_code})` → PUT /users/me → 后端 `UpdateCurrentUser`（internal/api/handlers/user.go:148-215）只解析 username/nickname/avatar/theme/language/timezone，`email` 与 `verification_code` 被静默丢弃并返回"更新成功"；全后端也不存在独立的用户改邮箱端点（grep change-email/update-email 无结果）。前端随后 `profileForm.email = newEmail` 并提示成功（683-686 行）——邮箱实际从未变更，验证码发到新邮箱后也无任何消费流程。
- **建议**: 后端新增/接通改邮箱流程（发验证码 → 校验 verification_code → 更新 email → 重置 is_verified），前端改为调用该专用端点；在端点就绪前移除该入口，避免虚假成功提示。

### [HIGH] 头像上传功能不可用：el-upload action="#"，无 http-request/on-success，选中文件后什么都不发生
- **位置**: 55-66,697-707  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 模板中 `el-upload action="#" :show-file-list="false" :before-upload="beforeAvatarUpload"`（55-66 行）：beforeAvatarUpload 只做类型/大小校验（697-707 行），没有 http-request 覆盖、没有 on-success/on-error/on-change。校验通过返回 true 后 Element Plus 向当前页 URL POST 文件，结果无人处理；`profileForm.avatar` 也从未被更新，saveProfile（493-527 行）提交的是服务端旧值。"修改头像"从选择图片到保存整条链路都是死的。
- **建议**: 补 `:http-request` 自定义上传（携带 JWT 上传至后端头像接口并回填 URL），或至少在 before-upload 校验后调用真实上传 API 并把返回 URL 写入 profileForm.avatar 再保存。

### [MEDIUM] 密码策略两套实现且规则不一致：本页 vs UnifiedAuth 页
- **位置**: 302-330,228-236  |  **类别**: duplication  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: 本文件 validatePasswordStrength + weakPasswords 黑名单（302-330 行）要求"长度≥minPasswordLength 且 4 类字符中至少 3 类且不在弱口令名单"；UnifiedAuth.vue 的 passwordValidator（228-236 行）要求"字母+数字且 4 类中至少 3 类"，无黑名单。同属认证/设置流程，两个页面密码规则不一致，用户会得到不同校验结果。
- **建议**: 把密码策略收敛到公共工具（如 settingsStore.validatePassword/getPasswordError，settings.js 已有 177-192 行的实现）或独立 utils/passwordPolicy.js，两页共用同一规则与提示文案。

### [LOW] 多处 `xxxFormRef.value.validate()` 无空引用保护
- **位置**: 495,530,673  |  **类别**: error-handling  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: saveProfile（495 行）、changePassword（530 行）、confirmEmailChange（673 行）直接 `await xxxFormRef.value.validate()`，若模板未渲染对应表单（ref 为 undefined），会抛 TypeError 并落入通用 catch，用户看到"保存失败: Cannot read properties of null"这类原始报错。
- **建议**: 统一加守卫：`if (!xxxFormRef.value) { ElMessage.error('表单未初始化'); return }`。

### [LOW] 偏好保存拆两次请求且主题单选与 store 加载不同步
- **位置**: 589-649  |  **类别**: logic  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: savePreferenceSettings 先 PUT /users/theme（经 themeStore.setTheme）再 PUT /users/preferences 只带 timezone，成功/失败消息四路分支（609-633 行）复杂易错；且 `preferenceForm.theme` 在 setup 时快照 themeStore.currentTheme（286 行），onMounted 里 themeStore.loadUserTheme()（720 行）拉回服务端主题后不回写该表单，单选可能显示陈旧值。
- **建议**: loadUserTheme 完成后同步 `preferenceForm.theme = themeStore.currentTheme`；theme 与 timezone 尽量合并为一次 /users/preferences 请求，避免两请求+四分支消息逻辑。

### [LOW] Setting 图标与 isMobile 未使用
- **位置**: 228,252,257,724  |  **类别**: maintainability  |  **来源组**: F6-user-views-rest (用户端其余视图)
- **问题**: `Setting` 从图标导入并在 components 注册（228、252 行）但模板从未使用；`isMobile`（257 行）返回（724 行）但模板未引用（本页布局靠 CSS media query 而非 isMobile 分支）。
- **建议**: 删除未使用的 Setting 导入/注册与 isMobile 返回值。

## frontend/src/views/admin/AbnormalUsers.vue

### [CRITICAL] 「标记正常」是纯前端假操作：无任何 API 调用，只弹成功提示
- **位置**: 559-572  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: markAsNormal() 内 confirmWarning 通过后直接 ElMessage.success('用户已标记为正常') + loadAbnormalUsers()，全程没有调用任何后端接口（utils/api.js 中 adminAPI 只有 getAbnormalUsers/getUserDetails，无 mark-normal 方法，见 api.js:553）。用户点击后数据没有任何变化，却收到“已标记为正常”的成功提示，属于误导性假成功。
- **建议**: 新增后端接口（如 POST /admin/users/:id/mark-normal）并封装 adminAPI.markUserNormal(userId)，在弹成功提示前 await 该调用，失败时提示真实错误；同时给按钮加 loading 状态。

### [MEDIUM] useRoute 重复声明 + setTimeout(500) 魔法延时打开详情
- **位置**: 436, 583-592  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: setup 顶部已 `const route = useRoute()`（436 行，之后从未使用），onMounted 内又 `const route = useRoute()` 重新声明（585 行）形成遮蔽；且用 setTimeout(500) 等待列表加载后再 viewUserDetails，与 loadAbnormalUsers 并发存在竞态：慢网络下列表未返回就打开抽屉，或 500ms 不够。
- **建议**: 删除外层无用声明，改为 watch(() => route.query.user_id, id => { if (id) viewUserDetails(Number(id)) }, { immediate: true })，并在 viewUserDetails 前 await loadAbnormalUsers()。

### [MEDIUM] getDefaultDateRange 用 toISOString 取 UTC 日期，UTC+8 下“今天”会偏一天
- **位置**: 454-462  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `firstDay.toISOString().split('T')[0]` 把本地零点转成 UTC 再取日期：在 UTC+8（本系统目标时区）每天 00:00–08:00 之间，“今天”会变成昨天、月初变上月最后一天，导致默认筛选区间错位。
- **建议**: 用本地日期格式化（手动拼接 getFullYear/getMonth/getDate），不要走 toISOString。

### [LOW] 日期参数风格与其它页面不一致（date_range[] vs start_time/end_time）
- **位置**: 473-491  |  **类别**: architecture  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 本页发送 `params['date_range[]'] = filters.dateRange`，而所有 logs 页与 Coupons 等用 start_time/end_time，Analytics 用 range=day/month/year；同系统多种日期参数风格，后端各自适配，前端契约难维护。
- **建议**: 统一为 start_time/end_time，或由后端封装统一 QueryDTO；至少在前端 api 层做归一化。

## frontend/src/views/admin/AbnormalUsers.vue, Tickets.vue, Knowledge.vue, Promotions.vue, Coupons.vue

### [MEDIUM] PaginationBar 尺寸切换是否重置页码的行为在页面间不一致
- **位置**: AbnormalUsers 251-256 / Tickets 180-185 / Knowledge 111-116 / Promotions 122-127 / Coupons 189-196  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: Coupons 用 `@size-change="handlePageSizeChange"`（重置 page=1）、日志页用 onSizeChange（重置 page=1）；而 AbnormalUsers/Tickets/Knowledge/Promotions 只绑 `@change="loadXxx"`，PaginationBar 的 change 事件携带的是旧 page（PaginationBar.vue:71-74），切 pageSize 后停留在原页码，第 5 页切 50 条/页可能落到空页。
- **建议**: 统一约定：所有使用方在 size-change 时重置 page=1；或让 PaginationBar 在 size-change 时自动 emit 带 page=1 的 change。

## frontend/src/views/admin/Analytics.vue

### [MEDIUM] 绕过 adminAPI 直接裸调 api.get/api.post 且硬编码路径
- **位置**: 522, 542, 565-569  |  **类别**: architecture  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: onTemplateChange/sendEmail/loadData 全部 `api.get(`/admin/analytics/revenue?range=${range}`)` 手拼 URL，与其他管理页（走 adminAPI 方法封装）不一致；且 templateName 直接插值进路径 `/admin/email-templates/${templateName}`，虽然当前来自固定列表，但后续若放开输入即成路径参数注入点。
- **建议**: 在 utils/api.js 为这些接口封装 adminAPI.getAnalyticsRevenue(range) 等方法，参数走 params 而不是拼 URL。

### [MEDIUM] CSV 导出未转义：CSV 注入（公式注入）+ 逗号/引号破坏结构
- **位置**: 590-665  |  **类别**: security  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: exportData 直接把 username/email/device_type 等用户可控字段拼进 CSV（646-648 行），未做引号包裹或等号前缀处理。用户名以 =、+、-、@ 开头时 Excel/WPS 会执行公式（CSV 注入）；字段内含逗号/换行还会破坏列结构。
- **建议**: 写 escapeCsv(value)：以 " 包裹并将内部 " 翻倍，对以 =+-@ 开头的值前置单引号；对每行字段统一套用。

### [LOW] openContactDialog 对空 expire_time 产生 NaN 判定
- **位置**: 486-514  |  **类别**: error-handling  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `new Date(user.expire_time).getTime()` 当 expire_time 为 null 时为 NaN，daysDiff=NaN，三个 if 分支都不命中（NaN<0 为 false），模板自动选择静默失效；isLongTimeNoLogin 对非法日期同样返回 false。
- **建议**: 先判断 expire_time/last_login 是否为合法日期（Number.isNaN 检查），非法时跳过模板自动选择并提示用户手动选择。

### [LOW] onTemplateChange 异步未衔接：选模板后立即发送会带旧主题/内容
- **位置**: 516-532  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: onTemplateChange 是 async 且模板下拉 @change 不 await；用户选择模板后马上点“发送邮件”，sendEmail 读到的是未更新的 subject/content，模板拉取失败仅 console.error 静默。
- **建议**: 在 sendEmail 前重新校验模板：若 templateName 非 custom 且 subject/content 为空则提示等待/重新拉取；或把模板内容缓存后同步赋值。

## frontend/src/views/admin/Config.vue

### [MEDIUM] saveSoftwareConfig 不检查响应 success，后端业务失败也弹“保存成功”
- **位置**: 243-253  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `await softwareConfigAPI.updateSoftwareConfig(softwareForm); ElMessage.success('软件配置保存成功')` 直接成功提示；同文件 saveEmailConfig/saveSubscriptionAccessConfig 都检查了 response.data.success，此处遗漏，后端返回 success:false 时用户被误导。
- **建议**: 改为与 saveEmailConfig 一致：检查 response.data.success，失败读 response.data.message 提示。

### [LOW] 大段死 CSS：avatar-uploader/backup-section/queue-stats/payment-form 等均无对应模板
- **位置**: 415-490, 755-760  |  **类别**: maintainability  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 模板中不存在 .avatar-uploader、.backup-section、.email-queue-section、.queue-stats、.stat-card、.stat-number、.stat-label、.queue-filter、.payment-form 对应节点，疑似从旧版复制残留，增加维护噪音。
- **建议**: 删除未使用样式块，或用 PurgeCSS 类审计工具排查。

### [LOW] 软件配置表单缩进混乱、空 el-col 占位
- **位置**: 12-114  |  **类别**: style  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 18-19、74、95、104 行出现深层嵌套缩进（如 el-form 内容缩进 14 空格），且有多个 `<el-col :span="12"></el-col>` 空占位，影响可读性。
- **建议**: 格式化模板并删除空 el-col。

## frontend/src/views/admin/ConfigUpdate.vue

### [HIGH] 轮询在组件卸载后仍会继续：poll 的 finally 无条件续订定时器
- **位置**: 483-512, 682-688  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: startPolling 的 poll() finally 中 `if (status.value.is_running) pollingTimer = setTimeout(poll, 1500)`；onUnmounted 调 stopPolling 只清了当前 timer，若卸载时正有一次 poll 在执行（await getStatus/getLogs 挂起中），其 finally 仍会再订一个新 timer，页面离开后请求持续打到后端（泄漏 + 无效流量）。
- **建议**: 引入 `let unmounted = false`，onUnmounted 置 true；poll 的 finally 先检查 `if (unmounted) return` 再决定是否续订。

### [MEDIUM] Options API export default + setup() 与同目录 script setup 风格割裂
- **位置**: 287-699  |  **类别**: style  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 本文件用 `export default { name, components, setup() {...} }` 且手工 return 一大串，而同目录 Tickets/Coupons/Knowledge/Promotions 都是 <script setup>；组件/图标注册方式也不同。
- **建议**: 统一改写为 <script setup>，删除 components 注册块，减少约 40 行样板。

### [LOW] 开始/停止/测试更新无二次确认
- **位置**: 515-562  |  **类别**: ux  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: startUpdate/stopUpdate/testUpdate 直接触发远程任务控制，无确认弹窗；误触“停止更新”会打断正在执行的更新任务。
- **建议**: 对 stopUpdate/testUpdate 加 confirmWarning，与 clearLogs 的确认模式保持一致。

## frontend/src/views/admin/Coupons.vue

### [HIGH] 创建按钮不重置表单/editingCoupon：编辑后直接点“创建”会变成再次编辑旧券
- **位置**: 5, 550-566  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: “创建优惠券”按钮只 `showCreateDialog = true`，而 UserLevels.showAddDialog / Knowledge.showCategoryDrawer() / Promotions.showDrawer() 都会先 resetForm + editingLevel=null。若用户编辑优惠券 A 后关闭抽屉（未保存），再点创建：抽屉标题仍为“编辑优惠券”、表单仍是 A 的数据，保存会走 updateCoupon(A.id) 而非创建。
- **建议**: 创建按钮改为 `handleCreate()`：editingCoupon=null + resetForm() + showCreateDialog=true；或 watch showCreateDialog 关闭时 resetForm。

### [MEDIUM] 硬编码 Asia/Shanghai 时区做时间往返转换，非东八区管理员会双偏
- **位置**: 492-495, 559-560  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 保存时 `dayjs(formData.valid_from).tz('Asia/Shanghai').format('YYYY-MM-DDTHH:mm:ss')`，编辑回填时又 tz 转换回 Date；若后端把该字符串当 UTC 存，或管理员不在东八区，会出现 ±8h 偏移。系统其它页面（Promotions）直接传 ISO。
- **建议**: 与后端确认时间字段语义（UTC 时间戳 or 本地字符串），前端统一传 ISO 或统一用同一时区常量，避免两端各猜一次。

### [LOW] 缺少 valid_from < valid_until 与 discount_value>0 的交叉校验
- **位置**: 412-418  |  **类别**: error-handling  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: couponRules 只做必填，未校验失效时间晚于生效时间、折扣值大于 0（0% 优惠券可被创建）；Promotions 有 start<end 校验（442-445），此处缺失。
- **建议**: 加自定义 validator 对比 valid_from/valid_until，并对 discount_value 加 min 约束（discount 类型 >=1）。

### [LOW] 编辑时删除 code 字段，若后端缺失即生成新码则每次编辑都换码
- **位置**: 497-499  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: saveCoupon 中 `if (!formData.code || ...) delete formData.code`，而 editCoupon 从未把 code 赋回表单；若后端 update 接口对缺 code 走“自动生成”分支，用户编辑保存后优惠券码会悄悄变化，影响已打印/分享的码。
- **建议**: 编辑时 Object.assign 带上 `code: coupon.code`，仅新建且留空时才 delete。

## frontend/src/views/admin/CustomNodes.vue

### [MEDIUM] testNodeFromLink 是空桩：未调任何接口却弹"测试连接通过"
- **位置**: 1373-1378  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: 链接查看弹窗的"测试连接"按钮走 `testNodeFromLink`，函数体只设置 loading 标志并 `ElMessage.success('测试连接通过')`，没有任何网络请求，向管理员展示虚假的成功结果。
- **建议**: 接入真实接口（如对解析出的 server:port 做 TCP 探测，或复用 testCustomNode 的后端能力），失败时提示真实错误。

### [LOW] 选择逻辑重复：handleGridSelect 纯转发、isAllSelected setter 与 toggleMobileSelectAll 双写
- **位置**: 936-952  |  **类别**: duplication  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `handleGridSelect` 只是 `handleMobileSelect(node, checked)` 的转发壳；移动端全选 checkbox 同时绑 `v-model="isAllSelected"`（走 setter）与 `@change="toggleMobileSelectAll"`，同一事件触发两次等效赋值，且两处实现完全重复。
- **建议**: 删除 handleGridSelect 直接复用 handleMobileSelect；全选只保留 v-model（去掉 setter 或去掉 @change 二选一）。

### [LOW] parseNodeLink 无失败分支，解析失败时静默无反馈
- **位置**: 1158-1173  |  **类别**: error-handling  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `if (res.data.success) {…}` 之外没有任何 else/catch 提示；接口返回 success=false 或抛错时 parsedNode 保持 null，预览区不出现，用户不知道为何没解析出来。
- **建议**: 补充 else 分支展示 `res.data.message`，catch 中提示解析失败。

### [LOW] copyLink 与 handleUserSearch 缺少异常处理
- **位置**: 1235-1240, 1259-1266  |  **类别**: error-handling  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: copyLink 的 `navigator.clipboard.writeText` 在非安全上下文会 reject，无 try/catch 产生未处理 Promise 拒绝；handleUserSearch 请求失败同样未 catch（仅 finally 复位 loading），错误无提示。
- **建议**: copyLink 参照 Settings.vue 的 copyText 加 try/catch；handleUserSearch 加 catch 并提示"搜索失败"。

### [LOW] 筛选/搜索变化不清空 selectedNodes，批量操作可能作用于当前不可见节点
- **位置**: 924-935  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: handleFilterChange/resetFilters 只重置页码与重新加载，不清空 selectedNodes；跨页勾选后切换筛选，批量测速/删除/分配针对的是旧列表快照中的节点 id，与当前视图不一致，存在误操作风险。
- **建议**: 在 handleFilterChange/resetFilters 及数据重载时清空 selectedNodes 并 tableRef.clearSelection()。

### [LOW] saveNode 的 validate(callback) 异步模式存在双击竞态窗口
- **位置**: 993-1011  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `await nodeFormRef.value.validate(async (valid) => {…})` 中保存逻辑在回调内异步执行，外层 await 在校验完成后即返回，`saving` 置位发生在回调内，双击保存按钮时第二次点击可能在校验完成前/后各触发一次 create/update。
- **建议**: 改为 `const valid = await nodeFormRef.value.validate().catch(() => false); if (!valid) return; saving.value = true; …` 的顺序化写法。

### [LOW] toggleNodeStatus 无防抖，快速连点会以旧值覆盖新值
- **位置**: 1047-1055  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: switch 双向绑定立即改 `node.is_active` 后异步 PUT；快速连续切换时两次请求携带相同/相反状态，后到响应不回滚，界面状态可能与服务端不一致（失败时仅简单取反回滚）。
- **建议**: 请求期间禁用该 switch（如 `:loading`），或对 toggle 加防抖/串行化。

### [LOW] DocumentCopy 图标导入并注册但从未使用
- **位置**: 690, 707  |  **类别**: maintainability  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `DocumentCopy` 仅在 import 与 components 注册中出现（grep 计数 2），模板无引用，属死代码。
- **建议**: 删除该图标 import/注册。

### [INFO] formatExpire/formatTime 用 toLocaleString，与全局 dayjs 时间格式不一致
- **位置**: 1361-1362  |  **类别**: style  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: 其他管理视图（如 Orders.vue）统一走 `@/utils/date` 的 formatDateTime（dayjs+时区），此处用浏览器本地化输出，同一节点列表里到期时间格式随浏览器环境漂移。
- **建议**: 统一改用 date 工具函数，保持全站时间格式一致。

## frontend/src/views/admin/Dashboard.vue

### [HIGH] 异常客户列表取数路径与后端返回结构不匹配，永远为空
- **位置**: 448-465  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: loadAbnormalUsers 用 `const data = response.data.data || []` 后 `Array.isArray(data) ? data.slice(0,5) : []`。后端 GetAbnormalUsers（handlers/dashboard.go:375）返回 {users, total, page, size} 对象而非数组，Array.isArray 恒 false，异常客户卡片恒为空；且此处裸用 api.get，未用已有封装 adminAPI.getAbnormalUsers（utils/api.js:553）。
- **建议**: 改为 `const data = response.data.data || {}; abnormalUsers.value = Array.isArray(data) ? data : (data.users || [])`，并改用 adminAPI.getAbnormalUsers()。

### [MEDIUM] formatTimeAgo 在 Safari 上解析空格分隔时间为 Invalid Date，显示 'NaN年前'
- **位置**: 561-578  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 后端 FormatBeijingTime（utils/common.go:302）输出 '2006-01-02 15:04:05'（无 T 无时区），`new Date(dateString)` 在 Safari/iOS 返回 Invalid Date，diffMs=NaN，最终显示 'NaN年前'。GetRecentUsers/GetRecentOrders 的 created_at 均为此格式。
- **建议**: 先 `dateString.replace(' ', 'T')` 再 new Date，并加 `if (isNaN(date.getTime())) return '未知'` 守卫；SystemLogs.formatDate 同理。

### [MEDIUM] 桌面表格与移动端卡片双通道维护 selectedExpiring，状态可能不同步
- **位置**: 600-611  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 桌面 el-table 用 @selection-change（table 内部勾选态），移动端 toggleExpiringSelection 手动 push/splice 同一数组。桌面勾选后切到移动端增删再切回，表格内部勾选态不回退，批量按钮计数与实际勾选不一致。
- **建议**: 以 selectedExpiring 为唯一数据源：桌面表格用 :row-key + reserve-selection，数据变化时用 toggleRowSelection 同步。

### [MEDIUM] 批量发送到期提醒无超时覆盖，后端串行发信超过 axios 10s 超时导致 UI 误报失败
- **位置**: 623-638  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: api.js 默认 TIMEOUT=10000，后端 BatchSendExpireReminder（user.go:2651 起）串行发 SMTP 且 user_ids 无上限；勾选上百用户必然超时，前端提示'发送失败'但邮件实际已发出。
- **建议**: 给 adminAPI.batchSendExpireReminder 传长超时（如 120s），发送前按数量提示；后端改异步队列并限单次最大数量。

### [MEDIUM] getExpireTagType 存在死分支
- **位置**: 612-617  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: `days <= 1` 与 `days <= 3` 都返回 'warning'，第二个分支不可达。
- **建议**: 合并为 `days <= 0 ? 'danger' : days <= 3 ? 'warning' : 'info'`，或为 1 天内引入独立色值。

### [MEDIUM] 到期订阅以 user_id 作 key 与选中标识，同一用户多条订阅时 key 重复
- **位置**: 275, 601  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 后端按订阅逐条返回（含独立 id），前端移动端 `:key="item.user_id || item.id"`（275）与选中逻辑（601）按用户 ID 标识：同一用户两条到期订阅 → Vue 重复 key，批量发送提示数量与实际不符。
- **建议**: 统一以订阅 id 为 key 与选中标识，发送时映射去重 user_id。

### [LOW] Promise.allSettled 外层 catch 永不触发
- **位置**: 646-657  |  **类别**: maintainability  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: allSettled 不会 reject，内部 5 个 load 函数又各自 try/catch，外层 catch 不可达。
- **建议**: 删除外层 try/catch。

### [LOW] getAbnormalTypeText 定义后未导出未使用
- **位置**: 492-499  |  **类别**: maintainability  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 函数未被模板引用，也未在 setup return 中导出。
- **建议**: 删除，或并入 statusMaps.js 统一管理。

### [LOW] CSS 媒体查询与响应式类大量重复定义
- **位置**: 726, 839, 1160, 1236, 1267  |  **类别**: style  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 768px 媒体查询出现 3 次、480px 出现 2 次，.desktop-only/.mobile-only 两处重复，且多处 !important 互相覆盖。
- **建议**: 收敛为单个媒体查询块；desktop-only/mobile-only 提升为全局工具类。

## frontend/src/views/admin/Dashboard.vue, Nodes.vue

### [MEDIUM] 订单/节点状态映射在视图内联重复且与 statusMaps.js 颜色不一致
- **位置**: 466-483 / 670-671  |  **类别**: duplication  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: Dashboard 订单映射 cancelled 为 danger（statusMaps 为 info）且缺 failed；Nodes 节点映射缺 maintenance。同一状态跨页颜色/文案不同。
- **建议**: 统一引用 statusMaps.js 的 getOrderStatus*/getNodeStatus*，以 statusMaps 为准校正颜色。

## frontend/src/views/admin/EmailDetail.vue

### [CRITICAL] response.success 永远为 undefined：本页加载/重试/删除全部判定失败
- **位置**: 230-249, 258-264, 280-286  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: api.js 响应拦截器返回完整 axios response（api.js:335 `return response`），因此 `const response = await adminAPI.getEmailDetail(emailId)` 后 `if (response.success)`（239 行）恒为 false → 永远弹“获取邮件详情失败”、emailDetail 恒为 null，页面只会渲染“邮件不存在或已被删除”；retryEmail（259）、deleteEmail（281）同样永远报失败，即使后端操作成功。
- **建议**: 统一改为 `if (response.data?.success)`（或复用 EmailQueue 的 handleResponse 工具），并解包 response.data.data。

### [MEDIUM] confirmDelete 参数误用：entityName 传了整句消息、count 传了字符串，弹窗文案错乱
- **位置**: 276-279  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `confirmDelete(`确定要删除发送到 ${emailDetail.value.to_email} 的邮件吗？`, '确认删除')`——按 confirmAction.js:40 签名 (entityName, count, options)，此处 entityName=整句消息、count='确认删除'（字符串，`count>1` 为 false），最终弹窗文案变成“确定删除该确定要删除发送到 xxx 的邮件吗？删除后不可恢复。”，标题丢失。
- **建议**: 改为 `confirmDelete('邮件', 1, { message: `确定要删除发送到 ${...} 的邮件吗？`, title: '确认删除' })`。

### [MEDIUM] confirmWarning 第二参传字符串，标题被吞并字符串被展开成对象
- **位置**: 253-256  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `confirmWarning(`确定要重试发送邮件到...吗？`, '确认重试')`——confirmWarning(message, options) 中 options 应为对象；字符串 '确认重试' 被当作 options，`options.title` 为 undefined（标题回落默认“确认操作”），`...options` 展开字符串成 {0:'确',1:'认',...} 数字键，意图完全失效。
- **建议**: 改为 `confirmWarning(msg, { title: '确认重试', confirmButtonText: '确认重试' })`（与 EmailQueue 用法一致）。

### [LOW] 状态/优先级映射第三份本地实现
- **位置**: 327-362  |  **类别**: duplication  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: getStatusTagType/getStatusText/getPriorityTagType/getPriorityText 在 statusMaps.js（EMAIL_STATUS_MAP/TICKET_PRIORITY_MAP 等）已存在，本文件重复定义，与 EmailQueue 各持一份，改一处漏两处。
- **建议**: 引入 statusMaps 的 getStatusConfig/getTicketPriorityText 等统一函数。

## frontend/src/views/admin/EmailQueue.vue

### [MEDIUM] 批量重试/删除用 Promise.all，单条失败即整体报错且部分成功不可见
- **位置**: 771-800  |  **类别**: error-handling  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `const promises = selectedEmails.value.map(email => adminAPI.retryEmail(email.id)); await Promise.all(promises)`——任一条失败 Promise.all 直接 reject，其它已成功的请求结果被吞掉，只提示“批量重试失败”，无法告知用户哪些成功哪些失败。
- **建议**: 改用 Promise.allSettled，统计成功/失败数量并汇总提示；或后端提供批量接口一次提交。

### [MEDIUM] 选中集合跨分页保留，翻页后批量操作作用于不可见行
- **位置**: 757-770, 294-303  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: selectedEmails 在翻页/筛选后不清空，移动端批量按钮（294-303）与桌面端按选中数量显示；用户勾选第 1 页几条后翻到第 3 页点批量删除，删的是不可见的旧选择，容易误删。
- **建议**: 在 handleCurrentChange/handleSizeChange/applyFilter/resetFilter 时清空 selectedEmails（或改为仅当前页可操作并提示）。

### [LOW] 本地 STATUS_MAP 重复实现 statusMaps.js 已有 EMAIL_STATUS_MAP
- **位置**: 459-465, 810-815  |  **类别**: duplication  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: utils/statusMaps.js 已导出 EMAIL_STATUS_MAP（text/type 结构），本文件仍手写 STATUS_MAP 并自实现 getStatusTagType/getStatusText；EmailDetail.vue 又写第三份。
- **建议**: 统一从 statusMaps.js 引入 EMAIL_STATUS_MAP + getStatusConfig(status, map)，删除本地两份映射。

### [LOW] onIframeLoad 内嵌套 setTimeout 未在卸载时清理
- **位置**: 567-607  |  **类别**: performance  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 外层 iframeLoadTimeout 在 onUnmounted 清理了（838-840），但第 589 行嵌套的 200ms setTimeout（改 iframe 高度）没有句柄，卸载后仍会执行 DOM 操作。
- **建议**: 把嵌套定时器句柄也存下来并在 onUnmounted 清除，或加 isUnmounted 守卫。

## frontend/src/views/admin/Invites.vue

### [MEDIUM] confirmDelete 参数用反，确认弹窗文案严重错乱
- **位置**: 865-868, 896-899  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `confirmDelete(message, '确认批量删除')` 把整段提示语当作 entityName、把按钮文案当作 count（confirmAction.js 签名是 `(entityName='数据', count=1, options={})`）。因 `'确认批量删除' > 1` 为 false 走单数分支，用户看到的是"确定删除该确定要删除选中的 N 个邀请码吗？已使用的邀请码将被禁用而不是删除。吗？删除后不可恢复。"这种拼接乱文。
- **建议**: 改为 `confirmDelete('邀请码', selectedCodes.length, { message, title: '确认批量删除', confirmButtonText: '确认删除' })`，relations 同理。

### [MEDIUM] 桌面表格 used_count/max_uses 直接渲染原始 max_uses，可能显示 [object Object]
- **位置**: 160-164  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: loadInviteCodes（L673-686）专门为后端可能返回的 sql.NullInt64 对象形态准备了 `max_uses_display` 解析，但桌面表格列仍写 `{{ scope.row.used_count }} / {{ scope.row.max_uses || '∞' }}`，未使用解析值；若后端返回 `{Valid:true,Int64:5}` 对象，该列会渲染成 "[object Object]"，与移动端卡片（用 max_uses_display）不一致。
- **建议**: 统一改为 `{{ scope.row.used_count }} / {{ scope.row.max_uses_display || scope.row.max_uses || '∞' }}`，桌面与移动共用解析后的字段。

### [LOW] 自实现 formatDate 与全局日期工具重复且格式不同
- **位置**: 934-945  |  **类别**: duplication  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: Invites.vue 内联 `formatDate`（toLocaleString('zh-CN')），而 Orders.vue 等用 `@/utils/date` 的 formatDateTime（dayjs+时区、格式如 YYYY-MM-DD HH:mm:ss），同后台两个列表时间格式不一致。
- **建议**: 删除本地 formatDate，统一 import `formatDateTime` 并处理空值。

### [LOW] 邀请设置抽屉与 Settings.vue 的"邀请设置"Tab 是同一配置的两套 UI
- **位置**: 454-517  |  **类别**: duplication  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: Invites.vue 抽屉（仅 inviter_reward/invitee_reward 两字段）与 Settings.vue invite Tab（另含 min_order_amount/new_user_only）都读写 `/admin/settings`+`/admin/settings/invite`。后端 updateSettingsCommon 是逐 key upsert，暂不会抹掉其它字段，但两处定义漂移风险高（如新增字段只在一处出现）。
- **建议**: 抽公共 InviteSettingsForm 组件（两字段模式可通过 prop 控制），两页共用同一份表单与保存逻辑。

### [LOW] 批量删除成功消息对响应形态敏感，缺字段时误报"成功删除 0 个"
- **位置**: 872-879  |  **类别**: error-handling  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `const data = response.data?.data || {}; const deletedCount = data.deleted_count || 0` — 若后端把计数放在 response.data 顶层，这里永远显示"成功删除 0 个邀请码"，误导管理员。
- **建议**: 做双形态取值：`const d = response.data?.data || response.data || {}`，并以后端返回结构为准收敛到一种。

### [LOW] .desktop-only 定义两次，.mobile-filter-form 系 CSS 全部无模板引用
- **位置**: 978-982, 1112-1139  |  **类别**: maintainability  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `.desktop-only` 块在 L978-982 与 L1134-1138 重复定义；`.mobile-filter-form`、`.mobile-filter-input`、`.mobile-filter-select`（L1112-1133、L1227-1239）在模板中没有任何元素使用（移动端实际用的是 mobile-search-input/mobile-action-bar）。
- **建议**: 删除重复的 .desktop-only 与整段 .mobile-filter-form 死样式。

### [LOW] statistics 用 snake_case 键，与代码其余 camelCase 命名不一致
- **位置**: 629-634  |  **类别**: style  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `statistics` reactive 键为 `total_codes/total_relations/total_reward/total_consumption`，同文件其它状态（codeTotal/relationTotal/selectedCodes 等）均为 camelCase，前后端字段映射时易混淆。
- **建议**: 组件内部统一 camelCase（totalCodes/totalRelations/…），仅在发送/接收边界做 snake_case 转换。

### [INFO] 邀请设置加载/保存与 Settings.vue 路径一致，但缺少 min_order_amount 等字段
- **位置**: 547-571  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: inviteSettings 仅含 inviter_reward/invitee_reward，抽屉说明"此设置将应用于所有新生成的邀请码"；若后端新加字段，此页不会展示。属信息级提示，配合上面的复用建议处理。
- **建议**: 与 Settings.vue 的 invite 设置保持同构（同字段集或同一组件）。

## frontend/src/views/admin/Knowledge.vue

### [MEDIUM] 前端直拼 Go sql.NullString JSON（{String, Valid}）泄漏后端序列化细节
- **位置**: 513-516, 492  |  **类别**: architecture  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: saveArticle 把 summary 转成 `{ String: data.summary, Valid: !!data.summary }`，showArticleDrawer 又读 `row.summary?.String`；Promotions 对 description 同样处理。这是把 Go 结构体序列化格式当作 API 契约，后端一旦改用 *string 或 ORM 配置变化，前端即坏。
- **建议**: 后端在 JSON 序列化层对 NullString 做自定义 MarshalJSON（输出纯字符串或 null），前端统一按 string 处理，删掉 {String, Valid} 拼装。

### [MEDIUM] validate()/confirmDelete() 在 try 外 await，取消/校验失败产生未处理 Promise 拒绝
- **位置**: 446-448, 471-475, 507-509, 534-538  |  **类别**: error-handling  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: saveCategory/saveArticle 的 `await catFormRef.value.validate()`、deleteCategory/deleteArticle 的 `await confirmDelete(...)` 都在 try 之外：用户点“取消”时 confirmDelete reject('cancel') 直接成为 unhandled rejection（控制台报错、无任何提示）；校验失败同理。
- **建议**: 把 validate/confirm 移入 try 或包一层 catch 忽略 'cancel'（参照 UserLevels.saveLevel 的 `error !== false` 处理）。

### [LOW] 编辑文章把整行（含 category 嵌套对象、created_at、view_count）展开进表单并回传
- **位置**: 488-504, 513  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: showArticleDrawer 直接 `articleForm.value = { ...row, ... }`，saveArticle 又 `data = {...articleForm.value}` 全量提交，会把 category 对象、view_count、created_at 等只读字段发给后端 update 接口，依赖后端忽略多余字段。
- **建议**: 表单只保留可编辑字段白名单（title/category_id/content/summary/sort_order/is_active）。

## frontend/src/views/admin/Nodes.vue

### [HIGH] resetForm 未重置 description 与 is_recommended，编辑残留带入新建节点
- **位置**: 663-669  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: resetForm 的 Object.assign 缺 description 与 is_recommended：编辑节点 A 后点'添加节点'，表单残留 A 的描述/推荐状态，直接保存把 A 的描述带到新节点。
- **建议**: resetForm 补 `description: '', is_recommended: false`，与 nodeForm 初始定义一致。

### [MEDIUM] 链接解析预览复用 createNode 端点，契约耦合于后端 preview 参数
- **位置**: 628-636  |  **类别**: architecture  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: parseNodeLink 用 createNode({node_link, preview:true}) 实现预览；当前后端支持 preview（node.go:546-548），但预览职责挂在创建端点下，后端改动即可能误创建。
- **建议**: 后端新增独立解析端点（POST /admin/nodes/parse），前端改用它。

### [MEDIUM] 全选状态用当前页长度比较，跨页选中残留时误判
- **位置**: 469-475  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: `selectedNodes.value.length === nodes.value.length` 未限定当前页：翻页后 selection 保留，全选/半选指示失真。
- **建议**: 基于当前页集合计算全选状态，或翻页前清空选择。

### [MEDIUM] types ref 计算后从未被模板使用
- **位置**: 399, 406-411, 433-439, 688  |  **类别**: maintainability  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: loadNodes 收集类型写入 types.value 并导出，但模板类型下拉只用 allNodeTypes，types 从未消费；按当前页收集的类型集也不完整。
- **建议**: 删除 types ref 及相关收集逻辑。

### [LOW] 节点状态映射与 statusMaps.js 重复且缺 maintenance
- **位置**: 670-671  |  **类别**: duplication  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 内联映射 online/offline/timeout，而 statusMaps.js NODE_STATUS_MAP 多 maintenance 分支，maintenance 节点本页显示'未知'。
- **建议**: 改用 getNodeStatusText/getNodeStatusType。

### [LOW] copyNodeLink 无错误处理，非安全上下文 clipboard 抛错
- **位置**: 678-681  |  **类别**: error-handling  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: navigator.clipboard.writeText 在 http 下 rejected，且无条件提示'复制成功'。
- **建议**: 加 try/catch，失败时提示手动复制；或复用 CopyableField 组件。

## frontend/src/views/admin/Orders.vue

### [MEDIUM] 同一页面两种分页/搜索 API 契约（skip/limit vs page/size），依赖后端容错兜底
- **位置**: 949-1021  |  **类别**: architecture  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: loadOrders 向 `/admin/orders` 传 `skip`/`limit`/`search`，而 loadRecharges 向 `/recharge/admin` 传 `page`/`size`/`keyword`。后端 `ParsePagination` 仅在未传 page/size 时用 skip 反推页码（`page==1 && size==10` 时生效），一旦后端默认值变化该契约即断裂；且 `/recharge/admin` 与 `/admin/orders` 前缀风格不一致，易被路由级权限中间件漏掉（当前已确认有 AdminMiddleware 保护）。
- **建议**: 统一为 `page`/`size`/`keyword` 一种参数风格（前端两处、后端 getPagination），并让前端显式传 page/size 而不是依赖 skip 反推。

### [LOW] 用户邮箱列三元表达式两个分支完全相同，属死逻辑
- **位置**: 134  |  **类别**: duplication  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `{{ activeTab === 'orders' ? (scope.row.user?.email || '-') : (scope.row.user?.email || '-') }}` 两个分支完全一样，三元运算无意义，且与第 132 列重复。属于复制粘贴残留。
- **建议**: 直接渲染 `{{ scope.row.user?.email || '-' }}`。

### [LOW] 支付时间列 label 三元表达式两个分支完全相同
- **位置**: 162  |  **类别**: duplication  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `:label="activeTab === 'orders' ? '支付时间' : '支付时间'"` 两分支一致，纯冗余；同时第 164 行该列的值直接输出原始字符串 `scope.row.payment_time/paid_at`，未像详情抽屉那样走 formatDateTime，格式不一致。
- **建议**: label 直接写 '支付时间'；值统一用 `formatDateTime(...)` 或后端统一返回已格式化字符串。

### [LOW] 导出接口失败时错误 JSON 会被当成 CSV 下载
- **位置**: 1366-1386  |  **类别**: error-handling  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: exportOrders 用 responseType: 'blob' 下载，未检查响应 Content-Type；后端返回 401/500 时 blob 内容是错误 JSON，仍被保存为 `orders_export_YYYYMMDD.csv`，用户拿到损坏文件。
- **建议**: 检查 `response.headers['content-type']` 是否含 text/csv；否则解析错误并提示（或直接走 axios 拦截器的 JSON 错误分支）。

### [LOW] selectedOrders 在切换标签/刷新后不清理，批量栏可能展示过期选择
- **位置**: 1024-1050  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: handleTabChange/resetSearch/searchOrders 均不重置 selectedOrders；在订单页勾选后切到充值页，头部仍显示"已选择 N 个订单"并可对上一页数据执行批量操作；数据重载后这些 id 也可能已不存在。
- **建议**: 在 handleTabChange 与数据重载成功后执行 `selectedOrders.value = []` 并 `tableRef?.clearSelection()`。

### [LOW] 未使用的图标导入/注册与死状态 ref
- **位置**: 742-767  |  **类别**: maintainability  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `Operation`、`HomeFilled`、`Filter`、`User`、`Timer` 仅在 import 和 components 注册中各出现一次（共 2 次），模板从未使用；`orders` ref（L782）被赋值（L965/968）但模板只渲染 `allRecords`/`recharges`，属死状态。
- **建议**: 删除未用图标与 `orders` ref，减少打包体积与维护噪音。

### [LOW] handleBulkAction 首个参数 actionType 从未使用；批量接口命名风格不统一
- **位置**: 1320-1363  |  **类别**: maintainability  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `handleBulkAction(actionType, apiPath, confirmMsg)` 中 actionType 三个调用处都传了值但函数体未用；且三个批量端点 `/admin/orders/bulk-mark-paid`、`/admin/orders/bulk-cancel` 与 `/admin/orders/batch-delete` 混用 `bulk-`/`batch-` 前缀（后端路由也如此），增加记忆成本。
- **建议**: 删除 actionType 参数；建议前后端统一为 `batch-*` 或 `bulk-*` 一种前缀。

### [LOW] qrSummary 直接把完整二维码（可能为超大 base64 data URI）渲染进 span
- **位置**: 1245-1249  |  **类别**: performance  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: 充值详情"支付二维码"字段 `qrSummary` 在非"同支付链接"时返回 `String(qrCode)` 全文，若 qrCode 是 `data:image/png;base64,...` 会输出数万字符到 DOM，拖慢渲染并撑爆布局。
- **建议**: 截断显示（如 `qrCode.length > 120 ? qrCode.slice(0, 60)+'…' : qrCode`）或仅显示哈希/前 N 字符。

### [INFO] loadOrders/loadRecharges 共享 loading 且无请求序号守卫
- **位置**: 949-977  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: 快速切换标签或连续搜索时，两个请求并发，先发后至的响应会覆盖新 tab 数据（当前 tab 数据 ref 不同，实际影响有限），但 loading 状态由后完成者决定，偶发遮罩提前消失。
- **建议**: 引入自增请求 token（`const reqId = ++seq; if (reqId !== seq) return`）或 AbortController 取消过期请求。

## frontend/src/views/admin/Packages.vue

### [HIGH] saveCustomPackageSettings 循环内串行 await 6 次网络请求，非原子且易部分失败
- **位置**: 833-865  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: for 循环内逐个 await configAPI.updateSystemConfig 发送 6 个 PUT：耗时 6×RTT，中途失败留下部分配置已更新、部分未更新的不一致状态。
- **建议**: 改为 Promise.allSettled 并行提交，汇总部分失败提示；优先使用后端批量配置接口。

### [MEDIUM] editPackage 用关键词判断描述是否自动生成，手写描述含关键词会被误判并覆盖
- **位置**: 640-667  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: `autoGeneratedKeywords.some(kw => descriptionValue.includes(kw))` 启发式判断，用户手写描述含'解锁流媒体'等词时被置为自动生成，改价格/时长后描述被覆盖。
- **建议**: 由后端返回自动生成标记，或仅当描述与自动生成模板完全一致时视为自动生成。

### [MEDIUM] resetForm 用 setTimeout(100ms) 触发 autoGenerateDescription，存在竞态
- **位置**: 704-724  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 100ms 魔法延时内用户输入会被自动生成的描述覆盖。
- **建议**: 去掉 setTimeout，清空 description 后同步调用 autoGenerateDescription()；如需异步用 nextTick。

### [LOW] @input 防抖与 @clear 立即搜索并存，清空输入时重复请求
- **位置**: 24-26, 71-73  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: el-input 清空同时触发 input 与 clear 事件：立即请求一次 + 500ms 后再请求一次。Nodes.vue（74-75）与 SystemLogs.vue（65-66）同模式。
- **建议**: @clear 仅重置不请求，由随后的 @input 防抖统一触发；或先 cancel 防抖再手动搜索。

### [LOW] 未使用的导入 HomeFilled 与 watch
- **位置**: 481, 495, 479  |  **类别**: maintainability  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: HomeFilled 导入并注册但模板未用；watch 导入未在 setup 使用。
- **建议**: 删除。

### [LOW] 价格直接插值未格式化，浮点精度可能显示异常
- **位置**: 134-138, 541-560  |  **类别**: ux  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: `¥{{ row.price }}` 直接输出 float64，可能显示浮点误差，与 Dashboard 的 formatMoney 不一致。
- **建议**: 统一用 utils/format.js 的 formatMoney。

## frontend/src/views/admin/PaymentConfig.vue

### [HIGH] 易支付 MD5/RSA 签名切换时密钥字段映射错乱，可能把 RSA 私钥当 MD5 key 提交
- **位置**: 746-762, 821-863  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: editConfig 把 config_json.merchant_private_key 同时回填 merchant_private_key 与 yipay_merchant_private_key；用户编辑 RSA 配置后切 MD5，saveConfig MD5 分支把表单里的 RSA 私钥当 MD5 key 提交，回调签名必失败。
- **建议**: MD5/RSA 分别绑定独立字段，editConfig 按 sign_type 只回填对应字段，buildRequestData 不再交叉赋值。

### [HIGH] 支付配置列表接口明文返回全部密钥并被编辑弹窗回显
- **位置**: 821-863  |  **类别**: security  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 后端 GetPaymentConfig（admin.go:978-1035）原样下发 MerchantPrivateKey/AlipayPublicKey/WechatAPIKey/PaypalSecret/StripeSecretKey/AccountNumber 等密钥，前端全量存内存并在 editConfig 回显到输入框；且 wechat 密钥输入框（441-445）无密码掩码。
- **建议**: 后端列表/详情接口对密钥脱敏（掩码或 has_key 标记）；前端编辑时密钥留空表示'不修改'，仅重新输入才提交。

### [MEDIUM] 20+ 表单字段的手写双向映射（buildRequestData/editConfig 各 6 分支）极易漂移
- **位置**: 537-561, 721-794  |  **类别**: architecture  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: DEFAULT_FORM_STATE 20+ 字段，正向/反向映射手写 if-else 各 6 分支，已出现字段含义不一致（merchant_private_key 在 yipay/codepay 分支不同），是密钥串用 bug 的根因。
- **建议**: 用 schema 驱动：每种 pay_type 定义字段描述表，保存/回填遍历 schema，补 6 种类型 round-trip 单测。

### [MEDIUM] handleApiError 用 message.includes('cancel') 吞掉错误，判定脆弱
- **位置**: 578-585  |  **类别**: error-handling  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 任何含 'cancel' 字符串的错误消息（如'订单已取消'）都会被当作用户取消而静默不提示。
- **建议**: 改为显式判断 error === 'cancel' || error === 'close'。

### [MEDIUM] loadPaymentConfigs 硬编码 { page: 1, size: 100 }，超 100 条截断且无分页
- **位置**: 697-720  |  **类别**: performance  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 后端默认 size 也仅 100（admin.go:955-967），无分页 UI，配置多时静默截断，批量操作基于截断集合。
- **建议**: 接入 PaginationBar 分页或 pay_type 过滤。

### [LOW] 手写 checkMobile/resize 监听与 useMobile composable 重复
- **位置**: 667, 947-953  |  **类别**: duplication  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 自实现 window.innerWidth<=768 监听且无节流，项目已有 useMobile（rAF 节流+自动清理）。
- **建议**: 改用 useMobile()。

## frontend/src/views/admin/Profile.vue

### [MEDIUM] loadSecuritySettings 解包层级错误：设置永远加载不出来
- **位置**: 599-612  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `const data = response.data || response`，而 adminAPI.getSecuritySettings() 返回 axios 响应，response.data 是 {success, data:{...}} 信封，data.login_notification/data.session_timeout 均为 undefined → Object.assign 全部落回默认值（login_notification=true、session_timeout='120'），用户已保存的设置每次进入页面都被默认值覆盖展示。
- **建议**: 改为 `const data = response.data?.data ?? response.data ?? response`，并补 response.data?.success 判断。

### [MEDIUM] uploadUrl 硬编码 '/api/v1' 前缀，与 axios baseURL 重复维护
- **位置**: 266  |  **类别**: maintainability  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `const uploadUrl = '/api/v1/admin/upload'` 把 BASE_URL（api.js:169 定义 '/api/v1'）写死；el-upload 不走 axios，一旦 BASE_URL 变更（如改 '/api' 或加环境前缀），头像上传静默 404。
- **建议**: 从 utils/api.js 导出 BASE_URL 常量并拼接：`uploadUrl = `${BASE_URL}/admin/upload``。

### [LOW] 内网判定把 172.0-15.x 也当内网
- **位置**: 625-638  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `ipAddress.startsWith('172.')` 会把公网 172.16 之外（如 172.32.x）也算内网；正确私有段是 172.16.0.0/12。
- **建议**: 用正则或 IP 解析判断 172.16–172.31，或引入 is-private-ip 工具。

## frontend/src/views/admin/Promotions.vue

### [MEDIUM] save()/remove() 中 validate/confirmDelete 在 try 外，取消与校验失败产生 unhandled rejection
- **位置**: 437-445, 473-477  |  **类别**: error-handling  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `await formRef.value.validate()`（439）与 `await confirmDelete('活动', 1, ...)`（474）都在 try 外：用户取消删除时 confirmDelete 抛 'cancel'，成为未处理的 Promise 拒绝，控制台报错且无提示（与 Knowledge 同一模式）。
- **建议**: 把确认/校验纳入 try-catch，catch 中忽略 'cancel'，参照 UserLevels.deleteLevel 的写法。

### [MEDIUM] 新建活动时间用 toISOString 而日期选择器 value-format 是 'YYYY-MM-DD HH:mm:ss'，格式自相矛盾
- **位置**: 417-432, 454-455  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: showDrawer 新建时 `start_time: tomorrow.toISOString()`（ISO 带时区），但 el-date-picker 声明 `value-format="YYYY-MM-DD HH:mm:ss"`——v-model 初始为 ISO 字符串时选择器可能无法正确回显；保存时又 `new Date(data.start_time).toISOString()` 转回 ISO，前端在两种格式间来回倒。
- **建议**: 统一为一种格式：要么选择器 value-format 与 ISO 一致，要么表单默认值用 dayjs 格式化为 'YYYY-MM-DD HH:mm:ss'，提交时不再二次转换。

### [LOW] description 同样拼 Go NullString {String, Valid}
- **位置**: 449-452  |  **类别**: architecture  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 与 Knowledge 相同：`data.description = { String: data.description, Valid: !!data.description }`，把后端序列化细节写死在前端。
- **建议**: 同 Knowledge 建议：后端自定义 NullString JSON 序列化后，前端删除此转换。

## frontend/src/views/admin/Settings.vue

### [HIGH] el-upload 上传 Logo 不携带 Authorization，必然 401
- **位置**: 40-49, 1088  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `uploadUrl = '/api/v1/admin/upload'` 走 el-upload 自带 XHR，而登录态是 axios 请求拦截器（api.js L310-312）注入的 `Authorization: Bearer`，el-upload 的原生请求完全绕开 axios，不会带该头；已确认后端 AuthMiddleware 只解析 `Authorization` 头（internal/middleware/auth.go L90-104，缺失直接 401），且未配置 cookie/session 兜底。因此 Logo 上传功能实际不可用。
- **建议**: 给 el-upload 加 `:headers="{ Authorization: 'Bearer ' + token }"`（从 secureStorage 取），或改用 `:http-request` 自定义函数经 axios 实例上传。

### [MEDIUM] 2202 行 god 组件：11 个设置 Tab 全部内联在单文件
- **位置**: 20-971, 977-1717  |  **类别**: architecture  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: 模板 971 行 + 脚本 740 行 + 样式 480 行，基本设置/注册/邀请/通知/公告/主题/节点监控/安全/备份/协议过滤/仓库同步各自的状态、加载、保存、轮询逻辑全部堆在同一 setup()，任何一处改动都牵动全局，且大量响应格式嗅探分支（loadSettings 中 notification/admin_notification 的 legacy 回填）难维护。
- **建议**: 按 tab 拆分为子组件（如 SettingsGeneral.vue / SettingsNotification.vue …），父组件只负责 tab 容器与 saveCurrentTab 调度；每个子组件内聚自己的表单与保存函数。

### [MEDIUM] beforeLogoUpload 用 `ElMessage.error(...) && false` 返回 undefined，校验形同虚设
- **位置**: 1691-1695  |  **类别**: logic  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `ElMessage.error()` 返回 undefined，`undefined && false` 求值为 undefined；Element Plus 的 before-upload 仅对严格 `false` 或 rejected Promise 阻止上传，因此非图片/超 2MB 文件不会被拦截。
- **建议**: 改为 `if (cond) { ElMessage.error('…'); return false }` 显式返回 false。

### [LOW] loadLocalBackups 假定响应必为数组，形态不符时 sort 抛异常
- **位置**: 1601-1608  |  **类别**: error-handling  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `(res.data?.data || res.data || []).sort(...)` — 若后端返回 `{data:{files:[…]}}` 这类嵌套结构，res.data.data 是对象，`.sort is not a function` 抛错被 catch 后仅提示"加载本地备份列表失败"，功能静默失效。
- **建议**: 显式判型：`const list = Array.isArray(res.data?.data) ? res.data.data : Array.isArray(res.data) ? res.data : []`，再 sort。

### [LOW] uploadStatusInterval 变量从未赋值，仅被清空
- **位置**: 1105, 1545  |  **类别**: maintainability  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: 轮询实际由 usePaymentStatusPolling 组合式函数管理，`uploadStatusInterval` 声明后从未 setInterval，checkUploadStatus 里的 `if (uploadStatusInterval.value) { clearInterval(...) }` 是恒假分支。
- **建议**: 删除 uploadStatusInterval 声明及对应清理分支。

### [LOW] formLayout 与 customerMethodSwitches 未被模板使用
- **位置**: 1127-1130, 1007-1009  |  **类别**: maintainability  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: `formLayout` computed 与 `customerMethodSwitches` 数组均被 return 但模板无任何引用（模板各处硬编码 label-position="top"，客户事件矩阵直接用 customerEventSwitches）。
- **建议**: 删除这两项死代码（formLayout 若计划用于移动端布局，请实际接入模板）。

### [LOW] 公告内容允许任意 HTML，若用户端 v-html 渲染存在存储型 XSS 面
- **位置**: 530-532  |  **类别**: security  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: 公告内容 `announcement_content` 标注"支持完整的 HTML 格式排版"并原样存储。虽然写入者是管理员，但一旦公告区被低权限角色或注入利用，且前端渲染用 v-html，即构成存储型 XSS 面（可窃取用户 token）。
- **建议**: 服务端做白名单 HTML 清洗（如 bluemonday）；渲染端确认公告组件对内容做 sanitize 而不是裸 v-html。

### [INFO] GeoIP 更新/切换在失败后仍刷新状态且 loading 不收敛于 try/finally
- **位置**: 1406-1417  |  **类别**: error-handling  |  **来源组**: F8-admin-views-2 (管理端 Orders/Settings/CustomNodes/Invites)
- **问题**: updateGeoIPDatabase/switchDatabase 依赖 handleSave 内部 catch 不抛异常才不中断，成功后无条件 `await loadGeoIPStatus()`；若未来 handleSave 抛错，geoipUpdating/switchingDatabase 将永久卡在 true。
- **建议**: 包一层 try/finally 确保 loading 复位，并按 handleSave 返回值决定是否刷新状态。

## frontend/src/views/admin/Statistics.vue

### [MEDIUM] region 图表初始化用 setTimeout 轮询 10 次×200ms 的脆弱 hack
- **位置**: 492-500  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: tryInit 最多轮询 10 次等待 canvas，超过 2 秒静默失败且无提示。
- **建议**: 改为 canvas ref 就绪时触发（nextTick+单次兜底）或 watchEffect。

### [MEDIUM] Chart.js 实例管理不一致且无卸载清理，存在内存泄漏
- **位置**: 396-477, 511-580  |  **类别**: performance  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: initUserChart/initRevenueChart 每次 new Chart 不销毁旧实例，且无 onUnmounted 销毁；仅 regionChart 有 destroy。keep-alive 反复激活会叠加泄漏。
- **建议**: 统一保存三个实例引用，重绘前 destroy，onUnmounted 全部销毁。

### [MEDIUM] 模板与 script 缩进混用 tab/空格，且与其他五个文件风格割裂
- **位置**: 224-320, 324-608  |  **类别**: style  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: region 模板段与整个 <script> 用 tab，其余 2 空格；另 .statistics-admin-container（611 行）为未使用的死选择器。
- **建议**: 统一 2 空格缩进，删除死选择器，接入 eslint/prettier。

### [LOW] formatMoney 与 Dashboard/utils/format.js 三处重复实现
- **位置**: 478-483  |  **类别**: duplication  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 本文件与 Dashboard.vue 各实现一份，utils/format.js:1 已有同功能实现。
- **建议**: 统一 import { formatMoney } from '@/utils/format'。

### [LOW] 大量未使用 CSS：region-detail-*、recent-activities、activity-* 等
- **位置**: 870-947  |  **类别**: maintainability  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: .region-detail-card/.detail-*/.recent-activities/.activity-* 等在模板中无对应元素。
- **建议**: 删除未命中模板的 CSS 块。

## frontend/src/views/admin/Subscriptions.vue

### [HIGH] “以用户身份登录”流程把仿冒会话令牌写入 sessionStorage/localStorage，且未限制仿冒管理员
- **位置**: 1399-1597  |  **类别**: security  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: goToUserBackend 调用 loginAsUser 换取用户 access_token，随后 persistUserBackendSession（1502-1512）把令牌明文写入 secureStorage 的 user_token/user_data（1504-1508 行），同时把含 token 的完整 sessionData 写入 sessionStorage 与 localStorage（1563-1564 行，TTL 5 分钟）。任何用户端 SPA 的 XSS 都能直接读走该令牌并持久持有仿冒会话；客户端也未限制目标用户（1539 行仅传 userId，可仿冒另一管理员）。整个流程无审计日志提示。
- **建议**: 优先改为后端发放短期有效、绑定设备指纹的仿冒 token 并以 HttpOnly Cookie 下发（前端零存储）；至少在客户端限制 targetUser.is_admin 并提示；在 admin 审计日志记录 login-as 行为；缩短 localStorage handoff TTL 并确保成功消费后立即删除。

### [HIGH] downloadQRCode 跨域图片无法触发下载，点击会让当前页导航到二维码图片
- **位置**: 1160-1165  |  **类别**: ux  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: downloadQRCode 直接 a.href = api.qrserver.com 的 URL + link.download。对跨域 URL，浏览器忽略 download 属性并回退为导航——在当前标签页打开二维码图片，把管理员所在的订阅列表页整个替换掉（页面丢失、需重新进入）。
- **建议**: 改为 fetch(currentQRCode) → blob → URL.createObjectURL 再触发同源 blob 下载；或弃用外链图片，用 qrcode 库本地生成 dataURL/blob（同时解决隐私问题）。

### [MEDIUM] 与 Users.vue 大量逐字重复：线路模式三件套、设备/重置 map、复制、备注自动保存
- **位置**: 978-1014, 1325-1376, 1377-1398, 1736-1771  |  **类别**: duplication  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: getUserLineMode/hasAssignedCustomNodes/updateUserLineMode/getLineTypeTagText/Type（978-1014）与 Users.vue（1109-1144）几乎逐字相同；getResetTypeTag/Text、getResetByTag/Text、getDeviceTypeTag/Text（1325-1376）与 Users.vue（1540-1575）完全重复；copyToClipboard（1377-1398）与 Users.vue（1576-1597）、UserDetailDialog（1243-1254）三处复制；备注自动保存（1736-1771，saveTimers/originalNotes/savedIndicatorTimers 三件套）与 Users.vue（1293-1341）同构。任何一处修 bug（如文案/重试逻辑）都要改三份。
- **建议**: 抽公共模块：@/composables/useUserLineMode.js、@/utils/clipboard.js、@/composables/useNotesAutosave.js，并把设备/重置状态 map 收进 @/utils/statusMaps.js 或新 deviceMaps.js。

### [MEDIUM] loadSubscriptions 缺少过期响应保护，快速排序/搜索会产生乱序覆盖
- **位置**: 890-936  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 与 Users.vue 的 loadUsersSeq 序号守卫不同，本组件连续点击排序（handleSortCommand/sortByApple 等）或输入搜索时，多个并发请求可能乱序返回，后发先至的旧响应会覆盖新数据；loading 状态也可能提前熄灭。
- **建议**: 仿照 Users.vue 引入请求序号（let seq=0; const s=++seq; ... if (s!==seq) return），或使用 AbortController 取消过期请求。

### [MEDIUM] updateSubscription 三处调用均不检查 {success:false} 响应，UI 与后端可能静默分叉
- **位置**: 1039-1050, 1077-1091, 1655-1666  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: updateExpireTime/updateDeviceLimit/toggleSubscriptionStatus 只 try/catch 网络错误，不校验 response.data?.success === false（后端 2xx + success:false 时）。toggleSubscriptionStatus 在 1661 行无条件把本地 is_active 翻成新值，后端未生效时 UI 显示已切换；updateExpireTime/updateDeviceLimit 失败则静默显示成功提示。同文件其它操作（如 createUser、batchDelete）都校验 success，此处不一致。
- **建议**: 统一模式：if (response.data?.success === false) throw new Error(response.data.message) 后再更新本地状态/提示成功。

### [MEDIUM] “清理设备数”只清理当前页订阅，与按钮名/用户预期不符
- **位置**: 1715-1735  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: clearAllDevices 用 subscriptions.value（仅当前页，pageSize 10-100）收集 subscription_ids 调 batchClearDevices，标题却叫“清理设备数”，无“仅当前页”提示；若有上千订阅，管理员以为一键全清，实际只清了一页。
- **建议**: 要么调用不带分页的批量清理接口（确认后端支持全量），要么把按钮文案与确认框明确为“清理当前页（N 条）订阅的设备”。

### [MEDIUM] 大块死代码：设备管理/重置 map/截断工具全部未被模板使用，含失效 import
- **位置**: 863-865, 873-889, 951-960, 1230-1324, 1325-1376, 1838-1855, 2078-2172  |  **类别**: maintainability  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 模板没有设备表格（设备管理在 UserDetailDialog 内），但本组件仍定义并 return：userDevices/loadingDevices/deletingDevice/loadUserDevices/deleteDevice（863-865、1230-1324，其中 1237 行同样把用户 id 当订阅 id 传入 getSubscriptionDevices）、getDeviceType*/getResetType*/getResetBy*（1325-1376）、truncateUserAgent/formatTime/formatLocation/truncateUrl（1838-1855）、clearSort/currentSortText/handleStatusFilter/getStatusFilterText（873-889、951-960）；import 里 HomeFilled/User/Filter（770-772 行）无模板引用。
- **建议**: 逐一删除上述未使用项及其 return/import；若列表页确需设备能力，改由 UserDetailDialog 提供或抽公共组件。

### [MEDIUM] 订阅令牌经查询参数发给第三方 api.qrserver.com，且逐行请求
- **位置**: 1112-1153, 922  |  **类别**: security  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: generateQRCode（1112-1153）把 sub://<btoa(订阅地址)>#... 作为 data 查询参数拼进 https://api.qrserver.com/v1/create-qr-code/?...，并作为 qr_code_url 在 loadSubscriptions 里对每一行预生成（922 行）。订阅地址内含用户专属 token（秘密凭证），等于把每个订阅令牌明文泄露给第三方且大概率进其访问日志；每页 10-100 行 = 10-100 个外部请求（隐私+性能双问题）。此外 btoa(universalUrl)（1123 行）遇到非 Latin-1 字符会抛异常，而它发生在列表 map 内——一条坏 URL 会打崩整个列表加载。
- **建议**: 改用本地生成（如 qrcode npm 包生成 dataURL 或同源 /qr 端点），彻底移除对 api.qrserver.com 的依赖；btoa 前先做字符校验或 try/catch 降级为不生成二维码。

### [LOW] 到期时间编辑被截断为日期（YYYY-MM-DD），丢失时分秒语义
- **位置**: 302-318, 1039-1050  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 桌面列内日期选择器 value-format="YYYY-MM-DD"（307 行）、移动端同样（584 行），addTime 也按 'YYYY-MM-DD' 写回（1070 行）；而 Users.vue 展示的是带时间的 datetime。同一订阅的到期时间经此编辑后时间部分被清零，跨页面展示不一致，按天+时精确计费的场景会失真。
- **建议**: 统一为 datetime 精度（value-format="YYYY-MM-DDTHH:mm:ss"），或至少在两处页面使用同一格式化与粒度。

### [LOW] searchQuery 与 searchForm.keyword 双源冗余，存在残留旧关键词的路径
- **位置**: 893-899, 937-942  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: loadSubscriptions 里 if (searchForm.keyword && !searchQuery.value) searchQuery.value = ... 后 search: searchForm.keyword || searchQuery.value（893-899）。若在不清空 searchQuery 的情况下直接改 keyword（例如从路由 query 初始化后用户清空输入但没触发 @clear/@input 的边界），请求会继续携带旧关键词；双状态容易漂移。
- **建议**: 删除 searchQuery，loadSubscriptions 直接用 searchForm.keyword；路由 query 初始化也写入 searchForm.keyword 即可。

### [LOW] 移动端用户 ID 回退链包含 subscription.id，显示错误 ID
- **位置**: 554  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: ID: {{ subscription.user?.id || subscription.user_id || subscription.id }}——当 user 缺失时回退到订阅 id，把订阅主键当成用户 ID 展示，误导排查。
- **建议**: 去掉 || subscription.id 回退，用户信息缺失时显示 'ID 未知' 或 '-'。

### [LOW] handleSortChange 清排序时不与表格内部排序指示同步
- **位置**: 2035-2046  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: order 为空时只重置 currentSort='add_time_desc' 并重新加载，el-table 自己记录的排序箭头仍停留在旧列/旧方向，服务端结果与表头指示不一致；且 created_at→add_time 的字段名映射只处理了一列，其它 prop 直接拼进 sort key，一旦列 prop 与后端 sort 字段不一致会静默按错误字段排序。
- **建议**: 清排序时同步重置 tableRef 的 sort 状态（tableRef.value?.clearSort?.()），并把 prop→sortKey 映射表显式声明。

### [LOW] 列设置 watch 在强制重置时仍会写入 localStorage，造成回跳
- **位置**: 826-861, 2056-2062  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: watch(visibleColumns) 在 newColumns.length === 0 时把值重置为 ['qq','actions'] 并 return（不保存），但重置本身再次触发 watch（数组变化），此时长度非 0 会正常保存——结果“全不选”最终仍被持久化为 ['qq','actions']，下次打开还是这两列，与“全不选”意图不符。
- **建议**: 在 clearAllColumns 里直接写入默认两列（不依赖 watch 重置），或在 watch 中跳过由重置产生的写入（比较新旧值）。

### [LOW] 缩进错乱与静默空 catch
- **位置**: 927-928, 1098-1099, 851-852, 858-859  |  **类别**: style  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 927-928 行 total.value = ... 缩进多了一层、else 分支对齐异常；1098-1099 行 catch 内 } 缩进错位；851-852 与 858-859 两个空 catch 静默吞掉 JSON.parse/localStorage 异常，无任何降级日志。
- **建议**: 统一格式化（项目若有 prettier/eslint 请跑一遍）；空 catch 至少补一行注释或 console.debug。

## frontend/src/views/admin/SystemLogs.vue

### [MEDIUM] 桌面与移动端筛选字段集不一致
- **位置**: 11-101, 103-175  |  **类别**: architecture  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 桌面含 task_type 无 module/username，移动含 module/username 无 task_type，同一页面设备间筛选能力不同。
- **建议**: 两套表单共用同一字段集合，仅做响应式布局差异。

### [MEDIUM] 日志详情'堆栈跟踪'区块引用后端不存在的 stack_trace 字段，永不显示
- **位置**: 400-403  |  **类别**: architecture  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 后端 formatAuditLogForAPIWithUsername（logs.go:445-458）无 stack_trace，前端 v-if 区块永不渲染；后端返回的 additional_info 前端又未展示。
- **建议**: 前端移除 stack_trace 区块或改展示 details/additional_info。

### [MEDIUM] 关键词输入每次防抖触发同时查询列表与统计两个接口
- **位置**: 538-542, 544  |  **类别**: performance  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: applyFilter 同时 loadLogs+loadLogsStats，输入停顿即发两个请求，GetLogsStats 是多次聚合 Count 查询，高频空转。
- **建议**: 统计仅在选择/重置/切页时刷新，输入防抖只刷列表。

### [LOW] formatDate 对空格分隔时间在 Safari 上返回 'Invalid Date'
- **位置**: 653-657  |  **类别**: logic  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: new Date('2006-01-02 15:04:05') 在 Safari 解析失败（与 Dashboard.formatTimeAgo 同源）。
- **建议**: 用 dayjs 按 'YYYY-MM-DD HH:mm:ss' 直接格式化，避免 new Date 解析。

### [LOW] exportLogs 无 loading/防重复点击
- **位置**: 559-593  |  **类别**: ux  |  **来源组**: F9-admin-views-3 (管理端 Dashboard/Packages/Nodes/PaymentConfig/Statistics/SystemLogs)
- **问题**: 导出无进行中反馈，后端最多导出 10000 条（logs.go:678），期间可重复点击。
- **建议**: 导出期间设 exporting ref 并绑定 :loading。

## frontend/src/views/admin/Tickets.vue

### [HIGH] 备注对话框不预填已有 admin_notes，空白保存会清空原备注
- **位置**: 546-562, 373-378  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: updateNotes 直接把 adminNotes.value 发给后端，而 adminNotes 只在关闭时清空（566-568），打开“添加备注”对话框时从不从 currentTicket.admin_notes 预填；工单已有备注时，管理员点开对话框直接保存即用空字符串覆盖原备注。
- **建议**: showNotesDialog=true 前执行 `adminNotes = currentTicket.admin_notes || ''`；updateNotes 增加内容非空校验（空则提示或跳过）。

### [MEDIUM] 关键词防抖搜索不重置页码
- **位置**: 459-460, 434-458  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: debouncedLoadTickets = debounce(loadTickets, 500)，loadTickets 直接以当前 pagination.page 请求；用户在第 5 页输入关键词搜索，仍请求第 5 页（很可能为空列表），而 Coupons 的 debouncedSearchCoupons 会先 page=1。
- **建议**: 防抖回调内先 pagination.page = 1 再 loadTickets（与 Coupons 对齐）。

### [MEDIUM] getTypeTagType 恒返回 'info'，类型颜色映射是死代码
- **位置**: 611-614  |  **类别**: maintainability  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `const getTypeTagType = (type) => { if (!type) return 'info'; return 'info' }` 无论 technical/billing/account/other 都返回 info，参数未使用，明显是未完成的映射。
- **建议**: 补上如 { technical:'warning', billing:'danger', account:'primary', other:'info' } 的真实映射，或直接删除该函数改用 statusMaps 中统一 map。

### [LOW] Search 图标导入未使用
- **位置**: 393  |  **类别**: maintainability  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `import { Refresh, Search } from '@element-plus/icons-vue'`，模板里只有 Refresh，Search 从未使用。
- **建议**: 删除 Search 导入。

### [LOW] 每次查看工单详情都全量重载列表
- **位置**: 470-483  |  **类别**: performance  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: viewTicket 打开抽屉后 `await loadTickets()` 重拉整页列表，仅为了本地清掉 unread 徽标；同时 markTicketReadLocally 已在本地改状态，又 dispatch window CustomEvent('ticket-viewed')，刷新+事件双通道重复。
- **建议**: 本地 markTicketReadLocally 后不必再 loadTickets；如需与服务端同步，只调一次且去掉自定义事件。

## frontend/src/views/admin/UserLevels.vue

### [MEDIUM] 桌面表格 scope.row.min_consumption.toFixed(2) 对 null/字符串直接抛 TypeError
- **位置**: 103-107  |  **类别**: error-handling  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `¥{{ scope.row.min_consumption.toFixed(2) }}`（105 行）没有空值保护；移动端卡片却用了 `level.min_consumption?.toFixed(2) || '0.00'`（155 行）防御。若后端返回 null 或字符串数字，桌面表格整列渲染崩溃。
- **建议**: 统一抽 formatMoney(value) 工具（如 `Number(value ?? 0).toFixed(2)`），桌面与移动端共用。

### [MEDIUM] 浏览器代码直接访问 process.env.NODE_ENV（vite.config 未 define），运行时可能 ReferenceError
- **位置**: 385-386, 407-409, 493-494, 515-517, 540-542  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: loadLevels/saveLevel/deleteLevel 中多处 `process.env.NODE_ENV === 'development'`，其中 385-386 与 493-494 是空 if 块。Vite 客户端构建默认不注入 process（vite.config.js 无 define 配置），浏览器里 process 未定义会抛 ReferenceError——在 loadLevels 中该异常会被 catch 捕获并弹“加载等级列表失败”，页面可能直接失效；另两处空块是死代码。
- **建议**: 全部替换为 Vite 原生 `import.meta.env.DEV` / `import.meta.env.PROD`，并删除两个空 if 块；如需保留可用 define 注入。

### [LOW] loadLevels 对响应做 5 分支猜测式解包
- **位置**: 388-401  |  **类别**: maintainability  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: response.data.data.levels / response.data.success&&Array / Array.isArray(response.data) / response.data.levels / 空 五层判断，说明前后端契约不明确、靠猜。同一模式出现在 AbnormalUsers（3 分支）等多处。
- **建议**: 与后端约定统一 envelope {success, data:{levels,total}}，前端只取一层并加类型守卫，删除猜测分支。

## frontend/src/views/admin/Users.vue

### [HIGH] 用户详情响应结构三处访问路径不一致：selectedUser.value.user.id 与 user_info.id 混用
- **位置**: 1428, 1485-1491  |  **类别**: architecture  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: viewUserDetails 把 getUserDetails 的 data 直接赋给 selectedUser，而 UserDetailDialog 内部统一用 user.user_info?.id || user.id、user.user_info?.username 取值（UserDetailDialog.vue 14-16 行）。但 Users.vue 的 loadUserCustomNodes（1428 行）、assignCustomNode（1485-1491 行）却用 selectedUser.value.user.id。若详情接口返回 { user_info: {...}, subscriptions: [...] } 而没有顶层 .user，则这两个函数永远走进“用户信息不存在”分支，专线分配功能整体不可用。
- **建议**: 统一取 id 的辅助函数（如 getUserId(data) = data?.user_info?.id || data?.id || data?.user?.id），三处（Users.vue、UserDetailDialog.vue、Subscriptions.vue）共用；并核对后端 /admin/users/:id/details 的实际响应字段。

### [HIGH] device_overlimit 筛选在客户端过滤导致分页/总数全部错误
- **位置**: 1196-1210  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: loadUsers() 先用 status=device_overlimit 请求后端（后端按 keyword/status 分页返回），拿到当前页数据后再 userList.filter(user => isDeviceOverlimit(user))，并把 total.value 设为过滤后的长度（最多等于 pageSize）。后果：第 2 页及以后页里超限的用户永远搜不出来；当第 1 页恰好没有超限用户时，筛选结果显示空列表且总数恒 ≤ 每页条数；device_overlimit 与真实总数据量无关。
- **建议**: 把 device_overlimit 作为独立查询条件下沉到后端（如 status=device_overlimit 由服务端统计设备数后再分页），前端删除过滤逻辑；至少也要改为循环翻页聚合后再分页（不推荐）。

### [HIGH] “分配专线节点”对话框的节点下拉永远为空：loadAvailableNodes 从未被调用
- **位置**: 1448-1457, 763  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 模板第 763 行在无搜索关键词时取 availableNodes，但 availableNodes 唯一的赋值入口 loadAvailableNodes()（1448-1457 行）在整个组件里从未被调用（只有定义和 return）。打开对话框后节点下拉框为空，必须手动搜索才有数据，首次使用即表现为坏按钮。
- **建议**: 在 showAssignNodeDialog 打开时（watch 或按钮 handler 内）调用 loadAvailableNodes()；或删除 availableNodes 分支，统一复用 handleNodeSearch()（空关键词时拉取全部）。

### [MEDIUM] 编辑表单存在竞态：异步 getUserDetails 可能在保存后回填旧值
- **位置**: 1000-1035  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: watch(editingUser) 中先同步填充 userForm，再 await adminAPI.getUserDetails(user.id) 用详情覆盖 device_limit/expire_time。若用户在详情请求返回前就点了“更新”，或在请求期间改动了这两个字段，迟到的响应会把已提交/已修改的值覆盖回表单，造成“保存后 UI 显示被改回、实际提交的是旧值”的错乱。
- **建议**: 给详情请求加序号/AbortController（类似 loadUsersSeq），响应回来时校验 dialog 仍打开且用户未变化；或移除该 fetch，直接使用列表行数据（列表已含 subscription.device_limit / expire_time）。

### [MEDIUM] viewUserBalance 意图未实现：activeBalanceTab/detailActiveTab 是死状态
- **位置**: 1361-1365, 867-868  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: viewUserBalance 设置 activeBalanceTab/detailActiveTab 后调用 viewUserDetails，但详情弹层是 UserDetailDialog，其内部 tab 由自身 activeTab（默认 initialTab='orders'）驱动，这两个 ref 在模板中从未被引用，也没有传给 dialog。因此点击余额只会打开详情抽屉且停在“订单记录”Tab，不会切到“充值记录”。
- **建议**: 把目标 tab 作为 prop 传给 UserDetailDialog（initial-tab="recharge"）并让 dialog 在 visible 时应用，或删除这两个无用 ref 和多余赋值。

### [MEDIUM] 大块死代码：内联设备/专线管理逻辑与辅助 map 从未被模板引用
- **位置**: 1366-1447, 1540-1601, 1154  |  **类别**: maintainability  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 模板中设备、专线节点管理已全部交给 UserDetailDialog（组件内自管 devices/customNodes），但 Users.vue 仍保留并 return 了 userDevices/loadingDevices/deletingDevice/deleteDevice/loadUserDevices/loadUserCustomNodes/unassignCustomNode（1366-1447）以及 getDeviceTypeTag/Text、getResetTypeTag/Text、getResetByTag/Text、getOrderStatusType/Text、getPaymentMethodText、copyToClipboard、formatLocation（1540-1601）等，全部无模板引用；resizeTimer（1154 行）从未被赋值，onUnmounted 里 clearTimeout 一个恒为 null 的变量。注意 loadUserDevices（1373-1374 行）还把用户 id 当作订阅 id 传给 getSubscriptionDevices，属潜伏 bug，随死代码一并清理。
- **建议**: 删除上述未使用状态/函数及其 return 项；如确需在列表页展示设备，应复用 UserDetailDialog 或抽公共 composable。

### [MEDIUM] 挂载时重复请求 + 日期变更双触发：每次搜索多发出请求
- **位置**: 1270-1279, 1249-1256, 1797-1799  |  **类别**: performance  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: watch(() => searchForm.date_range, debounce(...,300), { immediate: true }) 在 setup 阶段立即执行一次 searchUsers()，随后 onMounted 又调用 loadUsers()，组件挂载即发 2 个相同请求（靠 loadUsersSeq 丢弃一个，但网络请求已发出）。同样，移动端 handleDateRangeChange（1249-1256）设置 start/end 后立即 searchUsers，而 watcher 也会把 date_range→start/end 后再 searchUsers 一次，一次日期变更产生 2 次搜索。
- **建议**: 移除 immediate:true 或 onMounted 中的重复 loadUsers；日期变更只保留一条链路（要么 handler 直接搜，要么只靠 watcher），二选一。

### [MEDIUM] 单行删除/禁用绕过 checkAdminUsers 管理员保护（越权/自锁风险）
- **位置**: 1602-1640, 1725-1732  |  **类别**: security  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 批量删除（1738 行）与批量禁用（1771 行）都先调 checkAdminUsers 拒绝操作管理员用户，但行内操作 deleteUser（1602-1621）与 toggleUserStatus（1622-1640）没有该守卫：管理员可逐行删除其他管理员甚至自己，可把自己禁用导致后台自锁。前后端守卫不一致。
- **建议**: 把 checkAdminUsers 应用到单行 deleteUser/toggleUserStatus（至少对 is_admin 用户禁止删除/禁用），并在后端同样做硬校验（前端校验可被绕过）。

### [MEDIUM] 密码强度策略不一致：新建/编辑仅 6 位，重置却要求 8 位+复杂度
- **位置**: 921-936, 948-967  |  **类别**: security  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: userRules.password（928 行）只校验 value.length < 6，因此通过“添加用户/编辑用户”可设置 6 位弱密码；而“重置密码”对话框 validateResetPassword（950-964 行）要求 ≥8 位且大小写/数字/特殊字符至少三类并拒绝弱密码字典。同一系统两套强度标准，弱的一侧（创建路径）成为实际底线。
- **建议**: 统一策略：创建/编辑时复用 validateResetPassword（或至少统一最小长度与复杂度校验），并同步后端校验。

### [LOW] 批量操作成功数在 success_count=0 时回退为全量，掩盖全失败
- **位置**: 1699-1724, 1710-1711  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: const successCount = data.success_count || selectedUsers.value.length：当后端返回 {success:true, data:{success_count:0, fail_count:N}}（全部失败）时，0 || 全量 取到全量，前端弹“成功 N 个”误导性提示。
- **建议**: 改用空值合并：data.success_count ?? selectedUsers.value.length，并结合 fail_count 判断展示成功/失败比例。

### [LOW] getExpireText 可能渲染 'undefined天后到期'
- **位置**: 1106-1108, 276  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 模板（276 行）的展示条件是 days_until_expire !== null，undefined 会通过该判断；若 subscription 未过期但缺省 days_until_expire 字段（后端未算），getExpireText 直接模板字符串拼接出 'undefined天后到期'。
- **建议**: getExpireText 内对 days_until_expire 做判空：Number.isFinite(...) ? `${n}天后到期` : '即将到期'。

### [LOW] el-table-column 上的 @sort-change 永远不会触发
- **位置**: 178, 237, 290  |  **类别**: maintainability  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: Element Plus 的 sort-change 事件只在 el-table 上派发，el-table-column（balance 237 行、created_at 290 行）上的 @sort-change="handleSortChange" 是无效绑定（永不触发），真正的排序走的是表格级 178 行绑定。
- **建议**: 删除列上的两个无效 @sort-change 绑定，仅保留表格级事件。

### [LOW] 状态 map 与筛选选项不一致：桌面筛选缺 device_overlimit，且重复定义 STATUS_MAP
- **位置**: 823-832, 104-109  |  **类别**: style  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 移动端下拉有“设备超限”选项（44 行），桌面 select（104-109 行）只有 全部/活跃/待激活/禁用，同一页面两种筛选能力不同；同时 STATUS_MAP/SUBSCRIPTION_STATUS_MAP（823-832）与项目已有的 @/utils/statusMaps.js（USER_STATUS_MAP/SUBSCRIPTION_STATUS_MAP）重复定义，且文本与 statusMaps 存在差异（inactive 此处为“待激活”，UserDetailDialog 里为“未激活”）。
- **建议**: 删除本地 map，统一改用 @/utils/statusMaps.js 的 getStatusText/getStatusType；桌面与移动端筛选选项对齐。

## frontend/src/views/admin/components/UserDetailDialog.vue

### [HIGH] 抽屉打开状态下切换用户不重置内部状态，展示上一个用户的数据
- **位置**: 1031-1054  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: visible watcher 只在 val && !oldVal（从关闭到打开）时重置 devices/customNodes/checkinLogs/activeTab。如果抽屉已打开（visible 一直为 true），再点击另一个用户的“详情/余额”，Users.vue 更新 selectedUser 但本组件不触发任何重置：devices、customNodes、checkinLogs、activeTab 仍是上一个用户的，新用户打开即看到旧数据。
- **建议**: 新增 watch(() => this.user?.user_info?.id || this.user?.id)，id 变化时执行与 visible watcher 相同的重置+按需加载逻辑。

### [HIGH] confirmDelete/confirmWarning 参数用法与工具函数签名不符，确认框文案错乱
- **位置**: 1524-1527, 1695-1698  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 工具函数签名是 confirmDelete(entityName, count, options)（confirmAction.js 40 行），本文件 1524-1527 行却写成 confirmDelete(完整message, '确认删除')：entityName 被塞进整段确认语，count 变成字符串，结果弹窗文案变成“确定删除该确定要删除设备 xxx 吗？删除后不可恢复。”的拼接怪文。1695-1698 行 confirmWarning(完整message, '取消分配专线节点') 同样传了字符串第二个参数，被 ...options 展开成字符下标对象，title 选项被忽略（弹窗标题仍是默认“确认操作”）。
- **建议**: 统一改为 confirmDelete('设备', 1, { message: ... }) 与 confirmWarning(message, { title: '取消分配专线节点' })。

### [MEDIUM] 订单状态 'completed' 未纳入映射：订单表显示英文原文
- **位置**: 1197-1224  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: getStatusText/getStatusType 覆盖 active/inactive/paid/pending/cancelled/refunded/expired/success/failed，但没有 'completed'；Users.vue 的 getOrderStatusText 明确映射 completed: '已完成'。本文件订单记录 Tab 遇到 completed 会显示 type=info + 原始英文 'completed'，与其他状态中文文案不一致。
- **建议**: 补上 completed（及后端实际会返回的其它订单状态）到两处 map，或统一改用 statusMaps.js 的 ORDER_STATUS_MAP。

### [MEDIUM] 设备 Tab 无 loaded 标记：空设备用户每次切换都重新请求
- **位置**: 1055-1063  |  **类别**: performance  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: activeTab watcher 中 devices 分支的条件是 this.devices.length === 0 && !this.loadingDevices，没有类似 checkinLoaded 的标记。用户设备为 0 时，每次切到“设备记录”Tab 都会重新调用 loadDevices()（还会按订阅数分批并发请求）；checkins 分支有 !this.checkinLoaded 守卫，两者不一致。
- **建议**: 为 devices 增加 devicesLoaded 标记，首次加载后置 true，与 checkins 模式对齐。

### [MEDIUM] 模板循环内重复构建订阅 URL：getTypedSubscriptionUrl/getSubscriptionUrlWithExclude 每次渲染重算
- **位置**: 1133-1196, 95-166  |  **类别**: performance  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 模板每个订阅块的复制按钮、title 属性、more-urls collapse 都多次调用 getSubscriptionUrlWithExclude(sub, getTypedSubscriptionUrl(sub, type))（101、110、123、132、136、146 行），而 getMoreSubscriptionUrls（1136-1148）每次调用重新创建 6 个客户端对象并做 URL 拼接；每行订阅每帧重复数十次 URL 解析与拼接，属于无谓计算。
- **建议**: 拿到详情数据后为每个 sub 预计算 displayUrls 字典（或引入按 sub.id 缓存的 computed），模板只读缓存。

### [LOW] formatDate/formatDateTime 完全相同；getCurrentUserId 同时存在 computed 与方法两份
- **位置**: 1125-1132, 997-999, 1255-1257  |  **类别**: duplication  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: formatDate 与 formatDateTime（1125-1132）函数体完全一致（都调用 formatDateUtil），只是名字不同；getCurrentUserId 既有 computed（997-999）又有 methods 版本（1255-1257），逻辑重复。
- **建议**: 删除 formatDateTime（统一用 formatDate）与方法版 getCurrentUserId（统一用 computed），减少维护面。

### [LOW] 签到分页默认 size 不一致：data 里 10，visible watcher 里改成 20
- **位置**: 974-978, 1041  |  **类别**: logic  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: checkinPagination.size 初始化 10（977 行），但每次打开抽屉 watcher 强制设为 20（1041 行），两处默认值漂移。
- **建议**: 统一为同一个常量（10 或 20），并让 watcher 只重置 page 不重置 size，或两者同源。

### [LOW] getDeviceTypeColor 返回 'primary'：el-tag type 可能不支持该值
- **位置**: 1081-1092, 249  |  **类别**: style  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: getDeviceTypeColor 对 mobile 返回 'primary'（1084 行），Element Plus 的 el-tag type 合法值为 success/info/warning/danger/''（'primary' 仅高版本才支持，且依赖主题配置）；若版本不支持会退化为默认样式，颜色语义丢失。与 Users.vue/Subscriptions.vue 的 getDeviceTypeTag（同样返回 'primary'）重复定义且可能不一致。
- **建议**: 确认所用 Element Plus 版本是否支持 tag type='primary'；不支持则改为 'success' 等合法值，并把这套 device 类型 map 抽成共享模块。

### [LOW] 订阅到期时间原样显示 ISO 字符串，未格式化
- **位置**: 38-59, 52  |  **类别**: ux  |  **来源组**: F7-admin-views-1 (管理端 Users/UserDetailDialog/Subscriptions)
- **问题**: 模板第 52 行 {{ sub.expire_time || '未设置' }} 直接输出后端原始值（如 2026-06-30T16:00:00+08:00），而其它页面（Users.vue 323 行）用 formatDate 显示。同一字段两种展示格式，且原始格式对管理员不友好。
- **建议**: 改用 formatDate(sub.expire_time) 或 formatDateTime 统一格式化。

## frontend/src/views/admin/logs/*.vue（BalanceLogs/EmailLogs/CommissionLogs/SubscriptionLogs/SubscriptionResetLogs/RegistrationLogs/AuditLogs）

### [HIGH] 7 个日志页约 80% 代码复制粘贴：filter 双端表单、fetch/debouncedFetch/resetFilter/onSizeChange/paginationLayout 完全同构
- **位置**: 整体  |  **类别**: duplication  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 每页都重复：桌面+移动双 filter-bar、fetch() 的参数拼装（page/page_size/start_time/end_time）、debounce(fetch,500)、resetFilter、onSizeChange、paginationLayout computed、ResponsiveDataView+MobileLogFields 移动卡片结构、本地 Xxx_MAP 映射。仅列字段与接口名不同。任何分页/错误处理改动需同步 7 处（如本次发现的静默 catch 只在部分页面出现）。
- **建议**: 抽 `useLogList(apiFn, mapFns)` composable（参数拼装、分页重置、debounce、错误提示）与通用 `LogFilterBar` 组件；页面只需声明列定义与映射。预计每个文件缩至 ~100 行。

## frontend/src/views/admin/logs/AuditLogs.vue

### [MEDIUM] 客户端过滤与服务器分页冲突：total 含被过滤类型，页码漂移/空页
- **位置**: 272-294  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: fetch 拿到 data.logs 后前端 filter 掉 scheduler_/system_error/business_/security_ 类型（284-287），但 `total.value = data.total` 仍是后端全量总数（含被过滤类型）→ 例如 total=57、每页 20 但实际可见 40 条，翻到第 3 页永远空；且浪费带宽拉取被丢弃的数据。
- **建议**: 把过滤条件（排除类型白名单）作为参数传给后端（如 exclude_types=scheduler_,system_error,...），前端不做二次过滤；或后端审计日志查询默认排除这些类型。

### [LOW] ACTION_TYPE_MAP 大量重复键与冗余项（约 110 条，多处重复定义）
- **位置**: 100-168  |  **类别**: maintainability  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: batch_delete_users/update_user/delete_user/create_package/update_package/delete_package/create_invite_code/update_invite_code/delete_invite_code/create_backup/delete_backup/admin_delete_device/batch_delete_devices/batch_clear_devices/clear_config_update_logs/start_config_update/stop_config_update/update_config_update_config/update_system_config/update_system_config_batch/create_system_config/update_geoip_database/switch_geoip_database/flush_cache/create_custom_node 等键在 100-168 行内重复出现 2 次（JS 取后者，但表明靠复制粘贴维护），且 getActionTypeText 中 business_/scheduler_/system_error 分支（236-239）返回空字符串——这些类型在 fetch 已被过滤，属死分支。
- **建议**: 改为从后端下发 action_type 文案或收敛为独立 JSON 文件，删除重复键与死分支；对未知类型回退显示原文。

## frontend/src/views/admin/logs/BalanceLogs.vue

### [LOW] catch 静默清空列表，无任何用户提示
- **位置**: 161-165  |  **类别**: error-handling  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: fetch 失败仅 `list.value = []`，不弹 ElMessage，与 Tickets/Coupons 等页面的错误提示行为不一致；管理员会误以为“没有数据”。
- **建议**: catch 中 ElMessage.error 提示（可复用 error.response?.data?.message），与其它页面对齐。

### [LOW] onUnmounted(() => {}) 空钩子 + 未使用导入
- **位置**: 115, 184  |  **类别**: maintainability  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 184 行 `onUnmounted(() => {})` 空实现，且 onUnmounted 从 vue 导入（115 行）只为它服务。
- **建议**: 删除空 onUnmounted 与对应导入。

## frontend/src/views/admin/logs/CommissionLogs.vue

### [LOW] catch 静默失败（同 BalanceLogs）
- **位置**: 177-181  |  **类别**: error-handling  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: fetch 失败仅置空 list 无提示；且 getStatusColor 未知状态返回 ''（149-151），el-tag type='' 属合法但无颜色区分。
- **建议**: 补 ElMessage 提示；未知状态回退 'info' 与其它页面一致。

## frontend/src/views/admin/logs/EmailLogs.vue

### [LOW] 状态筛选枚举不完整：缺 sending/cancelled
- **位置**: 8-12  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 状态筛选只提供 待发送/已发送/失败（8-12 行），而 EmailQueue 与 EMAIL_STATUS_MAP 都有 sending（发送中）与 cancelled（已取消），后端日志里这两种状态的记录将无法筛出。
- **建议**: 补全 `sending`/`cancelled` 选项，或直接从 statusMaps 的 EMAIL_STATUS_MAP 派生选项。

## frontend/src/views/admin/logs/RegistrationLogs.vue

### [MEDIUM] 自研列宽拖拽的 document 级监听器在组件卸载时未清理
- **位置**: 145-163  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: startResize 在 mousedown 后 addEventListener('mousemove'/'mouseup', ...)，只在 mouseup 里移除；若用户在拖拽过程中切换路由（组件卸载），监听器残留到下一次 mouseup，期间 document.body.style.cursor 也一直为 col-resize。
- **建议**: 在 onUnmounted 中主动 removeEventListener 并恢复 cursor/userSelect；或改用 el-table 内置 resizable + border 属性（Element Plus 自带列宽拖拽）。

## frontend/src/views/admin/logs/SubscriptionLogs.vue

### [MEDIUM] getSubType/getDeviceInfo 用正则解析后端描述文本，耦合脆弱
- **位置**: 169-196  |  **类别**: logic  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `d.match(/^\[(.+?)\]/)` 从 description 提取订阅类型、`d.replace(/^\[.+?\]\s*/,'')` 提取设备信息，并回退 JSON.parse(before_data/after_data)；后端一旦改描述格式（如去掉方括号前缀），类型/设备列静默显示错误或 '-'，且每次渲染都做正则+JSON.parse（移动端模板多次调用 getSubType/getDeviceInfo）。
- **建议**: 要求后端在日志接口直接返回 subscription_type/software_name/device_name 结构化字段，前端删除解析逻辑；至少缓存解析结果避免模板内重复调用。

## frontend/src/views/admin/logs/SubscriptionResetLogs.vue

### [LOW] URL 截断无条件追加 '...'，短 URL 也显示省略号且可能切断订阅令牌
- **位置**: 72-74, 107  |  **类别**: ux  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: `(row.old_subscription_url || '').substring(0, 40) + '...'` 对长度不足 40 的 URL 也硬加 '...'；订阅 URL 尾部往往是敏感令牌，截断后无法核对，也不提供完整值（无 tooltip）。
- **建议**: 仅当 length>40 时截断并加 '...'，并加 el-tooltip 展示完整 URL 或提供复制按钮。

## frontend/src/views/admin/（跨文件）

### [MEDIUM] 分页参数命名不统一：size vs page_size，响应容器各异
- **位置**: AbnormalUsers 475 / Tickets 439 / Coupons 424 vs Knowledge 398 / Promotions 388 / logs 各 fetch  |  **类别**: architecture  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 同一系统里 AbnormalUsers/Tickets/Coupons 发 `size`，Knowledge/Promotions/全部日志页发 `page_size`；响应容器也各异（data.users/data.tickets/data.coupons/data.list/data.logs/data.levels）。前端每个页面都要写多分支防御解包（UserLevels 5 分支），说明后端 API 契约未统一。
- **建议**: 后端统一分页 DTO（page/page_size/total/list）与成功信封 {success,data}；前端 api 层做一次映射，删除各页面 3-5 分支解包。

### [LOW] formatDate/formatTime 在 8+ 个文件重复实现且行为各异
- **位置**: formatDate 定义于 AbnormalUsers 573 / Tickets 577 / Analytics 452 / Profile 613 / Knowledge 372 / Promotions 321 / EmailQueue 825 / EmailDetail 363  |  **类别**: duplication  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: 有的 toLocaleString('zh-CN') 含秒、有的不含秒、有的返回 '-'、有的返回 ''、有的用 utils/date 的 formatDateTime/formatTime（Coupons/EmailQueue）。同一字段在不同页面显示格式不一致（如 created_at 在 EmailQueue 走 formatDateTime，在 EmailLogs 直接原样输出）。
- **建议**: 统一收敛到 utils/date 的 formatDateTime/formatDate，全站引用，删除各文件本地实现。

### [LOW] 状态/类型中文映射多处本地重复，statusMaps.js 统一库未用上
- **位置**: Tickets 591-633 / EmailQueue 459-465 / EmailDetail 327-362 / 各 logs 页 Xxx_MAP  |  **类别**: duplication  |  **来源组**: F10-admin-views-4 (管理端其余视图 + logs)
- **问题**: statusMaps.js 已提供 TICKET_STATUS_MAP/TICKET_PRIORITY_MAP/EMAIL_STATUS_MAP 等，但 Tickets 仍手写 getStatusText/getStatusTagType/getPriorityText/getPriorityTagType，EmailQueue/EmailDetail 各写一份邮件状态映射，日志页各写 Xxx_MAP。同一状态文案在 3-4 处维护。
- **建议**: 全面改用 statusMaps.js 的 getStatusConfig/getStatusText/getStatusType 及 Xxx_MAP 常量，删除本地映射。

## frontend/tsconfig.json

### [MEDIUM] strict:false + checkJs:false 使 type-check 脚本形同虚设
- **位置**: 6-17  |  **类别**: architecture  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: tsconfig 关闭 strict、checkJs 且 allowJs:true + include 全量 .js，`npm run type-check`（vue-tsc --noEmit）对项目主体（JS 代码）几乎零检查输出；target ES2020 与 vite esbuild target esnext 也不一致，类型层面无法发现任何 null/undefined/拼写错误。
- **建议**: 逐步开启 strict，先让 .d.ts/.ts 纯类型文件进入严格检查，再逐目录开启 JS checkJs；target 与 vite 构建目标对齐。

### [LOW] jsx: preserve 对纯 Vue 项目无用
- **位置**: 7  |  **类别**: style  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 项目无 .tsx/.jsx 源码（include 含 tsx），jsx 配置属复制模板残留。
- **建议**: 删除 jsx 配置（或同时删 include 中的 tsx）。

## frontend/vite.config.js

### [MEDIUM] element-plus 整包别名是死配置，且是静默缺导出陷阱
- **位置**: 181-192  |  **类别**: architecture  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: alias 把 `^element-plus$` 重定向到 src/utils/elementPlusServices.js，而该文件只 re-export ElMessage/ElMessageBox/ElNotification 三个服务（已核实）；但全库 48 处 import 全部走 `@/utils/elementPlusServices`，没有任何 `from 'element-plus'` 的裸包导入——别名从未命中。它同时是陷阱：未来有人写 `import { ElLoading } from 'element-plus'` 会拿到 undefined 而非编译报错，且 `import ElementPlus from 'element-plus'`（全量安装）也会静默失效。
- **建议**: 删除该 alias；如担心误用，在 elementPlusServices.js 顶部加导出完整性断言（如校验 ElLoading 存在则报错）。

### [MEDIUM] VITE_USER_PREVIEW_MOCK=1 时 mock 中间件吞掉所有写请求，存在误开数据丢失风险
- **位置**: 91-168  |  **类别**: security  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 164 行 `if (req.method === 'POST' || req.method === 'PUT' || req.method === 'DELETE') return previewJson(res, { ok: true })`——对 /api/v1 下任意写请求返回假成功；一旦开发者在联调时误设该环境变量，表单提交、订单操作会“成功”但实际未落库，且无任何醒目告警。
- **建议**: mock 仅覆盖 GET 只读路径；写请求直接 next() 透传代理；启动时若开启 mock 打印醒目警告日志。

### [LOW] chunkSizeWarningLimit 注释与代码含义相反
- **位置**: 238  |  **类别**: logic  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: 注释写“降低警告阈值，鼓励更好的代码分割”，实际值 1000 是 Vite 默认 500 的两倍（调高=减少警告），注释与行为完全相反，会误导后续维护者。
- **建议**: 修正注释，或按真实意图取值（鼓励分割应设 <500）。

### [LOW] manualChunks 未给 Element Plus 单独分 chunk
- **位置**: 220-237  |  **类别**: performance  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: element-plus 组件走 per-component 按需引入（es/components/*/index.mjs），代码随各页面 chunk 内联，跨页面不共享；vue-vendor/charts/utils 都有独立 chunk 而 EP 没有，首屏与缓存命中都受影响。
- **建议**: 增加 `if (id.includes('/node_modules/element-plus')) return 'element-plus'` 分块（注意与按需 CSS 配合）。

### [LOW] dev server 绑定 0.0.0.0 且代理到无鉴权后端
- **位置**: 198-206  |  **类别**: security  |  **来源组**: F11-styles-config (样式/构建/审计脚本)
- **问题**: server.host '0.0.0.0' 使局域网内任意设备可访问 dev server 及其 /api 代理（target http://localhost:8000 的 Go 后端若本身无鉴权），LAN 攻击面随开发环境常驻。
- **建议**: 默认绑定 127.0.0.1，需要局域网调试时用 --host 显式开启。

## go.mod

### [MEDIUM] go 1.24.0 与全部部署脚本/镜像的 Go 1.21.5 版本失配
- **位置**: 3  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: module 声明 go 1.24.0，而 install.sh、install-vps.sh、bt-deploy.sh、Dockerfile 均安装/使用 Go 1.21.5（版本检查阈值 >=1.21）；部署链路依赖 GOTOOLCHAIN 自动下载 1.24，网络受限时编译失败，且本地 go.mod 校验与 CI 不一致。
- **建议**: 统一升级部署脚本与 Dockerfile 至 go 1.24.x；或在 go.mod 使用 toolchain 指令并显式声明最低 Go 版本，保证四处一致。

### [LOW] 依赖版本有老化迹象，且间接依赖清单偏大
- **位置**: 5-23  |  **类别**: style  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: google/uuid v1.4.0（2023）、gorm.io/gorm v1.25.5、gorm.io/driver/sqlite v1.5.4 等版本较旧；间接依赖包含 quic-go、sonic、xxh3 等大型包，多为支付 SDK/JSON 加速引入，整体可接受但值得定期升级。
- **建议**: 运行 go get -u ./... 并回归测试；对支付相关 SDK 升级单独验证签名/回调兼容性。

## go.sum

### [INFO] go.sum 与 go.mod 一致性良好，无明显问题
- **位置**: 1-199  |  **类别**: other  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 199 行哈希条目与 go.mod 的 require（含 indirect）一一对应（含 xyproto/randomstring、zeebo/xxh3 等传递测试依赖），无缺失或多余条目；go.sum 本身无需修改。
- **建议**: 无；建议在 CI 中执行 go mod verify 固化校验。

## install-vps.sh

### [HIGH] 通过明文 HTTP 下载阿里云盾卸载脚本并以 root 执行，存在供应链/中间人代码执行风险
- **位置**: 124  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: download_file "$script_name" "http://update.aegis.aliyun.com/download/uninstall.sh" ... 使用 http:// 明文传输并 chmod +x 后直接运行；任何可劫持 HTTP 流量的攻击者都能注入任意 root 代码。
- **建议**: 改为 https:// 地址，下载后校验 SHA-256 哈希再执行；或去掉脚本执行、仅提示用户手动卸载。

### [HIGH] 管理员密码明文回显，且被 tee 写入 /tmp 下默认权限的日志文件
- **位置**: 421-429, 16, 21  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: L21 exec > >(tee -a "$LOG_FILE") 将全部输出落盘到 /tmp/cboard_install_*.log（umask 022 下其他用户可读），安装完成时又 echo "密码: $A_PASS"，管理员密码明文留档在共享 /tmp。
- **建议**: 安装日志改放项目目录内 600 权限文件；不打印密码，改为提示"密码已保存于环境变量，请牢记"。

### [MEDIUM] GO_VERSION 固定 1.21.5，与 go.mod 要求的 go 1.24.0 不匹配
- **位置**: 13, 186-206, 263-265  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: install-vps.sh/bt-deploy.sh/install.sh 均安装 Go 1.21.5，而 go.mod 声明 go 1.24.0；安装检查仅要求 >=1.21。构建时依赖 GOTOOLCHAIN=auto 联网下载 1.24 工具链，离线/内网环境直接编译失败，且每次构建开销增大。
- **建议**: 把 GO_VERSION 统一升级到 1.24.x（三处脚本 + Dockerfile），或至少把版本检查改为与 go.mod 一致并显式 GOTOOLCHAIN=local。

### [MEDIUM] 重装时 rm -rf 整个项目目录，旧 .env 与 cboard.db 一并删除且无备份
- **位置**: 246-248  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: deploy_project 在用户确认覆盖后 rm -rf "$PROJECT_DIR"，随后仅在 .env 不存在时新建；重装 = 全量数据丢失（用户、订单、余额、密钥），只给了一句 y/N 确认。
- **建议**: 覆盖前自动备份 .env、cboard.db、uploads 到带时间戳目录；提示将丢失的数据量。

### [MEDIUM] force_restart 使用 pkill -f "server" 无差别杀死所有含 server 的进程
- **位置**: 456-458  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: pkill -f "server" 会匹配任意命令行包含 server 的进程（其他项目、数据库工具等），多站点主机上会误杀无关服务。
- **建议**: 改为精确匹配 pkill -f "${pd}/server" 或按 systemd unit 控制（systemctl restart cboard）。

### [LOW] sed 向 listen 80 所在 server 块内追加 443 ssl 监听，HTTP 与 HTTPS 同块共存且无 301 跳转
- **位置**: 358  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: sed -i "/listen 80;/a ...listen 443 ssl;..." 把 443 塞进同一 server 块，形成 80/443 同块监听；缺少 return 301 https:// 的跳转，用户访问 http:// 不会升级到 https，证书续期/SEO/安全均受影响。
- **建议**: 生成独立的 80→443 跳转 server 块与 443 server 块（参照 install.sh 的最终配置模板）。

## install.sh

### [HIGH] Docker 安装 Redis 不带密码并映射 0.0.0.0:6379，公网可直连未授权 Redis
- **位置**: 113  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: docker run -d --name redis --restart=always -p 6379:6379 redis:alpine 未设 requirepass，容器端口绑定所有网卡；后续 REDIS_PASSWORD 保持为空。公网服务器若防火墙未封 6379，攻击者可未授权访问 Redis（数据篡改、主从复制 RCE 等经典利用链），且 .env 中密码为空使应用也以无鉴权方式连接。
- **建议**: 绑定 127.0.0.1:6379 并生成随机密码写入 REDIS_PASSWORD；或使用 docker-compose 内网网络 + 仅容器间访问。

### [MEDIUM] Nginx 站点配置硬编码写入宝塔面板路径，非宝塔环境静默失效
- **位置**: 360-378  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: bt_path=/www/server/panel/vhost/nginx/${DOMAIN}.conf 被直接 cat 覆盖，但脚本号称通用部署；非宝塔主机该目录不存在（mkdir -p 会创建但 nginx 不 include），reload_nginx_force 后站点实际未生效，用户得不到任何提示。
- **建议**: 检测 BT 面板路径存在与否：存在写 BT 路径，否则写 /etc/nginx/conf.d/cboard.conf 并在 /etc/nginx/sites-enabled 建软链。

### [MEDIUM] sync_from_github 用 reset --hard + git clean -fd 强制同步，会丢弃本地提交并删除所有未跟踪文件
- **位置**: 847-848  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: git reset --hard origin/$branch 丢弃本地所有未提交修改，git clean -fd 删除全部未被 .gitignore 覆盖的未跟踪文件（.env、*.db、mmdb 等虽已 ignore 可幸免，但任何新产生的非忽略文件、临时目录会被静默删除），升级后本地补丁全部丢失且无备份提示。
- **建议**: 同步前自动备份（git stash / 打包 .env 与 uploads），用 git pull --rebase 替代 hard reset，clean 仅针对 frontend/dist 等已知构建产物。

### [MEDIUM] systemd 服务以 root 运行 Web 服务，且默认端口直连公网 8000 的资产路径未做权限隔离
- **位置**: 344, 292  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: cboard.service 中 User=root + WorkingDirectory=项目目录；结合 router.go 的静态文件读取与 admin 上传功能，进程越权面为 root。另 .env 中含 SMTP 密码、支付密钥等敏感值，root 进程 + 潜在文件读取放大泄露面。
- **建议**: 创建专用系统用户（如 cboard）运行服务，目录权限 750；仅保留对 uploads/logs/.env 的最小读写权限。

### [MEDIUM] 维护脚本对共享 Redis 执行 FLUSHDB，且 redis-cli 无 -a 密码参数
- **位置**: 597, 900  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: deep_clean 与 sync_from_github 均执行 redis-cli -h ... -p ... FLUSHDB；若 .env 配置了 REDIS_PASSWORD，此处未传 -a，连接会被拒绝（静默失败可接受），而共享实例则清空同 DB 其他应用数据。
- **建议**: 从 .env 读取密码传入 -a，且 FLUSHDB 前打印目标 host:port/db 并要求确认。

### [LOW] certbot 邮箱用 admin@${DOMAIN} 且证书目录用 find 模糊匹配
- **位置**: 382, 387  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: admin@${DOMAIN} 邮箱基本不存在，certbot 校验邮箱可能失败导致证书申请整体失败（被 2>/dev/null 吞掉）；find /etc/letsencrypt/live -name "*${DOMAIN}*" 可能匹配到前缀相同的其他域名目录。
- **建议**: 证书申请使用 --register-unsafely-without-email 或让用户提供邮箱；证书目录改为精确 /etc/letsencrypt/live/${DOMAIN}/。

### [LOW] 菜单循环等待"按回车返回"，管道/非交互执行会无限循环
- **位置**: 1078  |  **类别**: style  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: read -r -p "按回车键返回菜单..." temp 在无 TTY 环境立即返回导致菜单死循环刷屏；脚本仅 root 检查、无参数化入口，CI 无法使用。
- **建议**: 支持 ./install.sh <子命令> 直跑（如 deploy/sync/status），无 TTY 时跳过等待。

## internal/api/handlers/admin.go

### [HIGH] 支付配置接口明文回传商户私钥/密钥（merchant_private_key、wechat_api_key、paypal_secret、stripe_secret_key 等）
- **位置**: 978-1036, 1176, 1325-1354  |  **类别**: security  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: GetPaymentConfig 的 PaymentConfigResponse（978-1002 行）与 UpdatePaymentConfig 的 responseData（1325-1354 行）把全部密钥字段明文返回给前端，CreatePaymentConfig 第 1176 行更是直接返回 models.PaymentConfig 结构体（该模型所有密钥字段均为 json 可序列化，payment_config.go:13-25）。密钥被拉入浏览器侧意味着：管理面板任何一处 XSS、浏览器扩展、开发者工具误操作都能一次性窃取全部支付凭证；且 Update 采用"前端回传密钥"的 echo 模式，密钥安全完全依赖前端。
- **建议**: 读接口只返回脱敏信息（如 `has_secret: true` 或末 4 位掩码）；Update 接口对空值/掩码值视为"不修改"，仅在实际传入新值时落库；Create/Update 的响应同样脱敏。密钥只允许在服务端支付处理流程中读取。

### [MEDIUM] GetAdminTicket 是 GET 请求却修改数据（processTicketReadStatus 标记已读）
- **位置**: 378-477, 402  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 402 行 `processTicketReadStatus(db, &ticket, adminUserID, true)` 在 GET /admin/tickets/:id 中把该工单的已读状态落库（ticket_read_state），GET 携带写副作用：违反 REST 语义、与 HTTP 缓存/CDN 冲突（刷新页面即"已读"，列表接口的 has_unread 逻辑随之变化），且无任何幂等/防抖。
- **建议**: 把"标记已读"拆为独立的 POST /admin/tickets/:id/read（前端进入详情时显式调用）；或至少给 GET 响应加 Cache-Control: no-store 并接受副作用语义。

### [MEDIUM] GetUserTrend 的 days 无上限校验，且 DATE(created_at) 无法使用已有索引
- **位置**: 1040-1067  |  **类别**: performance  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 1043-1045 行 `fmt.Sscanf(daysStr, "%d", &days)` 忽略错误且不设上限，days=100000 时 `created_at >= startTime` 退化为近似全表扫描；第 1053 行 `DATE(created_at)` 对时间列做函数包裹，models/user.go:18 的 created_at 索引失效，用户量大时 GROUP BY 聚合全表。
- **建议**: 钳制 days 范围（如 1-365）；把 DATE(created_at) 改为 `created_at >= ? AND created_at < ?` 的范围谓词以命中索引。

### [MEDIUM] UpdatePaymentConfig 以 INFO 级别打印 config_json 全文与新旧值，可能包含支付密钥
- **位置**: 1209-1212, 1303-1309, 1323  |  **类别**: security  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 1211 行 `utils.LogInfo("UpdatePaymentConfig: config_json内容=%+v", req.ConfigJSON)`、第 1305-1306 行打印新旧 config_json 全文、第 1323 行再打一遍——若 config_json 内含渠道密钥/Token（易支付、码支付等渠道常见），密钥会进日志文件/日志采集系统，扩大泄露面。
- **建议**: 日志只记录长度与哈希（如 `config_json_len=%d, sha256=%s`），禁止打印原始值；删除 1209-1212 行的请求体回显日志。

### [MEDIUM] GetAdminEmailConfig 明文返回 email 类别全部 SystemConfig，含 SMTP 密码
- **位置**: 931-939  |  **类别**: security  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: email 服务从 email 类别的 SystemConfig 读取 smtp_password/email_password（email.go:92），而本接口把该类别所有 key 原样返回（935-938 行 `configMap[config.Key] = config.Value`）。SMTP 密码一旦泄露可被用于冒充站点发钓鱼邮件或消耗邮件配额。
- **建议**: 对 email 配置做敏感 key 白名单/黑名单（smtp_password、email_password 等只写不回），读取时返回 has_value 掩码。

### [LOW] 支付配置的响应结构体与字段映射在 Get/Create/Update 三处重复声明
- **位置**: 978-1002, 1004-1036, 1325-1354  |  **类别**: duplication  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: PaymentConfigResponse 结构体（978-1002 行）与 UpdatePaymentConfig 的 responseData 映射（1325-1354 行）逐字段重复，CreatePaymentConfig 又直接序列化模型（1176 行）——三处对同一组字段维护三份拷贝，新增渠道字段时极易漏改其中一处（本次密钥脱敏改造也将被迫三处同步）。
- **建议**: 定义统一的 `toPaymentConfigResponse(cfg *models.PaymentConfig, maskSecrets bool)` 转换函数，三处调用；Create 也走同一响应构造。

### [LOW] GetUserLevel 与多处 db.Raw 统计的错误被忽略
- **位置**: 156-166, 366-375, 680-691, 847-859  |  **类别**: error-handling  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: GetUserLevel 第 688 行 `database.GetDB().First(&userLevel, user.UserLevelID.Int64)` 错误被吞，用户无等级时返回零值结构体且 success=true，前端无法区分"无等级"与"查询失败"；GetAdminInviteStatistics（156-166 行）、GetAdminTicketStatistics（366-375 行）、GetEmailQueueStatistics（847-859 行）的 db.Raw 错误均未检查。
- **建议**: 统一检查 db.Raw/First 的错误：查询失败返回 500，用户无等级返回空对象并显式说明；统计接口失败时记录日志。

### [LOW] err == gorm.ErrRecordNotFound 直接比较 + 字符串主键 First 查询，非数字 id 触发 500
- **位置**: 385, 702, 866, 575  |  **类别**: error-handling  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 多处用 `err == gorm.ErrRecordNotFound`（385、702、866 行）而非 errors.Is，GORM 在部分路径会包装错误导致比较失效落入 500 分支；同时 `db.First(&userLevel, id)`（575 行）、`db.First(&paymentConfig, id)`（1216 行）等以 `c.Param("id")` 字符串直接查询，传入非数字（如 "abc"）时生成非法 SQL 报错返回 500 而非 400。
- **建议**: 统一 `errors.Is(err, gorm.ErrRecordNotFound)`；对路径参数先做 strconv 校验（或 GORM 数值化查询），非法 id 返回 400；可提取 fetchByID 公共 helper。

### [LOW] 支付配置 Status 的 0 值语义矛盾：Create 把 0 当"未设置"置为 1，Update 的 >=0 恒真
- **位置**: 1133-1135, 1273-1275  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: CreatePaymentConfig 第 1133-1135 行 `if req.Status == 0 { req.Status = 1 }`（0 被当作未提供），而 UpdatePaymentConfig 第 1273 行 `if req.Status >= 0 { paymentConfig.Status = req.Status }` 对任意非负值（含 0）都执行覆盖——同一字段在创建/更新时对 0 的处理完全相反，且 `>= 0` 对 int 而言只有负数才不生效，判断形同虚设。
- **建议**: Status 改为 `*int` 指针区分"未提供"与"显式 0/1"，创建时也按显式值落库；或统一约定 0=禁用、1=启用并去掉创建时的默认改写。

### [LOW] UpdatePaymentConfig 修改 pay_type 后不重算 NotifyURL，回调地址残留旧渠道
- **位置**: 1221-1289  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 1281 行仅当 `paymentConfig.NotifyURL.String == ""` 时才按新 pay_type 生成回调地址；若管理员把 alipay 改为 wechat（或反之），已存在的 NotifyURL 保持 alipay 后缀不变，支付回调将打到错误地址导致回调丢失。
- **建议**: pay_type 发生变化且 NotifyURL 是自动生成值（或与旧 pay_type 匹配）时强制按新类型重建 NotifyURL；并把 notifySuffix 映射提取为公共函数供 Create/Update 复用。

### [LOW] parseBatchUintIDs 的兜底逻辑遍历 payload 全部字段，可能误匹配无关数组
- **位置**: 1450-1490  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 1482-1487 行在指定 key 都找不到后，遍历请求体所有字段，把第一个可解析为 uint 数组的值当作 ID 列表返回。若请求体含其它数组型字段（如 config 数组、tags），会静默采用错误字段执行批量删除/禁用——破坏性操作的入参来源不可预期。
- **建议**: 删除"遍历全部字段"的兜底分支，只接受显式 key（ids/id_list/调用方传入的别名）；解析失败即报 400。

### [LOW] GetAdminInvites/GetAdminInviteRelations 的用户搜索用子查询 + LIKE 前导通配，无法走索引
- **位置**: 29-48, 94-106  |  **类别**: performance  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 31 行 `user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)` 与第 96/100 行同类写法：LIKE '%kw%' 前导通配符使 username/email 索引失效，且 Count 与 Find 各执行一次带子查询的语句；invite_codes/invite_relations 表随用户量增长后此页会明显变慢。
- **建议**: 改为先查 users 表（分页+索引友好）得到 user_id 集合再 IN 关联，或给 invite_codes/invite_relations 增加冗余的 username 快照列；LIKE 前缀匹配（'kw%'）可命中索引。

### [LOW] ClearEmailQueue 用 Where("1 = 1") 拼无条件删除，且无二次确认
- **位置**: 910-929  |  **类别**: style  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 917 行 `result = db.Where("1 = 1").Delete(&models.EmailQueue{})` 全量清空邮件队列，仅靠"管理员才能访问"兜底；无确认参数/无回收站，误点或误调即丢失全部待发邮件。
- **建议**: 删除 `Where("1 = 1")`（直接 db.Delete），并在 handler 层要求显式确认字段（如 confirm=true）或将"清空全部"改为先置为 canceled 的软处理。

## internal/api/handlers/analytics.go

### [HIGH] GetChurnWarning 硬编码 `s.is_active = 1`，PostgreSQL 下布尔列与整数比较直接报错
- **位置**: 175-180  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `JOIN subscriptions s ON s.user_id = u.id AND s.is_active = 1` — SQLite/MySQL 的布尔以 0/1 存储可运行，但 PostgreSQL 的 boolean 列与 `= 1` 比较会抛 "operator does not exist: boolean = integer"，与项目宣称的三数据库支持冲突；且 JOIN 未去重，同一用户多条活跃订阅会产生重复行，LIMIT 50 实际命中用户数少于 50。
- **建议**: 改用 GORM 参数绑定 `s.is_active = ?`（传 true），查询按 u.id 去重（GROUP BY u.id 或先取 min(expire_time) 再过滤）。

### [MEDIUM] analytics.go 全部 Raw().Scan() 均无错误处理，失败时静默返回零值
- **位置**: 56-64  |  **类别**: error-handling  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: GetUserAnalytics（line 56-64、72-80）、GetRetentionAnalytics（line 126-137）、GetChurnWarning（line 175-180）、GetDeviceAnalytics（line 195-200）共 6 处查询全部丢弃 .Error，数据库异常时前端拿到 200 + 全零数据，无法区分"确实为零"与"查询失败"。
- **建议**: 统一补 `.Error` 检查并返回 500；可抽一个 `scanOr500(c, db, sql, params, dest)` 辅助函数消除重复。

### [MEDIUM] GetUserAnalytics 把 WAU/MAU 强制置为 DAU，且数据源选择依赖全量 ActivityCount
- **位置**: 66-84  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: timeRange 为 month/year 时 `activityStats.WAU = activityStats.DAU; activityStats.MAU = activityStats.DAU`（line 67-69、81-83）——周活跃/月活跃被压平成日活跃，指标失去意义；且 `if activityStats.ActivityCount > 0` 判断的是全时段累计，只要 user_activities 表里有一条历史数据就永远不走 users.last_login 兜底，数据源切换不可控。
- **建议**: 按 range 计算独立时间窗：DAU=当日、WAU=近7天、MAU=近30天，month/year 范围下用各自窗口的 distinct user_id；数据源选择改为"范围内 ActivityCount>0"。

### [LOW] GetRevenueAnalytics 金额返回字符串、比率返回数字，响应类型混合
- **位置**: 289-298  |  **类别**: architecture  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `current`/`previous`/`avg_order` 经 formatMoney 返回 "%.2f" 字符串，而 `change_rate` 是 float64、`order_count` 是 int64——同一响应里同类指标类型不一致，前端需分情况 Number() 或字符串拼接，容易出精度/比较 bug。
- **建议**: 统一金额类型（建议服务端保留 float64 并固定两位，或全部返回字符串），并给前端类型标注；formatMoney 命名建议改为 FormatMoney 并加注释说明契约。

### [LOW] 留存分析的 7/30 日队列是"单日快照"而非滚动窗口，且 retained 依赖 last_login 更新时机
- **位置**: 109-137  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: d7Start/d7End 是 7 天前那一天的 1 天队列、d30 同理（line 109-113），与注释"7天前注册的用户中之后仍登录过的比例"基本吻合但口径敏感（差一天注册就被排除）；retained 以 last_login 为准，若 last_login 只在登录接口更新，则订阅续费等其他活跃不计数，留存被低估。
- **建议**: 明确队列口径（注册日 = 自然日）并在注释/文档写明；如可行，活跃信号改为 union(users.last_login, user_activities.created_at)。

## internal/api/handlers/auth.go

### [HIGH] ResetPasswordByCode 无任何限流，6位重置验证码可被暴力枚举
- **位置**: 1182-1319  |  **类别**: security  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 路由 router.go:61 `auth.POST("/reset-password", handlers.ResetPasswordByCode)` 未挂任何 rate-limit 中间件，handler 内也没有失败计数（对比 VerifyCode 端点有 VerificationAttempt 表计数、SendVerificationCode/ForgotPassword 有 VerifyCodeRateLimitMiddleware）。攻击者可用任意 IP 对同一邮箱高频调用该接口穷举 100 万种 6 位验证码（第 1229-1268 行仅做 3 次 SELECT），一旦命中即可重置任意已注册账号密码。
- **建议**: 为 /reset-password 增加 per-email+per-IP 的失败次数限制（如 5 分钟内失败≥5 次即锁定，写入 VerificationAttempt 表并校验 purpose="reset_password"），并在路由层挂 VerifyCodeRateLimitMiddleware；验证码比对可改为对 code 做哈希后查询以降低时序侧信道。

### [MEDIUM] 用错误字符串匹配判断唯一约束，PostgreSQL 下重复邮箱/用户名注册返回 500
- **位置**: 115-126  |  **类别**: error-handling  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 116 行 `strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "Duplicate entry")` 只匹配 SQLite（UNIQUE constraint failed）与 MySQL（Duplicate entry）的报错文本；PostgreSQL 的报错是 `duplicate key value violates unique constraint "idx_users_email"`，两个子串都不命中，直接落到第 125 行返回 500 "创建用户失败"。同样的模式也用于第 117-121 行区分 email/username。
- **建议**: 不要用错误字符串做业务判断：注册前已做过 Count 预检（73-81 行），可用 errors.Is(err, gorm.ErrDuplicatedKey)（GORM v2 提供）或映射错误码（SQLite 1555/MySQL 1062/PG 23505）来区分；或删除预检、完全依赖事务内返回的明确错误。

### [MEDIUM] 注册验证码在用户创建事务之前就被标记已使用
- **位置**: 88-91, 479-490  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: Register 在第 88 行调用 verifyRegisterCode，后者在第 488-489 行 `verificationCode.MarkAsUsed(); db.Save(&verificationCode)` 将验证码置为已用，此时用户创建事务（103-134 行）尚未执行。若事务因唯一约束竞态（115-126 行）或 createDefaultSubscription 失败而回滚，验证码已被消费，用户必须重新获取验证码；且 db.Save 的错误也被忽略。
- **建议**: 把"标记验证码已用"移入用户创建事务内（与创建用户同事务提交），或在事务失败时回滚/重新标记验证码未使用；MarkAsUsed 的错误应检查并记录。

### [MEDIUM] 邀请奖励 goroutine 与主流程的 db.Save(user) 存在余额覆盖竞态
- **位置**: 146-169, 839-858, 860-936  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 148 行 `go distributeInviteRewardAfterCommit(db, user.ID)` 异步发放奖励（distributeReward 内用 `gorm.Expr("balance + ?", amount)` 更新余额），而主流程第 151 行重新 `First(&user)` 读到旧余额后，第 167 行 `db.Save(user)` 会全字段写回（含 Balance）。若 goroutine 的余额 UPDATE 落在两次 DB 调用之间提交，奖励会被旧值覆盖，静默丢失；且奖励发放与关系标记（InviterRewardGiven/InviteeRewardGiven）分两次无事务写库，崩溃后无法幂等重试。
- **建议**: 奖励发放改为在注册事务内完成（或单独的事务+幂等键），发放完成后再响应；至少应把 LastLogin 更新改成 `db.Model(&models.User{}).Where("id = ?", user.ID).Update("last_login", now)` 避免全字段 Save 覆盖并发写入。

### [MEDIUM] Login 与 LoginJSON 对维护模式处理不一致，且维护期间 refresh/logout 被全局中间件阻断
- **位置**: 231-250, 653-682  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: checkMaintenanceMode 只在 LoginJSON（262 行）调用，Login（231-250 行）完全没有该检查：维护模式下走 /auth/login 仍能正常签发令牌、写入 login_history；而全局 MaintenanceMiddleware（maintenance.go:97-106）只放行 /auth/login、/auth/login-json、/api/v1/admin 等前缀，/auth/refresh 与 /auth/logout 不在放行列表——维护期间用户无法登出也无法续期令牌，管理员会话到期后同样被锁死。
- **建议**: 把维护模式检查统一收敛到 MaintenanceMiddleware（放行 admin 登录、阻断普通用户登录并返回维护文案），并移除 handler 内的重复逻辑；同时在 allowedPaths 中加入 /auth/refresh 与 /auth/logout，保证会话可续期、可吊销。

### [MEDIUM] 邀请码只校验非空不校验有效性，"需要邀请码"配置形同虚设
- **位置**: 93-100, 766-836  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: Register 第 96 行仅检查 `req.InviteCode == ""`，而 processInviteCode 对不存在的邀请码在第 781 行 `return nil` 静默返回（注释"邀请码不存在或未激活，静默返回"）。因此开启 invite_code_required=true 后，用户随便填一个字符串即可注册，不产生任何邀请关系，门槛完全失效。
- **建议**: 在 Register 中把邀请码校验改为强校验（存在且 is_active、未过期、未达上限），校验失败返回明确错误；processInviteCode 应返回 (bool, error) 让调用方区分"未使用邀请码"与"邀请码无效"。

### [MEDIUM] getMinPasswordLength 每次请求查库，注册/改密/重置链路反复查询 SystemConfig
- **位置**: 47-55, 83, 970, 1032, 1213  |  **类别**: performance  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: getMinPasswordLength 每次调用都执行一次 SystemConfig 查询；Register 单请求内叠加 getMinPasswordLength（83 行）+ verifyRegisterCode 的 email_verification_required（451-455 行）+ invite_code_required（94-100 行）共 4-5 次同表查询，ChangePassword/ResetPassword/ResetPasswordByCode/CreateUser 各自再查。这些配置是低频变更数据，完全可缓存。
- **建议**: 为 SystemConfig 增加带 TTL 的读缓存（参考 maintenance.go 中 maintenanceCache 的做法）或启动时一次性加载到内存；handler 只读缓存，配置更新时失效。

### [MEDIUM] 登出时 refresh token 可绕过黑名单，RefreshToken 黑名单写入错误被忽略
- **位置**: 326-328, 384-394  |  **类别**: security  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: Logout 只在请求体携带 refresh_token 时才将其加入黑名单（384-394 行），前端若只带 access token 登出，refresh token 仍可无限期续期，"登出"实际未吊销会话；同时 RefreshToken 第 327 行 `_ = models.AddToBlacklist(...)` 忽略错误——若黑名单写入失败，被轮换的旧 refresh token 可被重放。
- **建议**: Logout 时要求或默认携带 refresh_token 一并黑名单化（前后端契约对齐）；RefreshToken 中黑名单失败应记录并考虑拒绝本次轮换；AddToBlacklist 失败时返回错误。

### [LOW] 1473 行 god file 混入认证/密码/验证码/邀请奖励多个职责，且 goroutine 内使用 gin.Context
- **位置**: 1-1473, 172-189, 201-213  |  **类别**: architecture  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: auth.go 把登录、注册、令牌刷新、改密、找回密码、验证码、邀请奖励（含 3 个奖励函数）全部堆在一个文件（注释"从 password.go/verification.go 合并"印证了合并史）；多处 goroutine 在 handler 返回后继续读取 gin.Context（172-189 行读 c.GetHeader、201-213 行读 c.GetHeader），gin 官方明确不建议在 handler 返回后使用 Context（存在竞态/悬挂风险）。
- **建议**: 按职责拆分文件（auth_login.go / auth_password.go / auth_verification.go / invite_reward.go 或迁入 services）；所有 goroutine 入口先取出所需值（IP/UA/UserID 等基本类型）再闭包捕获，杜绝捕获 *gin.Context。

### [LOW] generateVerificationCode 忽略 rand.Read 错误，与 ForgotPassword 分支不一致
- **位置**: 1467-1472  |  **类别**: error-handling  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 1469 行 `rand.Read(b)` 未检查错误，而 ForgotPassword 第 1113-1116 行同款代码检查了错误并返回 500。若 crypto/rand 读取失败（熵源故障等极端情况），code 恒等于 100000，所有验证码相同，且无任何日志提示。
- **建议**: 与 ForgotPassword 保持一致：检查 rand.Read 错误并返回 500；两处 6 位验证码生成可提取为 utils.GenerateNumericCode(n) 统一复用。

### [LOW] 注册日志的 inviterID 恒为 nil（User.InvitedBy 全库无写入，属死逻辑）
- **位置**: 196-200  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 197 行 `if user.InvitedBy.Valid` 读取的 InvitedBy 字段（models/user.go:43）在整个代码库中从未被赋值（grep 确认仅此处读取）；processInviteCode 只写 InviteCodeUsed 与 InviteRelation。因此 CreateRegistrationLog 的 inviter_id 永远是 nil，这段 4 行的转换逻辑与日志中的邀请人字段均无实际作用。
- **建议**: 要么在 processInviteCode 创建 InviteRelation 时同步回写 user.InvitedBy，要么删除这段死代码并从注册日志中去掉 inviter_id 参数。

### [LOW] 登录失败限流按 IP 记录且成功即重置，无法防分布式爆破、可被同 IP 用户干扰
- **位置**: 534-552, 711  |  **类别**: security  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: handleLoginFailure 只按 IP 计数（middleware.IncrementLoginAttempt(ip)），finalizeLogin 第 711 行对同一 IP 的成功登录执行 ResetLoginAttempt(ip)。后果：1) 攻击者可用自己的正确账号在共享出口 IP 上反复重置他人的锁定计数；2) 分布式 IP 对单账号爆破无法被 IP 维度限流捕获；3) NAT 后大量用户共享 IP 时，一人爆破会锁定全屋用户。
- **建议**: 限流键改为 IP+账号组合（如 sha256(ip|email)），失败计数按账号维度保留；成功登录只重置该账号（而非整个 IP）的计数。

### [LOW] handleValidationError 文案错位：Email 超长(max) 返回"邮箱不能为空"
- **位置**: 399-432  |  **类别**: ux  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 403-408 行 switch 中，Email 字段只要 tag 不是 "email" 就统一返回"邮箱不能为空，请输入您的邮箱地址"，但注册约束是 `required,email,max=255`，触发 max 违规时文案与原因完全不符；Password 分支（417-425 行）同样把 min/required 之外的 tag 兜底为"密码验证失败"，语义含糊。
- **建议**: 按 tag 分别映射文案（required→不能为空，max→长度超限，email→格式错误），未知 tag 返回通用"字段校验失败: <field> <tag>"。

## internal/api/handlers/backup.go

### [HIGH] 备份 zip 打包 .env 与 config.yaml：密钥随备份外泄，且自动上传到 GitHub/Gitee 仓库
- **位置**: 119-146, 148-196  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: CreateBackup 把 .env 和 config.yaml 一起打进 zip（119-146 行），.env 通常含 JWT_SECRET、数据库口令、SMTP 密码等；而 isAutoBackup 模式下该 zip 名固定 backup_auto.zip 并会在后台自动上传到远程仓库（172-189 行 client.UploadBackupWithProgress）。一旦远程仓库公开/泄露，等于把生产密钥发布到第三方平台。
- **建议**: 备份 zip 只含 cboard.db（如 BuildDBOnlyBackupZip 的语义）；.env/config.yaml 改为脱敏导出（仅非敏感键），或明确排除并提示管理员单独保管。

### [MEDIUM] RestoreBackup 先关闭线上数据库再替换文件：无校验、无锁、窗口期内全站 DB 请求报错
- **位置**: 357-492  |  **类别**: error-handling  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: database.CloseDatabase()（454 行）后拷贝文件（463 行）、ReopenDatabase()（476 行），期间所有持有旧句柄的 in-flight 请求/后台 goroutine（如邮件、签到、设备记录）全部失败；且对恢复的 DB 不做 SQLite 有效性/版本/schema 校验，误传一个损坏或异构库会直接让面板起不来（依赖 .before_restore 快照回滚，但快照本身也可能失败）。与 CreateBackup 之间也无互斥，可并发执行互相踩踏。
- **建议**: 恢复前用 SQLite 完整性检查（PRAGMA integrity_check / 打开并读 users 表头）验证文件；用全局恢复互斥锁串行化 Create/List/Restore；尽量在低峰期执行并向前端提示短暂不可用。

### [MEDIUM] 直接拷贝运行中的 SQLite 文件，WAL checkpoint 后仍可能产生撕裂快照
- **位置**: 27-30, 97-117  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: CreateBackup 依赖 PRAGMA wal_checkpoint(TRUNCATE)（29 行）+ 裸文件拷贝（110 行）；checkpoint 后仍有并发写事务时（备份期间用户下单/签到），拷贝到一半的文件可能不一致；且拷贝失败后 zip 残留不完整文件（110-114 行只返回错误不清理）。
- **建议**: 用 SQLite 在线备份 API（VACUUM INTO / sqlite3 backup）或对 DB 文件加写锁后再拷贝；失败时删除半成品 zip。

### [MEDIUM] extractDBFromZip 无解压大小限制，zip 炸弹可写满磁盘
- **位置**: 494-523  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: io.Copy(tmpFile, rc)（515 行）对 zip 内 cboard.db 的解压后大小没有任何上限；恶意构造的 zip（远程 restore 场景文件来自第三方仓库）可声明巨大未压缩尺寸，瞬间填满磁盘导致服务不可用。
- **建议**: 限制解压后大小（如 2× 当前 DB 大小或固定上限），用 io.LimitReader + 校验写入字节数。

### [LOW] 远程上传 goroutine 无 panic 恢复
- **位置**: 172-189  |  **类别**: error-handling  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: go func(){ client.UploadBackupWithProgress(...) } 内任何 panic（第三方库、nil 解引用）都会使整个进程崩溃；os.Remove(tmpFilePath) 失败仅打日志。
- **建议**: goroutine 内 defer recover 并写入状态管理器错误状态。

### [LOW] 响应中回传服务器绝对路径（path 字段）
- **位置**: 203-214  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: CreateBackup 响应包含 backupPath 绝对路径（205-206 行），虽然当前仅管理员可访问，但属于不必要的服务器文件系统信息泄露，日志/代理层可能转储。
- **建议**: 响应只返回 filename/size/upload 状态，去掉 path 字段。

## internal/api/handlers/checkin.go

### [MEDIUM] 用 err.Error() 字符串比较判断业务错误，脆弱且易漂移
- **位置**: 115-121  |  **类别**: style  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: if err.Error() == "今天已经签到过了" 依赖错误文案精确匹配：任何文案改动（如加句号、改措辞）都会让该分支变成 500"签到失败"，且错误文案同时透传给用户（117 行 err.Error() 直接返回）。
- **建议**: 定义 sentinel 错误（var ErrAlreadyCheckedIn = errors.New(...)）并用 errors.Is 判断；业务错误不要通过 ErrorResponse 透传原始 error。

### [LOW] 余额用 float64 累加，金额精度与 0 元奖励问题
- **位置**: 49-68, 70-75  |  **类别**: architecture  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 签到奖励 amount 由 0-100 分换算成 float64（56 行），user.Balance += amount 后写回（71-73 行）——浮点累加存在 0.1+0.2 类误差；且 0-100 分区间含 0，70% 概率下可能"签到成功，奖励 ¥0.00"。
- **建议**: 余额统一改用整数分存储（或 decimal），奖励下限设为 1 分，避免 0 元签到。

### [LOW] rand.Int 错误被忽略，失败路径会 nil 解引用 panic
- **位置**: 51-64  |  **类别**: error-handling  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 第一次 rand.Int(rand.Reader, big.NewInt(100)) 的 error 直接丢弃（51 行 randomPercent, _），若 crypto/rand 读取失败返回 nil，randomPercent.Int64()（52 行）即 nil 指针 panic；后两次调用虽检查了 error，但失败时 amount 保持 0，产生"签到成功但 0 元"。
- **建议**: 统一检查三次 rand.Int 的错误并返回 500；或抽一个 randAmount() 帮助函数。

### [LOW] GetCheckinStatus 的 Raw 查询错误被忽略，streak 查询固定拉 3650 行
- **位置**: 150-166  |  **类别**: error-handling  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 150-157 行 db.Raw(...).Scan(&checkinStats) 错误被忽略（DB 故障时静默返回全 0 统计）；159-166 行一次性 Pluck 3650 条 created_at 到内存再算连续天数，对多年老用户每请求全量拉取。
- **建议**: 检查 Scan 错误；streak 用 SQL 窗口函数或按天倒序 LIMIT 小步查询，避免全量 Pluck。

### [LOW] 用 c.Set/c.Get 在事务闭包与响应之间传值，耦合隐晦
- **位置**: 109-110, 124-125  |  **类别**: style  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 事务内把 amount/balance 塞进 gin.Context（109-110），事务外再 Get 出来（124-125）；若事务提前 return 或未来拆函数，取值容易漏掉。上下文本意是存请求态，不宜当返回值通道。
- **建议**: 用命名返回值或闭包外变量（在 WithTransaction 前声明 var amount, balance float64，闭包内赋值）传递结果。

## internal/api/handlers/config.go

### [MEDIUM] UpdateSystemConfig 批量模式按 key 单列 upsert，与全局 (key, category) 复合身份不一致
- **位置**: 206-220  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: `clause.OnConflict{Columns: []clause.Column{{Name: "key"}}}` 假设 key 唯一，但 CreateSystemConfig（L172-174）按 key+category 判重、单条更新（L245）按 key+category 查询、updateSettingsCommon（L80）按 key+category FirstOrInit——若表里存在同 key 不同 category（如 domain_name 同时在 general 与 system），批量 upsert 会更新错行或与唯一约束冲突；且批量模式把所有 key 强制写入 Category: CatSystem（L210），本属其他分类的键会在 system 下产生重复副本，GetAdminSettings 按分类查找时便读不到。
- **建议**: 批量模式改为按 (key, category) 复合冲突键（确保表上有对应复合唯一索引），或干脆让前端走 updateSettingsCommon 分类端点，删除该批量端点。

### [MEDIUM] updateSettingsCommon 用 fmt.Sprintf 序列化值，嵌套对象会被存成 Go map 字面量
- **位置**: 57-123  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: `valStr := fmt.Sprintf("%v", val)`（L71）只对 []interface{} 特判走 JSON，嵌套 map 值会落库为 `map[a:b]` 之类不可解析字符串；且无 key 白名单，任意 key 可写入任意分类；domain_name 从 general 重定向到 system 的特例（L67-69）又在 GetAdminSettings（L373-375）复制了一份，规则漂移风险高。
- **建议**: 统一用 json.Marshal 序列化所有非标量值；对每个分类维护允许的 key 集合；把 domain_name 重定向规则抽为共享函数。

### [MEDIUM] UpdateGeoIPDatabase 使用默认 http.Client（无超时）且 io.Copy 无大小限制，可能长时间挂起
- **位置**: 692-817  |  **类别**: performance  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: `http.Get(url)`（L747）走默认 client，无 timeout/context，配合 L786 无界 io.Copy，CDN 慢速或卡死时 handler 无限挂起（连接/内存占用泄漏）；另 L777-784 gzip.NewReader 失败提前 return 时未删除已创建的 tmp 文件；L798 os.Rename 跨文件系统（/tmp 与工作目录不同盘）会 EXDEV 失败。
- **建议**: 改用 `http.Client{Timeout: 2*time.Minute}` + `context.WithTimeout` + `io.Copy` 前限长（或 io.LimitReader 双倍估算）；gzip 初始化前先创建文件、失败统一 defer 清理；Rename 失败时回退 copy+remove。

### [MEDIUM] GetAdminSettings 把 API Token 类密钥明文返回前端
- **位置**: 270-408  |  **类别**: security  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: backup_gitee_token/backup_github_token（L307-310）、repo_sync_token（L314-316）、admin_telegram_bot_token/admin_bark_device_key（L332-336）等以明文值下发浏览器（虽挂了 NoStoreMiddleware 不缓存，但仍驻留 JS 内存并可能进日志/网络抓包）；GetSystemConfigs（L147-162）同样会全量返回含密钥的配置。
- **建议**: 对 token 类字段返回掩码（如 `****` + 是否已配置标志），保存时用单独的“仅写”端点；GetSystemConfigs 增加敏感字段过滤。

### [LOW] GetMobileConfig 返回非标准响应结构（message/code 与 camelCase 混排），与其他接口信封不一致
- **位置**: 949-978  |  **类别**: architecture  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 该公共端点返回 `{"message":"OK","code":1, "baseURL":...}` 自定义形态，与全局 utils.SuccessResponse 信封、snake_case 字段风格不一致，若被后续维护当作统一契约会踩坑（疑似旧移动端兼容遗留）。
- **建议**: 加注释标注 legacy 兼容用途并冻结契约；新前端统一走标准信封，逐步下线该端点。

### [LOW] UpdateGeoIPDatabase 与 SwitchGeoIPDatabase 手写重复的管理员校验
- **位置**: 692-704, 887-899  |  **类别**: duplication  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 两处都是 `c.Get("user_id")` → 查 User → 判 IsAdmin 的 ~10 行重复（L693-704 / L888-899），而路由本身已在 admin 组（AuthMiddleware+AdminMiddleware）保护下，属于冗余且风格与其余 handler（GetCurrentUser 判断）不一致。
- **建议**: 抽 `requireAdmin(c) bool` 助手统一鉴权判定，或直接信任路由中间件删除手写校验。

### [LOW] sendTestNotification 在 goroutine 里吞掉发送错误，测试接口永远报“已发送”
- **位置**: 651-690  |  **类别**: error-handling  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: `_ = notification.NewNotificationService().SendAdminNotification("test", testData)`（L688）错误被丢弃且无日志，Telegram/Bark/Email 三个测试端点无论是否真的发出都返回“测试消息已发送，请检查…”，排障时极具误导性。
- **建议**: 改为同步执行并检查错误，失败时返回 500 + 错误详情；如确需异步，至少把错误 LogError 并返回“已入队”。

### [LOW] 分类名硬编码字符串与常量混用，GetAdminSettings 是 140 行巨型函数
- **位置**: 147-162, 270-408  |  **类别**: style  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: CatSystem/CatGeneral/CatAnnouncement/CatAdminNotification 有常量，但 "security"、"theme"、"node_health"、"protocol_filter"、"custom_package"、"backup"、"repo_sync"、"invite" 等散落硬编码；GetAdminSettings 内嵌全套默认值 + stringOnlyFields + 类型转换逻辑，职责过重。
- **建议**: 为每个分类定义常量并替换字符串；把默认值表、类型推断、合并逻辑拆成独立小函数/数据表，便于测试与增删分类。

## internal/api/handlers/coupon.go

### [MEDIUM] UpdateCoupon 中恒真 else-if 分支导致任意更新都会清空适用套餐
- **位置**: 424-428  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `if req.ApplicablePackages != "" { ... } else if req.ApplicablePackages == "" { coupon.ApplicablePackages = "" }` — else-if 条件恰是 if 的取反，恒为真：任何不携带 applicable_packages 字段的更新（如只改名称/状态）都会把已配置的适用套餐列表覆盖为空串，静默丢失限制配置。
- **建议**: 改为 `if req.ApplicablePackages != "" { coupon.ApplicablePackages = normalize(...) }`，只有显式传空串才清空（用 *string 指针区分"未传"与"传空"）。

### [LOW] VerifyCoupon 用字符串包含判断错误类型，脆弱且依赖错误文案
- **位置**: 78-85  |  **类别**: architecture  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `if strings.Contains(err.Error(), "不存在") { status = http.StatusNotFound }` — 把 discountService 的错误文案当 API 契约，服务端改一行文案就改变客户端看到的 HTTP 状态码。
- **建议**: discountService 定义哨兵错误（如 ErrCouponNotFound），用 errors.Is 判断；文案仅用于 message。

### [LOW] UpdateCoupon 的 bind 失败日志标签误写为 "CreateCoupon: bind JSON failed"
- **位置**: 360  |  **类别**: duplication  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: UpdateCoupon（line 359-363）与 CreateCoupon（line 200-202）几乎逐行重复，且日志前缀是复制粘贴残留，线上按关键字搜 UpdateCoupon 的入参错误会全部落到 CreateCoupon 名下。
- **建议**: 修正日志前缀为 "UpdateCoupon: bind JSON failed"；将两处 bind+error 响应抽成公共辅助函数。

### [LOW] CreateCoupon 唯一码存在 TOCTOU：预检查与 Create 之间可能撞码
- **位置**: 221-237  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: 自动生成码时先 `db.Where("code = ?").First` 再 Create（line 221-230），两语句间并发下可能撞 uniqueIndex，Create 失败只返回 500 而不重试；20 次重试上限虽可接受，但错误信息不区分"撞码"与"其他错误"。
- **建议**: Create 失败时检查是否唯一约束错误（如 mysql 1062 / sqlite 约束），命中则继续重试生成而非直接 500。

### [LOW] 公开 GetCoupons/GetCoupon 返回完整 Coupon，泄露库存与用量数据
- **位置**: 20-31  |  **类别**: security  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `/coupons` 与 `/coupons/:code` 无需登录（router.go:189-190），却经自定义 MarshalJSON（models/coupon.go:107-109）输出 used_quantity、max_uses_per_user、min_amount、max_discount、total_quantity、applicable_packages 等运营字段——匿名用户可观察优惠券被领取/使用情况。
- **建议**: 为公开接口定义精简 DTO（仅 code/name/description/type/discount_value/valid 期/状态），运营字段只在 /coupons/admin 返回。

## internal/api/handlers/custom_node.go

### [HIGH] UpdateCustomNode 部分更新时 normalizeCustomNodeConfig 会用旧列值覆盖新配置
- **位置**: 602-607  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 仅更新 Config 字段时（如管理员粘贴新链接/新服务器），node.Domain/node.Port/node.Protocol 仍是旧值且非空，normalizeCustomNodeConfig 中的 setStringInConfigMap(data, domain, "Server",...) / setIntInConfigMap(data, port, "Port",...) 会无条件把新配置 JSON 里的 Server/Port/Type 改写成旧的列值，导致配置被静默损坏。例如新配置 {\"Type\":\"vless\",\"Server\":\"new.com\",\"Port\":8443} 会被改回旧 domain/port/protocol。
- **建议**: 部分更新时不要用旧列值去规范化新配置：只有请求中显式提供了 Domain/Port/Protocol 字段（改用 *string/*int 指针或 fieldmask）才写入配置；或去掉 normalizeCustomNodeConfig 的写回逻辑，仅在解析时派生缺失字段。

### [HIGH] 恒真条件 if req.DisplayName != "" || req.DisplayName == "" 使 DisplayName 无条件被覆盖
- **位置**: 574  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 该条件恒为 true，导致 DisplayName 总是被 req.DisplayName 覆盖：前端未传 display_name 时会把库里已有的显示名清空；与上方 Name 的 `if req.Name != ""` 保护逻辑不一致，属典型复制粘贴死分支。
- **建议**: 改为 `if req.DisplayName != "" { node.DisplayName = req.DisplayName }`；若需支持显式清空，则改用 *string 指针区分“未传”与“传空串”。

### [HIGH] TestCustomNode/BatchTestCustomNodes 是假测试：不做任何连通性检查就置为 active 并伪造 100ms 延迟
- **位置**: 931-1056  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: TestCustomNode 仅解析 Config、要求 Server 非空，随后 `node.Status = "active"; db.Save(&node)` 并返回硬编码 `latency: 100 // 模拟延迟`。与 node.go 的 TestNode（走 node_health.NewNodeHealthService 真实探测并回写 status/latency/last_test）语义完全相反：管理界面点了“测试”后不可达节点也会被标记 active，污染用户订阅列表。BatchTestCustomNodes 同样伪造，且 L1054 返回的 "success" 用的是 `len(results)` 而不是已算出的 successCount，结果数恒等于总数。
- **建议**: 复用 node.go TestNode 的真实链路（nodeID>1000000 分支已实现真实测试），删除这两个假测试处理器或改为调用 node_health 服务；批量版用 CreateInBatches/批量 Save 替代循环内逐条 db.Save，并修正 success 计数。

### [MEDIUM] DeleteCustomNode/BatchDeleteCustomNodes 无事务且吞掉关联删除错误
- **位置**: 623-683  |  **类别**: error-handling  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 先 `db.Where(...).Delete(&models.UserCustomNode{})`（L637/L671）再删节点，两处删除的错误都被忽略（L637、L671 无 err 检查）；若节点删除失败，用户分配关系已丢失，数据进入不一致状态。
- **建议**: 把“删分配关系 + 删节点”放进 db.Transaction；检查每步 error 并在失败时回滚；受影响用户缓存清理可放到事务外统一执行。

### [MEDIUM] AssignCustomNodeToUser 缺少用户/节点存在性校验，且 check-then-create 存在竞态
- **位置**: 1161-1231  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 不像 BatchAssign 先校验存在性：userID 参数非法时 parseUint 返回 0，仍会创建 UserCustomNode{UserID:0, CustomNodeID:...}；CustomNodeID 也不校验是否存在。L1177-1181 先查后插无唯一约束保护（UserCustomNode 的 idx_user_node 是非唯一索引，见 models/custom_node.go），并发双击会插入重复分配。
- **建议**: 分配前校验用户与节点存在；用唯一索引 (user_id, custom_node_id) + upsert（clause.OnConflict DoNothing）替代 check-then-create；对 c.Param("id") 先 strconv 校验并返回 400。

### [MEDIUM] ImportCustomNodeLinks 无去重且逐条 db.Create（N+1 写入）
- **位置**: 404-541  |  **类别**: performance  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: importCustomNodesFromLinks 对每条 link 单独 `db.Create`（L532），无 seen map 去重：同一链接提交两次会插入两条重复节点；请求数组也无长度上限。而同文件 ImportCustomNodeSubscription（L461-470）已有 link 去重，两处行为不一致。
- **建议**: 统一为：先 seen map 去重，再批量构造 []models.CustomNode 用 db.CreateInBatches(_, 100) 一次写入；对 links 长度加上限（如 500）。

### [MEDIUM] BatchAssignCustomNodes 双重循环逐条 Create + 每用户 Save，无事务且吞错
- **位置**: 685-777  |  **类别**: performance  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: L743 对每个 (userID, nodeID) 组合执行一次 `db.Create(&userNode)`（O(U×N) 条 SQL），且 `if err := db.Create(...).Error; err == nil` 只计数不报错；L764 对每个用户再 `db.Save(u)`。整体不在事务内，中途失败产生部分分配。另外 L703/L710 用 count 与 len 比较判“部分不存在”，请求数组含重复 ID（如 [1,1]）会误报“部分节点/用户不存在”。L771 joinUserNames(db, req.UserIDs) 又对 users 表重复查询——L724-729 已查出 userMap。
- **建议**: 构造 all 新关系用 CreateInBatches 一次写入并收集失败数；用户字段更新用 Updates(map) 批量；整段包事务；对 NodeIDs/UserIDs 先去重再校验；日志拼接直接复用 userMap。

### [LOW] GetCustomNodeLink 的 NodeConfig 回退分支基本不可达，且重复创建 service
- **位置**: 1058-1114  |  **类别**: duplication  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: json.Unmarshal 对合法 JSON 几乎不会返回错误（Go 的 JSON 匹配大小写不敏感），因此先试 ProxyNode 后“仅在 err!=nil 时”走 NodeConfig 分支（L1080-1101）在配置格式合法时永远走不到；且两个分支各自 `config_update.NewConfigUpdateService()`。
- **建议**: 改为“解析成功但关键字段为空时回退”的判定（参照 node.go GetNodeLink 的 `proxyNode.Server == ""` 回退逻辑），service 提升到函数顶部创建一次。

### [LOW] GetCustomNodeUsers 与 BatchGetCustomNodeUsers 的用户组装逻辑整段复制
- **位置**: 114-163  |  **类别**: duplication  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 两处 6 字段 gin.H 组装 + `if un.User.ID != 0` 过滤完全相同（L114-126 vs L151-163），未来加字段极易漏改一处。
- **建议**: 抽 `func buildCustomNodeUserView(un models.UserCustomNode) (gin.H, bool)` 共用。

### [LOW] parseUint 静默吞解析错误返回 0
- **位置**: 1268-1271  |  **类别**: error-handling  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: `i, _ := strconv.ParseUint(s, 10, 32); return uint(i)` 对非数字入参返回 0，被用于创建 UserCustomNode（L1184）、清缓存（L1230）、审计日志（L1243-1248），错误 ID 会被静默当作“用户0”处理。
- **建议**: 返回 (uint, error) 或在校验失败时直接 ErrorResponse(400)，不要在业务路径里吞错。

### [LOW] 批量操作的缓存/字段重置循环为每用户 3-4 条 SQL
- **位置**: 1313-1366  |  **类别**: performance  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: clearUserCustomNodeCache（查订阅 + 逐订阅清缓存）与 resetSpecialNodeFieldsIfNoCustomNodes（count + find + save）在 BatchDelete/BatchUnassign/Migrate 的 per-user 循环里被调用（L644-647、L678-681、L820-823、L917-920），100 个用户就是 300+ 条查询。
- **建议**: 批量查询受影响订阅与用户一次，再循环清缓存；resetSpecialNodeFields 的 count 改为一次 `GROUP BY user_id` 聚合后按需更新。

### [LOW] GetCustomNodes 的 size 无上限（对比 GetAdminNodes 封顶 100）
- **位置**: 72-79  |  **类别**: performance  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: `size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))` 只做 size<1 回退，无上限；size=1000000 会把整表查出并序列化。另外 L36-42 的 is_active 过滤把除 "true" 外的任意值都当 false。
- **建议**: 加 `if size > 100 { size = 100 }`；is_active 用 `strconv.ParseBool` 解析失败时忽略该过滤条件。

## internal/api/handlers/dashboard.go

### [HIGH] 用户面板按 category="system" 查公告，而公告实际存于 category="announcement"，公告永不显示
- **位置**: 115-128  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: GetUserDashboard 用 `db.Where("category = ? AND key IN ?", "system", []string{"announcement_enabled","announcement_content"})`（L118）读取公告；但系统写入/读取公告的规范分类是 CatAnnouncement（"announcement"）：updateSettingsCommon(c, CatAnnouncement)（config.go L416）、GetPublicSettings 校验 conf.Category == CatAnnouncement（config.go L571-573）、GetAdminSettings 默认值也在 CatAnnouncement 下。全仓 grep 无任何向 "system" 分类写公告的代码，因此用户仪表盘 notice.enabled 恒为 false、content 恒为空。
- **建议**: L118 的 category 改为 CatAnnouncement（"announcement"），与 GetPublicSettings 保持同一存储口径；并补一条回归测试断言两处读同一分类。

### [HIGH] GetAbnormalUsers 全表聚合 + 内存分页：多轮 GROUP BY 全表扫描且无 DB 分页
- **位置**: 299-436, 652-719  |  **类别**: performance  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: collectAbnormalUserCandidateIDs 对全库调用 loadAbnormalUserRiskMetrics(db, nil, ...)（L399），后者执行 7+ 个 GROUP BY 聚合（订阅、重置、设备、IP、地区、登录失败），其中 login_attempts 用 `lower(username)=lower(email) OR lower(...)` 关联（L711），lower() 使索引失效做全表扫描；随后 L368 把全部候选用户一次性 `Find(&users)` 加载进内存，再在 L371-373 过滤+分页。用户量大时单请求触发 10+ 条重量级 SQL 与整表内存拷贝。
- **建议**: 候选挖掘下沉为一条 UNION 聚合 SQL（或物化窗口），DB 层直接分页（LIMIT/OFFSET + COUNT）；login_attempts 关联改为索引友好的规范化字段（存 user_id 或用函数索引）；risk 指标对已分页用户子集二次聚合。

### [MEDIUM] GetUserDashboard 响应同时输出 camelCase 与 snake_case 两组字段，契约膨胀
- **位置**: 130-190  |  **类别**: architecture  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 顶层与 subscription 块同时包含 clashUrl/universalUrl/expiryDate（camelCase）与 expire_time/subscription_status/remaining_days/has_special_nodes（snake_case），其中 expire_time 与 expiryDate 各出现两次（L152-153、L162-163）；同一语义字段 4 份键，前端任取其一，后端维护成本高且易漂移。
- **建议**: 与前端约定单一命名风格（建议统一 snake_case），旧键经一段时间兼容期后下线；将 URL 组与 subscription 组抽成构建函数消除重复。

### [MEDIUM] MarkUserNormal 只翻 is_active/is_verified，指标型异常会立刻复发且无审计日志
- **位置**: 438-458  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 异常判定基于时间窗内的订阅/重置/登录失败计数与设备超限（L515-577），标记“正常”并不清除这些历史计数，用户刷新列表大概率再次出现在异常名单（如设备仍超限、重置次数仍在窗内）；同时该操作不写审计日志，与 DeleteUser/ResetPassword 等管理动作不一致。
- **建议**: 标记正常时同步处理可清除的指标（如设备超限解除、重置计数清零或设置豁免标记），并调用 CreateAuditLogSimple 记录操作人。

### [LOW] GetDashboard 原始 SQL 用北京当前时间与库中 expire_time 直接比较，时区口径未归一
- **位置**: 203-207  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: `expire_time > ?` 传入 utils.GetBeijingTime()（L206-207），若数据库按 UTC 存储 expire_time，统计会系统性偏移 8 小时；dashboard.go L48 用 ToBeijingTime 归一化而此处直接比，两处口径不一致。
- **建议**: 统一约定（库内统一存北京时或 UTC）并在 SQL 比较前做 ToBeijingTime/UTC 转换；为该统计加 60s 级缓存。

### [LOW] GetUserDashboard 单请求约 8-10 条 SQL 且无缓存
- **位置**: 20-193  |  **类别**: performance  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: user、user_level、subscription、device count、custom node count、announcement、CalculateUserPaymentSummary（内部多条统计 SQL）串行执行，且该接口是用户每次进入面板都调用的高频接口，未做任何 Redis 缓存。
- **建议**: 按 user_id 加短 TTL（如 30-60s）缓存聚合结果，订阅/设备/余额相关键变更时主动失效（项目已有 cache_service，可复用）。

### [INFO] GetRecentUsers/GetRecentOrders 无明显问题
- **位置**: 220-269  |  **类别**: other  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 均带 Limit(10)、错误检查、Preload("User") 避免 N+1、统一走 utils.SuccessResponse 信封，未发现逻辑或安全问题。
- **建议**: 无需修改。

## internal/api/handlers/device.go

### [MEDIUM] 事务内 Count 错误被忽略，current_devices 可能被写成错误值
- **位置**: 135-145, 169-179, 219-231  |  **类别**: error-handling  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: DeleteDevice（140 行）、RemoveDevice（174 行）、BatchDeleteDevices（225 行）中 tx.Model(&models.Device{}).Where(...).Count(&count) 的 error 全部被丢弃；若 Count 失败，count 保持 0，随后 Update("current_devices", 0) 会把订阅的设备计数清零，直接影响设备上限强制逻辑。
- **建议**: Count 后检查 err 并 return，失败则回滚整个事务。

### [LOW] 循环内重新定义 getString/formatIP 闭包，包级已有同名函数
- **位置**: 41-69  |  **类别**: duplication  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GetDevices 循环内每次迭代都新建 getString 与 formatIP 闭包（41-46、58-69），而 subscription.go:96-114 已存在包级 getString 与 formatIP（用 strings.HasPrefix 更健壮）。重复实现导致两处逻辑可能漂移（本文件的 ip[:7] 切片写法不如 HasPrefix 清晰）。
- **建议**: 删除循环内闭包，直接复用包级 getString/formatIP；循环内只保留业务拼装。

### [LOW] GetDeviceStats：SQL 错误被忽略，total 与 active+inactive 可能对不上
- **位置**: 241-276  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: db.Raw(...).Scan(&stats)（252-258）与 db.Model(&models.Subscription{}).Count（259）的错误都被忽略；inactive_devices 用 is_active = false 统计，而 total_devices 用 COUNT(*)，若存在 is_active 为 NULL 的行（GORM bool 默认非空，但历史脏数据可能），total != active + inactive。
- **建议**: 检查 Scan/Count 错误；inactive 改为 SUM(CASE WHEN is_active IS NULL OR is_active = false ...) 或对三值做一致处理。

### [LOW] 备注长度用 len() 按字节计，与提示"200个字符"不符
- **位置**: 316-319  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: len(req.Remark) > 200 按 UTF-8 字节数判断，中文一个字符占 3 字节，用户只能填约 66 个汉字却提示 200 字符；同时模型列 varchar(255)，200 字节限制与列宽不匹配。
- **建议**: 用 utf8.RuneCountInString(req.Remark) 计算字符数，并同步放宽模型列宽或校验上限。

### [LOW] GeoIP 查询整段被注释掉，location 永远为空字符串
- **位置**: 74-81  |  **类别**: maintainability  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 74-81 行是一大段注释掉的 GeoIP 代码，location 恒为 ""，但响应仍输出 "location" 字段；既无功能也无用途，读者无法判断是待启用功能还是遗留。
- **建议**: 删除注释块；若需按 IP 查位置，走单条查询接口或在服务层统一实现并输出 null。

### [LOW] GetDevices 无分页且 Pluck 错误被忽略
- **位置**: 15-38  |  **类别**: performance  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 订阅设备列表一次性全量返回（32-37 行无 Limit/Offset），历史设备不清理时列表无限增长；25 行 Pluck 的 error 被忽略，DB 异常时静默返回空列表，用户看到"无设备"假象。
- **建议**: 加分页（page/size）或按设备上限截断；检查 Pluck 错误并返回 500。

### [LOW] RemoveDevice/BatchDeleteDevices 仅靠路由中间件鉴权，handler 内无任何校验
- **位置**: 152-184, 186-239  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: RemoveDevice 与 BatchDeleteDevices 不校验当前用户身份/管理员角色（当前靠 router.go:346-347 的 AuthMiddleware+AdminMiddleware 兜底）。一旦未来被错误挂到别的路由组，就是完全开放的任意设备删除；批量删除还接受任意 device_ids 列表。
- **建议**: 在 handler 内显式校验管理员身份（如 middleware.GetCurrentUser + IsAdmin），形成纵深防御；批量删除前对 id 数量设上限。

## internal/api/handlers/download.go

### [HIGH] 未认证的下载解析接口 = SSRF + 开放重定向组合漏洞
- **位置**: 22-59  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GET /api/v1/download/resolve（router.go:305，无任何中间件）接收任意 target，服务端逐个发起 HEAD/GET 探测（42-48 行 isDownloadURLReachable），随后 302 到探测成功的 URL。攻击者可：(1) SSRF——探测内网/云元数据（169.254.169.254）、内网端口存活，作为内网可达性探针；(2) 开放重定向——构造 https://evil.example 让受害者经本服务跳转钓鱼页；(3) 放大——每个请求派生 len(prefixes) 个并发出站请求。
- **建议**: 该接口改为仅对白名单下载源生效：校验 target 域名属于站点配置的软件源（如 github.com 等），禁止 IP/内网段/元数据地址；出站请求用 DialContext 拒绝私有网段；或干脆由前端直连、服务端不做探测。

### [MEDIUM] 每次请求派生 N 个 goroutine 且 ctx 取消不生效，探测无法提前终止
- **位置**: 42-48, 160-186  |  **类别**: performance  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: resultCh 虽带缓冲不泄漏，但 select 命中或 ctx.Done 后其余 goroutine 仍各自跑完 2.5s 超时（isDownloadURLReachable 的 client 是独立 Timeout，不用外层 ctx），攻击者可并发打满出站连接；代理前缀列表来自 system_configs（61-77 行），管理员可配任意多条，进一步放大。
- **建议**: 给 http.Client 传入 c.Request.Context() 派生的 ctx；限制候选数（如 ≤5）；对 /download/resolve 加频率限制。

### [MEDIUM] 探测请求默认跟随重定向且不二次校验协议，SSRF 可经跳板到达任意地址
- **位置**: 160-186  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: http.Client 默认跟随最多 10 次重定向；初始校验只检查 target 前缀（30 行），重定向后可落到 http://127.0.0.1:xxx 等任意内网地址；HEAD 失败还会降级 GET bytes=0-0（175-180 行），对 SSRF 目标多一种探测方式。
- **建议**: 自定义 CheckRedirect 限制同源/同协议，禁止跳到私有网段；不提供 GET 降级（或仅对白名单域名降级）。

### [LOW] 仅校验 http/https 前缀，明文 http 与畸形 URL 未过滤
- **位置**: 30-33  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: strings.HasPrefix 校验可被 "http://evil.com\nLocation:..." 之类畸形串绕过部分网关过滤（重定向 Header 注入风险依赖 Go 标准库对 CRLF 的处理，Go 会拒绝非法头，风险较低），且允许明文 http 流量。
- **建议**: 用 url.Parse 严格解析并校验 Scheme ∈ {http,https}、Host 非空、无 userinfo；至少要求 https。

## internal/api/handlers/email_template.go

### [INFO] 无明显问题：管理端邮件模板查询（两点小建议）
- **位置**: 13-37  |  **类别**: other  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 两个接口均挂在 admin 组（router.go:563-564），无越权；逻辑简单无错误路径遗漏。小点：每次请求都全量查模板表无缓存，且 GetEmailTemplateByName 的 name 参数来自路径，url 编码的斜杠可能被路由截断（Gin 默认会处理），风险低。
- **建议**: 可选：对模板做进程内 TTL 缓存（模板变更时失效），降低高频邮件渲染时的 DB 压力。

## internal/api/handlers/geoip.go

### [LOW] GeoIP 查询接口无认证、无速率限制，可被当作免费地理位置探测源
- **位置**: 13-36, 39-67  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 路由 /api/v1/geoip/lookup 与 /batch-lookup（router.go:169-170）无任何中间件，匿名用户可无限批量查询任意 IP 的地理位置（最多 100 个/次），既泄露服务端 GeoIP 数据能力，也容易被打爆缓存/DB 连接。
- **建议**: 至少挂 TryAuthMiddleware + 速率限制（如复用 ratelimit 中间件），或限制每日匿名配额。

### [LOW] 命中/未命中的响应结构不一致
- **位置**: 23-34  |  **类别**: style  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 查询失败返回 {"location": nil, "message": "..."}（24-28 行），成功返回 {"location": "xx"}（32-35 行）——location 在两种情况下类型不同（null vs string），前端需特判。
- **建议**: 统一为 {"ip":..., "location": "" 或 nil} + 可选 message，保持字段类型一致。

## internal/api/handlers/invite.go

### [MEDIUM] 同一资源不同接口的 JSON 契约不一致：sql.NullInt64/NullTime 原样序列化
- **位置**: 124-136, 317-328  |  **类别**: architecture  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: CreateInviteCode 响应（124-136）和 ValidateInviteCode 响应（317-328）直接把 inviteCode.MaxUses（sql.NullInt64）和 inviteCode.ExpiresAt（sql.NullTime）塞进 gin.H；这两个类型无自定义 MarshalJSON（internal/core/database 中 NullTime/NullInt64 只是 sql 类型构造函数），序列化结果是 {"Int64":0,"Valid":false} / {"Time":...,"Valid":...}。而 GetInviteCodes（159-167）却手动格式化成 int/字符串。同一字段在三个接口返回三种形态，前端必须分别兼容，极易踩坑。
- **建议**: 为 sql.NullInt64/sql.NullTime 统一封装带 MarshalJSON 的类型（或抽一个 formatMaxUses/formatExpiresAt 工具），所有接口统一输出 int/null 与格式化字符串；或直接复用 GetInviteCodes 的格式化逻辑。

### [LOW] 邀请码查重循环逻辑冗余且有竞态盲区
- **位置**: 83-95  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 先查 code = ? 再查 UPPER(code) = ?（87-89）是冗余的：生成的 code 已是大写，两查询结果等价；且若第一次查询因非 RecordNotFound 的 DB 错误返回（86 行 else 分支不区分"找到"与"出错"），第二次查询若返回 ErrRecordNotFound 就直接 break，可能放行一个实际已存在/冲突的 code。另外查重与 db.Create 之间无锁（TOCTOU），并发下靠 Code 列的 uniqueIndex（models/invite.go:10）兜底，撞码时直接返回"创建邀请码失败"500，对用户不友好。
- **建议**: 删掉 UPPER 二次查询；区分"查库出错"与"已存在"；撞 uniqueIndex 时在错误分支重试 GenerateInviteCode 而非直接 500。

### [LOW] GenerateInviteCode 忽略 rand.Read 错误且熵被截断
- **位置**: 21-25  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: rand.Read(b) 的 error 被忽略：失败时 b 全零，会生成 "AAAAAAAA" 这类固定 code；且 base64.URLEncoding.EncodeToString(b)[:8] 取前 8 字符后再 strings.ToUpper，字符集被压缩到约 37 个，8 位码实际熵仅约 41 bit，10 次重试下高并发撞码概率不可忽略。
- **建议**: 检查 rand.Read 错误并重试/报错；改为用 8 字节随机数直接编码完整 11 字符，或用 crypto 安全的字母数字生成器保证 8 位全大写字母+数字。

### [LOW] 奖励金额/使用次数无任何输入校验
- **位置**: 34-43, 53-71  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: InviterReward/InviteeReward/MinOrderAmount 完全由请求方传入，可为负值或极大值（负数奖励会让被邀请人扣余额）；MaxUses 为 0 时被当作"不限次"（108-110 行只在大千 0 时赋值），与前端语义可能冲突；也无 maxUses 上限。
- **建议**: 服务端对 reward 做 >=0 与上限校验，对 max_uses 做 1~N 校验，拒绝非法值返回 400。

### [LOW] GetMyInviteCodes 是无意义转发壳
- **位置**: 285-287  |  **类别**: maintainability  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: func GetMyInviteCodes(c *gin.Context) { GetInviteCodes(c) } 直接转发，且路由（router.go:268）同时注册了 GET /invites 与 GET /invites/my-codes 两个入口，行为完全一致，属冗余暴露。
- **建议**: 删除该转发函数及 /my-codes 路由（或保留路由但由前端统一调用 /invites）。

## internal/api/handlers/knowledge.go

### [MEDIUM] 管理端分页参数未校验：page_size=0 时 GORM Limit(0) 等价于不限制，返回全表
- **位置**: 118-141  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GetAdminKnowledgeArticles 用 strconv.Atoi(c.DefaultQuery(...)) 且不检查错误与范围（120-121 行）：传 page_size=0 → Limit(0) 被 GORM 视为"取消限制"→ 一次性返回全部文章；传非数字 → page=0 → Offset 为负 → 查询报错但 .Find 错误被忽略（134 行）静默返回空。GetAdminPromotions（promotion.go:29-36）同样问题。
- **建议**: 统一用 utils.ParsePagination（已封顶 100/10000），并检查 Find/Count 的错误。

### [MEDIUM] 文章列表硬编码 Limit(200) 无分页，且对 content 做 LIKE 全表扫
- **位置**: 23-43  |  **类别**: performance  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GetKnowledgeArticles 固定 Limit(200)（41 行）无分页参数；keyword 时 title LIKE 或 content LIKE '%kw%'（36 行）——content 是大字段，无索引，全表扫描；列表还返回完整 Content 字段，载荷巨大。
- **建议**: 支持 page/size 分页（复用 utils.ParsePagination）；列表查询 Select 掉 content 大字段（详情接口再取）；搜索只对 title 建索引或引入全文检索。

### [MEDIUM] 公开文章详情接口不过滤 is_active，草稿/下架文章可被直接读取
- **位置**: 46-57  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GetKnowledgeArticle 用 db.Preload("Category").First(&article, id)（51 行）不带 is_active = true 条件，而该路由是公开的（router.go:295，无任何中间件）；管理员建了草稿或下架的文章，只要知道 ID（自增可枚举）就能读到全文，越权泄露未发布内容。
- **建议**: 查询加 .Where("is_active = ?", true)；管理端预览走独立 admin 接口。

### [MEDIUM] 创建/更新直接 ShouldBindJSON 到模型，存在批量赋值（mass assignment）风险
- **位置**: 69-100, 144-175  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: CreateKnowledgeCategory/Article、UpdateKnowledgeCategory/Article 直接把请求体绑定进 models 结构体（71、93、146、168 行），客户端可注入 ID、CreatedAt、ViewCount、IsActive 等任意字段；Update 分支更危险：绑定发生在 First 之后（93 行），请求里带 id 字段可改主键、带 view_count 可篡改统计。
- **建议**: 定义独立的 Request DTO（只含可控字段白名单），再映射到模型；ID 等字段禁止出现在 DTO。

### [LOW] 删分类的级联删文章不在事务内，且 delete 错误全部被忽略
- **位置**: 103-115  |  **类别**: error-handling  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: DeleteKnowledgeCategory 先删分类再删文章（111-112 行）无事务包裹；若第二步失败，留下孤儿文章且接口仍返回"删除成功"。db.Save/db.Delete 的错误在多处被忽略（97、111-112、172、186 行），失败时用户收到成功响应。
- **建议**: 用 database 事务包裹两步删除并检查错误；所有写操作检查 err 并返回 500。

### [LOW] view_count 自增非原子，并发浏览丢失计数
- **位置**: 56  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: db.Model(&article).Update("view_count", article.ViewCount+1) 是读-改-写，两次并发浏览可能都读到同一 ViewCount 只 +1；且 Update 错误被忽略，浏览量统计失真。
- **建议**: 改为 db.Model(&models.KnowledgeArticle{}).Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + 1"))。

## internal/api/handlers/logs.go

### [MEDIUM] GetAuditLogs 忽略 Count/Find 错误且分页响应形状与其他日志接口不一致
- **位置**: 509-514  |  **类别**: error-handling  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `query.Count(&total)` 与 `query.Find(&logs)` 的错误被完全忽略，数据库故障时返回空列表 + HTTP 200；同时 GetAuditLogs 返回 `{logs,total,page,page_size}`（line 564-569），而 GetSystemLogs 返回 `{logs,total,page,size}`（line 642-647），genericSuccessResponse 又带 `total_pages`，三种分页信封并存。
- **建议**: 补上两个 `.Error` 检查并按 GetSystemLogs 模式统一错误响应；抽出统一的分页信封函数（含 total_pages），删除 genericSuccessResponse 中关于 "attempts" 的死注释。

### [MEDIUM] GetLogsStats 对同一作用域执行 4 次独立 COUNT，且基础作用域无法走索引
- **位置**: 650-669  |  **类别**: performance  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `systemLogsBaseWhere`（line 20）由 `action_type = ? OR action_type LIKE 'scheduler_%' OR 'system_%' OR 'security_%' OR 'business_%' OR resource_type = ...` 组成，LIKE 前缀条件无法利用普通索引；GetLogsStats 在同一 baseQuery 上再跑 4 次 Count（total/error/warning/info），加上 GetSystemLogs 的 Count+Find，单次页面加载对超大 audit_logs 表产生 5~6 次全表扫描。
- **建议**: 改为单条 `SELECT COUNT(*) total, SUM(CASE WHEN status>=400 OR action_type='system_error' THEN 1 ELSE 0 END) error, ... FROM audit_logs WHERE <scope>` 一次聚合；并评估 `(action_type, created_at)` 复合索引。

### [MEDIUM] ClearLogs 直接 DELETE 全表审计日志，无确认/归档/软删除
- **位置**: 696-707  |  **类别**: security  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `db.Where("1 = 1").Delete(&models.AuditLog{})` 清空整个 audit_logs 表（含安全事件 security_*、登录失败等合规记录），没有任何二次确认参数或归档步骤；清空动作本身虽在删除后补写了一条审计，但管理员可借此销毁全部历史痕迹。
- **建议**: 增加 `confirm=true` 请求参数、限制为单日/单模块范围删除，或删除前先 INSERT INTO audit_logs_archive SELECT 归档；同时为删除操作单独记录 before/after 数据。

### [LOW] applyAuditLogFilters 与 applySystemLogsFilters 存在约 60% 重复逻辑
- **位置**: 80-177  |  **类别**: duplication  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: module/username/keyword/时间范围四个过滤块在两个函数中几乎逐行重复（line 92-110 vs 159-176）；GetBalanceLogs/GetCommissionLogs/GetSubscriptionResetLogs 中内联 `func() string {...}()` 提取 order_no 的写法也重复出现两次（line 956-961、1038-1043）。
- **建议**: 抽取 `applyCommonAuditFilters(query, c, tablePrefix)` 公共函数；抽取 `orderNoOf(log)` 辅助函数替换两处 IIFE。

### [LOW] applyTimeRangeFilter 对非法时间参数静默丢弃过滤条件
- **位置**: 44-56  |  **类别**: error-handling  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `if t, err := time.ParseInLocation(TimeLayout, startTime, ...); err == nil { ... }` — start_time/end_time 格式错误时过滤被静默忽略，接口返回全量数据而非报错，用户难以察觉筛选未生效。
- **建议**: 解析失败时返回 400（"时间格式错误，应为 YYYY-MM-DD HH:MM:SS"），并考虑兼容纯日期 "2006-01-02" 输入。

### [LOW] 搜索关键词被改写且 EscapeLikePattern 在 SQLite 下无效
- **位置**: 103-107  |  **类别**: security  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: utils.SanitizeSearchKeyword（validator.go:29-53）会从关键词中删除 "select"/"union"/"exec" 等子串——搜索 "selected" 会变成 "ed"，属于针对参数化查询多余的 SQL 注入防御 theater；同时 EscapeLikePattern 仅转义 `%`/`\`，而 SQLite 的 LIKE 默认转义字符不是反斜杠，关键词里的 `%` 仍会当通配符生效（该问题波及 logs.go/package.go/recharge.go 所有 LIKE 搜索）。
- **建议**: 删除 SanitizeSearchKeyword 的关键字剥除逻辑（参数化查询已防注入），保留长度截断与字符白名单；LIKE 查询统一加 `ESCAPE '\'` 子句以在三类数据库上行为一致。

### [LOW] formatLogForCSV 的 db 参数从未使用，CSV 字段转义不完整
- **位置**: 473-496  |  **类别**: style  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `func formatLogForCSV(db *gorm.DB, log models.AuditLog) string` 的函数体从不使用 `db`；CSV 仅对 message 字段做了引号/换行转义，其余字段（resource_type、username 等）未加引号，一旦内容含逗号即破坏列对齐。
- **建议**: 删除未用的 db 参数；对全部字段统一 `"..."` 包裹并转义内部引号，或改用 encoding/csv 的 Writer。

### [LOW] 可空值辅助函数语义不一致：getNullableInt64 返回 interface{}（nil）而 getNullableString 返回 ""
- **位置**: 194-213  |  **类别**: style  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: 同一文件内 `getNullableString(sql.NullString)` 返回空串、`getNullableInt64(sql.NullInt64)` 返回 nil（interface{}），导致 JSON 输出中同是空值有时是 `""` 有时是 `null`；GetEmailLogs（line 1168-1169）还同时输出 `to_email` 与 `recipient_email` 两个重复键。
- **建议**: 统一策略（建议全部返回空串或全部返回 nil），删除冗余的 `recipient_email` 键，前端只依赖一个字段名。

## internal/api/handlers/monitoring.go

### [INFO] 无明显问题：仅管理端可用的轻量监控接口（附两处小建议）
- **位置**: 13-55  |  **类别**: other  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GetSystemInfo/GetDatabaseStats 均在 admin 组（router.go:521-522，含 Auth+Admin 中间件），逻辑简单正确；唯一小点：runtime.ReadMemStats 会短暂 STW，对管理端低频调用无影响；GetDatabaseStats 在 Ping 失败时 status 置 disconnected 但 HTTP 仍返回 200，前端需自行判断 body。
- **建议**: 可保持现状；如需更严格监控语义，可在 DB 异常时返回 503。无阻塞问题。

## internal/api/handlers/node.go

### [HIGH] findNodeIDsByKey/findExistingNode 全表加载 + 循环调用导致 O(N²) 全表扫描
- **位置**: 113-165  |  **类别**: performance  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: findNodeIDsByKey 用 `db.Find(&nodes)` 无任何过滤加载整个 nodes 表再在 Go 里逐条算 key（L126-139）；findExistingNode 加载该类型全部活跃节点（L113-124）。它们在 collectEquivalentNodeIDs 的循环（L155）、ImportNodeLinks 的链接循环（L620）、DeleteNode（L754）中被反复调用——1000 节点 × 100 链接 = 10 万次 key 计算与全表扫描。同文件 processAndImportLinks（L167-175）已经示范了正确的“一次加载进 map”模式，未被复用。
- **建议**: 统一封装 `buildNodeKeyIndex(db) map[string][]*models.Node` 一次加载；collectEquivalentNodeIDs、ImportNodeLinks、DeleteNode 全部改用该索引；findNodeIDsByKey 至少加 `is_active = ?` 过滤。

### [HIGH] TestNode/BatchTestNodes/ImportFromClash 仅挂 AuthMiddleware，任意登录用户可导入节点并翻转节点状态
- **位置**: 772-902, 1037-1052  |  **类别**: security  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: router.go L179-185 的 nodesAuth 组只有 `middleware.AuthMiddleware()`，无 AdminMiddleware：任何普通用户都能 (1) POST /nodes/import-from-clash 导入无限量节点并触发 clearNodeCaches()（数据污染+缓存雪崩 DoS）；(2) POST /nodes/:id/test 对任意节点（含 id>1000000 的专线节点）发起真实健康检查并回写 status/latency，node_health.UpdateNodeStatus 在 offline/timeout 时还会置 is_active=false（L281-285），可被用来把节点批量下线；(3) 这些处理器还写“管理员操作: ...”审计日志，审计主体与角色错位。
- **建议**: 给 nodesAuth 组补 `middleware.AdminMiddleware()`（或把这三个路由并入 admin 组）；审计日志记录真实操作者 user_id/username 而非硬编码“管理员操作”。

### [MEDIUM] 专线虚拟 ID 魔法数 1000000 散落多处且与真实节点 ID 可能碰撞
- **位置**: 325, 783  |  **类别**: architecture  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: GetNodes 用 `ID: cn.ID + 1000000`（L325），TestNode 用 `nodeID > 1000000` 判定专线（L783），BatchDeleteNodes 注释承认真实节点 ID 可能超过 1000000 并做了存在性回退（L933-959），但 GetNodes/TestNode/GetNodeStats/BatchTestNodes 没有同等防护——真实节点 ID ≥1000000 时前端 ID 冲突、测试路由误判为专线。
- **建议**: 定义 `const CustomNodeIDOffset = 1000000` 统一引用；在 GetNodes 组装时若发现真实节点 ID 已占用该偏移区间，跳过/报错而非静默碰撞。

### [MEDIUM] processAndImportLinks 吞掉批量写入错误并逐条 Save 已存在节点
- **位置**: 196-209  |  **类别**: error-handling  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: `if err := db.CreateInBatches(newNodes, 100).Error; err == nil { importedCount = len(newNodes) }`（L206-208）失败时 importedCount 静默为 0 且无任何日志；循环内对已存在节点逐条 `db.Save(existing)`（L201）是 N 次写。
- **建议**: CreateInBatches 失败时 LogError 并让调用方（ImportFromClash）感知；已存在节点的字段更新合并为批量 Updates 或至少记录失败。

### [MEDIUM] BatchTestNodes 不识别专线虚拟 ID（>1000000），且 body 解析与 BatchDeleteNodes 整段复制
- **位置**: 863-902, 904-922  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: TestNode 对 id>1000000 走专线分支（L783-838），但 BatchTestNodes 把 ID 原样交给 node_health.BatchTestNodes（只查 nodes 表，L234），前端从 GetNodes（专线 ID=cn.ID+1000000）拿到的 ID 批量测试会静默返回空结果——单测/批测语义不一致。另外 L867-881 与 L908-922 的“json.Unmarshal 失败后用 map 柔性解析 node_ids”是复制粘贴的 ~15 行。
- **建议**: 批测前复用 BatchDeleteNodes 的 ID 分流逻辑（L936-959），专线部分走 TestCustomNode 等价实现；把柔性 node_ids 解析抽成公共助手 `parseFlexNodeIDs(c) ([]uint, error)`。

### [MEDIUM] BatchDeleteNodes 混合批次时普通节点已删空会提前 return，跳过同请求中的专线节点删除
- **位置**: 984-1001  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 当 normalNodeIDs 非空但全部已被删（equivalentIDs 为空）时 L990-994 直接 SuccessResponse 返回，此时 customNodeIDs 里的专线节点尚未处理，被静默跳过——与注释宣称的“避免误判”意图相悖。
- **建议**: 把“普通节点集合已不存在”降级为仅跳过该集合（deletedCount 记 0 并继续），等 customNodeIDs 处理完再统一判断 deletedCount==0 返回提示。

### [MEDIUM] GetNodeStats 把专线节点无条件计入在线数，忽略 status 与到期过滤
- **位置**: 375-400  |  **类别**: logic  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: 对 special 用户 `stats.TotalNodes++; stats.OnlineNodes++`（L382-383）不看 cn.Status（error/offline 也计在线）也不看 cn.ExpireTime/FollowUserExpire 到期过滤；而 GetNodes（L288-299）会跳过到期专线节点——两个接口对同一用户给出的节点数与在线数口径不一致。
- **建议**: 复用与 GetNodes 相同的“是否对用户可见/未到期”判定逻辑（抽公共函数），并按 cn.Status == "active"/"online" 才计入 OnlineNodes。

### [MEDIUM] CreateNode/UpdateNode 直接绑定完整 models.Node，存在 mass assignment
- **位置**: 521-596, 646-671  |  **类别**: security  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: UpdateNode `c.ShouldBindJSON(&node)` 把请求体整体写回已加载的 model：客户端可在 body 里带 `id`（随后 db.Save 按新主键更新别的行或静默 0 行）、`created_at`、`order_index`、`latency`、`last_test`、`is_recommended` 等字段全部被采纳；CreateNode 手动路径（L533-537）同样绑定完整 struct，仅覆盖 status/is_manual/is_active，客户端可注入 order_index/latency 伪造健康数据。
- **建议**: 用白名单 DTO（参照 UpdateCustomNode 的字段级 if 更新）或 `db.Model(&node).Select("name","region",...).Updates(map)` 只更新允许字段；禁止更新 id/created_at。

### [LOW] GetNodes 重复调用 GetCurrentUser 且去重逻辑与 GetAdminNodes 复制
- **位置**: 242-262, 347  |  **类别**: duplication  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: L242 与 L347 各调一次 `middleware.GetCurrentUser(c)`；L226-240 与 L444-456 的 seenKeys 去重循环逐字重复；L248 `sub.ExpireTime.Before(utils.GetBeijingTime())` 未做 ToBeijingTime 归一化，与 dashboard.go L48 的转换用法不一致，时区口径存在 8 小时偏差风险。
- **建议**: 复用第一次取到的 user 指针；抽 `dedupeNodes(all []models.Node) []models.Node`；时间比较统一先 ToBeijingTime 再比。

### [LOW] regionMatcher 用 sync.Once 且失败后永不重试
- **位置**: 41-59  |  **类别**: error-handling  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: LoadRegionConfig 一次性失败（如文件瞬时不可读）后 regionMatcher 固定为空 matcher，所有节点地区都解析为“未知”，直到进程重启。
- **建议**: 失败时不清空已有 matcher 并允许下次调用重试（用 RWMutex + 懒加载而非 Once），或保留 Once 但失败时打 Warn 并回退默认。

### [LOW] #nosec G117 注解引用不存在的 gosec 规则，属于无效注释
- **位置**: 99-100, 312-313, 801-802  |  **类别**: style  |  **来源组**: A5-handlers-node-config (node/custom_node/config/dashboard)
- **问题**: gosec 并无 G117 规则（G101 硬编码凭据等才是有效编号），三处 `// #nosec G117 - Password field is proxy node password...` 不会抑制任何扫描器告警，纯装饰性且误导后续维护者。
- **建议**: 删除这些假注解，或按实际 gosec 编号修正；代理密码本身属业务必需，可统一在 config JSON 序列化处加一条说明注释。

## internal/api/handlers/notification.go

### [MEDIUM] sendNotificationEmail 裸 goroutine：无 panic 恢复、全量用户循环无上限
- **位置**: 78-98  |  **类别**: error-handling  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: go func() 内无 recover，QueueEmail 全部错误被忽略（87、93 行 _=）；广播时一次性 SELECT 全部 is_active 用户并逐条 QueueEmail（91-94），用户量大时该 goroutine 长时间占用连接且不可取消；emailService 每次调用都 new（80 行），若其内部含连接池会重复建池。
- **建议**: 加 defer recover；广播邮件改为分页游标取用户并受 ctx 取消控制；把 QueueEmail 交回统一邮件队列 worker，handler 只入队一条批量任务。

### [MEDIUM] 草稿状态静默失效：status=draft 且未传 is_active 时被当成已发布
- **位置**: 277-283  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: CreateAdminNotification 的状态判断：req.Status=="published" 或 req.IsActive!=nil&&*req.IsActive 才置 true；否则若 req.IsActive==nil 直接落到 else 分支 IsActive=true（默认发布）。管理员传 status="draft" 但不传 is_active 时，通知会被创建为 active，草稿直接对用户可见。
- **建议**: 显式处理 draft：req.Status==notifStatusDraft 时强制 IsActive=false；否则默认值逻辑只对 Status 为空生效。

### [MEDIUM] 全局广播通知（user_id IS NULL）永远无法标记已读，未读数与列表不一致
- **位置**: 100-116, 118-134, 136-158  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GetNotifications 会返回 user_id IS NULL 的广播（108 行），但 MarkAsRead（145 行）与 GetUnreadCount（127 行）只匹配 user_id = user.ID；DeleteNotification 同样（185 行）。结果是：广播在列表里出现但点"已读"报 404，未读角标永远不包含广播，前端体验与数据矛盾。
- **建议**: 为广播引入"用户级已读表"（notification_reads user_id+notification_id），MarkAsRead/UnreadCount/MarkAllAsRead 均基于该表计算；或至少允许对 user_id IS NULL 的通知写入已读状态。

### [LOW] parsePaginationParams 重复造轮子且 size 无上限
- **位置**: 42-58  |  **类别**: duplication  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: utils.ParsePagination（utils/response.go:139）已实现 page/size 解析且 size 封顶 100、page 封顶 10000；本文件 42-58 行又写了一份，且 size 无上限（size=1000000 时 GetAdminNotifications 一次性载入百万行）。
- **建议**: 删除本地 parsePaginationParams，改用 utils.ParsePagination；或至少给 size 加最大上限。

### [LOW] requireAuth/errorResponse/successResponse 与全局辅助函数重复
- **位置**: 25-40  |  **类别**: duplication  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 本文件自造 requireAuth、errorResponse、successResponse 三个包装（25-40），而 user.go:2700 已有 getCurrentUserOrError，其他 handler 直接调 utils.ErrorResponse。同一 handlers 包内四种取用户/回错误写法并存（notification.go 的 requireAuth、user.go 的 getCurrentUserOrError、checkin.go 的 c.Get("user_id").(uint)、invite.go 的 middleware.GetCurrentUser 直调）。
- **建议**: 统一收敛为一个包级辅助函数（如 getCurrentUserOrError），其余文件全部替换。

### [LOW] 多处 GORM 错误被忽略（Count/查询）
- **位置**: 198-201, 248-263  |  **类别**: error-handling  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GetAdminNotifications 的 query.Count(&total)（200 行）错误被丢弃；CreateAdminNotification 中 req.Content/Title 允许空字符串仅靠 binding:required（251-252 行）挡住 JSON 缺失，但传空串仍可入库。
- **建议**: Count 检查错误；Content/Title 增加 min 长度校验或服务端 trim 后非空校验。

### [LOW] UpdateAdminNotification：无法将定向通知改为广播，SendEmail 参数为死参数
- **位置**: 308-370  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: req.UserID 只支持"有值则设置"（353-359），无法把 user_id 清回 NULL 实现"改为广播"；且 Update 分支收到 SendEmail=true 时没有任何发信逻辑（301-303 行只有 Create 分支处理），参数被静默忽略。
- **建议**: UserID 改为 *uint 支持置 NULL（如传 null 时清空）；Update 分支实现 SendEmail 或从请求结构体中移除该字段。

## internal/api/handlers/order.go

### [HIGH] UpgradeDevices 支付链接生成失败后余额已被扣除且不退回
- **位置**: 1770-1841  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 事务（L1770-1826）内先创建订单并 `Update("balance", gorm.Expr("balance - ?", balanceUsed))` 扣余额，事务提交后（L1833）才在事务外调用 `generatePaymentURL`；若生成支付链接失败（L1837-1841），只把订单 MarkPendingOrderStatus 为 failed 并返回错误，已扣的 balanceUsed 没有任何回滚/退款逻辑——用户既没拿到升级，余额也没了。
- **建议**: 支付链接生成失败时在同一事务内回补余额并写余额日志（或引入独立的"余额扣减预留/释放"机制），保证失败路径与成功路径余额一致。

### [HIGH] CreateOrderRequest 结构体是无人引用的死代码
- **位置**: 780-788  |  **类别**: maintainability  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 全仓 grep 仅命中定义处（order.go:780），CreateOrder 实际绑定的是 `orderServicePkg.CreateOrderParams`（L797）。这个导出的 CreateOrderRequest 与真实请求参数（PackageID/CouponCode/PaymentMethod/Amount/UseBalance/Currency/DurationMonths）高度相似，极易误导后续维护者，属于遗留残留。
- **建议**: 删除 CreateOrderRequest 定义；若 CreateOrderParams 缺少字段，直接扩展 service 层结构体。

### [MEDIUM] GetOrder 返回原始模型，与 GetOrders 的 formatOrderData 形状不一致
- **位置**: 950-979  |  **类别**: architecture  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: GetOrders/GetAdminOrders 输出 `{orders:[formatOrderData...], total, page, size}`，而 GetOrder 直接 `utils.SuccessResponse(c, ..., order)` 返回原始 Order 模型（不含 user 信息、package 未格式化、amount 语义不同），同一资源的详情与列表 API 契约分裂，前端需维护两套字段映射，也更容易在列表"amount"（余额加回后）与详情"amount"（原值）之间产生歧义。
- **建议**: GetOrder 也走 formatOrderData（并 Preload User/Package/Coupon），保证详情与列表字段完全同构。

### [MEDIUM] GetOrder 中 isAdmin 断言未检查存在性，存在 panic 隐患
- **位置**: 958-959  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `isAdmin, _ := c.Get("is_admin"); admin := isAdmin.(bool)`（L958-959）直接类型断言；同文件 GetOrders（L868-869）做了 `exists` 检查。一旦路由中间件调整（如换 TryAuthMiddleware 或去掉 AuthMiddleware），GetOrder 会 panic 而非返回 401。
- **建议**: 与 GetOrders 保持一致：`isAdmin, exists := c.Get("is_admin"); admin := exists && isAdmin.(bool)`。

### [MEDIUM] RefundAdminOrder 无幂等保护：重复点击可二次退款，且网关退款后内部处理失败无对账
- **位置**: 1190-1307  |  **类别**: security  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `order.Status != "paid"` 检查与 `yipayService.RefundOrder` 调用之间无行锁/状态原子更新，两个并发请求都能通过检查并各自调用退款 API；若网关退款成功但 `ProcessRefundOrder` 失败（L1285），款项已退而订单仍 paid，管理员重试会再退一次，且没有 refund 流水表做对账。
- **建议**: 退款前置一个 refunds 记录表（状态机：requested→success/failed），用唯一键（order_id）幂等；网关退款前先占位/加锁，成功后事务内更新订单与退款记录，失败时留下可对账的半成品记录。

### [MEDIUM] ExportOrders 存在 CSV 公式注入且字段未转义
- **位置**: 1430-1491  |  **类别**: security  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 用户名/邮箱/套餐名等用户可控值通过 `fmt.Sprintf` 直接拼入 CSV（L1471-1484），无引号包裹、无 =+-@ 前缀处理；与 subscription.go 的 ExportSubscriptions 是同一类问题，Excel 打开可执行公式、含逗号字段破坏列对齐。
- **建议**: 统一封装一个安全的 CSV 写入工具（encoding/csv.Writer + 公式前缀转义），两个导出 handler 共用。

### [MEDIUM] 多处直接把内部错误文本 err.Error() 返回给前端
- **位置**: 818, 1829, 2189  |  **类别**: security  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: CreateOrder（L818）、UpgradeDevices（L1829 `txErr.Error()`）、CreateCustomOrder（L2189 `err.Error()`）等把 DB/服务层原始错误拼进响应 message，可能泄露表名、SQL 片段、支付网关内部信息；且同一错误既有 400 又有 500（L818 一律 400，L1829 一律 500），契约混乱。
- **建议**: 对外统一返回安全文案 + 稳定错误码，原始错误只进日志（utils.LogError），并约定业务错误 4xx / 系统错误 5xx 的映射规则。

### [LOW] 订单状态过滤同时接受英文与中文值，契约混乱
- **位置**: 881-897  |  **类别**: architecture  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: statusMap 把中文（待支付/已支付/已取消/已过期）映射到英文状态，同时又允许任意英文原值直接透传（else 分支 `query.Where("status = ?", status)`）——同一参数两种编码体系并存，前端传错一个未列出的值会得到空结果而非报错。
- **建议**: 统一只接受英文状态枚举（pending/paid/cancelled/expired），不识别中文值，避免双轨契约。

### [LOW] GetOrderStats 返回 totalAmount/paidAmount 两个同值字段
- **位置**: 1502-1509  |  **类别**: duplication  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `"totalAmount": summary.PaidAmount, "paidAmount": summary.PaidAmount` 两个键同值，命名又容易让人误以为 totalAmount 是订单总额，前端契约语义不清。
- **建议**: 只保留一个明确命名字段（如 paid_amount），前端相应调整。

### [LOW] 订单关键词过滤 SQL 在合并列表与非合并列表两处重复
- **位置**: 1079-1087, 270-278  |  **类别**: duplication  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: GetAdminOrders 非合并分支（L1079-1087）与 buildAdminMergedOrderWhere（L273-276）维护两份几乎相同的 `order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?` 拼装逻辑，后续改关键词规则需同步两处。
- **建议**: 抽一个 buildOrderKeywordWhere(db, keyword) 返回 where 片段与参数，两处复用。

### [LOW] CreateCustomOrder 用 time.Now() 本地时间设订单过期，时区口径不一致
- **位置**: 2156  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `expireTime := time.Now().Add(30 * time.Minute)`（L2156）用服务器本地时间，而全站订单/订阅统一走 `utils.GetBeijingTime()`（如 GetOrders 的日期过滤、GetOrderStatusByNo 的过期判断 L1562），非 UTC+8 服务器上 30 分钟过期判定会有偏差。
- **建议**: 改为 `utils.GetBeijingTime().Add(30 * time.Minute)`。

## internal/api/handlers/package.go

### [MEDIUM] CreatePackage 未校验负价格/负时长，与 UpdatePackage 的校验不一致
- **位置**: 72-107  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `binding:"required"` 对数值只拒绝零值，`price: -5`、`duration_days: -3` 都能通过创建；而 UpdatePackage 有显式 `*req.Price < 0` / `*req.DurationDays < 1` 校验（line 169-189）——同一实体创建与更新校验不对称。
- **建议**: 抽取共享的 validatePackage 函数（price ≥ 0、duration_days ≥ 1、device_limit ≥ 0），Create/Update 都调用。

### [MEDIUM] GetAdminPackages 的 size 无上限、page 无上限
- **位置**: 253-270  |  **类别**: performance  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: 手动 Sscanf 解析后仅 `if size < 1 { size = 20 }`，没有像 GetAdminRechargeRecords（≤100）或 utils.ParsePagination（≤100、page ≤ 10000）那样的上限——`?size=1000000` 可一次拉取百万行，`?page=999999` 产生超大 OFFSET。
- **建议**: 直接复用 utils.ParsePagination(c)（自带 page/size 上限），或补齐 `size > 100 → 100`、`page > 10000 → 10000`。

### [LOW] 套餐→响应映射在三个函数里重复三份，且 GetPackage 返回原始模型
- **位置**: 38-50  |  **类别**: duplication  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: 字段映射块在 GetPackages（line 38-50）、UpdatePackage 响应（line 212-224）、GetAdminPackages（line 301-313）逐字重复三遍；GetPackage（line 59-70）又直接返回原始 models.Package——同一实体的列表/详情/更新响应形状不一致（时间格式、description 为 null 等），前端 Packages.vue 只能多形态兼容（line 1034-1044）。
- **建议**: 抽取 `formatPackage(pkg models.Package) gin.H` 供三处复用，并让 GetPackage 也走该函数，统一契约。

### [LOW] DeletePackage 硬删除可能被订单/订阅引用的套餐，产生孤儿引用
- **位置**: 229-251  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: models.Package 关联 Orders 与 Subscriptions（models/package.go:21-22），DeletePackage 直接 `db.Delete(&pkg)` 硬删——已有订单的历史套餐记录会丢失名称/价格，历史订单展示出现空引用；同时删除有活跃订阅的套餐无任何拦截。
- **建议**: 改为软删除（is_active=false）或删除前检查 `SELECT COUNT(*) FROM orders WHERE package_id = ?`，有引用则拒绝并提示。

### [LOW] 缓存清理错误用 log.Printf 而项目其余处用 utils.LogError，风格不一致
- **位置**: 110-113  |  **类别**: style  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: Create/Update/DeletePackage 三处 `log.Printf("failed to clear packages cache: %v", err)`（line 112、208、246）绕过统一日志体系（无 request_id/级别），与 handlers 其余文件的 utils.LogError/LogWarn 不一致。
- **建议**: 统一改用 utils.LogError("ClearPackagesCache", err, nil)，并顺带把缓存清空调用抽成公共 helper 消除三处重复。

## internal/api/handlers/payment.go

### [MEDIUM] CreatePayment 忽略 GetCurrentUser 的 ok 返回值
- **位置**: 142  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `u, _ := middleware.GetCurrentUser(c)`（L142）丢弃 ok；若该路由将来脱离 AuthMiddleware（或中间件调整），`u.ID` 会对 nil 指针解引用直接 panic。同文件 GetPaymentStatus（L1124）就正确地做了 ok 检查，风格不一致。
- **建议**: 改为 `user, ok := middleware.GetCurrentUser(c); if !ok { ...401... }`，与其余 handler 保持一致。

### [MEDIUM] updatePaymentTransactionTx 只把最新的 pending 交易置为成功，旧交易永久滞留 pending
- **位置**: 348-372  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 用户多次点支付会为同一订单创建多条 pending PaymentTransaction；回调入账时 `Order("created_at DESC").First()` 只更新最新一条，其余 pending 记录永远不会被标记（失败或成功），状态悬空影响对账与 GetPaymentStatus 查询。
- **建议**: 按 order_id（或 user+amount）一次性 `Update status = success` 该订单全部 pending 交易；或在成功一条后把其余标记为 superseded。

### [MEDIUM] PaymentNotify 对回调参数做多轮全量 INFO 日志，未认证端点可被日志洪水攻击
- **位置**: 530-623  |  **类别**: performance  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 同一个回调被记录 4+ 遍：L531-533（URL+remote_addr）、L604-610（逐参数 `参数[k]=v` 打全量值，仅 sign/rsa_sign 打 ***）、L612-613、L623。该端点无认证，攻击者可伪造海量回调把敏感回调参数（订单号、金额、外部交易号、passback 透传内容）刷进日志，既泄露信息又撑爆磁盘/日志系统。
- **建议**: 收敛为单条结构化日志（仅 order_no、payment_type、金额、结果），彻底去掉逐参数循环打印；对回调端点加来源/频率限制。

### [LOW] CreatePayment 零金额免费履约后不发送支付成功通知
- **位置**: 170-184  |  **类别**: error-handling  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `amount <= 0.01` 分支直接 FinalizePaidOrder("优惠抵扣") 并返回，未调用 sendPaymentNotifications，用户享受了优惠全抵但收不到任何支付成功/订阅通知（对比正常支付回调路径 L920 会发）。
- **建议**: 零金额履约成功后同样调用 `sendPaymentNotifications(db, order.OrderNo)`，保持通知一致性。

### [LOW] parseCallbackAmount 用 fmt.Sscanf 解析金额，可接受畸形前缀
- **位置**: 238-276  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `fmt.Sscanf(amountStr, "%f", &callbackAmount)` 对 "12.5abc"、"12..5" 之类字符串会解析出前缀 12.5 而不报错（忽略 err），畸形回调金额可能被静默采纳；金额校验由此依赖 amountMatches 的 ±0.01 容差兜底。
- **建议**: 用 strconv.ParseFloat 严格解析并检查 err，解析失败返回 (0,false) 走金额无法校验分支。

### [LOW] yipay/codepay 互查回退可能命中错误的商户配置导致验签失败
- **位置**: 625-647  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: FindEnabledPaymentConfig(paymentType) 失败后，代码把 yipay↔codepay 互相回退（L630-639），若库里同时存在 yipay 与 codepay 配置且 notify 路径配错，会拿另一家商户密钥验签，正常回调被误拒（属可用性问题而非漏洞，因为验签本身会失败）。
- **建议**: 回退仅用于历史兼容场景，并增加告警日志提示管理员修正 notify_url 配置；长期应删除该回退。

### [LOW] finalizeStatusQueriedPayment 用陈旧订单对象判断是否发通知
- **位置**: 730-733  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `if err == nil && order.Status == "pending"`（L730）里的 order 是调用 FinalizePaidOrder 之前加载的快照，若订单此前已被其他回调置为 paid，这里仍会重复 sendPaymentNotifications（重复邮件/站内信）。
- **建议**: FinalizePaidOrder 返回更新后的订单对象，以返回值状态判断；或先重查订单再决定是否发通知。

### [LOW] GetPaymentMethods 缓存写入为无界 goroutine 且并发冷启动重复构建
- **位置**: 136  |  **类别**: performance  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `go cacheService.SetPaymentMethodsCache(cacheData)` 每个冷请求都派一个 goroutine 写缓存（L136），并发冷启动会重复查询 DB 并重复写；写缓存失败无任何处理。
- **建议**: 用 singleflight 合并并发冷启动的 DB 查询，写缓存改同步（该操作很快）或限制并发。

## internal/api/handlers/promotion.go

### [HIGH] 活动重复参与检查在事务外且无唯一约束：并发双请求可重复领取奖励（资金类竞态）
- **位置**: 127-131, 134-215  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: "是否已参与"检查（127-131 行）在事务外执行，而 PromotionParticipation 上 (promotion_id, user_id) 的 idx_promotion_participation_unique（models/promotion_participation.go:11-12）只是普通复合索引不是 UNIQUE；两个并发请求同时通过检查，事务内各自更新余额并创建参与记录，余额被送两次。
- **建议**: 把参与检查移进事务内（配合 SELECT ... FOR UPDATE 锁用户行，参照 checkin.go:34 的做法），并给 (promotion_id, user_id) 加真正 UNIQUE 约束，Create 撞唯一键时返回"已参与"。

### [MEDIUM] balance 分支的余额日志写入全局 DB，脱离当前事务，回滚后日志残留
- **位置**: 161-175  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: utils.CreateBalanceLog（utils/logs.go:131-137）内部取 database.GetDB() 全局连接而非本事务 tx：余额更新在 tx 内、余额日志在 tx 外提交；若后续 tx.Create(&participation)（210 行）失败回滚，余额回退但日志已落库，账实不符。
- **建议**: 改用 utils.CreateBalanceLogWithDB(tx, ...)（logs.go:139 已存在）把日志纳入同一事务。

### [MEDIUM] free_days 用 First 任意取一条订阅赠送天数，多订阅用户可能加错对象
- **位置**: 179-198  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: tx.Where("user_id = ?", user.ID).First(&subscription)（182 行）无排序条件，用户拥有多条订阅时随机/按主键取一条延长，其他订阅不变；赠送逻辑也未校验该订阅是否已过期/已停用。
- **建议**: 要求前端传 subscription_id 并校验归属，或对用户所有订阅统一赠送（事务内逐条处理）。

### [MEDIUM] MinAmount/MaxDiscount/PackageIDs 在参与逻辑中完全未使用
- **位置**: 94-215  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: Promotion 模型有 MinAmount、MaxDiscount、PackageIDs（models/promotion.go:14-16），但 ParticipatePromotion 对 balance/free_days 直接按 DiscountValue 发放，不校验订单门槛/适用套餐/封顶，这些配置字段形同虚设，管理员配置了也无效。
- **建议**: 明确各 reward 类型的适用条件：balance/free_days 至少校验活动窗口与 PackageIDs 适用性；percentage/fixed 在订单核销时校验 MinAmount/MaxDiscount。

### [LOW] GetMyPromotionParticipations 无分页
- **位置**: 241-259  |  **类别**: performance  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 参与记录全量返回（250-253 行无 Limit/Offset），老用户参与活动多时响应体无界增长。
- **建议**: 复用 utils.ParsePagination 加分页并返回 total/page。

### [LOW] 500 响应直接透传内部错误字符串给客户端
- **位置**: 217-219  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: if err != nil { utils.ErrorResponse(c, http.StatusInternalServerError, err.Error(), nil) } 把 fmt.Errorf 包装的内部错误（含 SQL 错误原文，如 149 行 "赠送余额失败: %v"）原样返回给用户，泄露实现细节。
- **建议**: 对外返回固定文案，err 只进日志（utils.LogError）。

### [LOW] participation.RewardType 先赋 promotion.DiscountType 又逐个 case 覆盖，初始值实为死赋值
- **位置**: 138-142, 177, 197, 204  |  **类别**: style  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 构造 participation 时 RewardType: promotion.DiscountType（139 行），随后 balance/free_days/discount 各 case 又各自覆盖（177、197、204 行），默认分支直接 return error——初始赋值永远不生效，阅读时容易误以为 balance 的初始值有意义。
- **建议**: 构造时省略 RewardType，在 case 内按需赋值，default 分支返回错误。

## internal/api/handlers/recharge.go

### [MEDIUM] 同一充值资源三套响应契约：裸数组+响应头 / 对象包裹 / 单对象
- **位置**: 254-257  |  **类别**: architecture  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: GetRechargeRecords 返回裸数组并只在 X-Total-Count/X-Page/X-Page-Size 响应头带分页（line 254-257）；GetAdminRechargeRecords 返回 `{recharges,total,page,size}` 包体（line 448-453）；GetRechargeRecord 返回单对象（line 275）。前端 Orders.vue 只能 `Array.isArray(data)` 特判（line 509），契约漂移风险高。
- **建议**: 统一为 `{items, total, page, size}` 信封（或全部走响应头+裸数组），删除两套实现中的一套，前端去掉特判。

### [MEDIUM] GetAdminRechargeRecords 关键词条件重复出现两次 order_no LIKE，且未做 LIKE 转义
- **位置**: 384-397  |  **类别**: duplication  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `recharge_records.order_no LIKE ? OR recharge_records.order_no LIKE ?`（一条普通、一条 `%RCH%...` 变体）是冗余条件；且此处只调了 SanitizeSearchKeyword 未调 EscapeLikePattern（其他日志接口都转义）——关键词含 `%`/`_` 时变成通配符（`%` 匹配全表），与全站搜索行为不一致。
- **建议**: 删掉重复条件（保留一条 LIKE）；统一补 `utils.EscapeLikePattern(...)` 并使用相同条件字符串，countQuery/findQuery 共用同一段 where 构造。

### [LOW] GetRechargeStatusByNo 刷新后重查记录忽略错误，可能返回过期状态
- **位置**: 293-301  |  **类别**: error-handling  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `performPaymentStatusQuery` 成功后会 `db.Where("order_no = ?", orderNo).First(&record)` 重查，但该查询的 Error 被忽略（line 295）——若重查失败，返回的还是刷新前的 pending 记录；同时该重查未限定 user_id（依赖 order_no 全局唯一）。
- **建议**: 检查重查 Error；若失败则返回 500 或提示"状态刷新失败，请稍后重试"，避免向用户展示可能过期的状态。

### [LOW] CreateRecharge 未用事务：建单、建支付交易、生成支付链接、保存 URL 多步非原子
- **位置**: 92-192  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: RechargeRecord 创建（line 109）→ PaymentTransaction 创建（line 141）→ generatePaymentURL（line 156）→ `db.Save(&recharge)` 保存 URL（line 172）是四条独立语句；中间任何一步崩溃会留下 pending 记录无交易单，且三处"标记 failed"的 `Update("status", "failed")`（line 121/145/165/176）都未检查 Error。line 172 保存 URL 失败仅打日志继续返回成功，用户拿到无支付链接的 pending 单。
- **建议**: 整体包进 db.Transaction；支付 URL 保存失败应视为创建失败并回滚或明确返回错误；所有 Update 检查 .Error。

### [LOW] 管理员列表 formatRechargeRecord 输出 payment_qr_code/payment_url 等支付凭证字段
- **位置**: 40-73  |  **类别**: security  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: formatRechargeRecord 无条件输出 payment_qr_code、payment_url、payment_transaction_id（line 57-59），GetAdminRechargeRecords 批量列表也带这些字段——批量响应放大支付凭证暴露面；另 CreateRecharge 全程未写审计日志（与订单创建不一致）。
- **建议**: 管理列表用精简 DTO（不包含支付凭证，详情接口再给）；CreateRecharge/CancelRecharge 补 CreateAuditLogSimple。

### [LOW] paymentConfig.Status == 1 分支外的 else 分支实际不可达
- **位置**: 127-184  |  **类别**: style  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: FindEnabledPaymentConfig（common.go:437-476）内部已经按 `status = 1` 过滤，返回的配置 Status 恒为 1，因此 line 180-184 的 `else { 标记 failed; "支付配置未启用" }` 是死分支；`Status == 1` 魔法数字也缺少命名常量。
- **建议**: 删除不可达的 else 分支；引入 `models.PaymentConfigStatusEnabled` 常量替代字面量 1。

## internal/api/handlers/repo_sync.go

### [MEDIUM] ServeRepoSyncFile 用 os.Stat 跟随符号链接，文件分支存在符号链接逃逸
- **位置**: 69-93  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 目录列表分支显式跳过符号链接（123-125 行），说明作者已知悉该风险；但文件服务分支 os.Stat(dirPath)（81 行）会跟随 symlink，若同步目录内存在指向目录外的符号链接，c.File 将把任意文件（如 /etc/passwd、cboard.db）公开出去。当前 repo_sync 下载的是普通文件，但手工放置或未来功能可能引入 symlink。
- **建议**: 改用 os.Lstat 拒绝符号链接（IsDir 分支也一并处理），或对最终路径再做一次 filepath.EvalSymlinks + JoinWithinBaseDir 校验。

### [LOW] 手写 HTML 目录页 + 自制 htmlEscape/escapeURLPath，建议用 html/template
- **位置**: 96-211  |  **类别**: duplication  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 目录列表页用 strings.Builder 拼 HTML（153-190 行）并自实现 htmlEscape（208-211 行）；转义正确性依赖人工维护（当前实现是对的），且易与 Go 标准库 html/template 的上下文感知转义（URL 属性、JS 上下文）产生差距。
- **建议**: 改用 html/template 渲染该列表页（模板内置转义），删除手写 htmlEscape；URL 拼接用 url.URL 结构体。

### [LOW] TestRepoSyncConnection 忽略绑定错误；目录页无缓存头
- **位置**: 29-49, 52-65  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: c.ShouldBindJSON(&req) 错误被 _ = 丢弃（36 行），非法 JSON 时全部用空值走测试，错误信息不明确；另外公开目录页/文件无 Cache-Control 策略（GetClientSubscribeXBoardCompat 明确禁缓存，此处未设），大文件每次都被客户端重新拉取。
- **建议**: 绑定失败返回 400；对同步文件按内容大小/类型设置合理的缓存头（如 ETag 或 max-age）。

## internal/api/handlers/statistics.go

### [MEDIUM] GetRevenueChart 吞掉 DB 错误并以 200 返回空数组，还会把空结果写入缓存
- **位置**: 271-294  |  **类别**: error-handling  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `if err == nil { defer rows.Close(); ... }` — 查询失败时静默返回空 labels/data，且 line 294 `go cacheService.SetStatisticsCache(cacheKey, result, 5*time.Minute)` 会把空结果缓存 5 分钟（缓存毒化）；循环后也缺少 `rows.Err()` 检查。
- **建议**: 查询出错时返回 500 且不写缓存；循环结束后检查 `rows.Err()`；缓存写入放在成功路径之后。

### [MEDIUM] GetRevenueChart 的 days 参数未校验，可为 0/负数/超大值
- **位置**: 226-230  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `fmt.Sscanf(daysParam, "%d", &days)` 可解析出 0 或负数（`AddDate(0,0,-days)` 使起始时间在未来，返回空数据）或超大值（如 999999，导致整表范围扫描）；"30abc" 这类输入还会被部分解析为 30。
- **建议**: 解析后 clamp 到 1..365，非法输入返回 400；用 strconv.Atoi 并检查整串消费。

### [MEDIUM] GetRegionStats 登录次数跨表重复计数，UserCount 与 LoginCount 口径不一致
- **位置**: 394-450  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: processEntry 对 audit_logs 与 user_activities 两条来源都执行 `stat.LoginCount++`，同一用户同一会话可能在两表各计一次；而 `userRegionMap` 按首见即止（line 436-439），造成某区域 LoginCount 远大于 UserCount，percentage 基于不重复用户却展示不匹配的登录次数。
- **建议**: 明确数据源口径：优先只统计一张表，或按 user_id+24h 时间桶去重后再计数；注释说明 UserCount/LoginCount 各自定义。

### [MEDIUM] GetRegionStats 全表装载 audit_logs + user_activities 并在内存做聚合与逐行 GeoIP 解析
- **位置**: 339-466  |  **类别**: performance  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `db.Select("DISTINCT user_id, location, ip_address, created_at")...Find(&auditLogs)` 无 LIMIT/时间窗，把整张 audit_logs 表拉进内存；对无 location 的记录逐行调用 `geoip.GetLocationWithCache(ipStr)`（line 398-402，首次未命中时是真实解析），数据量大时请求会非常慢。
- **建议**: 在 SQL 内按国家/城市聚合（对 JSON location 用 CASE 或拆列存 country/city 字段），或限制最近 N 天窗口；GeoIP 解析改为异步批量任务而非请求内同步执行。

### [LOW] GetStatistics 响应同时输出 snake_case 顶层字段与 camelCase 的 overview 对象，同一数据两套命名
- **位置**: 199-218  |  **类别**: architecture  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: payload 同时含 `total_users` 等顶层键和 `overview.totalUsers/activeSubscriptions/totalRevenue` 副本（数字完全重复）；前端 admin/Statistics.vue 只能写 `data.total_users || data.totalUsers` 兼容（line 366-369），任何字段漂移都会静默出错。
- **建议**: 确定单一规范（建议 snake_case 或沿用 overview 对象），删除冗余顶层键，前端去掉 || 兼容分支，避免双份维护。

### [LOW] GetStatistics 所有 db.Raw().Scan() 均未检查错误，且异步缓存写入为 fire-and-forget goroutine
- **位置**: 50-83  |  **类别**: error-handling  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: 用户聚合、订阅聚合两条 Raw 查询的 `.Error` 被丢弃（line 50-58、76-83），DB 故障时返回全零统计的 200；`go cacheService.SetStatisticsCache(...)`（line 221、294）不回收错误，Redis 异常只在后台日志出现。
- **建议**: 为每条 Raw 查询补 `.Error` 检查并 500；缓存写入 goroutine 内做错误日志与 recover（防 panic 拖垮进程）。

### [LOW] 订阅统计桶重叠：已过期且未激活的订阅同时计入 expired 与 inactive
- **位置**: 76-85  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `expired` 条件 `expire_time <= now` 未限定 `is_active = true`，与 `inactive`（is_active=false）桶重叠，前端展示的活跃/过期/未激活百分比加起来不等于 100%，口径混乱。
- **建议**: expired 桶加上 `is_active = true` 前提，或在 SQL 中用互斥 CASE 保证各桶不相交，并注释统计口径。

## internal/api/handlers/subscription.go

### [CRITICAL] ConvertSubscriptionToBalance 存在并发双花：行锁失效 + 事务内未复核订阅状态
- **位置**: 1147-1163  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 1) `tx.Set("gorm:query_option", "FOR UPDATE")` 是 GORM v1 语法，项目用的 gorm.io/gorm v1.25.5（v2）已不再处理该键，行锁实际不生效（同仓 payment.go:350/385 用的是正确的 `clause.Locking{Strength:"UPDATE"}`）。2) 即使锁生效也防不住：`sub` 在事务外加载（L1131），两个并发请求都通过 L1137 的过期校验并计算出金额，随后串行进入事务，第二次仍会 `Update("balance", newBalance)` 再次入账，而 `tx.Delete(&sub)` 对已删除的行返回 nil 错误（无 RowsAffected 检查），用户可重复兑换拿到双倍余额。
- **建议**: 改用 `tx.Clauses(clause.Locking{Strength:"UPDATE"})` 在事务内重新锁定并读取订阅行，事务内复核过期状态与存在性，检查 `tx.Delete(&sub).RowsAffected == 0` 时返回错误回滚；同时用 `gorm.Expr("balance + ?", convertedAmount)` 原子累加余额。

### [HIGH] performSubscriptionReset 非事务执行：设备删除失败时 URL 已轮换但接口报失败
- **位置**: 209-280  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `db.Save(sub)`（改 URL）→ `db.Create(&reset)` → 最后 `return db.Where(...).Delete(&models.Device{}).Error` 三段操作没有包在事务里。若设备删除失败，调用方（ResetSubscription/BatchResetSubscriptions）会向管理员报"重置失败"，但订阅 URL 实际已更换、旧 token 缓存已清，用户拿着旧地址会失联，日志与真实状态不一致。
- **建议**: 把 Save(sub) + Create(reset) + Delete(devices) 放进 `db.Transaction`，任一步失败整体回滚；Delete 返回前检查 RowsAffected 并返回有意义的错误。

### [HIGH] GetUniversalSubscription 缺少过期/停用校验，与 GetSubscriptionConfig 行为不一致
- **位置**: 1943-2023  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: GetSubscriptionConfig（L1700-1719）会先校验 `isExpired`/`isInactive` 并返回错误配置；而 GetUniversalSubscription（L1943 起）只查 `subscription_url` 是否存在就直接生成配置、记录设备，expired/inactive 的订阅仍可继续拉取 universal 配置，绕过管理员停用/到期限制。
- **建议**: 在 GetUniversalSubscription 中复用与 GetSubscriptionConfig 相同的过期/激活校验逻辑（抽公共函数），不通过时返回 generateErrorConfigBase64 错误配置。

### [HIGH] 客户端 IP 取自可伪造请求头，设备级访问控制可被绕过
- **位置**: 1721, 1960, 894  |  **类别**: security  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 订阅拉取与重置均用 `utils.GetRealClientIP(c)` 作为设备标识/审计依据，而该函数（internal/utils/network.go:180-215）直接信任 `CF-Connecting-IP`/`True-Client-IP`/`X-Forwarded-For`/`X-Real-IP` 请求头（只要不是内网 IP 即采用）。攻击者直连源站时可任意伪造 IP，配合 UA 轮换即可伪装成不同设备，绕过 FindExistingDevice/设备上限逻辑（L1641、L1726、L1971）并污染审计日志。
- **建议**: 仅在可信反向代理（Cloudflare/Nginx）后面才接受这些头，且校验来源 socket 对端是否在信任的代理网段；直连时只使用 c.ClientIP()。

### [MEDIUM] ExtendSubscription 允许负数天数且忽略 Save 错误
- **位置**: 919-940  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `Days int json:"days" binding:"required"` 只要求非零，负数（如 -30）同样通过校验并执行 `sub.ExpireTime.AddDate(0, 0, req.Days)` 把订阅时间缩短；`db.Save(sub)` 的错误被直接忽略（L940），失败时接口仍返回"订阅已延长"。
- **建议**: 加 `binding:"required,gt=0"` 并在服务端二次校验 `req.Days <= 0` 直接 400；检查 db.Save 返回的错误并处理。

### [MEDIUM] UpdateSubscription 按 DateFormat 解析到期时间为 UTC 午夜，且解析失败被静默忽略
- **位置**: 803-809  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `time.Parse(DateFormat, *req.ExpireTime)` 无时区时按 UTC 解析，得到的是北京时间 08:00 的到期时间，与全站 GetBeijingTime 口径偏差 8 小时（管理员设置 2026-01-01，实际 01-01 08:00 即过期）；两种格式都解析失败时既不报错也不提示，管理员以为改成功了。此外 `Status` 字段任意字符串直接入库，无白名单。
- **建议**: 改用 `time.ParseInLocation(DateFormat, v, utils.BeijingTZ)`（与 order.go:904 一致）；解析失败返回 400 明确报错；对 status 做 allowed 值校验。

### [MEDIUM] StopConfigUpdate 是空实现，停止按钮实际不生效
- **位置**: 2244-2248  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 函数体仅记录审计日志并返回"停止指令已发送"，注释自述"假设有一个 Stop 方法或类似的机制，原代码只有响应"——没有任何调用服务端 Stop/RunUpdateTask 的取消逻辑，管理员点击停止后任务继续运行，属于遗留的半成品死分支。
- **建议**: 给 config_update 服务补真正的停止机制（context cancel / atomic stop flag），或在路由层移除该入口，避免假操作误导管理员。

### [MEDIUM] StartConfigUpdate 与 TestConfigUpdate 完全重复且无并发互斥
- **位置**: 2233-2258  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 两个 handler 体完全一样（都 `go service.RunUpdateTask()`），连续点击可并发启动多个 RunUpdateTask 协程，且与定时任务可能重叠执行，造成重复拉取、节点重复写入；GetStatus 的 is_running 未被用作互斥。
- **建议**: 合并为单一入口，在 RunUpdateTask 内用原子标志/单飞锁保证同一时刻只有一个更新任务，Test 与 Start 复用同一实现。

### [MEDIUM] ExportSubscriptions / GetExpiringSubscriptions 全量加载无上限
- **位置**: 1215-1235, 1548-1608  |  **类别**: performance  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: ExportSubscriptions 对全表 `Preload("User").Find(&subs)` 并在内存拼 CSV，无分页/上限；GetExpiringSubscriptions 同样把窗口内所有即将到期订阅全量查出，站点规模大时内存与响应时间不可控。
- **建议**: 导出走游标/分批查询（如按 ID 分段 1000 条一批）并限制单次导出总量；到期列表加 Limit 分页参数。

### [MEDIUM] 每次订阅拉取无节制地派生 3 个 goroutine，无池化与背压
- **位置**: 1743-1790, 223-235, 871-883  |  **类别**: performance  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: GetSubscriptionConfig 每次请求并发 3 个 goroutine（RecordDeviceAccess、clash_count+1、CreateSubscriptionLog），ResetSubscription/UpdateSubscription/Batch* 等也各自 go 清缓存/发邮件/记日志；高峰拉取下 goroutine 无上限堆积，且多数只用 5s timeout 包裹 select 打日志，真正耗时的 DB 调用并没有传入 ctx（见 asyncSubscriptionLog L126-142 注释与实现不符）。
- **建议**: 改用带容量上限的工作池/worker 队列（如 errgroup 或 semaphore），或至少让 CreateSubscriptionLog 真正接收 ctx 以便超时中断 DB 操作。

### [MEDIUM] ExportSubscriptions 存在 CSV 公式注入且未做字段转义
- **位置**: 1215-1235  |  **类别**: security  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 用户名/邮箱/订阅地址等用户可控字段直接 `fmt.Sprintf("%d,%d,%s,%s,...")` 拼入 CSV 且不加引号：以 `=`、`+`、`-`、`@` 开头的用户名（如 `=HYPERLINK("http://evil")`）在 Excel 打开时会执行公式；含逗号/引号/换行的字段还会破坏 CSV 结构。
- **建议**: 用 encoding/csv.Writer 生成并对以 =+-@ 开头的单元格加单引号前缀（或按 OWASP 指引转义），对用户名/邮箱/订阅 URL 统一处理。

### [LOW] formatDeviceList 输出大量重复字段且与别处格式化逻辑重复
- **位置**: 148-189  |  **类别**: duplication  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 同一字段成对输出（device_name/name、device_type/type、ip_address/ip、last_access/last_seen），与 buildSubscriptionListData、user.go 中订阅格式化逻辑各自维护一份 gin.H 映射，改字段名需同步多处；formatIP/getString 也散落在本文件。
- **建议**: 抽统一的 DeviceDTO/SubscriptionDTO 结构体（含 json tag），各 handler 只负责组装一次，避免前端兼容字段靠复制粘贴维护。

### [LOW] ResetUserSubscription 吞掉逐订阅重置错误且不发通知邮件
- **位置**: 982-1006  |  **类别**: error-handling  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 循环里 `_ = performSubscriptionReset(db, &subCopy, ...)` 静默吞掉错误（与 BatchResetSubscriptions 计数失败不同），且整个用户重置流程不发送任何重置邮件/站内通知（单条重置 ResetSubscription L915 会发），行为不一致。
- **建议**: 统计成功/失败数并在响应与审计中体现；重置后按与单条重置一致的通知逻辑给用户发邮件。

### [LOW] GetUserSubscriptionDevices 用服务器本地时间而非北京时间
- **位置**: 680  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `oneDayAgo := time.Now().Add(-24 * time.Hour)`（L680）用服务器本地时区，而库中 LastAccess 与其他统计均按北京时间口径；服务器非 UTC+8 时"24 小时在线"统计窗口偏移。
- **建议**: 改用 `utils.GetBeijingTime().Add(-24 * time.Hour)`，与全站时区口径一致。

### [LOW] UpdateSubscriptionConfig 每次调用都轮换订阅 URL 且未清旧 token 缓存
- **位置**: 1848-1898  |  **类别**: logic  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 用户每次"更新订阅配置"都会 `subscription.SubscriptionURL = utils.GenerateSubscriptionURL()` 生成新 token，重复调用会使旧地址立刻失效（已配置好的客户端断连），且没有清旧 URL 的 config 缓存（对比 ResetSubscription L222-235 有清）。
- **建议**: 若非必要不要每次调用都轮换 token（可仅返回现有 URL）；如确需轮换，先清旧 URL 缓存再保存，并提示用户旧地址将失效。

### [LOW] 管理端 GetSubscriptionDevices 全量拉取设备无分页
- **位置**: 704-716  |  **类别**: performance  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: `database.GetDB().Where("subscription_id = ?", sub.ID).Find(&devices)` 无分页无 Limit，再全量 formatDeviceList，与用户端 GetUserSubscriptionDevices 的分页实现不一致；大设备量下响应慢且浪费内存。
- **建议**: 复用 GetUserSubscriptionDevices 的分页+统计模式，或至少加 Limit 上限。

### [LOW] buildSubscriptionListData 在已 Preload User 的情况下仍二次批量查用户
- **位置**: 524-531  |  **类别**: performance  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: GetAdminSubscriptions 已 `Preload("User")`，但 buildSubscriptionListData 又执行 `db.Where("id IN ?", userIDs).Find(&users)` 建 userMap，仅为了兜底已删除用户场景，正常情况属于冗余查询。
- **建议**: 仅当存在 `sub.User.ID == 0` 的订阅时才发起 userMap 查询；否则直接使用预加载数据。

## internal/api/handlers/subscription_access.go

### [MEDIUM] shouldBlockBrowserSubscriptionAccess 在最高频公开端点每次请求查库
- **位置**: 15-28  |  **类别**: performance  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: 订阅拉取（GetSubscriptionConfig/GetUniversalSubscription 均调用）是最热门的公开端点，每次拉取都执行一次 `Where("key = ? AND category = ?", ...)` 读 SystemConfig；该配置几乎不变，纯属浪费。
- **建议**: 把该开关缓存到 Redis/进程内存（TTL 30-60s），或复用现有 config 缓存服务，仅配置变更时失效。

### [LOW] 客户端标识列表存在重复项
- **位置**: 41-45  |  **类别**: duplication  |  **来源组**: A4-handlers-sub-order-pay (subscription/order/payment)
- **问题**: clientMarkers 中 `"sing-box"` 出现两次（L42 和 L43），纯冗余；且 `"singbox"`、`"hiddify"` 等新客户端标记需要人工维护。
- **建议**: 去重并集中维护客户端 UA 关键词表（可放配置），避免列表膨胀后误判。

## internal/api/handlers/ticket.go

### [HIGH] GetTicket 向工单所属普通用户泄露 admin_notes / rating / rating_comment
- **位置**: 450-464  |  **类别**: security  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `/tickets/:id` 路由仅挂 AuthMiddleware（router.go:236），普通用户可访问；GetTicket 对任何请求者（含非管理员）无条件输出 `responseData["admin_notes"]`、`rating`、`rating_comment`、`resolved_at`、`closed_at`（line 450-464）。管理员内部备注（AdminNotes 字段注释为内部备注）被直接暴露给用户，属敏感信息泄露/越权读取。
- **建议**: 仅当 `isAdmin` 为 true 时才注入 admin_notes/rating/rating_comment 字段；用户端响应应显式构造白名单字段。

### [MEDIUM] ReplyTicket 中 db.Save(&ticket) 的状态迁移错误被忽略
- **位置**: 573-575  |  **类别**: error-handling  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `if shouldSaveTicket { db.Save(&ticket) }` 不检查 Error——回复创建成功但状态迁移失败时静默返回成功，工单停留在旧状态，用户和管理员看到的状态不一致。
- **建议**: 检查 Save 错误并返回 500；更优做法是用事务包裹"创建回复 + 更新工单状态"。

### [MEDIUM] 用户回复会把已关闭/已解决的工单自动重新打开，且可回复已取消工单
- **位置**: 562-575  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `if !isAdmin && (ticket.Status == "pending" || ticket.Status == "resolved" || ticket.Status == "closed")` 会把状态改回 processing 并清空 ResolvedAt/ClosedAt（line 567-571）——用户刚 CloseTicket 关闭的工单，回一条消息就悄悄复活；对 `cancelled` 状态无任何分支，回复后工单保持 cancelled 但多出未处理回复，状态机不一致。
- **建议**: 明确闭环策略：closed/cancelled 工单禁止用户回复（返回 400 提示需联系管理员），只有管理员可重新打开；将状态迁移逻辑收敛到单一函数并写状态机注释。

### [MEDIUM] ticketReadJoin 用 fmt.Sprintf 把 userID 拼进 SQL 子查询，未参数化
- **位置**: 105-110  |  **类别**: security  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: `fmt.Sprintf("LEFT JOIN (SELECT ... WHERE user_id = %d ...)", userID)`（line 107）把用户 ID 直接嵌入 SQL 字符串，GetUnreadTicketRepliesCount（line 214、220）与 GetTickets（line 358）都依赖它。当前 user.ID 来自已认证的 DB 记录风险有限，但这是注入形状的脆弱模式，一旦 ID 来源变化即成为 SQL 注入点。
- **建议**: 改用 GORM 支持的参数化 Joins：`Joins("LEFT JOIN (SELECT ticket_id, MAX(read_at) AS read_at FROM ticket_reads WHERE user_id = ? GROUP BY ticket_id) ticket_read_state ON ...", userID)`，并把子查询常量提取为公共函数避免三处重复。

### [LOW] asyncNotifyTicketReply 每回复至少起 1~2 个无 recover 的 goroutine
- **位置**: 140-194  |  **类别**: error-handling  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: 用户回复时 `go notification.NewNotificationService().SendAdminNotification(...)`，管理员回复时再包一层 goroutine 做站内通知+邮件+全量回复历史查询（line 154-191）；goroutine 内任何 panic 无 recover 会直接崩溃整个进程，且邮件/通知服务异常无法反馈到请求方。
- **建议**: 统一走带 recover + 错误日志的异步任务封装（或队列入库），避免在每个 HTTP 请求里裸起 goroutine；回复历史查询可改为只取最近 N 条。

### [LOW] 管理员未读计数语义与接口名不符：零回复新工单也算"未读回复"
- **位置**: 200-227  |  **类别**: logic  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: 管理员分支 `Where("ticket_read_state.read_at IS NULL OR ...")` 会把"从未读过的工单"（含 0 回复）计入 unread count，接口名是"未读回复数"但实际返回"未读工单数"；两分支的 JOIN 子查询与 GetTickets 重复三份。
- **建议**: 统一语义（按回复还是按工单）并在字段/注释说明；将 ticket_reads 子查询提取为共享常量，前端文档同步。

### [LOW] checkDBError 对非 NotRecordFound 错误统一报"获取工单失败"，消息与调用场景不符
- **位置**: 34-44  |  **类别**: style  |  **来源组**: A6-handlers-log-stats-ticket (logs/statistics/analytics/ticket/coupon/recharge/package)
- **问题**: UpdateTicketStatus、CloseTicket 也复用 checkDBError，但数据库内部错误时用户看到的是"获取工单失败"（而非"更新工单失败"），误导排障；且 UpdateTicketStatus 处理器内没有任何管理员校验，完全依赖路由中间件（目前仅注册在 admin 路由下，属隐式依赖）。
- **建议**: checkDBError 增加自定义错误文案参数；UpdateTicketStatus 开头用 getIsAdmin(c) 显式守卫（与 CloseTicket 的用户态守卫对称）。

## internal/api/handlers/user.go

### [HIGH] CreateUser 订阅创建失败仅记日志，接口仍返回"创建成功"
- **位置**: 1462-1532  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 1508-1511 行订阅创建失败时只 `utils.AppLogger.Error` 后继续，第 1589 行仍返回 201 "创建成功"。用户已创建但无订阅（核心业务对象缺失），前端展示与库内状态不一致；且用户创建与订阅创建不在同一事务，失败无法回滚。
- **建议**: 把用户+订阅+默认设置放进同一事务（参考 Register 的 WithTransaction 写法）；订阅创建失败应整体回滚并返回 500。

### [HIGH] CreateUser 把明文密码写入管理员通知和欢迎邮件
- **位置**: 1551-1586  |  **类别**: security  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 1556 行 `"password": plainPassword, // 明文密码` 进入 SendAdminNotification，第 1580 行明文密码再拼进 GetUserCreatedTemplate 邮件内容（1564-1586 行）。邮件不是安全信道（可能被转发、留存、第三方邮件服务扫描），明文密码一旦泄露即账户失守；且写密码进通知/邮件属于系统性反模式。
- **建议**: 不在通知与邮件中携带明文密码；改为首次登录强制改密（生成一次性 reset token 或直接复用 ResetPasswordByCode 流程），邮件只含一次性改密链接。

### [MEDIUM] BatchEnableUsers 与 BatchDisableUsers 约 120 行复制粘贴，仅 Update 值与文案不同
- **位置**: 2423-2540  |  **类别**: duplication  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 两个函数结构完全一致：绑 ID 数组→校验非空→(Disable 版)自我保护与管理员检查→查 targetUsers→Update is_active→清订阅缓存→写审计日志→返回计数。除 is_active 目标值和两处守卫外逐行重复，属于典型的可参数化合并。
- **建议**: 合并为 `batchSetUserActive(c, active bool)` 一个函数，路由层分别注册；顺带补上两个函数中 `Find(&targetUsers)` 的错误检查。

### [MEDIUM] CreateUser/UpdateUser 的 expire_time 解析失败被静默吞掉，前端误以为已生效
- **位置**: 1473-1496, 1733-1741  |  **类别**: error-handling  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: CreateUser 第 1475-1489 行两个 Parse 全失败时静默回退到默认月份；UpdateUser 第 1734-1740 行三个 Parse（2006-01-02T15:04:05 / 空格格式 / RFC3339）全失败时 ExpireTime 保持不变，且无任何报错。管理员填写了错误格式的到期时间，接口返回成功但实际未设置，属于典型静默失败。
- **建议**: 解析失败时返回 400 并明确提示支持的格式；把三格式解析提取为 utils.ParseFlexibleTime 统一处理并复用。

### [MEDIUM] DeleteUser/BatchDeleteUsers 删除顺序错误：先删 subscriptions 再按子查询删 devices，该语句恒为空
- **位置**: 1860-1889, 2310-2328  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 1861 行先 `Delete(&models.Subscription{})`，随后第 1873 行 `tx.Where("subscription_id IN (SELECT id FROM subscriptions WHERE user_id = ?)", user.ID).Delete(&models.Device{})` 的子查询此时已查不到任何订阅，形同死代码；真正兜底的是第 1882 行按 user_id 删除，但 Device.UserID 是可空的 `*int64`（models/device.go:9），仅按 subscription_id 关联的历史/异常设备会被残留为孤儿数据。BatchDeleteUsers（2318/2324 行）同样问题。
- **建议**: 在删除 subscriptions 之前先按 subscription_id IN (子查询) 删除 devices，再删 subscriptions，最后删按 user_id 关联的 devices（双条件覆盖），并把该串行删除序列提取为公共 helper 供单删/批删复用。

### [MEDIUM] 删除用户不彻底：CheckinRecord/UserCustomNode/LoginAttempt/VerificationCode/余额与佣金日志等全部残留
- **位置**: 1860-2045, 2310-2421  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: DeleteUser/BatchDeleteUsers 清理了订阅/设备/订单/充值/工单/通知/登录历史/邀请码等，但遗漏了 CheckinRecord、UserCustomNode（专属节点分配）、LoginAttempt、VerificationCode、VerificationAttempt、BalanceLog、CommissionLog、AuditLog、SubscriptionLog、EmailQueue 等表，用户数据残留违反隐私与一致性预期；同时第 2038 行账号删除邮件声称"数据保留30天"，实际第 2007 行立即硬删用户行，声明与行为矛盾。
- **建议**: 整理一张"用户级联删除清单"（含上述遗漏表），统一由公共删除服务执行；删除邮件文案与实际策略对齐（要么软删除+30 天定时清理，要么去掉保留声明）。

### [MEDIUM] UpdateUserStatus 无自我保护：管理员可禁用自己、取消自己的管理员身份
- **位置**: 2093-2211  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: UpdateUserStatus 第 2134-2136 行可把任意用户（含自己）IsAdmin 置 false，第 2127-2129 行可禁用自己；对比 BatchDisableUsers（2489-2497 行）明确禁止"禁用当前登录管理员"、DeleteUser（1851-1858 行）有"最后一个管理员"保护——三个入口的保护策略互相矛盾，误操作可把唯一管理员锁在门外或降为普通用户。
- **建议**: 在 UpdateUserStatus 中补充：目标为当前登录用户且执行禁用/取消管理员时拒绝；保留"至少一个活跃管理员"的不变量检查（与 DeleteUser 对齐）。

### [MEDIUM] GetUserDetails 每订阅 4 次 SystemConfig 查询（N+1），GetUserSubscription 同样
- **位置**: 543-591, 563-565  |  **类别**: performance  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 563-565 行循环内对每个订阅调用 getSubscriptionURLs 与 getMultiClientSubscriptionURLs，二者内部各自调用 utils.GetBuildBaseURL(c.Request, database.GetDB())（subscription.go:69,76），而 GetBuildBaseURL 每次都执行 2 次 SystemConfig 查询（network.go:90-93）——订阅数 N 时共 4N 次 DB 查询；admin.go 的 GetUserSubscription（710-713 行）同样每次请求 2-4 次。
- **建议**: 把 baseURL 提出循环（在 GetUserDetails 开头算一次传入 getSubscriptionURLs/getMultiClientSubscriptionURLs），或为 GetBuildBaseURL 增加短 TTL 缓存（domain_name 变更频率极低）。

### [MEDIUM] LoginAsUser 不检查目标用户 IsActive，可"登录为"已禁用账号
- **位置**: 2048-2091  |  **类别**: security  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 2052-2060 行仅检查目标用户存在且非管理员，未检查 `targetUser.IsActive`，直接签发 access/refresh token。对比 finalizeLogin 第 685 行对禁用账号返回 403，impersonation 路径绕过了禁用状态检查——被管理员禁用的用户仍可被"代登录"并使用其权限（尽管是管理员本人操作，但违背禁用语义且留下可被误解的日志）。
- **建议**: 在签发 token 前增加 `if !targetUser.IsActive { 403 }`；并在安全日志中记录执行 impersonation 的管理员与目标用户。

### [LOW] 多处 goroutine 捕获 *gin.Context 并在 handler 返回后使用
- **位置**: 1514-1531, 1537-1562, 1750-1762, 1795-1821, 2033-2042  |  **类别**: architecture  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: CreateUser（1514-1531 行日志 goroutine 内调 utils.GetRealClientIP(c) 与 middleware.GetCurrentUser(c)）、UpdateUser（1750-1762、1795-1821 行）、DeleteUser（2033-2042 行）等均在 `go func()` 内继续读 gin.Context。gin 文档明确禁止 handler 返回后在 goroutine 中使用 Context（请求结束后 Context 状态不可靠，存在数据竞争）。
- **建议**: goroutine 入口先把需要的值（IP、UA、adminUser、ID 等）取出为局部变量再闭包捕获；参考 subscription.go 的 asyncSubscriptionLog 把 ctx 显式传入并带超时的写法。

### [LOW] GetUsers/GetUserDetails 十余处 DB 调用不检查错误，出错时静默返回空数据
- **位置**: 266, 521, 533-537, 593-594, 653, 709, 723-726  |  **类别**: error-handling  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 典型如 GetUsers 第 266 行 `query.Count(&total)` 错误被忽略；GetUserDetails 中 `Preload("Package").Find(&subs)`（521 行）、设备在线数聚合（533-537 行）、订单（594 行）、充值（653 行）、签到（709 行）、重置计数（723-726 行）等全部不检查 .Error。DB 瞬时故障时接口返回"成功"但数据为空，难以排查。
- **建议**: 对关键聚合查询统一检查错误（失败返回 500 或至少记日志），或为这些只读聚合封装批量查询 helper 统一错误处理。

### [LOW] SendEmailToUser 模板替换使用 time.Now()，与全项目北京时间不一致
- **位置**: 3251-3255  |  **类别**: logic  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 3254 行 `daysLeft := int(subscription.ExpireTime.Sub(time.Now()).Hours() / 24)` 用系统时区 time.Now()，而项目统一使用 utils.GetBeijingTime()；服务部署在非东八区时 days_left 计算偏差可达一天，且已过期订阅会得到负数。
- **建议**: 统一改用 utils.GetBeijingTime()，并对负数天数钳制为 0（与 BatchSendExpireReminder 2663-2666 行行为对齐）。

### [LOW] 注释掉的 GeoIP 代码块与三份重复的 IP 格式化/指针取值函数
- **位置**: 656-667, 676-682, 728-767, 824-829, 1249-1282  |  **类别**: maintainability  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 676-682 行与 824-829 行是整段注释掉的 GeoIP 查询代码（死代码）；formatIPForRecharge（656-667 行）与 formatIPForUA（756-767 行）逐字重复，getStringPtr（728-733 行）与 getString（750-755 行）重复，文件尾部 normalizeNullableIP/normalizePointerIP/normalizeIP（1263-1282 行）与 formatIP 系列又是同一逻辑的第三份；subscription.go 里还有第四份 formatIP。
- **建议**: 删除注释块；IP 归一化（::1→127.0.0.1、去 ::ffff: 前缀）与 *string→string 收敛为 utils 层公共函数（如 utils.NormalizeIP / utils.StrPtrValue），全项目统一引用。

### [LOW] GetUserDetails 的 ua_records 按 Go map 迭代输出，顺序随机且 devices 全量加载无 Limit
- **位置**: 768-805  |  **类别**: performance  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 775-787 行把去重后的 UA 存进 map，第 789 行 `for _, d := range uaMap` 迭代输出——Go map 遍历顺序随机，同一用户两次请求返回顺序不同，前端列表会跳动；且第 770-773 行 `Find(&devices)` 无 Limit，设备量大时全量读出再内存去重。
- **建议**: 改为按 slice 保序去重（保留 LastAccess 最新者），并对查询加 Limit（如 500）；排序在 SQL 层完成。

### [LOW] GetCurrentUser/UpdateCurrentUser 同时返回 avatar 与 avatar_url 两个键，API 契约含糊
- **位置**: 139-142, 209-212  |  **类别**: style  |  **来源组**: A3-handlers-core (auth/user/admin)
- **问题**: 第 140-142 行对同一值同时输出 "avatar" 与 "avatar_url"，UpdateCurrentUser（209-212 行）同样；admin.go GetUserSubscription 又出现 expire_time/expiryDate 双键（756-757 行）。前端两个字段都读，等于把契约混乱下推给前端适配，且新增字段时极易漏同步。
- **建议**: 统一为一个规范键（如 avatar_url），前端一次性迁移；必要时在 DTO 层做别名而非在多个 handler 里手工重复拼接 gin.H。

## internal/api/handlers/xboard_compat.go

### [MEDIUM] 过期分支先调用配置服务再决定是否给错误包，base64 错误配置实际几乎不可达
- **位置**: 162-171  |  **类别**: logic  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: isExpired && !isSpecialValid 时先 GenerateClientConfig（164 行），而该服务内部（config_update.go:2359-2360）对非 normal 状态会生成"错误节点"配置并返回非空 content——因此 165-168 行几乎总是返回服务生成的错误配置，169 行的 generateErrorConfigBase64（"订阅已过期，请续费"）基本是死代码；且该错误路径用默认 text/plain Content-Type 输出 base64 串，客户端解析行为不确定，与正常路径（220 行设置 Content-Type/Content-Disposition）不一致。
- **建议**: 过期时直接返回 generateErrorConfigBase64（跳过 GenerateClientConfig），保持 Content-Type 与正常路径一致（如 text/plain; charset=utf-8 + 相同头部）；或在配置服务内统一处理过期状态，删除外层重复分支。

### [MEDIUM] 每次订阅拉取派生两个 goroutine，无 recover、无节流，高并发下 goroutine 堆积
- **位置**: 195-204  |  **类别**: performance  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: 设备记录（196-198 行）与次数自增（200-203 行）各起一个 goroutine，且记录设备时 deviceManager.RecordDeviceAccess 内部有多次 DB 读写（device_manager.go:928 起，最长链路含别名去重）；客户端刷新频繁时热门订阅可堆积大量 goroutine；次数自增 goroutine 不依赖设备记录结果，设备写入失败时计数照样 +1，统计口径漂移。
- **建议**: 用有界 worker 队列或把设备记录改为同步+短超时（参考 asyncSubscriptionLog 的带超时模式）；把计数自增并入设备记录事务内完成；goroutine 内加 recover。

### [LOW] map 取值强类型断言与 Count 错误忽略
- **位置**: 72-73, 95-96  |  **类别**: error-handling  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: urls["clash_url"].(string)（72-73 行）依赖 getMultiClientSubscriptionURLs 恒返回该键（当前成立，subscription.go:79），但无 ok 判断，未来改动键名即 panic；95-96 行在线设备 Count 错误被忽略，DB 抖动时返回 0 设备。
- **建议**: 断言带 ok 检查并给出兜底值；Count 检查错误。

### [LOW] 公开订阅接口无速率限制，token 即口令可被暴力枚举
- **位置**: 114-232  |  **类别**: security  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GET /client/subscribe 是公开路由（router.go:134），token 为订阅 URL 的随机串；接口无任何限流（对比 /auth/login 有 LoginRateLimitMiddleware），攻击者可高速枚举/撞 token，撞中即获得他人订阅配置（含全部节点地址）。
- **建议**: 对该路由挂订阅级速率限制（按 IP 或按 token 计数），并考虑失败次数封禁。

### [LOW] 兼容接口直接 c.JSON 绕过统一响应包装，与其他接口格式分裂
- **位置**: 21-50, 98-112  |  **类别**: style  |  **来源组**: A7-handlers-rest (其余 handlers)
- **问题**: GetCurrentUserXBoardCompat/GetUserSubscriptionXBoardCompat 直接 c.JSON(http.StatusOK, gin.H{...})（49、111 行），而项目其余接口统一走 utils.SuccessResponse 的 {success,message,data} 包装；虽为 XBoard 兼容有意为之，但 handlers 包内两种响应范式并存，后续维护者易混淆；GetUserSubscriptionXBoardCompat 在无订阅时返回空 {}（66 行），前端无提示。
- **建议**: 在文件中注释说明兼容契约来源；对空订阅场景返回明确错误对象（如 {subscribe_url:""}）而非空对象。

## internal/api/router/router.go

### [HIGH] 优惠券列表/详情接口完全公开，未登录即可枚举全部有效优惠码
- **位置**: 187-194  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: coupons.GET("") → GetCoupons 与 coupons.GET("/:code") → GetCoupon 挂在 api 组上且无 AuthMiddleware（仅 /coupons/my 与 /coupons/admin 有鉴权）。GetCoupons 返回所有 status=active 且在有效期内的完整 Coupon 模型（含 code/discount 字段），GetCoupon 按 code 返回任意优惠券且不过滤 status。攻击者可直接 GET /api/v1/coupons 拿到全部可用折扣码用于下单，或逐个枚举 code。
- **建议**: 列表接口改为仅管理员可见（或只返回优惠活动展示字段、绝不返回 code/discount_value）；GetCoupon 增加 status='active' 过滤并加限流；如需登录前校验折扣码，应走现有的 POST /coupons/verify 流程。

### [HIGH] SetTrustedProxies(nil) 与 GetRealClientIP 盲信 XFF 的信任模型互相矛盾，直连部署时可伪造 IP 绕过限流
- **位置**: 15-17  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 路由设置 r.SetTrustedProxies(nil) 使 gin 的 c.ClientIP() 不再信任代理头，但 internal/middleware/ratelimit.go:210/243 与 internal/utils/network.go:180-215 的 GetRealClientIP 无条件信任 CF-Connecting-IP / True-Client-IP / X-Forwarded-For（仅过滤内网段）。而 bt-deploy.sh 生成 HOST=0.0.0.0、docker-compose.yml 暴露 8000 端口，服务可直接被访问：任何人可伪造 X-Forwarded-For 绕过登录锁定、验证码限流与按 IP 的审计归因。
- **建议**: 统一 IP 信任模型：仅当请求来自可信代理（nginx）时才解析 XFF/X-Real-IP，否则只用 RemoteAddr；或在 GetRealClientIP 中增加来源网段白名单（如仅信任 127.0.0.1/内网网关）。

### [MEDIUM] 静态资源处理器把用户可控的 *filepath 直接拼接进文件系统路径，缺少显式路径包含校验
- **位置**: 28-35  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: serveImmutableAsset 执行 c.File("./frontend/dist/assets" + c.Param("filepath"))，/assets/ 与 /static/ 两个通配路由共用。实测 Go 1.24 下 net/http 对含 '..' 或 %2e%2e 的路径返回 400（"invalid URL path"），当前不可利用；但该处理器没有任何自身包含校验，一旦未来更换前端（如 Caddy/nginx 透传、旧 Go 版本、或代理层规范化路径），即可越权读取服务器任意文件（.env、cboard.db）。
- **建议**: 改用 http.Dir + 显式校验：解析后 path.Clean 并断言结果仍以 assets 目录为前缀，否则 404；或直接使用 r.Static("/assets", "./frontend/dist/assets") 交给标准库安全实现。

### [MEDIUM] /repo-sync/*filepath 无鉴权公开文件服务，可能泄露仓库同步产物
- **位置**: 37-38  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: r.GET("/repo-sync/*filepath", handlers.ServeRepoSyncFile) 挂在所有中间件之前（维护模式下也可用），路径完全由用户控制。若 handler 未做目录约束，可读取 repo-sync 目录之外的任意文件或泄露内部节点配置。
- **建议**: 检查 ServeRepoSyncFile 是否对 filepath 做 Clean+前缀校验；至少加只读防遍历校验与速率限制，并确认不会回退到默认文件。

### [MEDIUM] GET /nodes 与 /nodes/stats 仅 TryAuth，未登录用户可获取节点地址与统计
- **位置**: 173-178  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: nodes.GET("") 与 nodes.GET("/stats") 使用 TryAuthMiddleware（可选鉴权），未登录/未持有效订阅的访客也能拿到节点列表（服务器地址、端口、协议）与在线状态统计，配合公开的 /coupons 可被爬虫批量采集，增加被滥用/探测风险。
- **建议**: 节点列表改为需登录或需携带有效订阅 token 才返回完整地址；未登录时只返回在线状态等无敏感字段，或直接要求 AuthMiddleware。

### [LOW] 多处重复路由指向同一 handler
- **位置**: 164, 350-351, 238-239  |  **类别**: duplication  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: api.GET("/payment-methods/active") 与 payment.GET("/methods") 均调 GetPaymentMethods；admin.GET("/dashboard") 与 admin.GET("/stats") 均调 GetDashboard；tickets.POST("/:id/reply") 与 "/:id/replies" 均调 ReplyTicket。重复注册同一行为造成契约混乱与维护双份成本。
- **建议**: 保留单一规范路径（如 /payment-methods/active），旧路径改为显式注释说明兼容原因，或由前端统一后删除。

### [LOW] /static/*filepath 被映射到 assets 目录，与路径语义不符
- **位置**: 33  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: r.GET("/static/*filepath", serveImmutableAsset) 与 /assets 共用同一 handler，均指向 ./frontend/dist/assets/，即 /static/foo 实际读 assets/foo。若前端构建产物放在 dist/static/ 下会全部 404；该别名与真实目录结构不一致，属于残留兼容路由。
- **建议**: 删除 /static 别名或在 frontend 构建配置中确认无 /static 前缀引用；如需保留，应映射到 dist/static 目录并补回归测试。

### [LOW] /health 返回硬编码版本号 1.0.0，与 .env VERSION 及实际版本脱节
- **位置**: 40-45  |  **类别**: style  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: health 接口中 "version": "1.0.0" 为写死值，而 bt-deploy.sh 生成的 .env 也写入 VERSION=1.0.0，两者无关联，发布新版本后监控/探活拿到的版本始终错误。
- **建议**: 从 config 读取 VERSION 注入，或在构建期用 -ldflags -X 注入版本号。

### [INFO] 支付回调路由与 CSRF 豁免顺序正确，无明显问题
- **位置**: 65-76  |  **类别**: other  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: POST/GET /payment/notify/:type 注册在 CSRFMiddleware 之前且带专门日志；订阅公共路由用 CSRFExemptMiddleware 隔离；admin 组统一挂 Auth+Admin+NoStore。整体路由分层清晰。
- **建议**: 无需修改；建议后续为 /geoip/batch-lookup 等公开 POST 接口补限流（当前无任何速率限制）。

## internal/core/auth/auth.go

### [MEDIUM] HashPassword 按 72 字节截断可能切断多字节字符
- **位置**: 28-38  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: password[:72]（line 30）是字节切片，对含中文/Emoji 的密码会在多字节字符中间截断，产生无效 UTF-8 字节序列；该哈希对应一个用户永远无法再输入的密码（用户以为的密码与哈希不一致）。ValidatePasswordStrength 的 len(password)（line 41）同样是字节计数，与 line 50 的 rune 遍历语义不一致。
- **建议**: 先按 rune 数校验/截断：[]rune(password) 超过 72 时返回错误提示“密码过长”，或在截断前用 utf8 边界安全截断并记录。

### [LOW] AuthenticateUser 用 LOWER(email) 函数查询导致索引失效
- **位置**: 97-109  |  **类别**: performance  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: db.Where("LOWER(email) = ?", emailNorm)（line 100）在 MySQL/Postgres 上无法命中 email 列的普通 B-tree 索引，用户量增大后登录查询变全表扫描。同时项目已有 utils.NormalizeEmail（common.go:30）但此处未复用，规范化逻辑分散。
- **建议**: 注册时存规范化邮箱（小写 trim），登录时精确匹配 email = ? 以命中索引；确需 LOWER 则建函数索引。

### [INFO] VerifyPassword 的格式预检合理，无明显问题
- **位置**: 14-26  |  **类别**: other  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: len<7 与前缀 $2a/$2b/$2y 检查（line 19-22）能廉价拒绝非法哈希（防无效 bcrypt 调用）；AuthenticateUser 返回统一错误文案不泄露用户是否存在。整体健康。
- **建议**: 无。

## internal/core/cache/redis.go

### [MEDIUM] Redis 键无应用命名空间前缀，且使用包级共享 context
- **位置**: 12-16  |  **类别**: architecture  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 键形如 user:1、users:list、subscription:config:<url>:clash，无 "cboard:" 前缀；与共用同一 Redis 实例的其他应用存在键冲突，且 cache.FlushAll()（FlushDB）会清空整个 DB 0。所有操作复用无超时的 context.Background()（line 14），Redis 卡死会拖住请求。
- **建议**: 统一键前缀常量 cboard:；每次调用用 context.WithTimeout(ctx, 1-2s)；Redis 配置支持 DB 号环境变量。

### [LOW] redisDB 硬编码 0，不支持 REDIS_DB 环境变量
- **位置**: 27  |  **类别**: style  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: redisDB := 0 直接写死（line 27），多环境隔离（测试/生产分库）无法通过配置实现。
- **建议**: 读取 REDIS_DB 环境变量（strconv.Atoi，默认 0）。

### [INFO] ClearSubscriptionConfigCache 双键删除实现正确
- **位置**: 110-130  |  **类别**: other  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: clash 与 base64 两个键一次 Del 删除（line 126），失败时返回包装错误；带 Context 变体供请求内调用。无明显问题。
- **建议**: 无。

## internal/core/cache/user_cache.go

### [MEDIUM] GetUsersBatch 循环内逐个 GET，N 次往返
- **位置**: 84-100  |  **类别**: performance  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: for 循环里每个 id 一次 redisClient.Get（line 85-99），O(N) 次网络往返；注释“逐个获取，简化实现”表明是有意简化。数据量大时（批量拉用户列表）延迟线性增长。
- **建议**: 用 redisClient.MGet(ctx, keys...) 一次取回（NIL 处理同现状），或复用 Pipeline。

### [LOW] 整个 UserCache 文件是死代码（零调用方）
- **位置**: 17-212  |  **类别**: maintainability  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 全库 grep 确认 UserCache、包级 userCache 及其全部方法（GetUser/SetUser/GetUsersBatch/GetOrSetUser/GetUserStats 等）没有任何调用方；用户认证实际走 middleware/auth.go 的 sync.Map 缓存。近 200 行无人使用的缓存层增加了维护负担，且其 GetOrSetUser 存在缓存击穿风险（无 singleflight），一旦启用即踩坑。
- **建议**: 删除该文件，或若计划启用：GetUsersBatch 改用 MGET/Pipeline、GetOrSetUser 加 singleflight（如 golang.org/x/sync/singleflight）防惊群。

### [LOW] 常量声明位置与对齐不一致
- **位置**: 166-168  |  **类别**: style  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: userStatsCacheKey / userStatsCacheExpire 声明在方法之后（line 167-168），而 userCachePrefix 等常量在文件头部（line 22-27），且头部常量对齐未 gofmt（userListCacheKey 与 userCacheExpire 缩进不一致）。
- **建议**: 常量统一收拢到文件顶部 const 块并 gofmt 对齐。

## internal/core/config/config.go

### [HIGH] 生产环境校验只检查 SECRET_KEY，与 getSecretKey 支持的 JWT_SECRET_KEY 不一致
- **位置**: 276-281  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: getSecretKey（line 205-235）明确支持 JWT_SECRET_KEY 环境变量（优先级最高，其警告日志也写“JWT_SECRET_KEY 或 SECRET_KEY”），但 validateConfig 只校验 viper 的 SECRET_KEY（line 278）。生产环境仅设置强 JWT_SECRET_KEY 时，SECRET_KEY 为空 → isWeakSecretKey("")=true → 启动直接失败，与自身文档矛盾，属于典型误配置陷阱。
- **建议**: validateConfig 复用同一个有效密钥来源：effective := os.Getenv("JWT_SECRET_KEY"); if effective=="" { effective = viper.GetString("SECRET_KEY") } 再 isWeakSecretKey(effective)。

### [MEDIUM] 生产校验强制同时要求 MySQL 与 Postgres 密码，与所选数据库无关
- **位置**: 282-287  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 使用 Postgres 或 SQLite 的生产部署也会因为 MySQLPassword==""（line 282-284）而启动失败（反之亦然）。校验应只针对 DatabaseURL/USE_* 实际选中的数据库，否则合法部署被误杀。
- **建议**: 根据 cfg.DatabaseURL 前缀判断生效的数据库类型，只校验对应密码；SQLite 部署跳过密码校验。

### [LOW] getInt 把 0 当作未设置，无法显式配置 0 值
- **位置**: 160-166  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: getInt 在 viper.GetInt(key)==0 时返回默认值（line 162），导致 JWT_EXPIRE_HOURS=0、WORKERS=0 等合法（或至少应显式拒绝的）输入被静默替换，且无法区分“未设置”与“显式为 0”。
- **建议**: 用 viper.IsSet(key) 判断是否显式设置（与 getBool 一致），再决定是否取默认值。

### [LOW] Algorithm（JWT_ALGORITHM）字段配置后从未被使用
- **位置**: 30-104  |  **类别**: maintainability  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 全库 grep 确认 Algorithm 仅在 config.go 赋值（line 30/103），common.go 的 CreateAccessToken/RefreshToken 硬编码 SigningMethodHS256。用户配置 JWT_ALGORITHM=RS256 等完全无效，属误导性配置。
- **建议**: 要么实现基于 Algorithm 的签发逻辑（HS256/HS384/RS256 分支），要么删除该配置并在文档中注明仅支持 HS256。

## internal/core/database/database.go

### [HIGH] custom_nodes 表重建：备份语句错误被忽略后仍 DROP 原表，存在数据丢失风险
- **位置**: 187-229  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: line 198 DB.Exec("CREATE TABLE custom_nodes_backup AS SELECT * FROM custom_nodes") 与 line 223 的备份执行结果均未检查 .Error；随后无条件 DB.Exec("DROP TABLE custom_nodes")（line 201/225）。若备份因磁盘满/权限失败，原表数据被永久删除；且整个重建（备份→DROP→AutoMigrate 重建）不在事务内，AutoMigrate 后续失败时旧数据已不可恢复。
- **建议**: 每步检查 .Error，备份失败立即 abort 迁移并提示人工处理；将“备份+重建”包进 gorm 事务；DROP 前用 count 校验备份行数与源表一致。

### [MEDIUM] 迁移中的 Raw/Scan 与 DDL 执行普遍不检查错误
- **位置**: 149-234  |  **类别**: error-handling  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: line 157/160/171-186 等 DB.Raw(...).Scan(&xxx) 均未检查 .Error，表结构探测失败时按“不存在”继续，可能触发错误的 ALTER/DROP 分支；line 196-198 的备份 DDL 同样未检错。一旦 SQLite 探测查询失败（只读库、锁），迁移路径会走错分支。
- **建议**: 为每个探测/DDL 语句检查 .Error，失败时 log 并返回错误，避免静默走错迁移分支。

### [MEDIUM] NullInt64/NullFloat64 无条件 Valid:true，0 值无法表达 NULL
- **位置**: 444-469  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: NullInt64(0) 返回 Valid:true 的 0 而非 NULL（NullString/NullTime 才按内容判 Valid，line 444-449/465-469）。审计/日志等调用方传 0 时会写入 0 而非 NULL，改变数据库语义（如 sum 统计、is null 过滤），且与 NullString 的语义不一致。
- **建议**: NullInt64(i)：Valid: i != 0；或明确区分“零值也有效”的两个 helper，并在调用处按语义选用。

### [LOW] CloseDatabase 对所有数据库类型执行 SQLite PRAGMA
- **位置**: 420-434  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")（line 424）在 MySQL/Postgres 连接上是非法 SQL，错误被忽略；另外 wal_checkpoint 可能长时间阻塞（WAL 较大时），且 close 流程无超时。
- **建议**: 仅当 DB.Dialector.Name()=="sqlite" 时执行 checkpoint，其余类型直接 sqlDB.Close()。

### [LOW] HasColumn 使用 Go 字段名 FulfilledAt 探测列，可能永远为 false
- **位置**: 153-288  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: GORM migrator 的 HasColumn 对模型按命名策略取列名，传入 "FulfilledAt"（line 153/288）大概率匹配不到实际列 fulfilled_at（SQLite 分支用了 pragma 所以没问题；MySQL/Postgres 分支 fulfilledAtExisted 可能恒 false，触发不必要的回填 UPDATE 逻辑判断）。
- **建议**: SQLite 分支外统一用 DB.Migrator().HasColumn(&models.Order{}, "fulfilled_at")（数据库列名）探测。

## internal/middleware/auth.go

### [MEDIUM] 2 分钟内存用户缓存导致禁用/改权存在 TOCTOU 窗口
- **位置**: 17-46  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: AuthMiddleware 从 authUserCache 取用户（TTL 2 分钟，line 24）并只检查缓存中的 IsActive/IsAdmin（line 151-158）。InvalidateAuthUserCache 仅 3 处调用（user.go:2209 状态更新、auth.go:987、dashboard.go:456），管理员改密码、用户自改资料（邮箱/用户名）、其他用户变更路径不会失效缓存：被禁用/降权的用户最长 2 分钟内仍持有原权限（如降权管理员仍可访问 admin 接口）。
- **建议**: admin 路由每次从 DB 校验 IsAdmin/IsActive（admin 流量低，可接受），或缩短 TTL、并在所有用户写路径统一调用 InvalidateAuthUserCache。

### [LOW] AuthMiddleware 与 TryAuthMiddleware 重复约 40 行认证逻辑
- **位置**: 88-255  |  **类别**: duplication  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: token 解析、黑名单检查、VerifyToken、claims.Type 校验、用户加载/缓存写入在 line 88-166 与 line 201-255 完全重复，仅失败处理不同（abort vs c.Next()）。后续加安全逻辑（如版本号校验）需改两处。
- **建议**: 抽公共函数 resolveUser(c, token) (user, claims, err)，两个中间件分别定义失败策略。

### [INFO] token 黑名单按 token 哈希缓存 5 分钟，DB 查询频率可控
- **位置**: 64-78  |  **类别**: performance  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: isTokenBlacklistedCached 对每个唯一 token 只查一次 DB（5 分钟 TTL，line 62-78），models.IsTokenBlacklisted 的查询有 token_hash 唯一索引且带 expires_at 条件，模型层实现正确。无明显问题。
- **建议**: 无。

## internal/middleware/brotli.go

### [HIGH] compressResponseWriter 未转发 Flush，全局压缩下 SSE 推送失效
- **位置**: 26-37  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: router.go:20 将 CompressionMiddleware 全局挂载；compressResponseWriter 只重写 Write/WriteString，Flush 被提升到内嵌的原始 gin.ResponseWriter（未压缩）。subscription.go:2285/2298 的 SSE 端点 StreamConfigUpdateLogs 调 c.Writer.Flush() 时，数据还滞留在 brotli/gzip 缓冲里，原始 writer 刷出的是空/不完整字节流，且 Content-Encoding 已声明压缩——浏览器解压失败或事件延迟到缓冲满才到达，SSE 实际不可用。
- **建议**: 实现 Flush() 转发到压缩 writer：w.writer.(interface{ Flush() error }).Flush()；对 SSE 路由（text/event-stream）整体跳过压缩（按 Content-Type 判断）。

### [MEDIUM] c.Next() 内 panic 时压缩 writer 不 Close、不进池，且响应流损坏
- **位置**: 51-79  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: bw.Close()/gzip 的 Close 与 pool.Put 都在 c.Next() 之后顺序执行（line 61-62/77-78）；handler panic 时直接跳过，writer 泄漏（池项丢失），且未写入的压缩数据不会发送。ErrorRecoveryMiddleware 虽能 recover，但压缩 writer 已处于悬挂状态。
- **建议**: 用 defer 包裹 Close+Put：defer func(){ bw.Close(); brotliWriterPool.Put(bw) }()，确保 panic 路径也回收。

### [LOW] 未实现 ReadFrom，文件流式响应走 Write 逐块拷贝
- **位置**: 31-37  |  **类别**: performance  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: c.File（router.go:30/582，handler repo_sync.go:92）通过 io.Copy 下发，compressResponseWriter 无 ReadFrom，退化为每次 Write 调用进入压缩器，少一次 sendfile/零拷贝优化机会。
- **建议**: 可选实现 ReadFrom 直接灌入压缩 writer；对已压缩内容（图片等）按 Content-Type 跳过压缩更实际。

## internal/middleware/csrf.go

### [HIGH] 每次 GET 都重新生成并覆盖 CSRF token，多标签页/并发 GET 使合法请求 403
- **位置**: 127-137  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: GET/HEAD/OPTIONS 分支对每个请求调用 GenerateToken(sessionID)（line 129），而 GenerateToken 用 map[sessionID] 覆盖写（line 76-79）。同一会话打开两个标签页、或表单加载后有任何 GET（轮询/资源刷新）都会使已下发 token 失效：后续 POST 携带旧 token → 403，同时每个失败都会写一条 MEDIUM 安全日志（line 174-176），制造噪音。前端 api.js（frontend/src/utils/api.js:363-377）靠 403 重试自愈，但多标签场景仍会出现间歇性 CSRF 失败。
- **建议**: token 与 session 绑定但不在每次 GET 轮换：首次发放后复用，直到过期（24h）；或采用 double-submit cookie 模式（token 存 cookie + 请求头比对），无需服务端状态。

### [MEDIUM] CSRF token 仅存进程内存，多副本部署会间歇性 403
- **位置**: 17-64  |  **类别**: architecture  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: CSRFManager.tokens 是进程内 map（line 18）。配置项 Workers=4（config.go:52）暗示可能多进程/多副本部署；token 由实例 A 发放、请求落到实例 B 时 ValidateToken 必然失败，出现随机 403。且进程重启后所有已发放 token 失效。
- **建议**: 改用无状态的 double-submit cookie 方案，或把 token 状态放入 Redis（带 TTL）。

### [MEDIUM] CSRFExemptMiddleware 设置 csrf_exempt 但 CSRFMiddleware 从不读取
- **位置**: 271-276  |  **类别**: maintainability  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: router.go:128 给 subscribePublic 挂 CSRFExemptMiddleware，它只是 c.Set("csrf_exempt", true)（line 273）；CSRFMiddleware（line 117-184）全流程未检查该标志。当前 subscribe 路由恰好都是 GET 本就豁免，但若未来加 POST 路由，豁免静默失效；该中间件是功能性死代码。
- **建议**: 在 CSRFMiddleware 校验前检查 c.GetBool("csrf_exempt") 直接放行；或删除 CSRFExemptMiddleware 与 router.go:128。

### [LOW] isValidReferer 对配置源用前缀匹配，子域后缀可绕过
- **位置**: 246-252  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: strings.HasPrefix(referer, o)（line 248）中 o 为配置源如 "https://example.com"，referer "https://example.com.evil.com/page" 会被判定合法（合法域名后接任意后缀）。Origin 校验用的是相等比较（line 207）无此问题，Referer 校验口径不一致。
- **建议**: 用 url.Parse(referer) 取 host，与配置源 host 做精确相等或规范的后缀匹配（host==o.host 或 strings.HasSuffix(host, "."+o.host)）。

## internal/middleware/maintenance.go

### [HIGH] 维护页面对 siteName/message/logoURL 未转义直接拼入 HTML，存在存储型 XSS
- **位置**: 125-158  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: fmt.Sprintf 将 data.siteName、data.message、getLogoHTML(data.logoURL) 原样插入 HTML（line 158），getLogoHTML 把 logoURL 直接放进 src="%s"（line 172）。这些值来自 SystemConfig（管理员可改，或设置接口被攻破时）：可注入 <script>/onerror 载荷，维护模式下全站用户都会加载该页面 → 存储型 XSS 且传播面大。
- **建议**: 用 html/template 渲染，或对三个字段先 html.EscapeString；logoURL 仅允许 http/https/data:image 白名单 scheme。

### [LOW] 缓存过期瞬间并发请求全部回源查询 DB（无 singleflight）
- **位置**: 32-84  |  **类别**: performance  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 双检锁只保证单实例不重复写，但在缓存过期后的同一瞬间，多个并发请求都会通过第一层 RLock 检查并各自执行 DB 查询（line 49-63），属轻微惊群。
- **建议**: 加 singleflight 或每实例只允许一个刷新者（如短锁 + 刷新中标志）。

### [INFO] 维护白名单设计合理（admin/登录/健康检查放行），无安全问题
- **位置**: 97-106  |  **类别**: other  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: allowedPaths（line 97-106）覆盖登录、健康检查、支付回调与静态资源，其余 API 在维护期间返回 503；路径前缀匹配可控。无明显问题。
- **建议**: 无。

## internal/middleware/ratelimit.go

### [HIGH] 登录/注册/验证码限流键取自可伪造的客户端头，攻击者可完全绕过限流
- **位置**: 241-295  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: LoginRateLimitMiddleware/RegisterRateLimitMiddleware/VerifyCodeRateLimitMiddleware 的 key 全部来自 utils.GetRealClientIP（line 243/311/346），而该函数无条件信任 CF-Connecting-IP/True-Client-IP/X-Forwarded-For（network.go:180-215）；router.go:15 甚至显式 SetTrustedProxies(nil) 表明部署不信任代理头。攻击者每次请求换一个伪造公网 IP，登录失败锁定（5 次/15 分钟）形同虚设，暴力破解/撞库/验证码轰炸完全不受限。
- **建议**: 仅在配置了可信代理（如 TRUSTED_PROXY 环境变量）时才解析代理头；否则一律用 c.ClientIP()（已按 SetTrustedProxies 处理）或 RemoteAddr。

### [MEDIUM] 限流状态仅存进程内存：多实例 N 倍放宽、重启清零
- **位置**: 17-66  |  **类别**: architecture  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: visitors map 与清理 goroutine 均为进程内状态（NewRateLimiter line 40 起 goroutine），多 worker/多副本部署时每个实例独立计数，攻击者可把请求分散到各实例实现 N 倍绕过；进程重启后锁定全部失效。ReloadLoginRateLimiter 也只在启动与 settings 更新时加载（config.go:104 调用）。
- **建议**: 登录/注册等高危限流改用 Redis 原子计数（INCR+EXPIRE）或现成库（ulule/limiter 带 Redis store）。

### [MEDIUM] RateLimitMiddleware 与 generalRateLimiter 从未被挂载（死代码），且响应头硬编码 100
- **位置**: 208-239  |  **类别**: maintainability  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 全库 grep 确认 RateLimitMiddleware 无任何调用点，generalRateLimiter（line 176）仅定义未使用；同时该中间件成功/失败分支的 X-RateLimit-Limit 都硬编码 "100"（line 225/234），与 limiter.rate 无关，若未来启用会返回误导性头。
- **建议**: 挂载到普通 API 组（api.Use(middleware.RateLimitMiddleware(generalRateLimiter))），并把头值改为 strconv.Itoa(limiter.rate)。

### [LOW] Allow/Check 双路径耦合脆弱：锁定只发生在显式 IncrementLoginAttempt 时
- **位置**: 68-139  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 中间件用 Check（不计数，line 248），真正的计数/锁定在 handlers/auth.go 里显式调 IncrementLoginAttempt(ip)（line 535/672/686）才发生；一旦某登录分支漏调，限流完全不生效。且 IncrementLoginAttempt 的 key 与中间件 key 都取自可伪造 IP（同前述问题）。
- **建议**: 让中间件自身对失败计数（登录 handler 返回状态后由统一逻辑计数），避免双路径不一致。

## internal/middleware/security.go

### [MEDIUM] CSP 允许 unsafe-inline/unsafe-eval，XSS 防线基本失效
- **位置**: 24  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: script-src 'self' 'unsafe-inline' 'unsafe-eval'（line 24）使 CSP 对脚本注入几乎无防护（任何内联脚本都可执行）；connect-src 'self' 可能阻断跨域 API/订阅客户端调用。生产构建（Vite）不需要 unsafe-eval。
- **建议**: 按环境区分 CSP：生产去掉 unsafe-inline/unsafe-eval，connect-src 按实际调用域（API、订阅）白名单配置。

### [LOW] CORS 在配置缺失时回退 AllowOrigins:["*"] 且 AllowCredentials:true
- **位置**: 41-56  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: config.AppConfig 为 nil 时 origins 为 ["*"]（line 43-46），与 AllowCredentials:true 组合会被浏览器拒绝（且若未来代码路径配置了 cookie 会话，通配源+凭据是危险组合）；正常路径下配置源列表是安全的。
- **建议**: 回退时也禁止通配符：cfg 为 nil 时直接使用与 validateConfig 一致的本地源默认值，或报错拒绝启动。

### [INFO] 请求 ID/恢复/安全头中间件实现正确
- **位置**: 117-131  |  **类别**: other  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: RequestIDMiddleware 透传或生成 X-Request-ID 并写入 context；ErrorRecoveryMiddleware 对生产隐藏堆栈、通过 SanitizeErrorPath 清理路径后落日志，并用 system_error_logged 防重复记录。无明显问题。
- **建议**: 无。

## internal/models/activity.go

### [MEDIUM] GetLocationInfo 用逗号嗅探 CSV/JSON 格式，多键 JSON 必然走错分支
- **位置**: 48-72  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 第 53 行用 `strings.Contains(locationStr, ",")` 决定解析方式：任何含多个键的 JSON（如 `{"country":"US","city":"NY"}`）都含逗号，会进入 CSV Split 分支，得到 country=`{"country":"US"`、city=`"city":"NY"}` 这样的垃圾值；JSON 分支实际只对无逗号的单键 JSON 可达，属于死分支。存 JSON 还是 CSV 取决于写入方，格式识别不应靠逗号。
- **建议**: 先判断 `strings.HasPrefix(strings.TrimSpace(locationStr), "{")` 走 JSON 分支，否则按 CSV 处理；或干脆新增 country/city 两个独立列，彻底去掉格式嗅探。

### [LOW] CreatedAt 同时有单列 index 与复合索引，存在冗余索引写放大
- **位置**: 19  |  **类别**: performance  |  **来源组**: A2-models (全部模型)
- **问题**: `CreatedAt` 声明了 `index`（自动生成 idx_user_activities_created_at）又作为 `idx_user_activities_user_created_at` 的 priority:2，单列索引与复合索引前缀重复；活动表写入量大，冗余索引增加每次 INSERT 成本。
- **建议**: 确认无单独按 created_at 的查询后去掉单列 `index`，只保留复合索引（user_id, created_at）即可覆盖用户维度时间排序。

## internal/models/audit_log.go

### [MEDIUM] RequestParams/BeforeData/AfterData 原样落库，密码/令牌等敏感参数明文进审计表
- **位置**: 20-23  |  **类别**: security  |  **来源组**: A2-models (全部模型)
- **问题**: `RequestParams sql.NullString type:json` 直接序列化请求参数（含登录/改密接口的 password、token 等）并以明文 JSON 存 text 列；BeforeData/AfterData 同理可能包含用户敏感字段。审计日志保留期长，等于敏感信息长期明文存储。
- **建议**: 写入审计前做脱敏白名单：过滤 password/secret/token/code 等 key（可用统一的 sanitize 函数），或对整列加密存储。

### [LOW] 缺少 (resource_type, resource_id) 复合索引，资源维度审计查询退化
- **位置**: 10-13, 24  |  **类别**: performance  |  **来源组**: A2-models (全部模型)
- **问题**: ResourceType、ResourceID 各自只有单列索引，按「某资源的所有审计」查询（WHERE resource_type=? AND resource_id=?）无法命中单一索引前缀；同时 CreatedAt 上挂了 3 个复合 + 1 个单列共 4 个索引，写放大明显。
- **建议**: 新增 (resource_type, resource_id, created_at) 复合索引，删除冗余的单列 resource_type/created_at 索引（保留 user_id 相关复合索引）。

## internal/models/checkin.go

### [MEDIUM] 无数据库级防重复约束，同日重复签到依赖服务层事务
- **位置**: 5-10  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: CheckinRecord 只有 (user_id, created_at) 复合索引，没有唯一性约束防止同一用户同一天多次签到；并发请求下若服务层先查后插（无唯一约束兜底）会写入重复签到记录并多发余额。
- **建议**: 增加 `CheckinDate string` 列并建 `uniqueIndex:(user_id, checkin_date)`，让数据库成为最终防线；金额校验 Amount>0 也建议下沉到模型校验。

### [LOW] Amount float64 无取值约束
- **位置**: 8  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 签到金额用 float64 decimal(10,2)，模型层无非负/非零校验；若服务层漏校验，可写入负数金额（恶意构造或 bug）。
- **建议**: 在创建入口统一校验 Amount>0，或定义模型级 BeforeCreate 钩子做防御。

## internal/models/config.go

### [MEDIUM] SystemConfig.Value 明文 text 存储密钥类配置，且获取配置接口全量回传
- **位置**: 8-18  |  **类别**: security  |  **来源组**: A2-models (全部模型)
- **问题**: 支付密钥、SMTP 密码等敏感配置与普通配置共用明文的 Value 列；admin.go:945-948 的获取配置接口把整个 configMap 直接返回前端，一旦某 key 属于敏感项即泄露。IsPublic 只是查询侧标记，不防泄露。
- **建议**: 按 key 白名单区分敏感项：敏感值加密存储（AES-GCM）+ 接口返回时脱敏（如只回显掩码），仅 IsPublic 的 key 可进公开接口。

### [LOW] Announcement 时间窗口无校验，TargetUsers 自由字符串
- **位置**: 32-34  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: StartTime/EndTime 为指针但模型不校验 end>start；TargetUsers 是 default:all 的自由字符串（'all'/'vip'/'user:123' 等语义未枚举），前端与后端契约易漂移。
- **建议**: 在保存服务处校验时间窗口合法性；TargetUsers 改为枚举 + 独立关联表（announcement_targets）支持按用户/等级定向。

## internal/models/coupon.go

### [MEDIUM] DiscountValue 一个 float 字段承载三种券类型语义（百分比/固定金额/赠送天数）
- **位置**: 12-15, 25-46  |  **类别**: architecture  |  **来源组**: A2-models (全部模型)
- **问题**: discount 时是折扣比例、fixed 时是金额、free_days 时是天数，全部塞进 `DiscountValue float64`；Type 为裸字符串（常量虽定义但字段类型仍是 string，编译期不约束）。计算优惠的服务代码必须按 Type 分支解释该字段，极易误用（如把 0.8 当 0.8 元）。
- **建议**: 为三种类型分别定义字段（Percent/FixedAmount/FreeDays）或按类型定义子结构，用 typed constant 且字段类型用 `CouponType`；金额字段全部改为整数分。

### [MEDIUM] MinAmount NullFloat64 配 `default:0`：NULL 语义失效，响应恒带 min_amount
- **位置**: 32, 91-93  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: `sql.NullFloat64` 零值（Valid=false）会被 GORM 的 default:0 写库为 0，读回后 Valid 恒为 true，ToCouponResponse 里 `if c.MinAmount.Valid` 恒真——「无最低消费限制」被序列化成 min_amount=0，前端无法区分 0 与未设置；MaxDiscount 无 default 则行为正常，两者语义不一致。
- **建议**: 去掉 `default:0` 标签让 NULL 保留；或反过来给 MinAmount 用非空 float64 + 语义上的 0=无限制，并统一 CouponResponse 的指针输出逻辑。

### [LOW] ToCouponResponse + MarshalJSON 模式与 user_level.go 完全重复
- **位置**: 73-109  |  **类别**: maintainability  |  **来源组**: A2-models (全部模型)
- **问题**: coupon.go 与 user_level.go 各自手写一份「脱敏响应结构体 + ToXxxResponse + MarshalJSON」样板；新增需要脱敏的模型会继续复制第三份。
- **建议**: 抽一个泛型 helper 或统一用 `json:"-"` 隐藏敏感字段 + 单结构体输出，减少样板。

## internal/models/custom_node.go

### [MEDIUM] Protocol 字段与 NodeConfig.Type 双份协议来源，命名不一致
- **位置**: 11, 26-28  |  **类别**: architecture  |  **来源组**: A2-models (全部模型)
- **问题**: CustomNode.Protocol（默认 ''）和 NodeConfig.Type 描述同一件事但名字不同；handlers/custom_node.go:168 的 normalizeCustomNodeConfig 要同时维护 Protocol/Domain/Port 与 Config 内 JSON 的一致性，任一写路径漏同步就产生「字段说 vless、Config 里 type=vmess」的漂移。
- **建议**: 以 Config JSON 为唯一真相源，Protocol/Domain/Port 只作展示冗余（写入时从 Config 反解）；或删掉 NodeConfig 结构体、直接落结构化列。

### [LOW] Status 与 IsActive 双状态字段，优先级语义未定义
- **位置**: 15-16  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: `Status string default:inactive` 与 `IsActive bool default:true` 并存（node.go 同样如此），出现 status=inactive + is_active=true 的组合时节点是否可用取决于各查询处如何拼条件，容易不一致。
- **建议**: 收敛为单一状态（枚举 active/inactive 或 bool），另一字段作为派生展示值或直接删除。

## internal/models/device.go

### [MEDIUM] LastAccess 用 autoCreateTime 标记：仅建行时赋值，后续访问不会自动更新
- **位置**: 29  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 设备每次访问都应刷新 LastAccess（设备管理页「最后访问」），但 `gorm:"autoCreateTime"` 只在 INSERT 时写一次；服务层若依赖 GORM 自动更新（未手动 Set），LastAccess 将永远停留在首次创建时间，与 LastSeen 语义混淆。
- **建议**: 改为普通 `time.Time` 并在访问打点处显式 `UPDATE devices SET last_access=?, access_count=access_count+1 WHERE id=?`（顺带把 AccessCount 的读改写竞态一并消除）。

### [LOW] DeviceFingerprint 无唯一性约束，AccessCount 非原子自增
- **位置**: 11, 31  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 同一设备指纹在同一订阅下可重复插入多行（靠服务层先查后插）；AccessCount 若按读-改-写更新在并发访问下丢计数。
- **建议**: 对 (subscription_id, device_fingerprint) 建唯一索引并 upsert；计数用原子 UPDATE 表达式。

## internal/models/invite.go

### [HIGH] invite_relations 无 InviteeID 唯一约束，并发注册可产生多条邀请关系重复发奖
- **位置**: 33-45  |  **类别**: security  |  **来源组**: A2-models (全部模型)
- **问题**: 一个被邀请用户应只属于一条邀请关系；表上只有单列索引，没有 (invitee_id) 唯一约束。并发注册（同一邀请码被多人同时使用、或同一人并发注册）时服务层查重后插入会双双成功，导致双份邀请奖励入账。
- **建议**: 给 InviteeID 加唯一索引；发放奖励的流程用事务 + `INSERT ... ON CONFLICT DO NOTHING` 幂等语义，并在 CommissionLog 上以 (invite_relation_id, commission_type) 唯一索引兜底（见 logs.go 发现）。

### [LOW] PackageIDs 逗号拼接文本、MaxUses/UsedCount 计数非原子
- **位置**: 13, 18  |  **类别**: maintainability  |  **来源组**: A2-models (全部模型)
- **问题**: `PackageIDs sql.NullString type:text` 存逗号分隔的套餐 ID，解析要 Split + 类型转换，无法用 SQL 过滤；UsedCount 自增同样存在读改写竞态。
- **建议**: PackageIDs 改关联表 invite_code_packages 或 JSON 数组列；UsedCount 用原子 UPDATE 并在 MaxUses 处做条件更新。

## internal/models/knowledge.go

### [MEDIUM] ViewCount 读改写自增存在并发丢更新
- **位置**: 28  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 浏览量若在服务层 `SELECT view_count → +1 → UPDATE` 更新，并发浏览会互相覆盖，计数偏小；模型层无原子保障。
- **建议**: 用 `UPDATE knowledge_articles SET view_count = view_count + 1 WHERE id=?` 原子自增，或引入 Redis 计数后异步落库。

### [LOW] 文章列表查询缺少 (category_id, sort_order, is_active) 复合索引
- **位置**: 24-31  |  **类别**: performance  |  **来源组**: A2-models (全部模型)
- **问题**: CategoryID 有单列索引，但分类下按 sort_order 排序 + is_active 过滤的查询（前端知识库最常见路径）无法走单一索引，分类内文章多时回表排序。
- **建议**: 新增 (category_id, is_active, sort_order) 复合索引，覆盖列表主查询。

## internal/models/logs.go

### [MEDIUM] CommissionLog 无 (invite_relation_id, commission_type) 唯一约束，佣金可重复结算
- **位置**: 83-99  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 订单佣金、注册奖励等按 commission_type 记录，但表上无唯一约束；结算任务重跑或并发执行时，同一条邀请关系可插入两条相同类型佣金记录，造成重复发放。
- **建议**: 加 `uniqueIndex:(invite_relation_id, commission_type)`（对 register_reward 这类每关系一次的类型），结算用 upsert；order_commission 可再加 (related_order_id, commission_type) 唯一键。

### [MEDIUM] 四个日志模型重复同一段字段块（IP/UA/Location/Description/Operator）
- **位置**: 8-103  |  **类别**: maintainability  |  **来源组**: A2-models (全部模型)
- **问题**: RegistrationLog、SubscriptionLog、BalanceLog、CommissionLog 各自手写 IPAddress/UserAgent/Location/Description/Operator(UserID)/CreatedAt 等字段（audit_log.go 也是），字段定义、json tag、索引策略逐份复制，后续加字段要改五处。
- **建议**: 抽取嵌入式结构体 `LogBase{ IPAddress; UserAgent; Location; Description; CreatedAt ... }` 嵌入各日志模型，统一 tag 与索引策略。

## internal/models/node.go

### [LOW] Status/IsActive/IsManual 三状态并存且 Config 用 text 存 JSON
- **位置**: 12, 18-21  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 节点同时有 status(offline/online)、is_active、is_manual 三个布尔/枚举维度，查询处拼接条件易不一致；`Config *string type:text` 与其他模型的 `type:json` 风格不一致（MySQL 下 text 列无 JSON 校验）。
- **建议**: 状态收敛为单一枚举；Config 改 `type:json`（SQLite 下退化为 text 也无碍），并统一 status 常量。

### [LOW] Load/Speed/Uptime/Latency 单位与语义仅靠命名，无注释
- **位置**: 13-16  |  **类别**: maintainability  |  **来源组**: A2-models (全部模型)
- **问题**: Uptime 是秒还是分钟、Load 是 0-1 还是百分比、Speed 是 Mbps 还是倍数，模型层完全无注释，前后端展示与监控采集处各自猜测，容易发生 1000 倍量级换算错误。
- **建议**: 为这些字段补单位注释（如 `// Uptime 秒`），并固定为整型秒、0.0-1.0 负载、Mbps 速度的单一约定。

## internal/models/notification.go

### [MEDIUM] 未读数查询无 (user_id, is_read) 复合索引
- **位置**: 10, 14  |  **类别**: performance  |  **来源组**: A2-models (全部模型)
- **问题**: 站内信最频繁的查询是 `WHERE user_id=? AND is_read=false ORDER BY created_at DESC`；UserID 有单列索引但 is_read 无索引，用户通知量大时该查询需回表过滤 + filesort。
- **建议**: 加 `index:(user_id, is_read, created_at)` 复合索引；广播通知（user_id IS NULL）单独走 is_active 索引。

### [LOW] EmailQueue.Attachments 用 text 存 JSON，且无 CreatedAt 索引
- **位置**: 42-57  |  **类别**: maintainability  |  **来源组**: A2-models (全部模型)
- **问题**: 附件列表以字符串 JSON 存 text 列（与全库 `type:json` 风格不一致）；邮件轮询按 status=pending 取件但没有 created_at 索引，队列积压时排序扫描变慢。
- **建议**: Attachments 改 `type:json`；加 (status, created_at) 复合索引并配合重试时间字段。

## internal/models/order.go

### [HIGH] 订单金额用 float64 decimal(10,2)，叠加 DiscountAmount 的 default:0 使 NULL 语义失效
- **位置**: 13, 22-23  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: Amount/DiscountAmount/FinalAmount 全部 float64：浮点加减在累计优惠、退款分摊时会产生 0.1+0.2 类精度误差；且 `DiscountAmount sql.NullFloat64 gorm:"...;default:0"` 与 coupon.MinAmount 同病——GORM 把零值写 0，Valid 恒 true，无折扣订单也序列化 discount_amount=0，契约上无法区分。
- **建议**: 金额统一改为 int64 分（或 decimal 库），去掉 DiscountAmount 的 default:0 让 NULL 表达「无折扣」，FinalAmount 计算收敛到服务层一处并做金额守恒断言（Amount-Discount==FinalAmount）。

### [LOW] PaymentMethodName 冗余快照 + ExtraData 自由 JSON 文本
- **位置**: 16, 24  |  **类别**: architecture  |  **来源组**: A2-models (全部模型)
- **问题**: PaymentMethodName 把支付方式名冗余进订单（可接受但需保证与 payment_configs 同步）；ExtraData 用 `type:text` 存 JSON，无 schema 约束，前端与回调写入方各自定义 key 结构。
- **建议**: ExtraData 改 `type:json` 并文档化 key 契约；PaymentMethodName 写入统一走支付服务赋值，避免多处拼接。

## internal/models/package.go

### [MEDIUM] 设备上限策略三处重复（Package/Subscription/UserLevel），无优先级解析约定
- **位置**: 14  |  **类别**: architecture  |  **来源组**: A2-models (全部模型)
- **问题**: DeviceLimit 同时存在于套餐（default:3）、订阅（无默认值）、用户等级（default:3）三张表，查询处按什么顺序回退（等级>套餐>订阅?）完全靠服务层临时决定，改一处漏一处就会产生「升级套餐后设备数不变」类问题。
- **建议**: 把设备上限收敛为单一来源（如订阅创建时从套餐快照落库），UserLevel 仅作展示/兜底，并抽一个 `ResolveDeviceLimit(sub, pkg, level)` 公共函数。

### [LOW] Price/DurationDays 无正数校验
- **位置**: 12-13  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 模型层不约束 Price>0、DurationDays>0；若管理端表单漏校验可创建 0 元或负数天数套餐，订单/订阅计算直接受污染。
- **建议**: 在套餐创建/更新入口统一校验，或在模型加 BeforeSave 钩子防御。

## internal/models/payment.go

### [HIGH] Amount 单位是「分」而 Order.Amount 是「元」，金钱核心单位不一致
- **位置**: 13-14  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: `Amount int // 金额（分）` 与 order.go 的 `Amount float64 decimal(10,2)`（元）并存：创建支付单时做 元→分 转换，回调核对时又要 分→元，任何一处除 100/乘 100 或四舍五入策略不同（如 0.1 元 → 10 分 vs 9 分）都会造成金额核对失败或入账偏差；Currency 默认 CNY 而 crypto 钱包支付仍记 CNY 币种。
- **建议**: 全项目统一整数分，Order.Amount 也改 int64 分；Currency 按支付方式实际币种写入，不允许默认硬编码。

### [MEDIUM] PaymentCallback 无幂等键，CallbackData 为 not null 字符串
- **位置**: 31-41  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 回调表没有 (payment_transaction_id, callback_type) 唯一约束，网关重试回调时若服务层先查后插会重复落回调记录、重复处理；`CallbackData string gorm:"type:json;not null"` 在回调体为空时直接插入失败，且与全库 NullString 风格不一致。
- **建议**: 加 (payment_transaction_id, callback_type) 唯一索引并 upsert；CallbackData 改 `sql.NullString type:json`，处理逻辑以 transaction 状态机 + 唯一键保证幂等。

## internal/models/payment_config.go

### [HIGH] 支付密钥字段带 json tag 且无 json:"-"，管理端接口原样回传密钥
- **位置**: 12-25, 951-989  |  **类别**: security  |  **来源组**: A2-models (全部模型)
- **问题**: MerchantPrivateKey/WechatAPIKey/PaypalSecret/StripeSecretKey/WalletAddress 等字段全部带 `json:"merchant_private_key,omitempty"` 之类标签；admin.go:951 的 GetPaymentConfig 直接把整行模型映射进 PaymentConfigResponse（admin.go:978-989 同名字段逐项复制）返回前端——商户私钥、Stripe Secret 进入浏览器可被 XSS/调试工具直接读取。
- **建议**: 模型层密钥字段全部 `json:"-"`（或单独建不含密钥的响应 DTO，脱敏只回显掩码）；管理端如需回显用独立结构体只带非敏感字段，密钥只能写不能读。

### [MEDIUM] GetConfig() 全项目无调用方，60 行密钥映射逻辑为死代码且 yipay/codepay 分支复制粘贴
- **位置**: 39-108  |  **类别**: maintainability  |  **来源组**: A2-models (全部模型)
- **问题**: grep 全 internal 目录确认 `PaymentConfig.GetConfig()` 零调用（支付服务各自直读字段，如 services/payment/alipay.go:24 直接取 p.MerchantPrivateKey）；该函数把 merchant_private_key/api_key/paypal_secret/stripe_secret_key 全部塞进返回 map，一旦未来被某个「获取支付配置」接口复用就是密钥泄露事故；且 yipay 与 codepay 两个 case 体完全相同。
- **建议**: 删除该函数（及其引入的 encoding/json 依赖）；若确需配置聚合，仅输出非敏感字段并显式白名单。

### [LOW] ConfigJSON 无条件合并进配置 map，任意 key 可覆盖类型化字段
- **位置**: 98-105  |  **类别**: security  |  **来源组**: A2-models (全部模型)
- **问题**: `for k, v := range jsonData { config[k] = v }` 允许自由 JSON 覆盖 pay_type/status 等字段且无白名单；ConfigJSON 本身明文存库。
- **建议**: 配置合并改为显式 key 白名单或独立命名空间（config_json.*），并对敏感值加密存储。

## internal/models/promotion.go

### [LOW] 活动缺少总量/每人限次字段，参与控制完全依赖参与表
- **位置**: 8-23  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: Promotion 无 TotalQuantity、PerUserLimit、CurrentParticipants，配合 PromotionParticipation 唯一索引，活动风控（限量抢购、每人一次）无法在模型层表达，只能散落在下单服务里临时判断。
- **建议**: 活动表增加总量/每人限次/已参与数字段，下单时用事务 + 条件更新（WHERE used < total）原子扣减。

### [LOW] Type/DiscountType 纯字符串 + 注释枚举，无类型常量
- **位置**: 11-12  |  **类别**: maintainability  |  **来源组**: A2-models (全部模型)
- **问题**: promotion.go 用裸字符串（'flash_sale'/'percentage'…）加行尾注释，而 ticket.go/coupon.go 已用 typed const 模式；新促销类型无编译期约束，服务层到处散落字符串比较。
- **建议**: 按 coupon.go 模式定义 `PromotionType`/`DiscountType` 常量并将字段类型改为常量类型。

## internal/models/promotion_participation.go

### [MEDIUM] Promotion/User 外键 OnDelete:CASCADE 硬删会销毁参与审计记录
- **位置**: 22-23  |  **类别**: architecture  |  **来源组**: A2-models (全部模型)
- **问题**: 删除活动或用户时级联删除参与记录，营销对账、风控追溯所需的历史全部消失；全库其他模型（coupon、order 等）又不声明外键约束，外键策略在模型间不一致。
- **建议**: 参与记录改为 OnDelete:RESTRICT 或 SET NULL，配合活动软删除；统一全库外键策略（要么都声明约束，要么都不声明，靠应用层保证）。

### [MEDIUM] (promotion_id, user_id) 唯一索引会阻断同活动多订单参与
- **位置**: 11-12  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: `idx_promotion_participation_unique` 把每个用户在每个活动下的参与记录限为一条；若活动允许同一用户多笔订单分别享受（如多单闪购、每单立减），该索引直接写不进去——是「防重复领取」还是「误伤多单」，取决于服务层是否已另行去重，当前设计二义性明显。
- **建议**: 明确活动参与语义：一次性领取类活动保留该唯一索引；按订单参与类活动把唯一键改为 (promotion_id, order_id)。

## internal/models/recharge.go

### [MEDIUM] 充值记录无过期时间，pending 的扫码/转账订单永不失效
- **位置**: 12, 16-17  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: PaymentQRCode/PaymentURL 生成的待支付充值单没有 expire_time 字段，用户生成二维码后不支付，pending 行无限堆积（无清理任务也无过期判断）；金额 float64 同样是分/元换算的隐患点。
- **建议**: 增加 ExpireTime 列（如 15 分钟）并由调度任务把超时 pending 置为 expired；金额统一整数分。

## internal/models/security.go

### [HIGH] 验证码 Used 读改写非原子，并发校验可复用同一验证码
- **位置**: 40, 52-57  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: `Used int` + `MarkAsUsed()` 是典型的先查后写：两个并发校验请求都读到 Used=0 都通过 IsUsed()，然后各自 MarkAsUsed——同一验证码可被并发兑换两次（注册/改密场景 = 账号接管向量）；IsUsed 用 `Used == 1` 而字段是 int，语义也应为 bool。
- **建议**: 校验消费改为原子条件更新：`UPDATE verification_codes SET used=1 WHERE id=? AND used=0 AND expires_at>NOW()`，用 RowsAffected 判断是否成功；字段类型改 bool。

### [LOW] 验证码表无 (email, purpose, expires_at) 复合索引且无过期清理
- **位置**: 34-42  |  **类别**: performance  |  **来源组**: A2-models (全部模型)
- **问题**: 校验查询 `WHERE email=? AND purpose=? AND used=0` 只有 email 单列索引，且过期验证码行无清理机制（同 token_blacklist 问题），表随时间增长。
- **建议**: 加 (email, purpose, expires_at) 复合索引；调度任务定期删除 expires_at < NOW() 的过期验证码与登录尝试记录。

## internal/models/subscription.go

### [LOW] DeviceLimit 无默认值 + Status/IsActive 双状态
- **位置**: 12, 16-17  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: `DeviceLimit int` 零值即 0（无限制?）与套餐/等级的默认 3 语义冲突，消费方需自行回退；Status(active/…) 与 IsActive 并存，与 node/custom_node 相同的双状态问题第三次出现。
- **建议**: DeviceLimit 统一走套餐快照并显式 default；Status/IsActive 收敛为单一枚举。

### [LOW] 订阅 URL 相关列长度不一致，注释笔误
- **位置**: 15, 38-39  |  **类别**: maintainability  |  **来源组**: A2-models (全部模型)
- **问题**: SubscriptionURL 是 varchar(100)，而 SubscriptionReset.Old/NewSubscriptionURL 是 varchar(255)——同一类数据两套长度，重置历史里超 100 字符的 URL 会与主表不一致；`// 猫咪订阅次数` 疑为「Clash 订阅次数」笔误。
- **建议**: 统一为 varchar(255) 并修正注释；顺带为 UniversalCount/ClashCount 补语义注释。

## internal/models/ticket.go

### [MEDIUM] 已读状态双机制并存：TicketReply.IsRead/ReadBy/ReadAt 与独立 TicketRead 表
- **位置**: 66-69, 100-110  |  **类别**: architecture  |  **来源组**: A2-models (全部模型)
- **问题**: 回复级已读用 IsRead/ReadBy/ReadAt 三字段，票单级已读又建了 TicketRead 表（handlers/ticket.go:94-128 两处都在读写）——同一概念两套实现，改一处漏一处会导致已读状态不同步。
- **建议**: 保留其一：回复级已读用 TicketReply 字段即可，票单级已读可并入同一模型（如 Ticket.LastReadAt/LastReadByUserID），删除 TicketRead 表；避免双轨。

### [LOW] Rating 无 1-5 范围约束；FilePath 存上传路径存在路径遍历前提
- **位置**: 44, 84-86  |  **类别**: security  |  **来源组**: A2-models (全部模型)
- **问题**: `Rating *int64` 模型层不校验 1-5，越界值直接入库污染评分统计；TicketAttachment.FilePath 由上传流程写入，若路径由用户文件名拼接（未在服务层重命名），下载接口按 FilePath 读文件即路径遍历面。
- **建议**: 保存前校验 Rating ∈ [1,5]；上传文件统一重命名为服务端生成的 UUID 文件名，FilePath 只存服务端相对路径，下载时做路径规范化检查（filepath.Clean + 前缀检查）。

## internal/models/token_blacklist.go

### [HIGH] IsTokenBlacklisted 对 DB 错误 fail-open：黑名单判定失败时放行已登出令牌
- **位置**: 35-45  |  **类别**: security  |  **来源组**: A2-models (全部模型)
- **问题**: err != nil 的两个分支都 `return false`（gorm.ErrRecordNotFound 与其它错误同路），DB 抖动/超时/连接池耗尽时查询失败会被当作「不在黑名单」，配合 middleware/auth.go:64-78 的 5 分钟缓存，已登出 token 在故障窗口内可继续通过认证——登出失效语义被静默破坏。
- **建议**: 区分错误：ErrRecordNotFound → false；其它错误 → 记日志并 fail-closed（返回 true 或直接 500，宁可误杀不可放行）。

### [MEDIUM] CleanExpiredTokens 全项目无调用方，黑名单表无界增长
- **位置**: 47-49  |  **类别**: performance  |  **来源组**: A2-models (全部模型)
- **问题**: grep 确认 CleanExpiredTokens 只有定义没有调用（登出时 AddToBlacklist 不断插行，auth.go:327/379/391 每次登出都写）；过期行永不清理，配合每次请求的黑名单查询（虽带 5 分钟缓存），表膨胀后查询与清理成本持续上升。
- **建议**: 在 scheduler 里注册每日任务调用 CleanExpiredTokens（按批次 LIMIT 删除防长事务），或对 ExpiresAt 分区/定期归档。

## internal/models/user.go

### [MEDIUM] User 是 god object：约 40 列 + 13 个关联切片混合安全、偏好、邀请、等级、推送多领域
- **位置**: 8-74  |  **类别**: architecture  |  **来源组**: A2-models (全部模型)
- **问题**: 同一结构体里既有认证安全字段（VerificationToken/ResetToken/Password）、又有通知偏好（NotificationTypes/SMS/Push/Bark）、邀请统计（TotalInviteCount/TotalInviteReward）、会员等级（UserLevelID/LevelExpiresAt/TotalConsumption）、特殊节点策略、Telegram 绑定、管理员备注——13 个 `[]X` 关联也全挂 User 上。任何一处 Preload 全关联或直接序列化都会顺带暴露无关领域数据，且每次改表都要动同一结构体。
- **建议**: 拆分为 User(核心认证) + UserProfile(昵称/头像/偏好) + UserStats(邀请/消费统计) + UserSecurity(令牌/密钥字段 json:"-") 等 has-one 结构，接口按需组装 DTO。

### [MEDIUM] Balance 浮点余额 + NotificationTypes 文本列存 JSON
- **位置**: 33, 41  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: `Balance float64 decimal(10,2)` 是充值/消费核心，浮点 + 读改写更新在并发充值时丢更新（且与 order/payment 的分/元混用叠加）；`NotificationTypes string type:text` 实际存 JSON（services/notification/notification.go:285 json.Unmarshal 解析），与全库 `type:json` 风格不一致。
- **建议**: Balance 改整数分并所有增减走 `UPDATE users SET balance = balance + ? WHERE ...` 原子表达式（服务层用事务 + 乐观锁）；NotificationTypes 改 `type:json` 或拆成独立偏好表。

### [MEDIUM] BarkDeviceKey 与管理员备注 Notes 暴露在 User 的 JSON 序列化中
- **位置**: 36, 59  |  **类别**: security  |  **来源组**: A2-models (全部模型)
- **问题**: `BarkDeviceKey` 和 `Notes`（管理员备注）都带 `json:"..."` 且未加 `json:"-"`：任何直接返回 models.User 的接口（如用户列表、个人资料聚合）都会把推送设备密钥和管理员内部备注发给客户端——备注泄露是信息泄露，Bark key 泄露可被他人向用户设备推送骚扰消息。
- **建议**: BarkDeviceKey 与 Notes 改 `json:"-"`，推送 key 通过专用设置接口读写（写不读），备注仅管理端专用 DTO 返回。

### [LOW] 邀请信息在 User 与 InviteRelation 双份冗余
- **位置**: 43-46  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: InvitedBy/InviteCodeUsed/TotalInviteCount/TotalInviteReward 与 invite_relations、commission_logs 数据可推导；计数若由多处累加（注册时 + 下单时）极易漂移，出现「关系表有条记录但 User 计数少 1」的对账差异。
- **建议**: 计数统一以 invite_relations/commission_logs 为准，User 上的冗余字段只读缓存并定时对账（或直接删除，查询时 JOIN/聚合）。

## internal/models/user_level.go

### [LOW] LevelOrder 唯一索引使等级重排需要多行协调更新
- **位置**: 12  |  **类别**: logic  |  **来源组**: A2-models (全部模型)
- **问题**: 调整等级顺序时任何一行改 order 都可能与既有值冲突，服务层需要整体平移或临时负值两步法，模型层无任何提示。
- **建议**: 排序改 SortOrder 非唯一 + 查询时 ORDER BY sort_order, id，避免重排时的唯一约束舞蹈。

### [LOW] UserLevelResponse/ToUserLevelResponse/MarshalJSON 与 coupon.go 样板重复
- **位置**: 30-71  |  **类别**: maintainability  |  **来源组**: A2-models (全部模型)
- **问题**: 与 Coupon 完全相同的「响应 DTO + 转换 + MarshalJSON 覆盖」三件套（字段逐项手抄），两处已在耦合演进（coupon 后续加字段容易漏改这边）。
- **建议**: 与 coupon.go 一起抽取公共的响应转换辅助；同时注意 MarshalJSON 覆盖后 Users 关联永远无法序列化，若未来有需要会踩坑，应显式注释说明。

## internal/services/backup_service/backup_service.go

### [HIGH] 远程备份默认指向硬编码作者仓库 moneyfly/backup、moneyfly1/backup
- **位置**: 36-117  |  **类别**: security  |  **来源组**: A9-services-rest (其余 services)
- **问题**: DefaultPlatformConfig（100-117 行）与 LoadRemoteBackupConfig（47-52 行）在未配置 owner/repo 时硬编码 moneyfly（gitee）与 moneyfly1（github）——管理员只填 token 开启备份后，数据库备份会被推到他人私有仓库，属敏感数据外泄隐患。
- **建议**: 远程备份默认禁用；未显式配置 owner/repo 时拒绝启用并给出明确错误；去掉内置 owner 默认值。

### [LOW] LoadRemoteBackupConfig 与 LoadPlatformConfig 的 token/owner/repo 三段读取完全重复
- **位置**: 81-98,36-79  |  **类别**: duplication  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 两处各写一遍『查 token→查 owner→查 repo』（63-76 与 85-96 行），仅入参不同，可抽公共 helper。
- **建议**: 抽 loadRepoConfig(db, target) (token, owner, repo string) 公共函数。

### [LOW] BuildDBOnlyBackupZip 在数据库文件缺失时静默生成空 zip 并返回 nil 错误
- **位置**: 119-176  |  **类别**: error-handling  |  **来源组**: A9-services-rest (其余 services)
- **问题**: cboard.db 不存在时 if 块直接跳过（134-160 行），函数仍返回成功（含 0 字节 zip）——远程上传会把空包当有效备份推上去；且此路径未做 WAL checkpoint，依赖调用方先行执行。
- **建议**: DB 文件缺失时返回明确错误；在函数内先执行 WAL checkpoint 或文档化前置条件。

## internal/services/cache_service/cache_service.go

### [MEDIUM] ClearKnowledgeArticlesCache 无法清除按分类的缓存键
- **位置**: 266-277  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 分类文章缓存键为 knowledge:articles:category:<id>，Clear 只删 active 总键与 categories 键（269-274 行），注释自认『无法清除所有分类缓存』——文章变更后分类缓存最长 1 小时脏数据，且无定时兜底。
- **建议**: 缓存值携带分类列表或改用 Redis SCAN+DEL 前缀删除，或在写文章时按分类清除。

### [LOW] 统计数据缓存只有 Get/Set 没有 Clear 方法
- **位置**: 341-358  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: GetStatisticsCache/SetStatisticsCache（341-358 行）无对应清除入口，统计数据只能等 TTL 过期，数据更新后无法主动失效。
- **建议**: 补 ClearStatisticsCache(key) 并在统计变更处调用。

## internal/services/cache_service/flush.go

### [LOW] FlushAllCache 直接 FLUSHALL，会清空共享 Redis 实例上的所有键
- **位置**: 9-22  |  **类别**: security  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 若该 Redis 实例与其它应用/服务共用，FlushAllCache 的 FLUSHALL（15 行）将连带清除非本应用的键，属破坏性操作且无确认参数。
- **建议**: 改为按本项目键前缀批量删除（SCAN+DEL），或至少校验 Redis db index 为专用库后再 FLUSHALL。

## internal/services/cache_service/warmup.go

### [LOW] 预热字段映射与 handler 层查询重复，易漂移
- **位置**: 29-117  |  **类别**: duplication  |  **来源组**: A9-services-rest (其余 services)
- **问题**: warmupPackages/warmupAnnouncements 手工映射 map[string]interface{}（39-50、70-77 行），与套餐/公告列表 handler 的返回结构各自维护；公告预热还硬编码 Limit(10)，与业务查询条件可能不一致，导致预热数据与实时数据字段/条数不同。
- **建议**: 预热复用与 handler 相同的 DTO/Response 序列化函数，保证一致性。

### [LOW] WarmupCache 起 goroutine 后立即打印完成，日志与实际进度不符
- **位置**: 13-27  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 三个 warmup* 以 go 启动（22-24 行）后立刻 LogInfo『缓存预热: 完成』（26 行）——此时预热还在异步执行，日志误导运维；goroutine 内错误仅 log.Printf，无聚合反馈。
- **建议**: 用 WaitGroup 等待三个预热完成后再打『完成』日志。

## internal/services/config_update/cache.go

### [MEDIUM] 全局 cacheGen 代际导致任一缓存清理使所有用户订阅缓存全部失效
- **位置**: 119-141, 214-216  |  **类别**: performance  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: ClearCustomNodesCache / ClearAllSubscriptionCache / ClearSubscriptionConfigCache 都执行 cacheGen.Add(1)——任一用户的专线变更或任一订阅清理都会使全站所有 gen 标记条目（系统节点、所有订阅配置）代际失配而失效，多用户下缓存命中率被频繁打击；这是为防"删除前读、删除后写回"竞态而做的全局闸，粒度太粗。
- **建议**: 按缓存族拆分代际：system 节点、订阅配置、custom 节点各持独立 gen（或 Redis 内直接用删除+写入原子化），仅在真正全局清库时 bump 全局 gen。

### [LOW] GetCustomNodesCache/SetCustomNodesCache 无调用方，专线节点每次实时查库
- **位置**: 82-116  |  **类别**: maintainability  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: grep 确认 GetCustomNodesCache/SetCustomNodesCache 两个方法全库无引用（只有 ClearCustomNodesCache 在 custom_node.go:1315 被调用）；appendCustomNodes 每次都执行 Joins 联表查询——专线节点多的用户在每次订阅拉取时重复查询且无缓存，方法对是死代码，也是错过的性能机会。
- **建议**: 在 appendCustomNodes 中接入 Get/SetCustomNodesCache（沿用代际机制），或删除该对方法避免误导。

### [INFO] #nosec G117 引用了不存在的 gosec 规则号
- **位置**: 71-72, 108-109, 203-204  |  **类别**: style  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: gosec 规则编号中不存在 G117（常见的凭据规则是 G101/G401 等），这些注释无法真正抑制任何扫描器告警，属于误导性注释；同时把节点密码/配置写入 Redis JSON 本就该评估是否需加密存储。
- **建议**: 替换为真实的 gosec 规则号（如 #nosec G101 - 代理密码非用户凭据），或直接删除注释并确保扫描配置豁免该文件。

## internal/services/config_update/config_update.go

### [HIGH] CreateInBatches 入库错误被静默吞掉，事务照常提交
- **位置**: 1092-1096  |  **类别**: error-handling  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `if err := db.CreateInBatches(newNodes, 100).Error; err == nil { stats.Created = len(newNodes) }`——err 非 nil 时不返回错误、不记日志，函数返回 stats（Created 保持 0），外层事务随后正常 commit，日志仍打印"入库完成 => 新增: 0"式的成功信息。批量插入失败时节点全部静默丢失，且无任何告警。
- **建议**: err 非 nil 时 `return stats, fmt.Errorf("批量入库失败: %w", err)` 向上传播使事务回滚，并在 errorf 中记录失败详情；至少也要 warnf 记录错误。

### [HIGH] 事务先删后查导致 importNodesToDatabaseWithOrderTx 的更新分支永远为空
- **位置**: 422-431, 1027-1098  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: RunUpdateTask 在事务内先 `tx.Where("is_manual = ?", false).Delete(&models.Node{})` 删光全部自动节点，随后才调用 importNodesToDatabaseWithOrderTx；而后者第一件事是 `db.Where("is_manual = ?", false).Find(&existing)` 重新查询这些已被删除的节点——同一事务内必然查不到。因此 existingMap 恒空、duplicateExistingIDs 恒空、`stats.Updated` 恒为 0，终态日志"更新: %d"永远显示 0，且 1036-1049 行的"清理重复采集节点"与 1074-1079 行的更新分支均为死逻辑（importNodesToDatabaseWithOrder 直连 s.db 的版本同样如此）。
- **建议**: 二选一：1) 删除事务内的预删除，让 import 函数自己按 key 做 upsert（保留 existing 分支）；2) 保留预删除但删除整个 existing 查询/更新/重复清理逻辑，并把最终日志中的"更新"字段去掉，避免误导。推荐方案 1，同时去掉 1031-1049 的全表 Find（大数据量下是整表加载）。

### [HIGH] generateClashYAML 直接改写共享缓存节点指针的 Name/Type，造成数据竞争与跨请求输出漂移
- **位置**: 1212-1245, 1374-1395  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: appendSystemNodes 从 Redis 系统节点缓存取出的 `[]*ProxyNode` 与缓存内是同一批指针（SetSystemNodesCache 存的就是这些指针）；generateClashYAML 里 `p.Type = "socks5"`（1381 行）与名字去重 `p.Name = name`（1388 行）直接修改共享对象。后果：1) 并发 clash 订阅请求对同一批指针并发写→数据竞争；2) 顺序性污染——一次 clash 请求把 socks 永久改成 socks5 后，后续 base64 通用订阅输出从 GOST `socks://` 变成普通 `socks5://`（nodeToLink 对二者生成不同链接），且名字后缀 _N 残留。缓存代际机制只能防"删除后写回"，防不了对已缓存对象的就地改写。
- **建议**: generateClashYAML 在改名/改型前对每个节点做深拷贝（新 ProxyNode + 复制 Options map），或让 appendSystemNodes 每次从缓存值复制出一份独立节点列表再返回；ParseCache 同理（见 parser.go 发现）。

### [MEDIUM] ParseLinks 完成顺序乱序导致 orderIndex 每次更新随机漂移
- **位置**: 531-558  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `results := s.parserPool.ParseLinks(links)` 的结果来自 channel 的完成顺序（见 parser.go 95-100），`for idx, result := range results` 中的 idx 并非输入链接下标，但 `orderIndex: urlIndex*10000 + idx` 用它作为节点排序依据——同一批订阅源每次运行节点先后顺序都可能不同，订阅列表顺序不稳定，且与 processFetchedNodes 里的去重顺序联动。
- **建议**: ParseResult 增加输入下标字段（或按 links 下标重建有序结果），orderIndex 用输入下标计算；也可在 ParseLinks 返回前按输入顺序重排。

### [MEDIUM] DeviceLimit==0 被判定为设备超限，语义存疑
- **位置**: 1146-1149  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `if sub.DeviceLimit == 0 { ctx.Status = StatusDeviceOverLimit; return ctx }`——把 0 解释为"不允许任何设备"。若数据库该字段默认值或后台创建订阅时未显式赋值（0），所有订阅将直接返回设备超限错误节点，用户全部不可用；而 1151 行又用 `> 0` 判断，两处对 0 的语义不一致。
- **建议**: 确认 DeviceLimit=0 的业务含义：若为"不限设备"应改为跳过限制；若确实为"禁连"，在创建订阅的后端接口处显式校验该字段不能为 0，避免误配导致全站订阅不可用。

### [MEDIUM] 每次订阅请求都触发 refreshSystemConfig 两次额外 DB 查询
- **位置**: 1267, 1300, 2355  |  **类别**: performance  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: GenerateClashConfig / GenerateUniversalConfig / generateClientConfig 每次被客户端拉取都调用 s.refreshSystemConfig()，其中 GetDomainFromDB 与 support_qq 各一次 DB 查询；订阅端点高频轮询（客户端 10 分钟 TTL 缓存过期即重拉）下，这两个查询完全没有缓存。
- **建议**: 为 siteURL/supportQQ 增加短 TTL（如 60s）内存缓存，仅系统配置变更时失效；或在 GetSubscriptionContext 校验通过路径上按需刷新。

### [LOW] deleteAutoImportedNodes / importNodesToDatabaseWithOrder 为死代码，GetStatus 的 next_update 恒空
- **位置**: 982-989, 1023-1025, 247  |  **类别**: maintainability  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: grep 全库确认 deleteAutoImportedNodes（982 行，与 RunUpdateTask 事务内删除逻辑重复）与 importNodesToDatabaseWithOrder（1023 行，非事务版本）均无调用方；GetStatus 返回的 next_update 恒为 ""，前端若展示会显示空值。
- **建议**: 删除两个无调用方法；GetStatus 要么实现调度器下次运行时间，要么去掉该字段并同步前端契约。

### [LOW] 订阅源抓取无 scheme/内网地址校验（管理员级 SSRF）
- **位置**: 643-714  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: FetchNodesFromURLs/fetchURLContent 直接对配置的 urlStr 发起 GET，无 scheme 白名单（http.NewRequest 会拒绝非 http 协议，但可指向 127.0.0.1/内网 IP），且 http.Client 默认跟随重定向。虽仅管理员可配置，但若配置面板被低权限角色接触或配置导入来源不可信，存在内网探测风险。
- **建议**: 发起请求前解析并校验 scheme ∈ {http, https}，可选地拒绝回环/内网地址（或提供显式开关允许内网源）；限制重定向次数并校验最终跳转地址。

### [LOW] 日志级别失真与一处可疑死分支
- **位置**: 302-323, 872  |  **类别**: style  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: log() 中仅 ERROR 走 AppLogger.Error，WARN/SUCCESS/DEBUG 全部降级为 AppLogger.Info，运维侧无法按级别过滤；extractNodeLinks 中 `(strings.HasPrefix(content[start:end], "ss://") && start >= 3 && content[start-3:start] == "vme")`（872 行）这条防御分支无法从正则 `(?:^|\s)` 的约束下被触发（"vme" 前不是空白/开头），疑似死分支，增加维护负担。
- **建议**: log() 按 level 映射到 AppLogger 对应级别；872 行补一条注释说明触发场景或直接删除，避免后人误读。

## internal/services/config_update/node_parser.go

### [MEDIUM] parseSSR 端口/密码解析错误被静默吞掉
- **位置**: 266-304  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `port, _ := strconv.Atoi(mainParts[l-5])` 与 `password, _ := DecodeBase64(mainParts[l-1])` 均忽略错误：端口非数字时为 0，密码 base64 解码失败时为空串，且无任何错误返回——格式错误的 SSR 链接会生成 Port=0、无密码的节点入库。对比 parseVMess 有 `port <= 0 || port > 65535` 校验，SSR 完全没有。
- **建议**: 校验 port ∈ (0, 65535]，password 解码失败时返回错误（"SSR 密码解码失败"），让坏链接计入 parseFailed 统计而非进入库。

### [MEDIUM] getPort 对缺失端口的链接静默默认 443/8388
- **位置**: 903-912  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: 链接无端口时 ss/ssr 默认 8388、其余全部默认 443——wireguard/tuic/socks 等协议无端口时得到无意义的 443，且完全不报错，坏链接被当作有效节点入库（与 isValidNodeLink 的宽松校验叠加）。
- **建议**: 对无端口且非默认端口的协议返回解析错误，或至少在校验层（isValidNodeLink / ParseNodeLink）拒绝 Port<=0 的节点。

### [LOW] SOCKS/HTTP 用户名存入 UUID 字段的隐式约定
- **位置**: 487-500, 1837-1840  |  **类别**: architecture  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: extractAuthToNode(preferPwd=false) 把 userinfo 用户名写进 n.UUID，nodeToMap 的 socks/http 分支再 `res["username"] = n.UUID` 取出——跨文件依赖"UUID 字段兼职存用户名"的隐式约定，且 nodeToLink 的 socks 分支 `buildStandardNodeURL(sc, n.UUID, n.Password, ...)` 恰好依赖它。约定断裂（如未来某解析器把真实 UUID 存入该字段）会产生难排查的错误链接。
- **建议**: 为 ProxyNode 增加显式 Username 字段，解析器按语义写入，nodeToMap/nodeToLink 直接读 Username；或在该约定处加注释并集中封装 getter/setter。

### [LOW] 字节截断可能切断 UTF-8（错误信息与日志预览）
- **位置**: 78-81, 103-108  |  **类别**: style  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: ParseNodeLink 的 `link[:10]` 与 truncateLink 的 `link[:maxLen]` 均按字节截断，若链接/名称含多字节字符（订阅节点名常见 emoji/中文），截断点可能落在 UTF-8 序列中间，产生乱码日志与不完整错误信息。
- **建议**: 按 rune 截断：`string([]rune(s)[:min(len([]rune(s)), n)])`，或使用 utf8 安全截断函数（全库统一）。

## internal/services/config_update/parser.go

### [HIGH] ParseCache 缓存 *ProxyNode 指针被调用方就地改写，跨请求污染
- **位置**: 72, 129-146  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: p.cache.Set(link, node) 缓存的是解析结果指针；processFetchedNodes 随后 `result.Node.Name = s.ensureUniqueName(...)` 直接改名字，generateClashYAML 又改 Name/Type——这些改动全部落到 5 分钟 TTL 的共享缓存对象上。下一次运行/下一次订阅生成读到的节点是上次被改名/改型后的残留状态，且并发运行时是数据竞争。与 config_update.go 的系统节点缓存问题同源。
- **建议**: ParseCache.Get 返回深拷贝（clone ProxyNode 及 Options），或明确约定缓存对象只读、写入路径统一 copy-on-write；两者选一并在代码注释中声明。

### [MEDIUM] convertClashProxyToLink 只支持 5 种协议，Clash 订阅中其他节点被静默丢弃
- **位置**: 247-261  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: switch 只处理 vmess/vless/trojan/ss/shadowsocks/hysteria2，hysteria、tuic、wireguard、socks、http 等类型走 default 只打 DEBUG 日志后丢弃；而 parseClashYAML 只要 YAML 含 proxies 键就整体接管解析，混合订阅中的非支持节点（可能占相当比例）全部丢失且仅 DEBUG 可见。
- **建议**: default 分支对不支持类型在转换失败统计中计数（failedCount），并最终在 WARN 日志汇总；或直接复用 nodeToMap 反向构建链接，避免协议白名单维护两份。

### [MEDIUM] 缓存过期时每个 key 启动一个 goroutine 删除
- **位置**: 138-142  |  **类别**: performance  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: Get 命中过期条目时 `go c.delete(key)`——高流量下过期条目成批触发时会产生大量瞬时 goroutine（每个过期 key 一个），且删除与后续 Set 竞争。
- **建议**: Get 内直接调用 c.delete(key)（RLock 内不能加写锁，可改为先收集过期 key 再统一删除），或仅标记过期、由 cleanup 批量回收。

## internal/services/config_update/region.go

### [MEDIUM] 同长度关键词排序不稳定导致地区匹配结果不确定
- **位置**: 91-93, 102-122  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `sort.Slice(rm.regionKeywords, func(i, j int) bool { return rm.regionKeywords[i].length > rm.regionKeywords[j].length })` 对相同长度的关键词（如"美国"vs"日本"、"香港"vs"澳门"）不保证顺序，而 map 迭代顺序在 Go 中是随机的——同一节点名在多次匹配中可能命中不同地区（如"美国日本专线"），结果非确定。
- **建议**: 改用 sort.SliceStable，并在长度相同时按 keyword 字典序作为次级排序键，保证确定性输出。

### [LOW] UpdateMaps 无调用方；LoadRegionConfig 的错误分支为死分支
- **位置**: 124-145, 25-44  |  **类别**: maintainability  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: grep 确认 RegionMatcher.UpdateMaps 全库无调用（管理员更新地区配置的入口未接入，region 配置只能靠重启时加载）；LoadRegionConfig 中 `len(builtInRegionMap)==0 && len(builtInServerMap)==0` 恒为假（region_maps.go 恒非空），getDefaultRegionConfig 分支不可达。
- **建议**: 若管理员配置地区映射的 API 存在，接入 UpdateMaps 并刷新 regionMatcher；否则删除 UpdateMaps 与 getDefaultRegionConfig 死分支。

## internal/services/config_update/region_maps.go

### [INFO] 单字/单字符关键词存在固有误判（如"澳"→澳大利亚、"德"→德国）
- **位置**: 3-242  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: builtInRegionMap 含"澳""港""日""美""德"等单字 key：节点名"澳门"会命中"澳"→澳大利亚，"德克萨斯"命中"德"→德国。靠 region.go 的长度降序排序缓解（长 key 优先），但排序不稳定（见 region.go 发现），结果不确定。属启发式匹配的固有取舍，数据本身无逻辑错误。
- **建议**: 删除明显歧义的单字 key（或降权），并为"澳门""德克萨斯"等常见多字词补充显式条目；在文档中说明匹配规则为最长前缀优先。

## internal/services/config_update/sse_manager.go

### [LOW] RemoveClient 关闭 channel，若调用方同时 close 会 double-close panic
- **位置**: 39-46  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: RemoveClient 在锁内 close(ch)；SSE handler（本目录之外）通常在 defer 中调用 RemoveClient，若 handler 自身也 close 了 ch（如超时清理路径），第二次 close 会 panic 并可能拖垮整个服务（此文件无法确认调用方，需人工核对 subscription.go 的 SSE 端点）。
- **建议**: 约定 channel 所有权归 SSEManager：RemoveClient 用 sync.Once 或内部标记保证只 close 一次；handler 只负责调用 RemoveClient 不自行 close。

### [INFO] 广播/历史实现整体健康
- **位置**: 49-86, 89-97  |  **类别**: performance  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: broadcastToClients 使用 select-default 非阻塞投递，慢客户端直接跳过，避免写阻塞；GetHistoryLogs 返回浅拷贝副本，条目 map 创建后不再被修改，无并发安全问题；通道满丢弃策略合理。无明显问题。
- **建议**: 无需修改；若担心条目后续被写可改为深拷贝（当前无此风险）。

## internal/services/config_update/transport_opts.go

### [LOW] 整个 transport_opts.go 无任何引用（死代码），ToMap 与 nodeToMap 重复
- **位置**: 1-99  |  **类别**: maintainability  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: grep 确认 TransportOpts/WSOpts/GRPCOpts/H2Opts/RealityOpts/ToMap 全库无引用；ToMap 的字段拼装逻辑与 config_update.go 的 nodeToMap/Options 拷贝逻辑高度重复，若未来有人误用会造成两套行为不一致。
- **建议**: 删除该文件，或将其作为 nodeToMap 的重构目标（用强类型 TransportOpts 替代 map[string]any 的 Options），二者择一。

## internal/services/device/device_manager.go

### [MEDIUM] 设备指纹不含 IP/订阅维度，相同 UA 的不同物理设备被判为同一台
- **位置**: 848-885,177-208  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: GenerateDeviceHash 在 deviceID 为空时只用 UA 解析出的软件/OS/型号特征（ipAddress 参数被忽略，878-884 行），FindExistingDevice 按 subscription_id+device_hash 命中——同一订阅下两台 UA 相同的真实设备会被当成同一台，AccessCount 与设备计数失真，设备上限控制被绕过（第二台不占额度）。
- **建议**: 指纹中加入可区分的稳定输入（如 IP+UA），或对该冲突场景显式处理并在设备上限校验中体现。

### [MEDIUM] 查找-创建非原子，并发访问可产生重复设备行
- **位置**: 177-208,1013-1019  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: RecordDeviceAccess 先 FindExistingDevice 再 Create，两步之间无锁/唯一约束（964-1015 行）；并发首次拉取时各自查不到，各创建一行，current_devices 重复累加（1017-1019 行 Count 后 Update 且错误被忽略）。
- **建议**: 为 devices 表加 (subscription_id, device_hash) 唯一索引，Create 冲突时回退查询更新；或对同订阅的 RecordDeviceAccess 加锁。

### [MEDIUM] UA 解析热路径内大量 regexp.MustCompile 每次调用重新编译
- **位置**: 42-92,246-643  |  **类别**: performance  |  **来源组**: A9-services-rest (其余 services)
- **问题**: matchSoftware（253 行）、parseOSInfo（350/358/385/400/409/418 行）、parseDeviceInfo（516 行等）、parseVersion（637 行）等函数体内每次调用都 regexp.MustCompile；RecordDeviceAccess 每次订阅访问都走完整 ParseUserAgent，GenerateDeviceHash 又触发一次完整解析，属高频路径。
- **建议**: 所有正则提升为包级 var；GenerateDeviceHash 复用 ParseUserAgent 结果避免二次解析。

### [LOW] current_devices 重算的 Count/Update 错误被静默忽略
- **位置**: 238-240,1017-1019  |  **类别**: error-handling  |  **来源组**: A9-services-rest (其余 services)
- **问题**: dm.db.Model(&models.Device{})...Count(&deviceCount) 与 Update("current_devices", deviceCount)（238-240、1017-1019 行）均未检查错误，DB 故障时设备数静默失准。
- **建议**: 检查并记录错误，失败时记日志并考虑告警。

### [LOW] ipAddress 为空时仍写入空字符串指针
- **位置**: 887-901  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: updateExistingDeviceAccess 无条件 device.IPAddress = &ipAddress（889 行），ipAddress=="" 时把空串覆盖进已有设备，丢失原 IP。
- **建议**: 仅在 ipAddress != "" 时更新 IPAddress。

## internal/services/discount/coupon.go

### [MEDIUM] ValidateCoupon 中每用户使用次数 Count 错误被忽略，查询失败时绕过限额
- **位置**: 146-152  |  **类别**: error-handling  |  **来源组**: A9-services-rest (其余 services)
- **问题**: db.Model(&models.CouponUsage{})...Count(&usageCount)（148 行）未检查 err——DB 异常时 usageCount 为 0，MaxUsesPerUser 限额被静默绕过（ReserveCouponUsageTx 内已正确处理，此处未处理）。
- **建议**: Count 出错时按『已达上限』处理（fail-closed）或返回错误。

### [LOW] 优惠券最低消费门槛基于等级折扣后的金额判断
- **位置**: 82-88,125-154  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: CreateOrder 调用 QuoteCouponForPreparedAmount 传入的 amountBeforeCoupon 是等级折扣后的 finalAmount（order.go:292），MinAmount 判断（143 行）以折后金额为基数——若产品意图是原始订单金额门槛，此处为策略偏差，需与产品确认。
- **建议**: 明确门槛基数（原始金额 vs 折后金额）并在 Quote 参数中显式传入，避免隐式语义。

## internal/services/email/email.go

### [MEDIUM] SendVerificationEmail 直发+入队双通道存在重复发送与错误标记竞态
- **位置**: 330-370  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 函数先 SendEmail 直发，再 QueueEmail 入队；直发成功后用 db.Where(to_email+subject+type+status).Order(created_at DESC).First() 把最新一条 pending 同主题邮件标记为 sent（355-363 行）。与每分钟运行的 ProcessEmailQueue 存在竞态窗口，且同一收件人同一 subject 多次请求时 First() 可能命中非本次入队记录，其余 pending 记录后续被队列补发，导致用户收到重复验证码邮件。
- **建议**: 二选一：只直发（失败才入队）或只入队由队列统一发送；若保留双通道，让 QueueEmail 返回记录 ID 并按 ID 精确标记，同时加去重约束。

### [LOW] 端口解析用 Sscanf 前缀匹配且 DB 查询不检查错误
- **位置**: 60-66,103-119  |  **类别**: error-handling  |  **来源组**: A9-services-rest (其余 services)
- **问题**: fmt.Sscanf(pStr,"%d",&port) 对 "587abc" 等非法串会前缀匹配成功；getEmailConfigFromDB 中 db.Where(...).Find() 未检查 db 为 nil（database.GetDB() 返回 nil 时 panic）也未检查查询错误。
- **建议**: 用 strconv.Atoi 严格解析；对 db==nil 与查询错误做防御处理。

### [LOW] tls 字段与 SMTPTLS 回退分支是死代码
- **位置**: 28,41-57  |  **类别**: maintainability  |  **来源组**: A9-services-rest (其余 services)
- **问题**: encryption := "tls"; if cfg.SMTPTLS { encryption = "tls" }（43-46 行）两个分支结果相同；结构体字段 tls（28 行）两个构造器都赋值，但 SendEmail 实际只按 s.encryption 分支，s.tls 从未被读取。
- **建议**: 删除 tls 字段与无意义 if；按 cfg.SMTPTLS 直接决定 encryption，或删除回退分支。

## internal/services/email/template.go

### [MEDIUM] EmailTemplateService.db 字段死代码，GetTemplate 绕过它直连 database.GetDB()
- **位置**: 14-30  |  **类别**: maintainability  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 结构体声明 db *gorm.DB 并在构造器赋值，但 GetTemplate 内部直接 database.GetDB()（26 行）——字段从未使用，依赖注入无效，测试无法注入 mock DB。
- **建议**: GetTemplate 改用 s.db，或删除 db 字段并在构造器断言 database.GetDB() 非 nil。

### [LOW] RenderTemplate 每次调用都重新编译正则
- **位置**: 36  |  **类别**: performance  |  **来源组**: A9-services-rest (其余 services)
- **问题**: regexp.MustCompile 位于函数体内（36 行），每渲染一封邮件都重新编译一次，覆盖验证码/密码重置/订阅等高频场景。
- **建议**: 提升为包级 var，包初始化时编译一次。

## internal/services/email/templates.go

### [HIGH] 邮件 HTML 注入/XSS：模板内容以 template.HTML 直插且大量用户数据未转义
- **位置**: 128-147,736-1079  |  **类别**: security  |  **来源组**: A9-services-rest (其余 services)
- **问题**: GetBaseTemplate 以 template.HTML(content) 直插内容（136 行）；GetAdminNotificationTemplate 工单分支用 fmt.Sprintf 直接拼接 ticketNo/title/type/priority 等用户可控字段（1017-1044 行），GetWelcomeTemplate/GetUserCreatedTemplate 拼 username/email 同样不转义；而 GetAdminReplyNotificationTemplate（1092-1148 行）却对同类字段做了 HTMLEscapeString——转义策略严重不一致，存在向管理员/用户邮件注入 HTML 的存储型 XSS 风险。
- **建议**: 统一约定：所有用户可控字段先 template.HTMLEscapeString（或改用 html/template 字段级自动转义）再进入 fmt.Sprintf，不依赖调用方自觉。

### [MEDIUM] GetBaseURL 兜底返回开发地址 http://localhost:5173
- **位置**: 109-126  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: DB domain、BASE_URL 环境变量、config.BaseURL 均缺失时返回 "http://localhost:5173"（125 行）——生产环境未配置域名时，密码重置/订阅/欢迎邮件里的链接全部指向本地开发地址，用户无法点击。
- **建议**: 生产模式无显式 baseURL 时应报错或由调用方显式传入，不要静默落到 localhost。

### [MEDIUM] GetSubscriptionTemplate 的 universalURL 参数从未被使用
- **位置**: 235-272  |  **类别**: maintainability  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 签名接收 universalURL 但函数体只构建 clashURL 的 url-list（238 行），调用方（internal/api/handlers/subscription.go:1265）仍构造并传入，属死参数+调用方无效工作。
- **建议**: 删除该参数并同步修改调用方，或补上 universal 订阅地址的展示。

### [LOW] GetAdminNotificationTemplate 内 order_created/order_paid 的升级详情构建整段复制粘贴
- **位置**: 751-830,334-377  |  **类别**: duplication  |  **来源组**: A9-services-rest (其余 services)
- **问题**: order_created/order_paid 两分支各约 40 行几乎相同的 upgradeSection 拼装（751-776 与 805-830 行），与 GetDeviceUpgradePaymentSuccessTemplate（334-377 行）逻辑亦重复，仅颜色/措辞不同。
- **建议**: 抽 buildUpgradeSectionHTML(data, color) 公共函数，三处调用。

## internal/services/geoip/cache.go

### [MEDIUM] WarmupCache 每个 IP 一个 goroutine，叠加内部写缓存 goroutine 形成风暴
- **位置**: 71-82  |  **类别**: performance  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 循环里为每个 IP go func(){ GetLocationWithCache(ip) }()（77-79 行），而 GetLocationWithCache 在 miss 时又异步再起一个写 Redis 的 goroutine（36-46 行）——预热 1000 个 IP 即 2000 个 goroutine 并发，10ms sleep 无法缓解压力。
- **建议**: 预热改为有界并发 worker pool（如 10），并让预热路径同步写缓存。

### [LOW] 失败结果以 NULL 哨兵缓存 24 小时
- **位置**: 85-127  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: GetLocationWithFallbackCached 在查询失败时也缓存 "NULL" 24h（113-118 行）——若某 IP 查询时第三方/DB 短暂故障，该 IP 位置信息 24 小时内恒为空，无法自动恢复。
- **建议**: 失败缓存 TTL 缩短（如 10 分钟，与 Ping0 内存缓存一致），成功才缓存 24h。

## internal/services/geoip/geoip.go

### [MEDIUM] GetLocationFromIPW 与 GetLocationFromPing0 高度重复，且高度依赖第三方网页结构
- **位置**: 461-613,615-714  |  **类别**: duplication  |  **来源组**: A9-services-rest (其余 services)
- **问题**: GetLocationFromIPW 与 GetLocationFromPing0 共享同一套『固定 UA 的 http.Get + 5s 超时 + 正则解析 HTML』骨架（约 150 行重复）；解析依赖 ping0.cc 第二行文本（lines[1]，669 行）与 ipw.cn 的字段名，chinaPattern 中 (?i)(?:china|中国|cn|...) 的 cn 子串会误匹配 cnn.com 等文本，第三方改版即失效。
- **建议**: 抽公共 httpFetch 骨架；优先走双方公开 JSON API 或本地 MMDB，避免脆弱的 HTML 正则。

### [LOW] GetLocationWithFallback 中剥离 ::ffff: 前缀的 ipAddress 变量从未被使用
- **位置**: 719-752  |  **类别**: maintainability  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 721-724 行把 ipAddress 改为去掉 ::ffff: 前缀，但后续全部调用 GetLocation(originalIP)/GetLocationFromPing0(originalIP)（内部自己剥离），剥离后的 ipAddress 是死变量；最终错误信息只保留最后一个 err，掩盖了首个方法的真实失败原因。
- **建议**: 删除死变量；用 errors.Join 聚合各方法失败原因后再返回。

## internal/services/git/git.go

### [HIGH] Gitee 的 access_token 拼进 URL query 串，token 会进入访问/代理日志
- **位置**: 484-490,549-555  |  **类别**: security  |  **来源组**: A9-services-rest (其余 services)
- **问题**: ListContents/DownloadFile 对 Gitee 把 token 附加为 ?access_token=xxx（486-489、552-555 行），同时请求头又设置 Authorization（497、562 行）——query 里的 token 冗余且危险：会被网关/反向代理/访问日志/浏览器历史留存。
- **建议**: Gitee 同样只走 Authorization: token xxx 头，删除 query 传参。

### [MEDIUM] TestConnection 对 Gitee 不附加 access_token，私有仓库连接测试必然失败
- **位置**: 373-399  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: TestConnection 的 GET /api/v5/repos/O/R 既无 Authorization 头也不（与 ListContents 不同）拼 access_token（373-399 行）——私有 Gitee 仓库即使 token 正确也会 401，连接测试与实际上传行为不一致。
- **建议**: TestConnection 复用与 ListContents 一致的鉴权逻辑。

### [MEDIUM] UploadStatusManager.CleanOldStatuses 无任何调用点，全局状态 map 只增不减
- **位置**: 413-452,635-652  |  **类别**: maintainability  |  **来源组**: A9-services-rest (其余 services)
- **问题**: grep 全仓库确认 CleanOldStatuses 只定义未调用（635-652 行）；全局 globalUploadStatusManager.statuses 由 backup handler 持续 SetStatus/UpdateStatus（backup.go:162/326）从不清理——长时间运行内存缓慢泄漏。
- **建议**: 在 scheduler 定时调用 CleanOldStatuses，或在 SetStatus 时顺带清理超阈值任务。

### [LOW] 上传重试路径不关闭失败响应的 Body，且每次调用新建 Transport
- **位置**: 249-295  |  **类别**: performance  |  **来源组**: A9-services-rest (其余 services)
- **问题**: client.Do 出错时若返回了 resp（Go 允许 err 与 resp 同时非 nil），循环 continue/return 均未关闭其 Body——每次失败 attempt 泄漏一个连接；同时每次 UploadFileWithProgress 新建整套 http.Transport（201-211 行）。
- **建议**: 循环内统一 if resp != nil { resp.Body.Close() }；Transport 提升为客户端级复用。

## internal/services/node_health/node_health.go

### [HIGH] 离线节点被置 is_active=false 后永远不再被健康检查覆盖，无法自动恢复
- **位置**: 272-288,290-328  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: UpdateNodeStatus 对 offline/timeout 置 is_active=false（282 行），而 CheckAllNodes 只查 is_active=true（292 行）——一次瞬时抖动（ping.pe 解析失败、网络抖动）即把节点永久移出检查队列；若 ping.pe 整体不可用，全部节点被一次清空且永不恢复，直到管理员手动激活。
- **建议**: 健康检查应覆盖全部节点（is_active 仅表达用户可见状态），或对 offline 节点连续多次失败确认后才置 false，并保留自动回测入口。

### [MEDIUM] testViaWebPage 两个分支都调用 testViaPingPe，配置的 test_url 实际被忽略
- **位置**: 129-137,139-170  |  **类别**: maintainability  |  **来源组**: A9-services-rest (其余 services)
- **问题**: if strings.Contains(s.testURL, "ping.pe") { return s.testViaPingPe(...) }; return s.testViaPingPe(...)（132-136 行）——无论 test_url 配置成什么都走 ping.pe，配置项形同虚设（死分支）。
- **建议**: 实现非 ping.pe 的通用网页测速逻辑，或删除该分支与误导性配置项。

### [LOW] TestNodeWithContext 超时后底层 goroutine 仍继续执行
- **位置**: 357-388  |  **类别**: performance  |  **来源组**: A9-services-rest (其余 services)
- **问题**: ctx.Done 返回『测试超时』后，go func 里的 TestNode（30s HTTP）仍在后台跑完，结果被丢弃（361-387 行）——每次超时产生短暂存活的 goroutine，且 ctx 未传播到 HTTP 请求层，无法真正取消。
- **建议**: 把 ctx 传入 testViaPingPe 的 http.NewRequestWithContext，实现真正可取消。

## internal/services/notification/notification.go

### [HIGH] doJSONPost 使用无超时的默认 http.Client 且被 goroutine 调用，存在永久泄漏
- **位置**: 490-515,397-446  |  **类别**: performance  |  **来源组**: A9-services-rest (其余 services)
- **问题**: http.Post 使用默认 client（无 Timeout）；notifyTelegram/notifyBark 又各自 go func 调用。当 Telegram 或 Bark 自定义 server_url 无响应/挂起时，连接与 goroutine 永久不回收；Bark server_url 可配置，等同无超时出站请求面。
- **建议**: doJSONPost 改用带超时（如 10s）的共享 http.Client，并给 goroutine 内调用加超时与发送上限。

### [MEDIUM] SendAdminNotification 恒返回 nil，错误全被吞掉
- **位置**: 370-395  |  **类别**: architecture  |  **来源组**: A9-services-rest (其余 services)
- **问题**: notifyTelegram/notifyBark/notifyEmail 全部只写日志不返回错误，SendAdminNotification 无论何种失败都返回 nil（370-395 行）——调用方（调度器、订单流程）无法感知管理员通知失败，错误契约形同虚设。
- **建议**: 让三个 notify* 返回 error 并聚合，SendAdminNotification 返回聚合错误；至少全渠道失败时显式返回错误。

### [LOW] getString/getFloat/getInt 与 email 包 getStringFromData/getFloatFromData 完全重复
- **位置**: 560-589,1150-1166  |  **类别**: duplication  |  **来源组**: A9-services-rest (其余 services)
- **问题**: notification 包与 email/templates.go 各自实现一份『map 取值带默认值』工具（getString/getFloat/getInt 与 getStringFromData/getFloatFromData），函数体逐字相同，应上移公共 utils 包。
- **建议**: 在 internal/utils 提供 GetMapString/GetMapFloat/GetMapInt，两包共用。

### [LOW] Telegram 消息 parse_mode=HTML 且内容未转义用户数据
- **位置**: 517-551  |  **类别**: security  |  **来源组**: A9-services-rest (其余 services)
- **问题**: sendTelegramMessage 以 parse_mode="HTML" 发送（523 行），消息由 template.go 各 builder 用 fmt.Sprintf 直接插入 username/email/工单标题等用户数据，用户可借工单标题注入 Telegram 富文本内容。
- **建议**: 发送前对消息体做 HTML 转义（& < > " '），或改用 parse_mode="MarkdownV2" 并转义保留字。

## internal/services/notification/template.go

### [MEDIUM] Telegram 与 Bark 两套 builder 逐行重复约 700 行
- **位置**: 14-973  |  **类别**: duplication  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 14 组事件×2 渠道各自实现 buildXxxTelegram 与 buildXxxBark，数据抽取与盒线样式完全相同（如 buildUserRegisteredTelegram 212-231 与 buildUserRegisteredBark 594-614）；buildUpgradeDetailsTelegram/buildUpgradeDetailsBark 更是逐字重复。
- **建议**: 改为统一数据提取+单一格式化函数（渠道参数化），按渠道输出带/不带 HTML 标签的模板。

### [LOW] buildOrderPaidTelegram 末尾 fmt.Sprintf 无任何占位符
- **位置**: 148-154  |  **类别**: maintainability  |  **来源组**: A9-services-rest (其余 services)
- **问题**: msg += fmt.Sprintf(...)（148-154 行）字符串里没有任何 % 动词，fmt.Sprintf 纯属冗余（buildOrderPaidBark 中同样存在）。
- **建议**: 直接字符串拼接，去掉 fmt.Sprintf。

## internal/services/order/order.go

### [CRITICAL] updateUserLevelTx 的等级选择与升降级判定方向疑似整体颠倒
- **位置**: 1252-1290  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 选择逻辑 if targetLevel == nil || level.LevelOrder < targetLevel.LevelOrder（1259 行）在按 level_order ASC 遍历时保留的是『满足消费门槛的 LevelOrder 最小』级别（即最低档）；随后守卫 if currentLevel.LevelOrder < targetLevel.LevelOrder { shouldUpgrade = false }（1271 行）又在当前等级排序值小于目标时禁止升级。若 LevelOrder 越大代表等级越高（管理端『排序』常规用法，见 admin.go/UserLevels.vue 无方向说明），则：用户永远不会升级（当前 1 < 目标 2 → 禁止），且会被降级到最低达标档（当前 3 > 目标 1 → 放行降级）——两个比较方向互相矛盾，无论 LevelOrder 语义取哪个方向，至少一个判断是错的。
- **建议**: 确认 LevelOrder 方向后统一：选择『达标等级中 LevelOrder 最大』为 target；降级守卫应为 currentLevel.LevelOrder > targetLevel.LevelOrder 时禁止；补单元测试覆盖『升级』与『不降级』两个用例。

### [HIGH] ProcessRefundOrder 未使用数据库事务，回退操作各自独立提交
- **位置**: 1431-1515  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 退款流程依次独立执行：回退累计消费（1454-1458）、rollbackPackageOrder/rollbackDeviceUpgradeOrder（1471-1481，内部用 s.db）、回退余额（1484-1487）、rollbackInviteRewards（1490）、释放折扣（1492）、保存用户（1497）、置订单 refunded（1503）——任一步失败即产生半退款状态（如订阅已回退但订单仍 paid），且与并发支付回调无隔离。
- **建议**: 把整个退款流程包进单个 db.Transaction（tx 版本贯穿 rollback* 与邀请回退），并先对订单行 FOR UPDATE 加锁。

### [MEDIUM] 退款回退按『当前订阅』直接减去本次时长，叠加续费后回退错误
- **位置**: 1517-1628  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: rollbackPackageOrder 对 subscription.ExpireTime.AddDate(0,0,-totalDurationDays)（1608 行）——若用户支付后又续费/叠加其他订单，当前订阅到期时间并非本次订单单独贡献，减去的天数会错误侵蚀后续订单时长；rollbackDeviceUpgradeOrder 同理。
- **建议**: 退款回退基于订单激活时快照（old_expire_time/old_device_limit 已在 ExtraData 中），恢复快照而非相对扣除。

### [MEDIUM] DeleteOrders 允许物理删除 paid/refunded 财务订单记录
- **位置**: 183-218  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 仅对非 paid/refunded 订单跳过释放折扣（202-206 行），paid/refunded 订单同样被 tx.Delete 物理删除（210 行）——财务流水、审计与关联 PaymentTransaction 被抹除（未检查关联交易）。
- **建议**: paid/refunded 订单只允许软删除/归档，禁止物理删除；删除前检查关联 PaymentTransaction。

### [LOW] ExtraData 解析逻辑在 4 处重复（type/devices/months/days/coupon_free_days）
- **位置**: 976-1001,1182-1198,1517-1563  |  **类别**: duplication  |  **来源组**: A9-services-rest (其余 services)
- **问题**: processPackageOrderTx、processDeviceUpgradeOrderTx、rollbackPackageOrder、rollbackDeviceUpgradeOrder 各自重复 json.Unmarshal+类型断言提取同一组字段（type/devices/months/days/coupon_free_days），语义稍有出入（如 rollback 里 duration_months 覆盖 custom months），易漂移。
- **建议**: 抽 parseOrderExtraData(extra sql.NullString) (*OrderExtraData, error) 公共结构体统一解析。

### [LOW] 数据库事务内 spawn goroutine 发送管理员通知，回滚后产生假阳性通知
- **位置**: 1101-1114,904-911  |  **类别**: error-handling  |  **来源组**: A9-services-rest (其余 services)
- **问题**: processPackageOrderTx 在事务回调内 go func(){ SendAdminNotification(subscription_created...) }()（1101 行）——goroutine 可能在事务提交前启动，若后续步骤失败回滚，『订阅创建』通知已发出；904-911 行更新营销活动状态失败仅记日志。
- **建议**: 把通知发送移到事务成功提交之后，或通过事务提交后 hook/事件表驱动。

### [LOW] 批量取消/删除接口无 userID 维度校验，需确认仅管理员可达
- **位置**: 51-99,101-135  |  **类别**: security  |  **来源组**: A9-services-rest (其余 services)
- **问题**: MarkPendingOrderStatus 在 userID>0 时正确加过滤（65-67 行），但 CancelPendingOrders/DeleteOrders 接收任意 orderIDs 无用户维度（101-135、183-218 行，须依赖路由鉴权）；若误挂到用户路由即构成越权。
- **建议**: 在 handler 层显式区分 admin 路由并在服务层断言调用方角色，批量接口补充订单归属校验。

## internal/services/payment/alipay.go

### [MEDIUM] 三个通知解析方法功能重叠，DecodeNotification 不验签易被误用
- **位置**: 171-229  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: ParseNotification（GetTradeNotification 验签）、VerifyNotify（VerifySign 验签）、DecodeNotification（client.DecodeNotification 仅解码不验签）三套并存；handler 实际用的是 VerifyNotify，但 DecodeNotification 若被未来代码用于异步通知路径，伪造回调可直接通过。三方法字段映射逻辑完全重复（NotifyID/TradeNo/... 逐字段拷贝）。
- **建议**: 收敛为单一入口：验签 + 解码合并到一个方法（如 ParseAndVerify），删除 DecodeNotification 或在注释中显著标注"不验签，仅用于已验签场景"；用公共函数消除三处重复字段拷贝。

### [INFO] isProduction 字段赋值后从未使用
- **位置**: 21, 46, 88-91  |  **类别**: maintainability  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: AlipayService.isProduction 在构造函数中根据 ConfigJSON 计算并赋值，但结构体其余代码无任何读取；gateway 选择逻辑（WithPastSandboxGateway/WithNewSandboxGateway）只在构造期生效。
- **建议**: 删除该字段，或将 isProduction 用于日志/监控输出；顺带 59-65 行的沙箱网关分支日志可合并精简。

## internal/services/payment/applepay.go

### [CRITICAL] VerifyNotify 无条件返回 true——苹果支付回调可被任意伪造
- **位置**: 91-93  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `func (s *ApplePayService) VerifyNotify(params map[string]string) bool { return true }`，且 payment.go:669-673 对 effectivePayType=="applepay" 时直接取该返回值，verify 通过后 handler 进入订单入账流程（processPaidOrder/processPaidRecharge），且对 applepay 分支没有 trade_status 校验——攻击者只需知道订单号（或遍历订单号），POST /api/v1/payment/notify 携带 payment_type=applepay & out_trade_no=<订单号> 即可把任意订单标记为已支付，实现零成本白嫖。
- **建议**: 在实现真正的 Apple 服务端验证（App Store Server API 收据/交易校验，或 Apple Pay merchant validation + payment token 解密验签）之前，禁止启用 applepay 渠道：VerifyNotify 返回 false 并在构造函数/配置校验处拒绝该渠道上线；同时 handler 为 applepay 增加与支付宝同级的 trade_status/金额/商户校验。

### [HIGH] CreatePayment 返回假的 applepay:// scheme URL，VerifyPaymentToken 形同虚设
- **位置**: 77-89  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: CreatePayment 直接返回 `fmt.Sprintf("applepay://payment?order_no=%s&amount=%.2f", ...)`，未发起任何 Apple Pay 会话；VerifyPaymentToken 仅 `json.Unmarshal` 后无条件 return true，不校验 token 签名/来源。整个渠道是未完成的桩实现：若前端照此流程展示"支付"，用户付不了款，但回调却可被伪造（与 VerifyNotify 组合成完整漏洞链）。
- **建议**: 要么完整实现 Apple Pay（merchant validation → 客户端 token → 服务端解密验证 → 仅验证通过才置单），要么将该渠道标记为"未实现/disabled"，前端不展示、后端拒绝下单。

### [MEDIUM] 私钥/证书解析错误被静默吞掉，配置错误时服务仍"正常"创建
- **位置**: 33-52  |  **类别**: error-handling  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: NewApplePayService 中 base64 解码失败、x509 解析失败、类型断言失败全部静默跳过（err 非 nil 时直接不赋值），最终只检查 merchantID 非空即返回成功——用户配了错误的私钥/证书也不会得到任何报错，配合 VerifyNotify 恒 true，出错完全无感知。
- **建议**: 私钥/证书解析失败应返回 error（"Apple Pay 私钥解析失败: %v"），与支付宝构造函数的严格校验风格对齐；privateKey/certificate 字段当前无使用方，可一并确认其用途。

## internal/services/payment/codepay.go

### [MEDIUM] 签名字符串整体打印进日志，密钥脱敏靠 Replace 不可靠
- **位置**: 116-117  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `utils.LogInfo("码支付签名字符串(隐藏密钥): %s", strings.Replace(signStr, s.Key, "***KEY***", 1))`——signStr 含全部参数明文（含金额、订单号），且 Replace 只替换第一个出现处：若 s.Key 恰好是某参数值的子串，日志中会残留部分密钥；若密钥过短（如 8 位数字），拼接到参数串后极易与参数值撞串，脱敏失效。
- **建议**: 不再打印签名串本身，改为打印参与签名的 key 列表与各自长度；确需打印时用明确脱敏后的摘要（如 sha256 前 8 位）。

### [LOW] 与易支付同族代码大量重复（SupportedTypes、响应结构、签名）
- **位置**: 213-235, 28-36  |  **类别**: duplication  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: GetCodepaySupportedTypes 与 yipay.go 的 GetYipaySupportedTypes 逐行重复；CodepayResponse 与 YipayResponse 字段几乎一致；codepaySign 与 yipay.go 的 buildSignString+calcMD5FromStr 逻辑重复（排序、过滤空值、拼 key）。两渠道协议同源（epay 家族），可抽象公共基类。
- **建议**: 抽取 epay 基础类型（EpayResponse、Signer、SupportedTypes 解析）到独立文件，Codepay/Yipay 只保留差异化部分（签名类型、URL 推导）。

## internal/services/payment/query.go

### [MEDIUM] 商户 key 明文放在查单 URL query 中，且查单响应不验签
- **位置**: 84-87, 96-118  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `query.Set("key", key)` 把商户 MD5 密钥拼进 GET URL——即使 HTTPS 下也会进入代理/网关/应用访问日志；同时 queryEpayOrder 对响应只做格式解析，不校验响应中的 sign 字段，若 QueryURL 配置为 http（isLocalDomain 场景）或链路被劫持，攻击者可伪造"status=1"响应把订单判为已支付（配合 handler 的 IsPaid 判断直接入账）。
- **建议**: 1) 校验响应 sign（epay 查单响应含 sign，用同一签名算法核对后再取 status）；2) 强制 QueryURL 使用 https（本地调试例外需显式开关）；3) 日志中绝不记录完整查单 URL。

### [LOW] IsPaid 状态白名单过宽，可能误判未支付订单
- **位置**: 24-35  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `case "1", "2", "SUCCESS", "TRADE_SUCCESS", "TRADE_FINISHED", "PAID", "COMPLETE", "COMPLETED", "OK", "200"`——"2"（部分平台是"处理中/已关闭"）、"OK"、"200"（可能是 HTTP 风格的成功码而非支付状态）都被视为已支付；status 字段来源 firstQueryValue(raw, "trade_status", "status", ...) 在不同平台语义不一，误判会把未支付订单入账。
- **建议**: 收敛到各平台明确语义的状态值（如仅 1/SUCCESS/TRADE_SUCCESS/TRADE_FINISHED），平台差异通过 Adapter 的字段映射显式声明，去掉模糊的 "2"/"OK"/"200"。

### [INFO] 双格式响应解析（JSON→query）整体健壮
- **位置**: 121-147  |  **类别**: error-handling  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: JSON 解码优先 + UseNumber 保精度，失败回退 url.ParseQuery，HTML 响应有前置拦截，空响应有显式错误——解析路径无明显问题；stringifyQueryValue 对嵌套对象/数组退化为 fmt.Sprintf 可接受。
- **建议**: 无需修改；可选改进：对 JSON 解析成功但 result 为空的场景与 query 解析成功但为空的场景分别给出更明确的错误文案。

## internal/services/payment/wechat.go

### [MEDIUM] VerifyNotify 就地 delete(params, "sign") 修改调用方 map
- **位置**: 230-245  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `delete(params, "sign")` 直接修改 handler 传入的 params map——若调用方在验签后还需要 sign 值（记录、转发、幂等校验），数据已被删除；且该方法不检查 return_code/result_code（handler 层补查了，但接口契约上 VerifyNotify 语义不清）。
- **建议**: 改为复制 map 再删：`check := make(map[string]string, len(params)); for k, v := range params { if k != "sign" { check[k] = v } }`，保持入参只读。

### [MEDIUM] 统一下单响应未校验微信签名
- **位置**: 61-89  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: CreatePayment 读取 unifiedorder 响应后只检查 return_code/result_code，未按微信 v2 规范用 APIKey 验证响应 sign——中间人/网关异常可篡改 code_url 指向恶意二维码（用户扫码即中招）。对比 QueryOrder 有验签（173-183 行），下单反而没有，不一致。
- **建议**: 复用 s.Sign：取出响应 XML 中的 sign，过滤后重新计算并 EqualFold 比对，失败即报"微信下单响应签名验证失败"。

### [LOW] parseWechatXMLParams 嵌套元素上下文丢失
- **位置**: 199-228  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: 解析器在 EndElement 时把 currentKey 直接置空，不恢复父级元素——遇到嵌套 XML（如 <xml><a><b>v</b><c>w</c></a></xml>）时 </b> 后 currentKey 被清空，<c> 的子文本会错挂到错误键或丢失。微信支付扁平响应不受影响，但该函数是通用解析器，未来若响应结构变化会出隐蔽 bug。
- **建议**: 用栈保存元素名路径，或直接用 encoding/xml 的结构体解码（定义微信响应 struct + xml tags），替换手写 token 遍历。

### [LOW] nonce 用时间戳伪随机生成，可预测
- **位置**: 92-99  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `b[i] = charset[time.Now().UnixNano()%int64(len(charset))]`——基于时钟的"随机"串可被预测，且存在取模偏置；微信 nonce_str 若被预测，配合签名算法可实现重放/构造攻击（风险低但不应依赖）。
- **建议**: 改用 crypto/rand 生成：`rand.Read` + base64 编码或按 charset 索引，确保每次下单 nonce 不可预测。

## internal/services/payment/yipay.go

### [CRITICAL] RSA 商户私钥前 50 字符被明文写入日志
- **位置**: 604-644 (609)  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `utils.LogInfo("易支付RSA签名: 私钥长度=%d, 内容前50字符=%s", len(s.MerchantPrivateKey), s.MerchantPrivateKey[:min(50, len(s.MerchantPrivateKey))])`——把商户 RSA 私钥的一部分直接打进日志（日志会进文件/可能进 SSE 面板）。PEM 私钥头"-----BEGIN PRIVATE KEY-----"占 26 字符，前 50 字符已含约 24 字节真实密钥材料；配合暴力补全/其他泄露面，私钥泄露风险极高。
- **建议**: 删除该日志行；确需调试时只打印长度与 PEM 头类型（`strings.Contains(..., "BEGIN")`），绝不输出密钥字节。同时全库检索其他打印私钥/密钥材料的日志。

### [MEDIUM] 退款 RSA 签名失败静默降级为 MD5
- **位置**: 996-1016  |  **类别**: error-handling  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: 新格式退款中 `sign, err = s.signRSASign(signStr); if err != nil { ... sign = s.calcMD5FromStr(signStr); params["sign_type"] = "MD5" }`——配置 RSA 的商户在私钥错误/签名失败时，退款请求被静默降级成 MD5 并照样发出，与配置的签名强度不符，且失败原因只有一行 LogError，调用方拿到的是"成功发出"的假象。
- **建议**: 签名失败应直接返回 error 中止退款，除非用户显式配置"允许 RSA 失败降级 MD5"开关。

### [MEDIUM] isLocalDomain 子串匹配误伤生产域名
- **位置**: 70-79, 81-90  |  **类别**: logic  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `strings.Contains(domain, "10.")` / `"172."` / `"local"` 等子串匹配：域名 "example10.com"、"my172.io"、"locality.cn"、"10x.example.com" 都会被判定为本地环境，从而用 `http://` 生成回调地址（buildBaseURL 86-88 行）——生产环境若域名恰好含这些子串，回调地址变成明文 http 且端口错误，支付回调直接失败。
- **建议**: 用 url.Parse 取出 Host 后精确判断：hostname == "localhost" / "127.0.0.1"，IP 用 net.ParseIP + IsLoopback/IsPrivate 判断，域名以 ".local" 结尾才算 local。

### [MEDIUM] MD5+RSA 模式缺 rsa_sign 时降级为仅 MD5 校验并通过
- **位置**: 529-533  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: `utils.LogWarn("MD5+RSA模式: 缺少rsa_sign，仅通过MD5校验"); return true`——调用方配置 MD5+RSA 本意是 RSA 提供更强保证，缺 rsa_sign 时却静默放行；一旦 MD5 密钥泄露（或该平台 MD5 密钥较短），攻击者可完全伪造回调，RSA 形同虚设。
- **建议**: MD5+RSA 模式下 rsa_sign 缺失/验签失败一律返回 false 并告警，明确"配置了 MD5+RSA 就必须验 RSA"；若平台确实不传 rsa_sign，应引导用户改用 MD5 模式而非静默降级。

### [LOW] Sign/calculateMD5Sign 无外部调用方
- **位置**: 501-549  |  **类别**: maintainability  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: grep 确认 YipayService.Sign 与 calculateMD5Sign 全库无引用（VerifyNotify 自行调用 calcMD5FromStr），是死代码；且 Sign 的命名与 wechat.go 的 Sign（签名核心逻辑）语义混淆，易被误用。
- **建议**: 删除这两个方法，或让 VerifyNotify/退款逻辑复用 calculateMD5Sign 消除重复。

### [LOW] submit 回退 URL 与请求参数含敏感信息且无完整验签
- **位置**: 451-459, 461-499  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: buildSubmitURL 日志打印完整 URL（含 sign/rsa_sign，虽非密钥本身但可被日志平台索引）；postForm 对响应只检查 HTTP 状态不做业务签名验证。与 query.go 的 key-in-URL 问题叠加，属 epay 家族共性的"日志与传输敏感性"问题。
- **建议**: 日志只打印主机与参数 key 列表；对 mapi 响应（含 code/trade_no）增加签名校验（响应带 sign 时）；统一日志脱敏工具函数。

### [LOW] 二维码提取流程会递归抓取网关返回页面（潜在低危 SSRF）
- **位置**: 653-786  |  **类别**: security  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: extractQRCodeFromPaymentPage 按 JS 重定向递归 GET（最多 15 跳），handleFormRedirect 的 actionURL 虽经 utils.ValidateHTTPURL 校验，但该 URL 来自网关返回的 HTML 表单——整体依赖"管理员配置的网关可信"这一假设，若网关被攻破或被配置指向恶意站点，服务端会按其页面重定向抓取任意 URL。
- **建议**: 为抓取目标增加主机白名单（仅允许网关配置的域名/IP）并拒绝回环地址；对 extractQRCodeFromHTML 中提取的 URL 同样校验后再返回给前端。

## internal/services/payment/yipay_adapter.go

### [LOW] detectByURL 所有分支恒返回 "standard"，适配器框架是空骨架
- **位置**: 137-159, 161-163  |  **类别**: maintainability  |  **来源组**: A8-services-config-update-payment (config_update + payment services)
- **问题**: detectByURL 中 fhymw.com/yi-zhifu.cn/ezfp.cn/myzfw.com/8-pay.cn/... 全部命中同一分支返回 "standard"，其余情况也返回 "standard"——分支列表没有任何区分作用，整个 URL 特征检测是死逻辑；RegisterAdapter 无调用方，adapters map 只有标准实现，SupportsSignatureType 恒 true 使 NewYipayService 中的告警永不触发。
- **建议**: 要么为不同平台实现真实差异（字段名/URL 推导差异），要么删除框架只保留 StandardYipayAdapter，并把 detectYipayPlatform 简化为直接返回标准适配器，减少误导。

## internal/services/promotion/promotion.go

### [MEDIUM] promotionAppliesToPackage 在 JSON 解析失败时 fail-open 放行所有套餐
- **位置**: 46-61  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: json.Unmarshal 出错返回 true（53 行）——PackageIDs 数据损坏时折扣可作用于任意套餐，与优惠券侧 parseApplicablePackages 的容错语义不一致（应 fail-closed）。
- **建议**: 解析失败时返回 false（不适用），并记日志便于发现脏数据。

### [LOW] ApplyDiscount 只取最早一条 pending 折扣，无最优选择
- **位置**: 19-44  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: Order(created_at ASC).First()（27 行）取最早创建的参与记录，不比较折扣金额大小——用户同时拥有多个可用折扣时可能拿到金额较小者。
- **建议**: 候选记录中选折扣金额最大者，或明确『先到先得』策略并注释。

## internal/services/repo_sync/repo_sync.go

### [HIGH] 同步文件经 GET /repo-sync/*filepath 完全公开访问，含目录列表
- **位置**: 403-432,67-68  |  **类别**: security  |  **来源组**: A9-services-rest (其余 services)
- **问题**: listLocalFiles 直接生成公开 URL /repo-sync/<rel>（423 行），router.go:38 注册公开路由 ServeRepoSyncFile（handlers/repo_sync.go:67 注释『公开访问同步目录』）——若同步的私有仓库含敏感文件（配置、密钥、用户数据），将被无鉴权下载，且目录浏览页同样公开。
- **建议**: 对 /repo-sync 增加访问控制（登录/管理员），或仅放行白名单扩展名并关闭目录列表。

### [LOW] removeStaleFiles 会删除本地目录中一切非远程文件
- **位置**: 382-401  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 本地 repo_sync 目录里运维手工放置/第三方写入的文件只要不在 keep 集合即被物理删除（393-396 行）——属高危默认行为。
- **建议**: 仅删除上次同步记录过的文件（持久化清单），或对非同步来源文件保留并告警。

### [LOW] Tick→SyncNow 链路重复加载配置，状态写入逐条 upsert
- **位置**: 152-186,189-256  |  **类别**: performance  |  **来源组**: A9-services-rest (其余 services)
- **问题**: Tick 里 ShouldRunNow 一次 LoadConfig，SyncNow 内再次 LoadConfig（两次 DB 全量读）；saveStatus 4 行逐条 OnConflict upsert（444-451 行）可合并为单次批量 upsert。
- **建议**: SyncNow 接受已加载的 cfg 参数；saveStatus 用批量 upsert。

## internal/services/scheduler/scheduler.go

### [MEDIUM] 到期提醒的 ±1 小时窗口配合 24h ticker，大部分订阅永远收不到 7/3/1 天提醒
- **位置**: 114-156  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 提醒窗口为 targetTime±1h（153-155 行），而 ticker 是 24h 整点触发：只有到期时间恰好落在『每日 tick 时刻±1h』内的订阅能命中提醒窗口，其余订阅（如到期时间为 tick+2h）只会在过期后被『已过期』分支补发，7/3/1 天提醒系统性缺失。
- **建议**: 改为按到期时间区间查询（expire_time in [now, now+7d] 等）并去重，或改用小时级 ticker 扫描窗口。

### [MEDIUM] Scheduler.Stop 后无法重新 Start（stopChan 已 close）且 running 非原子
- **位置**: 45-80  |  **类别**: logic  |  **来源组**: A9-services-rest (其余 services)
- **问题**: Stop() close(stopChan)（73 行）后再次 Start() 会重起 7 个 goroutine，但它们 select 到已关闭的 stopChan 立即退出——调度器不可重启；running bool 无锁保护，并发 Start/Stop 有数据竞争。
- **建议**: 用原子标志+每次 Start 重建 stopChan（或 context cancel 模式），支持重启。

### [MEDIUM] 自动备份把 .env（含 SMTP/密钥）与整库打进 zip 存放于 uploads/backups
- **位置**: 575-685,641-664  |  **类别**: security  |  **来源组**: A9-services-rest (其余 services)
- **问题**: runAutoBackup 把 .env、config.yaml 与 cboard.db 全部打进 backup_auto.zip（599-664 行），明文敏感包存放于 uploads/backups 目录；远程备份默认目标还是 backup_service 硬编码的 moneyfly/moneyfly1 仓库（见 backup_service.go 发现）。
- **建议**: 备份 zip 加密或排除 .env；对 backups 目录做强访问控制；远程备份默认关闭直到显式配置 owner/repo。

### [LOW] cleanupExpiredDataNow 中多个 Delete 的错误被忽略
- **位置**: 266-291  |  **类别**: error-handling  |  **来源组**: A9-services-rest (其余 services)
- **问题**: VerificationCode/LoginAttempt/EmailQueue 三个 Delete 未检查返回错误（270-275 行），只有 AuditLog 检查了 RowsAffected——DB 故障时过期数据静默不清理。
- **建议**: 检查各 Delete 错误并记日志。

### [LOW] checkAndRunNodeUpdate 中 enableSchedule 判断重复两遍
- **位置**: 385-391  |  **类别**: maintainability  |  **来源组**: A9-services-rest (其余 services)
- **问题**: 385 行与 389 行连续两个 if !enableSchedule { return }，第二处为死代码。
- **建议**: 删除重复分支。

## internal/utils/audit.go

### [MEDIUM] CheckBruteForcePattern 用 before_data 的 JSON 整体去重统计用户名，且 LIKE 未转义
- **位置**: 503-551  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: Group("before_data")（line 530）对整段 JSON 去重：同一用户名若附带不同附加字段会被计为多个“不同用户名”，撞库误报；line 540 的 before_data LIKE '%'+username+'%' 未做 EscapeLikePattern 转义，用户名含 %/_ 时匹配失真（参数化无注入风险，但计数偏差）。且该函数每次登录尝试执行 2-3 次 AuditLog COUNT 查询，无 (ip_address,created_at)/(action_type,created_at) 索引时随表增长越来越慢。
- **建议**: 把用户名作为独立列（或 before_data 内固定键）存储并 Group 该键；LIKE 前套用 utils.EscapeLikePattern；为 AuditLog 补复合索引。

### [MEDIUM] CreateBusinessLogAsync 无界起 goroutine，且 goroutine 内无 recover
- **位置**: 394-462  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 每次调用 go func(){...}()（line 415），订阅拉取等高频路径可瞬间产生大量 goroutine 排队写库；DB 抖动时写失败只记日志，goroutine 堆积；goroutine 内任何 panic（如 json.Marshal 异常数据）会直接崩溃整个进程（无 recover）。
- **建议**: 改用工位池/有缓冲 channel + 单消费者批量插入；goroutine 入口 defer recover() 并记日志。

### [LOW] CreateBusinessLog / Fast / Async / CreateAuditLogSimpleFast 是四份近重复实现
- **位置**: 255-392  |  **类别**: duplication  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 四函数约 150 行结构几乎一致（IP/UA/用户提取、level→status 映射、AuditLog 构造，line 255-392），差异仅为 skipGeoIP、异步、字段多寡；修改一处（如加字段）需同步四处。
- **建议**: 抽 logOptions{skipGeoIP, async, includeRequest} + 单一 buildAuditLog 构造器，四函数变薄壳。

### [LOW] CreateSystemErrorLog 未对 c==nil 防护，而 CreateBusinessLog 防护了
- **位置**: 553-630  |  **类别**: error-handling  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: CreateBusinessLog 显式处理 c==nil（line 268-285），CreateSystemErrorLog 只在 line 554 对 c.Set 判空，却在 line 587/608-610 直接使用 c.Request.URL.Path —— 若未来从定时任务以 nil 调用即 panic，风格也不一致。
- **建议**: 统一 nil 防护：c==nil 时跳过 IP/Path 字段。

### [LOW] 审计日志同步写库 + 每次 GeoIP 查询，热点路径延迟叠加
- **位置**: 49-122  |  **类别**: performance  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: CreateAuditLog 在请求路径同步 db.Create（line 114）并调用 geoip.GetLocationWithCache（line 73）；登录/下单等接口一次请求可能产生多条审计，DB 慢时直接拖慢主流程。
- **建议**: 审计写入统一走异步队列（参考 Fast/Async 变体），GeoIP 已有缓存可保留。

### [LOW] buildRequestParams 把完整 query 参数（含可能的敏感参数）写入审计库
- **位置**: 17-47  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: params["query"] = c.Request.URL.Query()（line 34）原样序列化全部查询串；若某接口在 query 中携带 token/密码类参数（部分支付回调 URL 会带签名参数），会明文落入 AuditLog 表。
- **建议**: 按 response.go 的 sensitiveFields 白名单对 query 键做脱敏后再入库。

## internal/utils/common.go

### [HIGH] 订单号生成存在 TOCTOU 竞态：并发请求可拿到相同订单号
- **位置**: 185-204  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: generateOrderNo 流程（line 185-204）：读当日最大序列（getMaxSequence）→ 自增 → checkExists 探测（独立查询）→ 返回给调用方，由调用方在**另一个事务**里插入订单。两个并发请求可同时读到相同 maxSeq 并各自通过 checkExists（此时都未插入），生成相同订单号；若无 order_no 唯一索引则重复订单落库（支付串号、对账错乱），有唯一索引则创建订单报错。重试循环（line 195-201）无法覆盖“探测后插入前”的窗口。
- **建议**: 给 orders/recharge_records 的 order_no 建唯一索引，改为“直接插入，冲突时重试递增序列”（乐观重试）；或引入 DB 原子计数器/自增列拼订单号。

### [MEDIUM] GenerateOrderNo 接收 interface{}，非 *gorm.DB 时静默退化为无 DB 生成器
- **位置**: 206-243  |  **类别**: architecture  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: db 类型断言失败时生成器带 nil DB（line 213-215）：getMaxSequence 返回 0、checkExists 返回 false → 每笔订单号恒为 ORD<date>001，所有订单撞号。调用方传错类型（如 *sql.DB 或 nil）不会报错，故障静默且难排查。
- **建议**: 签名改为 *gorm.DB 并显式处理 nil（返回错误或要求非 nil），杜绝静默降级。

### [MEDIUM] GetSessionTimeout 每次签发 token 执行 2 次 DB 查询且无缓存
- **位置**: 609-659  |  **类别**: performance  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: CreateAccessToken（line 644）每次调用 GetSessionTimeout → 用户级 + 全局级 2 次 SystemConfig 查询（line 613-627）；登录、刷新、每 1 小时续期都触发。系统配置属于低频变更数据，查询结果可长缓存。
- **建议**: 加 1-5 分钟 TTL 的内存缓存（配置更新时失效），或启动时一次性加载到内存。

### [LOW] findMax*Sequence / check*Exists 四函数复制粘贴，GenerateRechargeOrderNo 的 userID 参数未使用
- **位置**: 114-157  |  **类别**: duplication  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: findMaxOrderSequence 与 findMaxRechargeOrderSequence（line 114-157）、checkOrderNoExists 与 checkRechargeOrderNoExists（line 160-175）仅表名不同；GenerateRechargeOrderNo(userID uint, ...)（line 219）的 userID 从未被使用（死参数），接口 orderNoGenerator 的 getTableName 也未被 generateOrderNo 使用，接口设计名不副实。
- **建议**: 把表名/前缀作为参数合一；删除无用参数与 getTableName；或在生成器内真正按表名动态查询。

### [LOW] CalculatePaymentSummary 全表扫描 orders+recharge_records 无时间范围下推
- **位置**: 394-435  |  **类别**: performance  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: SQL 内 range_paid/range_revenue 用 CASE WHEN created_at>=? AND created_at<? 过滤（line 414-417），外层无 WHERE，两个表每次仪表盘加载都全量聚合；缺少 (status, created_at) 复合索引时随数据量线性退化。
- **建议**: 把时间范围下沉为 WHERE created_at >= ?（子查询内），并补 (status, created_at) 索引。

### [LOW] GenerateCouponCode 模偏差 + 随机失败时回退时间戳，且每字符一次 crand.Read
- **位置**: 245-259  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: charset[randBytes[0] % 36]（line 256）存在模偏差（256 % 36 ≠ 0），弱化 8 位码熵；fallback 用 UnixNano（line 253）可预测（攻击者可枚举优惠码）；每次循环单独 Read 1 字节效率低。
- **建议**: 一次读足 8 字节再用拒绝采样（rejection sampling）消除偏差；失败时返回错误而非可预测回退。

### [LOW] GenerateRandomString 失败 panic，与同文件其他生成函数回退风格不一致
- **位置**: 754-765  |  **类别**: style  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: crand.Int 失败时 panic(fmt.Sprintf(...))（line 760），而 GenerateSubscriptionURL/GenerateCouponCode 等失败时回退时间戳。同语义函数错误策略分裂，panic 会让运行中的服务器（如重启令牌场景）直接崩溃。
- **建议**: 返回 (string, error) 让调用方决策，或统一回退策略并记录日志。

## internal/utils/crypto.go

### [MEDIUM] NormalizePrivateKey 第二分支（PKCS#8）不可达，PKCS#8 密钥被误标为 RSA
- **位置**: 25-47  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 第一分支条件 strings.HasPrefix(cleanKey, "MII")（line 25）已覆盖所有以 MII 开头的串（包括 MIIE/MIIEv，即 PKCS#8 前缀），因此 line 37 的 strings.HasPrefix(cleanKey, "MIIE") 分支永远不会执行——死分支。PKCS#8 base64 密钥会被包上 "-----BEGIN RSA PRIVATE KEY-----" 错误标签，依赖 PEM 标签解析的调用方（部分支付 SDK）会解析失败。
- **建议**: 调换顺序：先判 MIIE（PKCS#8）再判 MII（PKCS#1 RSA），或统一用 encoding/pem + x509.ParsePKCS8PrivateKey/ParsePKCS1PrivateKey 试解析确定类型。

### [LOW] FormatPEMKey 与 FormatPEMPublicKey 是同一实现的两份拷贝
- **位置**: 96-158  |  **类别**: duplication  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 两个函数除 label 字符串外逻辑完全一致（去标记、去空白、64 字符折行，line 96-158）；FormatPEMKey 已带 keyType 参数，FormatPEMPublicKey 可删除。
- **建议**: 删除 FormatPEMPublicKey，调用处改 FormatPEMKey(key, "PUBLIC KEY")。

### [LOW] 未知类型密钥（如 EC）按长度猜测为 RSA 并错误包装
- **位置**: 49-56  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: EC 私钥（"MHcCAQEE..."）不以 MII 开头，落入 len(cleanKey)>100 分支被包成 RSA PRIVATE KEY（line 51），标签与内容不符；短于 100 的合法密钥直接返回空串。猜测式包装应改为解析失败即报错。
- **建议**: 先尝试 x509 解析判断真实类型再包装；无法识别时返回错误，不做猜测。

## internal/utils/logs.go

### [LOW] 日志函数依赖全局 database.GetDB()，与 WithDB 变体风格分裂
- **位置**: 20-217  |  **类别**: architecture  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: CreateBalanceLog/CreateCommissionLog 提供 WithDB 变体（line 136/183）而 CreateRegistrationLog/CreateSubscriptionLog/UpdateCommissionLogStatus 只能走全局 DB（line 21/84/209），测试与复用不便；包内两种依赖风格并存。
- **建议**: 统一为显式传 *gorm.DB（全局 DB 仅在入口处注入），便于单测与事务内调用。

### [LOW] CreateRegistrationLog 邀请码为空但 inviterID 非空时来源标记为 direct
- **位置**: 42-49  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: inviteCode=="" 时 RegisterSource 恒为 "direct"（line 39），即使 inviterID != nil 或注册实际经由邀请关系；来源统计会失真。
- **建议**: 以 inviterID 是否为 nil 作为邀请来源的判定条件之一（inviteCode 为空但带 inviterID 时标记为 invite_code）。

## internal/utils/network.go

### [HIGH] GetRealClientIP 无条件信任客户端可控头，配合限流构成暴力破解绕过
- **位置**: 180-258  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 函数按优先级读取 CF-Connecting-IP/True-Client-IP/X-Forwarded-For（line 182/189/196），且未校验请求是否来自可信代理；router.go:15 已 SetTrustedProxies(nil)（gin 层面不信任任何代理头），但本函数完全绕过 gin。攻击者直接连服务器时伪造 X-Forwarded-For: 8.8.8.8、8.8.4.4… 即可使登录/注册/验证码限流键（ratelimit.go 全部使用本函数）每次不同 → 限流完全失效；审计/安全日志中的 IP 也可被任意伪造（日志投毒）。
- **建议**: 增加可信代理判定：仅当部署配置了 TRUSTED_PROXY（CIDR 列表，对照 RemoteAddr）时才解析这些头；否则直接返回 c.ClientIP()/RemoteAddr。同时按 XFF 语义从左（最接近客户端）取第一个可信链上的 IP。

### [MEDIUM] IsPrivateIP 对 IPv4-mapped IPv6 判断错误，ValidateHTTPURL 存在 SSRF 盲区
- **位置**: 126-178  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 当 ip 是 16 字节的 IPv4-mapped 地址（::ffff:10.0.0.1）时，ip.To4()!=nil，但 line 139/143/147/151/155 直接用 ip[0]/ip[1] 判断——16 字节表示的 ip[0]=0x00（映射前缀），10/172/192.168 等私有段全部漏判 → ValidateHTTPURL（line 52-56）认为公网放行，实际请求打向内网（配合 DNS rebinding 的 TOCTOU：LookupIP 校验后请求再解析一次）。另遗漏 100.64.0.0/10（CGNAT）、0.0.0.0/8、198.18.0.0/15、TEST-NET 等保留段。
- **建议**: 先 v4 := ip.To4()，用 v4[0]/v4[1] 判断；补齐 CGNAT/保留段清单；ValidateHTTPURL 校验后 pin 住已解析 IP 直连（自建 dialer 并回填 Host），消除 DNS rebinding。

### [MEDIUM] BuildBaseURL 信任 X-Forwarded-Proto 与 Host 头，可致链接投毒
- **位置**: 61-84  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 未配置 domain_name 时 scheme 取 X-Forwarded-Proto（line 69-72，客户端可控），host 直接用 r.Host（line 83，可伪造）：生成到邮件/订阅/回调里的链接会带上攻击者主机名，构成钓鱼/链接投毒向量。
- **建议**: host 与配置的合法域名白名单比对（或至少校验 Host 非空且匹配 ServerName）；X-Forwarded-Proto 仅在可信代理后采信。

### [LOW] GetBuildBaseURL 与 GetDomainFromDB 重复做两次 DB 查询
- **位置**: 86-110  |  **类别**: duplication  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 两函数各自执行 general→system 两次 First 查询（line 90-94/104-108，各 2 次 DB 往返），在订阅/邮件等每请求路径重复触发；逻辑也完全重复。
- **建议**: 抽 shared lookupDomain(db) 并加进程内短 TTL 缓存（配置变更时失效）。

## internal/utils/response.go

### [MEDIUM] ParsePagination 的 limit 仅在 size==20 时生效，分页逻辑明显错误
- **位置**: 175-183  |  **类别**: logic  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: limit 分支条件 if size == 20 { size = limit }（line 180-182）：客户端只传 ?limit=50 时（size 默认 10）limit 被忽略；传 ?size=20&limit=10 反而把 size 覆盖成 10。skip 分支也只在 page==1&&size==10 时换算（line 171-173）。这套参数互操作规则无法推导出合理语义，前端按文档传参即踩坑。
- **建议**: 定义明确优先级：limit/offset 与 page/size 二选一（后者存在时前者忽略），统一 clamp 到 [1,10000]/[1,100]，删除 size==20 特判。

### [LOW] SuccessResponse 的 code 参数被忽略，恒返回 0
- **位置**: 50-60  |  **类别**: architecture  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: SuccessResponse(c, code, ...) 中 code 只用于 HTTP 状态码，响应体 Code 恒为 ErrCodeSuccess（line 53）；与 ErrorResponse 的 errCode 语义（业务码）不对应，API 契约里“业务码”形同虚设。
- **建议**: 要么让 SuccessResponse 的 code 同时作为业务码回写，要么删除参数只保留 HTTP 状态，避免误导调用方。

### [LOW] 日志脱敏对 context 键精确匹配、对值做子串匹配，覆盖面不完整
- **位置**: 206-309  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: sensitiveFields 键匹配要求 keyLower 精确等于白名单项（line 254-256，枚举有限），新增敏感字段名（如 "password_hash"）即漏脱敏；值侧 sanitizeSensitiveValue 的子串匹配会把正常长串（如带 "token" 的 URL）误判为敏感而截断，两方向都有误报/漏报。
- **建议**: 改为大小写不敏感的子串匹配敏感词列表，或统一用 JSON 序列化后整体扫描脱敏。

## internal/utils/safe_convert.go

### [INFO] 转换工具实现正确，Must 变体静默归零需调用方注意
- **位置**: 40-78  |  **类别**: other  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: Safe* 变体带溢出检查返回 error，Must* 变体溢出返回 0（line 40-78）。审计代码用 MustSafeUintToInt64 把 userID 转 int64（uint 最大值在 64 位平台不会溢出），实际安全；仅提示：Must 系列无法区分“合法 0”与“溢出归零”，跨平台（32 位）时需警惕。
- **建议**: 无（如需更严谨，可在 Must 变体内对溢出打一次日志）。

## internal/utils/validator.go

### [MEDIUM] SanitizeSearchKeyword 剥离 SQL 关键字，破坏合法搜索词且无实际防护
- **位置**: 29-53  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: ReplaceAll 删除 "select"/"union"/"exec" 等子串（line 38-42）：搜索 "selection" 会变成 "ion"、"unioned" 变 "ed"，用户检索体验被破坏；而所有查询均已参数化（grep 确认 LIKE ? 绑定参数），关键字剥离提供的是虚假安全感。
- **建议**: 只保留长度上限 + 允许字符白名单（已有 line 44-50），去掉关键字黑名单删除逻辑。

### [MEDIUM] EscapeLikePattern 转义在 SQLite/Postgres 上无效（LIKE 无默认 ESCAPE 字符）
- **位置**: 86-92  |  **类别**: security  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: 函数把 % 转成 \%（line 90），但全部调用方（admin.go/logs.go/node.go 等十余处）用的都是普通 LIKE ? 且未带 ESCAPE '\\' 子句。MySQL 默认以反斜杠转义故生效；SQLite/Postgres 的 LIKE 没有默认转义符，\% 是“字面反斜杠+任意串”，搜索仍会通配匹配，可能返回超出预期的行（信息面放大）；注释（line 87）也承认未用 ESCAPE 子句。_ 未转义同样会单字符通配。
- **建议**: 所有 LIKE 查询统一追加 ESCAPE '\\' 子句（或改用 instr/ILIKE），并同步转义 _；在注释中明确各后端行为。

### [LOW] SanitizeErrorPath 与 response.go 的 sanitizeErrorPath 重复实现且行为不一致
- **位置**: 95-143  |  **类别**: duplication  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: validator.go:95 的导出函数保留“最后两段目录+文件名”，response.go:277 的同名私有函数行为相近但细节不同（Windows 盘符前缀处理、"..." 截断逻辑只在前者）。两处各自演化容易分叉。
- **建议**: 合并为单一导出函数（保留更完备的 validator.go 版本），response.go 改为调用它。

### [LOW] ValidateEmail/ValidateUsername 每次调用重新编译正则
- **位置**: 10-14  |  **类别**: performance  |  **来源组**: A1-backend-core (main/middleware/utils/core)
- **问题**: regexp.MatchString 在每次调用时编译 pattern（line 12/57），登录/注册高频路径重复开销；Go 正则 RE2 编译不便宜。
- **建议**: 包级 var emailRe = regexp.MustCompile(...) 并复用。

## scripts/admin_tool.go

### [HIGH] 用户名/邮箱 OR 查询会命中并接管任意用户账户，将其提升为管理员
- **位置**: 118-149  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: db.Where("username = ? OR email = ?", username, email).First(&user) 中 OR 条件可能在多行命中时取任意一条；当普通用户已注册该 email 时，脚本会把该普通用户直接 is_admin=true 并重置其密码，造成越权接管。updatePassword（L55）同样使用 OR 查询，可能把非管理员账号当管理员重置。
- **建议**: 改用两次独立精确查询（先 username 后 email），校验目标用户 IsAdmin 或处于已知状态；至少把 OR 查询改为 AND 语义并按 ID 定位。

### [MEDIUM] 存在任一管理员时直接改写第一个管理员，可能误改他人账号
- **位置**: 122-135  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 当按 username/email 未命中但库里已有任意管理员时，脚本将 existingAdmin（First 取到的第一个管理员）的 username/email/password 整体覆盖为目标值——会把现有管理员账号改名、改密，多管理员场景下破坏力明显。
- **建议**: 命中已有管理员时仅重置其密码，不修改 username/email；改名需求应走管理后台接口。

### [MEDIUM] 非 production 环境允许默认密码 admin123，且脚本会明文回显密码
- **位置**: 103-111, 172-176  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 仅当 ENV=="production" 时才强制要求 ADMIN_PASSWORD；部署脚本（install.sh/bt-deploy.sh）均未设置 ENV，导致默认 admin123 直接生效。L83 与 L172-176 还会把新密码明文打印到终端/日志，若被 tee 到日志文件则泄露。
- **建议**: 去掉 ENV 判断，任何环境未提供 ADMIN_PASSWORD 都拒绝创建或强制交互输入；回显改为仅提示"密码已从环境变量读取"。

### [LOW] updatePassword 分支通过 os.Exit 退出且无返回值契约，与主流程风格不一致
- **位置**: 48-52  |  **类别**: style  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: updatePassword 在密码过短时 os.Exit(1)，其余路径 log.Fatalf，主流程随后 return；CLI 工具可接受但错误处理风格不统一，且 os.Exit 会跳过 defer 清理。
- **建议**: 统一改为返回 error 由 main 处理，避免裸 os.Exit 散落。

## scripts/configure_payment.sh

### [CRITICAL] 真实商户支付凭据被硬编码并提交进 Git 仓库
- **位置**: 10-14  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: YIPAY_PID="REDACTED_PID"、YIPAY_KEY="REDACTED_MERCHANT_KEY"、网关/提交/查询 URL 直接写死，注释还注明"从截图获取"——这是部署者真实的三方支付商户密钥，已随 configure_payment.sh/configure_payment.sql 进入版本历史，任何拿到仓库的人都能冒用该商户发起支付回调或对账。
- **建议**: 立即在码支付/易支付后台重置密钥；用 git filter-repo 重写历史清除凭据；脚本改为从环境变量或交互输入读取，严禁写死。

### [MEDIUM] 脚本只回显 SQL 不执行，且生成 MySQL 方言 SQL，易误导用户
- **位置**: 45-104  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 脚本声称"配置工具"，实际只把 INSERT ... ON DUPLICATE KEY UPDATE 打印到终端让用户手动粘贴执行；SQL 使用 MySQL 专属语法，在项目支持的 SQLite/PostgreSQL 上直接报错；CONFIG_JSON 内嵌换行未转义，粘贴执行时同样会语法失败。
- **建议**: 改为直接以 sqlite3/psql/mysql 客户端按数据库类型执行，或用 Go 脚本调用 database 层写入；至少显式标注"仅 MySQL"并转义 JSON。

### [LOW] 域名输入未校验，可能生成 https://https://... 的畸形回调地址
- **位置**: 17-33  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: read DOMAIN 后直接拼 "https://${DOMAIN}/..."，用户若输入带协议/路径的域名会生成非法 URL；localhost 判定依赖子串匹配，`mylocalhost.com` 也会被当成本地地址降级为 http。
- **建议**: 校验 DOMAIN 只含 [a-z0-9.-]（允许 :port），统一小写并去除尾斜杠。

## scripts/configure_payment.sql

### [CRITICAL] SQL 文件中同样硬编码真实支付商户密钥
- **位置**: 5-6, 53-55, 70-71  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 与 configure_payment.sh 相同，PID REDACTED_PID 与密钥 REDACTED_MERCHANT_KEY 明文出现在 INSERT 与 ON DUPLICATE KEY UPDATE 子句中并已纳入版本控制，构成商户凭据泄露。
- **建议**: 从仓库删除凭据并重写历史；SQL 改为引用 @pid/@key 变量或从配置文件读取，文档只留占位符。

### [MEDIUM] JSON_OBJECT/JSON_ARRAY/ON DUPLICATE KEY 均为 MySQL 专属语法，与项目三库支持矛盾
- **位置**: 60-82, 108-132  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 项目 go.mod 同时引入 sqlite/mysql/postgres 驱动，而本脚本依赖 MySQL 的 JSON 函数与 upsert 语法；在 SQLite（datetime('now') 体系）或 PostgreSQL 上执行直接报错，无法作为通用迁移脚本使用。
- **建议**: 按数据库类型拆分脚本，或改用 GORM 迁移/初始化代码完成配置写入，保证跨库一致。

## scripts/download_dbip.go

### [MEDIUM] 下载无超时、无大小限制、无完整性校验，失败时旧数据库已被删除
- **位置**: 95-131  |  **类别**: error-handling  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: http.Get 未设置 client.Timeout，网络挂起时脚本永久阻塞；io.Copy 无上限，恶意/异常响应可写满磁盘；L55 在下载前先 os.Remove(dbipFile)，新文件下载失败后旧 MMDB 已丢失，且不验证解压结果是否为合法 MMDB。
- **建议**: 使用 http.Client{Timeout: 5m}，按 Content-Length 限制最大体积，先下载到 .tmp 文件校验（gzip 解压成功 + 文件头）后再原子 rename 替换。

### [LOW] 文件已存在时的交互提示在非交互环境会阻塞或误判
- **位置**: 42-56  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 仅当 CI/BUILD_MODE 环境变量存在时跳过交互；普通定时任务/无人值守运行遇到已存在文件会卡在 fmt.Scanln。
- **建议**: 增加 --force/--yes 参数；未提供时默认跳过下载而不是等待输入。

## scripts/download_geoip.go

### [HIGH] 先创建输出文件再下载，失败会残留空/半文件，CI 后续运行据此跳过下载导致损坏数据库上线
- **位置**: 78-104  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: downloadFile 先 os.Create(geoipFile) 再 http.Get+io.Copy；下载失败时空文件已落盘，下次在 CI/BUILD_MODE 下 os.Stat 命中即"跳过下载"，把损坏的空 GeoLite2-City.mmdb 直接用于生产。与 download_dbip.go（先 Remove 后下载）行为不一致。
- **建议**: 统一先下载到 .part 临时文件，成功后再 rename 到最终路径；下载完成后校验文件大小>阈值（如 1MB）且为合法 MMDB。

### [LOW] 与 download_dbip.go 存在大段复制粘贴（提示、文件大小格式化、目录创建）
- **位置**: 20-76  |  **类别**: duplication  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 两个下载脚本的 os.MkdirAll、exists→覆盖确认、size 格式化逻辑几乎逐字重复，仅 URL/文件名不同；后续修 bug 需改两处。
- **建议**: 抽取公共函数（如 ensureOverwrite、formatSize、downloadToFile）到 scripts/internal 共享包。

## scripts/flush_cache.go

### [MEDIUM] FlushAllCache 对共享 Redis 实例执行 FLUSHDB，会清空同库其他应用的缓存
- **位置**: 39-41  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: cache_service.FlushAllCache → cache.FlushAll → redisClient.FlushDB(ctx) 清空当前 DB。若面板与其他应用共用同一 Redis 实例/DB0（install.sh 推荐的 docker run -p 6379:6379 正是无密码共享场景），会把无关数据全部冲掉。
- **建议**: 为面板使用独立 Redis DB（REDIS_DB 配置，如 15），Flush 前校验 DB 号并提示；或改用按前缀 SCAN+DEL 只清本应用 key。

## scripts/init_knowledge.sql

### [HIGH] 脚本开头无条件 DELETE 清空知识库表，重跑即抹掉线上已有内容
- **位置**: 3-4  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: DELETE FROM knowledge_articles; DELETE FROM knowledge_categories; 没有任何条件或保护；运营人员手动编辑过的分类/文章在重复执行（migrate_new_features.sh 第 5 步会再次导入）时被整体清空。
- **建议**: 删除两个 DELETE，改为 INSERT OR IGNORE / ON CONFLICT 幂等写入；或显式提示并交互确认。

### [MEDIUM] datetime('now') 与硬编码分类 id 使脚本仅适用于全新 SQLite 库
- **位置**: 7-12  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: datetime('now') 是 SQLite 专属函数（MySQL 需 NOW()、PG 需 now()）；分类 id 硬编码 1-5，若表已有自增数据会主键冲突；articles 不带 id 依赖插入顺序，被 update_knowledge_tutorials.sql 的按 id 更新隐含耦合。
- **建议**: 改用 CURRENT_TIMESTAMP（三库通用）并去掉显式 id，用 name 唯一键 upsert；明确标注 SQLite-only 或提供多方言版本。

### [LOW] 知识库内容硬编码业务促销规则，易与实际运营配置冲突
- **位置**: 1-433  |  **类别**: maintainability  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 文章内容写死"充值满100送5%/满500送10%""邀请返20%佣金"等营销数字，若后台实际配置不同，用户看到的教程与真实规则不一致；内容与代码同源导致运营改文案需改 SQL 重发。
- **建议**: 把营销数字改为占位符或从系统设置动态渲染；文章建议迁移到后台可编辑的种子数据机制。

## scripts/migrate_new_features.sh

### [MEDIUM] 迁移脚本仅支持 SQLite（sqlite3 CLI），与项目多数据库支持不一致
- **位置**: 42-60, 137-154  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 全脚本使用 sqlite3 命令与 pragma_table_info，MySQL/PostgreSQL 部署无法使用该迁移；迁移体系割裂为 Go 的 AutoMigrate 与手写 SQL 两套，schema 变更难以同步。
- **建议**: 将新增表/字段迁移并入 GORM AutoMigrate 或提供 mysql/pg 版 SQL；sqlite3 专属脚本标注适用范围并给出替代路径。

### [MEDIUM] DROP TABLE 不处理外键引用，开启 foreign_keys 时迁移直接失败或级联删数据
- **位置**: 52-55  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 用户选择 y 后依次 DROP checkin_records/knowledge_articles/knowledge_categories/promotions；若已有 add_promotion_participations.sql 建立的 FK 引用 promotions（ON DELETE CASCADE），SQLite 在 PRAGMA foreign_keys=ON 下 DROP 会失败，静默中断迁移；同时删除决定没有二次确认就删整表。
- **建议**: DROP 前检查外键依赖并按依赖顺序处理；删除步骤改为逐表确认 + 显示将丢失的数据量。

### [LOW] 交互式 read 在非交互执行时按"否"跳过导入，行为不可预期
- **位置**: 159-187  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 步骤 5/6 的 read -p 无默认值，非交互管道执行时立即返回空值并按 [Nn] 分支跳过数据导入，而步骤 3 的表创建照常进行，造成"表存在但无初始数据"的半完成状态。
- **建议**: 为 read 提供默认值（如 "Y/n" 默认 Y）或增加 --yes/--skip-data 参数显式控制。

## scripts/migrations/add_promotion_participations.sql

### [MEDIUM] AUTOINCREMENT + VARCHAR/DECIMAL 混用，仅 SQLite 可直跑，MySQL/PostgreSQL 不兼容
- **位置**: 2-17  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: CREATE TABLE 使用 SQLite 的 AUTOINCREMENT 主键，同时使用 VARCHAR(50)/DECIMAL(10,2)（SQLite 类型亲和可接受，但 MySQL 需 AUTO_INCREMENT、PG 需 SERIAL/BIGSERIAL 与 NUMERIC），作为项目级迁移文件无法在其余两种受支持数据库执行。
- **建议**: 改由 GORM AutoMigrate 管理该表，或提供三方言迁移；保留此 SQL 时在文件头注明仅限 SQLite。

### [LOW] 缺少 (user_id, promotion_id) 唯一约束，重复参与依赖应用层兜底
- **位置**: 19-22  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 四个单列索引已建，但无 UNIQUE(user_id, promotion_id)，并发请求下应用层先查后插存在竞态，同一用户可对同一活动重复参与领取奖励。
- **建议**: 增加 UNIQUE INDEX idx_promotion_part_user_promo ON promotion_participations(user_id, promotion_id)（业务允许多次时改为 order_id 维度约束）。

## scripts/unlock_user.go

### [MEDIUM] 解锁时按 username/email 删除全部登录尝试记录，含成功记录，破坏审计线索
- **位置**: 104-107  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: DELETE FROM login_attempts WHERE username=? OR username=? 删除该用户所有成功+失败尝试（结果输出"清除登录记录: N 条（包括成功和失败的记录）"）；且用户名与邮箱可能命中其他同名用户的历史记录。审计/取证信息被无差别抹除。
- **建议**: 仅删除失败记录（success=false）且限定 user 归属（按 user_id 或用户唯一标识），保留成功登录历史。

### [LOW] 多个 DB 查询忽略错误返回值
- **位置**: 75-87, 109-120  |  **类别**: error-handling  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: Count(&failedAttempts)、Find(&recentAttempts)、Find(&loginHistories)、Find(&auditLogs) 均未检查 .Error；DB 异常时静默输出 0 条并继续执行删除，可能让管理员误以为已解锁。
- **建议**: 对关键查询检查 err 并给出明确失败提示，删除操作前再次确认。

## scripts/update_device_locations.go

### [HIGH] 绕过 config.LoadConfig 直接打开 ./cboard.db，MySQL/PostgreSQL 部署或自定义路径下静默操作空库
- **位置**: 23-27  |  **类别**: architecture  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 脚本硬编码 gorm.Open(sqlite.Open(filepath.Join(rootDir, "cboard.db")))。项目支持 MySQL/PostgreSQL（go.mod 含 driver/mysql、driver/postgres），此类部署下脚本会新建一个空的 cboard.db 文件并报告"找到 0 个设备"，管理员误以为已执行。与 admin_tool.go/unlock_user.go/flush_cache.go 用 config+database.InitDatabase 的方式不一致。
- **建议**: 改用 config.LoadConfig + database.InitDatabase() 统一初始化，从 DATABASE_URL 解析连接串，删除硬编码路径。

### [LOW] 逐设备 UPDATE，设备量大时产生 N 次单行写；无事务包裹
- **位置**: 54-75  |  **类别**: performance  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 循环内每台设备一次 db.Model(&device).Update("location", ...)，上万设备即上万条 SQL；且整个批处理无事务，中途失败会留下部分更新，与运行中的主服务并发写同一行还可能出现 SQLite busy。
- **建议**: 按 500 条一批 UPDATE ... WHERE id IN (...) 批量提交，并用事务包裹；优先在业务低峰执行。

## scripts/update_knowledge_tutorials.sql

### [MEDIUM] UPDATE 依赖硬编码 id（3/4/5/6/2），与 init_knowledge.sql 插入顺序强耦合
- **位置**: 42, 87, 140, 203, 248  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: UPDATE knowledge_articles SET content=... WHERE id = N 假设文章自增 id 恰好是 init_knowledge.sql 的插入顺序；只要初始数据顺序变动、或用户库中已有其他 id 的文章（先导入过其他版本数据），就会改错文章或更新 0 行，且脚本不校验 RowsAffected。
- **建议**: 改用稳定的业务键定位，如 WHERE category_id=2 AND title='Windows 使用教程'，或按标题先 SELECT id 再更新；更新后校验受影响行数并告警。

### [LOW] 验证查询使用 substr() 为 SQLite 专属，且脚本整体无事务
- **位置**: 250-251  |  **类别**: style  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: SELECT substr(content,1,50) 在 MySQL/PG 语义不同；整个文件多条 UPDATE 无 BEGIN/COMMIT，中途失败留下部分更新的知识库。
- **建议**: 用 LEFT()/通用截断或去掉验证查询；包裹事务并输出每条 UPDATE 的 RowsAffected。

## start.sh

### [HIGH] 缺 .env 时写入公开的固定 SECRET_KEY，JWT 签名密钥可被伪造
- **位置**: 32  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 自动生成的 .env 中 SECRET_KEY=change-me-to-a-strong-random-32-bytes-minimum-length 是公开常量；任何部署在未准备 .env 时启动都使用该密钥签发 JWT，攻击者可离线伪造任意用户/管理员 token。
- **建议**: .env 不存在时用 openssl rand / head -c 32 /dev/urandom 生成随机密钥；启动时校验 SECRET_KEY 不是示例值并拒绝启动。

### [MEDIUM] npm install 通过管道 tee|tail 判断成败，管道退出码恒为 0，重试分支永不触发
- **位置**: 346-393  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: if npm install --legacy-peer-deps 2>&1 | tee /tmp/npm_install.log | tail -30; then 的退出码取 tail（恒 0），安装失败也会 INSTALL_SUCCESS=true，整个三连重试/换源逻辑形同虚设，最终仅靠 vite 存在性兜底。
- **建议**: 使用 set -o pipefail 或先执行 npm install 记录 $? 再 tee；成功判定改为检查 $? 与 vite 可执行文件双重条件。

### [MEDIUM] 兜底逻辑强制安装 vite@4.5.0，无视 package.json 要求的版本
- **位置**: 400  |  **类别**: logic  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: npm install vite@4.5.0 --legacy-peer-deps 在找不到 vite 时硬装 4.5.0；若项目依赖 vite 5/6/7（plugin-vue 版本匹配），会降级或与依赖树冲突，反而制造新的坏环境。
- **建议**: 直接按 package.json 执行 npm ci/npm install；如确需固定版本，从 package.json devDependencies 读取。

### [MEDIUM] pkill -f "vite"/"bin/server" 误杀其他项目进程，且全局改写用户 npm registry 配置
- **位置**: 127-128, 315  |  **类别**: security  |  **来源组**: A10-router-scripts-deploy (路由/脚本/部署配置)
- **问题**: 启动前 pkill -f "vite" 会杀掉机器上所有 vite 进程（其他前端项目）；npm config set registry 无提示地改写用户级 .npmrc，离开本脚本后其他项目仍被指向 npmmirror，属于全局副作用。
- **建议**: 按 PID 文件与端口精确管理进程；registry 设置改为脚本内环境变量（npm_config_registry）而非持久化配置。
