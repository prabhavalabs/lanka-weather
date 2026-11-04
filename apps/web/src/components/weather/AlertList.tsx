import { CheckIcon, ExternalLinkIcon, InfoIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { formatDayTime } from "@/lib/format"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

import { fillNodes } from "./fillNodes"
import { estimateLabel, severityBadge, severityColors, severityLabel } from "./severity"
import type { WeatherAlert } from "./types"

interface AlertListProps {
  alerts: WeatherAlert[]
  /** Shown when there are no alerts. */
  emptyText?: string
  /** Accessible name of the list. */
  label?: string
  /** The empty state's icon: a tick for "none in force", or "info" for a note. */
  emptyIcon?: "check" | "info"
  className?: string
}

const order = ["emergency", "warning", "watch", "advisory"]

/**
 * Alerts, most severe first, each with its issuer, area and validity.
 * Estimates (alerts lanka-weather derives itself) carry an "Estimate" badge
 * and the data they come from, so they are never mistaken for official
 * warnings.
 */
export function AlertList({
  alerts,
  emptyText,
  label,
  emptyIcon = "check",
  className,
}: AlertListProps) {
  const t = useT()
  if (alerts.length === 0) {
    return (
      <p
        className={cn(
          "flex items-center gap-2 rounded-xl border border-border bg-inset px-3 py-3 text-sm text-muted-foreground",
          className,
        )}
      >
        {emptyIcon === "info" ? (
          <InfoIcon className="size-4 shrink-0" aria-hidden="true" />
        ) : (
          <CheckIcon className="size-4 shrink-0" aria-hidden="true" />
        )}
        {emptyText ?? t("weather.alerts.none")}
      </p>
    )
  }

  const sorted = [...alerts].sort((a, b) => order.indexOf(a.severity) - order.indexOf(b.severity))

  return (
    <ul
      className={cn("flex flex-col gap-2", className)}
      aria-label={label ?? t("weather.alerts.label")}
    >
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
          {/* Badges wrap under the title when the words run long (Tamil, Sinhala). */}
          <div className="flex flex-wrap items-start justify-between gap-x-3 gap-y-1.5">
            <p
              className="min-w-0 flex-1 basis-40 text-sm leading-snug font-medium break-words text-foreground"
              lang={alert.titleLang}
            >
              {alert.title}
            </p>
            <div className="mt-px flex max-w-full flex-wrap items-center gap-1.5">
              {alert.estimate && (
                <Badge variant="outline" title={t("weather.alerts.estimateTitle")}>
                  {t("terms.estimate")}
                </Badge>
              )}
              <Badge variant={severityBadge[alert.severity]}>
                {alert.badgeLabel ??
                  (alert.estimate ? estimateLabel(alert.severity) : severityLabel(alert.severity))}
              </Badge>
            </div>
          </div>
          <p className="mt-1 text-xs break-words text-muted-foreground">
            {alert.issuer}
            {alert.area && ` · ${alert.area}`}
          </p>
          {alert.summary && (
            <p
              className="mt-2 line-clamp-6 text-sm break-words whitespace-pre-line text-foreground/90"
              lang={alert.summaryLang}
            >
              {alert.summary}
            </p>
          )}
          {alert.excerpt && (
            <p className="mt-1 text-[0.6875rem] text-muted-foreground">
              {t("weather.alerts.excerpt")}
            </p>
          )}
          {alert.basis && (
            <p className="mt-1.5 text-xs break-words text-muted-foreground">
              {fillNodes(t("weather.alerts.derivedFrom"), {
                basis: <span lang={alert.basisLang}>{alert.basis}</span>,
              })}
            </p>
          )}
          <p className="numeric mt-2 text-[0.6875rem] text-muted-foreground">
            <time dateTime={alert.issuedAt}>{formatDayTime(alert.issuedAt)}</time>
            {alert.validUntil && (
              <>
                {" → "}
                <time dateTime={alert.validUntil}>{formatDayTime(alert.validUntil)}</time>
              </>
            )}
          </p>
          {alert.link && (
            <a
              href={alert.link.href}
              target="_blank"
              rel="noreferrer"
              className="mt-1.5 inline-flex items-center gap-1 rounded-sm text-xs font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
            >
              {alert.link.label}
              <ExternalLinkIcon className="size-3" aria-hidden="true" />
              <span className="sr-only">{t("weather.alerts.newTab")}</span>
            </a>
          )}
        </li>
      ))}
    </ul>
  )
}
