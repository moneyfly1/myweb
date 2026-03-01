-- 更新后的知识库数据 - 统一使用Clash系列客户端
-- 使用方法：sqlite3 your_database.db < update_knowledge.sql

-- 更新Windows教程
UPDATE knowledge_articles SET content = '<h2>Windows 客户端使用教程</h2>
<h3>推荐客户端：Clash Verge（原Clash for Windows已停更）</h3>
<h4>下载安装</h4>
<ol>
<li>访问软件教程页面，下载 Clash Verge 最新版本</li>
<li>解压下载的压缩包到任意文件夹（建议：C:\Program Files\Clash Verge）</li>
<li>运行 Clash Verge.exe</li>
<li>首次运行可能需要允许防火墙访问</li>
</ol>
<h4>导入订阅</h4>
<ol>
<li>复制您的订阅链接（在"订阅管理"页面）</li>
<li>打开 Clash Verge，点击左侧"订阅"</li>
<li>点击右上角"+"号，选择"导入"</li>
<li>粘贴订阅链接，点击"导入"</li>
<li>等待订阅更新完成</li>
</ol>
<h4>开始使用</h4>
<ol>
<li>点击左侧"代理"</li>
<li>选择"规则模式"（推荐）</li>
<li>在节点列表中选择一个节点（可以点击"延迟测试"选择最快节点）</li>
<li>点击左侧"设置"，开启"系统代理"</li>
<li>现在您可以正常访问网络了</li>
</ol>
<h4>常见问题</h4>
<p><strong>Q: 无法连接？</strong><br>
A: 检查是否开启了系统代理，尝试切换其他节点。如果还是不行，尝试以管理员身份运行。</p>
<p><strong>Q: 速度慢？</strong><br>
A: 使用延迟测试功能选择最快节点，或切换到其他地区的节点。</p>
<p><strong>Q: 软件闪退？</strong><br>
A: 检查是否有杀毒软件拦截，尝试添加到白名单。</p>
<h4>高级设置</h4>
<ul>
<li><strong>TUN模式</strong>：可以代理所有应用，包括不支持系统代理的软件</li>
<li><strong>开机自启</strong>：在设置中开启，方便使用</li>
<li><strong>自动更新订阅</strong>：建议开启，保持节点信息最新</li>
</ul>' WHERE id = 3;

-- 更新macOS教程
UPDATE knowledge_articles SET content = '<h2>macOS 客户端使用教程</h2>
<h3>推荐客户端：ClashX Pro</h3>
<h4>下载安装</h4>
<ol>
<li>访问软件教程页面，下载 ClashX Pro 最新版本</li>
<li>打开下载的 dmg 文件</li>
<li>将 ClashX Pro 拖动到应用程序文件夹</li>
<li>首次运行需要在"系统偏好设置 - 安全性与隐私"中允许</li>
<li>授予必要的系统权限（网络扩展权限）</li>
</ol>
<h4>导入订阅</h4>
<ol>
<li>复制您的订阅链接</li>
<li>点击菜单栏的 ClashX Pro 图标</li>
<li>选择"配置 - 托管配置 - 管理"</li>
<li>点击"添加"，粘贴订阅链接</li>
<li>设置配置名称（如：我的订阅）</li>
<li>点击"确定"，等待订阅更新</li>
</ol>
<h4>开始使用</h4>
<ol>
<li>点击菜单栏的 ClashX Pro 图标</li>
<li>选择"设置为系统代理"</li>
<li>选择"出站模式 - 规则判断"（推荐）</li>
<li>在"代理"中选择一个节点</li>
<li>图标变为彩色表示已连接</li>
</ol>
<h4>常见问题</h4>
<p><strong>Q: 无法连接？</strong><br>
A: 检查系统代理是否开启，尝试切换其他节点。检查是否有其他代理软件冲突。</p>
<p><strong>Q: 速度慢？</strong><br>
A: 使用"代理 - 延迟测试"功能选择最快节点。</p>
<p><strong>Q: 权限问题？</strong><br>
A: 在"系统偏好设置 - 安全性与隐私 - 隐私 - 网络"中检查权限。</p>
<h4>高级功能</h4>
<ul>
<li><strong>增强模式</strong>：可以代理所有应用，包括不支持系统代理的软件</li>
<li><strong>开机启动</strong>：在设置中开启，方便使用</li>
<li><strong>自动更新订阅</strong>：建议开启，保持节点信息最新</li>
<li><strong>规则模式</strong>：国内直连，国外走代理，速度更快</li>
</ul>
<h4>提示</h4>
<p>建议开启"开机启动"和"自动更新订阅"，以获得最佳使用体验。如果遇到问题，可以尝试重启软件或重置配置。</p>' WHERE id = 4;

