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
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Clock class="h-5 w-5 text-blue-400" />
          <h2 class="text-base font-semibold text-white">
            {{ t('cron.title') }} ({{ jobs.length }})
          </h2>
        </div>
        <button
          @click="showForm = true"
          class="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus class="h-4 w-4" />
          {{ t('cron.add') }}
        </button>
      </div>

      <!-- Add Job Form Modal -->
      <div v-if="showForm" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
        <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-semibold text-white">{{ t('cron.add') }}</h3>
            <button
              @click="closeForm"
              class="text-gray-400 hover:text-white transition-colors"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div v-if="formError" class="mb-4 rounded-lg bg-red-900/30 border border-red-700 p-3 text-sm text-red-300">
            {{ formError }}
          </div>

          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-300 mb-1">
                {{ t('cron.name') }} ({{ t('common.no') }})
              </label>
              <input
                v-model="formName"
                type="text"
                placeholder="e.g. Daily cleanup"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-300 mb-1">
                {{ t('cron.schedule') }} <span class="text-red-400">*</span>
              </label>
              <input
                v-model="formSchedule"
                type="text"
                placeholder="e.g. 0 0 * * *"
                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-300 mb-1">
                {{ t('cron.command') }} <span class="text-red-400">*</span>
              </label>
              <input
                v-model="formCommand"
                type="text"
                placeholder="e.g. cleanup --older-than 7d"
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
              {{ submitting ? t('common.loading') : t('cron.add') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Jobs Table -->
      <div v-if="jobs.length === 0" class="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center">
        <Clock class="h-10 w-10 text-gray-600 mx-auto mb-3" />
        <p class="text-gray-400">{{ t('cron.empty') }}</p>
      </div>
      <div v-else class="bg-gray-900 rounded-xl border border-gray-800 overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-gray-800">
              <th class="text-left px-4 py-3 text-gray-400 font-medium">ID</th>
              <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('cron.name') }}</th>
              <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('cron.command') }}</th>
              <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('cron.next_run') }}</th>
              <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('cron.last_status') }}</th>
              <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('cron.enabled') }}</th>
              <th class="text-right px-4 py-3 text-gray-400 font-medium">{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="job in jobs"
              :key="job.id"
              class="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors"
            >
              <td class="px-4 py-3 text-gray-400 font-mono text-xs">
                {{ job.id.slice(0, 8) }}
              </td>
              <td class="px-4 py-3 text-white font-medium">
                {{ job.name ?? '-' }}
              </td>
              <td class="px-4 py-3 text-gray-300 font-mono text-xs max-w-[200px] truncate">
                {{ job.command }}
              </td>
              <td class="px-4 py-3 text-gray-400 text-xs">
                {{ formatDate(job.next_run) }}
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-1.5">
                  <component :is="statusIcon(job.last_status)" class="h-4 w-4" />
                  <span class="text-gray-300 text-xs capitalize">
                    {{ job.last_status ?? '-' }}
                  </span>
                </div>
              </td>
              <td class="px-4 py-3">
                <span
                  :class="[
                    'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                    job.enabled
                      ? 'bg-green-900/40 text-green-400 border border-green-700/50'
                      : 'bg-gray-800 text-gray-500 border border-gray-700'
                  ]"
                >
                  {{ job.enabled ? t('cron.enabled') : t('cron.disabled') }}
                </span>
              </td>
              <td class="px-4 py-3 text-right">
                <div v-if="confirmDelete === job.id" class="flex items-center justify-end gap-2">
                  <span class="text-xs text-red-400">{{ t('cron.confirm_delete') }}</span>
                  <button
                    @click="handleDelete(job.id)"
                    class="text-red-400 hover:text-red-300 text-xs font-medium"
                  >
                    {{ t('common.yes') }}
                  </button>
                  <button
                    @click="confirmDelete = null"
                    class="text-gray-400 hover:text-white text-xs font-medium"
                  >
                    {{ t('common.no') }}
                  </button>
                </div>
                <button
                  v-else
                  @click="confirmDelete = job.id"
                  class="text-gray-400 hover:text-red-400 transition-colors"
                >
                  <Trash2 class="h-4 w-4" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import {
  Clock,
  Plus,
  Trash2,
  X,
  CheckCircle,
  XCircle,
  AlertCircle,
} from 'lucide-vue-next'
import type { CronJob } from '../types/api'
import { getCronJobs, addCronJob, deleteCronJob } from '../lib/api'
import { useI18n } from '../lib/i18n'

const { t } = useI18n()

const jobs = ref<CronJob[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const showForm = ref(false)
const confirmDelete = ref<string | null>(null)

const formName = ref('')
const formSchedule = ref('')
const formCommand = ref('')
const formError = ref<string | null>(null)
const submitting = ref(false)

function formatDate(iso: string | null): string {
  if (!iso) return '-'
  const d = new Date(iso)
  return d.toLocaleString()
}

function statusIcon(status: string | null) {
  if (!status) return null
  switch (status.toLowerCase()) {
    case 'ok':
    case 'success':
      return h(CheckCircle, { class: 'h-4 w-4 text-green-400' })
    case 'error':
    case 'failed':
      return h(XCircle, { class: 'h-4 w-4 text-red-400' })
    default:
      return h(AlertCircle, { class: 'h-4 w-4 text-yellow-400' })
  }
}

function closeForm() {
  showForm.value = false
  formError.value = null
  formName.value = ''
  formSchedule.value = ''
  formCommand.value = ''
}

const handleAdd = async () => {
  if (!formSchedule.value.trim() || !formCommand.value.trim()) {
    formError.value = t('cron.schedule') + ' and ' + t('cron.command') + ' are required'
    return
  }
  submitting.value = true
  formError.value = null
  try {
    const job = await addCronJob({
      name: formName.value.trim() || undefined,
      schedule: formSchedule.value.trim(),
      command: formCommand.value.trim(),
    })
    jobs.value = [...jobs.value, job]
    closeForm()
  } catch (err) {
    formError.value = err instanceof Error ? err.message : t('common.error')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: string) => {
  try {
    await deleteCronJob(id)
    jobs.value = jobs.value.filter((j) => j.id !== id)
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('common.error')
  } finally {
    confirmDelete.value = null
  }
}

const fetchJobs = () => {
  loading.value = true
  getCronJobs()
    .then((data) => {
      jobs.value = data
    })
    .catch((err) => {
      error.value = err instanceof Error ? err.message : t('common.error')
    })
    .finally(() => {
      loading.value = false
    })
}

onMounted(() => {
  fetchJobs()
})
</script>
