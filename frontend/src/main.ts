// 应用入口：按需注册 Element Plus 组件，避免入口加载完整组件库。
import { createApp } from 'vue'
import { ElButton } from 'element-plus/es/components/button/index'
import { ElDialog } from 'element-plus/es/components/dialog/index'
import { ElEmpty } from 'element-plus/es/components/empty/index'
import { ElIcon } from 'element-plus/es/components/icon/index'
import { ElInput } from 'element-plus/es/components/input/index'
import { ElInputNumber } from 'element-plus/es/components/input-number/index'
import { ElLoadingDirective } from 'element-plus/es/components/loading/index'
import { ElMessage } from 'element-plus/es/components/message/index'
import { ElMessageBox } from 'element-plus/es/components/message-box/index'
import { ElOption, ElSelect } from 'element-plus/es/components/select/index'
import { ElProgress } from 'element-plus/es/components/progress/index'
import { ElSkeleton } from 'element-plus/es/components/skeleton/index'
import { ElSwitch } from 'element-plus/es/components/switch/index'
import { ElTabPane, ElTabs } from 'element-plus/es/components/tabs/index'
import 'element-plus/es/components/button/style/css'
import 'element-plus/es/components/dialog/style/css'
import 'element-plus/es/components/empty/style/css'
import 'element-plus/es/components/icon/style/css'
import 'element-plus/es/components/input/style/css'
import 'element-plus/es/components/input-number/style/css'
import 'element-plus/es/components/loading/style/css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/message-box/style/css'
import 'element-plus/es/components/option/style/css'
import 'element-plus/es/components/progress/style/css'
import 'element-plus/es/components/select/style/css'
import 'element-plus/es/components/skeleton/style/css'
import 'element-plus/es/components/switch/style/css'
import 'element-plus/es/components/tab-pane/style/css'
import 'element-plus/es/components/tabs/style/css'
import './style.css'
import App from './App.vue'

const app = createApp(App)

app
  .component('ElButton', ElButton)
  .component('ElDialog', ElDialog)
  .component('ElEmpty', ElEmpty)
  .component('ElIcon', ElIcon)
  .component('ElInput', ElInput)
  .component('ElInputNumber', ElInputNumber)
  .component('ElOption', ElOption)
  .component('ElProgress', ElProgress)
  .component('ElSelect', ElSelect)
  .component('ElSkeleton', ElSkeleton)
  .component('ElSwitch', ElSwitch)
  .component('ElTabPane', ElTabPane)
  .component('ElTabs', ElTabs)
  .use(ElMessage)
  .use(ElMessageBox)
  .directive('loading', ElLoadingDirective)
  .mount('#app')
