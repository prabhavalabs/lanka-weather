import * as React from "react"
import { ChevronRightIcon, CloudOffIcon, MapPinIcon, TriangleAlertIcon, XIcon } from "lucide-react"

import { DottedGrid } from "@/components/DottedGrid"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { StatValue } from "@/components/StatValue"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { compassPoint, formatClock, formatDayTime, formatNumber } from "@/lib/format"
import { cn } from "@/lib/utils"

import { AlertList } from "./AlertList"
import { ConditionIcon } from "./ConditionIcon"
import { DailyForecast } from "./DailyForecast"
import { ForecastStrip } from "./ForecastStrip"
import { estimateLabels } from "./severity"
import type {
  CurrentConditions,
  ForecastDay,
  ForecastHour,
  Place,
  WeatherAlert,
  WindReading,
} from "./types"
import { WindCompass } from "./WindCompass"

type Tab = "hourly" | "daily" | "wind" | "alerts"

interface LocationWeatherPanelProps {
  place: Place
  /** Null while loading or where there is no weather. */
  current: CurrentConditions | null
  /** Hourly forecast from the current hour, earliest first (up to 72 hours). */
  forecast: ForecastHour[]
  /** Daily forecast from today (up to 7 days). */
  daily?: ForecastDay[]
  /** Today's local date (YYYY-MM-DD), for the daily list. */
  today?: string
  wind: WindReading | null
  /** Estimates derived by lanka-weather, labelled as such. */
  alerts: WeatherAlert[]
  /** Official warnings (Department of Meteorology, DMC); authoritative, listed first. */
  officialAlerts?: WeatherAlert[]
  /** Shown in place of official warnings while there are none. */
  officialNote?: string
  /**
   * `loading` shows placeholders; `unavailable` says weather is not
   * available here (yet), with `unavailableText`.
   */
  status?: "loading" | "ready" | "unavailable"
  unavailableText?: string
  /** Under the current conditions, e.g. a region's aggregate. */
  highlight?: React.ReactNode
  /** After the forecast: nearby gauges and the like. */
  children?: React.ReactNode
  /** Sources and data age, at the foot. */
  sources?: React.ReactNode
  onClose?: () => void
  className?: string
}

const tabLabels: Record<Tab, string> = {
  hourly: "72 hours",
  daily: "7 days",
  wind: "Wind",
  alerts: "Warnings",
}

/**
 * Weather at a chosen place: current conditions, the next 72 hours and 7
 * days, wind, and warnings, official first and estimates labelled.
 * Presentational; the parent supplies the data.
 */
