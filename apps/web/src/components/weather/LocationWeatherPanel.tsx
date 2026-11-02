import * as React from "react"
import { MapPinIcon, XIcon } from "lucide-react"

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
import { ForecastStrip } from "./ForecastStrip"
import type { CurrentConditions, ForecastHour, Place, WeatherAlert, WindReading } from "./types"
import { WindCompass } from "./WindCompass"

interface LocationWeatherPanelProps {
  place: Place
  /** Null while loading. */
  current: CurrentConditions | null
  /** The next 48 hours, earliest first. */
  forecast: ForecastHour[]
  wind: WindReading | null
  alerts: WeatherAlert[]
  /** Source line for the footer, e.g. "Open-Meteo · ECMWF IFS". */
  source?: string
  onClose?: () => void
  className?: string
}

/**
 * Weather at a chosen place: current conditions, the next 48 hours, wind and
 * warnings. Presentational; the parent supplies the data.
 */
export function LocationWeatherPanel({
  place,
  current,
  forecast,
  wind,
  alerts,
  source,
  onClose,
  className,
}: LocationWeatherPanelProps) {
  const [hour, setHour] = React.useState(0)
  const selected = forecast[hour]

  return (
    <Panel
      icon={<MapPinIcon />}
      title={place.name}
      subtitle={place.area}
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
      footer={
        current && (
          <>
            <span>{source ?? "Forecast"}</span>
            <span className="numeric">
              <time dateTime={current.observedAt}>{formatDayTime(current.observedAt)}</time>
            </span>
          </>
        )
      }
    >
      {current ? (
        <div className="flex items-start gap-4">
          <StatValue
            className="flex-1"
            value={formatNumber(current.temperatureC, 1)}
            unit="°C"
            caption={`${current.summary} · feels like ${formatNumber(current.feelsLikeC)}°`}
          />
          <span className="flex size-14 items-center justify-center rounded-2xl border border-border bg-inset">
            <ConditionIcon
              condition={current.condition}
              className="size-7"
              title={current.summary}
            />
          </span>
        </div>
      ) : (
        <div className="flex flex-col gap-2" role="status" aria-label="Loading weather">
          <Skeleton className="h-9 w-32" />
          <Skeleton className="h-4 w-48" />
        </div>
      )}

      <Tabs defaultValue="forecast" className="gap-3">
        <TabsList className="w-full">
          <TabsTrigger value="forecast">48 hours</TabsTrigger>
          <TabsTrigger value="wind">Wind</TabsTrigger>
          <TabsTrigger value="alerts">
            Warnings
            {alerts.length > 0 && (
              <span className="numeric ml-0.5 rounded-full bg-brand px-1.5 text-[0.6875rem] leading-4 text-brand-foreground">
                {alerts.length}
              </span>
            )}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="forecast" className="flex flex-col gap-3">
          <ForecastStrip hours={forecast} selectedIndex={hour} onSelect={setHour} />
          {selected && (
            <LabeledRows>
              <LabeledRow
                label="Time"
                value={
                  hour === 0 ? `Now · ${formatClock(selected.time)}` : formatDayTime(selected.time)
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
            </LabeledRows>
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

        <TabsContent value="alerts">
          <AlertList alerts={alerts} />
        </TabsContent>
      </Tabs>

      {current && (
        <LabeledRows>
          <LabeledRow
            label="Rain, last 24 h"
            value={`${formatNumber(current.rainfall24hMm, 1)} mm`}
          />
          <LabeledRow label="Humidity" value={`${current.humidityPercent}%`} />
          <LabeledRow label="Pressure" value={`${formatNumber(current.pressureHpa)} hPa`} />
        </LabeledRows>
      )}
    </Panel>
  )
}
