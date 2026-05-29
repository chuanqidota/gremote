import { createApp } from 'vue'
import App from './App.vue'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import 'element-plus/es/components/message/style/css'
import router from './router'

const app = createApp(App)
app.provide('ELEMENT_LOCALE', zhCn)
app.use(router)
app.mount('#app')
