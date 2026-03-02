<template>
  <div id="app">
    <!-- Loading state -->
    <div v-if="loading" class="min-h-screen bg-gray-950 flex items-center justify-center">
      <p class="text-gray-400">{{ t('common.loading') }}</p>
    </div>

    <!-- Pairing dialog when not authenticated -->
    <div v-else-if="!isAuthenticated" class="min-h-screen bg-gray-950 flex items-center justify-center">
      <div class="bg-gray-900 rounded-xl p-8 w-full max-w-md border border-gray-800">
        <div class="text-center mb-6">
          <h1 class="text-2xl font-bold text-white mb-2">ZeroClaw</h1>
          <p class="text-gray-400">{{ t('auth.enter_code') }}</p>
        </div>
        <form @submit.prevent="handlePair">
          <input
            v-model="pairingCode"
            type="text"
            :placeholder="t('auth.pairing_code')"
            class="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white text-center text-2xl tracking-widest focus:outline-none focus:border-blue-500 mb-4"
            maxlength="6"
            autofocus
          />
          <p v-if="error" class="text-red-400 text-sm mb-4 text-center">{{ error }}</p>
          <button
            type="submit"
            :disabled="loadingPair || pairingCode.length < 6"
            class="w-full py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded-lg font-medium transition-colors"
          >
            {{ loadingPair ? t('common.loading') : t('auth.pair_button') }}
          </button>
        </form>
      </div>
    </div>

    <!-- Main app when authenticated -->
    <router-view v-else />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useAuth } from './hooks/useAuth'
import { useI18n } from './lib/i18n'

const { isAuthenticated, loading, pair, logout } = useAuth()
const { t } = useI18n()

const pairingCode = ref('')
const loadingPair = ref(false)
const error = ref('')

const handlePair = async () => {
  if (pairingCode.value.length < 6) return
  
  loadingPair.value = true
  error.value = ''
  
  try {
    await pair(pairingCode.value)
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : t('auth.pairing_failed')
  } finally {
    loadingPair.value = false
  }
}

const handleUnauthorized = () => {
  logout()
}

onMounted(() => {
  window.addEventListener('zeroclaw-unauthorized', handleUnauthorized)
})

onUnmounted(() => {
  window.removeEventListener('zeroclaw-unauthorized', handleUnauthorized)
})
</script>
