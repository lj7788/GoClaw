<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Brain class="h-5 w-5 text-blue-400" />
        <h2 class="text-base font-semibold text-white">
          {{ t('memory.title') }} ({{ entries.length }})
        </h2>
      </div>
      <div class="flex items-center gap-3">
        <button
          v-if="selectedKeys.size > 0"
          @click="showBatchDeleteConfirm = true"
          class="flex items-center gap-2 bg-red-600 hover:bg-red-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Trash2 class="h-4 w-4" />
          {{ t('common.delete') }} ({{ selectedKeys.size }})
        </button>
        <button
          @click="showForm = true"
          class="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus class="h-4 w-4" />
          {{ t('memory.add') }}
        </button>
      </div>
    </div>

    <!-- Search and Filter -->
    <div class="flex flex-col sm:flex-row gap-3">
      <div class="relative flex-1">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
        <input
          v-model="search"
          type="text"
          @keydown.enter="handleSearch"
          :placeholder="t('memory.search')"
          class="w-full bg-gray-900 border border-gray-700 rounded-lg pl-10 pr-4 py-2.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>
      <div class="relative">
        <Filter class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
        <select
          v-model="categoryFilter"
          class="bg-gray-900 border border-gray-700 rounded-lg pl-10 pr-8 py-2.5 text-sm text-white appearance-none focus:outline-none focus:ring-2 focus:ring-blue-500 cursor-pointer"
        >
          <option value="">{{ t('memory.all_categories') }}</option>
          <option v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</option>
        </select>
      </div>
      <button
        @click="handleSearch"
        class="px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
      >
        {{ t('common.search') }}
      </button>
    </div>

    <!-- Error banner -->
    <div v-if="error" class="rounded-lg bg-red-900/30 border border-red-700 p-3 text-sm text-red-300">
      {{ t('common.error') }}: {{ error }}
    </div>

    <!-- Batch Delete Confirmation Modal -->
    <div v-if="showBatchDeleteConfirm" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold text-white">{{ t('common.confirm') }} {{ t('common.delete') }}</h3>
          <button @click="showBatchDeleteConfirm = false" class="text-gray-400 hover:text-white transition-colors">
            <X class="h-5 w-5" />
          </button>
        </div>
        <p class="text-gray-300 mb-6">
          {{ t('memory.confirm_delete') }} ({{ selectedKeys.size }})
        </p>
        <div class="flex justify-end gap-3">
          <button
            @click="showBatchDeleteConfirm = false"
            class="px-4 py-2 text-sm font-medium text-gray-300 hover:text-white border border-gray-700 rounded-lg hover:bg-gray-800 transition-colors"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            @click="handleBatchDelete"
            class="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-lg transition-colors"
          >
            {{ t('common.confirm') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Add Memory Form Modal -->
    <div v-if="showForm" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold text-white">{{ t('memory.add') }}</h3>
          <button @click="closeForm" class="text-gray-400 hover:text-white transition-colors">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div v-if="formError" class="mb-4 rounded-lg bg-red-900/30 border border-red-700 p-3 text-sm text-red-300">
          {{ formError }}
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">
              {{ t('memory.key') }} <span class="text-red-400">*</span>
            </label>
            <input
              v-model="formKey"
              type="text"
              placeholder="e.g. user_preferences"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">
              {{ t('memory.content') }} <span class="text-red-400">*</span>
            </label>
            <textarea
              v-model="formContent"
              :placeholder="t('memory.content')"
              rows="4"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">
              {{ t('memory.category') }} ({{ t('common.no') }})
            </label>
            <input
              v-model="formCategory"
              type="text"
              placeholder="e.g. preferences, context, facts"
              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button
            @click="closeForm"
            class="px-4 py-2 text-sm font-medium text-gray-300 hover:text-white border border-gray-700 rounded-lg hover:bg-gray-800 transition-colors"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            @click="handleAdd"
            :disabled="submitting"
            class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors disabled:opacity-50"
          >
            {{ submitting ? t('common.loading') : t('common.save') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Memory Table -->
    <div v-if="loading" class="flex items-center justify-center h-32">
      <div class="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent" />
    </div>
    <div v-else-if="entries.length === 0" class="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center">
      <Brain class="h-10 w-10 text-gray-600 mx-auto mb-3" />
      <p class="text-gray-400">{{ t('memory.empty') }}</p>
    </div>
    <div v-else class="bg-gray-900 rounded-xl border border-gray-800 overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800">
            <th class="text-left px-4 py-3 text-gray-400 font-medium">
              <input
                type="checkbox"
                :checked="entries.length > 0 && selectedKeys.size === entries.length"
                @change="handleSelectAll"
                class="rounded border-gray-600 text-blue-600 focus:ring-blue-500"
              />
            </th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('memory.key') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('memory.content') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('memory.category') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('memory.timestamp') }}</th>
            <th class="text-right px-4 py-3 text-gray-400 font-medium">{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="entry in entries"
            :key="entry.id"
            class="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors"
          >
            <td class="px-4 py-3">
              <input
                type="checkbox"
                :checked="selectedKeys.has(entry.key)"
                @change="handleSelectKey(entry.key)"
                class="rounded border-gray-600 text-blue-600 focus:ring-blue-500"
              />
            </td>
            <td class="px-4 py-3 text-white font-medium font-mono text-xs">
              {{ entry.key }}
            </td>
            <td class="px-4 py-3 text-gray-300 max-w-[300px]">
              <span :title="entry.content">{{ truncate(entry.content, 80) }}</span>
            </td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-800 text-gray-300 capitalize">
                {{ entry.category }}
              </span>
            </td>
            <td class="px-4 py-3 text-gray-400 text-xs whitespace-nowrap">
              {{ formatDate(entry.timestamp) }}
            </td>
            <td class="px-4 py-3 text-right">
              <div v-if="confirmDelete === entry.key" class="flex items-center justify-end gap-2">
                <span class="text-xs text-red-400">{{ t('common.confirm') }}?</span>
                <button @click="handleDelete(entry.key)" class="text-red-400 hover:text-red-300 text-xs font-medium">
                  {{ t('common.yes') }}
                </button>
                <button @click="confirmDelete = null" class="text-gray-400 hover:text-white text-xs font-medium">
                  {{ t('common.no') }}
                </button>
              </div>
              <button
                v-else
                @click="confirmDelete = entry.key"
                class="text-gray-400 hover:text-red-400 transition-colors"
              >
                <Trash2 class="h-4 w-4" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Brain,
  Search,
  Plus,
  Trash2,
  X,
  Filter,
} from 'lucide-vue-next'
import type { MemoryEntry } from '../types/api'
import { getMemory, storeMemory, deleteMemory } from '../lib/api'
import { useI18n } from '../lib/i18n'

const { t } = useI18n()

const entries = ref<MemoryEntry[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const search = ref('')
const categoryFilter = ref('')
const showForm = ref(false)
const confirmDelete = ref<string | null>(null)
const selectedKeys = ref(new Set<string>())
const showBatchDeleteConfirm = ref(false)

const formKey = ref('')
const formContent = ref('')
const formCategory = ref('')
const formError = ref<string | null>(null)
const submitting = ref(false)

function truncate(text: string, max: number): string {
  if (text.length <= max) return text
  return text.slice(0, max) + '...'
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString()
}

const categories = computed(() => {
  return Array.from(new Set(entries.value.map((e) => e.category))).sort()
})

const fetchEntries = (q?: string, cat?: string) => {
  loading.value = true
  getMemory(q || undefined, cat || undefined)
    .then((data) => {
      entries.value = data
    })
    .catch((err) => {
      error.value = err instanceof Error ? err.message : t('common.error')
    })
    .finally(() => {
      loading.value = false
    })
}

const handleSearch = () => {
  fetchEntries(search.value, categoryFilter.value)
}

const closeForm = () => {
  showForm.value = false
  formError.value = null
  formKey.value = ''
  formContent.value = ''
  formCategory.value = ''
}

const handleAdd = async () => {
  if (!formKey.value.trim() || !formContent.value.trim()) {
    formError.value = t('memory.key') + ' and ' + t('memory.content') + ' are required'
    return
  }
  submitting.value = true
  formError.value = null
  try {
    await storeMemory(formKey.value.trim(), formContent.value.trim(), formCategory.value.trim() || undefined)
    fetchEntries(search.value, categoryFilter.value)
    closeForm()
  } catch (err) {
    formError.value = err instanceof Error ? err.message : t('common.error')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (key: string) => {
  try {
    const result = await deleteMemory(key)
    if (result.deleted) {
      entries.value = entries.value.filter((e) => e.key !== key)
      selectedKeys.value.delete(key)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('common.error')
  } finally {
    confirmDelete.value = null
  }
}

const handleBatchDelete = async () => {
  const keysToDelete = Array.from(selectedKeys.value)
  if (keysToDelete.length === 0) return

  try {
    for (const key of keysToDelete) {
      const result = await deleteMemory(key)
      if (!result.deleted) {
        throw new Error(`Failed to delete ${key}`)
      }
    }
    entries.value = entries.value.filter((e) => !selectedKeys.value.has(e.key))
    selectedKeys.value.clear()
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('common.error')
  } finally {
    showBatchDeleteConfirm.value = false
  }
}

const handleSelectAll = () => {
  if (selectedKeys.value.size === entries.value.length) {
    selectedKeys.value.clear()
  } else {
    selectedKeys.value = new Set(entries.value.map((e) => e.key))
  }
}

const handleSelectKey = (key: string) => {
  if (selectedKeys.value.has(key)) {
    selectedKeys.value.delete(key)
  } else {
    selectedKeys.value.add(key)
  }
}

onMounted(() => {
  fetchEntries()
})
</script>
