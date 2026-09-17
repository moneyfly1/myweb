#!/usr/bin/env node
/**
 * 知识库内容种子脚本（幂等）
 *
 * 作用：把客户端教程正文与帮助中心原有的使用指南/常见问题迁入知识库，
 * 让运营可以在后台维护，前端页面只负责渲染（避免教程在多处硬编码）。
 *
 * 幂等策略：按「文章标题」在目标分类内查找，存在则 PUT 更新，不存在则 POST 新建。
 * 重复执行不会产生重复文章。
 *
 * 用法：
 *   node scripts/seed-knowledge.mjs --base https://dy.moneyfly.top \
 *        --user admin --pass '密码' [--dry-run]
 */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))

// ===== 参数 =====
const args = process.argv.slice(2)
const getArg = (name, fallback = '') => {
  const idx = args.indexOf(`--${name}`)
  return idx >= 0 && args[idx + 1] ? args[idx + 1] : fallback
}
const BASE = (getArg('base', process.env.MFY_BASE || 'https://dy.moneyfly.top')).replace(/\/$/, '')
const USER = getArg('user', process.env.MFY_USER || 'admin')
const PASS = getArg('pass', process.env.MFY_PASS || '')
const DRY_RUN = args.includes('--dry-run')

if (!PASS) {
  console.error('缺少管理员密码：--pass <password> 或环境变量 MFY_PASS')
  process.exit(1)
}

// ===== 补充文章：承接原帮助中心「使用指南 / 常见问题」的内容 =====
// 迁入知识库后，帮助中心不再硬编码这些文案
const SUPPLEMENTARY = [
  {
    category: '新手入门',
    title: '如何购买套餐与查看订单',
    summary: '从选套餐到支付完成，以及订单在哪里查看、如何继续支付',
    content: `<h2>购买套餐</h2>
<ol>
<li>进入「套餐购买」页面，选择普通套餐或自定义套餐。</li>
<li>确认设备数量、使用周期，如有优惠券可选择使用。</li>
<li>选择支付方式并完成支付。</li>
</ol>
<h2>查看订单</h2>
<ul>
<li>套餐购买、升级设备数量、账户充值都会进入「订单记录」。</li>
<li>订单记录中可查看状态、金额、时间等明细。</li>
<li>状态为「待支付」的订单可以继续支付，也可以取消。</li>
</ul>
<h2>常见问题</h2>
<ul>
<li><strong>支付成功但订单未更新：</strong>稍等 1-2 分钟后刷新页面；若仍未更新，请携带订单号提交工单。</li>
<li><strong>订单可以退款吗：</strong>请参考「退款政策」一文，或提交工单说明情况。</li>
</ul>`,
  },
  {
    category: '账户相关',
    title: '如何修改密码或邮箱',
    summary: '在个人资料中修改登录密码与邮箱，以及忘记密码的处理方式',
    content: `<h2>修改密码</h2>
<ol>
<li>进入「个人资料」或「用户设置」页面。</li>
<li>填写当前密码与新密码，保存。</li>
<li>保存成功后建议重新登录一次确认新密码可用。</li>
</ol>
<h2>修改邮箱</h2>
<ol>
<li>在「个人资料」页面填写新邮箱。</li>
<li>按提示输入邮箱验证码完成验证。</li>
<li>保存后系统会同步账号状态。</li>
</ol>
<h2>忘记密码</h2>
<ul>
<li>在登录页点击「忘记密码」，按邮件提示重置。</li>
<li>若收不到邮件，请检查垃圾邮件目录，或提交工单联系我们。</li>
</ul>`,
  },
  {
    category: '常见问题',
    title: '订阅地址可以分享给他人吗？',
    summary: '订阅地址的分享风险、设备数占用与泄露后的处理办法',
    content: `<p><strong>不建议分享。</strong>订阅地址与你的账户权益绑定，他人使用会占用你的设备数量。</p>
<ul>
<li>发现地址泄露或设备数被占满时，请在「订阅管理」中<strong>重置订阅地址</strong>。</li>
<li>重置后旧地址立即失效，需要在新设备上重新导入。</li>
<li>重置会清空已注册设备，请提前确认。</li>
</ul>`,
  },
  {
    category: '常见问题',
    title: '支持哪些客户端？怎么选？',
    summary: '自研客户端与第三方客户端的区别，以及按系统选择建议',
    content: `<h2>优先推荐</h2>
<p>优先使用本站自研的 <strong>MoneyFly</strong> 客户端：针对本站线路优化，复制订阅地址即可使用，无需繁琐配置。可在「客户端中心」下载。</p>
<h2>第三方客户端</h2>
<ul>
<li><strong>Windows：</strong>Clash Verge、Clash Part、V2rayN、Hiddify、FlClash</li>
<li><strong>macOS：</strong>Clash Verge、Clash Part、V2rayN、Hiddify、FlClash（注意区分 Apple 芯片与 Intel 版本）</li>
<li><strong>Android：</strong>Clash Meta、V2rayNG、Hiddify、FlClash</li>
<li><strong>iOS：</strong>Shadowrocket（需在 App Store 购买）</li>
</ul>
<h2>如何选择</h2>
<ul>
<li>只想省事：直接用 MoneyFly。</li>
<li>需要自定义分流规则：选 Clash 系列（Clash Verge / Clash Part）。</li>
<li>设备性能较弱或需要多协议：选 Hiddify、FlClash 或 V2rayN/V2rayNG。</li>
</ul>
<p>各客户端的详细安装与导入步骤，请到「客户端中心」查看对应教程。</p>`,
  },
]

