import { createApp } from 'vue'
import { createPinia } from 'pinia'
import {
  ElAlert, ElButton, ElConfigProvider, ElDrawer, ElEmpty, ElForm, ElFormItem,
  ElIcon, ElInput, ElInputNumber, ElOption, ElSelect, ElSkeleton, ElSkeletonItem, ElSwitch,
  ElTable, ElTableColumn, ElTabPane, ElTabs, ElTag
} from 'element-plus'
import 'element-plus/es/components/alert/style/css'
import 'element-plus/es/components/button/style/css'
import 'element-plus/es/components/drawer/style/css'
import 'element-plus/es/components/empty/style/css'
import 'element-plus/es/components/form/style/css'
import 'element-plus/es/components/icon/style/css'
import 'element-plus/es/components/input/style/css'
import 'element-plus/es/components/input-number/style/css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/select/style/css'
import 'element-plus/es/components/skeleton/style/css'
import 'element-plus/es/components/switch/style/css'
import 'element-plus/es/components/table/style/css'
import 'element-plus/es/components/tabs/style/css'
import 'element-plus/es/components/tag/style/css'
import {
  Lock, User, DataLine, Warning, TrendCharts, DataAnalysis, Aim, Connection,
  CircleClose, CircleCheck, Remove, ArrowRight, Refresh, Menu, SwitchButton,
  Sunny, Moon, Search, CopyDocument, Setting
} from '@element-plus/icons-vue'

import App from './App.vue'
import router from './router'
import { buttonFeedback } from './directives/buttonFeedback'
import './composables/useTheme'
import './assets/styles/index.css'

const app = createApp(App)
app.directive('feedback', buttonFeedback)
// Register only the components and icons used by this app; keep the existing design system.
const components = [
  ElAlert, ElButton, ElConfigProvider, ElDrawer, ElEmpty, ElForm, ElFormItem,
  ElIcon, ElInput, ElInputNumber, ElOption, ElSelect, ElSkeleton, ElSkeletonItem, ElSwitch,
  ElTable, ElTableColumn, ElTabPane, ElTabs, ElTag
]
for (const component of components) app.component(component.name, component)
const icons = { Lock, User, DataLine, Warning, TrendCharts, DataAnalysis, Aim, Connection, CircleClose, CircleCheck, Remove, ArrowRight, Refresh, Menu, SwitchButton, Sunny, Moon, Search, CopyDocument, Setting }
for (const [name, component] of Object.entries(icons)) app.component(name, component)

app.use(createPinia())
app.use(router)
app.mount('#app')
