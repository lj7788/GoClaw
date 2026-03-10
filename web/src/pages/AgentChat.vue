<template>
  <div class="flex h-[calc(100vh-3.5rem)]">
    <div v-if="error"
      class="absolute top-0 left-0 right-0 px-4 py-2 bg-red-900/30 border-b border-red-700 flex items-center gap-2 text-sm text-red-300 z-10">
      <AlertCircle class="h-4 w-4 flex-shrink-0" />
      {{ error }}
    </div>

    <SessionList
      ref="sessionListRef"
      :current-session-id="currentSessionId"
      @session-select="handleSessionSelect"
      @new-session="handleNewSession"
    />

    <div class="flex-1 flex flex-col">
      <div class="flex-1 overflow-y-auto p-4 space-y-4">
        <div v-if="messages.length === 0" class="flex flex-col items-center justify-center h-full text-gray-500">
          <Bot class="h-12 w-12 mb-3 text-gray-600" />
          <p class="text-lg font-medium">ZeroClaw 智能体</p>
          <p class="text-sm mt-1">发送消息开始对话</p>
        </div>

        <div v-for="msg in messages" :key="msg.id"
          :class="['flex items-start gap-3', msg.role === 'user' ? 'flex-row-reverse' : '']">
          <div :class="[
            'flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center',
            msg.role === 'user' ? 'bg-blue-600' : 'bg-gray-700'
          ]">
            <User v-if="msg.role === 'user'" class="h-4 w-4 text-white" />
            <Bot v-else class="h-4 w-4 text-white" />
          </div>
          <div :class="[
            'max-w-[75%] rounded-xl px-4 py-3',
            msg.role === 'user'
              ? 'bg-blue-600 text-white'
              : 'bg-gray-800 text-gray-100 border border-gray-700'
          ]">
            <p class="text-sm whitespace-pre-wrap break-words">{{ msg.content }}</p>
            <audio v-if="msg.audioUrl" controls class="w-full max-w-[300px] mt-2" :src="msg.audioUrl">
              您的浏览器不支持音频播放
            </audio>
            <a v-if="msg.audioUrl" :href="msg.audioUrl" target="_blank" rel="noopener noreferrer"
              class="text-xs text-blue-400 hover:text-blue-300 mt-1 inline-block">
              下载音频
            </a>
            <p :class="['text-xs mt-1', msg.role === 'user' ? 'text-blue-200' : 'text-gray-500']">
              {{ formatTime(msg.timestamp) }}
            </p>
          </div>
        </div>

        <div v-if="typing" class="flex items-start gap-3">
          <div class="flex-shrink-0 w-8 h-8 rounded-full bg-gray-700 flex items-center justify-center">
            <Bot class="h-4 w-4 text-white" />
          </div>
          <div class="bg-gray-800 border border-gray-700 rounded-xl px-4 py-3">
            <div class="flex items-center gap-1">
              <span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0ms" />
              <span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 150ms" />
              <span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 300ms" />
            </div>
            <p class="text-xs text-gray-500 mt-1">正在输入...</p>
          </div>
        </div>

        <div ref="messagesEndRef" />
      </div>

      <div class="border-t border-gray-800 bg-gray-900 p-4">
        <div class="flex items-center gap-3">
          <div class="flex-1 relative">
            <textarea ref="inputRef" v-model="input" @keydown="handleKeyDown"
              :placeholder="connected ? '输入消息...' : '连接中...'" :disabled="!connected" rows="4"
              class="w-full bg-gray-800 border border-gray-700 rounded-xl px-4 py-3 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50 resize-none" />
          </div>
          <div>
            <div class="flex items-center justify-center mt-2 gap-2" style="position: relative;top: -20px;">
              <span :class="['inline-block h-2 w-2 rounded-full', connected ? 'bg-green-500' : 'bg-red-500']" />
              <span class="text-xs text-gray-500">
                {{ connected ? '已连接' : '已断开' }}
              </span>
            </div>
            <button @click="handleSend" :disabled="!connected || !input.trim()"
              class="flex-shrink-0 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded-xl p-3 transition-colors">
              <Send class="h-5 w-5" />
            </button>
          </div>

        </div>

      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted, onMounted, nextTick, watch } from 'vue'
