<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-white mb-6">{{ t('channels.title') }}</h1>

    <!-- DingTalk Section -->
    <div class="bg-gray-800 rounded-lg p-6 mb-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-semibold text-white flex items-center gap-2">
          <svg class="w-6 h-6 text-blue-400" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
          </svg>
          {{ t('channels.dingtalk') }}
        </h2>
        <span :class="[
          'px-3 py-1 rounded-full text-sm font-medium',
          dingtalkStatus?.connected ? 'bg-green-900 text-green-300' : 'bg-gray-700 text-gray-300'
        ]">
          {{ dingtalkStatus?.connected ? t('channels.connected') : t('channels.disconnected') }}
        </span>
      </div>

      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('channels.client_id') }}</label>
          <input
            v-model="dingtalkConfig.client_id"
            type="text"
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Enter Client ID"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">{{ t('channels.client_secret') }}</label>
          <input
            v-model="dingtalkConfig.client_secret"
            type="password"
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="Enter Client Secret"
          />
        </div>

        <div class="flex gap-3">
          <button
            @click="saveDingTalkConfigAndConnect"
            :disabled="dingtalkLoading || !dingtalkConfig.client_id || !dingtalkConfig.client_secret"
            class="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
          >
            {{ dingtalkLoading ? t('common.loading') : t('channels.connect') }}
          </button>

          <button
            v-if="dingtalkStatus?.connected"
            @click="disconnectDingTalkChannel"
            :disabled="dingtalkLoading"
            class="px-4 py-2 bg-red-600 hover:bg-red-700 disabled:bg-gray-600 text-white font-medium rounded-lg transition-colors"
          >
            {{ t('channels.disconnect') }}
          </button>
        </div>

        <p v-if="dingtalkError" class="text-red-400 text-sm">{{ dingtalkError }}</p>
        <p v-if="dingtalkSuccess" class="text-green-400 text-sm">{{ dingtalkSuccess }}</p>
      </div>
    </div>

    <!-- WeChat Section -->
    <div class="bg-gray-800 rounded-lg p-6">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-semibold text-white flex items-center gap-2">
          <svg class="w-6 h-6 text-green-400" fill="currentColor" viewBox="0 0 24 24">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
          </svg>
          {{ t('channels.weixin') }}
        </h2>
        <span :class="[
          'px-3 py-1 rounded-full text-sm font-medium',
          weixinStatus?.status ? 'bg-green-900 text-green-300' : 'bg-gray-700 text-gray-300'
        ]">
          {{ weixinStatus?.status ? t('channels.connected') : t('channels.disconnected') }}
        </span>
      </div>

      <div class="space-y-4">
        <!-- QR Code Display -->
        <div v-if="showQRCode" class="flex flex-col items-center">
          <div class="w-64 h-64 bg-white rounded-lg flex items-center justify-center p-4">
            <img v-if="qrcodeUrl" :src="qrcodeUrl" alt="QR Code" class="max-w-full max-h-full" />
            <div v-else class="text-gray-500">{{ t('common.loading') }}</div>
          </div>
          <p class="mt-4 text-gray-300 text-center">{{ t('channels.scan_qrcode') }}</p>
          <p class="text-sm text-gray-400">{{ weixinLoginStatus }}</p>
        </div>

        <!-- Action Buttons -->
        <div class="flex gap-3">
          <button
            v-if="!weixinStatus?.status && !showQRCode"
            @click="generateWeixinQRCode"
            :disabled="weixinLoading"
            class="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-600 text-white font-medium rounded-lg transition-colors"
          >
            {{ weixinLoading ? t('common.loading') : t('channels.generate_qrcode') }}
          </button>

          <button
            v-if="showQRCode"
            @click="cancelQRCode"
            class="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white font-medium rounded-lg transition-colors"
          >
            {{ t('common.cancel') }}
          </button>

          <button
            v-if="weixinStatus?.status"
            @click="disconnectWeixinChannel"
            :disabled="weixinLoading"
            class="px-4 py-2 bg-red-600 hover:bg-red-700 disabled:bg-gray-600 text-white font-medium rounded-lg transition-colors"
          >
            {{ t('channels.disconnect') }}
          </button>
        </div>

        <p v-if="weixinError" class="text-red-400 text-sm">{{ weixinError }}</p>
        <p v-if="weixinStatus?.account_id" class="text-gray-400 text-sm">
          Account: {{ weixinStatus.account_id }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from '../lib/i18n'
import {
  getChannelsStatus,
  saveDingTalkConfig,
  connectDingTalk,
  disconnectDingTalk,
  getWeixinQRCode,
  getWeixinStatus,
  disconnectWeixin,
  type ChannelStatus,
  type WeixinStatusResponse
} from '../lib/channels-api'

const { t } = useI18n()

// DingTalk state
const dingtalkConfig = ref({
  client_id: '',
  client_secret: '',
  allowed_users: [] as string[]
})
const dingtalkStatus = ref<ChannelStatus | null>(null)
const dingtalkLoading = ref(false)
const dingtalkError = ref('')
const dingtalkSuccess = ref('')

// WeChat state
const weixinStatus = ref<WeixinStatusResponse | null>(null)
const weixinLoading = ref(false)
const weixinError = ref('')
const showQRCode = ref(false)
const qrcodeUrl = ref('')
const weixinLoginStatus = ref('')
const weixinSessionKey = ref('')
let weixinPollTimer: ReturnType<typeof setInterval> | null = null

// Fetch all channel status
async function fetchStatus() {
  try {
    const response = await getChannelsStatus()
    const channels = response.channels || {}
    dingtalkStatus.value = channels.dingtalk || null
    weixinStatus.value = channels.weixin ? {
      status: channels.weixin.connected,
      account_id: channels.weixin.account_id,
      message: channels.weixin.message
    } : null
  } catch (err) {
    console.error('Failed to fetch channel status:', err)
  }
}

// DingTalk functions
async function saveDingTalkConfigAndConnect() {
  dingtalkLoading.value = true
  dingtalkError.value = ''
  dingtalkSuccess.value = ''

  try {
    // Save config
    await saveDingTalkConfig({
      client_id: dingtalkConfig.value.client_id,
      client_secret: dingtalkConfig.value.client_secret,
      allowed_users: dingtalkConfig.value.allowed_users
    })

    // Connect
    await connectDingTalk()
    dingtalkSuccess.value = t('channels.binding_success')
    await fetchStatus()
  } catch (err: any) {
    dingtalkError.value = err.message || t('channels.binding_failed')
  } finally {
    dingtalkLoading.value = false
  }
}

async function disconnectDingTalkChannel() {
  dingtalkLoading.value = true
  dingtalkError.value = ''

  try {
    await disconnectDingTalk()
    await fetchStatus()
  } catch (err: any) {
    dingtalkError.value = err.message
  } finally {
    dingtalkLoading.value = false
  }
}

// WeChat functions
async function generateWeixinQRCode() {
  weixinLoading.value = true
  weixinError.value = ''
  showQRCode.value = true

  try {
    console.log('[generateWeixinQRCode] Fetching QR code...')
    const response = await getWeixinQRCode()
    console.log('[generateWeixinQRCode] Response:', response)
    weixinSessionKey.value = response.session_key

    const qrcodeData = response.qrcode_url || ''

    if (!qrcodeData) {
      console.error('[generateWeixinQRCode] Empty qrcode_url')
      throw new Error('Failed to get QR code')
    }

    // QRCodeImgContent 可能是:
    // 1. 一个图片 URL (https://...)
    // 2. base64 图片数据 (data:image/...)
    // 3. 纯 base64 字符串
    if (qrcodeData.startsWith('http')) {
      // 如果是 URL，生成二维码让用户扫描
      qrcodeUrl.value = `https://api.qrserver.com/v1/create-qr-code/?size=256x256&data=${encodeURIComponent(qrcodeData)}`
    } else if (qrcodeData.startsWith('data:image')) {
      // 已经是 base64 图片，直接显示
      qrcodeUrl.value = qrcodeData
    } else {
      // 纯 base64 字符串，添加前缀
      qrcodeUrl.value = `data:image/png;base64,${qrcodeData}`
    }

    weixinLoginStatus.value = t('channels.waiting_scan')

    // Start polling for status
    startWeixinPolling()
  } catch (err: any) {
    console.error('[generateWeixinQRCode] Error:', err)
    weixinError.value = err.message
    showQRCode.value = false
  } finally {
    weixinLoading.value = false
  }
}

function startWeixinPolling() {
  weixinPollTimer = setInterval(async () => {
    try {
      const status = await getWeixinStatus(weixinSessionKey.value)
      if (status.status === "confirmed") {
        // Binding successful
        weixinLoginStatus.value = t('channels.binding_success')
        stopWeixinPolling()
        showQRCode.value = false
        await fetchStatus()
      } else if (status.status === "expired") {
        weixinLoginStatus.value = t('channels.qrcode_expired')
        stopWeixinPolling()
      }
    } catch (err) {
      console.error('Polling error:', err)
    }
  }, 2000)
}

function stopWeixinPolling() {
  if (weixinPollTimer) {
    clearInterval(weixinPollTimer)
    weixinPollTimer = null
  }
}

function cancelQRCode() {
  stopWeixinPolling()
  showQRCode.value = false
  qrcodeUrl.value = ''
  weixinSessionKey.value = ''
  weixinLoginStatus.value = ''
}

async function disconnectWeixinChannel() {
  weixinLoading.value = true
  weixinError.value = ''

  try {
    await disconnectWeixin()
    await fetchStatus()
  } catch (err: any) {
    weixinError.value = err.message
  } finally {
    weixinLoading.value = false
  }
}

onMounted(() => {
  fetchStatus()
})

onUnmounted(() => {
  stopWeixinPolling()
})
</script>