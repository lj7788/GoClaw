<template>
  <div class="max-w-4xl mx-auto px-4 py-8">
    <div class="bg-white shadow rounded-lg overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200">
        <h1 class="text-2xl font-bold text-gray-900">用户中心</h1>
      </div>
      
      <div v-if="loading" class="px-6 py-12 flex justify-center">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
      
      <div v-else-if="user" class="px-6 py-8">
        <div class="mb-8">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">个人信息</h2>
          <div class="flex items-center space-x-6">
            <div class="w-24 h-24 rounded-full overflow-hidden border-2 border-gray-200">
              <img :src="user.avatar || 'https://via.placeholder.com/100'" alt="头像" class="w-full h-full object-cover" />
            </div>
            <div>
              <h3 class="text-xl font-medium text-gray-900">{{ user.nickname }}</h3>
              <p class="text-gray-600">{{ user.email || '未设置邮箱' }}</p>
              <p class="text-gray-500 text-sm">注册时间: {{ formatDate(user.created_at) }}</p>
              <p class="text-gray-500 text-sm">状态: {{ getUserStatus(user.status) }}</p>
            </div>
          </div>
        </div>
        
        <div class="mb-8">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">修改信息</h2>
          <form @submit.prevent="updateUserInfo" class="space-y-4">
            <div>
              <label for="email" class="block text-sm font-medium text-gray-700 mb-1">邮箱</label>
              <input 
                type="email" 
                id="email" 
                v-model="formData.email" 
                class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="请输入邮箱"
              />
            </div>
            <div>
              <label for="avatar" class="block text-sm font-medium text-gray-700 mb-1">头像URL</label>
              <input 
                type="text" 
                id="avatar" 
                v-model="formData.avatar" 
                class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="请输入头像URL"
              />
            </div>
            <div class="flex space-x-4">
              <button 
                type="submit" 
                :disabled="submitting"
                class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
              >
                {{ submitting ? '保存中...' : '保存修改' }}
              </button>
              <button 
                type="button" 
                @click="resetForm"
                class="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-500"
              >
                重置
              </button>
            </div>
          </form>
        </div>
      </div>
      
      <div v-else class="px-6 py-12 text-center">
        <p class="text-gray-600">获取用户信息失败</p>
        <button 
          @click="fetchUserInfo"
          class="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          重新获取
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getUserInfo, updateUserInfo as apiUpdateUserInfo } from '../lib/api'

const loading = ref(true)
const submitting = ref(false)
const user = ref<any>(null)
const formData = ref({
  email: '',
  avatar: ''
})

const fetchUserInfo = async () => {
  loading.value = true
  try {
    const response = await getUserInfo()
    user.value = response.user
    formData.value = {
      email: user.value.email || '',
      avatar: user.value.avatar || ''
    }
  } catch (error) {
    console.error('获取用户信息失败:', error)
  } finally {
    loading.value = false
  }
}

const updateUserInfo = async () => {
  submitting.value = true
  try {
    const response = await apiUpdateUserInfo(formData.value)
    user.value = response.user
    alert('信息更新成功')
  } catch (error: any) {
    alert('更新失败: ' + (error.message || '未知错误'))
  } finally {
    submitting.value = false
  }
}

const resetForm = () => {
  if (user.value) {
    formData.value = {
      email: user.value.email || '',
      avatar: user.value.avatar || ''
    }
  }
}

const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleString()
}

const getUserStatus = (status: number) => {
  switch (status) {
    case 0: return '待审核'
    case 1: return '已通过'
    case 2: return '已拒绝'
    default: return '未知'
  }
}

onMounted(() => {
  fetchUserInfo()
})
</script>

<style scoped>
/* 自定义样式 */
</style>