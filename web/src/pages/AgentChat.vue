<template>
  <div class="flex flex-col h-[calc(100vh-3.5rem)]">
    <div v-if="error" class="px-4 py-2 bg-red-900/30 border-b border-red-700 flex items-center gap-2 text-sm text-red-300">
      <AlertCircle class="h-4 w-4 flex-shrink-0" />
      {{ error }}
    </div>

    <div class="flex-1 overflow-y-auto p-4 space-y-4">
      <div v-if="messages.length === 0" class="flex flex-col items-center justify-center h-full text-gray-500">
        <Bot class="h-12 w-12 mb-3 text-gray-600" />
        <p class="text-lg font-medium">ZeroClaw 智能体</p>
        <p class="text-sm mt-1">发送消息开始对话</p>
      </div>

      <div
        v-for="msg in messages"
        :key="msg.id"
        :class="['flex items-start gap-3', msg.role === 'user' ? 'flex-row-reverse' : '']"
      >
        <div
          :class="[
            'flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center',
            msg.role === 'user' ? 'bg-blue-600' : 'bg-gray-700'
          ]"
        >
          <User v-if="msg.role === 'user'" class="h-4 w-4 text-white" />
          <Bot v-else class="h-4 w-4 text-white" />
        </div>
        <div
          :class="[
            'max-w-[75%] rounded-xl px-4 py-3',
            msg.role === 'user'
              ? 'bg-blue-600 text-white'
              : 'bg-gray-800 text-gray-100 border border-gray-700'
          ]"
        >
          <p class="text-sm whitespace-pre-wrap break-words">{{ msg.content }}</p>
          <audio v-if="msg.audioUrl" controls class="w-full max-w-[300px] mt-2" :src="msg.audioUrl">
            您的浏览器不支持音频播放
          </audio>
          <a
            v-if="msg.audioUrl"
            :href="msg.audioUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-xs text-blue-400 hover:text-blue-300 mt-1 inline-block"
          >
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
      <div class="flex items-center gap-3 max-w-4xl mx-auto">
        <div class="flex-1 relative">
          <textarea
            ref="inputRef"
            v-model="input"
            @keydown="handleKeyDown"
            :placeholder="connected ? '输入消息...' : '连接中...'"
            :disabled="!connected"
            rows="4"
            class="w-full bg-gray-800 border border-gray-700 rounded-xl px-4 py-3 text-sm text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:opacity-50 resize-none"
          />
        </div>
        <button
          @click="handleSend"
          :disabled="!connected || !input.trim()"
          class="flex-shrink-0 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded-xl p-3 transition-colors"
        >
          <Send class="h-5 w-5" />
        </button>
      </div>
      <div class="flex items-center justify-center mt-2 gap-2">
        <span
          :class="['inline-block h-2 w-2 rounded-full', connected ? 'bg-green-500' : 'bg-red-500']"
        />
        <span class="text-xs text-gray-500">
          {{ connected ? '已连接' : '已断开' }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted, nextTick, watch } from 'vue'
import { Send, Bot, User, AlertCircle } from 'lucide-vue-next'
import type { WsMessage } from '../types/api'
import { WebSocketClient } from '../lib/ws'
import { useAuth } from '../hooks/useAuth'

interface ChatMessage {
  id: string
  role: 'user' | 'agent'
  content: string
  timestamp: Date
  audioUrl?: string
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

const { isAuthenticated, loading } = useAuth()
const messages = ref<ChatMessage[]>([])
const input = ref('')
const typing = ref(false)
const connected = ref(false)
const error = ref<string | null>(null)

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

// Connect WebSocket when auth state becomes ready
watch([loading, isAuthenticated], ([newLoading, newAuth]) => {
  if (!newLoading && newAuth && !wsRef.value?.connected) {
    connectWebSocket()
  }
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
    wsRef.value.sendMessage(trimmed)
    typing.value = true
    pendingContentRef.value = ''
  } catch {
    error.value = '发送消息失败。请重试。'
  }

  input.value = ''
  inputRef.value?.focus()
}

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}


onUnmounted(() => {
  wsRef.value?.disconnect()
})
</script>
