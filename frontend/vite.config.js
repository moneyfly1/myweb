import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { resolve } from 'path'
import { fileURLToPath, URL } from 'node:url'

const __dirname = fileURLToPath(new URL('.', import.meta.url))
const kebabCase = value => value
  .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
  .replace(/([A-Z])([A-Z][a-z])/g, '$1-$2')
  .toLowerCase()

const elementPlusComponentDirs = {
  ElAnchorLink: 'anchor',
  ElAside: 'container',
  ElAvatarGroup: 'avatar',
  ElBreadcrumbItem: 'breadcrumb',
  ElButtonGroup: 'button',
  ElCarouselItem: 'carousel',
  ElCheckboxButton: 'checkbox',
  ElCheckboxGroup: 'checkbox',
  ElCollapseItem: 'collapse',
  ElDescriptionsItem: 'descriptions',
  ElDropdownItem: 'dropdown',
  ElDropdownMenu: 'dropdown',
  ElFooter: 'container',
  ElFormItem: 'form',
  ElHeader: 'container',
  ElMain: 'container',
  ElMenuItem: 'menu',
  ElMenuItemGroup: 'menu',
  ElOption: 'select',
  ElOptionGroup: 'select',
  ElRadioButton: 'radio',
  ElRadioGroup: 'radio',
  ElSkeletonItem: 'skeleton',
  ElSplitterPanel: 'splitter',
  ElStep: 'steps',
  ElSubMenu: 'menu',
  ElTabPane: 'tabs',
  ElTableColumn: 'table',
  ElTimelineItem: 'timeline',
  ElTourStep: 'tour',
}

const elementPlusComponentResolver = {
  type: 'component',
  resolve(name) {
    if (!/^El[A-Z]/.test(name)) return undefined
    if (/^ElIcon.+/.test(name)) {
      return {
        name: name.replace(/^ElIcon/, ''),
        from: '@element-plus/icons-vue',
      }
    }

    const styleName = kebabCase(name.slice(2))
    const componentDir = elementPlusComponentDirs[name] || styleName
    return {
      name,
      from: `element-plus/es/components/${componentDir}/index.mjs`,
      sideEffects: [
        'element-plus/es/components/base/style/css',
        `element-plus/es/components/${styleName}/style/css`,
      ],
    }
  },
}

const elementPlusDirectiveResolver = {
  type: 'directive',
  resolve(name) {
    if (name !== 'Loading') return undefined
    return {
      name: 'ElLoadingDirective',
      from: 'element-plus/es/components/loading/index.mjs',
      sideEffects: [
        'element-plus/es/components/base/style/css',
        'element-plus/es/components/loading/style/css',
      ],
    }
  },
}

export default defineConfig({
  root: resolve(__dirname),
  publicDir: resolve(__dirname, 'public'),
  plugins: [
    vue(),
    Components({
      resolvers: [elementPlusComponentResolver, elementPlusDirectiveResolver],
      dts: resolve(__dirname, 'components.d.ts'),
    }),
  ],
  resolve: {
    alias: [
      {
        find: /^element-plus$/,
        replacement: resolve(__dirname, 'src/utils/elementPlusServices.js'),
      },
      {
        find: '@',
        replacement: resolve(__dirname, 'src'),
      },
    ],
  },
  optimizeDeps: {
    esbuildOptions: {
      target: 'esnext',
    },
  },
  server: {
    port: 5173,
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: process.env.NODE_ENV === 'development', // 仅开发环境开启sourcemap
    minify: 'terser', // 使用 Terser 进行压缩
    cssCodeSplit: true,
    terserOptions: {
      compress: {
        drop_console: process.env.NODE_ENV === 'production', // 生产环境移除console
        drop_debugger: process.env.NODE_ENV === 'production', // 生产环境移除debugger
      },
    },
    rollupOptions: {
      output: {
        entryFileNames: 'assets/[name].[hash].js',
        chunkFileNames: 'assets/[name].[hash].js',
        assetFileNames: 'assets/[name].[hash].[ext]',
        manualChunks(id) {
          if (id.includes('/node_modules/vue') || id.includes('/node_modules/vue-router') || id.includes('/node_modules/pinia')) {
            return 'vue-vendor'
          }
          if (id.includes('/node_modules/chart.js')) {
            return 'charts'
          }
          if (id.includes('/node_modules/axios') || id.includes('/node_modules/dayjs') || id.includes('/node_modules/dompurify')) {
            return 'utils'
          }
        }
      },
    },
    chunkSizeWarningLimit: 1000, // 降低警告阈值，鼓励更好的代码分割
    reportCompressedSize: false,
  },
  css: {
    preprocessorOptions: {
      scss: {
        api: 'modern-compiler',
        silenceDeprecations: ['legacy-js-api'],
      },
    },
  },
})
