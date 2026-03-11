<template>
  <aside :class="[
    'h-screen bg-gray-900 flex flex-col border-r border-gray-800',
    collapsed ? 'w-16' : 'w-60'
  ]">
    <div :class="[
      'flex items-center gap-2 py-5 border-b border-gray-800',
      collapsed ? 'justify-center px-2' : 'px-5'
    ]">
      <div class="h-8 w-8 rounded-lg bg-blue-600 flex items-center justify-center text-white font-bold text-sm">
        GC
      </div>
      <span v-if="!collapsed" class="text-lg font-semibold text-white tracking-wide">
        GoClaw
      </span>
    </div>

    <nav class="flex-1 overflow-y-auto py-4 px-2 space-y-1">
      <router-link
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        :end="item.to === '/'"
        :title="collapsed ? t(item.labelKey) : undefined"
        :class="[
          'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
          collapsed ? 'justify-center' : '',
          isActive(item.to)
            ? 'bg-blue-600 text-white'
            : 'text-gray-300 hover:bg-gray-800 hover:text-white'
        ]"
      >
        <component :is="item.icon" class="h-5 w-5 flex-shrink-0" />
        <span v-if="!collapsed">{{ t(item.labelKey) }}</span>
      </router-link>
    </nav>

    <button
      @click="$emit('toggle')"
      class="flex items-center justify-center p-3 border-t border-gray-800 text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
    >
      <ChevronLeft
        :class="[
          'h-5 w-5 transition-transform duration-300',
          collapsed ? 'rotate-180' : ''
        ]"
      />
    </button>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from '../../lib/i18n'
import { useRoute } from 'vue-router'
import {
  LayoutDashboard,
  MessageSquare,
  Wrench,
  Clock,
  Puzzle,
  Brain,
  Settings,
  DollarSign,
  Activity,
  Stethoscope,
  ChevronLeft
} from 'lucide-vue-next'

defineProps<{
  collapsed: boolean
}>()

defineEmits<{
  'toggle': []
}>()

const route = useRoute()
const { t } = useI18n()

const navItems = [
  { to: '/', icon: LayoutDashboard, labelKey: 'nav.dashboard' },
  { to: '/agent', icon: MessageSquare, labelKey: 'nav.agent' },
  { to: '/tools', icon: Wrench, labelKey: 'nav.tools' },
  { to: '/cron', icon: Clock, labelKey: 'nav.cron' },
  { to: '/integrations', icon: Puzzle, labelKey: 'nav.integrations' },
  { to: '/memory', icon: Brain, labelKey: 'nav.memory' },
  { to: '/config', icon: Settings, labelKey: 'nav.config' },
  { to: '/cost', icon: DollarSign, labelKey: 'nav.cost' },
  { to: '/logs', icon: Activity, labelKey: 'nav.logs' },
  { to: '/doctor', icon: Stethoscope, labelKey: 'nav.doctor' }
]

const isActive = (path: string): boolean => {
  if (path === '/') {
    return route.path === '/'
  }
  return route.path.startsWith(path)
}
</script>