-- 更新iOS教程
UPDATE knowledge_articles SET content = '<h2>iOS 客户端使用教程</h2>
<h3>推荐客户端：Shadowrocket（小火箭）</h3>
<h4>下载安装</h4>
<p>Shadowrocket 是付费应用（约$2.99），需要使用美区 Apple ID 在 App Store 下载。</p>
<blockquote>
<p><strong>获取方式：</strong></p>
<ul>
<li>方式1：使用自己的美区 Apple ID 购买（推荐）</li>
<li>方式2：在淘宝搜索"美区 Apple ID"购买共享账号</li>
<li>方式3：联系客服获取共享账号（仅用于下载，下载后请立即退出）</li>
</ul>
</blockquote>
<h4>导入订阅</h4>
<ol>
<li>复制您的订阅链接</li>
<li>打开 Shadowrocket</li>
<li>点击右上角"+"</li>
<li>选择"类型 - Subscribe"</li>
<li>在"URL"中粘贴订阅链接</li>
<li>设置备注名称（如：我的订阅）</li>
<li>点击"完成"</li>
<li>等待订阅更新完成</li>
</ol>
<h4>开始使用</h4>
<ol>
<li>在首页选择一个节点（可以点击"连通性测试"选择最快节点）</li>
<li>点击顶部的连接开关</li>
<li>首次使用需要允许添加 VPN 配置，输入密码或使用 Face ID</li>
<li>连接成功后，状态栏会显示 VPN 图标</li>
<li>现在可以正常访问网络了</li>
</ol>
<h4>配置建议</h4>
<ul>
<li><strong>全局路由</strong>：建议选择"配置"模式（自动分流）</li>
<li><strong>DNS</strong>：建议使用"系统"或"1.1.1.1,8.8.8.8"</li>
<li><strong>连接时测试</strong>：开启后自动选择最快节点</li>
<li><strong>按需连接</strong>：开启后可以自动连接</li>
</ul>
<h4>常见问题</h4>
<p><strong>Q: 无法连接？</strong><br>
A: 检查订阅是否过期，尝试更新订阅或切换其他节点。</p>
<p><strong>Q: 速度慢？</strong><br>
A: 使用"连通性测试"选择延迟低的节点，或切换到其他地区。</p>
<p><strong>Q: 耗电快？</strong><br>
A: 关闭"按需连接"，不使用时手动断开连接。</p>
<h4>高级功能</h4>
<ul>
<li><strong>场景模式</strong>：可以为不同场景设置不同的代理规则</li>
<li><strong>脚本功能</strong>：支持 JavaScript 脚本，实现高级功能</li>
<li><strong>MITM</strong>：可以解密 HTTPS 流量（需要安装证书）</li>
</ul>' WHERE id = 5;

