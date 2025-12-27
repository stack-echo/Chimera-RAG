import { createApp } from 'vue'
import ArcoVue from '@arco-design/web-vue';
import '@arco-design/web-vue/dist/arco.css'; // 引入样式
import App from './App.vue'
import { createPinia } from 'pinia'
import router from './router'

const app = createApp(App);
app.use(createPinia())
app.use(ArcoVue);
app.use(router)
app.mount('#app');