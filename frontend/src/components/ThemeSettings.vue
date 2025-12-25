<template>
  <div class="theme-settings">
    <el-drawer
      v-model="visible"
      title="主题设置"
      direction="rtl"
      size="400px"
    >
      <div class="settings-content">
        <div class="setting-section">
          <h3 class="section-title">主题风格</h3>
          <div class="theme-grid">
            <div
              v-for="theme in themes"
              :key="theme.value"
              class="theme-item"
              :class="{ active: currentTheme === theme.value }"
              @click="selectTheme(theme.value)"
            >
              <div class="theme-preview" :style="getThemePreviewStyle(theme)">
                <div class="preview-header"></div>
                <div class="preview-sidebar"></div>
                <div class="preview-content"></div>
              </div>
              <div class="theme-info">
                <span class="theme-name">{{ theme.label }}</span>
                <span class="theme-desc">{{ getThemeDescription(theme.value) }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="setting-section">
          <h3 class="section-title">布局设置</h3>
          <div class="setting-item">
            <span class="setting-label">侧边栏模式</span>
            <el-switch
              v-model="sidebarCollapsed"
              active-text="收起"
              inactive-text="展开"
              @change="toggleSidebar"
            />
          </div>
          <div class="setting-item">
            <span class="setting-label">紧凑模式</span>
            <el-switch
              v-model="compactMode"
              active-text="开启"
              inactive-text="关闭"
              @change="toggleCompactMode"
            />
          </div>
        </div>
        <div class="setting-section">
          <h3 class="section-title">显示设置</h3>
          <div class="setting-item">
            <span class="setting-label">显示面包屑</span>
            <el-switch
              v-model="showBreadcrumb"
              active-text="显示"
              inactive-text="隐藏"
              @change="toggleBreadcrumb"
            />
          </div>
          <div class="setting-item">
            <span class="setting-label">显示搜索框</span>
            <el-switch
              v-model="showSearch"
              active-text="显示"
              inactive-text="隐藏"
              @change="toggleSearch"
            />
          </div>
        </div>
        <div class="setting-section">
          <h3 class="section-title">动画设置</h3>
          <div class="setting-item">
            <span class="setting-label">页面切换动画</span>
            <el-switch
              v-model="pageAnimation"
              active-text="开启"
              inactive-text="关闭"
              @change="togglePageAnimation"
            />
          </div>
          <div class="setting-item">
            <span class="setting-label">加载动画</span>
            <el-switch
              v-model="loadingAnimation"
              active-text="开启"
              inactive-text="关闭"
              @change="toggleLoadingAnimation"
            />
          </div>
        </div>
        <div class="setting-section">
          <el-button
            type="default"
            @click="resetSettings"
            class="reset-btn"
          >
            <i class="el-icon-refresh"></i>
            重置所有设置
          </el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useThemeStore } from '@/store/theme'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  }
})
const emit = defineEmits(['update:modelValue'])
const sidebarCollapsed = ref(localStorage.getItem('sidebarCollapsed') === 'true')
const compactMode = ref(localStorage.getItem('compactMode') === 'true')
const showBreadcrumb = ref(localStorage.getItem('showBreadcrumb') !== 'false')
const showSearch = ref(localStorage.getItem('showSearch') !== 'false')
const pageAnimation = ref(localStorage.getItem('pageAnimation') !== 'false')
const loadingAnimation = ref(localStorage.getItem('loadingAnimation') !== 'false')
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})
const themeStore = useThemeStore()
const currentTheme = computed(() => themeStore.currentTheme)
const themes = computed(() => themeStore.availableThemes)
const selectTheme = async (themeName) => {
  await themeStore.setTheme(themeName)
}
const toggleSidebar = (value) => {
  localStorage.setItem('sidebarCollapsed', value)
  window.dispatchEvent(new CustomEvent('toggle-sidebar', { detail: value }))
}
const toggleCompactMode = (value) => {
  localStorage.setItem('compactMode', value)
  document.body.classList.toggle('compact-mode', value)
}
const toggleBreadcrumb = (value) => {
  localStorage.setItem('showBreadcrumb', value)
}
const toggleSearch = (value) => {
  localStorage.setItem('showSearch', value)
}
const togglePageAnimation = (value) => {
  localStorage.setItem('pageAnimation', value)
}
const toggleLoadingAnimation = (value) => {
  localStorage.setItem('loadingAnimation', value)
}
const resetSettings = () => {
  sidebarCollapsed.value = false
  compactMode.value = false
  showBreadcrumb.value = true
  showSearch.value = true
  pageAnimation.value = true
  loadingAnimation.value = true
  themeStore.setTheme('light')
  localStorage.removeItem('sidebarCollapsed')
  localStorage.removeItem('compactMode')
  localStorage.removeItem('showBreadcrumb')
  localStorage.removeItem('showSearch')
  localStorage.removeItem('pageAnimation')
  localStorage.removeItem('loadingAnimation')
  document.body.classList.remove('compact-mode')
  window.dispatchEvent(new CustomEvent('reset-settings'))
}
const getThemePreviewStyle = (theme) => {
  const themeColors = {
    light: { primary: '#409EFF', background: '#f5f7fa', text: '#303133', border: '#DCDFE6' },
    dark: { primary: '#409EFF', background: '#1a1a1a', text: '#ffffff', border: '#4c4c4c' },
    blue: { primary: '#1890ff', background: '#f0f2f5', text: '#262626', border: '#d9d9d9' },
    green: { primary: '#52c41a', background: '#f6ffed', text: '#262626', border: '#b7eb8f' },
    purple: { primary: '#722ed1', background: '#f9f0ff', text: '#262626', border: '#d3adf7' },
    orange: { primary: '#fa8c16', background: '#fff7e6', text: '#262626', border: '#ffd591' },
    red: { primary: '#f5222d', background: '#fff1f0', text: '#262626', border: '#ffccc7' },
    cyan: { primary: '#13c2c2', background: '#e6fffb', text: '#262626', border: '#87e8de' },
    luck: { primary: '#FFD700', background: '#FFFEF0', text: '#2C2416', border: '#FFD700' },
    aurora: { primary: '#7B68EE', background: '#0F0C1D', text: '#E6E6FA', border: '#4B0082' },
    auto: { primary: '#909399', background: '#f5f7fa', text: '#303133', border: '#DCDFE6' },
    default: { primary: '#409EFF', background: '#f5f7fa', text: '#303133', border: '#DCDFE6' }
  }
  const config = themeColors[theme.value] || themeColors.light
  return {
    '--preview-primary': config.primary,
    '--preview-background': config.background,
    '--preview-text': config.text,
    '--preview-border': config.border
  }
}
const getThemeDescription = (themeName) => {
  const descriptions = {
    light: '经典浅色主题，适合大多数用户',
    dark: '深色主题，护眼舒适',
    blue: '商务蓝色主题，专业大气',
    green: '清新绿色主题，自然舒适',
    purple: '优雅紫色主题，神秘浪漫',
    orange: '活力橙色主题，温暖明亮',
    red: '热情红色主题，醒目突出',
    cyan: '清新青色主题，清爽自然',
    luck: 'Luck幸运主题，金色温暖，带来好运',
    aurora: 'Aurora极光主题，梦幻紫色，神秘优雅',
    auto: '跟随系统主题，自动切换',
    default: '经典蓝色主题，适合大多数用户'
  }
  return descriptions[themeName] || '自定义主题'
}
</script>

