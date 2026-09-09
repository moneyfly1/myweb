<template>
  <div v-if="attachments && attachments.length" class="ticket-attachments">
    <div class="attachments-title">{{ title }}</div>
    <div class="attachment-grid">
      <template v-for="att in attachments" :key="att.id || att.file_path">
        <!-- 图片预览 -->
        <el-image
          v-if="isImageType(att) || isImageName(att)"
          :src="att.file_path"
          :preview-src-list="imageList"
          fit="cover"
          class="attachment-image"
          :preview-teleported="true"
          hide-on-click-modal
        >
          <template #error>
            <div class="att-fallback"><el-icon><Picture /></el-icon><span>{{ shortName(att) }}</span></div>
          </template>
        </el-image>
        <!-- 视频 -->
        <div v-else-if="isVideoName(att)" class="attachment-video-wrap">
          <video :src="att.file_path" controls preload="metadata" class="attachment-video"></video>
          <div class="video-name">{{ shortName(att) }}</div>
        </div>
        <!-- 其他附件：下载卡片 -->
        <div v-else class="attachment-file" @click="openAttachment(att)">
          <div class="att-file-icon"><el-icon><Document /></el-icon></div>
          <div class="att-file-info">
            <div class="att-file-name" :title="att.file_name || att.file_path">{{ att.file_name || shortName(att) }}</div>
            <div class="att-file-meta">{{ att.file_size ? formatSize(att.file_size) : '' }} · 点击下载</div>
          </div>
          <el-icon class="att-download-icon"><Download /></el-icon>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Picture, Document, Download } from '@element-plus/icons-vue'

const props = defineProps({
  attachments: { type: Array, default: () => [] },
  title: { type: String, default: '附件' }
})

const IMAGE_TYPES = ['image', 'image/jpeg', 'image/png', 'image/gif', 'image/webp']
const IMAGE_EXTS = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg']
const VIDEO_EXTS = ['mp4', 'mov', 'avi', 'mkv', 'webm', 'm4v']

const isImageType = (att) => IMAGE_TYPES.includes(att.file_type) || String(att.file_type || '').startsWith('image')
const isImageName = (att) => {
  const n = String(att.file_name || att.file_path || '').toLowerCase()
  return IMAGE_EXTS.some(e => n.endsWith('.' + e))
}
const isVideoName = (att) => {
  const n = String(att.file_name || att.file_path || '').toLowerCase()
  return VIDEO_EXTS.some(e => n.endsWith('.' + e))
}
const shortName = (att) => {
  const n = att.file_name || ''
  return n.length > 14 ? n.slice(0, 12) + '…' : (n || '附件')
}
const imageList = computed(() => {
  return (props.attachments || []).filter(a => isImageType(a) || isImageName(a)).map(a => a.file_path)
})
const formatSize = (bytes) => {
  if (!bytes) return ''
  if (bytes < 1024) return bytes + 'B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + 'KB'
  return (bytes / 1024 / 1024).toFixed(1) + 'MB'
}
const openAttachment = (att) => {
  if (att.file_path) window.open(att.file_path, '_blank')
}
</script>

<style scoped>
.ticket-attachments {
  margin-top: 8px;
}
.attachments-title {
  font-size: 11px;
  color: #909399;
  margin-bottom: 6px;
}
.attachment-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.attachment-image {
  width: 84px;
  height: 84px;
  border-radius: 6px;
  border: 1px solid #e4e7ed;
  cursor: pointer;
}
.att-fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  gap: 4px;
  font-size: 10px;
  color: #909399;
  background: #f5f7fa;
  padding: 4px;
  box-sizing: border-box;
}
.attachment-video-wrap {
  width: 200px;
}
.attachment-video {
  width: 200px;
  max-height: 130px;
  border-radius: 6px;
  background: #000;
}
.video-name {
  font-size: 11px;
  color: #606266;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
}
.attachment-file {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  cursor: pointer;
  max-width: 260px;
  background: #fafafa;
  transition: border-color 0.2s;
}
.attachment-file:hover {
  border-color: #409eff;
}
.att-file-icon {
  color: #409eff;
  font-size: 20px;
  flex-shrink: 0;
}
.att-file-info {
  min-width: 0;
  flex: 1;
}
.att-file-name {
  font-size: 12px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.att-file-meta {
  font-size: 10px;
  color: #909399;
  margin-top: 2px;
}
.att-download-icon {
  color: #909399;
  font-size: 14px;
  flex-shrink: 0;
}
</style>
