<template>
  <div class="audit-page">
    <h2>Login Audit</h2>

    <div class="audit-filters">
      <el-input
        v-model="search"
        placeholder="Search user, source, or target..."
        clearable
        style="width: 280px"
        @input="onSearch"
      />
      <el-date-picker
        v-model="dateRange"
        type="datetimerange"
        range-separator="to"
        start-placeholder="Start"
        end-placeholder="End"
        format="YYYY-MM-DD HH:mm:ss"
        value-format="YYYY-MM-DD HH:mm:ss"
        @change="onDateChange"
      />
      <el-button type="primary" @click="fetchData">Query</el-button>
    </div>

    <el-table :data="data" v-loading="loading" border stripe style="margin-top: 16px">
      <el-table-column prop="user" label="User" width="120" />
      <el-table-column prop="source" label="Source" width="160" />
      <el-table-column prop="target" label="Target" width="160" />
      <el-table-column prop="startTime" label="Start Time" width="180" />
      <el-table-column prop="endTime" label="End Time" width="180" />
      <el-table-column prop="key" label="Key" min-width="200" show-overflow-tooltip />
      <el-table-column label="Actions" width="100" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link @click="onPlayback(row.key)">
            Playback
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="page"
      v-model:page-size="pageSize"
      :total="count"
      :page-sizes="[10, 20, 50]"
      layout="total, sizes, prev, pager, next"
      style="margin-top: 16px; justify-content: flex-end"
      @size-change="fetchData"
      @current-change="fetchData"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAudit } from '../composables/useAudit'

const { data, count, loading, fetch } = useAudit()

const search = ref('')
const dateRange = ref<[string, string] | null>(null)
const page = ref(1)
const pageSize = ref(10)

let searchTimer: ReturnType<typeof setTimeout> | null = null

function buildQuery() {
  return {
    offset: (page.value - 1) * pageSize.value,
    limit: pageSize.value,
    search: search.value || undefined,
    startTime: dateRange.value?.[0],
    endTime: dateRange.value?.[1],
  }
}

function fetchData() {
  fetch(buildQuery())
}

function onSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    fetchData()
  }, 300)
}

function onDateChange() {
  page.value = 1
}

function onPlayback(key: string) {
  window.open(`/playback?key=${key}`, '_blank')
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.audit-page {
  padding: 24px;
  background: #f5f7fa;
  min-height: 100vh;
}

.audit-page h2 {
  margin-bottom: 16px;
  color: #303133;
}

.audit-filters {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
