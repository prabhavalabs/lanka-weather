import { RadioTowerIcon, TriangleAlertIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { StatValue } from "@/components/StatValue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { formatAge, formatDay, formatDayTime, formatNumber } from "@/lib/format"
import { cn } from "@/lib/utils"
import {
  feedNotice,
  formatRain,
  formatTemperature,
  markerTemperature,
  rainLabel,
  readingLabel,
} from "./present"
import { useMetStation } from "./queries"
import { TemperatureChart } from "./TemperatureChart"

interface MetStationPanelProps {
  id: string
  onClose: () => void
  className?: string
}

const historyDays = 7

/** One Department of Meteorology station: its latest reading and history. All observed. */
export function MetStationPanel({ id, onClose, className }: MetStationPanelProps) {
  const detail = useMetStation(id)
  const station = detail.data?.station.properties
  const meta = detail.data?.meta
  const notice = meta ? feedNotice(meta) : null
  const shown = station ? markerTemperature(station) : null
  const isMax = station?.temperatureC === null && station?.tempMaxC !== null

  return (
    <Panel
      icon={<RadioTowerIcon />}
      title={station?.name ?? (detail.error ? "Weather station" : "Loading…")}
      subtitle={
        station?.wmoId
          ? `Met station · WMO ${station.wmoId}`
          : station && !station.reportsTemperature
            ? "Met rain station"
            : "Met station"
      }
      className={cn("w-full", className)}
      action={
        <Button variant="outline" size="icon" onClick={onClose} aria-label="Close weather station">
          <XIcon />
        </Button>
      }
      footer={
        meta && (
          <p>
            Source:{" "}
            <a
              href={meta.sourceUrl}
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-2 hover:text-foreground"
            >
              Department of Meteorology, Sri Lanka
            </a>
            , through the{" "}
            <a
              href="https://github.com/nuuuwan/lk_weather_3h"
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-2 hover:text-foreground"
            >
              lk_weather_3h
            </a>{" "}
            and{" "}
            <a
              href="https://github.com/nuuuwan/weather_lk"
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-2 hover:text-foreground"
            >
              weather_lk
            </a>{" "}
            copies.
          </p>
        )
      }
    >
      {detail.isPending && (
        <div className="flex flex-col gap-3" role="status" aria-label="Loading station">
          <Skeleton className="h-10 w-36" />
          <Skeleton className="h-40 w-full rounded-2xl" />
        </div>
      )}
      {detail.error && !detail.data && (
        <p
          className="rounded-2xl border border-border bg-inset px-4 py-6 text-center text-sm"
          role="alert"
        >
          This station could not be loaded.
        </p>
      )}
      {station && (
        <>
          <div className="flex items-center gap-2">
            <Badge variant="chip" className="text-foreground">
              Observed
            </Badge>
            {station.stale && (
              <Badge variant="chip" className="text-foreground">
                Out of date
              </Badge>
            )}
          </div>
          {notice && (
            <p
              role="status"
              className="flex items-start gap-2 rounded-xl border border-border bg-inset px-3 py-2.5 text-sm text-foreground"
            >
              <TriangleAlertIcon
                className="mt-0.5 size-4 shrink-0 text-brand-text"
                aria-hidden="true"
              />
              {notice}
            </p>
          )}
          {shown !== null && (
            <StatValue
              label={isMax ? "Maximum, daily report" : "Temperature"}
              value={formatNumber(shown, 1)}
              unit="°C"
              caption={
                station.observedAt
                  ? `${formatDayTime(station.observedAt)} · ${formatAge(station.observedAt)}`
                  : undefined
              }
            />
          )}
          <LabeledRows>
            <LabeledRow
              label="Observed"
              value={`${formatAge(station.observedAt)}${station.stale ? " · out of date" : ""}`}
              tone={station.stale ? "warning" : "default"}
            />
            <LabeledRow label="Reading" value={readingLabel(station)} numeric={false} />
            {station.rainMm !== null && (
              <LabeledRow label={rainLabel(station)} value={formatRain(station.rainMm)} />
            )}
            {station.rain24hMm !== null && station.rainHours !== 24 && (
              <LabeledRow label="Rain, 24 h" value={formatRain(station.rain24hMm)} />
            )}
            {station.humidityPct !== null && (
              <LabeledRow label="Humidity" value={`${Math.round(station.humidityPct)}%`} />
            )}
            {station.tempMaxC !== null && (
              <LabeledRow label="Maximum" value={formatTemperature(station.tempMaxC)} />
            )}
            {station.tempMinC !== null && (
              <LabeledRow label="Minimum" value={formatTemperature(station.tempMinC)} />
            )}
            {station.elevationM !== null && (
              <LabeledRow label="Elevation" value={`${Math.round(station.elevationM)} m`} />
            )}
            {station.positionApproximate && (
              <LabeledRow label="Position" value="Approximate, from its name" numeric={false} />
            )}
          </LabeledRows>
          {detail.data && detail.data.threeHourly.length > 0 && (
            <div>
              <h3 className="mb-2 text-xs font-medium text-muted-foreground">
                {historyDays} days to the newest report
              </h3>
              <TemperatureChart readings={detail.data.threeHourly} days={historyDays} />
            </div>
          )}
          {detail.data && detail.data.daily.length > 0 && (
            <div>
              <h3 className="mb-1 text-xs font-medium text-muted-foreground">Daily reports</h3>
              <LabeledRows>
                {detail.data.daily
                  .slice(-5)
                  .reverse()
                  .map((day) => (
                    <LabeledRow
                      key={day.date}
                      label={formatDay(day.observedAt)}
                      value={`${formatRain(day.rainMm)}${
                        day.tempMaxC !== null && day.tempMinC !== null
                          ? ` · ${formatNumber(day.tempMinC)}–${formatNumber(day.tempMaxC)}°`
                          : ""
                      }`}
                    />
                  ))}
              </LabeledRows>
            </div>
          )}
        </>
      )}
    </Panel>
  )
}
