import { LandmarkIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

interface OverviewButtonProps {
  /** Icon only, for the phone bar. */
  compact?: boolean
  onClick: () => void
  className?: string
}

/** Opens "Sri Lanka now", from the header on desktop and phone. */
export function OverviewButton({ compact = false, onClick, className }: OverviewButtonProps) {
  const t = useT()
  return (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      onClick={onClick}
      aria-label={t("overview.button.label")}
      className={cn(!compact && "rounded-full bg-card/95 shadow-panel backdrop-blur-md", className)}
    >
      <LandmarkIcon />
      {!compact && t("overview.button.text")}
    </Button>
  )
}
