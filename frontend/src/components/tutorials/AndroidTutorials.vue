<template>
  <div class="android-tutorials">
    <el-collapse v-model="activeNames">
      <!-- Clash Meta -->
      <el-collapse-item title="Clash Meta" name="clash-meta">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 Clash Meta 最新版本。</p>
          
          <h3>2. 安装步骤</h3>
          <ol>
            <li>下载完成后，点击安装包</li>
            <li>如果提示"未知来源"，请先开启"允许安装未知应用"</li>
            <li>按照提示完成安装</li>
            <li>安装完成后，在应用列表中找到 Clash Meta</li>
          </ol>
          
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <h4>方法一：复制订阅链接</h4>
            <ol>
              <li>在用户仪表盘中，复制"Clash订阅地址"</li>
              <li>打开 Clash Meta</li>
              <li>点击"配置" → "新建配置"</li>
              <li>选择"从URL下载"</li>
              <li>粘贴订阅链接，点击"下载"</li>
              <li>选择下载的配置文件</li>
            </ol>
            
            <h4>方法二：一键导入</h4>
            <ol>
              <li>在用户仪表盘中，点击"Clash Meta"按钮</li>
              <li>选择"一键导入"</li>
              <li>手机会自动打开 Clash Meta 并导入配置</li>
            </ol>
          </div>
          
          <h3>4. 使用方法</h3>
          <ol>
            <li>导入配置后，在"代理"页面选择节点</li>
            <li>点击"已停止"按钮开启代理</li>
            <li>选择"规则模式"或"全局模式"</li>
            <li>开始使用代理服务</li>
          </ol>
          
          <div class="tips">
            <h4>💡 使用技巧</h4>
            <ul>
              <li>建议使用"规则模式"，可以自动分流国内外网站</li>
              <li>可以在"规则"页面自定义分流规则</li>
              <li>支持订阅自动更新，建议开启</li>
              <li>可以设置开机自启动</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>

      <!-- V2rayNG -->
      <el-collapse-item title="V2rayNG" name="v2rayng">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 V2rayNG 最新版本。</p>
          
          <h3>2. 安装步骤</h3>
          <ol>
            <li>下载完成后，点击安装包</li>
            <li>如果提示"未知来源"，请先开启"允许安装未知应用"</li>
            <li>按照提示完成安装</li>
            <li>安装完成后，在应用列表中找到 V2rayNG</li>
          </ol>
          
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <h4>方法一：复制订阅链接</h4>
            <ol>
              <li>在用户仪表盘中，复制"通用订阅地址"</li>
              <li>打开 V2rayNG</li>
              <li>点击右上角"+"号</li>
              <li>选择"从剪贴板导入"或"从URL导入"</li>
              <li>粘贴订阅链接，点击"确定"</li>
              <li>点击"更新订阅"获取节点</li>
            </ol>
            
            <h4>方法二：一键导入</h4>
            <ol>
              <li>在用户仪表盘中，点击"V2rayNG"按钮</li>
              <li>选择"一键导入"</li>
              <li>手机会自动打开 V2rayNG 并导入配置</li>
            </ol>
          </div>
          
          <h3>4. 使用方法</h3>
          <ol>
            <li>导入订阅后，在服务器列表中选择节点</li>
            <li>点击右下角的"V"按钮开启代理</li>
            <li>选择"路由"模式（推荐）或"全局"模式</li>
            <li>开始使用代理服务</li>
          </ol>
          
          <div class="tips">
            <h4>💡 使用技巧</h4>
            <ul>
              <li>建议使用"路由"模式，支持自动分流</li>
              <li>可以在"路由设置"中自定义分流规则</li>
              <li>支持订阅自动更新，建议开启</li>
              <li>可以设置开机自启动</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>

      <!-- Hiddify -->
      <el-collapse-item title="Hiddify" name="hiddify-android">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 Hiddify 最新版本。</p>
          
          <h3>2. 安装步骤</h3>
          <ol>
            <li>下载完成后，点击安装包</li>
            <li>如果提示"未知来源"，请先开启"允许安装未知应用"</li>
            <li>按照提示完成安装</li>
            <li>安装完成后，在应用列表中找到 Hiddify</li>
          </ol>
          
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <h4>方法一：复制订阅链接</h4>
            <ol>
              <li>在用户仪表盘中，复制"通用订阅地址"</li>
              <li>打开 Hiddify</li>
              <li>点击"订阅" → "添加订阅"</li>
              <li>粘贴订阅链接，点击"确定"</li>
              <li>点击"更新订阅"获取节点</li>
            </ol>
            
            <h4>方法二：一键导入</h4>
            <ol>
              <li>在用户仪表盘中，点击"Hiddify"按钮</li>
              <li>选择"一键导入"</li>
              <li>手机会自动打开 Hiddify 并导入配置</li>
            </ol>
          </div>
          
          <h3>4. 使用方法</h3>
          <ol>
            <li>导入订阅后，在服务器列表中选择节点</li>
            <li>点击"连接"按钮开启代理</li>
            <li>选择"自动"模式（推荐）或"全局"模式</li>
            <li>开始使用代理服务</li>
          </ol>
          
          <div class="tips">
            <h4>💡 使用技巧</h4>
            <ul>
              <li>Hiddify 支持多种协议</li>
              <li>界面友好，操作简单</li>
              <li>建议使用"自动"模式进行智能分流</li>
              <li>可以设置开机自启动</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const activeNames = ref(['clash-meta'])
</script>

<style scoped>
.android-tutorials {
  padding: 20px;
}

.tutorial-content {
  line-height: 1.6;
}

.tutorial-content :is(h3) {
  color: #2c3e50;
  margin-top: 20px;
  margin-bottom: 10px;
  border-left: 4px solid #e74c3c;
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
</style>
