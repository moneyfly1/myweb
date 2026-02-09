<template>
  <div class="ios-tutorials">
    <el-collapse v-model="activeNames">
      <el-collapse-item title="Shadowrocket" name="shadowrocket">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 Shadowrocket 最新版本。</p>
          <div class="warning">
            <h4>⚠️ 重要提示</h4>
            <p>Shadowrocket 需要从 App Store 购买下载，或者使用企业证书版本。请确保从可信来源获取。</p>
          </div>
          <h3>2. 安装步骤</h3>
          <ol>
            <li>从 App Store 搜索"Shadowrocket"并购买下载</li>
            <li>或者下载企业证书版本（需要信任证书）</li>
            <li>安装完成后，在桌面找到 Shadowrocket 图标</li>
            <li>首次打开需要授予 VPN 权限</li>
          </ol>
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <h4>方法一：复制订阅链接</h4>
            <ol>
              <li>在用户仪表盘中，复制"通用订阅地址"</li>
              <li>打开 Shadowrocket</li>
              <li>点击右上角"+"号</li>
              <li>选择"类型" → "Subscribe"</li>
              <li>在"URL"字段粘贴订阅链接</li>
              <li>点击"完成"保存</li>
              <li>点击"更新"获取节点</li>
            </ol>
            <h4>方法二：一键导入</h4>
            <ol>
              <li>在用户仪表盘中，点击"Shadowrocket"按钮</li>
              <li>选择"一键导入"</li>
              <li>手机会自动打开 Shadowrocket 并导入配置</li>
            </ol>
            <h4>方法三：扫描二维码</h4>
            <ol>
              <li>在用户仪表盘中，找到"二维码"部分</li>
              <li>打开 Shadowrocket</li>
              <li>点击右上角"+"号</li>
              <li>选择"扫描二维码"</li>
              <li>扫描仪表盘中的二维码</li>
              <li>点击"完成"保存</li>
            </ol>
          </div>
          <h3>4. 使用方法</h3>
          <ol>
            <li>导入订阅后，在服务器列表中选择节点</li>
            <li>点击右上角的开关按钮开启代理</li>
            <li>选择"配置"模式（推荐）或"全局"模式</li>
            <li>开始使用代理服务</li>
          </ol>
          <h3>5. 高级设置</h3>
          <div class="advanced-settings">
            <h4>分流规则设置</h4>
            <ol>
              <li>点击"配置" → "规则"</li>
              <li>可以添加自定义分流规则</li>
              <li>支持按域名、IP、关键词等分流</li>
            </ol>
            <h4>DNS 设置</h4>
            <ol>
              <li>点击"配置" → "DNS"</li>
              <li>可以设置自定义 DNS 服务器</li>
              <li>推荐使用 8.8.8.8 或 1.1.1.1</li>
            </ol>
            <h4>日志设置</h4>
            <ol>
              <li>点击"配置" → "日志"</li>
              <li>可以查看连接日志</li>
              <li>帮助排查连接问题</li>
            </ol>
          </div>
          <div class="tips">
            <h4>💡 使用技巧</h4>
            <ul>
              <li>建议使用"配置"模式，支持自动分流</li>
              <li>可以在"规则"页面自定义分流规则</li>
              <li>支持订阅自动更新，建议开启</li>
              <li>可以设置快捷开关，方便快速开启/关闭</li>
              <li>支持多订阅源，可以同时使用多个服务</li>
            </ul>
          </div>
          <div class="troubleshooting">
            <h4>🔧 常见问题解决</h4>
            <ul>
              <li><strong>无法连接：</strong>检查网络连接，尝试切换节点</li>
              <li><strong>速度慢：</strong>尝试选择延迟较低的节点</li>
              <li><strong>订阅更新失败：</strong>检查订阅链接是否正确</li>
              <li><strong>应用闪退：</strong>重启应用或重新安装</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>
<script setup>
import { ref } from 'vue'
const activeNames = ref(['shadowrocket'])
</script>
<style scoped>
.ios-tutorials {
  padding: 20px;
}
.tutorial-content {
  line-height: 1.6;
}
.tutorial-content :is(h3) {
  color: #2c3e50;
  margin-top: 20px;
  margin-bottom: 10px;
  border-left: 4px solid #9b59b6;
  padding-left: 10px;
}
.tutorial-content :is(h4) {
  color: #34495e;
  margin-top: 15px;
  margin-bottom: 8px;
}
.tutorial-content :is(ol), .tutorial-content :is(ul) {
  margin: 10px 0;
  padding-left: 20px;
}
.tutorial-content :is(li) {
  margin: 5px 0;
}
.subscription-methods {
  background: #f8f9fa;
  padding: 15px;
  border-radius: 8px;
  margin: 15px 0;
}
.advanced-settings {
  background: #fff3cd;
  padding: 15px;
  border-radius: 8px;
  margin: 15px 0;
  border-left: 4px solid #ffc107;
}
.tips {
  background: #e8f5e8;
  padding: 15px;
  border-radius: 8px;
  margin: 15px 0;
  border-left: 4px solid #27ae60;
}
.tips :is(h4) {
  color: #27ae60;
  margin-top: 0;
}
.troubleshooting {
  background: #f8d7da;
  padding: 15px;
  border-radius: 8px;
  margin: 15px 0;
  border-left: 4px solid #dc3545;
}
.troubleshooting :is(h4) {
  color: #721c24;
  margin-top: 0;
}
.warning {
  background: #fff3cd;
  padding: 15px;
  border-radius: 8px;
  margin: 15px 0;
  border-left: 4px solid #ffc107;
}
.warning :is(h4) {
  color: #856404;
  margin-top: 0;
}
</style>
