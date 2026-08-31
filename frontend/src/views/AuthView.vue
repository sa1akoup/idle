<!-- 账号入口：登录或注册后进入行动终端。 -->
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus/es/components/message/index'
import api, { getApiError } from '../api'
import type { User } from '../types'

const emit = defineEmits<{
  authenticated: [user: User]
}>()

const props = defineProps<{
  error?: string
}>()

const mode = ref<'login' | 'register'>('login')
const username = ref('')
const password = ref('')
const loading = ref(false)

async function submit() {
  loading.value = true
  try {
    const endpoint = mode.value === 'login' ? '/auth/login' : '/auth/register'
    const { data } = await api.post<User>(endpoint, { username: username.value, password: password.value })
    emit('authenticated', data)
  } catch (error) {
    ElMessage.error(getApiError(error, mode.value === 'login' ? '登录失败' : '注册失败'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="auth-shell">
    <section class="auth-panel">
      <div class="auth-kicker">GREY ZONE / ONLINE</div>
      <h1>行动终端</h1>
      <p class="auth-subtitle">登录账号，继续你的封锁区行动记录。</p>
      <p v-if="props.error" class="auth-error">{{ props.error }}</p>
      <div class="auth-switch" role="tablist" aria-label="账号操作">
        <button type="button" :class="{ active: mode === 'login' }" @click="mode = 'login'">登录</button>
        <button type="button" :class="{ active: mode === 'register' }" @click="mode = 'register'">注册</button>
      </div>
      <form @submit.prevent="submit">
        <label>
          <span>用户名</span>
          <input v-model.trim="username" autocomplete="username" minlength="3" maxlength="32" required />
        </label>
        <label>
          <span>密码</span>
          <input v-model="password" type="password" :autocomplete="mode === 'login' ? 'current-password' : 'new-password'" minlength="8" maxlength="72" required />
        </label>
        <button class="auth-submit" type="submit" :disabled="loading">
          {{ loading ? '处理中...' : mode === 'login' ? '进入终端' : '创建账号' }}
        </button>
      </form>
    </section>
  </main>
</template>
