import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/aleph.v1': 'http://localhost:8080',
      '/aleph.nlp.v1': 'http://localhost:8080',
      '/aleph.registry.v1': 'http://localhost:8080',
      '/aleph.tool.v1': 'http://localhost:8080',
    },
  },
  build: {
    chunkSizeWarningLimit: 150,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/react-dom') || id.includes('node_modules/react/') || id.includes('node_modules/zustand') || id.includes('node_modules/lucide-react')) {
            return 'vendor'
          }
          if (id.includes('node_modules/@connectrpc') || id.includes('node_modules/@bufbuild')) {
            return 'connectrpc'
          }
          if (id.includes('node_modules/d3') || id.includes('node_modules/react-leaflet') || id.includes('node_modules/leaflet')) {
            return 'd3'
          }
        },
      },
    },
  },
})
