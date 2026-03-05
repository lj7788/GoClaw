import { createRouter, createWebHashHistory } from 'vue-router'
import Layout from '../components/layout/Layout.vue'
import Dashboard from '../pages/Dashboard.vue'
import AgentChat from '../pages/AgentChat.vue'
import Tools from '../pages/Tools.vue'
import Cron from '../pages/Cron.vue'
import Integrations from '../pages/Integrations.vue'
import Memory from '../pages/Memory.vue'
import Config from '../pages/Config.vue'
import Cost from '../pages/Cost.vue'
import Logs from '../pages/Logs.vue'
import Doctor from '../pages/Doctor.vue'
import WechatLogin from '../pages/WechatLogin.vue'
import UserCenter from '../pages/UserCenter.vue'
import AdminLogin from '../pages/AdminLogin.vue'
import Admin from '../pages/Admin.vue'
import LoginSuccess from '../pages/LoginSuccess.vue'
import LoginPending from '../pages/LoginPending.vue'

const routes = [
  {
    path: '/',
    component: Layout,
    children: [
      { path: '', component: Dashboard },
      { path: 'agent', component: AgentChat },
      { path: 'tools', component: Tools },
      { path: 'cron', component: Cron },
      { path: 'integrations', component: Integrations },
      { path: 'memory', component: Memory },
      { path: 'config', component: Config },
      { path: 'cost', component: Cost },
      { path: 'logs', component: Logs },
      { path: 'doctor', component: Doctor },
      { path: 'user', component: UserCenter },
      { path: 'admin', component: Admin }
    ]
  },
  {
    path: '/login',
    component: WechatLogin
  },
  {
    path: '/login/success',
    component: LoginSuccess
  },
  {
    path: '/login/pending',
    component: LoginPending
  },
  {
    path: '/admin/login',
    component: AdminLogin
  }
]

export const router = createRouter({
  history: createWebHashHistory(),
  routes
})
