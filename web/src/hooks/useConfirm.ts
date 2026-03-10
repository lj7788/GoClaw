import { ref } from 'vue'

export interface ConfirmOptions {
  title?: string
  message?: string
  confirmText?: string
  cancelText?: string
  type?: 'danger' | 'warning' | 'info' | 'success'
}

export function useConfirm() {
  const visible = ref(false)
  const title = ref('确认操作')
  const message = ref('确定要执行此操作吗？')
  const confirmText = ref('确定')
  const cancelText = ref('取消')
  const type = ref<'danger' | 'warning' | 'info' | 'success'>('danger')
  const resolvePromise = ref<((value: boolean) => void) | null>(null)

  const confirm = (options: ConfirmOptions = {}): Promise<boolean> => {
    title.value = options.title ?? '确认操作'
    message.value = options.message ?? '确定要执行此操作吗？'
    confirmText.value = options.confirmText ?? '确定'
    cancelText.value = options.cancelText ?? '取消'
    type.value = options.type ?? 'danger'
    visible.value = true

    return new Promise((resolve) => {
      resolvePromise.value = resolve
    })
  }

  const handleConfirm = () => {
    visible.value = false
    resolvePromise.value?.(true)
    resolvePromise.value = null
  }

  const handleCancel = () => {
    visible.value = false
    resolvePromise.value?.(false)
    resolvePromise.value = null
  }

  return {
    visible,
    title,
    message,
    confirmText,
    cancelText,
    type,
    confirm,
    handleConfirm,
    handleCancel
  }
}
