import { PersistQueryClientProvider } from "@tanstack/react-query-persist-client"
import { lazy, StrictMode, Suspense, type ReactNode } from "react"
import { createRoot } from "react-dom/client"
import { App } from "./App"
import "./index.css"
// Applies the saved theme and follows the device setting.
import "./lib/theme"
import { persistOptions, queryClient } from "./lib/query"

const root = document.getElementById("root")
if (!root) throw new Error("Missing #root element")

let page: ReactNode = <App />
// The design kit (every component with placeholder data) is served in
// development only, at /?kit; production builds drop it.
if (import.meta.env.DEV && new URLSearchParams(window.location.search).has("kit")) {
  const KitPage = lazy(() => import("./kit/KitPage"))
  page = (
    <Suspense fallback={null}>
      <KitPage />
    </Suspense>
  )
}

createRoot(root).render(
  <StrictMode>
    <PersistQueryClientProvider client={queryClient} persistOptions={persistOptions}>
      {page}
    </PersistQueryClientProvider>
  </StrictMode>,
)