// ===== 平台指引：原有的 4 篇「<平台> 使用教程」含逐客户端步骤，与客户端教程重复。
// 这里把标题改为「<平台> 客户端选择指引」并改写为"帮助选择客户端"的平台级内容，
// 就地更新（保留原文章 id），避免与客户端教程重复。
const PLATFORM_GUIDES = [
  {
    oldTitle: 'Windows 使用教程',
    newTitle: 'Windows 客户端选择指引',
    summary: 'Windows 平台有哪些客户端可选、怎么选，以及各自的详细教程入口',
    content: `<p>本页帮你挑选适合的 Windows 客户端；每个客户端的安装与导入订阅步骤，请到「客户端中心」查看对应教程。</p>
<h2>推荐顺序</h2>
<ol>
<li><strong>MoneyFly（本站自研）</strong>：省心首选，复制订阅地址即可使用。</li>
<li><strong>Clash Verge / Clash Part</strong>：需要自定义分流规则时选择。</li>
<li><strong>V2rayN</strong>：轻量、资源占用低，适合老机器。</li>
<li><strong>Hiddify / FlClash</strong>：跨平台，多设备统一体验。</li>
</ol>
<h2>如何安装</h2>
<ul>
<li>在「客户端中心」选择 Windows 标签，点击对应客户端的「下载」。</li>
<li>下载完成后双击安装包，按向导完成安装。</li>
<li>回到「订阅管理」复制订阅地址，在客户端中导入。</li>
</ul>
<p>遇到问题可先看「常见问题」分类，或提交工单。</p>`,
  },
  {
    oldTitle: 'macOS 使用教程',
    newTitle: 'macOS 客户端选择指引',
    summary: 'macOS 平台客户端选择、Apple 芯片与 Intel 版本区分说明',
    content: `<p>本页帮你挑选适合的 macOS 客户端；详细安装步骤请到「客户端中心」查看对应教程。</p>
<h2>先确认芯片类型</h2>
<ul>
<li>点左上角苹果菜单 →「关于本机」，查看「芯片」或「处理器」。</li>
<li><strong>Apple 芯片</strong>（M1/M2/M3/M4）下载 ARM 版本；<strong>Intel</strong> 下载 x64 版本。</li>
<li>在「客户端中心」的下载按钮上会区分「Apple 芯片」与「Intel」两个选项。</li>
</ul>
<h2>推荐顺序</h2>
<ol>
<li><strong>MoneyFly（本站自研）</strong>：省心首选。</li>
<li><strong>Clash Verge / Clash Part</strong>：需要自定义规则时选择。</li>
<li><strong>Hiddify / FlClash</strong>：跨平台统一体验。</li>
</ol>
<h2>首次打开被拦截</h2>
<ul>
<li>提示「无法验证开发者」时：在「应用程序」中右键点击该应用 →「打开」→ 再次确认「打开」。</li>
<li>仍被拦截时：到「系统设置 → 隐私与安全性」中点击「仍要打开」。</li>
</ul>`,
  },
  {
    oldTitle: 'iOS 使用教程',
    newTitle: 'iOS 客户端选择指引',
    summary: 'iOS 平台客户端选择与订阅导入方式说明',
    content: `<p>本页帮你挑选适合的 iOS 客户端；详细安装步骤请到「客户端中心」查看对应教程。</p>
<h2>可选客户端</h2>
<ul>
<li><strong>Shadowrocket</strong>：iOS 上最常用的客户端，需要在 App Store 购买后下载（付费应用，请自行确认区域账号可用）。</li>
</ul>
<h2>导入订阅</h2>
<ul>
<li>在「订阅管理」复制订阅地址。</li>
<li>打开 Shadowrocket，点右上角「+」→ 类型选择「Subscribe」→ 粘贴地址保存。</li>
<li>点「更新」获取节点，选择节点后开启连接。</li>
</ul>
<p>iOS 不支持本站自研的 MoneyFly 客户端（暂未提供 iOS 版本）。</p>`,
  },
  {
    oldTitle: 'Android 使用教程',
    newTitle: 'Android 客户端选择指引',
    summary: 'Android 平台客户端选择、APK 安装与订阅导入说明',
    content: `<p>本页帮你挑选适合的 Android 客户端；详细安装步骤请到「客户端中心」查看对应教程。</p>
<h2>推荐顺序</h2>
<ol>
<li><strong>MoneyFly（本站自研）</strong>：省心首选，复制订阅即可使用。</li>
<li><strong>Clash Meta / V2rayNG</strong>：常用且稳定。</li>
<li><strong>Hiddify / FlClash</strong>：跨平台统一体验。</li>
</ol>
<h2>安装 APK 注意事项</h2>
<ul>
<li>在手机「设置」中允许当前浏览器或文件管理器「安装未知应用」。</li>
<li>点击已下载的 APK 完成安装，首次启动按提示授予网络权限。</li>
</ul>
<h2>导入订阅</h2>
<ul>
<li>在「订阅管理」复制订阅地址。</li>
<li>在客户端中选择「从剪贴板导入」或「添加订阅」并粘贴地址。</li>
</ul>`,
  },
]

