<template>
  <div class="p-6 space-y-6">
    <!-- Loading state -->
    <div v-if="loading" class="flex items-center justify-center h-64">
      <div class="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent" />
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="p-6">
      <div class="rounded-lg bg-red-900/30 border border-red-700 p-4 text-red-300">
        {{ t('common.error') }}: {{ error }}
      </div>
    </div>

    <template v-else>
      <!-- Status Cards Grid -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <!-- Provider/Model Card -->
        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-3 mb-3">
            <div class="p-2 bg-blue-600/20 rounded-lg">
              <Cpu class="h-5 w-5 text-blue-400" />
            </div>
            <span class="text-sm text-gray-400">{{ t('dashboard.provider') }} / {{ t('dashboard.model') }}</span>
          </div>
          <p class="text-lg font-semibold text-white truncate">
            {{ status?.provider ?? 'N/A' }}
          </p>
          <p class="text-sm text-gray-400 truncate">{{ status?.model ?? 'N/A' }}</p>
        </div>

        <!-- Uptime Card -->
        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-3 mb-3">
            <div class="p-2 bg-green-600/20 rounded-lg">
              <Clock class="h-5 w-5 text-green-400" />
            </div>
            <span class="text-sm text-gray-400">{{ t('dashboard.uptime') }}</span>
          </div>
          <p class="text-lg font-semibold text-white">
            {{ formatUptime(status?.uptime_seconds ?? 0) }}
          </p>
          <p class="text-sm text-gray-400">{{ t('dashboard.uptime') }}</p>
        </div>

        <!-- Gateway Port Card -->
        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-3 mb-3">
            <div class="p-2 bg-purple-600/20 rounded-lg">
              <Globe class="h-5 w-5 text-purple-400" />
            </div>
            <span class="text-sm text-gray-400">{{ t('dashboard.gateway_port') }}</span>
          </div>
          <p class="text-lg font-semibold text-white">
            :{{ status?.gateway_port }}
          </p>
          <p class="text-sm text-gray-400">{{ t('dashboard.locale') }}: {{ status?.locale ?? 'N/A' }}</p>
        </div>

        <!-- Memory Backend Card -->
        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-3 mb-3">
            <div class="p-2 bg-orange-600/20 rounded-lg">
              <Database class="h-5 w-5 text-orange-400" />
            </div>
            <span class="text-sm text-gray-400">{{ t('dashboard.memory_backend') }}</span>
          </div>
          <p class="text-lg font-semibold text-white capitalize">
            {{ status?.memory_backend ?? 'N/A' }}
          </p>
          <p class="text-sm text-gray-400">
            {{ t('dashboard.paired') }}: {{ status?.paired ? t('common.yes') : t('common.no') }}
          </p>
        </div>
      </div>

      <!-- Second Row -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Cost Widget -->
        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-2 mb-4">
            <DollarSign class="h-5 w-5 text-blue-400" />
            <h2 class="text-base font-semibold text-white">{{ t('cost.title') }}</h2>
          </div>
          <div class="space-y-4">
            <div v-for="item in costItems" :key="item.label">
              <div class="flex justify-between text-sm mb-1">
                <span class="text-gray-400">{{ item.label }}</span>
                <span class="text-white font-medium">{{ formatUSD(item.value) }}</span>
              </div>
              <div class="w-full h-2 bg-gray-800 rounded-full overflow-hidden">
                <div
                  :class="['h-full rounded-full', item.color]"
                  :style="{ width: `${Math.max((item.value / maxCost) * 100, 2)}%` }"
                />
              </div>
            </div>
          </div>
          <div class="mt-4 pt-3 border-t border-gray-800 flex justify-between text-sm">
            <span class="text-gray-400">{{ t('cost.total_tokens') }}</span>
            <span class="text-white">{{ cost?.total_tokens.toLocaleString() ?? 0 }}</span>
          </div>
          <div class="flex justify-between text-sm mt-1">
            <span class="text-gray-400">{{ t('cost.request_count') }}</span>
            <span class="text-white">{{ cost?.request_count.toLocaleString() ?? 0 }}</span>
          </div>
        </div>

        <!-- Active Channels -->
        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-2 mb-4">
            <Radio class="h-5 w-5 text-blue-400" />
            <h2 class="text-base font-semibold text-white">{{ t('dashboard.channels') }}</h2>
          </div>
          <div class="space-y-2">
            <template v-if="Object.keys(status?.channels ?? {}).length === 0">
              <p class="text-sm text-gray-500">{{ t('common.no_data') }}</p>
            </template>
            <template v-else>
              <div
                v-for="(active, name) in status?.channels"
                :key="name"
                class="flex items-center justify-between py-2 px-3 rounded-lg bg-gray-800/50"
              >
                <span class="text-sm text-white capitalize">{{ name }}</span>
                <div class="flex items-center gap-2">
                  <span
                    :class="['inline-block h-2.5 w-2.5 rounded-full', active ? 'bg-green-500' : 'bg-gray-500']"
                  />
                  <span class="text-xs text-gray-400">
                    {{ active ? t('common.yes') : t('common.no') }}
                  </span>
                </div>
              </div>
            </template>
          </div>
        </div>

        <!-- Health Grid -->
        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-2 mb-4">
            <Activity class="h-5 w-5 text-blue-400" />
            <h2 class="text-base font-semibold text-white">{{ t('health.title') }}</h2>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <template v-if="Object.keys(status?.health?.components ?? {}).length === 0">
              <p class="text-sm text-gray-500 col-span-2">{{ t('common.no_data') }}</p>
            </template>
            <template v-else>
              <div
                v-for="(comp, name) in status?.health?.components"
                :key="name"
                :class="['rounded-lg p-3 border', healthBorder(comp.status)]"
              >
                <div class="flex items-center gap-2 mb-1">
                  <span :class="['inline-block h-2 w-2 rounded-full', healthColor(comp.status)]" />
                  <span class="text-sm font-medium text-white capitalize truncate">
                    {{ name }}
                  </span>
                </div>
                <p class="text-xs text-gray-400 capitalize">{{ comp.status }}</p>
                <p v-if="comp.restart_count > 0" class="text-xs text-yellow-400 mt-1">
                  {{ t('health.restart_count') }}: {{ comp.restart_count }}
                </p>
              </div>
            </template>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Cpu,
  Clock,
  Globe,
  Database,
  DollarSign,
  Radio,
  Activity,
} from 'lucide-vue-next'
import type { StatusResponse, CostSummary } from '../types/api'
import { getStatus, getCost } from '../lib/api'
import { useI18n } from '../lib/i18n'