import { Send, Bot, User, AlertCircle } from 'lucide-vue-next'
import type { WsMessage } from '../types/api'
import { WebSocketClient } from '../lib/ws'
import { useAuth } from '../hooks/useAuth'
import SessionList from '../components/SessionList.vue'

interface ChatMessage {
  id: string
  role: 'user' | 'agent'
  content: string
  timestamp: Date
  audioUrl?: string
}

interface Session {
  id: string
  title: string
  user_id?: string
  created_at: string
  updated_at: string
  message_count: number
  metadata?: Record<string, unknown>
}

interface StoredMessage {
  id: string
  session_id: string
  role: string
  content: string
  created_at: string
  metadata?: Record<string, unknown>
}

let fallbackMessageIdCounter = 0
const EMPTY_DONE_FALLBACK = '工具执行已完成，但未返回最终响应文本。'

function makeMessageId(): string {
  const uuid = globalThis.crypto?.randomUUID?.()
  if (uuid) return uuid

  fallbackMessageIdCounter += 1
  return `msg_${Date.now().toString(36)}_${fallbackMessageIdCounter.toString(36)}_${Math.random()
    .toString(36)
    .slice(2, 10)}`
}

const { loading } = useAuth()
const messages = ref<ChatMessage[]>([])
const input = ref('')
const typing = ref(false)
const connected = ref(false)
const error = ref<string | null>(null)
const currentSessionId = ref<string>('')
const sessionListRef = ref<{ loadSessions: () => void } | null>(null)

const wsRef = ref<WebSocketClient | null>(null)
const messagesEndRef = ref<HTMLDivElement | null>(null)
const inputRef = ref<HTMLTextAreaElement | null>(null)
const pendingContentRef = ref('')

const formatTime = (date: Date): string => {
  return date.toLocaleTimeString()
}

const scrollToBottom = async () => {
  await nextTick()
  messagesEndRef.value?.scrollIntoView({ behavior: 'smooth' })
}

watch([messages, typing], () => {
  scrollToBottom()
})

const loadSessionMessages = async (sessionId: string) => {
  try {
    const response = await fetch(`/api/sessions/${sessionId}/messages`)
    if (!response.ok) {
      throw new Error('Failed to load session messages')
    }
    const data = await response.json()
    const storedMessages: StoredMessage[] = data.messages || []
    
    messages.value = storedMessages.map((msg) => ({
      id: msg.id,
      role: msg.role as 'user' | 'agent',
      content: msg.content,
      timestamp: new Date(msg.created_at)
    }))
    
    currentSessionId.value = sessionId
  } catch (err) {
    console.error('Failed to load session messages:', err)
  }
}

const createNewSession = async () => {
  try {
    const response = await fetch('/api/sessions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ title: '' }), // 空标题会使用默认标题，然后会被更新
    })
    if (!response.ok) {
      throw new Error('Failed to create session')
    }
    const data = await response.json()
    const newSession: Session = data.session
    
    currentSessionId.value = newSession.id
    messages.value = []
    
    sessionListRef.value?.loadSessions()
  } catch (err) {
    console.error('Failed to create session:', err)
  }
}

const handleSessionSelect = async (session: Session) => {
  await loadSessionMessages(session.id)
}

const handleNewSession = async () => {
  await createNewSession()
}

const loadLatestSession = async () => {
  try {
    const response = await fetch('/api/sessions')
    if (!response.ok) {
      throw new Error('Failed to load sessions')
    }
    const data = await response.json()
    const sessions: Session[] = data.sessions || []
    
    // 获取今天的日期字符串 (YYYY-MM-DD)
    const today = new Date().toISOString().split('T')[0]
    
    // 筛选今天的会话（按 updated_at 判断）
    const todaySessions = sessions.filter((s) => (s.updated_at as string).startsWith(today))
    
    if (todaySessions.length > 0) {
      // 加载今天最新的会话
      await loadSessionMessages(todaySessions[0].id as string)
    } else {
      // 今天没有会话，新建一个
      await createNewSession()
    }
  } catch (err) {
    console.error('Failed to load latest session:', err)
    await createNewSession()
  }
}

