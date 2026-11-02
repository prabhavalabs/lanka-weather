import { BellIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface WarningsButtonProps {
  /** Warnings and advisories in force; shown as a badge when above zero. */
  count: number
  /** The data has not been checked recently; a zero must not read as calm. */
  stale?: boolean
  /** Icon and badge only, for the phone bar. */
  compact?: boolean
  onClick: () => void
  className?: string
}

/** Opens the official warnings, from the header on desktop and phone. */
export function WarningsButton({
  count,
  stale = false,
  compact = false,
  onClick,
  className,
}: WarningsButtonProps) {
  const state = stale ? ", out of date" : ""
  return (
    <Button
      type="button"
      variant="secondary"
      size="sm"
      onClick={onClick}
      aria-label={`Official warnings, ${count} in force${state}`}
      className={cn(!compact && "rounded-full bg-card/95 shadow-panel backdrop-blur-md", className)}
    >
      <BellIcon />
      {!compact && "Warnings"}
      {count > 0 && (
        <span className="numeric rounded-full bg-brand px-1.5 text-[0.6875rem] leading-4 text-brand-foreground">
          {count}
        </span>
      )}
      {stale && count === 0 && <span aria-hidden="true">?</span>}
    </Button>
  )
}
