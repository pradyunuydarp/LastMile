import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
      // Real-time offers, trip rooms and live locations ride socket.io.
      '/socket.io': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        ws: true,
      },
    },
  },
})
