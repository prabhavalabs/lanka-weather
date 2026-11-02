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
    // MapLibre GL alone is about 1.1 MB minified and is not split further.
    chunkSizeWarningLimit: 1600,
    rolldownOptions: {
      output: {
        // MapLibre GL gets a chunk of its own: it changes only with upgrades, so browsers keep
        // it cached across app releases, and the app's own chunk stays small. Search, the
        // location panel, the wind renderer and the design kit load on demand (React.lazy).
        codeSplitting: {
          groups: [{ name: "maplibre", test: /[\\/]node_modules[\\/].*maplibre-gl[\\/]/ }],
        },
      },
    },
  },
  server: {
    proxy: {
      "/v1": apiTarget,
      "/healthz": apiTarget,
    },
  },
})
