import { createApp } from 'vue'
import ArcoVue from '@arco-design/web-vue';
import '@arco-design/web-vue/dist/arco.css'; // 引入样式
import App from './App.vue'

const app = createApp(App);
app.use(ArcoVue);
app.mount('#app');