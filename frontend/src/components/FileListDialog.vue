<template>
  <el-dialog
    :model-value="visible"
    title="文件浏览"
    width="640px"
    @update:model-value="$emit('update:visible', $event)"
    @open="onOpen"
  >
    <div class="file-path-bar">
      <el-input
        v-model="customPath"
        size="small"
        placeholder="输入路径，回车跳转"
        @keyup.enter="navigateToPath"
      />
      <el-button size="small" @click="navigateToPath">跳转</el-button>
      <el-button size="small" type="primary" @click="$emit('upload')">
        上传
      </el-button>
    </div>
    <el-table
      :data="files"
      v-loading="loading"
      max-height="400"
      @row-click="onRowClick"
      style="cursor: pointer"
    >
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="size" label="大小" width="120" />
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          {{ row.type === 'directory' ? '文件夹' : '文件' }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button
            v-if="row.type === 'file'"
            size="small"
            type="primary"
            link
            @click.stop="onDownload(row)"
          >
            下载
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { FileItem } from '../types'
import type { useFileManager } from '../composables/useFileManager'

const props = defineProps<{
  visible: boolean
  fileManager: ReturnType<typeof useFileManager>
}>()

defineEmits<{
  'update:visible': [value: boolean]
  upload: []
}>()

const files = computed(() => props.fileManager.files.value)
const loading = computed(() => props.fileManager.loading.value)
const currentPath = computed(() => props.fileManager.currentPath.value)
const customPath = ref(currentPath.value)

watch(currentPath, (val) => {
  customPath.value = val
})

function onOpen() {
  customPath.value = props.fileManager.currentPath.value || '/tmp'
  props.fileManager.fetchFiles(customPath.value)
}

function navigateToPath() {
  const path = customPath.value.trim() || '/tmp'
  props.fileManager.fetchFiles(path)
}

function onRowClick(row: FileItem) {
  if (row.type === 'directory') {
    const sep = currentPath.value.endsWith('/') ? '' : '/'
    props.fileManager.fetchFiles(currentPath.value + sep + row.name)
  }
}

function onDownload(row: FileItem) {
  props.fileManager.download(currentPath.value, row.name)
}
</script>

<style scoped>
.file-path-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  align-items: center;
}
</style>
