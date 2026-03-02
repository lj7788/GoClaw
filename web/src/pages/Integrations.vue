<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center gap-2">
      <Puzzle class="h-5 w-5 text-blue-400" />
      <h2 class="text-base font-semibold text-white">
        {{ t('integrations.title') }} ({{ integrations.length }})
      </h2>
    </div>

    <!-- Error -->
    <div v-if="error" class="rounded-lg bg-red-900/30 border border-red-700 p-3 text-sm text-red-300">
      {{ t('common.error') }}: {{ error }}
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center h-64">
      <div class="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent" />
    </div>

    <template v-else>
      <!-- Category Filter Tabs -->
      <div class="flex flex-wrap gap-2">
        <button
          v-for="cat in categories"
          :key="cat"
          @click="activeCategory = cat"
          :class="[
            'px-3 py-1.5 rounded-lg text-sm font-medium transition-colors capitalize',
            activeCategory === cat
              ? 'bg-blue-600 text-white'
              : 'bg-gray-900 text-gray-400 border border-gray-700 hover:bg-gray-800 hover:text-white'
          ]"
        >
          {{ cat === 'all' ? 'All' : formatCategory(cat) }}
        </button>
      </div>

      <!-- Integration Cards -->
      <div v-if="filteredIntegrations.length === 0" class="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center">
        <Puzzle class="h-10 w-10 text-gray-600 mx-auto mb-3" />
        <p class="text-gray-400">{{ t('integrations.empty') }}</p>
      </div>

      <div v-else>
        <div v-for="[category, items] in Object.entries(groupedIntegrations)" :key="category">
          <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-3 capitalize">
            {{ formatCategory(category) }}
          </h3>
          <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 mb-6">
            <div
              v-for="integration in items"
              :key="integration.name"
              class="bg-gray-900 rounded-xl border border-gray-800 p-5 hover:border-gray-700 transition-colors"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <h4 class="text-sm font-semibold text-white truncate">
                    {{ integration.name }}
                  </h4>
                  <p class="text-sm text-gray-400 mt-1 line-clamp-2">
                    {{ integration.description }}
                  </p>
                </div>
                <span
                  :class="[
                    'flex-shrink-0 inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium border',
                    statusBadge(integration.status).classes
                  ]"
                >
                  <component :is="statusBadge(integration.status).icon" class="h-3 w-3" />
                  {{ statusBadge(integration.status).label }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { Puzzle, Check, Zap, Clock } from 'lucide-vue-next'
import type { Integration } from '../types/api'
import { getIntegrations } from '../lib/api'
import { useI18n } from '../lib/i18n'

const { t } = useI18n()

const integrations = ref<Integration[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const activeCategory = ref('all')

function formatCategory(category: string): string {
  if (!category) return category
  return category.replace(/([a-z])([A-Z])/g, '$1 $2').replace(/Ai/g, 'AI')
}

function statusBadge(status: Integration['status']) {
  switch (status) {
    case 'Active':
      return {
        icon: h(Check, { class: 'h-3 w-3' }),
        label: t('integrations.active'),
        classes: 'bg-green-900/40 text-green-400 border-green-700/50'
      }
    case 'Available':
      return {
        icon: h(Zap, { class: 'h-3 w-3' }),
        label: t('integrations.available'),
        classes: 'bg-blue-900/40 text-blue-400 border-blue-700/50'
      }
    case 'ComingSoon':
      return {
        icon: h(Clock, { class: 'h-3 w-3' }),
        label: t('integrations.coming_soon'),
        classes: 'bg-gray-800 text-gray-400 border-gray-700'
      }
  }
}

const categories = computed(() => {
  const cats = new Set(integrations.value.map(i => i.category))
  return ['all', ...Array.from(cats).sort()]
})

const filteredIntegrations = computed(() => {
  if (activeCategory.value === 'all') return integrations.value
  return integrations.value.filter(i => i.category === activeCategory.value)
})

const groupedIntegrations = computed(() => {
  const groups: Record<string, Integration[]> = {}
  for (const item of filteredIntegrations.value) {
    const cat = item.category || 'other'
    if (!groups[cat]) {
      groups[cat] = []
    }
    groups[cat].push(item)
  }
  return Object.fromEntries(
    Object.entries(groups).sort(([a], [b]) => a.localeCompare(b))
  )
})

onMounted(() => {
  getIntegrations()
    .then((data) => {
      integrations.value = data
    })
    .catch((err) => {
      error.value = err instanceof Error ? err.message : t('common.error')
    })
    .finally(() => {
      loading.value = false
    })
})
</script>
