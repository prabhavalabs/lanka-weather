import { InfoIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { openAbout } from "./state"

/** "Sources, credits and disclaimer", under the small print in the sidebar and the phone menu. */
export function AboutLink({ className }: { className?: string }) {
  const t = useT()
  return (
    <button
      type="button"
      onClick={openAbout}
      className={cn(
        "inline-flex items-center gap-1.5 self-start rounded-sm text-left text-xs font-medium text-foreground underline underline-offset-2 outline-none hover:text-brand-text focus-visible:ring-2 focus-visible:ring-ring",
        className,
      )}
    >
      <InfoIcon className="size-3.5 shrink-0" aria-hidden="true" />
      {t("about.link")}
    </button>
  )
}

/** The same page from an icon button, by the sidebar's title. */
export function AboutButton() {
  const t = useT()
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="outline" size="icon" onClick={openAbout} aria-label={t("about.link")}>
          <InfoIcon />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom">{t("about.link")}</TooltipContent>
    </Tooltip>
  )
}
