import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    port: 5173,
    host: '0.0.0.0',
    allowedHosts: ['7xt72525ja38.vicp.fun'],
    proxy: {
      '/api': {
        target: 'http://localhost:4096',
        changeOrigin: true,
        secure: false,
        ws: true,
      },
    },
  },
})
