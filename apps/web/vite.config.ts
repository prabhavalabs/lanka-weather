import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import path from "node:path"
import { defineConfig } from "vite"

const apiTarget = process.env.LANKAWEATHER_API_URL ?? "http://localhost:8080"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "src") },
  },
  define: {
    __BUILD_ID__: JSON.stringify(process.env.GITHUB_SHA ?? new Date().toISOString()),
  },
  build: {
    // MapLibre GL alone is about 1 MB minified; it is the map, so it is not split further.
    chunkSizeWarningLimit: 1600,
  },
  server: {
    proxy: {
      "/v1": apiTarget,
      "/healthz": apiTarget,
    },
  },
})