// ===== 存量文章纠错：原内容与实际业务不符（支付方式、充值优惠、佣金提现、
// 流量额度、套餐等级、客服时间等），逐篇改写为与系统实际一致。
// 按分类 + 标题就地更新（保留原文章 id，不产生重复文章）。
const LEGACY_FIXES = [
  {
    category: '新手入门',
    title: '什么是代理服务？',
    summary: '代理服务的原理、用途与本站服务特点',
    content: `<h2>什么是代理服务？</h2>
<p>代理服务是一种网络服务，它充当您的设备和互联网之间的中介。通过代理服务器，您可以：</p>
<ul>
<li>访问被限制的网站和服务</li>
<li>降低真实 IP 暴露的风险</li>
<li>改善跨境访问的稳定性</li>
<li>绕过地理位置限制</li>
</ul>
<h2>工作原理</h2>
<p>当您使用代理服务时，您的网络请求会先发送到代理服务器，然后由代理服务器代表您访问目标网站。这样，目标网站看到的是代理服务器的 IP 地址，而不是您的真实 IP。</p>
<h2>本站服务特点</h2>
<ul>
<li><strong>线路优化</strong>：采用优质线路，跨境访问更稳定</li>
<li><strong>加密传输</strong>：客户端与服务端之间加密传输</li>
<li><strong>多端支持</strong>：支持 Windows、macOS、iOS、Android，并提供自研客户端 MoneyFly</li>
<li><strong>按设备数计费</strong>：套餐按同时在线设备数量与时长计费，不限制流量</li>
<li><strong>客服支持</strong>：联系方式与服务时间请见「帮助中心 → 联系我们」</li>
</ul>
<h2>需要准备什么</h2>
<ol>
<li>注册账号并购买套餐</li>
<li>在「客户端中心」下载合适的客户端（推荐自研 MoneyFly）</li>
<li>在「订阅管理」复制订阅地址，导入客户端即可使用</li>
</ol>`,
  },
  {
    category: '新手入门',
    title: '如何开始使用？',
    summary: '从注册到导入订阅的完整起步流程',
    content: `<h2>快速开始指南</h2>
<h3>第一步：注册账户</h3>
<p>访问本站，填写邮箱和密码完成注册。建议使用常用邮箱，以便接收到期提醒等重要通知。</p>
<h3>第二步：购买套餐</h3>
<ol>
<li>登录后进入「套餐购买」页面。</li>
<li>按需选择套餐：本站套餐以<strong>同时在线设备数量</strong>与<strong>使用时长</strong>区分，具体价格与档位以页面显示为准。</li>
<li>选择支付方式，完成支付后订阅会自动开通。</li>
</ol>
<h3>第三步：下载客户端</h3>
<p>在「客户端中心」按系统下载客户端：</p>
<ul>
<li><strong>Windows：</strong>MoneyFly（自研，推荐）、Clash Verge、Clash Part、V2rayN、Hiddify、FlClash</li>
<li><strong>macOS：</strong>MoneyFly（自研，推荐）、Clash Verge、Clash Part、V2rayN、Hiddify、FlClash（注意区分 Apple 芯片与 Intel 版本）</li>
<li><strong>Android：</strong>MoneyFly（自研，推荐）、Clash Meta、V2rayNG、Hiddify、FlClash</li>
<li><strong>iOS：</strong>Shadowrocket（App Store 付费应用）</li>
</ul>
<h3>第四步：导入订阅</h3>
<ol>
<li>进入「订阅管理」页面，复制订阅地址（或使用页面上的二维码）。</li>
<li>在客户端中「添加订阅 / 从剪贴板导入」，粘贴地址并保存。</li>
<li>选择节点并开启连接即可使用。</li>
</ol>
<p>各客户端的详细步骤见「客户端中心」中对应教程；遇到问题可先看「常见问题」分类或提交工单。</p>`,
  },
  {
    category: '常见问题',
    title: '无法连接怎么办？',
    summary: '按顺序排查订阅、设备数、节点与本地网络问题',
    content: `<h2>无法连接的排查顺序</h2>
<ol>
<li><strong>确认订阅状态</strong>：登录本站查看「订阅管理」，确认套餐未过期、账号状态正常。</li>
<li><strong>确认设备数量未超限</strong>：在「设备管理」查看在线设备数是否已达套餐上限；若已满，请移除不再使用的设备或在仪表盘升级设备数量。</li>
<li><strong>更新订阅</strong>：在客户端中执行「更新订阅」，获取最新节点。</li>
<li><strong>切换节点</strong>：当前节点可能临时不可用，换一个节点重试。</li>
<li><strong>检查本地网络</strong>：确认设备本身能正常上网（可先打开一个国内网站验证）。</li>
<li><strong>关闭其它代理软件</strong>：多个代理软件同时运行会互相冲突。</li>
</ol>
<h2>分系统排查</h2>
<h3>Windows</h3>
<ul>
<li>检查客户端是否已开启「系统代理」</li>
<li>尝试以管理员身份运行客户端</li>
<li>检查杀毒软件 / 防火墙是否拦截</li>
</ul>
<h3>macOS</h3>
<ul>
<li>检查「系统设置 → 网络 → 代理」是否被其它软件改动</li>
<li>在客户端中重新开启系统代理</li>
</ul>
<h3>Android / iOS</h3>
<ul>
<li>确认已在系统中允许 VPN 配置</li>
<li>尝试重启客户端或设备</li>
<li>关闭省电模式（可能限制后台运行）</li>
</ul>
<p>以上都无效时，请提交工单并附上：订阅状态截图、客户端名称与版本、报错提示，便于我们快速定位。</p>`,
  },
  {
    category: '常见问题',
    title: '速度慢怎么办？',
    summary: '从节点选择、客户端设置到本地网络逐项优化',
    content: `<h2>提升连接速度的方法</h2>
<h3>选择合适的节点</h3>
<ul>
<li><strong>使用测速功能</strong>：多数客户端支持测速，优先选择延迟低的节点</li>
<li><strong>选择距离较近的节点</strong>：通常延迟更低</li>
<li><strong>避开高峰时段</strong>：晚间为使用高峰，可换时段再试</li>
</ul>
<h3>优化客户端设置</h3>
<ul>
<li>使用「规则模式」而非「全局模式」，国内网站直连可以明显提速</li>
<li>开启客户端的「自动选择 / 延迟测速」功能</li>
<li>关闭不需要的功能（如 UDP 转发、广告拦截）后再对比</li>
</ul>
<h3>检查本地网络</h3>
<ul>
<li>关闭其它占用带宽的程序（下载、视频、云盘同步等）</li>
<li>尽量使用有线网络，或靠近路由器使用 WiFi</li>
<li>重启路由器后再测试</li>
</ul>
<h3>需要更高带宽</h3>
<p>如果您对速度有更高要求，可到「套餐购买」查看是否有更合适的档位（例如专线类套餐）。具体以页面展示为准，也可提交工单咨询。</p>`,
  },
  {
    category: '账户相关',
    title: '如何充值？',
    summary: '账户充值的支付方式、到账时间与余额用途',
    content: `<h2>充值步骤</h2>
<ol>
<li>登录本站，进入「账户充值」（或仪表盘的充值入口）。</li>
<li>输入充值金额。</li>
<li>选择支付方式（以页面当前展示的可选项为准，例如支付宝）。</li>
<li>完成支付，余额自动到账。</li>
</ol>
<h2>关于支付方式</h2>
<ul>
<li>可用的支付方式由本站后台配置决定，请以充值页面实际显示的选项为准。</li>
<li>若某种方式暂不可用，说明该渠道当前未开放，可先用其它方式或联系客服。</li>
</ul>
<h2>余额用途</h2>
<ul>
<li>余额可用于购买套餐、升级设备数量等站内消费。</li>
<li>余额支付会即时到账并直接抵扣订单金额。</li>
<li><strong>余额不支持提现</strong>，也不能转赠他人。</li>
</ul>
<h2>充值与订单记录</h2>
<ul>
<li>充值记录与订单记录都在「订单记录」页面，可按时间查看金额与状态。</li>
<li>支付成功但余额未到账时，请稍等 1-2 分钟刷新；仍未到账请携带订单号提交工单。</li>
</ul>
<h2>充值优惠</h2>
<p>本站<strong>没有固定的充值赠送规则</strong>。如遇活动优惠，会在站内公告或活动页面明确说明，以活动页面为准，请勿相信其它渠道的"充值返利"说法。</p>`,
  },
  {
    category: '账户相关',
    title: '邀请奖励说明',
    summary: '邀请码奖励的发放条件、金额来源与到账方式',
    content: `<h2>如何邀请好友</h2>
<ol>
<li>登录本站，进入「我的邀请」页面。</li>
<li>生成或复制您的专属邀请码 / 邀请链接。</li>
<li>把邀请码发给好友，好友注册时填写即可绑定邀请关系。</li>
</ol>
<h2>奖励规则</h2>
<ul>
<li><strong>奖励金额由邀请码上设置的奖励决定</strong>：每个邀请码可单独设置邀请人奖励与被邀请人奖励，具体金额请以「我的邀请」页面显示为准。</li>
<li><strong>发放条件</strong>：被邀请人<strong>首次购买套餐成功后</strong>才发放奖励（注册本身不发放）；若邀请码设置了"最低订单金额"，订单需达到该金额；若开启了"仅限新用户"，则仅对从未购买过的用户生效。</li>
<li><strong>发放方式</strong>：奖励直接发放到<strong>账户余额</strong>，可在「余额明细」中看到类型为"佣金"的记录。</li>
<li><strong>一次性发放</strong>：同一邀请关系只发放一次，不会按好友后续消费比例持续返佣。</li>
</ul>
<h2>奖励用途</h2>
<ul>
<li>奖励到账后即为余额，可用于购买套餐、升级设备数量。</li>
<li><strong>余额不支持提现</strong>，也不能转赠他人。</li>
</ul>
<h2>注意事项</h2>
<ul>
<li>不允许自己邀请自己（同一账号/同一支付方式等会被判定为异常）。</li>
<li>禁止刷单、批量注册等作弊行为，违规所得将被撤销。</li>
<li>对奖励有疑问时，请提交工单并附上邀请码与被邀请人账号。</li>
</ul>`,
  },
  {
    category: '账户相关',
    title: '退款政策',
    summary: '退款申请方式、审核流程与不予退款的情形',
    content: `<h2>申请方式</h2>
<p>本站为虚拟订阅服务，如需退款请<strong>提交工单</strong>，写明订单号、退款原因与联系方式，由客服人工核实处理。</p>
<h2>审核流程</h2>
<ol>
<li>提交工单说明情况。</li>
<li>客服核实订单与订阅使用情况（如开通时间、设备使用记录等）。</li>
<li>审核通过后按原支付渠道或双方确认的方式退回。</li>
<li>到账时间以支付渠道为准，通常为 1-7 个工作日。</li>
</ol>
<h2>可能不予退款的情形</h2>
<ul>
<li>订阅已实际使用较长时间（以客服核实结果为准）。</li>
<li>存在违规使用、共享账号、恶意退款等行为。</li>
<li>通过活动 / 优惠券获得的部分优惠金额，不予退还。</li>
</ul>
<h2>注意事项</h2>
<ul>
<li>退款金额以该订单实际支付金额为准；使用余额支付的部分退回余额。</li>
<li>退款成功后，对应的订阅时长与设备数量会被收回，请知悉。</li>
<li>如对审核结果有异议，可在工单中补充说明，我们会再次核实。</li>
</ul>`,
  },
]

