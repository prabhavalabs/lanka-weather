import { DottedGrid } from "@/components/DottedGrid"
import { formatClock, formatNumber, formatWeekday } from "@/lib/format"
import { cn } from "@/lib/utils"

import { ConditionIcon } from "./ConditionIcon"
import { conditionLabels } from "./conditions"
import type { ForecastHour } from "./types"

interface ForecastStripProps {
  /** Hourly forecast, usually the next 48 hours, earliest first. */
  hours: ForecastHour[]
  /** Highlighted hour; the first is "Now". */
  selectedIndex?: number
  /** Makes the hours selectable, e.g. to scrub the map's forecast time. */
  onSelect?: (index: number) => void
  /** Rainfall that fills a bar, in mm per hour. */
  rainScaleMm?: number
  className?: string
}

/**
 * An hour-by-hour strip: time, condition, temperature and a rainfall bar,
 * scrolling sideways on a dotted inset. Day boundaries are labelled.
 */
export function ForecastStrip({
  hours,
  selectedIndex = 0,
  onSelect,
  rainScaleMm = 15,
  className,
}: ForecastStripProps) {
  if (hours.length === 0) {
    return (
      <DottedGrid className={cn("px-4 py-8 text-center text-sm text-muted-foreground", className)}>
        No forecast available.
      </DottedGrid>
    )
  }

  return (
    // inline-size containment keeps 48 hours of columns from widening a grid or flex parent.
    <DottedGrid className={cn("overflow-hidden [contain:inline-size]", className)}>
      <ol
        className="flex snap-x overflow-x-auto overscroll-x-contain px-2 py-3 [scrollbar-width:thin]"
        aria-label="Hourly forecast"
      >
        {hours.map((hour, index) => {
          const selected = index === selectedIndex
          const clock = formatClock(hour.time)
          const newDay = index > 0 && clock === "00:00"
          const label = `${index === 0 ? "Now" : `${formatWeekday(hour.time)} ${clock}`}: ${conditionLabels[hour.condition]}, ${formatNumber(hour.temperatureC)} °C, ${formatNumber(hour.rainfallMm, 1)} mm rain, ${hour.rainChancePercent}% chance`
          const content = (
            <>
              <span
                className={cn(
                  "numeric text-[0.6875rem]",
                  selected ? "text-foreground" : "text-muted-foreground",
                )}
              >
                {index === 0 ? "Now" : newDay ? formatWeekday(hour.time) : clock.slice(0, 2)}
              </span>
              <ConditionIcon condition={hour.condition} className="size-4" />
              <span className="numeric text-sm text-foreground">
                {formatNumber(hour.temperatureC)}°
              </span>
              <span className="flex h-10 w-full items-end justify-center" aria-hidden="true">
                <span
                  className={cn(
                    "w-1.5 rounded-full",
                    hour.rainfallMm === 0 ? "hatched" : selected ? "bg-brand" : "bg-chart-3",
                  )}
                  style={{
                    height: `${Math.max(8, Math.min(1, hour.rainfallMm / rainScaleMm) * 100)}%`,
                  }}
                />
              </span>
            </>
          )
          const cellClass = cn(
            "flex w-11 flex-col items-center gap-1.5 rounded-xl border border-transparent py-2",
            selected && "border-brand bg-brand-soft",
          )
          return (
            <li
              key={hour.time}
              className={cn(
                "shrink-0 snap-start",
                newDay && "ml-1 border-l border-dashed border-border pl-1",
              )}
            >
              {onSelect ? (
                <button
                  type="button"
                  aria-pressed={selected}
                  aria-label={label}
                  onClick={() => onSelect(index)}
                  className={cn(
                    cellClass,
                    "outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring",
                  )}
                >
                  {content}
                </button>
              ) : (
                <div className={cellClass} aria-label={label} role="group">
                  {content}
                </div>
              )}
            </li>
          )
        })}
      </ol>
    </DottedGrid>
  )
}
