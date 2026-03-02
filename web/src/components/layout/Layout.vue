<template>
  <div class="min-h-screen bg-gray-950 text-white">
    <Sidebar />

    <div class="ml-60 flex flex-col min-h-screen">
      <Header />

      <main class="flex-1 overflow-y-auto">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import Sidebar from './Sidebar.vue'
import Header from './Header.vue'
</script>
