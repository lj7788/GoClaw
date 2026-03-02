<template>
  <div class="flex flex-col h-[calc(100vh-3.5rem)]">
    <!-- Toolbar -->
    <div class="flex items-center justify-between px-6 py-3 border-b border-gray-800 bg-gray-900">
      <div class="flex items-center gap-3">
        <Activity class="h-5 w-5 text-blue-400" />
        <h2 class="text-base font-semibold text-white">{{ t('logs.title') }}</h2>
        <div class="flex items-center gap-2 ml-2">
          <span :class="['inline-block h-2 w-2 rounded-full', connected ? 'bg-green-500' : 'bg-red-500']" />
          <span class="text-xs text-gray-500">
            {{ connected ? t('logs.connected') : t('logs.disconnected') }}
          </span>
        </div>
        <span class="text-xs text-gray-500 ml-2">
          {{ filteredEntries.length }} {{ t('common.actions') }}
        </span>
      </div>

      <div class="flex items-center gap-2">
        <!-- Pause/Resume -->
        <button
          @click="paused = !paused"
          :class="[
            'flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
            paused ? 'bg-green-600 hover:bg-green-700 text-white' : 'bg-yellow-600 hover:bg-yellow-700 text-white'
          ]"
        >
          <Play v-if="paused" class="h-3.5 w-3.5" />
          <Pause v-else class="h-3.5 w-3.5" />
          {{ paused ? t('logs.resume') : t('logs.pause') }}
        </button>

        <!-- Jump to Bottom -->
        <button
          v-if="!autoScroll"
          @click="jumpToBottom"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium bg-blue-600 hover:bg-blue-700 text-white transition-colors"
        >
          <ArrowDown class="h-3.5 w-3.5" />
          {{ t('logs.title') }}
        </button>
      </div>
    </div>

    <!-- Event type filters -->
    <div v-if="allTypes.length > 0" class="flex items-center gap-2 px-6 py-2 border-b border-gray-800 bg-gray-900/80 overflow-x-auto">
      <Filter class="h-4 w-4 text-gray-500 flex-shrink-0" />
      <span class="text-xs text-gray-500 flex-shrink-0">Filter:</span>
      <label
        v-for="type in allTypes"
        :key="type"
        class="flex items-center gap-1.5 cursor-pointer flex-shrink-0"
      >
        <input
          type="checkbox"
          :checked="typeFilters.has(type)"
          @change="toggleTypeFilter(type)"
          class="rounded bg-gray-800 border-gray-600 text-blue-500 focus:ring-blue-500 focus:ring-offset-0 h-3.5 w-3.5"
        />
        <span class="text-xs text-gray-400 capitalize">{{ type }}</span>
      </label>
      <button
        v-if="typeFilters.size > 0"
        @click="typeFilters.clear()"
        class="text-xs text-blue-400 hover:text-blue-300 flex-shrink-0 ml-1"
      >
        {{ t('common.clear') || 'Clear' }}
      </button>
    </div>

    <!-- Log entries -->
    <div
      ref="containerRef"
      @scroll="handleScroll"
      class="flex-1 overflow-y-auto p-4 space-y-2"
    >
      <div v-if="filteredEntries.length === 0" class="flex flex-col items-center justify-center h-full text-gray-500">
        <Activity class="h-10 w-10 text-gray-600 mb-3" />
        <p class="text-sm">
          {{ paused ? t('logs.pause') : t('logs.empty') }}
        </p>
      </div>
      <div
        v-else
        v-for="entry in filteredEntries"
        :key="entry.id"
        class="bg-gray-900 border border-gray-800 rounded-lg p-3 hover:border-gray-700 transition-colors"
      >
        <div class="flex items-start gap-3">
          <span class="text-xs text-gray-500 font-mono whitespace-nowrap mt-0.5">
            {{ formatTimestamp(entry.event.timestamp) }}
          </span>
          <span
            :class="['inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border capitalize flex-shrink-0', eventTypeBadgeColor(entry.event.type)]"
          >
            {{ entry.event.type }}
          </span>
          <p class="text-sm text-gray-300 break-all min-w-0">
            {{ formatDetail(entry.event) }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import {
  Activity,
  Pause,
  Play,
  ArrowDown,
  Filter,
} from 'lucide-vue-next'
import type { SSEEvent } from '../types/api'
import { SSEClient } from '../lib/sse'
import { useI18n } from '../lib/i18n'

const { t } = useI18n()

interface LogEntry {
  id: string
  event: SSEEvent
}

const entries = ref<LogEntry[]>([])
const paused = ref(false)
const connected = ref(false)
const autoScroll = ref(true)
const typeFilters = ref(new Set<string>())

const containerRef = ref<HTMLElement | null>(null)
const sseRef = ref<SSEClient | null>(null)
const entryIdRef = ref(0)

function formatTimestamp(ts?: string): string {
  if (!ts) return new Date().toLocaleTimeString()
  return new Date(ts).toLocaleTimeString()
}

function formatDetail(event: SSEEvent): string {
  return event.message ?? event.content ?? event.data ?? JSON.stringify(
    Object.fromEntries(
      Object.entries(event).filter(([k]) => k !== 'type' && k !== 'timestamp')
    )
  )
}

function eventTypeBadgeColor(type: string): string {
  switch (type.toLowerCase()) {
    case 'error':
      return 'bg-red-900/50 text-red-400 border-red-700/50'
    case 'warn':
    case 'warning':
      return 'bg-yellow-900/50 text-yellow-400 border-yellow-700/50'
    case 'tool_call':
    case 'tool_result':
      return 'bg-purple-900/50 text-purple-400 border-purple-700/50'
    case 'message':
    case 'chat':
      return 'bg-blue-900/50 text-blue-400 border-blue-700/50'
    case 'health':
    case 'status':
      return 'bg-green-900/50 text-green-400 border-green-700/50'
    default:
      return 'bg-gray-800 text-gray-400 border-gray-700'
  }
}

const allTypes = computed(() => {
  return Array.from(new Set(entries.value.map((e) => e.event.type))).sort()
})

const filteredEntries = computed(() => {
  if (typeFilters.value.size === 0) return entries.value
  return entries.value.filter((e) => typeFilters.value.has(e.event.type))
})

function toggleTypeFilter(type: string) {
  if (typeFilters.value.has(type)) {
    typeFilters.value.delete(type)
  } else {
    typeFilters.value.add(type)
  }
}

function jumpToBottom() {
  if (containerRef.value) {
    containerRef.value.scrollTop = containerRef.value.scrollHeight
  }
  autoScroll.value = true
}

function handleScroll() {
  if (!containerRef.value) return
  const { scrollTop, scrollHeight, clientHeight } = containerRef.value
  const isAtBottom = scrollHeight - scrollTop - clientHeight < 50
  autoScroll.value = isAtBottom
}

// Auto-scroll to bottom
function updateScroll() {
  if (autoScroll.value && containerRef.value) {
    nextTick(() => {
      if (containerRef.value) {
        containerRef.value.scrollTop = containerRef.value.scrollHeight
      }
    })
  }
}

onMounted(() => {
  const client = new SSEClient()

  client.onConnect = () => {
    connected.value = true
  }

  client.onError = () => {
    connected.value = false
  }

  client.onEvent = (event: SSEEvent) => {
    if (paused.value) return
    entryIdRef.value += 1
    const entry: LogEntry = {
      id: `log-${entryIdRef.value}`,
      event,
    }
    entries.value = [...entries.value, entry]
    // Cap at 500 entries for performance
    if (entries.value.length > 500) {
      entries.value = entries.value.slice(-500)
    }
    updateScroll()
  }

  client.connect()
  sseRef.value = client
})

onUnmounted(() => {
  if (sseRef.value) {
    sseRef.value.disconnect()
  }
})
</script>