defineExpose({
  handleSessionSelect,
  handleNewSession
})

// Connect WebSocket when auth state becomes ready
watch(loading, (newLoading) => {
  if (!newLoading && !wsRef.value?.connected) {
    connectWebSocket()
  }
})

// Try to connect on page mount
onMounted(() => {
  setTimeout(() => {
    if (!wsRef.value?.connected) {
      connectWebSocket()
    }
  }, 1000)
  
  setTimeout(() => {
    if (!currentSessionId.value) {
      loadLatestSession()
    }
  }, 500)
})

const connectWebSocket = () => {
  if (wsRef.value?.connected) return

  const ws = new WebSocketClient()

  ws.onOpen = () => {
    connected.value = true
    error.value = null
  }

  ws.onClose = () => {
    connected.value = false
  }

  ws.onError = () => {
    error.value = '连接错误。正在尝试重新连接...'
  }

  ws.onMessage = (msg: WsMessage) => {
    switch (msg.type) {
      case 'chunk':
        typing.value = true
        pendingContentRef.value += msg.content ?? ''
        break

      case 'message':
      case 'done': {
        const content = (msg.full_response ?? msg.content ?? pendingContentRef.value ?? '').trim()
        const finalContent = content || EMPTY_DONE_FALLBACK

        messages.value = [
          ...messages.value,
          {
            id: makeMessageId(),
            role: 'agent',
            content: finalContent,
            timestamp: new Date()
          }
        ]

        pendingContentRef.value = ''
        typing.value = false
        
        // Refresh session list to show updated message count
        sessionListRef.value?.loadSessions()
        break
      }

      case 'tool_call':
        messages.value = [
          ...messages.value,
          {
            id: makeMessageId(),
            role: 'agent',
            content: `[Tool Call] ${msg.name ?? 'unknown'}(${JSON.stringify(msg.args ?? {})})`,
            timestamp: new Date()
          }
        ]
        // Refresh session list to show updated message count
        sessionListRef.value?.loadSessions()
        break

      case 'tool_result': {
        const output = msg.output ?? ''
        let audioUrl = null
        try {
          const parsed = JSON.parse(output)
          if (parsed.type === 'audio' && parsed.url) {
            audioUrl = parsed.url
          }
        } catch {
          // Not JSON, treat as plain text
        }

        if (audioUrl) {
          messages.value = [
            ...messages.value,
            {
              id: makeMessageId(),
              role: 'agent',
              content: '🎵 语音已生成',
              audioUrl,
              timestamp: new Date()
            }
          ]
        } else {
          messages.value = [
            ...messages.value,
            {
              id: makeMessageId(),
              role: 'agent',
              content: `[Tool Result] ${output}`,
              timestamp: new Date()
            }
          ]
        }
        // Refresh session list to show updated message count
        sessionListRef.value?.loadSessions()
        break
      }

      case 'error':
        messages.value = [
          ...messages.value,
          {
            id: makeMessageId(),
            role: 'agent',
            content: `[错误] ${msg.message ?? '未知错误'}`,
            timestamp: new Date()
          }
        ]
        typing.value = false
        pendingContentRef.value = ''
        // Refresh session list to show updated message count
        sessionListRef.value?.loadSessions()
        break
    }
  }

  ws.connect()
  wsRef.value = ws
}

const handleSend = () => {
  const trimmed = input.value.trim()
  if (!trimmed || !wsRef.value?.connected) return

  messages.value = [
    ...messages.value,
    {
      id: makeMessageId(),
      role: 'user',
      content: trimmed,
      timestamp: new Date()
    }
  ]

  try {
    wsRef.value.sendMessage(trimmed, currentSessionId.value)
    typing.value = true
    pendingContentRef.value = ''
  } catch {
    error.value = '发送消息失败。请重试。'
  }

  input.value = ''
  inputRef.value?.focus()
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    handleSend()
  }
}


onUnmounted(() => {
  wsRef.value?.disconnect()
})
</script>
