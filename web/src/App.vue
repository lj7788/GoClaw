<template>
  <div id="app">
    <router-view />
    <Teleport to="body">
      <div class="fixed top-4 right-4 z-50 space-y-2">
        <transition-group name="toast">
          <div v-for="toast in toasts" :key="toast.id" :class="[
            'min-w-[300px] max-w-md p-4 rounded-lg shadow-lg border backdrop-blur-sm flex items-start space-x-3',
            toast.type === 'success' ? 'bg-green-900/80 border-green-700' :
              toast.type === 'error' ? 'bg-red-900/80 border-red-700' :
                toast.type === 'warning' ? 'bg-yellow-900/80 border-yellow-700' :
                  'bg-blue-900/80 border-blue-700'
          ]">
            <span class="text-xl flex-shrink-0 mt-0.5">
              {{ toast.type === 'success' ? '✅' :
                toast.type === 'error' ? '❌' :
                  toast.type === 'warning' ? '⚠️' : 'ℹ️' }}
            </span>
            <div class="flex-1">
              <p :class="[
                'text-sm font-medium',
                toast.type === 'success' ? 'text-green-100' :
                  toast.type === 'error' ? 'text-red-100' :
                    toast.type === 'warning' ? 'text-yellow-100' :
                      'text-blue-100'
              ]">
                {{ toast.message }}
              </p>
            </div>
            <button @click="removeToast(toast.id)"
              class="flex-shrink-0 text-gray-400 hover:text-white transition-colors">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </transition-group>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, inject } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from './hooks/useAuth'
import { useStore } from './store'
import { getStatus } from './lib/api'

const { logout } = useAuth()

const router = useRouter()
const store = useStore()
const toasts = inject('toasts') as any

const removeToast = (id: number) => {
  const index = toasts.value.findIndex((t: any) => t.id === id)
  if (index > -1) {
    toasts.value.splice(index, 1)
  }
}

const handleUnauthorized = () => {
  logout()
  if (store.status?.loginMode === 'wechat') {
    router.push('/login')
  } else if (store.status?.loginMode === 'paired') {
    router.push('/paired')
  }
}

onMounted(async () => {
  window.addEventListener('zeroclaw-unauthorized', handleUnauthorized)

  try {
    const status = await getStatus()
    status.loginMode = status.paired ? 'paired' : status.wechatlogin ? 'wechat' : 'none'
    store.setStatus(status)

    const whiteList = ['/login', '/login/success', '/login/pending', '/paired', '/admin/login']
    const currentPath = router.currentRoute.value.path

    if (status.loginMode === 'wechat' && !store.isLogin && !whiteList.includes(currentPath)) {
      router.push('/login')
    } else if (status.loginMode === 'paired' && !store.isLogin && !whiteList.includes(currentPath)) {
      router.push('/paired')
    }
  } catch (err) {
    console.error('Failed to get status:', err)
  }
})

onUnmounted(() => {
  window.removeEventListener('zeroclaw-unauthorized', handleUnauthorized)
})
</script>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(100%);
}

.toast-move {
  transition: transform 0.3s ease;
}
</style>