export function LocationWeatherPanel({
  place,
  current,
  forecast,
  daily = [],
  today,
  wind,
  alerts,
  officialAlerts = [],
  officialNote,
  status = current ? "ready" : "loading",
  unavailableText = "Weather is not available here yet.",
  highlight,
  children,
  sources,
  onClose,
  className,
}: LocationWeatherPanelProps) {
  const [hour, setHour] = React.useState(0)
  const [tab, setTab] = React.useState<Tab>("hourly")
  const selected = forecast[hour]
  const warningCount = alerts.length + officialAlerts.length
  const top = [...officialAlerts, ...alerts][0]

  // A new place starts at "now".
  const placeKey = `${place.latitude},${place.longitude}`
  const [shownPlace, setShownPlace] = React.useState(placeKey)
  if (shownPlace !== placeKey) {
    setShownPlace(placeKey)
    setHour(0)
  }

  return (
    <Panel
      icon={<MapPinIcon />}
      title={place.name}
      subtitle={
        place.localName ? (
          <>
            <span lang="si">{place.localName}</span>
            {place.area && ` · ${place.area}`}
          </>
        ) : (
          place.area
        )
      }
      className={cn("w-full", className)}
      action={
        onClose && (
          <Button
            variant="outline"
            size="icon"
            onClick={onClose}
            aria-label={`Close weather for ${place.name}`}
          >
            <XIcon />
          </Button>
        )
      }
    >
      {status === "loading" && (
        <div className="flex flex-col gap-3" role="status" aria-label="Loading weather">
          <Skeleton className="h-10 w-36" />
          <Skeleton className="h-4 w-52" />
          <Skeleton className="h-28 w-full rounded-2xl" />
        </div>
      )}

      {status === "unavailable" && (
        <DottedGrid className="flex flex-col items-center gap-2 px-4 py-8 text-center">
          <CloudOffIcon className="size-5 text-muted-foreground" aria-hidden="true" />
          <p className="text-sm text-foreground">{unavailableText}</p>
        </DottedGrid>
      )}

      {status === "ready" && current && (
        <>
          <div className="flex items-start gap-4">
            <StatValue
              className="flex-1"
              label="Modelled now"
              value={formatNumber(current.temperatureC, 1)}
              unit="°C"
              caption={
                current.feelsLikeC === null
                  ? current.summary
                  : `${current.summary} · feels like ${formatNumber(current.feelsLikeC)}°`
              }
            />
            <span className="flex size-14 shrink-0 items-center justify-center rounded-2xl border border-border bg-inset">
              <ConditionIcon
                condition={current.condition}
                className="size-7"
                title={current.summary}
              />
            </span>
          </div>

          <CurrentGrid current={current} wind={wind} />
        </>
      )}

      {highlight}

      {status === "ready" && top && (
        <button
          type="button"
          onClick={() => setTab("alerts")}
          className="flex items-center gap-2.5 rounded-xl border border-border bg-inset px-3 py-2.5 text-left text-sm outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring"
        >
          <TriangleAlertIcon className="size-4 shrink-0 text-brand-text" aria-hidden="true" />
          <span className="min-w-0 flex-1">
            <span className="block truncate font-medium text-foreground">{top.title}</span>
            <span className="block truncate text-xs text-muted-foreground">
              {top.estimate
                ? `Estimate · ${estimateLabels[top.severity].toLowerCase()}`
                : top.issuer}
              {warningCount > 1 && ` · ${warningCount - 1} more`}
            </span>
          </span>
          <ChevronRightIcon className="size-4 text-muted-foreground" aria-hidden="true" />
        </button>
      )}

      {status !== "unavailable" && (
        <Tabs value={tab} onValueChange={(value) => setTab(value as Tab)} className="gap-3">
          <TabsList className="w-full">
            {(Object.keys(tabLabels) as Tab[]).map((key) => (
              <TabsTrigger key={key} value={key}>
                {tabLabels[key]}
                {key === "alerts" && warningCount > 0 && (
                  <span className="numeric ml-0.5 rounded-full bg-brand px-1.5 text-[0.6875rem] leading-4 text-brand-foreground">
                    {warningCount}
                  </span>
                )}
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="hourly" className="flex flex-col gap-3">
            {status === "loading" ? (
              <Skeleton className="h-32 w-full rounded-2xl" />
            ) : (
              <>
                <ForecastStrip hours={forecast} selectedIndex={hour} onSelect={setHour} />
                {selected && (
                  <LabeledRows>
                    <LabeledRow
                      label="Time"
                      value={
                        hour === 0
                          ? `Now · ${formatClock(selected.time)}`
                          : formatDayTime(selected.time)
                      }
                    />
                    <LabeledRow
                      label="Temperature"
                      value={`${formatNumber(selected.temperatureC, 1)} °C`}
                    />
                    <LabeledRow
                      label="Rain"
                      value={`${formatNumber(selected.rainfallMm, 1)} mm · ${selected.rainChancePercent}%`}
                      tone={selected.rainfallMm >= 10 ? "signal" : "default"}
                    />
                    {selected.windSpeedKmh !== undefined && selected.windSpeedKmh !== null && (
                      <LabeledRow
                        label="Wind"
                        value={
                          `${formatNumber(selected.windSpeedKmh)} km/h` +
                          (selected.windFromDegrees !== null &&
                          selected.windFromDegrees !== undefined
                            ? ` ${compassPoint(selected.windFromDegrees)}`
                            : "") +
                          (selected.gustKmh !== null && selected.gustKmh !== undefined
                            ? ` · gusts ${formatNumber(selected.gustKmh)}`
                            : "")
                        }
                      />
                    )}
                  </LabeledRows>
                )}
              </>
            )}
          </TabsContent>

          <TabsContent value="daily">
            {status === "loading" ? (
              <Skeleton className="h-56 w-full rounded-2xl" />
            ) : (
              <DailyForecast days={daily} today={today} />
            )}
          </TabsContent>

          <TabsContent value="wind">
            {wind ? (
              <div className="flex items-center gap-5">
                <WindCompass
                  speedKmh={wind.speedKmh}
                  gustKmh={wind.gustKmh}
                  fromDegrees={wind.fromDegrees}
                  size={148}
                />
                <LabeledRows className="flex-1">
                  <LabeledRow
                    label="From"
                    value={`${compassPoint(wind.fromDegrees)} · ${Math.round(wind.fromDegrees)}°`}
                  />
                  <LabeledRow label="Speed" value={`${formatNumber(wind.speedKmh)} km/h`} />
                  <LabeledRow
                    label="Gusts"
                    value={`${formatNumber(wind.gustKmh)} km/h`}
                    tone={wind.gustKmh >= 50 ? "signal" : "default"}
                  />
                </LabeledRows>
              </div>
            ) : (
              <Skeleton className="h-36 w-full" />
            )}
          </TabsContent>

          <TabsContent value="alerts" className="flex flex-col gap-4">
            <section className="flex flex-col gap-2">
              <h3 className="text-xs font-medium text-muted-foreground">Official warnings</h3>
              <AlertList
                alerts={officialAlerts}
                label="Official warnings"
                emptyText={officialNote ?? "No official warnings in force for this area."}
                emptyIcon={officialNote ? "info" : "check"}
              />
            </section>
            <section className="flex flex-col gap-2">
              <div>
                <h3 className="text-xs font-medium text-muted-foreground">Estimates</h3>
                <p className="mt-0.5 text-xs text-muted-foreground">
                  Derived by Lanka Weather from forecasts and gauges. Not official warnings.
                </p>
              </div>
              <AlertList
                alerts={alerts}
                label="Estimates"
                emptyText="No hazards estimated from the forecast."
              />
            </section>
          </TabsContent>
        </Tabs>
      )}

      {children}

      {sources && <div className="text-xs text-muted-foreground">{sources}</div>}
    </Panel>
  )
}

/** Small label-over-value cells in a hairline grid. */
function CurrentGrid({ current, wind }: { current: CurrentConditions; wind: WindReading | null }) {
  const cells: { label: string; value: string; note?: string }[] = [
    {
      label: "Humidity",
      value: current.humidityPercent === null ? "–" : `${Math.round(current.humidityPercent)}%`,
      note:
        current.dewPointC === null || current.dewPointC === undefined
          ? ""
          : `dew ${formatNumber(current.dewPointC)}°`,
    },
    {
      label: "Rain now",
      value: current.rainfallMm === null ? "–" : `${formatNumber(current.rainfallMm, 1)} mm`,
      note: "this hour",
    },
    {
      label: "Chance",
      value:
        current.rainChancePercent === null || current.rainChancePercent === undefined
          ? "–"
          : `${Math.round(current.rainChancePercent)}%`,
      note: "this hour",
    },
    {
      label: "Rain, 24 h",
      value:
        current.rainNext24hMm === null || current.rainNext24hMm === undefined
          ? "–"
          : `${formatNumber(current.rainNext24hMm, 1)} mm`,
      note: "ahead",
    },
    {
      label: "Wind",
      value: wind ? `${formatNumber(wind.speedKmh)} km/h` : "–",
      note: wind ? compassPoint(wind.fromDegrees) : "",
    },
    {
      label: "Cloud",
      value:
        current.cloudCoverPercent === null || current.cloudCoverPercent === undefined
          ? "–"
          : `${Math.round(current.cloudCoverPercent)}%`,
      note: "cover",
    },
    {
      label: "Pressure",
      value: current.pressureHpa === null ? "–" : `${formatNumber(current.pressureHpa)}`,
      note: "hPa",
    },
    {
      label: "UV index",
      value:
        current.uvIndex === null || current.uvIndex === undefined
          ? "–"
          : formatNumber(current.uvIndex),
      note: uvWords(current.uvIndex),
    },
  ]
  return (
    <dl className="grid grid-cols-4 gap-px overflow-hidden rounded-xl border border-border bg-border">
      {cells.map((cell) => (
        <div key={cell.label} className="flex min-w-0 flex-col gap-0.5 bg-card px-2.5 py-2">
          <dt className="truncate text-[0.6875rem] text-muted-foreground">{cell.label}</dt>
          <dd className="numeric truncate text-sm text-foreground">{cell.value}</dd>
          {cell.note !== undefined && (
            <dd className="numeric truncate text-[0.6875rem] text-muted-foreground">
              {cell.note || " "}
            </dd>
          )}
        </div>
      ))}
    </dl>
  )
}

/** WHO UV index categories. */
function uvWords(uv: number | null | undefined) {
  if (uv === null || uv === undefined) return undefined
  if (uv < 3) return "low"
  if (uv < 6) return "moderate"
  if (uv < 8) return "high"
  if (uv < 11) return "very high"
  return "extreme"
}
