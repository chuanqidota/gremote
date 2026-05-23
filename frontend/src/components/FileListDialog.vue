<template>
  <el-dialog
    :model-value="visible"
    title="Browse Files"
    width="640px"
    @update:model-value="$emit('update:visible', $event)"
    @open="onOpen"
  >
    <div class="file-path-bar">
      <el-button size="small" @click="goUp" :disabled="currentPath === '/'">
        Up
      </el-button>
      <el-input :model-value="currentPath" readonly size="small" />
      <el-button size="small" type="primary" @click="$emit('upload')">
        Upload
      </el-button>
    </div>
    <el-table
      :data="files"
      v-loading="loading"
      max-height="400"
      @row-click="onRowClick"
      style="cursor: pointer"
    >
      <el-table-column prop="name" label="Name" />
      <el-table-column prop="size" label="Size" width="120" />
      <el-table-column prop="type" label="Type" width="100">
        <template #default="{ row }">
          {{ row.type === 'directory' ? 'Folder' : 'File' }}
        </template>
      </el-table-column>
      <el-table-column label="Action" width="100">
        <template #default="{ row }">
          <el-button
            v-if="row.type === 'file'"
            size="small"
            type="primary"
            link
            @click.stop="onDownload(row)"
          >
            Download
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
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

function onOpen() {
  props.fileManager.fetchFiles('/tmp')
}

function onRowClick(row: FileItem) {
  if (row.type === 'directory') {
    const sep = currentPath.value.endsWith('/') ? '' : '/'
    props.fileManager.fetchFiles(currentPath.value + sep + row.name)
  }
}

function goUp() {
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  const parent = '/' + parts.join('/')
  props.fileManager.fetchFiles(parent || '/')
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
