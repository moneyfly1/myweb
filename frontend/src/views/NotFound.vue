<template>
  <div class="not-found-container">
    <div class="not-found-content">
      <div class="error-code">404</div>
      <h1>页面未找到</h1>
      <p>抱歉，您访问的页面不存在或已被删除。</p>
      <div class="actions">
        <router-link to="/dashboard">
          <el-button type="primary">
            返回首页
          </el-button>
        </router-link>
        <el-button @click="$router.go(-1)">
          返回上页
        </el-button>
      </div>
    </div>
  </div>
</template>
<script>
export default {
  name: 'NotFound'
}
</script>
<style scoped>
.not-found-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-lighter, #f5f7fa);
  padding: 20px;
}
.not-found-content {
  text-align: center;
  color: var(--el-text-color-primary, #303133);
  max-width: 500px;
  padding: 32px;
  border: 1px solid var(--el-border-color-lighter, #ebeef5);
  border-radius: 8px;
  background: var(--el-bg-color, #fff);
}
.error-code {
  font-size: 8rem;
  font-weight: bold;
  margin-bottom: 1rem;
  color: var(--el-color-primary, #409eff);
  line-height: 1;
}
.not-found-content h1 {
  font-size: 2rem;
  margin-bottom: 1rem;
  font-weight: 600;
}
.not-found-content :is(p) {
  font-size: 1.1rem;
  margin-bottom: 2rem;
  color: var(--el-text-color-secondary, #606266);
}
.actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
}
@media (max-width: 768px) {
  .not-found-container {
    align-items: flex-start;
    padding-top: 15vh;
  }
  .not-found-content {
    width: 100%;
    padding: 24px 16px;
  }
  .error-code {
    font-size: 6rem;
  }
  .not-found-content h1 {
    font-size: 1.5rem;
  }
  .actions {
    flex-direction: column;
    align-items: center;
  }
}
</style> 