const { t } = useI18n()

const status = ref<StatusResponse | null>(null)
const cost = ref<CostSummary | null>(null)
const error = ref<string | null>(null)
const loading = ref(true)

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h ${m}m`
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

function formatUSD(value: number): string {
  return `$${value.toFixed(4)}`
}

function healthColor(statusVal: string): string {
  switch (statusVal.toLowerCase()) {
    case 'ok':
    case 'healthy':
      return 'bg-green-500'
    case 'warn':
    case 'warning':
    case 'degraded':
      return 'bg-yellow-500'
    default:
      return 'bg-red-500'
  }
}

function healthBorder(statusVal: string): string {
  switch (statusVal.toLowerCase()) {
    case 'ok':
    case 'healthy':
      return 'border-green-500/30 bg-gray-800/50'
    case 'warn':
    case 'warning':
    case 'degraded':
      return 'border-yellow-500/30 bg-gray-800/50'
    default:
      return 'border-red-500/30 bg-gray-800/50'
  }
}

const maxCost = computed(() => {
  if (!cost.value) return 0.001
  return Math.max(
    cost.value.session_cost_usd,
    cost.value.daily_cost_usd,
    cost.value.monthly_cost_usd,
    0.001
  )
})

const costItems = computed(() => {
  if (!cost.value) return []
  return [
    { label: t('cost.session'), value: cost.value.session_cost_usd, color: 'bg-blue-500' },
    { label: t('cost.daily'), value: cost.value.daily_cost_usd, color: 'bg-green-500' },
    { label: t('cost.monthly'), value: cost.value.monthly_cost_usd, color: 'bg-purple-500' },
  ]
})

onMounted(async () => {
  try {
    const [s, c] = await Promise.all([getStatus(), getCost()])
    status.value = s
    cost.value = c
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('common.error')
  } finally {
    loading.value = false
  }
})
</script>
