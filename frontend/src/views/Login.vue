<template>
  <div v-feedback class="login-wrap">
    <div class="login-brand">
      <BrandLogo />
    </div>
    <el-button class="theme-switch" :aria-label="isDark ? '切换浅色主题' : '切换深色主题'" @click="toggleTheme">
      <el-icon><Sunny v-if="isDark" /><Moon v-else /></el-icon>
    </el-button>
    <main class="login-main">
      <section class="login-intro">
        <h1>面向银行大模型生产应用的<br class="desktop-break" />全链路安全管控平台</h1>
        <p>事前硬拦截、事中语义脱敏、事后链上定责。</p>
        <div class="security-path" aria-label="安全管控闭环">
          <div><el-icon><Lock /></el-icon><span>访问管控</span><small>角色与数据密级校验</small></div>
          <el-icon class="path-arrow" aria-hidden="true"><ArrowRight /></el-icon>
          <div><el-icon><Aim /></el-icon><span>语义防护</span><small>输入检测与输出脱敏</small></div>
          <el-icon class="path-arrow" aria-hidden="true"><ArrowRight /></el-icon>
          <div><el-icon><Connection /></el-icon><span>审计溯源</span><small>四节点指纹对账</small></div>
        </div>
      </section>
      <section class="login-card card">
        <h2>登录工作台</h2>
        <p class="login-subtitle">使用您的账号进入安全管控平台</p>
        <div class="demo-accounts" aria-label="选择演示账号">
          <button v-for="account in demoAccounts" :key="account.username" type="button" class="account-button"
            :class="{ selected: form.username === account.username }" :aria-pressed="form.username === account.username"
            :disabled="loading" @click="chooseAccount(account)">
            <span>{{ account.dataLevel }} · {{ account.role }}</span><small>{{ account.username }}</small>
          </button>
        </div>
        <p class="account-hint">选择演示账号填入登录信息，密码与账号同名。</p>
        <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" class="login-error" />
        <el-form ref="formRef" :model="form" :rules="rules" label-position="top" :disabled="loading" @submit.prevent="handleLogin">
          <el-form-item label="用户名" prop="username">
            <el-input v-model="form.username" name="username" autocomplete="username" placeholder="请输入用户名" :prefix-icon="User" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="form.password" name="password" type="password" autocomplete="current-password" placeholder="请输入密码" show-password :prefix-icon="Key" />
          </el-form-item>
          <el-button type="primary" native-type="submit" class="submit" :loading="loading">{{ loading ? '正在登录' : '登录' }}<el-icon v-if="!loading" class="action-arrow"><ArrowRight /></el-icon></el-button>
        </el-form>
        <p v-if="isDemo" class="hint">演示环境：每个账号使用对应的角色与密级。</p>
        <p v-else class="hint">请使用已分配的账号与密码登录。</p>
      </section>
    </main>
    <footer class="login-footer">链安智御 · 全链路安全管控平台</footer>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Key } from '@element-plus/icons-vue'
import { useUserStore } from '@/store/user'
import { useTheme } from '@/composables/useTheme'
import { useMock } from '@/mock'
import BrandLogo from '@/components/BrandLogo.vue'
import { demoAccounts } from '@/config/demoAccounts'
import { getDefaultRoute, hasPermission } from '@/config/permissions'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const { isDark, toggleTheme } = useTheme()
const isDemo = useMock()
const formRef = ref()
const loading = ref(false)
const error = ref('')
const form = reactive({ username: 'reviewer', password: '' })
const rules = {
  username: [{ required: true, whitespace: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }]
}

function chooseAccount(account) {
  form.username = account.username
  form.password = account.password
  error.value = ''
  formRef.value?.clearValidate()
}

async function handleLogin() {
  if (loading.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  error.value = ''
  try {
    await userStore.login({ username: form.username.trim(), password: form.password })
    ElMessage.success('登录成功')
    const destination = typeof route.query.redirect === 'string' ? route.query.redirect : ''
    const target = destination.startsWith('/') && !destination.startsWith('//') ? router.resolve(destination) : null
    const permitted = target?.matched.some(record => record.meta.requiresAuth)
      && (!target.meta.permission || hasPermission(userStore.userInfo, target.meta.permission))
    await router.replace(permitted ? target.fullPath : getDefaultRoute(userStore.userInfo))
  } catch (reason) {
    error.value = reason.message || '登录失败，请检查账号与密码后重试'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-wrap { min-height: 100dvh; display: flex; flex-direction: column; padding: 42px 6vw 24px; background: var(--surface-sub); }
.login-brand { display: flex; flex-wrap: wrap; align-items: center; gap: 16px; }
.theme-switch { position: absolute; right: 6vw; top: 40px; width: 40px; padding: 0; color: var(--text-sub); font-size: 18px; }
.login-main { flex: 1; display: grid; grid-template-columns: minmax(0, 1fr) 400px; align-items: center; gap: 8vw; width: 100%; max-width: 1220px; margin: 60px auto; }
.login-intro h1 { margin: 0; font-size: 32px; font-weight: 650; line-height: 1.65; letter-spacing: -0.025em; }
.login-intro > p { color: var(--text-sub); font-size: 15px; margin: 20px 0 42px; }
.security-path { display: flex; gap: 18px; align-items: center; padding-top: 28px; border-top: 1px solid var(--border); }
.security-path > div { display: grid; gap: 6px; }
.security-path > div .el-icon { color: var(--brand); font-size: 22px; margin-bottom: 8px; }
.security-path span { font-size: 14px; font-weight: 600; }
.security-path small { color: var(--text-sub); font-size: 11px; }
.path-arrow { color: var(--text-sub); flex-shrink: 0; }
.login-card { padding: 36px; }
.login-card h2 { margin: 0 0 8px; font-size: 24px; font-weight: 600; }
.login-subtitle { color: var(--text-sub); font-size: 13px; margin: 0 0 28px; }
.login-error { margin-bottom: 20px; }
.demo-accounts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
.account-button { display: grid; gap: 4px; padding: 10px 12px; border: 1px solid var(--border); border-radius: 8px; text-align: left; color: var(--text-main); background: var(--surface-sub); cursor: pointer; }
.account-button span { font-size: 12px; font-weight: 500; }
.account-button small { font-size: 11px; color: var(--text-sub); }
.account-button:hover, .account-button.selected { border-color: var(--brand); background: var(--brand-light); }
.account-button:disabled { cursor: wait; opacity: 0.6; }
.account-hint { margin: 12px 0 22px; font-size: 12px; color: var(--text-sub); }
.submit { width: 100%; min-height: 46px; margin-top: 8px; }
.submit .el-icon { margin-left: 8px; }
.hint { margin: 20px 0 0; font-size: 12px; color: var(--text-sub); text-align: center; }
.login-footer { color: var(--text-sub); font-size: 11px; text-align: center; }
@media (max-width: 1100px) { .login-main { gap: 40px; grid-template-columns: minmax(0, 1fr) 360px; } .login-intro h1 { font-size: 25px; } .security-path { gap: 10px; } .security-path small { max-width: 8em; } }
@media (max-width: 800px) { .login-main { grid-template-columns: minmax(0, 1fr); max-width: 480px; margin: 40px auto; gap: 28px; } .security-path { display: none; } .login-intro > p { margin-bottom: 0; font-size: 13px; } .login-intro h1 { font-size: 23px; } .desktop-break { display: none; } }
@media (max-width: 480px) { .login-wrap { padding: 24px 20px 20px; } .login-brand { gap: 8px; } .theme-switch { top: 24px; right: 20px; } .login-card { padding: 28px 24px; } }
</style>
