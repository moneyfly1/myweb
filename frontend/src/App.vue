<template>
  <div id="app">
    <router-view v-slot="{ Component, route }">
      <transition name="page-fade" mode="out-in">
        <!--
          key 用顶层路径段（/admin 或 /）而非完整 path：
          管理端切换子菜单时 AdminLayout 不销毁重建，内部 keep-alive 缓存生效，
          页面切换无需重新下载 JS / 重新请求数据，显著提升流畅度。
        -->
        <component :is="Component" :key="route.meta.layoutKey || route.path.split('/')[1] || 'root'" />
      </transition>
    </router-view>
  </div>
</template>
<script>
export default {
  name: 'App'
}
</script>
<style>
/* 路由切换过渡：轻微淡入淡出，不造成眩晕感 */
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 0.15s ease;
}
.page-fade-enter-from,
.page-fade-leave-to {
  opacity: 0;
}
</style>
