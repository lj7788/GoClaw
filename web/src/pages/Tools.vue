<template>
  <div class="p-6 space-y-6">
    <!-- Error state -->
    <div v-if="error" class="rounded-lg bg-red-900/30 border border-red-700 p-4 text-red-300">
      {{ t('common.error') }}: {{ error }}
    </div>

    <!-- Loading state -->
    <div v-else-if="loading" class="flex items-center justify-center h-64">
      <div class="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent" />
    </div>

    <template v-else>
      <!-- Search -->
      <div class="relative max-w-md">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
        <input
          v-model="search"
          type="text"
          :placeholder="t('tools.search')"
          class="w-full bg-gray-900 border border-gray-700 rounded-lg pl-10 pr-4 py-2.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      <!-- Agent Tools Grid -->
      <div>
        <div class="flex items-center gap-2 mb-4">
          <Wrench class="h-5 w-5 text-blue-400" />
          <h2 class="text-base font-semibold text-white">
            {{ t('tools.title') }} ({{ filtered.length }})
          </h2>
        </div>

        <p v-if="filtered.length === 0" class="text-sm text-gray-500">
          {{ t('tools.empty') }}
        </p>
        <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          <div
            v-for="tool in filtered"
            :key="tool.name"
            class="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden"
          >
            <button
              @click="toggleExpand(tool.name)"
              class="w-full text-left p-4 hover:bg-gray-800/50 transition-colors"
            >
              <div class="flex items-start justify-between gap-2">
                <div class="flex items-center gap-2 min-w-0">
                  <Package class="h-4 w-4 text-blue-400 flex-shrink-0 mt-0.5" />
                  <h3 class="text-sm font-semibold text-white truncate">
                    {{ tool.name }}
                  </h3>
                </div>
                <ChevronDown v-if="expandedTool === tool.name" class="h-4 w-4 text-gray-400 flex-shrink-0" />
                <ChevronRight v-else class="h-4 w-4 text-gray-400 flex-shrink-0" />
              </div>
              <p class="text-sm text-gray-400 mt-2 line-clamp-2">
                {{ tool.description }}
              </p>
            </button>

            <div v-if="expandedTool === tool.name && tool.parameters" class="border-t border-gray-800 p-4">
              <p class="text-xs text-gray-500 mb-2 font-medium uppercase tracking-wider">
                {{ t('tools.parameters') }}
              </p>
              <pre class="text-xs text-gray-300 bg-gray-950 rounded-lg p-3 overflow-x-auto max-h-64 overflow-y-auto">{{ JSON.stringify(tool.parameters, null, 2) }}</pre>
            </div>
          </div>
        </div>
      </div>

      <!-- CLI Tools Section -->
      <div v-if="filteredCli.length > 0">
        <div class="flex items-center gap-2 mb-4">
          <Terminal class="h-5 w-5 text-green-400" />
          <h2 class="text-base font-semibold text-white">
            CLI {{ t('tools.title') }} ({{ filteredCli.length }})
          </h2>
        </div>

        <div class="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-gray-800">
                <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('common.name') }}</th>
                <th class="text-left px-4 py-3 text-gray-400 font-medium">Path</th>
                <th class="text-left px-4 py-3 text-gray-400 font-medium">Version</th>
                <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('memory.category') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="tool in filteredCli"
                :key="tool.name"
                class="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors"
              >
                <td class="px-4 py-3 text-white font-medium">
                  {{ tool.name }}
                </td>
                <td class="px-4 py-3 text-gray-400 font-mono text-xs truncate max-w-[200px]">
                  {{ tool.path }}
                </td>
                <td class="px-4 py-3 text-gray-400">
                  {{ tool.version ?? '-' }}
                </td>
                <td class="px-4 py-3">
                  <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-800 text-gray-300 capitalize">
                    {{ tool.category }}
                  </span>
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
  Wrench,
  Search,
  ChevronDown,
  ChevronRight,
  Terminal,
  Package,
} from 'lucide-vue-next'
import type { ToolSpec, CliTool } from '../types/api'
import { getTools, getCliTools } from '../lib/api'
import { useI18n } from '../lib/i18n'

const { t } = useI18n()

const tools = ref<ToolSpec[]>([])
const cliTools = ref<CliTool[]>([])
const search = ref('')
const expandedTool = ref<string | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const filtered = computed(() => {
  return tools.value.filter(
    (tool) =>
      tool.name.toLowerCase().includes(search.value.toLowerCase()) ||
      tool.description.toLowerCase().includes(search.value.toLowerCase())
  )
})

const filteredCli = computed(() => {
  return cliTools.value.filter(
    (tool) =>
      tool.name.toLowerCase().includes(search.value.toLowerCase()) ||
      tool.category.toLowerCase().includes(search.value.toLowerCase())
  )
})

function toggleExpand(name: string) {
  expandedTool.value = expandedTool.value === name ? null : name
}

onMounted(() => {
  Promise.all([getTools(), getCliTools()])
    .then(([t, c]) => {
      tools.value = t
      cliTools.value = c
    })
    .catch((err) => {
      error.value = err instanceof Error ? err.message : t('common.error')
    })
    .finally(() => {
      loading.value = false
    })
})
</script>
