<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Stethoscope class="h-5 w-5 text-blue-400" />
        <h2 class="text-base font-semibold text-white">{{ t('doctor.title') }}</h2>
      </div>
      <button
        @click="handleRun"
        :disabled="loading"
        class="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors disabled:opacity-50"
      >
        <template v-if="loading">
          <Loader2 class="h-4 w-4 animate-spin" />
          {{ t('doctor.running') }}
        </template>
        <template v-else>
          <Play class="h-4 w-4" />
          {{ t('doctor.run') }}
        </template>
      </button>
    </div>

    <!-- Error -->
    <div v-if="error" class="rounded-lg bg-red-900/30 border border-red-700 p-4 text-red-300">
      {{ error }}
    </div>

    <!-- Loading spinner -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-16">
      <Loader2 class="h-10 w-10 text-blue-500 animate-spin mb-4" />
      <p class="text-gray-400">{{ t('doctor.running') }}</p>
      <p class="text-sm text-gray-500 mt-1">
        {{ t('doctor.running') }}
      </p>
    </div>

    <!-- Results -->
    <template v-if="results && !loading">
      <!-- Summary Bar -->
      <div class="flex items-center gap-4 bg-gray-900 rounded-xl border border-gray-800 p-4">
        <div class="flex items-center gap-2">
          <CheckCircle class="h-5 w-5 text-green-400" />
          <span class="text-sm text-white font-medium">
            {{ okCount }} <span class="text-gray-400 font-normal">{{ t('doctor.ok') }}</span>
          </span>
        </div>
        <div class="w-px h-5 bg-gray-700" />
        <div class="flex items-center gap-2">
          <AlertTriangle class="h-5 w-5 text-yellow-400" />
          <span class="text-sm text-white font-medium">
            {{ warnCount }}
            <span class="text-gray-400 font-normal">{{ t('doctor.warn') }}</span>
          </span>
        </div>
        <div class="w-px h-5 bg-gray-700" />
        <div class="flex items-center gap-2">
          <XCircle class="h-5 w-5 text-red-400" />
          <span class="text-sm text-white font-medium">
            {{ errorCount }}
            <span class="text-gray-400 font-normal">{{ t('doctor.error') }}</span>
          </span>
        </div>

        <!-- Overall indicator -->
        <div class="ml-auto">
          <span
            v-if="errorCount > 0"
            class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-red-900/40 text-red-400 border border-red-700/50"
          >
            {{ t('doctor.error') }}
          </span>
          <span
            v-else-if="warnCount > 0"
            class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-yellow-900/40 text-yellow-400 border border-yellow-700/50"
          >
            {{ t('doctor.warn') }}
          </span>
          <span
            v-else
            class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-medium bg-green-900/40 text-green-400 border border-green-700/50"
          >
            {{ t('doctor.ok') }}
          </span>
        </div>
      </div>

      <!-- Grouped Results -->
      <div v-for="[category, items] in sortedGrouped" :key="category">
        <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-3 capitalize">
          {{ category }}
        </h3>
        <div class="space-y-2 mb-6">
          <div
            v-for="(result, idx) in items"
            :key="`${category}-${idx}`"
            :class="['flex items-start gap-3 rounded-lg border p-3', severityBorder(result.severity), severityBg(result.severity)]"
          >
            <component :is="severityIcon(result.severity)" class="h-4 w-4 flex-shrink-0" />
            <div class="min-w-0">
              <p class="text-sm text-white">{{ result.message }}</p>
              <p class="text-xs text-gray-500 mt-0.5 capitalize">
                {{ result.severity }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Empty state -->
    <div v-if="!results && !loading && !error" class="flex flex-col items-center justify-center py-16 text-gray-500">
      <Stethoscope class="h-12 w-12 text-gray-600 mb-4" />
      <p class="text-lg font-medium">{{ t('doctor.title') }}</p>
      <p class="text-sm mt-1">
        {{ t('doctor.empty') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import {
  Stethoscope,
  Play,
  CheckCircle,
  AlertTriangle,
  XCircle,
  Loader2,
} from 'lucide-vue-next'
import type { DiagResult } from '../types/api'
import { runDoctor } from '../lib/api'
import { useI18n } from '../lib/i18n'

const { t } = useI18n()

const results = ref<DiagResult[] | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

const okCount = computed(() => results.value?.filter((r) => r.severity === 'ok').length ?? 0)
const warnCount = computed(() => results.value?.filter((r) => r.severity === 'warn').length ?? 0)
const errorCount = computed(() => results.value?.filter((r) => r.severity === 'error').length ?? 0)

const grouped = computed(() => {
  if (!results.value) return {}
  return results.value.reduce<Record<string, DiagResult[]>>((acc, item) => {
    const key = item.category
    if (!acc[key]) acc[key] = []
    acc[key].push(item)
    return acc
  }, {})
})

const sortedGrouped = computed(() => {
  return Object.entries(grouped.value).sort(([a], [b]) => a.localeCompare(b))
})

function severityIcon(severity: DiagResult['severity']) {
  switch (severity) {
    case 'ok':
      return h(CheckCircle, { class: 'h-4 w-4 text-green-400' })
    case 'warn':
      return h(AlertTriangle, { class: 'h-4 w-4 text-yellow-400' })
    case 'error':
      return h(XCircle, { class: 'h-4 w-4 text-red-400' })
  }
}

function severityBorder(severity: DiagResult['severity']): string {
  switch (severity) {
    case 'ok':
      return 'border-green-700/40'
    case 'warn':
      return 'border-yellow-700/40'
    case 'error':
      return 'border-red-700/40'
  }
}

function severityBg(severity: DiagResult['severity']): string {
  switch (severity) {
    case 'ok':
      return 'bg-green-900/10'
    case 'warn':
      return 'bg-yellow-900/10'
    case 'error':
      return 'bg-red-900/10'
  }
}

const handleRun = async () => {
  loading.value = true
  error.value = null
  results.value = null
  try {
    const data = await runDoctor()
    results.value = data
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('common.error')
  } finally {
    loading.value = false
  }
}
</script>
