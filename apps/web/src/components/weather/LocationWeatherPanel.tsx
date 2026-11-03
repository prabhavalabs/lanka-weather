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
import { intlTag, useT, type MessageKey, type Translate } from "@/lib/i18n"
import { cn } from "@/lib/utils"

import { AlertList } from "./AlertList"
import { ConditionIcon } from "./ConditionIcon"
import { DailyForecast } from "./DailyForecast"
import { ForecastStrip } from "./ForecastStrip"
import { estimateLabel } from "./severity"
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
  /** What can be said about official warnings here; never that a place is safe. */
  officialNote?: string
  /** Latest official bulletins, for reading; not counted as warnings. */
  officialBulletins?: WeatherAlert[]
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

const tabs: Record<Tab, MessageKey> = {
  hourly: "weather.panel.tabs.hourly",
  daily: "weather.panel.tabs.daily",
  wind: "weather.panel.tabs.wind",
  alerts: "weather.panel.tabs.alerts",
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
  officialBulletins = [],
  status = current ? "ready" : "loading",
  unavailableText,
  highlight,
  children,
  sources,
  onClose,
  className,
}: LocationWeatherPanelProps) {
  const t = useT()
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
            <span lang={place.localNameLang ?? "si"}>{place.localName}</span>
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
            aria-label={t("weather.panel.closeFor", { name: place.name })}
          >
            <XIcon />
          </Button>
        )
      }
    >
      {status === "loading" && (
        <div className="flex flex-col gap-3" role="status" aria-label={t("weather.panel.loading")}>
          <Skeleton className="h-10 w-36" />
          <Skeleton className="h-4 w-52" />
          <Skeleton className="h-28 w-full rounded-2xl" />
        </div>
      )}

      {status === "unavailable" && (
        <DottedGrid className="flex flex-col items-center gap-2 px-4 py-8 text-center">
          <CloudOffIcon className="size-5 text-muted-foreground" aria-hidden="true" />
          <p className="text-sm text-foreground">
            {unavailableText ?? t("weather.panel.unavailable")}
          </p>
        </DottedGrid>
      )}

      {status === "ready" && current && (
        <>
          <div className="flex items-start gap-4">
            <StatValue
              className="flex-1"
              label={t("weather.panel.modelledNow")}
              value={formatNumber(current.temperatureC, 1)}
              unit="°C"
              caption={
                current.feelsLikeC === null
                  ? current.summary
                  : t("weather.panel.feelsLike", {
                      summary: current.summary,
                      value: formatNumber(current.feelsLikeC),
                    })
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
            <span className="block truncate font-medium text-foreground" lang={top.titleLang}>
              {top.title}
              {top.badgeLabel && ` · ${top.badgeLabel}`}
            </span>
            <span className="block truncate text-xs text-muted-foreground">
              {warningCount > 1
                ? t("weather.panel.andMore", { text: topLine(top, t), count: warningCount - 1 })
                : topLine(top, t)}
            </span>
          </span>
          <ChevronRightIcon className="size-4 text-muted-foreground" aria-hidden="true" />
        </button>
      )}

      {status !== "unavailable" && (
        <Tabs value={tab} onValueChange={(value) => setTab(value as Tab)} className="gap-3">
          <TabsList className="w-full">
            {(Object.keys(tabs) as Tab[]).map((key) => (
              <TabsTrigger key={key} value={key} className="min-w-0 px-1.5" title={t(tabs[key])}>
                <span className="truncate">{t(tabs[key])}</span>
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
                      label={t("weather.panel.time")}
                      value={
                        hour === 0
                          ? t("weather.panel.nowAt", { time: formatClock(selected.time) })
                          : formatDayTime(selected.time)
                      }
                    />
                    <LabeledRow
                      label={t("weather.panel.temperature")}
                      value={`${formatNumber(selected.temperatureC, 1)} °C`}
                    />
                    <LabeledRow
                      label={t("weather.panel.rain")}
                      value={t("weather.panel.rainValue", {
                        amount: formatNumber(selected.rainfallMm, 1),
                        chance: selected.rainChancePercent,
                      })}
                      tone={selected.rainfallMm >= 10 ? "signal" : "default"}
                    />
                    {selected.windSpeedKmh !== undefined && selected.windSpeedKmh !== null && (
                      <LabeledRow
                        label={t("weather.panel.wind")}
                        value={hourWind(selected, selected.windSpeedKmh, t)}
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
                    label={t("weather.panel.from")}
                    value={t("weather.panel.fromValue", {
                      direction: compassPoint(wind.fromDegrees),
                      degrees: Math.round(wind.fromDegrees),
                    })}
                  />
                  <LabeledRow
                    label={t("weather.panel.speed")}
                    value={t("weather.panel.speedValue", { speed: formatNumber(wind.speedKmh) })}
                  />
                  <LabeledRow
                    label={t("weather.panel.gusts")}
                    value={t("weather.panel.speedValue", { speed: formatNumber(wind.gustKmh) })}
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
              <h3 className="text-xs font-medium text-muted-foreground">
                {t("weather.panel.officialWarnings")}
              </h3>
              <AlertList
                alerts={officialAlerts}
                label={t("weather.panel.officialWarnings")}
                emptyText={officialNote ?? t("weather.panel.officialNone")}
                emptyIcon={officialNote ? "info" : "check"}
              />
              {officialAlerts.length > 0 && officialNote && (
                <p className="text-xs text-muted-foreground">{officialNote}</p>
              )}
              {officialBulletins.length > 0 && (
                <>
                  <h3 className="mt-2 text-xs font-medium text-muted-foreground">
                    {t("weather.panel.latestBulletins")}
                  </h3>
                  <AlertList
                    alerts={officialBulletins}
                    label={t("weather.panel.latestBulletinsLabel")}
                  />
                </>
              )}
            </section>
            <section className="flex flex-col gap-2">
              <div>
                <h3 className="text-xs font-medium text-muted-foreground">
                  {t("weather.panel.estimates")}
                </h3>
                <p className="mt-0.5 text-xs text-muted-foreground">
                  {t("weather.panel.estimatesNote")}
                </p>
              </div>
              <AlertList
                alerts={alerts}
                label={t("weather.panel.estimates")}
                emptyText={t("weather.panel.estimatesNone")}
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

/** The top alert's second line: an estimate's level, or the issuer of an official warning. */
function topLine(alert: WeatherAlert, t: Translate): string {
  if (!alert.estimate) return alert.issuer
  const level = estimateLabel(alert.severity).toLocaleLowerCase(intlTag())
  return t("weather.panel.estimateLevel", { level })
}

/** An hour's wind: "12 km/h SW · gusts 25". */
function hourWind(hour: ForecastHour, speedKmh: number, t: Translate): string {
  const speed = formatNumber(speedKmh)
  const wind =
    hour.windFromDegrees !== null && hour.windFromDegrees !== undefined
      ? t("weather.panel.speedFrom", { speed, direction: compassPoint(hour.windFromDegrees) })
      : t("weather.panel.speedValue", { speed })
  return hour.gustKmh !== null && hour.gustKmh !== undefined
    ? t("weather.panel.withGusts", { wind, gust: formatNumber(hour.gustKmh) })
    : wind
}

/** Small label-over-value cells in a hairline grid. */
function CurrentGrid({ current, wind }: { current: CurrentConditions; wind: WindReading | null }) {
  const t = useT()
  const cells: { label: string; value: string; note?: string }[] = [
    {
      label: t("weather.current.humidity"),
      value: current.humidityPercent === null ? "–" : `${Math.round(current.humidityPercent)}%`,
      note:
        current.dewPointC === null || current.dewPointC === undefined
          ? ""
          : t("weather.current.dew", { value: formatNumber(current.dewPointC) }),
    },
    {
      label: t("weather.current.rainNow"),
      value: current.rainfallMm === null ? "–" : `${formatNumber(current.rainfallMm, 1)} mm`,
      note: t("weather.current.thisHour"),
    },
    {
      label: t("weather.current.chance"),
      value:
        current.rainChancePercent === null || current.rainChancePercent === undefined
          ? "–"
          : `${Math.round(current.rainChancePercent)}%`,
      note: t("weather.current.thisHour"),
    },
    {
      label: t("weather.current.rain24h"),
      value:
        current.rainNext24hMm === null || current.rainNext24hMm === undefined
          ? "–"
          : `${formatNumber(current.rainNext24hMm, 1)} mm`,
      note: t("weather.current.ahead"),
    },
    {
      label: t("weather.current.wind"),
      value: wind ? `${formatNumber(wind.speedKmh)} km/h` : "–",
      note: wind ? compassPoint(wind.fromDegrees) : "",
    },
    {
      label: t("weather.current.cloud"),
      value:
        current.cloudCoverPercent === null || current.cloudCoverPercent === undefined
          ? "–"
          : `${Math.round(current.cloudCoverPercent)}%`,
      note: t("weather.current.cover"),
    },
    {
      label: t("weather.current.pressure"),
      value: current.pressureHpa === null ? "–" : `${formatNumber(current.pressureHpa)}`,
      note: "hPa",
    },
    {
      label: t("weather.current.uvIndex"),
      value:
        current.uvIndex === null || current.uvIndex === undefined
          ? "–"
          : formatNumber(current.uvIndex),
      note: uvWords(current.uvIndex, t),
    },
  ]
  return (
    <dl className="grid grid-cols-4 gap-px overflow-hidden rounded-xl border border-border bg-border">
      {cells.map((cell) => (
        <div key={cell.label} className="flex min-w-0 flex-col gap-0.5 bg-card px-2.5 py-2">
          <dt className="truncate text-[0.6875rem] text-muted-foreground" title={cell.label}>
            {cell.label}
          </dt>
          <dd className="numeric truncate text-sm text-foreground">{cell.value}</dd>
          {cell.note !== undefined && (
            <dd
              className="numeric truncate text-[0.6875rem] text-muted-foreground"
              title={cell.note || undefined}
            >
              {cell.note || " "}
            </dd>
          )}
        </div>
      ))}
    </dl>
  )
}

/** WHO UV index categories. */
function uvWords(uv: number | null | undefined, t: Translate) {
  if (uv === null || uv === undefined) return undefined
  if (uv < 3) return t("weather.uv.low")
  if (uv < 6) return t("weather.uv.moderate")
  if (uv < 8) return t("weather.uv.high")
  if (uv < 11) return t("weather.uv.veryHigh")
  return t("weather.uv.extreme")
}
