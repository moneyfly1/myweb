<template>
  <div class="windows-tutorials">
    <el-collapse v-model="activeNames">
      <!-- Clash for Windows -->
      <el-collapse-item title="Clash for Windows" name="clash-windows">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 Clash for Windows 最新版本。</p>
          
          <h3>2. 安装步骤</h3>
          <ol>
            <li>下载完成后，双击安装包</li>
            <li>按照安装向导完成安装</li>
            <li>安装完成后，桌面会出现 Clash for Windows 图标</li>
          </ol>
          
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <h4>方法一：复制订阅链接</h4>
            <ol>
              <li>在用户仪表盘中，找到"订阅地址"部分</li>
              <li>复制"Clash订阅地址"</li>
              <li>打开 Clash for Windows</li>
              <li>点击"配置" → "从URL下载"</li>
              <li>粘贴订阅链接，点击"下载"</li>
              <li>选择下载的配置文件，点击"确定"</li>
            </ol>
            
            <h4>方法二：一键导入（推荐）</h4>
            <ol>
              <li>在用户仪表盘中，找到"Clash系列软件"部分</li>
              <li>点击"Clash for Windows"按钮</li>
              <li>选择"一键导入"</li>
              <li>软件会自动打开并导入配置</li>
            </ol>
          </div>
          
          <h3>4. 使用方法</h3>
          <ol>
            <li>导入配置后，在"代理"页面选择节点</li>
            <li>点击"系统代理"开启系统代理</li>
            <li>选择"规则模式"或"全局模式"</li>
            <li>开始使用代理服务</li>
          </ol>
          
          <div class="tips">
            <h4>💡 使用技巧</h4>
            <ul>
              <li>建议使用"规则模式"，可以自动分流国内外网站</li>
              <li>可以在"规则"页面自定义分流规则</li>
              <li>支持订阅自动更新，建议开启</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>

      <!-- V2rayN -->
      <el-collapse-item title="V2rayN" name="v2rayn">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 V2rayN 最新版本。</p>
          
          <h3>2. 安装步骤</h3>
          <ol>
            <li>下载完成后，解压到任意文件夹</li>
            <li>双击 v2rayN.exe 启动软件</li>
            <li>首次运行会提示创建桌面快捷方式</li>
          </ol>
          
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <h4>方法一：复制订阅链接</h4>
            <ol>
              <li>在用户仪表盘中，复制"通用订阅地址"</li>
              <li>打开 V2rayN</li>
              <li>点击"订阅" → "订阅设置"</li>
              <li>点击"添加"，粘贴订阅链接</li>
              <li>点击"确定"保存</li>
              <li>点击"更新订阅"获取节点</li>
            </ol>
            
            <h4>方法二：一键导入</h4>
            <ol>
              <li>在用户仪表盘中，点击"V2rayN"按钮</li>
              <li>选择"一键导入"</li>
              <li>软件会自动打开并导入配置</li>
            </ol>
          </div>
          
          <h3>4. 使用方法</h3>
          <ol>
            <li>导入订阅后，在服务器列表中选择节点</li>
            <li>右键点击系统托盘图标，选择"系统代理" → "自动配置系统代理"</li>
            <li>选择"路由"模式（推荐）或"全局"模式</li>
            <li>开始使用代理服务</li>
          </ol>
          
          <div class="tips">
            <h4>💡 使用技巧</h4>
            <ul>
              <li>建议使用"路由"模式，支持自动分流</li>
              <li>可以在"路由设置"中自定义分流规则</li>
              <li>支持订阅自动更新，建议开启</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>

      <!-- Mihomo Part -->
      <el-collapse-item title="Mihomo Part" name="mihomo-windows">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 Mihomo Part 最新版本。</p>
          
          <h3>2. 安装步骤</h3>
          <ol>
            <li>下载完成后，解压到任意文件夹</li>
            <li>双击 mihomo.exe 启动软件</li>
            <li>首次运行会提示创建桌面快捷方式</li>
          </ol>
          
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <h4>方法一：复制订阅链接</h4>
            <ol>
              <li>在用户仪表盘中，复制"Clash订阅地址"</li>
              <li>打开 Mihomo Part</li>
              <li>点击"配置" → "从URL下载"</li>
              <li>粘贴订阅链接，点击"下载"</li>
              <li>选择下载的配置文件</li>
            </ol>
            
            <h4>方法二：一键导入</h4>
            <ol>
              <li>在用户仪表盘中，点击"Mihomo Part"按钮</li>
              <li>选择"一键导入"</li>
              <li>软件会自动打开并导入配置</li>
            </ol>
          </div>
          
          <h3>4. 使用方法</h3>
          <ol>
            <li>导入配置后，在"代理"页面选择节点</li>
            <li>点击"系统代理"开启系统代理</li>
            <li>选择"规则模式"或"全局模式"</li>
            <li>开始使用代理服务</li>
          </ol>
          
          <div class="tips">
            <h4>💡 使用技巧</h4>
            <ul>
              <li>Mihomo Part 是 Clash 的增强版本，功能更强大</li>
              <li>支持更多协议和功能</li>
              <li>建议使用"规则模式"进行智能分流</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>

      <!-- Sparkle -->
      <el-collapse-item title="Sparkle" name="sparkle-windows">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 Sparkle 最新版本。</p>
          
          <h3>2. 安装步骤</h3>
          <ol>
            <li>下载完成后，双击安装包</li>
            <li>按照安装向导完成安装</li>
            <li>安装完成后，桌面会出现 Sparkle 图标</li>
          </ol>
          
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <h4>方法一：复制订阅链接</h4>
            <ol>
              <li>在用户仪表盘中，复制"Clash订阅地址"</li>
              <li>打开 Sparkle</li>
              <li>点击"配置" → "从URL下载"</li>
              <li>粘贴订阅链接，点击"下载"</li>
              <li>选择下载的配置文件</li>
            </ol>
            
            <h4>方法二：一键导入</h4>
            <ol>
              <li>在用户仪表盘中，点击"Sparkle"按钮</li>
              <li>选择"一键导入"</li>
              <li>软件会自动打开并导入配置</li>
            </ol>
          </div>
          
          <h3>4. 使用方法</h3>
          <ol>
            <li>导入配置后，在"代理"页面选择节点</li>
            <li>点击"系统代理"开启系统代理</li>
            <li>选择"规则模式"或"全局模式"</li>
            <li>开始使用代理服务</li>
          </ol>
          
          <div class="tips">
            <h4>💡 使用技巧</h4>
            <ul>
              <li>Sparkle 界面简洁，操作简单</li>
              <li>支持自动更新订阅</li>
              <li>建议使用"规则模式"进行智能分流</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>

      <!-- Hiddify -->
      <el-collapse-item title="Hiddify" name="hiddify-windows">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 Hiddify 最新版本。</p>
          
          <h3>2. 安装步骤</h3>
          <ol>
            <li>下载完成后，双击安装包</li>
            <li>按照安装向导完成安装</li>
            <li>安装完成后，桌面会出现 Hiddify 图标</li>
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
              <li>软件会自动打开并导入配置</li>
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
            </ul>
          </div>
        </div>
      </el-collapse-item>

      <!-- Flash -->
      <el-collapse-item title="Flash" name="flash-windows">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>点击上方"立即下载"按钮下载 Flash 最新版本。</p>
          
          <h3>2. 安装步骤</h3>
          <ol>
            <li>下载完成后，双击安装包</li>
            <li>按照安装向导完成安装</li>
            <li>安装完成后，桌面会出现 Flash 图标</li>
          </ol>
          
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <h4>方法一：复制订阅链接</h4>
            <ol>
              <li>在用户仪表盘中，复制"Clash订阅地址"</li>
              <li>打开 Flash</li>
              <li>点击"配置" → "从URL下载"</li>
              <li>粘贴订阅链接，点击"下载"</li>
              <li>选择下载的配置文件</li>
            </ol>
            
            <h4>方法二：一键导入</h4>
            <ol>
              <li>在用户仪表盘中，点击"Flash"按钮</li>
              <li>选择"一键导入"</li>
              <li>软件会自动打开并导入配置</li>
            </ol>
          </div>
          
          <h3>4. 使用方法</h3>
          <ol>
            <li>导入配置后，在"代理"页面选择节点</li>
            <li>点击"系统代理"开启系统代理</li>
            <li>选择"规则模式"或"全局模式"</li>
            <li>开始使用代理服务</li>
          </ol>
          
          <div class="tips">
            <h4>💡 使用技巧</h4>
            <ul>
              <li>Flash 是轻量级的代理软件</li>
              <li>启动速度快，资源占用少</li>
              <li>建议使用"规则模式"进行智能分流</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup>
import { ref } from 'vue'

const activeNames = ref(['clash-windows'])
</script>

<style scoped>
.windows-tutorials {
  padding: 20px;
}

.tutorial-content {
  line-height: 1.6;
}

.tutorial-content :is(h3) {
  color: #2c3e50;
  margin-top: 20px;
  margin-bottom: 10px;
  border-left: 4px solid #3498db;
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
