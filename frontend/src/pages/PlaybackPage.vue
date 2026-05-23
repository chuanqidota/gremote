<template>
  <div class="playback-page">
    <div v-if="loading" class="loading">
      <span>Loading recording...</span>
    </div>
    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
      <el-button @click="$router.back()">Go Back</el-button>
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
    error.value = 'Missing key parameter'
    loading.value = false
    return
  }

  try {
    const url = await fetchRecordUrl(key)
    await nextTick()
    if (playerContainer.value) {
      AsciinemaPlayer.create(url, playerContainer.value, {
        autoPlay: true,
        speed: 1.0,
        idleTimeLimit: 2,
      })
    }
  } catch (e: any) {
    error.value = e?.message || 'Failed to load recording'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.playback-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #1e1e1e;
}

.loading,
.error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #ccc;
  font-size: 16px;
}

.player-container {
  width: 100%;
  max-width: 960px;
}
</style>