<style scoped lang="scss">
.theme-settings {
  .settings-content {
    padding: 20px;
  }
  .setting-section {
    margin-bottom: 30px;
    .section-title {
      font-size: 16px;
      font-weight: 600;
      margin-bottom: 16px;
      color: var(--theme-text);
      border-bottom: 1px solid var(--theme-border);
      padding-bottom: 8px;
    }
  }
  .theme-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
    .theme-item {
      border: 2px solid var(--theme-border);
      border-radius: 8px;
      overflow: hidden;
      cursor: pointer;
      transition: all 0.3s ease;
      &:hover {
        border-color: var(--theme-primary);
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      }
      &.active {
        border-color: var(--theme-primary);
        box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
      }
      .theme-preview {
        height: 80px;
        background: var(--preview-background);
        position: relative;
        overflow: hidden;
        .preview-header {
          height: 20px;
          background: var(--preview-primary);
          position: absolute;
          top: 0;
          left: 0;
          right: 0;
        }
        .preview-sidebar {
          width: 30px;
          height: 60px;
          background: white;
          border-right: 1px solid var(--preview-border);
          position: absolute;
          top: 20px;
          left: 0;
        }
        .preview-content {
          height: 60px;
          background: var(--preview-background);
          position: absolute;
          top: 20px;
          left: 30px;
          right: 0;
        }
      }
      .theme-info {
        padding: 12px;
        background: white;
        .theme-name {
          display: block;
          font-weight: 500;
          color: var(--theme-text);
          margin-bottom: 4px;
        }
        .theme-desc {
          display: block;
          font-size: 12px;
          color: #666;
          line-height: 1.4;
        }
      }
    }
  }
  .setting-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 0;
    border-bottom: 1px solid #f0f0f0;
    &:last-child {
      border-bottom: none;
    }
    .setting-label {
      font-weight: 500;
      color: var(--theme-text);
    }
  }
  .reset-btn {
    width: 100%;
    height: 40px;
    :is(i) {
      margin-right: 8px;
    }
  }
}
.compact-mode {
  :root {
    --header-height: 50px;
    --sidebar-width: 180px;
    --content-padding: 15px;
    --border-radius: 6px;
  }
  .el-card {
    .el-card__header {
      padding: 12px 16px;
    }
    .el-card__body {
      padding: 16px;
    }
  }
  .el-table {
    :is(th), :is(td) {
      padding: 8px 12px;
    }
  }
  .el-form {
    .el-form-item {
      margin-bottom: 16px;
    }
  }
}
@media (max-width: 768px) {
  .theme-settings {
    .theme-grid {
      grid-template-columns: 1fr;
    }
    .setting-item {
      flex-direction: column;
      align-items: flex-start;
      gap: 8px;
    }
  }
}
</style>
