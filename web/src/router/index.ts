import { createRouter, createWebHistory } from 'vue-router'
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
      { path: 'doctor', component: Doctor }
    ]
  }
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})
