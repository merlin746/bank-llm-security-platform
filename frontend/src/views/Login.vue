<template>
  <div class="login-wrap">
    <div class="login-card card">
      <div class="login-head">
        <el-icon :size="34" color="#1e6fff"><Lock /></el-icon>
        <h2>链安智御</h2>
        <p>面向银行大模型生产应用的全链路安全管控平台</p>
      </div>
      <el-form :model="form" :rules="rules" ref="formRef" @keyup.enter="handleLogin">
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名" :prefix-icon="User" />
        </el-form-item>
        <el-form-item prop="password">
          <el-input v-model="form.password" type="password" placeholder="密码" show-password :prefix-icon="Key" />
        </el-form-item>
        <el-button type="primary" class="submit" :loading="loading" @click="handleLogin">
          登 录
        </el-button>
      </el-form>
      <p class="hint">Mock 模式下任意账号密码均可登录（演示用）</p>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Key } from '@element-plus/icons-vue'
import { useUserStore } from '@/store/user'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref()
const loading = ref(false)
const form = reactive({ username: 'admin', password: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

async function handleLogin() {
  await formRef.value.validate()
  loading.value = true
  try {
    await userStore.login(form)
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } catch (e) {
    ElMessage.error(e.message || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0f2b52 0%, #1e6fff 100%);
}
.login-card {
  width: 380px;
  padding: 36px 32px;
}
.login-head {
  text-align: center;
  margin-bottom: 24px;
}
.login-head h2 {
  margin: 8px 0 4px;
}
.login-head p {
  margin: 0;
  font-size: 12px;
  color: #909399;
}
.submit {
  width: 100%;
}
.hint {
  margin: 12px 0 0;
  font-size: 12px;
  color: #c0c4cc;
  text-align: center;
}
</style>
