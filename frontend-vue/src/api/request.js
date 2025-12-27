// src/api/request.js
import axios from 'axios'
import { useUserStore } from '../store/user'
import { Message } from '@arco-design/web-vue'

// 创建 axios 实例
const request = axios.create({
    baseURL: 'http://localhost:8080/api/v1', // 后端地址
    timeout: 10000,
})

// 🟢 请求拦截器：每次请求自动带 Token
request.interceptors.request.use(config => {
    const userStore = useUserStore()
    if (userStore.token) {
        config.headers.Authorization = `Bearer ${userStore.token}`
    }
    return config
}, error => {
    return Promise.reject(error)
})

// 🔴 响应拦截器：统一处理错误
request.interceptors.response.use(response => {
    return response.data
}, error => {
    // 如果后端返回 401 Unauthorized
    if (error.response && error.response.status === 401) {
        const userStore = useUserStore()
        userStore.logout()
        Message.error('登录过期，请重新登录')
        // 这里可以触发路由跳转，或者 reload
        window.location.reload()
    } else {
        Message.error(error.response?.data?.error || '网络请求失败')
    }
    return Promise.reject(error)
})

export default request