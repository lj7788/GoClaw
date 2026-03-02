<template>
  <div class="p-6 space-y-6">
    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center h-64">
      <div class="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent" />
    </div>

    <template v-else>
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Settings class="h-5 w-5 text-blue-400" />
          <h2 class="text-base font-semibold text-white">{{ t('config.title') }}</h2>
        </div>
        <button
          @click="handleSave"
          :disabled="saving"
          class="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors disabled:opacity-50"
        >
          <Save class="h-4 w-4" />
          {{ saving ? t('common.loading') : t('config.save') }}
        </button>
      </div>

      <!-- Sensitive fields note -->
      <div class="flex items-start gap-3 bg-yellow-900/20 border border-yellow-700/40 rounded-lg p-4">
        <ShieldAlert class="h-5 w-5 text-yellow-400 flex-shrink-0 mt-0.5" />
        <div>
          <p class="text-sm text-yellow-300 font-medium">
            {{ t('config.sensitive_notice') || 'Sensitive fields are masked' }}
          </p>
          <p class="text-sm text-yellow-400/70 mt-0.5">
            {{ t('config.sensitive_warning') || 'API keys, tokens, and passwords are hidden for security. To update masked fields, replace the entire masked value with a new value.' }}
          </p>
        </div>
      </div>

      <!-- Success message -->
      <div v-if="success" class="flex items-center gap-2 bg-green-900/30 border border-green-700 rounded-lg p-3">
        <CheckCircle class="h-4 w-4 text-green-400 flex-shrink-0" />
        <span class="text-sm text-green-300">{{ success }}</span>
      </div>

      <!-- Error message -->
      <div v-if="error" class="flex items-center gap-2 bg-red-900/30 border border-red-700 rounded-lg p-3">
        <AlertTriangle class="h-4 w-4 text-red-400 flex-shrink-0" />
        <span class="text-sm text-red-300">{{ error }}</span>
      </div>

      <!-- Config Editor -->
      <div class="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
        <div class="flex items-center justify-between px-4 py-2 border-b border-gray-800 bg-gray-800/50">
          <span class="text-xs text-gray-400 font-medium uppercase tracking-wider">
            TOML {{ t('config.title') }}
          </span>
          <span class="text-xs text-gray-500">
            {{ config.split('\n').length }} lines
          </span>
        </div>
        <textarea
          v-model="config"
          spellcheck="false"
          class="w-full min-h-[500px] bg-gray-950 text-gray-200 font-mono text-sm p-4 resize-y focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-inset"
          style="tab-size: 4"
        />
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  Settings,
  Save,
  CheckCircle,
  AlertTriangle,
  ShieldAlert,
} from 'lucide-vue-next'
import { getConfig, putConfig } from '../lib/api'
import { useI18n } from '../lib/i18n'

const { t } = useI18n()

const config = ref('')
const loading = ref(true)
const saving = ref(false)
const error = ref<string | null>(null)
const success = ref<string | null>(null)

const handleSave = async () => {
  saving.value = true
  error.value = null
  success.value = null
  try {
    await putConfig(config.value)
    success.value = t('config.saved')
    setTimeout(() => {
      success.value = null
    }, 4000)
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('config.error')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  getConfig()
    .then((data) => {
      config.value = typeof data === 'string' ? data : JSON.stringify(data, null, 2)
    })
    .catch((err) => {
      error.value = err instanceof Error ? err.message : t('common.error')
    })
    .finally(() => {
      loading.value = false
    })
})
</script>
