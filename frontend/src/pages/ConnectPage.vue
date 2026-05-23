<template>
  <div class="connect-page">
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="100px"
      class="connect-form"
      status-icon
    >
      <h2>SSH Connection</h2>
      <el-form-item label="Host" prop="target">
        <el-input v-model="form.target" placeholder="192.168.1.1" />
      </el-form-item>
      <el-form-item label="Port" prop="port">
        <el-input v-model.number="form.port" placeholder="22" />
      </el-form-item>
      <el-form-item label="Username" prop="username">
        <el-input v-model="form.username" placeholder="root" />
      </el-form-item>
      <el-form-item label="Password" prop="password">
        <el-input v-model="form.password" type="password" show-password />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" @click="onSubmit">
          Connect
        </el-button>
        <el-button @click="onReset">Reset</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { obtainKey } from '../api'
import type { SSHInfo } from '../types'

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive<SSHInfo>({
  target: '',
  port: 22,
  username: '',
  password: '',
})

const rules: FormRules = {
  target: [{ required: true, message: 'Host is required', trigger: 'blur' }],
  port: [{ required: true, message: 'Port is required', trigger: 'blur' }],
  username: [{ required: true, message: 'Username is required', trigger: 'blur' }],
  password: [{ required: true, message: 'Password is required', trigger: 'blur' }],
}

async function onSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const key = await obtainKey(form)
    window.open(`/term?key=${key}`, '_blank')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.msg || e?.message || 'Connection failed')
  } finally {
    loading.value = false
  }
}

function onReset() {
  formRef.value?.resetFields()
}
</script>

<style scoped>
.connect-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f5f7fa;
}

.connect-form {
  width: 420px;
  padding: 32px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.connect-form h2 {
  text-align: center;
  margin-bottom: 24px;
  color: #303133;
}
</style>
