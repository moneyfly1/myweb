<template>
  <div class="tutorial-container">
    <div class="tutorial-header">
      <h1>软件安装与使用教程</h1>
      <p>详细指导如何安装和使用各种代理软件，以及如何导入订阅</p>
    </div>

    <div class="tutorial-nav">
      <el-tabs v-model="activeTab" type="border-card">
        <el-tab-pane label="Windows" name="windows">
          <WindowsTutorials />
        </el-tab-pane>
        <el-tab-pane label="Android" name="android">
          <AndroidTutorials />
        </el-tab-pane>
        <el-tab-pane label="macOS" name="macos">
          <MacOSTutorials />
        </el-tab-pane>
        <el-tab-pane label="iOS" name="ios">
          <iOSTutorials />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import WindowsTutorials from '@/components/tutorials/WindowsTutorials.vue'
import AndroidTutorials from '@/components/tutorials/AndroidTutorials.vue'
import MacOSTutorials from '@/components/tutorials/MacOSTutorials.vue'
import iOSTutorials from '@/components/tutorials/iOSTutorials.vue'

const activeTab = ref('windows')
</script>

<style scoped>
.tutorial-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.tutorial-header {
  text-align: center;
  margin-bottom: 30px;
}

.tutorial-header h1 {
  color: #2c3e50;
  margin-bottom: 10px;
}

.tutorial-header :is(p) {
  color: #7f8c8d;
  font-size: 16px;
}

.tutorial-nav {
  margin-top: 20px;
}
</style>
