<template>
  <div class="client-tutorials">
    <el-collapse v-model="activeNames">
      <el-collapse-item v-for="item in items" :key="item.name" :title="item.title" :name="item.name">
        <div class="tutorial-content">
          <h3>1. 软件下载</h3>
          <p>{{ item.download }}</p>
          <div v-if="item.warning" class="warning">
            <h4><el-icon><WarningFilled /></el-icon>重要提示</h4>
            <p>{{ item.warning }}</p>
          </div>
          <h3>2. 安装步骤</h3>
          <ol>
            <li v-for="(step, i) in item.install" :key="`install-${i}`">{{ step }}</li>
          </ol>
          <h3>3. 导入订阅</h3>
          <div class="subscription-methods">
            <template v-for="(method, mi) in item.subscription" :key="`method-${mi}`">
              <h4>{{ method.title }}</h4>
              <ol>
                <li v-for="(step, si) in method.steps" :key="`m${mi}-s${si}`">{{ step }}</li>
              </ol>
            </template>
          </div>
          <h3>4. 使用方法</h3>
          <ol>
            <li v-for="(step, i) in item.usage" :key="`usage-${i}`">{{ step }}</li>
          </ol>
          <template v-if="item.advanced && item.advanced.length">
            <h3>5. 高级设置</h3>
            <div class="advanced-settings">
              <template v-for="(section, ai) in item.advanced" :key="`adv-${ai}`">
                <h4>{{ section.title }}</h4>
                <ol>
                  <li v-for="(step, si) in section.steps" :key="`a${ai}-s${si}`">{{ step }}</li>
                </ol>
              </template>
            </div>
          </template>
          <div v-if="item.tips && item.tips.length" class="tips">
            <h4><el-icon><InfoFilled /></el-icon>使用技巧</h4>
            <ul>
              <li v-for="(tip, i) in item.tips" :key="`tip-${i}`">{{ tip }}</li>
            </ul>
          </div>
          <div v-if="item.troubleshooting && item.troubleshooting.length" class="troubleshooting">
            <h4><el-icon><Tools /></el-icon>常见问题解决</h4>
            <ul>
              <li v-for="(item2, i) in item.troubleshooting" :key="`ts-${i}`"><strong>{{ item2.label }}：</strong>{{ item2.text }}</li>
            </ul>
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { InfoFilled, WarningFilled, Tools } from '@element-plus/icons-vue'
defineProps({
  items: { type: Array, required: true }
})
const activeNames = ref([])
</script>

<style scoped>
.client-tutorials {
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
  display: flex;
  align-items: center;
  gap: 6px;
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
