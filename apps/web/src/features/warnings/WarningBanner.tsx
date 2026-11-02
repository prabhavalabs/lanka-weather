import { ChevronRightIcon, ExternalLinkIcon, TriangleAlertIcon, XIcon } from "lucide-react"
import { useEffect } from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useLocalStorage } from "@/lib/useLocalStorage"
import { cn } from "@/lib/utils"
import { bannerLead, colourVariant, formatShort, pruneDismissed, visibleBanner } from "./present"
import type { BannerItem } from "./types"

const edge: Record<string, string> = {
  Red: "border-status-major-flood",
  Amber: "border-status-minor-flood",
  Yellow: "border-status-alert",
}

interface WarningBannerProps {
  items: BannerItem[]
  /** The data behind it has not been checked recently. */
  stale?: boolean
  onOpen: () => void
}

/**
 * The prominent notice for a level 2 or 3 landslide warning or a bulletin that
 * calls itself a warning. Dismissing it hides what it announced; anything new
 * brings it back.
 */
export function WarningBanner({ items, stale = false, onOpen }: WarningBannerProps) {
  const [dismissed, setDismissed] = useLocalStorage<string[]>(
    "lanka-weather:dismissed-warnings",
    [],
  )

  // Forget dismissals of what is no longer announced.
  useEffect(() => {
    if (items.length === 0) return
    const kept = pruneDismissed(dismissed, items)
    if (kept.length !== dismissed.length) setDismissed(kept)
  }, [items, dismissed, setDismissed])

  const visible = visibleBanner(items, dismissed)
  const lead = bannerLead(visible)
  if (!lead) return null
  const others = visible.length - 1

  return (
    <div
      role="alert"
      data-testid="warning-banner"
      className={cn(
        "pointer-events-auto flex w-full max-w-xl items-start gap-2.5 rounded-2xl border-2 bg-card/97 py-2.5 pr-2 pl-3 text-sm shadow-panel backdrop-blur-md",
        edge[lead.colour ?? ""] ?? "border-brand",
      )}
    >
      <TriangleAlertIcon className="mt-0.5 size-4 shrink-0 text-brand-text" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <p className="leading-snug font-medium text-foreground">{lead.title}</p>
        <p className="mt-0.5 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs text-muted-foreground">
          {lead.colour && (
            <Badge variant={colourVariant(lead.colour)} className="h-5">
              {lead.colour}
            </Badge>
          )}
          <span>{lead.detail}</span>
          {lead.issuedAt && <span className="numeric">· {formatShort(lead.issuedAt)}</span>}
          {stale && <span>· Out of date</span>}
        </p>
        <p className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
          <button
            type="button"
            onClick={onOpen}
            className="inline-flex items-center gap-0.5 rounded-sm font-medium text-foreground underline underline-offset-2 outline-none hover:text-brand-text focus-visible:ring-2 focus-visible:ring-ring"
          >
            {others > 0 ? `All warnings (${others} more)` : "All warnings"}
            <ChevronRightIcon className="size-3.5" aria-hidden="true" />
          </button>
          {lead.pdfUrl && (
            <a
              href={lead.pdfUrl}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 rounded-sm font-medium text-foreground underline underline-offset-2 outline-none hover:text-brand-text focus-visible:ring-2 focus-visible:ring-ring"
            >
              Official PDF
              <ExternalLinkIcon className="size-3" aria-hidden="true" />
              <span className="sr-only">(opens in a new tab)</span>
            </a>
          )}
        </p>
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="size-8 shrink-0"
        aria-label="Dismiss this notice"
        onClick={() =>
          setDismissed([...new Set([...dismissed, ...visible.map((item) => item.key)])])
        }
      >
        <XIcon />
      </Button>
    </div>
  )
}
