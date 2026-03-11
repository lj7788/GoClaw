<template>
  <div class="min-h-screen bg-gray-950 text-white">
    <div
      :class="[
        'fixed top-0 left-0 h-screen transition-all duration-300 overflow-hidden z-20',
        sidebarCollapsed ? 'w-16' : 'w-60'
      ]"
    >
      <Sidebar :collapsed="sidebarCollapsed" @toggle="sidebarCollapsed = !sidebarCollapsed" />
    </div>

    <div
      :class="[
        'flex flex-col min-h-screen transition-all duration-300',
        sidebarCollapsed ? 'ml-16' : 'ml-60'
      ]"
    >
      <Header />

      <main class="flex-1 overflow-y-auto">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Sidebar from './Sidebar.vue'
import Header from './Header.vue'
import { useStore } from '@/store'
import { useRouter } from 'vue-router'
import { onMounted } from 'vue'
const router = useRouter()

const sidebarCollapsed = ref(false)

onMounted(() => {
  const store = useStore()
  if (store.status?.loginMode === 'wechat') {
    if (!store.isLogin) {
      router.push('/login')
    }

  }
  if (store.status?.loginMode === 'paired') {
    if (!store.isLogin) {
      router.push('/paired')
    }
  }
})

</script>
