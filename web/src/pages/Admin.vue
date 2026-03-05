<template>
  <div class="max-w-7xl mx-auto px-4 py-8">
    <div class="bg-white shadow rounded-lg overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
        <h1 class="text-2xl font-bold text-gray-900">管理员管理</h1>
        <button @click="activeTab = 'password'" class="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-500">
          修改密码
        </button>
      </div>
      
      <div class="px-6 py-8">
        <div v-if="activeTab === 'users'">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">用户审核</h2>
          
          <div v-if="loading" class="flex justify-center py-12">
            <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
          </div>
          
          <div v-else-if="users.length > 0" class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200">
              <thead class="bg-gray-50">
                <tr>
                  <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    用户名
                  </th>
                  <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    邮箱
                  </th>
                  <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    状态
                  </th>
                  <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    注册时间
                  </th>
                  <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    操作
                  </th>
                </tr>
              </thead>
              <tbody class="bg-white divide-y divide-gray-200">
                <tr v-for="user in users" :key="user.id">
                  <td class="px-6 py-4 whitespace-nowrap">
                    <div class="flex items-center">
                      <div class="flex-shrink-0 h-10 w-10">
                        <img :src="user.avatar || 'https://via.placeholder.com/40'" alt="" class="h-10 w-10 rounded-full">
                      </div>
                      <div class="ml-4">
                        <div class="text-sm font-medium text-gray-900">{{ user.nickname }}</div>
                      </div>
                    </div>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <div class="text-sm text-gray-900">{{ user.email || '未设置' }}</div>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <span :class="getUserStatusClass(user.status)" class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full">
                      {{ getUserStatus(user.status) }}
                    </span>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {{ formatDate(user.created_at) }}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    <button 
                      v-if="user.status === 0" 
                      @click="approveUser(user.id, 1)"
                      class="text-green-600 hover:text-green-900 mr-3"
                    >
                      通过
                    </button>
                    <button 
                      v-if="user.status === 0" 
                      @click="approveUser(user.id, 2)"
                      class="text-red-600 hover:text-red-900"
                    >
                      拒绝
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          
          <div v-else class="text-center py-12">
            <p class="text-gray-600">暂无用户</p>
          </div>
        </div>
        
        <div v-if="activeTab === 'password'">
          <h2 class="text-lg font-semibold text-gray-900 mb-4">修改密码</h2>
          <form @submit.prevent="changePassword" class="space-y-4 max-w-md">
            <div>
              <label for="old_password" class="block text-sm font-medium text-gray-700 mb-1">旧密码</label>
              <input 
                type="password" 
                id="old_password" 
                v-model="passwordForm.old_password" 
                required
                class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="请输入旧密码"
              />
            </div>
            <div>
              <label for="new_password" class="block text-sm font-medium text-gray-700 mb-1">新密码</label>
              <input 
                type="password" 
                id="new_password" 
                v-model="passwordForm.new_password" 
                required
                class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="请输入新密码"
              />
            </div>
            <div class="flex space-x-4">
              <button 
                type="submit" 
                :disabled="submitting"
                class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
              >
                {{ submitting ? '修改中...' : '修改密码' }}
              </button>
              <button 
                type="button" 
                @click="activeTab = 'users'"
                class="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-500"
              >
                取消
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAdminUsers, approveUser as apiApproveUser, changeAdminPassword } from '../lib/api'

const activeTab = ref('users')
const loading = ref(true)
const submitting = ref(false)
const users = ref<any[]>([])
const passwordForm = ref({
  old_password: '',
  new_password: ''
})

const fetchUsers = async () => {
  loading.value = true
  try {
    const response = await getAdminUsers()
    users.value = response.users
  } catch (error) {
    console.error('获取用户列表失败:', error)
  } finally {
    loading.value = false
  }
}

const approveUser = async (userId: number, status: number) => {
  try {
    await apiApproveUser({ user_id: userId, status })
    fetchUsers()
  } catch (error: any) {
    alert('操作失败: ' + (error.message || '未知错误'))
  }
}

const changePassword = async () => {
  submitting.value = true
  try {
    const response = await changeAdminPassword(passwordForm.value)
    alert(response.message)
    passwordForm.value = {
      old_password: '',
      new_password: ''
    }
    activeTab.value = 'users'
  } catch (error: any) {
    alert('修改失败: ' + (error.message || '未知错误'))
  } finally {
    submitting.value = false
  }
}

const getUserStatus = (status: number) => {
  switch (status) {
    case 0: return '待审核'
    case 1: return '已通过'
    case 2: return '已拒绝'
    default: return '未知'
  }
}

const getUserStatusClass = (status: number) => {
  switch (status) {
    case 0: return 'bg-yellow-100 text-yellow-800'
    case 1: return 'bg-green-100 text-green-800'
    case 2: return 'bg-red-100 text-red-800'
    default: return 'bg-gray-100 text-gray-800'
  }
}

const formatDate = (dateString: string) => {
  const date = new Date(dateString)
  return date.toLocaleString()
}

onMounted(() => {
  fetchUsers()
})
</script>

<style scoped>
/* 自定义样式 */
</style>