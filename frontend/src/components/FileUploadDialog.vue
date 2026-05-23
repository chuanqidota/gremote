<template>
  <el-dialog
    :model-value="visible"
    title="上传文件"
    width="480px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <el-form label-width="80px">
      <el-form-item label="目标路径">
        <el-input v-model="uploadPath" placeholder="/tmp" />
      </el-form-item>
      <el-form-item label="选择文件">
        <el-upload
          :auto-upload="false"
          :limit="1"
          :on-change="onFileChange"
          :file-list="fileList"
        >
          <el-button type="primary">选择文件</el-button>
        </el-upload>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:visible', false)">取消</el-button>
      <el-button
        type="primary"
        :loading="loading"
        :disabled="!selectedFile"
        @click="onUpload"
      >
        上传
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, type UploadFile } from 'element-plus'
import type { useFileManager } from '../composables/useFileManager'

const props = defineProps<{
  visible: boolean
  fileManager: ReturnType<typeof useFileManager>
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  uploaded: []
}>()

const uploadPath = ref('/tmp')
const selectedFile = ref<File | null>(null)
const fileList = ref<UploadFile[]>([])
const loading = ref(false)

function onFileChange(file: UploadFile) {
  selectedFile.value = file.raw ?? null
}

async function onUpload() {
  if (!selectedFile.value) return
  loading.value = true
  try {
    await props.fileManager.upload(selectedFile.value, uploadPath.value)
    ElMessage.success('上传成功')
    emit('uploaded')
    emit('update:visible', false)
    selectedFile.value = null
    fileList.value = []
  } catch (e: any) {
    ElMessage.error(e?.message || '上传失败')
  } finally {
    loading.value = false
  }
}
</script>
