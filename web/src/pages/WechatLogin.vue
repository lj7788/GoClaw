<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center p-4">
    <div class="max-w-md w-full space-y-8">
      <div class="text-center">
        <h2 class="mt-6 text-3xl font-extrabold text-gray-900">
          微信扫码登录
        </h2>
        <p class="mt-2 text-sm text-gray-600">
          使用微信扫码登录您的账号
        </p>
      </div>

      <div v-if="loading" class="bg-white p-8 rounded-lg shadow-md">
        <div class="flex justify-center">
          <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
        </div>
        <p class="mt-4 text-center text-gray-600">正在生成登录二维码...</p>
      </div>

      <div v-else-if="loginUrl" class="bg-white p-8 rounded-lg shadow-md">
        <div class="flex justify-center">
          <div class="w-64 h-64 bg-gray-100 flex items-center justify-center">
            <img :src="qrcodeUrl" alt="微信登录二维码" class="max-w-full max-h-full" />
          </div>
        </div>
        <p class="mt-4 text-center text-gray-600">请使用微信扫码登录</p>
        <button @click="refreshQrCode" class="mt-4 w-full py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
          刷新二维码
        </button>
      </div>

      <div v-else class="bg-white p-8 rounded-lg shadow-md">
        <button @click="generateLoginUrl" class="w-full py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
          开始微信登录
        </button>
      </div>

      <div v-if="error" class="bg-red-50 border border-red-200 rounded-md p-4">
        <div class="flex">
          <div class="flex-shrink-0">
            <svg class="h-5 w-5 text-red-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
            </svg>
          </div>
          <div class="ml-3">
            <h3 class="text-sm font-medium text-red-800">{{ error }}</h3>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { getWechatLoginURL } from '../lib/api'
import { setToken, setUser } from '../lib/auth'
import { useRouter } from 'vue-router'

const router = useRouter()
const loading = ref(false)
const loginUrl = ref<string>('')
const qrcodeUrl = ref<string>('')
const error = ref<string>('')
const pollingInterval = ref<number | null>(null)

const generateLoginUrl = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await getWechatLoginURL()
    loginUrl.value = response.login_url
    // 生成二维码图片（实际项目中可以使用qrcode库）
    qrcodeUrl.value = `https://api.qrserver.com/v1/create-qr-code/?size=256x256&data=${encodeURIComponent(response.login_url)}`
  } catch (err: any) {
    error.value = err.message || '生成登录链接失败'
  } finally {
    loading.value = false
  }
}

const refreshQrCode = () => {
  generateLoginUrl()
}

// 检查URL参数中是否有token（微信回调后）
const checkCallback = () => {
  const urlParams = new URLSearchParams(window.location.search)
  const token = urlParams.get('token')
  if (token) {
    setToken(token)
    router.push('/')
    return true
  }
  return false
}

// 建立WebSocket连接
let ws: WebSocket | null = null

const connectWebSocket = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/api/ws/chat`
  
  ws = new WebSocket(wsUrl)
  
  ws.onopen = function() {
    console.log('WebSocket连接成功')
  }
  
  ws.onmessage = function(event) {
    try {
      const data = JSON.parse(event.data)
      if (data.type === 'login.success') {
        setToken(data.token)
        setUser(data.user)
        router.push('/')
        stopPolling()
        if (ws) {
          ws.close()
        }
      }
    } catch (err) {
      console.error('WebSocket消息解析失败:', err)
    }
  }
  
  ws.onerror = function(error) {
    console.error('WebSocket错误:', error)
  }
  
  ws.onclose = function() {
    console.log('WebSocket连接关闭')
  }
}

// 轮询检测登录状态
const startPolling = () => {
  // 建立WebSocket连接
  connectWebSocket()
  
  // 每2秒检查一次URL参数（作为备用方案）
  pollingInterval.value = window.setInterval(() => {
    // 检查URL参数
    if (checkCallback()) {
      stopPolling()
      return
    }
  }, 2000)
}

const stopPolling = () => {
  if (pollingInterval.value) {
    window.clearInterval(pollingInterval.value)
    pollingInterval.value = null
  }
}

onMounted(() => {
  // 初始检查一次
  if (!checkCallback()) {
    // 如果没有token，开始轮询
    startPolling()
  }
  generateLoginUrl()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
/* 自定义样式 */
</style>