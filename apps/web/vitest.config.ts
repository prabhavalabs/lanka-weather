import path from "node:path"
import { defineConfig } from "vitest/config"

export default defineConfig({
  define: { __BUILD_ID__: JSON.stringify("test") },
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "src") },
  },
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
    // The contrast test reads the theme stylesheet as text.
    css: { include: [/theme\.css/] },
  },
})
