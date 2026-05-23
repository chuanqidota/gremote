<template>
  <div class="playback-page">
    <div class="playback-toolbar">
      <span class="back-link" @click="$router.back()">← 返回审计日志</span>
      <span class="toolbar-sep">|</span>
      <span class="toolbar-title">会话回放</span>
      <span class="toolbar-key">{{ key }}</span>
    </div>
    <div v-if="loading" class="loading">
      <span>加载录制中...</span>
    </div>
    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
      <el-button @click="$router.back()">返回</el-button>
    </div>
    <div v-else ref="playerContainer" class="player-container" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import * as AsciinemaPlayer from 'asciinema-player'
import 'asciinema-player/dist/bundle/asciinema-player.css'
import { useAudit } from '../composables/useAudit'

const route = useRoute()
const key = route.query.key as string

const { fetchRecordUrl } = useAudit()
const playerContainer = ref<HTMLDivElement>()
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  if (!key) {
    error.value = '缺少 key 参数'
    loading.value = false
    return
  }

  try {
    const url = await fetchRecordUrl(key)
    loading.value = false
    await nextTick()
    if (playerContainer.value) {
      AsciinemaPlayer.create(url, playerContainer.value, {
        autoPlay: true,
        speed: 1.0,
        idleTimeLimit: 2,
      })
    }
  } catch (e: any) {
    error.value = e?.message || '加载录制失败'
    loading.value = false
  }
})
</script>

<style scoped>
.playback-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #1e1e1e;
}

.playback-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background: #2d2d2d;
  border-bottom: 1px solid #3d3d3d;
  flex-shrink: 0;
}

.back-link {
  font-size: 12px;
  color: #ccc;
  cursor: pointer;
}

.toolbar-sep {
  color: #555;
  font-size: 12px;
}

.toolbar-title {
  font-size: 12px;
  color: #d4d4d4;
}

.toolbar-key {
  font-size: 11px;
  color: #909399;
}

.loading,
.error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  gap: 12px;
  color: #ccc;
  font-size: 16px;
}

.player-container {
  flex: 1;
  padding: 24px;
  max-width: 960px;
  width: 100%;
  margin: 0 auto;
}
</style>
