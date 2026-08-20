// 应用入口：加载 Element Plus 与游戏工作台全局样式。
import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import './style.css'
import App from './App.vue'

createApp(App).use(ElementPlus).mount('#app')
