<template>
  <div class="p-6 space-y-6">
    <!-- Error state -->
    <div v-if="error" class="rounded-lg bg-red-900/30 border border-red-700 p-4 text-red-300">
      {{ t('common.error') }}: {{ error }}
    </div>

    <!-- Loading state -->
    <div v-else-if="loading || !cost" class="flex items-center justify-center h-64">
      <div class="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent" />
    </div>

    <template v-else>
      <!-- Summary Cards -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-3 mb-3">
            <div class="p-2 bg-blue-600/20 rounded-lg">
              <DollarSign class="h-5 w-5 text-blue-400" />
            </div>
            <span class="text-sm text-gray-400">{{ t('cost.session') }}</span>
          </div>
          <p class="text-2xl font-bold text-white">
            {{ formatUSD(cost.session_cost_usd) }}
          </p>
        </div>

        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-3 mb-3">
            <div class="p-2 bg-green-600/20 rounded-lg">
              <TrendingUp class="h-5 w-5 text-green-400" />
            </div>
            <span class="text-sm text-gray-400">{{ t('cost.daily') }}</span>
          </div>
          <p class="text-2xl font-bold text-white">
            {{ formatUSD(cost.daily_cost_usd) }}
          </p>
        </div>

        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-3 mb-3">
            <div class="p-2 bg-purple-600/20 rounded-lg">
              <Layers class="h-5 w-5 text-purple-400" />
            </div>
            <span class="text-sm text-gray-400">{{ t('cost.monthly') }}</span>
          </div>
          <p class="text-2xl font-bold text-white">
            {{ formatUSD(cost.monthly_cost_usd) }}
          </p>
        </div>

        <div class="bg-gray-900 rounded-xl p-5 border border-gray-800">
          <div class="flex items-center gap-3 mb-3">
            <div class="p-2 bg-orange-600/20 rounded-lg">
              <Hash class="h-5 w-5 text-orange-400" />
            </div>
            <span class="text-sm text-gray-400">{{ t('cost.request_count') }}</span>
          </div>
          <p class="text-2xl font-bold text-white">
            {{ cost.request_count.toLocaleString() }}
          </p>
        </div>
      </div>

      <!-- Token Statistics -->
      <div class="bg-gray-900 rounded-xl border border-gray-800 p-5">
        <h3 class="text-base font-semibold text-white mb-4">
          {{ t('cost.total_tokens') }}
        </h3>
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div class="bg-gray-800/50 rounded-lg p-4">
            <p class="text-sm text-gray-400">{{ t('cost.total_tokens') }}</p>
            <p class="text-xl font-bold text-white mt-1">
              {{ cost.total_tokens.toLocaleString() }}
            </p>
          </div>
          <div class="bg-gray-800/50 rounded-lg p-4">
            <p class="text-sm text-gray-400">{{ t('cost.tokens') }} / {{ t('cost.requests') }}</p>
            <p class="text-xl font-bold text-white mt-1">
              {{ cost.request_count > 0 ? Math.round(cost.total_tokens / cost.request_count).toLocaleString() : '0' }}
            </p>
          </div>
          <div class="bg-gray-800/50 rounded-lg p-4">
            <p class="text-sm text-gray-400">{{ t('cost.usd') }} / 1K {{ t('cost.tokens') }}</p>
            <p class="text-xl font-bold text-white mt-1">
              {{ cost.total_tokens > 0 ? formatUSD((cost.monthly_cost_usd / cost.total_tokens) * 1000) : '$0.0000' }}
            </p>
          </div>
        </div>
      </div>

      <!-- Model Breakdown Table -->
      <div class="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
        <div class="px-5 py-4 border-b border-gray-800">
          <h3 class="text-base font-semibold text-white">
            {{ t('cost.by_model') }}
          </h3>
        </div>
        <div v-if="models.length === 0" class="p-8 text-center text-gray-500">
          {{ t('common.no_data') }}
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-gray-800">
                <th class="text-left px-5 py-3 text-gray-400 font-medium">{{ t('cost.model') }}</th>
                <th class="text-right px-5 py-3 text-gray-400 font-medium">{{ t('cost.usd') }}</th>
                <th class="text-right px-5 py-3 text-gray-400 font-medium">{{ t('cost.tokens') }}</th>
                <th class="text-right px-5 py-3 text-gray-400 font-medium">{{ t('cost.requests') }}</th>
                <th class="text-left px-5 py-3 text-gray-400 font-medium">%</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="m in sortedModels"
                :key="m.model"
                class="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors"
              >
                <td class="px-5 py-3 text-white font-medium">{{ m.model }}</td>
                <td class="px-5 py-3 text-gray-300 text-right font-mono">{{ formatUSD(m.cost_usd) }}</td>
                <td class="px-5 py-3 text-gray-300 text-right">{{ m.total_tokens.toLocaleString() }}</td>
                <td class="px-5 py-3 text-gray-300 text-right">{{ m.request_count.toLocaleString() }}</td>
                <td class="px-5 py-3">
                  <div class="flex items-center gap-2">
                    <div class="w-20 h-2 bg-gray-800 rounded-full overflow-hidden">
                      <div
                        class="h-full bg-blue-500 rounded-full"
                        :style="{ width: `${Math.max(modelShare(m), 2)}%` }"
                      />
                    </div>
                    <span class="text-xs text-gray-400 w-10 text-right">{{ modelShare(m).toFixed(1) }}%</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  DollarSign,
  TrendingUp,
  Hash,
  Layers,
} from 'lucide-vue-next'
import type { CostSummary } from '../types/api'
import { getCost } from '../lib/api'
import { useI18n } from '../lib/i18n'

const { t } = useI18n()

const cost = ref<CostSummary | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

function formatUSD(value: number): string {
  return `$${value.toFixed(4)}`
}

const models = computed(() => {
  if (!cost.value) return []
  return Object.values(cost.value.by_model)
})

const sortedModels = computed(() => {
  return [...models.value].sort((a, b) => b.cost_usd - a.cost_usd)
})

function modelShare(m: { cost_usd: number }): number {
  if (!cost.value || cost.value.monthly_cost_usd <= 0) return 0
  return (m.cost_usd / cost.value.monthly_cost_usd) * 100
}

onMounted(() => {
  getCost()
    .then((data) => {
      cost.value = data
    })
    .catch((err) => {
      error.value = err instanceof Error ? err.message : t('common.error')
    })
    .finally(() => {
      loading.value = false
    })
})
</script>
