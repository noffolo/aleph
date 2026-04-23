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
})
