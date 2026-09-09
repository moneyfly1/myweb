<template>
  <div class="ticket-attachment-upload">
    <div class="upload-buttons">
      <!-- 从相册/拍照选图（图片类型，iOS 会弹出拍照/相册选项） -->
      <el-button
        size="small"
        type="primary"
        plain
        :loading="uploadingImage"
        @click="triggerPicker('image')"
      >
        <el-icon><Picture /></el-icon>
        <span>相册选图</span>
      </el-button>
      <!-- 选择文件（视频 / 文档 / 压缩包等） -->
      <el-button
        size="small"
        :loading="uploadingFile"
        @click="triggerPicker('file')"
      >
        <el-icon><FolderOpened /></el-icon>
        <span>选择文件</span>
      </el-button>
      <span v-if="fileList.length" class="file-count">已选 {{ fileList.length }} 个</span>
    </div>
    <div class="upload-tip">{{ tip }}</div>
    <!-- 已选文件列表 -->
    <div v-if="fileList.length" class="uploaded-list">
      <div v-for="(f, idx) in fileList" :key="idx" class="uploaded-item">
        <div class="up-item-left">
          <el-icon v-if="isImage(f)" class="icon-img"><Picture /></el-icon>
          <el-icon v-else-if="isVideo(f)" class="icon-video"><VideoCamera /></el-icon>
          <el-icon v-else class="icon-file"><Document /></el-icon>
          <span class="up-item-name" :title="f.file_name || f.name">{{ f.file_name || f.name }}</span>
        </div>
        <div class="up-item-actions">
          <el-icon class="act" title="查看" @click="handlePreview(f)"><View /></el-icon>
          <el-icon class="act act-del" title="移除" @click="handleRemove(f)"><Delete /></el-icon>
        </div>
      </div>
    </div>
    <!-- 隐藏的 input[type=file]，配合按钮触发，移动端可弹相册/文件选择器 -->
    <input
      ref="pickerRef"
      type="file"
      class="hidden-file-input"
      @change="onPickerChange"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from '@/utils/elementPlusServices'
import { ticketAPI } from '@/utils/api'
import { Picture, FolderOpened, VideoCamera, Document, View, Delete } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  tip: { type: String, default: '支持图片 / 视频 / 文档 / 压缩包，单文件最大 30MB' },
  limit: { type: Number, default: 9 }
})
const emit = defineEmits(['update:modelValue', 'change'])

const pickerRef = ref(null)
const uploadingImage = ref(false)
const uploadingFile = ref(false)
const currentPickerType = ref('image')
const fileList = ref([])

const IMAGE_EXTS = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg']
const VIDEO_EXTS = ['mp4', 'mov', 'avi', 'mkv', 'webm', 'm4v']
// 从手机相册选择时只允许图片；从文件选择时允许所有支持的扩展
const ACCEPT_ALL = '.jpg,.jpeg,.png,.gif,.webp,.bmp,.svg,.mp4,.mov,.avi,.mkv,.webm,.m4v,.mp3,.wav,.pdf,.txt,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.zip,.rar,.7z,.csv,.log'
const ACCEPT_IMAGE = 'image/*'

const isImage = (f) => {
  const name = (f.file_name || f.name || '').toLowerCase()
  return IMAGE_EXTS.some(e => name.endsWith('.' + e)) || String(f.file_type || '').startsWith('image')
}
const isVideo = (f) => {
  const name = (f.file_name || f.name || '').toLowerCase()
  return VIDEO_EXTS.some(e => name.endsWith('.' + e))
}

// 触发隐藏的 input[type=file] —— 关键：每次选择后重置 value，
// 保证同一个文件可以再次选择；accept 区分图片来源，移动端可弹相册
const triggerPicker = (type) => {
  if (fileList.value.length >= props.limit) {
    ElMessage.warning(`最多上传 ${props.limit} 个文件`)
    return
  }
  currentPickerType.value = type
  const input = pickerRef.value
  if (!input) return
  input.accept = type === 'image' ? ACCEPT_IMAGE : ACCEPT_ALL
  input.value = ''
  input.click()
}

