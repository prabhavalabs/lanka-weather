import { lazy, Suspense, useEffect, useState } from "react"
import { toast } from "sonner"
import { translate } from "@/lib/i18n"
import { withoutInstallParam } from "./links"
import { canInstall, currentPlatform } from "./platform"
import { getInstallState, openInstallGuide, useInstallState } from "./state"
import { changeRecord, recordVisit, shouldOffer } from "./visits"

const InstallGuide = lazy(() =>
  import("./InstallGuide").then((module) => ({ default: module.InstallGuide })),
)

/** Waits this long into a visit before suggesting an install, so the map comes first. */
const promptDelay = 25_000

function offerInstall() {
  const now = Date.now()
  changeRecord((record) => ({ ...record, offers: record.offers + 1, lastOffer: now }))
  toast(translate("install.prompt.title"), {
    id: "install-prompt",
    description: translate("install.prompt.body"),
    duration: 15_000,
    action: {
      label: translate("install.prompt.show"),
      onClick: () => openInstallGuide({ reason: "prompt" }),
    },
    cancel: {
      label: translate("install.prompt.later"),
      onClick: () => changeRecord((record) => ({ ...record, declinedAt: Date.now() })),
    },
  })
}

/**
 * Mounts the install guide and opens it when asked: from `/?install` (a link
 * sent over WhatsApp or SMS), from openInstallGuide(), or after a gentle
 * suggestion on a later visit.
 */
export function InstallHost() {
  const { open } = useInstallState()
  // Loaded the first time it opens, then kept, so closing can animate.
  const [loaded, setLoaded] = useState(open)
  if (open && !loaded) setLoaded(true)

  useEffect(() => {
    const cleaned = withoutInstallParam(window.location.href)
    if (cleaned !== null) {
      window.history.replaceState(window.history.state, "", cleaned)
      openInstallGuide({ reason: "link" })
    }

    const record = changeRecord((current) => recordVisit(current, Date.now()))
    const platform = currentPlatform()
    if (cleaned !== null || !canInstall(platform) || !shouldOffer(record, Date.now())) return
    const timer = window.setTimeout(() => {
      if (getInstallState().open || !navigator.onLine || document.visibilityState !== "visible") {
        return
      }
      offerInstall()
    }, promptDelay)
    return () => window.clearTimeout(timer)
  }, [])

  if (!loaded) return null
  return (
    <Suspense fallback={null}>
      <InstallGuide />
    </Suspense>
  )
}
