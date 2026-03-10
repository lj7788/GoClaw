<template>
  <div class="flex flex-col h-full bg-gray-900 border-r border-gray-800">
    <div class="p-4 border-b border-gray-800">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-lg font-semibold text-white">会话历史</h2>
        <button
          @click="handleNewSession"
          class="flex items-center gap-2 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded-lg transition-colors"
        >
          <Plus class="h-4 w-4" />
          新建会话
        </button>
      </div>
      <div class="relative">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-500" />
        <input
          v-model="searchQuery"
          @input="handleSearch"
          type="text"
          placeholder="搜索会话..."
          class="w-full pl-10 pr-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>
    </div>

    <div class="flex-1 overflow-y-auto p-2 space-y-1">
      <div v-if="loading" class="flex items-center justify-center py-8">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
      </div>

      <div v-else-if="sessions.length === 0" class="flex flex-col items-center justify-center py-8 text-gray-500">
        <MessageSquare class="h-12 w-12 mb-2" />
        <p class="text-sm">暂无会话记录</p>
      </div>

      <div v-else>
        <div
          v-for="session in filteredSessions"
          :key="session.id"
          @click="handleSessionClick(session)"
          :class="[
            'flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors group',
            currentSessionId === session.id
              ? 'bg-blue-600/20 border border-blue-600/50'
              : 'hover:bg-gray-800 border border-transparent'
          ]"
        >
          <MessageSquare
            :class="[
              'h-5 w-5 flex-shrink-0',
              currentSessionId === session.id ? 'text-blue-500' : 'text-gray-500'
            ]"
          />
          <div class="flex-1 min-w-0">
            <h3
              :class="[
                'text-sm font-medium truncate',
                currentSessionId === session.id ? 'text-blue-400' : 'text-gray-200'
              ]"
            >
              {{ session.title }}
            </h3>
            <p class="text-xs text-gray-500 mt-0.5">
              {{ formatDate(session.updated_at) }} · {{ session.message_count }} 条消息
            </p>
          </div>
          <button
            @click.stop="handleDeleteSession(session)"
            class="opacity-0 group-hover:opacity-100 p-1.5 text-gray-500 hover:text-red-400 hover:bg-red-900/20 rounded transition-all"
          >
            <Trash2 class="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>

    <ConfirmDialog
      v-model="showConfirm"
      :title="confirmTitle"
      :message="confirmMessage"
      confirm-text="删除"
      type="danger"
      @confirm="confirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Plus, Search, MessageSquare, Trash2 } from 'lucide-vue-next'
import ConfirmDialog from './ConfirmDialog.vue'

interface Session {
  id: string
  title: string
  user_id?: string
  created_at: string
  updated_at: string
  message_count: number
  metadata?: Record<string, unknown>
}

const props = defineProps<{
  currentSessionId?: string
}>()

const emit = defineEmits<{
  'session-select': [session: Session]
  'new-session': []
}>()

const sessions = ref<Session[]>([])
const loading = ref(false)
const searchQuery = ref('')
const showConfirm = ref(false)
const confirmTitle = ref('确认删除')
const confirmMessage = ref('')
const deletingSession = ref<Session | null>(null)

const filteredSessions = computed(() => {
  if (!searchQuery.value) {
    return sessions.value
  }
  const query = searchQuery.value.toLowerCase()
  return sessions.value.filter(
    (session) =>
      session.title.toLowerCase().includes(query)
  )
})

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) {
    return '今天'
  } else if (days === 1) {
    return '昨天'
  } else if (days < 7) {
    return `${days} 天前`
  } else {
    return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
  }
}

const loadSessions = async () => {
  loading.value = true
  try {
    const response = await fetch('/api/sessions')
    if (!response.ok) {
      throw new Error('Failed to load sessions')
    }
    const data = await response.json()
    sessions.value = data.sessions || []
  } catch (error) {
    console.error('Failed to load sessions:', error)
  } finally {
    loading.value = false
  }
}

const handleSessionClick = (session: Session) => {
  emit('session-select', session)
}

const handleNewSession = () => {
  emit('new-session')
}

const handleDeleteSession = (session: Session) => {
  deletingSession.value = session
  confirmTitle.value = '确认删除'
  confirmMessage.value = `确定要删除会话"${session.title}"吗？此操作不可恢复。`
  showConfirm.value = true
}

const confirmDelete = async () => {
  if (!deletingSession.value) return
  
  try {
    const response = await fetch(`/api/sessions/${deletingSession.value.id}`, {
      method: 'DELETE',
    })
    if (!response.ok) {
      throw new Error('Failed to delete session')
    }
    sessions.value = sessions.value.filter((s) => s.id !== deletingSession.value?.id)
    if (props.currentSessionId === deletingSession.value.id) {
      emit('new-session')
    }
  } catch (error) {
    console.error('Failed to delete session:', error)
    alert('删除会话失败')
  } finally {
    deletingSession.value = null
  }
}

const handleSearch = () => {
}

onMounted(() => {
  loadSessions()
})

defineExpose({
  loadSessions
})
</script>
