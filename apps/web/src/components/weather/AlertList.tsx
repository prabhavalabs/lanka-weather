import { CheckIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { formatDayTime } from "@/lib/format"
import { cn } from "@/lib/utils"

import { severityBadge, severityColors, severityLabels } from "./severity"
import type { WeatherAlert } from "./types"

interface AlertListProps {
  alerts: WeatherAlert[]
  /** Shown when there are no alerts. */
  emptyText?: string
  className?: string
}

/** Official alerts, most severe first, each with its issuer, area and validity. */
export function AlertList({
  alerts,
  emptyText = "No warnings in force for this area.",
  className,
}: AlertListProps) {
  if (alerts.length === 0) {
    return (
      <p
        className={cn(
          "flex items-center gap-2 rounded-xl border border-border bg-inset px-3 py-3 text-sm text-muted-foreground",
          className,
        )}
      >
        <CheckIcon className="size-4" aria-hidden="true" />
        {emptyText}
      </p>
    )
  }

  const order = ["emergency", "warning", "watch", "advisory"]
  const sorted = [...alerts].sort((a, b) => order.indexOf(a.severity) - order.indexOf(b.severity))

  return (
    <ul className={cn("flex flex-col gap-2", className)} aria-label="Warnings">
      {sorted.map((alert) => (
        <li
          key={alert.id}
          className="relative overflow-hidden rounded-xl border border-border bg-inset py-3 pr-3 pl-4"
        >
          <span
            aria-hidden="true"
            className="absolute inset-y-2 left-1.5 w-1 rounded-full"
            style={{ backgroundColor: severityColors[alert.severity] }}
          />
          <div className="flex items-start justify-between gap-3">
            <p className="text-sm leading-snug font-medium text-foreground">{alert.title}</p>
            <Badge variant={severityBadge[alert.severity]} className="mt-px">
              {severityLabels[alert.severity]}
            </Badge>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            {alert.issuer} · {alert.area}
          </p>
          {alert.summary && <p className="mt-2 text-sm text-foreground/90">{alert.summary}</p>}
          <p className="numeric mt-2 text-[0.6875rem] text-muted-foreground">
            <time dateTime={alert.issuedAt}>{formatDayTime(alert.issuedAt)}</time>
            {alert.validUntil && (
              <>
                {" → "}
                <time dateTime={alert.validUntil}>{formatDayTime(alert.validUntil)}</time>
              </>
            )}
          </p>
        </li>
      ))}
    </ul>
  )
}
