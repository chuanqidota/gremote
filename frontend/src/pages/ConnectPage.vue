<template>
  <div class="connect-page">
    <div class="top-nav">
      <div class="nav-left">
        <span class="nav-brand">GRemote</span>
        <span class="nav-tag">控制台</span>
      </div>
      <span class="nav-link" @click="$router.push('/audit')">审计日志 →</span>
    </div>
    <div class="connect-body">
      <div class="connect-card">
        <el-tabs v-model="activeTab" class="connect-tabs">
          <el-tab-pane label="Linux (SSH)" name="ssh">
            <el-form
              ref="sshFormRef"
              :model="sshForm"
              :rules="sshRules"
              label-width="80px"
              status-icon
            >
              <div class="card-title">SSH 连接</div>
              <div class="card-subtitle">输入主机信息以建立远程连接</div>
              <el-form-item label="主机地址" prop="target">
                <el-input v-model="sshForm.target" placeholder="192.168.1.1" />
              </el-form-item>
              <el-row :gutter="14">
                <el-col :span="14">
                  <el-form-item label="用户名" prop="username">
                    <el-input v-model="sshForm.username" placeholder="root" />
                  </el-form-item>
                </el-col>
                <el-col :span="10">
                  <el-form-item label="端口" prop="port" label-width="58px">
                    <el-input v-model.number="sshForm.port" placeholder="22" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="密码" prop="password">
                <el-input v-model="sshForm.password" type="password" show-password />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="loading" @click="connectSSH" style="width: 100%">
                  连 接
                </el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>
          <el-tab-pane label="Windows (RDP)" name="rdp">
            <el-form
              ref="rdpFormRef"
              :model="rdpForm"
              :rules="rdpRules"
              label-width="80px"
              status-icon
            >
              <div class="card-title">RDP 连接</div>
              <div class="card-subtitle">输入Windows主机信息以建立远程桌面</div>
              <el-form-item label="主机地址" prop="target">
                <el-input v-model="rdpForm.target" placeholder="192.168.1.1" />
              </el-form-item>
              <el-row :gutter="14">
                <el-col :span="14">
                  <el-form-item label="用户名" prop="username">
                    <el-input v-model="rdpForm.username" placeholder="Administrator" />
                  </el-form-item>
                </el-col>
                <el-col :span="10">
                  <el-form-item label="端口" prop="port" label-width="58px">
                    <el-input v-model.number="rdpForm.port" placeholder="3389" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-form-item label="密码" prop="password">
                <el-input v-model="rdpForm.password" type="password" show-password />
              </el-form-item>
              <el-form-item label="域名">
                <el-input v-model="rdpForm.domain" placeholder="可选，如 WORKGROUP" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="loading" @click="connectRDP" style="width: 100%">
                  连 接
                </el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { obtainKey, obtainKeyRDP } from '../api'
import type { SSHInfo, RDPInfo } from '../types'

const activeTab = ref('ssh')
const loading = ref(false)

const sshFormRef = ref<FormInstance>()
const sshForm = reactive<SSHInfo>({
  target: '',
  port: 22,
  username: '',
  password: '',
})

const sshRules: FormRules = {
  target: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

const rdpFormRef = ref<FormInstance>()
const rdpForm = reactive<RDPInfo>({
  target: '',
  port: 3389,
  username: '',
  password: '',
  domain: '',
})

const rdpRules: FormRules = {
  target: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function connectSSH() {
  const valid = await sshFormRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const key = await obtainKey(sshForm)
    window.open(`/term?key=${key}`, '_blank')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.msg || e?.message || '连接失败')
  } finally {
    loading.value = false
  }
}

async function connectRDP() {
  const valid = await rdpFormRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const key = await obtainKeyRDP(rdpForm)
    window.open(`/rdp?key=${key}&host=${rdpForm.target}`, '_blank')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.msg || e?.message || '连接失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.connect-page {
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

.connect-body {
  display: flex;
  justify-content: center;
  align-items: center;
  flex: 1;
  padding: 24px;
}

.connect-card {
  width: 420px;
  padding: 28px 32px;
  background: #fff;
  border-radius: 4px;
  border: 1px solid #ebeef0;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
}

.connect-tabs {
  width: 100%;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 6px;
}

.card-subtitle {
  font-size: 12px;
  color: #909399;
  margin-bottom: 22px;
}
</style>
