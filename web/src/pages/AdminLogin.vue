<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center p-4">
    <div class="max-w-md w-full space-y-8">
      <div class="text-center">
        <h2 class="mt-6 text-3xl font-extrabold text-gray-900">
          管理员登录
        </h2>
        <p class="mt-2 text-sm text-gray-600">
          请输入管理员账号和密码
        </p>
      </div>

      <div class="bg-white p-8 rounded-lg shadow-md">
        <form @submit.prevent="login" class="space-y-6">
          <div>
            <label for="username" class="block text-sm font-medium text-gray-700 mb-1">用户名</label>
            <input 
              type="text" 
              id="username" 
              v-model="formData.username" 
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              placeholder="请输入用户名"
            />
          </div>
          <div>
            <label for="password" class="block text-sm font-medium text-gray-700 mb-1">密码</label>
            <input 
              type="password" 
              id="password" 
              v-model="formData.password" 
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              placeholder="请输入密码"
            />
          </div>
          <div>
            <button 
              type="submit" 
              :disabled="loading"
              class="w-full py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
            >
              {{ loading ? '登录中...' : '登录' }}
            </button>
          </div>
        </form>
      </div>

      <div v-if="error" class="bg-red-50 border border-red-200 rounded-md p-4">
        <div class="flex">
          <div class="flex-shrink-0">
            <svg class="h-5 w-5 text-red-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
            </svg>
          </div>
          <div class="ml-3">
            <h3 class="text-sm font-medium text-red-800">{{ error }}</h3>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { adminLogin } from '../lib/api'
import { setToken } from '../lib/auth'
import { useRouter } from 'vue-router'

const router = useRouter()
const loading = ref(false)
const error = ref<string>('')
const formData = ref({
  username: '',
  password: ''
})

const login = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await adminLogin(formData.value)
    setToken(response.token)
    router.push('/admin')
  } catch (err: any) {
    error.value = err.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* 自定义样式 */
</style>