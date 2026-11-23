import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Where the browser finds the gateway is decided at BUILD time:
// src/lib/backend.ts and src/contexts/RealtimeContext.tsx read
// import.meta.env.VITE_GATEWAY_URL. Unset means "same origin" (the dev proxy
// below, or nginx in the Docker image).
//
// On Render the static site is built with GATEWAY_HOST set to the gateway's
// public hostname (render.yaml), so turn that into https://<host> unless
// VITE_GATEWAY_URL was given explicitly. Vite reads VITE_* variables from
// process.env after this file runs, so setting it here is enough.
const gatewayHost = process.env.GATEWAY_HOST?.trim()
if (!process.env.VITE_GATEWAY_URL && gatewayHost) {
  process.env.VITE_GATEWAY_URL = /^https?:\/\//.test(gatewayHost) ? gatewayHost : `https://${gatewayHost}`
}

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