const onPickerChange = async (e) => {
  const input = e.target
  const files = Array.from(input.files || [])
  if (!files.length) return
  input.value = '' // 允许重复选择同一文件
  for (const file of files) {
    if (fileList.value.length >= props.limit) {
      ElMessage.warning(`最多上传 ${props.limit} 个文件`)
      break
    }
    const isImagePick = currentPickerType.value === 'image'
    // 相册入口如果选了视频等（部分安卓会放开），做类型校验
    if (isImagePick && !file.type.startsWith('image/')) {
      ElMessage.warning(`${file.name} 不是图片，请使用「选择文件」上传`)
      continue
    }
    if (isImagePick) uploadingImage.value = true
    else uploadingFile.value = true
    // eslint-disable-next-line no-await-in-loop
    await doUpload(file)
  }
  uploadingImage.value = false
  uploadingFile.value = false
}

const doUpload = async (file) => {
  const maxSize = 30 * 1024 * 1024
  if (file.size > maxSize) {
    ElMessage.error(`文件超过 30MB 上限：${file.name}`)
    return
  }
  const formData = new FormData()
  formData.append('file', file)
  try {
    const res = await ticketAPI.uploadAttachment(formData)
    const data = res.data?.data || {}
    if (res.data?.success && data.url) {
      fileList.value.push({
        name: file.name,
        file_name: data.file_name || file.name,
        file_path: data.url,
        file_size: data.file_size || file.size || 0,
        file_type: data.file_type || 'file',
        raw_type: file.type || ''
      })
      pushToModel()
      ElMessage.success(`「${file.name}」上传成功`)
    } else {
      ElMessage.error(res.data?.message || '上传失败')
    }
  } catch (e) {
    ElMessage.error(`「${file.name}」上传失败: ` + (e.response?.data?.message || e.message || '网络错误'))
  }
}

const pushToModel = () => {
  // 供父组件提交的数组：file_name / file_path / file_size / file_type
  const out = fileList.value.map(f => ({
    file_name: f.file_name || f.name || '',
    file_path: f.file_path || '',
    file_size: f.file_size || 0,
    file_type: f.file_type || 'file'
  }))
  emit('update:modelValue', out)
  emit('change', out)
}

const handleRemove = (f) => {
  fileList.value = fileList.value.filter(item => item.file_path !== f.file_path)
  pushToModel()
}

const handlePreview = (f) => {
  if (f.file_path) {
    // ?raw=1 独立 CF 缓存键，规避旧的 SPA-fallback HTML 缓存
    window.open(f.file_path.includes('?') ? f.file_path : f.file_path + '?raw=1', '_blank')
  }
}

defineExpose({ clear: () => { fileList.value = []; pushToModel() } })
</script>

<style scoped>
.ticket-attachment-upload {
  width: 100%;
}
.upload-buttons {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.upload-buttons .el-button {
  margin: 0;
}
.file-count {
  font-size: 12px;
  color: #909399;
}
.upload-tip {
  font-size: 11px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}
.uploaded-list {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.uploaded-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: #fafafa;
}
.up-item-left {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex: 1;
}
.up-item-left .el-icon {
  flex-shrink: 0;
  font-size: 16px;
}
.icon-img { color: #67c23a; }
.icon-video { color: #e6a23c; }
.icon-file { color: #409eff; }
.up-item-name {
  font-size: 12px;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.up-item-actions {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}
.up-item-actions .act {
  cursor: pointer;
  color: #909399;
  font-size: 15px;
}
.up-item-actions .act:hover { color: #409eff; }
.up-item-actions .act-del:hover { color: #f56c6c; }
.hidden-file-input {
  display: none;
}
</style>
