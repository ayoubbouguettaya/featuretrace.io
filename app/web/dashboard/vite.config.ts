import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    // `npm run dev` talks to a query API on the host (`make run-query`).
    proxy: {
      '/v1': process.env.QUERY_API_URL ?? 'http://localhost:3008',
    },
  },
})
