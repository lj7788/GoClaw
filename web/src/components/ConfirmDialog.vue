<template>
  <Teleport to="body">
    <div v-if="modelValue" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 bg-black/60" @click="handleCancel"></div>

      <div class="relative bg-gray-800 rounded-xl p-6 w-full max-w-sm shadow-2xl border border-gray-700">
        <div class="text-center">
          <div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full mb-4" :class="iconBgClass">
            <component :is="icon" class="h-6 w-6" :class="iconClass" />
          </div>
          <h3 class="text-lg font-semibold text-white mb-2">{{ title }}</h3>
          <p class="text-sm text-gray-400 mb-6">{{ message }}</p>
          <div class="flex gap-3">
            <button @click="handleCancel"
              class="flex-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg text-sm font-medium transition-colors">
              {{ cancelText }}
            </button>
            <button @click="handleConfirm"
              class="flex-1 px-4 py-2 rounded-lg text-sm font-medium transition-colors" :class="confirmButtonClass">
              {{ confirmText }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Trash2, AlertTriangle, Info, CheckCircle } from 'lucide-vue-next'

interface Props {
  modelValue: boolean
  title?: string
  message?: string
  confirmText?: string
  cancelText?: string
  type?: 'danger' | 'warning' | 'info' | 'success'
}

const props = withDefaults(defineProps<Props>(), {
  title: '确认操作',
  message: '确定要执行此操作吗？',
  confirmText: '确定',
  cancelText: '取消',
  type: 'danger'
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  confirm: []
  cancel: []
}>()

const iconMap = {
  danger: Trash2,
  warning: AlertTriangle,
  info: Info,
  success: CheckCircle
}

const iconBgClassMap = {
  danger: 'bg-red-100',
  warning: 'bg-yellow-100',
  info: 'bg-blue-100',
  success: 'bg-green-100'
}

const iconClassMap = {
  danger: 'text-red-600',
  warning: 'text-yellow-600',
  info: 'text-blue-600',
  success: 'text-green-600'
}

const confirmButtonClassMap = {
  danger: 'bg-red-600 hover:bg-red-700 text-white',
  warning: 'bg-yellow-600 hover:bg-yellow-700 text-white',
  info: 'bg-blue-600 hover:bg-blue-700 text-white',
  success: 'bg-green-600 hover:bg-green-700 text-white'
}

const icon = computed(() => iconMap[props.type])
const iconBgClass = computed(() => iconBgClassMap[props.type])
const iconClass = computed(() => iconClassMap[props.type])
const confirmButtonClass = computed(() => confirmButtonClassMap[props.type])

const handleConfirm = () => {
  emit('confirm')
  emit('update:modelValue', false)
}

const handleCancel = () => {
  emit('cancel')
  emit('update:modelValue', false)
}
</script>
