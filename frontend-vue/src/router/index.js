// src/router/index.js
import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '../store/user'
import Login from '../views/Login.vue' // 稍后创建
import Home from '../views/Home.vue'   // 把原来的 App.vue 内容移到这里

const routes = [
    { path: '/login', component: Login, meta: { requiresAuth: false } },
    { path: '/', component: Home, meta: { requiresAuth: true } },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

// 🔒 全局路由守卫
router.beforeEach((to, from, next) => {
    const userStore = useUserStore()
    // 如果页面需要登录，且用户没有 token
    if (to.meta.requiresAuth && !userStore.token) {
        next('/login')
    } else {
        next()
    }
})

export default router