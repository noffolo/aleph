import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  define: {
    'process.env.VITE_API_BASE_URL': JSON.stringify(process.env.VITE_API_BASE_URL || 'http://localhost:8080'),
  },
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
          // Leaflet maps — only loaded when a map view is rendered
          if (id.includes('node_modules/react-leaflet') || id.includes('node_modules/leaflet')) {
            return 'maps'
          }
          // D3 — split by visualization type
          if (id.includes('/d3-array') || id.includes('/d3-scale') || id.includes('/d3-shape') || id.includes('/d3-color') || id.includes('/d3-format') || id.includes('/d3-time') || id.includes('/d3-interpolate')) {
            return 'vendor'  // small, commonly used
          }
          if (id.includes('/d3-force') || id.includes('/d3-hierarchy') || id.includes('/d3-quadtree') || id.includes('/d3-delaunay')) {
            return 'd3-force'
          }
          if (id.includes('/d3-geo') || id.includes('/d3-tile')) {
            return 'd3-geo'
          }
        },
      },
    },
  },
})
