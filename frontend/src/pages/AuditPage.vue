<template>
  <div class="audit-page">
    <div class="top-nav">
      <div class="nav-left">
        <span class="nav-brand">GRemote</span>
        <span class="nav-tag">控制台</span>
      </div>
      <span class="nav-link" @click="$router.push('/connect')">远程连接 →</span>
    </div>
    <div class="audit-body">
      <div class="audit-card">
        <div class="audit-header">
          <span class="audit-title">登录审计</span>
          <div class="audit-filters">
            <el-input
              v-model="search"
              placeholder="搜索用户 / 来源 / 目标..."
              clearable
              style="width: 220px"
              size="small"
              @input="onSearch"
            />
            <el-date-picker
              v-model="dateRange"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DD HH:mm:ss"
              size="small"
              @change="onDateChange"
            />
            <el-button type="primary" size="small" @click="fetchData">查询</el-button>
          </div>
        </div>
        <el-tabs v-model="activeTab" @tab-change="onTabChange">
          <el-tab-pane v-if="displayMode !== 'windows'" label="Linux (SSH)" name="linux">
            <el-table :data="data" v-loading="loading" border stripe>
              <el-table-column prop="user" label="用户" width="100" />
              <el-table-column prop="source" label="来源 IP" width="140" />
              <el-table-column prop="target" label="目标主机" width="140" />
              <el-table-column prop="startTime" label="开始时间" width="180" />
              <el-table-column prop="endTime" label="结束时间" width="180" />
              <el-table-column prop="key" label="会话 Key" min-width="200" show-overflow-tooltip />
              <el-table-column label="操作" width="80" fixed="right">
                <template #default="{ row }">
                  <el-button type="primary" link size="small" @click="onPlayback(row.key, 'ssh')">
                    回放
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <el-tab-pane v-if="displayMode !== 'linux'" label="Windows (RDP)" name="windows">
            <el-table :data="data" v-loading="loading" border stripe>
              <el-table-column prop="user" label="用户" width="100" />
              <el-table-column prop="source" label="来源 IP" width="140" />
              <el-table-column prop="target" label="目标主机" width="140" />
              <el-table-column prop="startTime" label="开始时间" width="180" />
              <el-table-column prop="endTime" label="结束时间" width="180" />
              <el-table-column prop="key" label="会话 Key" min-width="200" show-overflow-tooltip />
              <el-table-column label="操作" width="80" fixed="right">
                <template #default="{ row }">
                  <el-button type="primary" link size="small" @click="onPlayback(row.key, 'rdp')">
                    回放
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="count"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          style="margin-top: 14px; justify-content: flex-end"
          @size-change="fetchData"
          @current-change="fetchData"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAudit } from '../composables/useAudit'
import { getConfig } from '../api'

const { data, count, loading, fetch } = useAudit()

const activeTab = ref('linux')
const search = ref('')
const dateRange = ref<[string, string] | null>(null)
const page = ref(1)
const pageSize = ref(10)
const displayMode = ref('all')

let searchTimer: ReturnType<typeof setTimeout> | null = null

function buildQuery() {
  let protocol: string
  if (displayMode.value === 'all') {
    protocol = activeTab.value === 'windows' ? 'rdp' : 'ssh'
  } else if (displayMode.value === 'linux') {
    protocol = 'ssh'
  } else {
    protocol = 'rdp'
  }
  return {
    offset: (page.value - 1) * pageSize.value,
    limit: pageSize.value,
    search: search.value || undefined,
    startTime: dateRange.value?.[0],
    endTime: dateRange.value?.[1],
    protocol,
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

function onTabChange() {
  page.value = 1
  fetchData()
}

function onPlayback(key: string, protocol: string) {
  window.open(`/playback?key=${key}&protocol=${protocol}`, '_blank')
}

onMounted(async () => {
  try {
    const config = await getConfig()
    displayMode.value = config.display_mode || 'all'
    if (displayMode.value === 'windows') {
      activeTab.value = 'windows'
    }
  } catch {
    displayMode.value = 'all'
  }
  fetchData()
})
</script>

<style scoped>
.audit-page {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: #f0f2f5;
}

.top-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 44px;
  padding: 0 20px;
  background: #fff;
  border-bottom: 1px solid #e8e8e8;
  flex-shrink: 0;
}

.nav-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.nav-brand {
  font-weight: 700;
  font-size: 15px;
  color: #006eff;
}

.nav-tag {
  font-size: 11px;
  color: #909399;
  background: #f0f2f5;
  padding: 1px 6px;
  border-radius: 2px;
}

.nav-link {
  font-size: 13px;
  color: #006eff;
  cursor: pointer;
}

.audit-body {
  padding: 20px;
  flex: 1;
}

.audit-card {
  background: #fff;
  border-radius: 4px;
  border: 1px solid #ebeef0;
  overflow: hidden;
  padding: 0 0 16px;
}

.audit-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  border-bottom: 1px solid #ebeef0;
}

.audit-title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.audit-filters {
  display: flex;
  gap: 8px;
  align-items: center;
}
</style>
