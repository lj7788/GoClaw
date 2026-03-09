<template>
  <header class="h-14 bg-gray-800 border-b border-gray-700 flex items-center justify-between px-6">
    <h1 class="text-lg font-semibold text-white">{{ t(pageTitleKey) }}</h1>

    <div class="flex items-center gap-4" >
      <button type="button" @click="toggleLanguage"
        class="px-3 py-1 rounded-md text-sm font-medium border border-gray-600 text-gray-300 hover:bg-gray-700 hover:text-white transition-colors">
        {{ localeDisplay }}
      </button>
      <template v-if="loginMode !== 'none'">
      <template v-if="isAuthenticated ">
        <div class="flex items-center gap-3">
          <div class="flex items-center gap-2">
            <img v-if="user?.avatar" :src="user.avatar" :alt="isAdmin ? user.username : user.nickname"
              class="w-8 h-8 rounded-full object-cover" />
            <span class="text-sm text-gray-300">{{ isAdmin ? (user?.username || '管理员') : (user?.nickname || '用户') }}</span>
          </div>
          <router-link to="/user" v-if="isChat &&!isAdmin"
            class="px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors">
            用户中心
          </router-link>
          <router-link to="/admin" v-if="isChat && isAdmin"
            class="px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors">
            管理后台
          </router-link>
          <button type="button" @click="showLogoutConfirm"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors">
            <LogOut class="h-4 w-4" />
            <span>{{ t('auth.logout') }}</span>
          </button>
        </div>
      </template>
      <template v-else>
        <router-link to="/login"
          class="px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors">
          微信登录
        </router-link>
        <router-link to="/admin/login"
          class="px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors">
          管理员登录
        </router-link>
      </template>
      </template>
    </div>
  </header>

  <!-- 退出登录确认对话框 -->
  <Teleport to="body">
    <div v-if="showConfirm" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/60" @click="cancelLogout"></div>

      <div class="relative bg-gray-800 rounded-xl p-6 w-full max-w-sm shadow-2xl border border-gray-700">
        <div class="text-center">
          <div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-red-100 mb-4">
            <LogOut class="h-6 w-6 text-red-600" />
          </div>
          <h3 class="text-lg font-semibold text-white mb-2">确认退出</h3>
          <p class="text-sm text-gray-400 mb-6">确定要退出登录吗？</p>
          <div class="flex gap-3">
            <button @click="cancelLogout"
              class="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg text-sm font-medium transition-colors">
              取消
            </button>
            <button @click="handleLogout"
              class="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg text-sm font-medium transition-colors">
              退出登录
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogOut } from 'lucide-vue-next'
import { useAuth } from '../../hooks/useAuth'
import { useStore } from '@/store'
import { useI18n, setLocale, type Locale } from '../../lib/i18n'
const store = useStore()
const route = useRoute()
const router = useRouter()
const { logout: doLogout, isAuthenticated, user } = useAuth()
const { t, locale, initLocale } = useI18n()

const showConfirm = ref(false)

const showLogoutConfirm = () => {
  showConfirm.value = true
}

const cancelLogout = () => {
  showConfirm.value = false
}

const routeTitleKeys: Record<string, string> = {
  '/': 'nav.dashboard',
  '/agent': 'nav.agent',
  '/tools': 'nav.tools',
  '/cron': 'nav.cron',
  '/integrations': 'nav.integrations',
  '/memory': 'nav.memory',
  '/config': 'nav.config',
  '/cost': 'nav.cost',
  '/logs': 'nav.logs',
  '/doctor': 'nav.doctor'
}

const pageTitleKey = computed(() => {
  return routeTitleKeys[route.path] ?? 'nav.dashboard'
})
const loginMode=computed(()=>{
  return store.status?.loginMode
})

const isChat=computed(()=>{
  return store.status?.loginMode === 'wechat'
})
const isAdmin=computed(()=>{
  return store.isAdmin
})

const localeDisplay = computed(() => {
  const loc = locale.value
  if (loc === 'en') return 'EN'
  if (loc === 'tr') return 'TR'
  return '中文'
})

const toggleLanguage = () => {
  const localeCycle: Locale[] = ['en', 'tr', 'zh-CN']
  const currentIndex = localeCycle.indexOf(locale.value as Locale)
  const nextIndex = currentIndex === -1 ? 0 : (currentIndex + 1) % localeCycle.length
  setLocale(localeCycle[nextIndex] ?? 'en')
}

const handleLogout = () => {
  doLogout()
  if (store.status?.loginMode === 'wechat') {
    router.push('/login')

  }
  if (store.status?.loginMode === 'paired') {
    router.push('/paired')
  }
}

onMounted(() => {
  initLocale()
})
</script>
