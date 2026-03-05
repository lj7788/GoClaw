<template>
  <header class="h-14 bg-gray-800 border-b border-gray-700 flex items-center justify-between px-6">
    <h1 class="text-lg font-semibold text-white">{{ t(pageTitleKey) }}</h1>

    <div class="flex items-center gap-4">
      <button
        type="button"
        @click="toggleLanguage"
        class="px-3 py-1 rounded-md text-sm font-medium border border-gray-600 text-gray-300 hover:bg-gray-700 hover:text-white transition-colors"
      >
        {{ localeDisplay }}
      </button>

      <template v-if="isAuthenticated">
        <div class="flex items-center gap-3">
          <div class="flex items-center gap-2">
            <img
              v-if="user?.avatar"
              :src="user.avatar"
              :alt="user.nickname"
              class="w-8 h-8 rounded-full object-cover"
            />
            <span class="text-sm text-gray-300">{{ user?.nickname || '用户' }}</span>
          </div>
          <router-link
            to="/user"
            class="px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors"
          >
            用户中心
          </router-link>
          <router-link
            to="/admin"
            class="px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors"
          >
            管理后台
          </router-link>
          <button
            type="button"
            @click="handleLogout"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors"
          >
            <LogOut class="h-4 w-4" />
            <span>{{ t('auth.logout') }}</span>
          </button>
        </div>
      </template>
      <template v-else>
        <router-link
          to="/login"
          class="px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors"
        >
          微信登录
        </router-link>
        <router-link
          to="/admin/login"
          class="px-3 py-1.5 rounded-md text-sm text-gray-300 hover:bg-gray-700 hover:text-white transition-colors"
        >
          管理员登录
        </router-link>
      </template>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { LogOut } from 'lucide-vue-next'
import { useAuth } from '../../hooks/useAuth'
import { useI18n, setLocale, type Locale } from '../../lib/i18n'

const route = useRoute()
const { logout: doLogout, isAuthenticated, user } = useAuth()
const { t, locale, initLocale } = useI18n()

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
}

onMounted(() => {
  initLocale()
})
</script>
