-- 知识库初始化数据
-- 清空现有数据
DELETE FROM knowledge_articles;
DELETE FROM knowledge_categories;

-- 插入分类
INSERT INTO knowledge_categories (id, name, icon, sort_order, is_active, created_at, updated_at) VALUES
(1, '新手入门', 'el-icon-guide', 1, 1, datetime('now'), datetime('now')),
(2, '客户端教程', 'el-icon-mobile-phone', 2, 1, datetime('now'), datetime('now')),
(3, '常见问题', 'el-icon-question', 3, 1, datetime('now'), datetime('now')),
(4, '进阶使用', 'el-icon-star-on', 4, 1, datetime('now'), datetime('now')),
(5, '账户相关', 'el-icon-user', 5, 1, datetime('now'), datetime('now'));

-- 插入文章
INSERT INTO knowledge_articles (category_id, title, content, summary, sort_order, is_active, view_count, created_at, updated_at) VALUES
-- 新手入门
(1, '什么是代理服务？', '<h2>什么是代理服务？</h2>
<p>代理服务是一种网络服务，它充当您的设备和互联网之间的中介。通过代理服务器，您可以：</p>
<ul>
<li>访问被限制的网站和服务</li>
<li>保护您的隐私和数据安全</li>
<li>提升网络访问速度</li>
<li>绕过地理位置限制</li>
</ul>
<h3>工作原理</h3>
<p>当您使用代理服务时，您的网络请求会先发送到代理服务器，然后由代理服务器代表您访问目标网站。这样，目标网站看到的是代理服务器的IP地址，而不是您的真实IP。</p>
<h3>我们的优势</h3>
<ul>
<li><strong>高速稳定</strong>：采用优质线路，保证连接速度和稳定性</li>
<li><strong>安全加密</strong>：使用先进的加密技术，保护您的数据安全</li>
<li><strong>多设备支持</strong>：支持Windows、Mac、iOS、Android等多种设备</li>
<li><strong>24/7客服</strong>：随时为您解决使用中的问题</li>
</ul>', '了解代理服务的基本概念和工作原理', 1, 1, 0, datetime('now'), datetime('now')),

(1, '如何开始使用？', '<h2>快速开始指南</h2>
<h3>第一步：注册账户</h3>
<p>访问我们的网站，点击"注册"按钮，填写邮箱和密码完成注册。建议使用常用邮箱，以便接收重要通知。</p>
<h3>第二步：选择套餐</h3>
<p>登录后，进入"套餐购买"页面，根据您的需求选择合适的套餐：</p>
<ul>
<li><strong>基础套餐</strong>：适合轻度使用，日常浏览网页</li>
<li><strong>标准套餐</strong>：适合中度使用，观看视频、下载文件</li>
<li><strong>高级套餐</strong>：适合重度使用，4K视频、大文件传输</li>
</ul>
<h3>第三步：下载客户端</h3>
<p>根据您的设备类型，下载对应的客户端：</p>
<ul>
<li>Windows：推荐使用 Clash for Windows 或 v2rayN</li>
<li>Mac：推荐使用 ClashX 或 Surge</li>
<li>iOS：推荐使用 Shadowrocket 或 Quantumult X</li>
<li>Android：推荐使用 Clash for Android 或 v2rayNG</li>
</ul>
<h3>第四步：导入订阅</h3>
<p>在"订阅管理"页面复制您的订阅链接，然后在客户端中导入订阅，即可开始使用。</p>', '从注册到使用的完整流程指南', 2, 1, 0, datetime('now'), datetime('now')),

-- 客户端教程
(2, 'Windows 使用教程', '<h2>Windows 客户端使用教程</h2>
<h3>推荐客户端：Clash for Windows</h3>
<h4>下载安装</h4>
<ol>
<li>访问软件教程页面，下载 Clash for Windows 最新版本</li>
<li>解压下载的压缩包到任意文件夹</li>
<li>运行 Clash for Windows.exe</li>
</ol>
<h4>导入订阅</h4>
<ol>
<li>复制您的订阅链接（在"订阅管理"页面）</li>
<li>打开 Clash for Windows，点击左侧"Profiles"</li>
<li>在顶部输入框粘贴订阅链接，点击"Download"</li>
<li>等待订阅更新完成</li>
</ol>
<h4>开始使用</h4>
<ol>
<li>点击左侧"Proxies"</li>
<li>选择"规则模式"（Rule）</li>
<li>在节点列表中选择一个节点</li>
<li>点击左侧"General"，打开"System Proxy"开关</li>
<li>现在您可以正常访问网络了</li>
</ol>
<h4>常见问题</h4>
<p><strong>Q: 无法连接？</strong><br>
A: 检查是否开启了系统代理，尝试切换其他节点。</p>
<p><strong>Q: 速度慢？</strong><br>
A: 尝试切换到延迟更低的节点，或使用测速功能选择最快节点。</p>', 'Clash for Windows 详细使用教程', 1, 1, 0, datetime('now'), datetime('now')),

(2, 'macOS 使用教程', '<h2>macOS 客户端使用教程</h2>
<h3>推荐客户端：ClashX</h3>
<h4>下载安装</h4>
<ol>
<li>访问软件教程页面，下载 ClashX 最新版本</li>
<li>打开下载的 dmg 文件</li>
<li>将 ClashX 拖动到应用程序文件夹</li>
<li>首次运行需要在"系统偏好设置 - 安全性与隐私"中允许</li>
</ol>
<h4>导入订阅</h4>
<ol>
<li>复制您的订阅链接</li>
<li>点击菜单栏的 ClashX 图标</li>
<li>选择"配置 - 托管配置 - 管理"</li>
<li>点击"添加"，粘贴订阅链接</li>
<li>点击"确定"，等待订阅更新</li>
</ol>
<h4>开始使用</h4>
<ol>
<li>点击菜单栏的 ClashX 图标</li>
<li>选择"设置为系统代理"</li>
<li>选择"出站模式 - 规则判断"</li>
<li>在"Proxy"中选择一个节点</li>
<li>现在您可以正常访问网络了</li>
</ol>
<h4>提示</h4>
<p>建议开启"开机启动"和"自动更新订阅"，以获得最佳使用体验。</p>', 'ClashX 详细使用教程', 2, 1, 0, datetime('now'), datetime('now')),

(2, 'iOS 使用教程', '<h2>iOS 客户端使用教程</h2>
<h3>推荐客户端：Shadowrocket</h3>
<h4>下载安装</h4>
<p>Shadowrocket 是付费应用（约$2.99），需要使用美区 Apple ID 在 App Store 下载。</p>
<blockquote>
<p>提示：如果您没有美区账号，可以在淘宝搜索"美区 Apple ID"购买，或使用我们提供的共享账号（仅用于下载，下载后请退出）。</p>
</blockquote>
<h4>导入订阅</h4>
<ol>
<li>复制您的订阅链接</li>
<li>打开 Shadowrocket</li>
<li>点击右上角"+"</li>
<li>选择"类型 - Subscribe"</li>
<li>在"URL"中粘贴订阅链接</li>
<li>点击"完成"</li>
</ol>
<h4>开始使用</h4>
<ol>
<li>在首页选择一个节点</li>
<li>点击顶部的连接开关</li>
<li>首次使用需要允许添加 VPN 配置</li>
<li>连接成功后，状态栏会显示 VPN 图标</li>
</ol>
<h4>配置建议</h4>
<ul>
<li>全局路由：建议选择"配置"模式</li>
<li>DNS：建议使用"系统"或"1.1.1.1"</li>
<li>开启"连接时测试"，自动选择最快节点</li>
</ul>', 'Shadowrocket 详细使用教程', 3, 1, 0, datetime('now'), datetime('now')),

(2, 'Android 使用教程', '<h2>Android 客户端使用教程</h2>
<h3>推荐客户端：Clash for Android</h3>
<h4>下载安装</h4>
<ol>
<li>访问软件教程页面，下载 Clash for Android APK</li>
<li>允许安装未知来源应用</li>
<li>安装并打开应用</li>
</ol>
<h4>导入订阅</h4>
<ol>
<li>复制您的订阅链接</li>
<li>打开 Clash for Android</li>
<li>点击"配置"</li>
<li>点击右上角"+"</li>
<li>选择"URL"</li>
<li>粘贴订阅链接，点击"保存"</li>
<li>等待订阅更新完成</li>
</ol>
<h4>开始使用</h4>
<ol>
<li>返回首页，点击"点击启动"</li>
<li>首次使用需要允许创建 VPN 连接</li>
<li>点击"代理"选择节点</li>
<li>连接成功后，通知栏会显示运行状态</li>
</ol>
<h4>模式说明</h4>
<ul>
<li><strong>全局模式</strong>：所有流量都通过代理</li>
<li><strong>规则模式</strong>：根据规则自动分流（推荐）</li>
<li><strong>直连模式</strong>：不使用代理</li>
</ul>', 'Clash for Android 详细使用教程', 4, 1, 0, datetime('now'), datetime('now')),

-- 常见问题
(3, '无法连接怎么办？', '<h2>无法连接的解决方案</h2>
<h3>检查清单</h3>
<ol>
<li><strong>检查订阅是否过期</strong>
<p>登录网站查看订阅状态，确保套餐未过期且有剩余流量。</p>
</li>
<li><strong>更新订阅</strong>
<p>在客户端中更新订阅配置，获取最新的节点信息。</p>
</li>
<li><strong>切换节点</strong>
<p>当前节点可能暂时不可用，尝试切换到其他节点。</p>
</li>
<li><strong>检查网络连接</strong>
<p>确保您的设备已连接到互联网。</p>
</li>
<li><strong>关闭其他代理软件</strong>
<p>多个代理软件同时运行可能导致冲突。</p>
</li>
<li><strong>检查防火墙设置</strong>
<p>某些防火墙可能会阻止代理连接，尝试临时关闭防火墙测试。</p>
</li>
</ol>
<h3>高级排查</h3>
<h4>Windows</h4>
<ul>
<li>检查系统代理设置是否正确</li>
<li>尝试以管理员身份运行客户端</li>
<li>检查是否有杀毒软件拦截</li>
</ul>
<h4>macOS</h4>
<ul>
<li>检查系统偏好设置中的网络代理配置</li>
<li>尝试重置网络设置</li>
</ul>
<h4>移动设备</h4>
<ul>
<li>检查 VPN 配置是否正确安装</li>
<li>尝试重启设备</li>
<li>检查是否开启了省电模式（可能限制后台运行）</li>
</ul>
<p>如果以上方法都无法解决问题，请联系客服获取帮助。</p>', '详细的连接问题排查指南', 1, 1, 0, datetime('now'), datetime('now')),

(3, '速度慢怎么办？', '<h2>提升连接速度的方法</h2>
<h3>选择合适的节点</h3>
<ul>
<li><strong>使用测速功能</strong>：大多数客户端都有测速功能，选择延迟最低的节点</li>
<li><strong>选择地理位置近的节点</strong>：通常距离越近，速度越快</li>
<li><strong>避开高峰期</strong>：晚上8-11点是使用高峰，可以选择其他时段</li>
</ul>
<h3>优化客户端设置</h3>
<h4>Clash 系列</h4>
<ul>
<li>开启"增强模式"</li>
<li>调整"混合端口"设置</li>
<li>使用"自动选择"功能</li>
</ul>
<h4>Shadowrocket</h4>
<ul>
<li>算法选择"chacha20-ietf-poly1305"</li>
<li>开启"TCP 快速打开"</li>
<li>关闭"UDP 转发"（如不需要）</li>
</ul>
<h3>检查本地网络</h3>
<ul>
<li>确保本地网络连接稳定</li>
<li>关闭其他占用带宽的程序</li>
<li>使用有线连接代替 WiFi（如可能）</li>
<li>重启路由器</li>
</ul>
<h3>升级套餐</h3>
<p>如果您经常需要高速连接，建议升级到更高级的套餐，享受更快的速度和更稳定的连接。</p>', '多种方法提升连接速度', 2, 1, 0, datetime('now'), datetime('now')),

(3, '设备数量限制说明', '<h2>设备数量限制说明</h2>
<h3>什么是设备限制？</h3>
<p>为了保证服务质量和防止账号共享，每个套餐都有同时在线设备数量的限制。例如，如果您的套餐限制为3台设备，那么最多可以同时在3台设备上使用。</p>
<h3>如何查看设备数量？</h3>
<ol>
<li>登录网站</li>
<li>进入"设备管理"页面</li>
<li>查看当前在线设备和设备限制</li>
</ol>
<h3>超出限制怎么办？</h3>
<p>如果您的在线设备数量达到限制，新设备将无法连接。您可以：</p>
<ul>
<li><strong>断开不使用的设备</strong>：在设备管理页面删除不再使用的设备</li>
<li><strong>升级套餐</strong>：选择支持更多设备的套餐</li>
<li><strong>购买设备升级包</strong>：单独增加设备数量限制</li>
</ul>
<h3>设备识别</h3>
<p>系统会自动识别设备类型（Windows、Mac、iOS、Android等）。如果设备信息显示不正确，不影响正常使用。</p>
<h3>注意事项</h3>
<ul>
<li>路由器算作一台设备</li>
<li>同一设备重新连接不会增加设备数量</li>
<li>长时间未使用的设备会自动下线</li>
</ul>', '了解设备数量限制和管理方法', 3, 1, 0, datetime('now'), datetime('now')),

-- 进阶使用
(4, '路由器配置教程', '<h2>路由器配置教程</h2>
<h3>为什么要配置路由器？</h3>
<p>在路由器上配置代理后，连接到该路由器的所有设备都可以自动使用代理，无需在每台设备上单独配置。</p>
<h3>支持的路由器</h3>
<ul>
<li>梅林固件（Merlin）</li>
<li>OpenWrt</li>
<li>老毛子固件（Padavan）</li>
</ul>
<h3>梅林固件配置</h3>
<ol>
<li>安装 Clash 插件</li>
<li>进入插件设置页面</li>
<li>粘贴订阅链接</li>
<li>选择运行模式（推荐"游戏模式"或"全局模式"）</li>
<li>应用设置并重启</li>
</ol>
<h3>OpenWrt 配置</h3>
<ol>
<li>安装 PassWall 或 SSR Plus 插件</li>
<li>进入插件配置页面</li>
<li>添加订阅</li>
<li>配置分流规则</li>
<li>启用服务</li>
</ol>
<h3>注意事项</h3>
<ul>
<li>路由器配置较为复杂，建议有一定技术基础</li>
<li>配置错误可能导致无法上网，请提前备份路由器设置</li>
<li>路由器性能会影响代理速度</li>
<li>路由器算作一台设备</li>
</ul>
<p>如需详细配置指导，请联系客服获取帮助。</p>', '在路由器上配置代理服务', 1, 1, 0, datetime('now'), datetime('now')),

(4, '分流规则说明', '<h2>分流规则说明</h2>
<h3>什么是分流规则？</h3>
<p>分流规则决定哪些网站通过代理访问，哪些网站直接连接。合理的分流规则可以提升访问速度，节省流量。</p>
<h3>常见模式</h3>
<h4>全局模式</h4>
<p>所有流量都通过代理。</p>
<ul>
<li>优点：简单直接，确保所有网站都能访问</li>
<li>缺点：国内网站访问速度可能变慢，消耗更多流量</li>
</ul>
<h4>规则模式（推荐）</h4>
<p>根据规则自动判断是否使用代理。</p>
<ul>
<li>优点：国内网站直连，国外网站走代理，速度快且节省流量</li>
<li>缺点：需要维护规则列表</li>
</ul>
<h4>直连模式</h4>
<p>所有流量都不通过代理。</p>
<ul>
<li>用途：临时禁用代理</li>
</ul>
<h3>自定义规则</h3>
<p>大多数客户端支持自定义规则，您可以：</p>
<ul>
<li>添加特定域名到代理列表</li>
<li>添加特定域名到直连列表</li>
<li>使用正则表达式匹配</li>
</ul>
<h3>推荐配置</h3>
<p>对于大多数用户，我们推荐使用"规则模式"，并使用客户端自带的规则列表。这些规则会定期更新，无需手动维护。</p>', '了解和配置分流规则', 2, 1, 0, datetime('now'), datetime('now')),

-- 账户相关
(5, '如何充值？', '<h2>充值指南</h2>
<h3>充值方式</h3>
<p>我们支持多种充值方式：</p>
<ul>
<li>支付宝</li>
<li>微信支付</li>
<li>加密货币（USDT）</li>
</ul>
<h3>充值步骤</h3>
<ol>
<li>登录网站</li>
<li>点击右上角"充值"按钮</li>
<li>输入充值金额</li>
<li>选择支付方式</li>
<li>完成支付</li>
<li>余额会自动到账</li>
</ol>
<h3>充值优惠</h3>
<ul>
<li>单次充值满100元，赠送5%</li>
<li>单次充值满500元，赠送10%</li>
<li>单次充值满1000元，赠送15%</li>
</ul>
<h3>注意事项</h3>
<ul>
<li>充值金额实时到账</li>
<li>充值后的余额可用于购买套餐</li>
<li>余额不支持提现</li>
<li>如遇到充值问题，请联系客服</li>
</ul>', '了解充值方式和流程', 1, 1, 0, datetime('now'), datetime('now')),

(5, '邀请奖励说明', '<h2>邀请奖励计划</h2>
<h3>如何邀请好友？</h3>
<ol>
<li>登录网站</li>
<li>进入"我的邀请"页面</li>
<li>复制您的专属邀请链接或邀请码</li>
<li>分享给好友</li>
</ol>
<h3>奖励规则</h3>
<h4>邀请人奖励</h4>
<ul>
<li>好友注册：获得5元余额</li>
<li>好友首次购买：获得好友消费金额的20%作为佣金</li>
<li>好友续费：持续获得20%佣金</li>
</ul>
<h4>被邀请人奖励</h4>
<ul>
<li>使用邀请码注册：获得10元余额</li>
<li>首次购买：额外获得5%折扣</li>
</ul>
<h3>佣金提现</h3>
<p>累计佣金可以：</p>
<ul>
<li>用于购买套餐</li>
<li>达到100元后可申请提现（需联系客服）</li>
</ul>
<h3>注意事项</h3>
<ul>
<li>不允许自己邀请自己</li>
<li>禁止通过不正当手段获取奖励</li>
<li>违规账户将被封禁</li>
</ul>', '通过邀请好友获得奖励', 2, 1, 0, datetime('now'), datetime('now')),

(5, '退款政策', '<h2>退款政策</h2>
<h3>退款条件</h3>
<p>在以下情况下，您可以申请退款：</p>
<ul>
<li>购买后7天内</li>
<li>流量使用不超过10%</li>
<li>无违规使用记录</li>
</ul>
<h3>不支持退款的情况</h3>
<ul>
<li>购买超过7天</li>
<li>流量使用超过10%</li>
<li>账户被封禁</li>
<li>使用优惠券或折扣购买的订单</li>
</ul>
<h3>退款流程</h3>
<ol>
<li>提交工单说明退款原因</li>
<li>客服审核（1-3个工作日）</li>
<li>审核通过后，款项原路退回</li>
<li>到账时间：3-7个工作日</li>
</ol>
<h3>注意事项</h3>
<ul>
<li>退款金额为实际支付金额</li>
<li>赠送的余额和流量不退还</li>
<li>每个账户最多申请3次退款</li>
</ul>
<p>如有疑问，请联系客服咨询。</p>', '了解退款政策和流程', 3, 1, 0, datetime('now'), datetime('now'));
