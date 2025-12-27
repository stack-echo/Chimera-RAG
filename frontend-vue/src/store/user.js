// src/store/user.js
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUserStore = defineStore('user', () => {
    // 从 localStorage 初始化，防止刷新丢失
    const token = ref(localStorage.getItem('token') || '')
    const userInfo = ref(JSON.parse(localStorage.getItem('userInfo') || '{}'))

    // 登录动作
    function setLoginState(newToken, newUser) {
        token.value = newToken
        userInfo.value = newUser
        // 持久化保存
        localStorage.setItem('token', newToken)
        localStorage.setItem('userInfo', JSON.stringify(newUser))
    }

    // 登出动作
    function logout() {
        token.value = ''
        userInfo.value = {}
        localStorage.removeItem('token')
        localStorage.removeItem('userInfo')
    }

    return { token, userInfo, setLoginState, logout }
})