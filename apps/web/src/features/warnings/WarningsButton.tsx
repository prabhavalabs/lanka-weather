import { BellIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { badgeText } from "@/features/notifications/present"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

interface WarningsButtonProps {
  /** Warnings and advisories in force; shown as a badge when above zero. */
  count: number
  /** The data has not been checked recently; a zero must not read as calm. */
  stale?: boolean
  /** Alerts this device was sent and has not seen: a red count on the corner. */
  unread?: number
  /** Icon and badge only, for the phone bar. */
  compact?: boolean
  onClick: () => void
  className?: string
}

/**
 * Opens the warnings panel, official warnings first and this device's alerts
 * on the second tab, from the header on desktop and phone.
 */
export function WarningsButton({
  count,
  unread = 0,
  stale = false,
  compact = false,
  onClick,
  className,
}: WarningsButtonProps) {
  const t = useT()
  const badge = badgeText(unread)
  const label = t(stale ? "warnings.button.labelStale" : "warnings.button.label", { count })
  return (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      onClick={onClick}
      aria-label={badge ? `${label}. ${t("notifications.buttonUnread", { count: unread })}` : label}
      className={cn(
        "relative",
        !compact && "rounded-full bg-card/95 shadow-panel backdrop-blur-md",
        className,
      )}
    >
      <BellIcon />
      {!compact && t("warnings.button.text")}
      {count > 0 && (
        <span className="numeric rounded-full bg-brand px-1.5 text-[0.6875rem] leading-4 text-brand-foreground">
          {count}
        </span>
      )}
      {stale && count === 0 && <span aria-hidden="true">?</span>}
      {badge && (
        <span
          aria-hidden="true"
          className="numeric absolute -top-1 -right-1 flex h-4.5 min-w-4.5 items-center justify-center rounded-full bg-destructive px-1 text-[0.625rem] leading-none font-semibold text-white ring-2 ring-background"
        >
          {badge}
        </span>
      )}
    </Button>
  )
}
