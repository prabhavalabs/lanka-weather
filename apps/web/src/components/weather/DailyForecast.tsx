import { formatNumber, formatWeekday } from "@/lib/format"
import { stepIndex } from "@/lib/colour"
import { useT } from "@/lib/i18n"
import { temperatureScale } from "@/lib/tokens"
import { cn } from "@/lib/utils"

import { capitalise } from "./capitalise"
import { ConditionIcon } from "./ConditionIcon"
import type { ForecastDay } from "./types"

interface DailyForecastProps {
  days: ForecastDay[]
  /** Today's local date (YYYY-MM-DD), labelled "Today". */
  today?: string
  className?: string
}

/** The temperature colour of a value, as a CSS variable. */
function temperatureColour(value: number) {
  const step = temperatureScale[Math.max(0, stepIndex(temperatureScale, value))]!
  return `var(${step.token})`
}

/** Noon of a local date, so the weekday is right in Sri Lanka time. */
function noon(date: string) {
  return `${date}T12:00:00+05:30`
}

/**
 * The days ahead, one row each: weekday, condition, chance of rain, the
 * day's low and high on a shared scale (coloured like the temperature
 * layer), and the day's rain.
 */
export function DailyForecast({ days, today, className }: DailyForecastProps) {
  const t = useT()
  const lows = days.map((day) => day.minC).filter((value): value is number => value !== null)
  const highs = days.map((day) => day.maxC).filter((value): value is number => value !== null)
  if (days.length === 0 || lows.length === 0 || highs.length === 0) {
    return (
      <p className="rounded-2xl border border-border bg-inset px-4 py-8 text-center text-sm text-muted-foreground">
        {t("weather.daily.none")}
      </p>
    )
  }
  const floor = Math.floor(Math.min(...lows))
  const ceiling = Math.ceil(Math.max(...highs))
  const span = Math.max(1, ceiling - floor)

  return (
    <ol
      className={cn("flex flex-col divide-y divide-border", className)}
      aria-label={t("weather.daily.label")}
    >
      {days.map((day) => {
        const label =
          day.date === today ? capitalise(t("format.today")) : formatWeekday(noon(day.date))
        const low = day.minC
        const high = day.maxC
        const left = low === null ? 0 : ((low - floor) / span) * 100
        const right = high === null ? 100 : ((high - floor) / span) * 100
        const spoken = [
          t("weather.daily.spoken", { day: label, summary: day.summary }),
          low !== null && high !== null
            ? t("weather.daily.spokenRange", { low: formatNumber(low), high: formatNumber(high) })
            : null,
          day.rainMm !== null
            ? t("weather.daily.spokenRain", { rain: formatNumber(day.rainMm, 1) })
            : null,
          day.rainChancePercent !== null
            ? t("weather.daily.spokenChance", { chance: Math.round(day.rainChancePercent) })
            : null,
        ]
          .filter(Boolean)
          .join(", ")
        return (
          <li
            key={day.date}
            className="grid grid-cols-[3.25rem_1.25rem_2.25rem_2rem_1fr_2rem_3.25rem] items-center gap-x-2 py-2 text-sm"
            aria-label={spoken}
          >
            <span
              className={cn("truncate", day.date === today ? "font-medium" : "")}
              title={label}
              aria-hidden
            >
              {label}
            </span>
            <ConditionIcon condition={day.condition} className="size-4" />
            <span className="numeric text-right text-[0.6875rem] text-muted-foreground" aria-hidden>
              {day.rainChancePercent !== null && day.rainChancePercent >= 10
                ? `${Math.round(day.rainChancePercent)}%`
                : ""}
            </span>
            <span className="numeric text-right text-muted-foreground" aria-hidden>
              {low === null ? "–" : `${formatNumber(low)}°`}
            </span>
            <span className="relative h-1.5 rounded-full bg-muted" aria-hidden>
              {low !== null && high !== null && (
                <span
                  className="absolute inset-y-0 rounded-full"
                  style={{
                    left: `${left}%`,
                    right: `${100 - right}%`,
                    backgroundImage: `linear-gradient(to right, ${temperatureColour(low)}, ${temperatureColour(high)})`,
                  }}
                />
              )}
            </span>
            <span className="numeric text-foreground" aria-hidden>
              {high === null ? "–" : `${formatNumber(high)}°`}
            </span>
            <span
              className={cn(
                "numeric text-right text-xs",
                day.rainMm !== null && day.rainMm >= 25
                  ? "text-brand-text"
                  : "text-muted-foreground",
              )}
              aria-hidden
            >
              {day.rainMm === null ? "–" : `${formatNumber(day.rainMm, 1)} mm`}
            </span>
          </li>
        )
      })}
    </ol>
  )
}
