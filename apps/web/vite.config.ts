import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import path from "node:path"
import { defineConfig } from "vite"
import { VitePWA } from "vite-plugin-pwa"

const apiTarget = process.env.LANKAWEATHER_API_URL ?? "http://localhost:8080"
const proxy = {
  "/v1": apiTarget,
  "/api": apiTarget,
  "/healthz": apiTarget,
}

/**
 * The service worker is built in production. The dev server runs without
 * one unless VITE_PWA_DEV=1, so hot reload is never answered from a cache.
 */
const pwaInDev = process.env.VITE_PWA_DEV === "1"

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      strategies: "injectManifest",
      srcDir: "src",
      filename: "sw.ts",
      // The app registers the worker itself (features/pwa/register.ts) and asks before updating.
      injectRegister: false,
      registerType: "prompt",
      // public/site.webmanifest is the one manifest, linked from index.html (ADR 0017).
      manifest: false,
      injectManifest: {
        // The app shell: scripts (MapLibre's worker is .mjs), styles, fonts and icons.
        globPatterns: ["**/*.{js,mjs,css,html,svg,png,ico,woff2,webmanifest}"],
        globIgnores: [
          // Install sheets only.
          "screenshots/**",
          // Font subsets the interface never shows (it is in English, Sinhala and Tamil);
          // the browser still fetches one if a page needs it.
          "assets/*-cyrillic-*.woff2",
          "assets/*-vietnamese-*.woff2",
        ],
        // MapLibre's chunk is about 1.1 MB.
        maximumFileSizeToCacheInBytes: 3 * 1024 * 1024,
      },
      devOptions: { enabled: pwaInDev, type: "module", navigateFallback: "index.html" },
    }),
  ],
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
  server: { proxy },
  // `pnpm preview` serves the production build, service worker included, against the same API.
  preview: { proxy },
})
