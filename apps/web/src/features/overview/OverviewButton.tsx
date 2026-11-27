import { LandmarkIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

interface OverviewButtonProps {
  /** Icon only, for the phone bar. */
  compact?: boolean
  /** "Sri Lanka now" is the panel on show: a click closes it. */
  open?: boolean
  onClick: () => void
  className?: string
}

/** Opens and closes "Sri Lanka now", from the header on desktop and phone. */
export function OverviewButton({
  compact = false,
  open = false,
  onClick,
  className,
}: OverviewButtonProps) {
  const t = useT()
  return (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      onClick={onClick}
      aria-label={t("overview.button.label")}
      aria-pressed={open}
      className={cn(
        !compact && "rounded-full bg-card/95 shadow-panel backdrop-blur-md",
        "aria-pressed:bg-foreground aria-pressed:text-background aria-pressed:hover:bg-foreground/90",
        className,
      )}
    >
      <LandmarkIcon />
      {!compact && t("overview.button.text")}
    </Button>
  )
}
