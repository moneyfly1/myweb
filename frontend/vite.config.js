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

const previewJson = (res, data) => {
  res.statusCode = 200
  res.setHeader('Content-Type', 'application/json; charset=utf-8')
  res.end(JSON.stringify({ success: true, data }))
}

const userPreviewMockPlugin = () => ({
  name: 'user-preview-mock',
  configureServer(server) {
    if (process.env.VITE_USER_PREVIEW_MOCK !== '1') return

    const now = new Date().toISOString()
    const dashboardInfo = {
      username: 'preview',
      balance: '128.50',
      membership: '普通会员',
      expire_time: '2026-12-31',
      expiryDate: '2026-12-31',
      remainingDays: 49,
      online_devices: 3,
      total_devices: 5,
      clashUrl: 'https://example.com/sub/clash',
      universalUrl: 'https://example.com/sub/universal',
      qrcodeUrl: '',
      user_level: { name: '普通会员', discount_rate: 1, color: '#409eff' },
      subscription: {
        status: 'active',
        currentDevices: 3,
        maxDevices: 5,
        expiryDate: '2026-12-31',
        expire_time: '2026-12-31',
        clashUrl: 'https://example.com/sub/clash',
        universalUrl: 'https://example.com/sub/universal',
        qrcodeUrl: ''
      }
    }
    const packages = [
      { id: 1, name: '基础套餐', price: 29, duration_days: 30, device_limit: 3, description: '适合个人日常使用', features: ['3 台设备', '标准线路', '邮件支持'], is_popular: false, is_recommended: false },
      { id: 2, name: '进阶套餐', price: 69, duration_days: 90, device_limit: 5, description: '适合多设备长期使用', features: ['5 台设备', '更多地区', '优先支持'], is_popular: true, is_recommended: true },
      { id: 3, name: '年度套餐', price: 199, duration_days: 365, device_limit: 8, description: '适合长期稳定使用', features: ['8 台设备', '年度折扣', '专属客服'], is_popular: false, is_recommended: true }
    ]
    const orders = [
      { id: 1, order_no: 'ORD20260701001', display_no: 'ORD20260701001', package_name: '进阶套餐', amount: 69, display_amount: '69.00', payment_method: 'alipay', status: 'paid', created_at: now, paid_at: now, record_type: 'order' },
      { id: 2, order_no: 'REC20260701001', display_no: 'REC20260701001', package_name: '账户充值', amount: 100, display_amount: '100.00', payment_method: 'alipay', status: 'paid', created_at: now, paid_at: now, record_type: 'recharge' },
      { id: 3, order_no: 'ORD20260701002', display_no: 'ORD20260701002', package_name: '升级设备数量', amount: 20, display_amount: '20.00', payment_method: 'balance', status: 'pending', created_at: now, paid_at: null, record_type: 'order' }
    ]
    const devices = [
      { id: 1, device_name: 'MacBook Pro', device_type: 'desktop', os_name: 'macOS', os_version: '15', ip_address: '203.0.113.10', location: '中国 上海', user_agent: 'Clash Verge', remark: '主力电脑', last_access: now },
      { id: 2, device_name: 'iPhone', device_type: 'mobile', os_name: 'iOS', os_version: '18', ip_address: '203.0.113.11', location: '中国 上海', user_agent: 'Shadowrocket', remark: '手机', last_access: now },
      { id: 3, device_name: 'Windows PC', device_type: 'desktop', os_name: 'Windows', os_version: '11', ip_address: '203.0.113.12', location: '中国 北京', user_agent: 'Clash for Windows', remark: '', last_access: now }
    ]
    const nodes = [
      { id: 1, name: '香港 01', country: '香港', region: 'HK', status: 'online', description: '标准线路' },
      { id: 2, name: '日本 01', country: '日本', region: 'JP', status: 'online', description: '标准线路' },
      { id: 3, name: '新加坡 01', country: '新加坡', region: 'SG', status: 'maintenance', description: '维护中' }
    ]

    server.middlewares.use('/api/v1', (req, res, next) => {
      const path = (req.url || '').split('?')[0]
      if (path === '/settings/public-settings') return previewJson(res, { site_name: 'CBoard', default_theme: 'light' })
      if (path === '/tickets/unread-count') return previewJson(res, { count: 2 })
      if (path === '/users/checkin/status') return previewJson(res, { checked_in: false })
      if (path === '/users/dashboard-info') return previewJson(res, dashboardInfo)
      if (path === '/users/me') return previewJson(res, { id: 1, username: 'preview', email: 'preview@example.com', balance: '128.50', is_admin: false })
      if (path === '/users/devices' || path === '/subscriptions/devices') return previewJson(res, devices)
      if (path === '/users/login-history') return previewJson(res, [{ id: 1, ip_address: '203.0.113.10', location: '中国 上海', device_info: 'Chrome / macOS', success: true, created_at: now }])
      if (path === '/subscriptions/user-subscription') return previewJson(res, dashboardInfo.subscription)
      if (path === '/packages/' || path === '/packages') return previewJson(res, packages)
      if (path === '/orders/' || path === '/orders') return previewJson(res, { items: orders, total: orders.length })
      if (path === '/recharge') return previewJson(res, orders.filter(item => item.record_type === 'recharge'))
      if (path === '/nodes/' || path === '/nodes') return previewJson(res, nodes)
      if (path === '/payment-methods/active') return previewJson(res, [{ id: 1, code: 'alipay', name: '支付宝' }, { id: 2, code: 'balance', name: '余额支付' }])
      if (path === '/software-config/') return previewJson(res, { clash: '', shadowrocket: '', v2ray: '', hiddify: '' })
      if (path === '/knowledge/categories') return previewJson(res, [{ id: 1, name: '使用指南' }, { id: 2, name: '客户端' }])
      if (path === '/knowledge/articles') return previewJson(res, { items: [{ id: 1, title: '如何导入订阅', category_name: '使用指南', created_at: now }], total: 1 })
      if (path === '/tickets/') return previewJson(res, { items: [{ id: 1, title: '订阅无法导入', status: 'open', type: 'technical', unread_count: 1, created_at: now }], total: 1 })
      if (path === '/invites/my-codes') return previewJson(res, [{ id: 1, code: 'PREVIEW2026', uses: 3, max_uses: 10, expires_at: '2026-12-31' }])
      if (path === '/invites/stats') return previewJson(res, { total_invites: 3, total_rewards: '30.00' })
      if (path === '/invites/reward-settings') return previewJson(res, { enabled: true, reward_amount: '10.00' })
      if (req.method === 'POST' || req.method === 'PUT' || req.method === 'DELETE') return previewJson(res, { ok: true })
      return next()
    })
  }
})

export default defineConfig({
  root: resolve(__dirname),
  publicDir: resolve(__dirname, 'public'),
  plugins: [
    userPreviewMockPlugin(),
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
