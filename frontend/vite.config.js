import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import { fileURLToPath, URL } from 'node:url'

// 获取当前文件所在目录
const __dirname = fileURLToPath(new URL('.', import.meta.url))

export default defineConfig({
  // 显式指定项目根目录（确保 Vite 能找到 index.html）
  root: resolve(__dirname),
  // 显式指定公共资源目录
  publicDir: resolve(__dirname, 'public'),
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
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
        // 文件名包含 hash，确保更新后浏览器能获取新文件
        entryFileNames: 'assets/[name].[hash].js',
        chunkFileNames: 'assets/[name].[hash].js',
        assetFileNames: 'assets/[name].[hash].[ext]',
        // manualChunks: (id) => {
        //   if (id.includes('node_modules')) {
        //     if (id.includes('vue') || id.includes('pinia') || id.includes('router')) {
        //       return 'vendor-core'
        //     }
        //     if (id.includes('element-plus')) {
        //       return 'vendor-ui'
        //     }
        //     if (id.includes('chart.js')) {
        //       return 'vendor-chart'
        //     }
        //     return 'vendor-libs'
        //   }
        // },
      },
    },
    chunkSizeWarningLimit: 2000,
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
