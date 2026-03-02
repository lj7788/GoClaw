import { ref, onMounted, onUnmounted } from 'vue'
import { getToken, setToken, clearToken, isAuthenticated as checkAuth } from '../lib/auth'
import { pair as apiPair, getPublicHealth } from '../lib/api'

export function useAuth() {
  const token = ref<string | null>(getToken())
  const authenticated = ref<boolean>(checkAuth())
  const loading = ref<boolean>(!checkAuth())

  const pair = async (code: string): Promise<void> => {
    const { token: newToken } = await apiPair(code)
    setToken(newToken)
    token.value = newToken
    authenticated.value = true
  }

  const logout = (): void => {
    clearToken()
    token.value = null
    authenticated.value = false
  }

  onMounted(() => {
    if (checkAuth()) return

    let cancelled = false
    getPublicHealth()
      .then((health: { require_pairing: boolean; paired: boolean }) => {
        if (cancelled) return
        if (!health.require_pairing || health.paired) {
          authenticated.value = true
        }
      })
      .catch(() => {
        // health endpoint unreachable — fall back to showing pairing dialog
      })
      .finally(() => {
        if (!cancelled) loading.value = false
      })

    const handler = (e: StorageEvent) => {
      if (e.key === 'zeroclaw_token') {
        const t = getToken()
        token.value = t
        authenticated.value = t !== null && t.length > 0
      }
    }

    window.addEventListener('storage', handler)

    onUnmounted(() => {
      cancelled = true
      window.removeEventListener('storage', handler)
    })
  })

  return {
    token,
    isAuthenticated: authenticated,
    loading,
    pair,
    logout
  }
}