function buildArticles() {
  const tutorialsPath = join(__dirname, 'client-tutorials.content.json')
  const tutorials = JSON.parse(readFileSync(tutorialsPath, 'utf8'))

  const articles = tutorials.map(item => ({
    category: '客户端教程',
    title: item.title,
    summary: item.summary,
    content: item.content,
    clientId: item.clientId,
  }))

  return [...articles, ...SUPPLEMENTARY]
}

// ===== HTTP =====
let token = ''

async function request(method, path, body) {
  const res = await fetch(`${BASE}/api/v1${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  let json = null
  try {
    json = JSON.parse(text)
  } catch {
    throw new Error(`${method} ${path} 返回非 JSON (${res.status}): ${text.slice(0, 200)}`)
  }
  if (!res.ok || json.success === false) {
    throw new Error(`${method} ${path} 失败 (${res.status}): ${json.message || text.slice(0, 200)}`)
  }
  return json.data ?? json
}

async function login() {
  const data = await request('POST', '/auth/login-json', { username: USER, password: PASS })
  token = data?.access_token || data?.token || ''
  if (!token) throw new Error('登录未返回 access_token')
  console.log('✅ 管理员登录成功')
}

async function ensureCategory(name) {
  const categories = await request('GET', '/admin/knowledge/categories')
  const list = Array.isArray(categories) ? categories : categories?.items || categories?.list || []
  const hit = list.find(c => String(c.name || '').trim() === name)
  if (hit) return hit.id
  const created = await request('POST', '/admin/knowledge/categories', {
    name,
    icon: 'document',
    sort_order: list.length + 1,
    is_active: true,
  })
  console.log(`  ➕ 新建分类：${name} (id=${created.id})`)
  return created.id
}

// 管理端文章列表只支持 category_id 分页（无 keyword），故按分类拉取后本地按标题匹配
async function findArticleByTitle(categoryId, title) {
  const pageSize = 100
  for (let page = 1; page <= 10; page += 1) {
    const data = await request(
      'GET',
      `/admin/knowledge/articles?category_id=${categoryId}&page=${page}&page_size=${pageSize}`
    )
    const list = data?.list || data?.items || (Array.isArray(data) ? data : [])
    const hit = list.find(a => String(a.title || '').trim() === title)
    if (hit) return hit
    if (list.length < pageSize) return null
  }
  return null
}

async function upsertArticle(article, categoryId) {
  const hit = await findArticleByTitle(categoryId, article.title)

  const payload = {
    category_id: categoryId,
    title: article.title,
    summary: article.summary,
    content: article.content,
    is_active: true,
  }

  if (hit) {
    await request('PUT', `/admin/knowledge/articles/${hit.id}`, payload)
    return { action: 'updated', id: hit.id }
  }
  const created = await request('POST', '/admin/knowledge/articles', payload)
  return { action: 'created', id: created?.id }
}

async function main() {
  const articles = buildArticles()
  console.log(`准备写入 ${articles.length} 篇文章（教程 10 篇 + 补充 ${SUPPLEMENTARY.length} 篇）`)
  if (DRY_RUN) {
    articles.forEach(a => console.log(`  [dry-run] ${a.category} / ${a.title} (${a.content.length} 字)`))
    console.log(`  另有 ${PLATFORM_GUIDES.length} 篇平台文章将被改写为「客户端选择指引」（就地重命名）`)
    console.log(`  以及 ${LEGACY_FIXES.length} 篇存量文章将被内容纠错：${LEGACY_FIXES.map(f => f.title).join('、')}`)
    return
  }

  await login()
  const categoryIds = new Map()
  const results = { created: 0, updated: 0, failed: 0, guides: 0, fixed: 0 }

  for (const article of articles) {
    try {
      if (!categoryIds.has(article.category)) {
        categoryIds.set(article.category, await ensureCategory(article.category))
      }
      const { action, id } = await upsertArticle(article, categoryIds.get(article.category))
      results[action] += 1
      console.log(`  ${action === 'created' ? '➕' : '♻️ '} [${article.category}] ${article.title} (id=${id})`)
    } catch (err) {
      results.failed += 1
      console.error(`  ❌ [${article.category}] ${article.title}: ${err.message}`)
    }
  }

  // 存量文章纠错（支付方式 / 充值优惠 / 佣金提现 / 流量额度 / 套餐等级 / 客服时间等）
  for (const fix of LEGACY_FIXES) {
    try {
      if (!categoryIds.has(fix.category)) {
        categoryIds.set(fix.category, await ensureCategory(fix.category))
      }
      const { action, id } = await upsertArticle(fix, categoryIds.get(fix.category))
      results.fixed += 1
      console.log(`  🩹 [${fix.category}] 内容纠错: ${fix.title} (${action}, id=${id})`)
    } catch (err) {
      results.failed += 1
      console.error(`  ❌ 内容纠错失败 ${fix.title}: ${err.message}`)
    }
  }

  // 平台文章改写（消除与客户端教程的重复）
  const tutorialCategoryId = categoryIds.get('客户端教程')
  for (const guide of PLATFORM_GUIDES) {
    try {
      const hit = await findArticleByTitle(tutorialCategoryId, guide.oldTitle)
      if (!hit) {
        console.log(`  ⏭️  未找到平台文章「${guide.oldTitle}」，跳过`)
        continue
      }
      await request('PUT', `/admin/knowledge/articles/${hit.id}`, {
        category_id: tutorialCategoryId,
        title: guide.newTitle,
        summary: guide.summary,
        content: guide.content,
        is_active: true,
      })
      results.guides += 1
      console.log(`  📝 [客户端教程] ${guide.oldTitle} → ${guide.newTitle} (id=${hit.id})`)
    } catch (err) {
      results.failed += 1
      console.error(`  ❌ 平台文章改写失败 ${guide.oldTitle}: ${err.message}`)
    }
  }

  console.log(
    `\n完成：新建 ${results.created}，更新 ${results.updated}，内容纠错 ${results.fixed}，平台文章改写 ${results.guides}，失败 ${results.failed}`
  )
  if (results.failed > 0) process.exitCode = 1
}

main().catch(err => {
  console.error('执行失败:', err.message)
  process.exit(1)
})
