<template>
  <div class="top-nav">
    <div class="nav-left">
      <span class="nav-brand">GRemote</span>
      <span class="nav-tag">控制台</span>
    </div>
    <span class="nav-link" @click="$router.push(to)">{{ linkText }}</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{ linkText: string; to: string }>()
</script>

<style scoped>
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
</style>
