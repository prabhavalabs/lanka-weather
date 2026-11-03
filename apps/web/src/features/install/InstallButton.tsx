import { DownloadIcon } from "lucide-react"
import { useMemo } from "react"
import { Button } from "@/components/ui/button"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { isStandalone } from "./platform"
import { openInstallGuide } from "./state"

/**
 * "Install app", for the sidebar footer and the phone's layer drawer.
 * Hidden inside the installed app, where there is nothing to install.
 */
export function InstallButton({ className }: { className?: string }) {
  const t = useT()
  const standalone = useMemo(() => isStandalone(), [])
  if (standalone) return null
  return (
    <Button
      variant="secondary"
      className={cn("h-10 w-full justify-start", className)}
      title={t("install.openHint")}
      onClick={() => openInstallGuide({ reason: "menu" })}
    >
      <DownloadIcon />
      {t("install.open")}
    </Button>
  )
}