-- 更新Android教程
UPDATE knowledge_articles SET content = '<h2>Android 客户端使用教程</h2>
<h3>推荐客户端：Clash for Android（CFA）</h3>
<h4>下载安装</h4>
<ol>
<li>访问软件教程页面，下载 Clash for Android APK</li>
<li>在手机设置中允许"安装未知来源应用"</li>
<li>点击下载的 APK 文件进行安装</li>
<li>安装完成后打开应用</li>
</ol>
<blockquote>
<p><strong>注意：</strong>部分手机可能需要在"应用管理"中给予 Clash 必要的权限。</p>
</blockquote>
<h4>导入订阅</h4>
<ol>
<li>复制您的订阅链接</li>
<li>打开 Clash for Android</li>
<li>点击"配置"</li>
<li>点击右上角"+"</li>
<li>选择"URL"</li>
<li>粘贴订阅链接</li>
<li>设置配置名称（如：我的订阅）</li>
<li>点击"保存"</li>
<li>等待订阅更新完成</li>
</ol>
<h4>开始使用</h4>
<ol>
<li>返回首页，点击"点击启动"</li>
<li>首次使用需要允许创建 VPN 连接</li>
<li>点击"代理"选择节点（可以使用"延迟测试"选择最快节点）</li>
<li>连接成功后，通知栏会显示运行状态</li>
<li>现在可以正常访问网络了</li>
</ol>
<h4>模式说明</h4>
<ul>
<li><strong>全局模式</strong>：所有流量都通过代理（耗流量）</li>
<li><strong>规则模式</strong>：根据规则自动分流（推荐，省流量）</li>
<li><strong>直连模式</strong>：不使用代理</li>
</ul>
<h4>常见问题</h4>
<p><strong>Q: 无法连接？</strong><br>
A: 检查是否授予了 VPN 权限，尝试更新订阅或切换节点。</p>
<p><strong>Q: 速度慢？</strong><br>
A: 使用"延迟测试"选择最快节点，或切换到规则模式。</p>
<p><strong>Q: 耗电快？</strong><br>
A: 在省电设置中将 Clash 添加到白名单，允许后台运行。</p>
<p><strong>Q: 自动断开？</strong><br>
A: 关闭手机的省电模式，或在应用管理中设置 Clash 为"无限制"。</p>
<h4>高级设置</h4>
<ul>
<li><strong>开机启动</strong>：在设置中开启，方便使用</li>
<li><strong>自动更新订阅</strong>：建议开启，保持节点信息最新</li>
<li><strong>允许局域网连接</strong>：可以让其他设备通过你的手机代理上网</li>
<li><strong>IPv6</strong>：如果网络支持，可以开启以获得更好的连接</li>
</ul>
<h4>推荐设置</h4>
<ol>
<li>模式：规则模式</li>
<li>开机启动：开启</li>
<li>自动更新订阅：开启（每天一次）</li>
<li>后台运行：允许</li>
</ol>' WHERE id = 6;

-- 更新"如何开始使用"教程，统一推荐Clash系列
UPDATE knowledge_articles SET content = '<h2>快速开始指南</h2>
<h3>第一步：注册账户</h3>
<p>访问我们的网站，点击"注册"按钮，填写邮箱和密码完成注册。建议使用常用邮箱，以便接收重要通知。</p>
<h3>第二步：选择套餐</h3>
<p>登录后，进入"套餐购买"页面，根据您的需求选择合适的套餐：</p>
<ul>
<li><strong>基础套餐</strong>：适合轻度使用，日常浏览网页、查资料</li>
<li><strong>标准套餐</strong>：适合中度使用，观看1080P视频、下载文件</li>
<li><strong>高级套餐</strong>：适合重度使用，4K视频、大文件传输、游戏加速</li>
</ul>
<h3>第三步：下载客户端</h3>
<p>我们推荐使用 Clash 系列客户端，稳定性好、功能强大：</p>
<ul>
<li><strong>Windows</strong>：Clash Verge（推荐）</li>
<li><strong>macOS</strong>：ClashX Pro（推荐）</li>
<li><strong>iOS</strong>：Shadowrocket（小火箭，需美区账号）</li>
<li><strong>Android</strong>：Clash for Android（推荐）</li>
</ul>
<p>所有客户端下载链接都可以在"软件教程"页面找到。</p>
<h3>第四步：导入订阅</h3>
<ol>
<li>登录网站，进入"订阅管理"页面</li>
<li>点击"复制订阅链接"按钮</li>
<li>打开客户端，找到"订阅"或"配置"选项</li>
<li>粘贴订阅链接并导入</li>
<li>等待订阅更新完成</li>
</ol>
<h3>第五步：连接使用</h3>
<ol>
<li>在客户端中选择一个节点（建议使用延迟测试选择最快的）</li>
<li>开启系统代理或 VPN 连接</li>
<li>选择"规则模式"（自动分流，国内直连，国外走代理）</li>
<li>现在可以正常访问网络了</li>
</ol>
<h3>常见问题</h3>
<p><strong>Q: 订阅链接在哪里？</strong><br>
A: 登录网站后，在"订阅管理"页面可以找到。</p>
<p><strong>Q: 如何选择节点？</strong><br>
A: 使用客户端的"延迟测试"功能，选择延迟最低的节点。</p>
<p><strong>Q: 什么是规则模式？</strong><br>
A: 规则模式会自动判断，国内网站直连，国外网站走代理，速度快且省流量。</p>
<p><strong>Q: 遇到问题怎么办？</strong><br>
A: 查看对应平台的详细教程，或联系客服获取帮助。</p>' WHERE id = 2;

-- 验证更新
SELECT id, title, substr(content, 1, 50) as content_preview FROM knowledge_articles WHERE category_id = 2;
