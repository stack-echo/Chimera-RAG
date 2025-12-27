<template>
  <div class="login-container">
    <div class="login-box">
      <h2 class="title">🦄 Chimera-RAG</h2>
      <a-tabs default-active-key="1">
        <a-tab-pane key="1" title="登录">
          <a-form :model="loginForm" @submit="handleLogin">
            <a-form-item field="username" label="用户名">
              <a-input v-model="loginForm.username" placeholder="请输入用户名" />
            </a-form-item>
            <a-form-item field="password" label="密码">
              <a-input-password v-model="loginForm.password" placeholder="请输入密码" />
            </a-form-item>
            <a-button type="primary" html-type="submit" long :loading="loading">立即登录</a-button>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="2" title="注册">
          <a-form :model="regForm" @submit="handleRegister">
            <a-form-item field="username" label="用户名">
              <a-input v-model="regForm.username" />
            </a-form-item>
            <a-form-item field="email" label="邮箱">
              <a-input v-model="regForm.email" />
            </a-form-item>
            <a-form-item field="password" label="密码">
              <a-input-password v-model="regForm.password" />
            </a-form-item>
            <a-button type="outline" html-type="submit" long :loading="loading">注册账号</a-button>
          </a-form>
        </a-tab-pane>
      </a-tabs>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import request from '../api/request' // 导入我们封装的 axios
import { useUserStore } from '../store/user'
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'

const userStore = useUserStore()
const router = useRouter()
const loading = ref(false)

// 表单数据
const loginForm = reactive({ username: '', password: '' })
const regForm = reactive({ username: '', password: '', email: '' })

// 登录逻辑
const handleLogin = async () => {
  loading.value = true
  try {
    const res = await request.post('/auth/login', loginForm)
    // res 已经是 response.data 了 (因为拦截器处理过)
    userStore.setLoginState(res.token, { username: res.username, id: res.user_id })
    Message.success('登录成功')
    router.push('/') // 跳转首页
  } catch (e) {
    // 错误在拦截器里处理了，这里不需要写 Message
  } finally {
    loading.value = false
  }
}

// 注册逻辑
const handleRegister = async () => {
  loading.value = true
  try {
    await request.post('/auth/register', regForm)
    Message.success('注册成功，请登录')
  } catch (e) {
    // error handled
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}
.login-box {
  width: 400px;
  padding: 40px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0,0,0,0.1);
}
.title {
  text-align: center;
  margin-bottom: 30px;
  color: #333;
}
</style>