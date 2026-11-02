import { ChevronRightIcon, RadioTowerIcon } from "lucide-react"
import type * as React from "react"
import { LegendSwatch } from "@/components/LegendSwatch"
import { Badge } from "@/components/ui/badge"
import { compassPoint, formatAge, formatNumber } from "@/lib/format"
import { aqiBand } from "../environment/present"
import type { AirQualityPoint, NearbyObservation, SeaPoint } from "../location/types"
import { markerTemperature } from "./present"

function OutOfDate() {
  return (
    <Badge variant="chip" className="h-5 px-1.5 text-[0.6875rem] text-foreground">
      Out of date
    </Badge>
  )
}

/** A label and value with a smaller line of detail under them. */
function SummaryRow({
  label,
  leading,
  value,
  detail,
  stale,
}: {
  label: string
  leading?: React.ReactNode
  value: string
  detail: (string | false)[]
  stale: boolean
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-0.5 py-2.5 text-sm">
      <dt className="flex items-center gap-2 text-muted-foreground">
        {leading}
        {label}
      </dt>
      <dd className="numeric text-right text-foreground">{value}</dd>
      <dd className="numeric flex w-full flex-wrap items-center justify-end gap-x-1.5 text-right text-xs text-muted-foreground">
        {detail.filter(Boolean).join(" · ")}
        {stale && <OutOfDate />}
      </dd>
    </div>
  )
}

interface ObservedSummaryProps {
  observed: NearbyObservation | null | undefined
  air: AirQualityPoint | null | undefined
  sea: SeaPoint | null | undefined
  onSelectStation: (id: string) => void
}

/**
 * What was measured nearby, set against the modelled conditions above it, and
 * air quality and the sea state as compact rows.
 */
export function ObservedSummary({ observed, air, sea, onSelectStation }: ObservedSummaryProps) {
  if (!observed && !air && !sea) return null
  const shown = observed ? markerTemperature(observed) : null
  return (
    <div className="flex flex-col gap-2">
      {observed && (
        <button
          type="button"
          onClick={() => onSelectStation(observed.id)}
          className="flex items-center gap-2.5 rounded-xl border border-border bg-inset px-3 py-2.5 text-left outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring"
        >
          <RadioTowerIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <span className="min-w-0 flex-1">
            <span className="flex items-center gap-1.5 text-sm text-foreground">
              <span className="font-medium">Observed</span>
              <span className="truncate">{observed.name}</span>
              <span className="numeric shrink-0 text-xs text-muted-foreground">
                {observed.distanceKm < 1 ? "under 1 km" : `${observed.distanceKm.toFixed(0)} km`}
              </span>
            </span>
            <span className="numeric block truncate text-xs text-muted-foreground">
              {[
                shown !== null && `${formatNumber(shown, 1)} °C`,
                observed.temperatureC === null && shown !== null && "max",
                observed.humidityPct !== null && `${Math.round(observed.humidityPct)}%`,
                observed.rainMm !== null &&
                  `${formatNumber(observed.rainMm, 1)} mm in ${observed.rainHours} h`,
              ]
                .filter(Boolean)
                .join(" · ")}
            </span>
            <span className="mt-0.5 flex items-center gap-1.5 text-xs text-muted-foreground">
              <span className="numeric">{formatAge(observed.observedAt)}</span>
              {observed.stale && <OutOfDate />}
            </span>
          </span>
          <ChevronRightIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
        </button>
      )}
      {(air || sea) && (
        <dl className="flex flex-col divide-y divide-border">
          {air && (
            <SummaryRow
              label="Air quality"
              leading={<LegendSwatch color={`var(${aqiBand(air.category).token})`} />}
              value={`${aqiBand(air.category).label} · AQI ${Math.round(air.usAqi)}`}
              detail={[
                "Estimate",
                air.pm25 !== null && `PM2.5 ${formatNumber(air.pm25, 1)} µg/m³`,
                formatAge(air.fetchedAt),
              ]}
              stale={air.stale}
            />
          )}
          {sea && (
            <SummaryRow
              label="Sea"
              value={`${sea.state[0]?.toUpperCase()}${sea.state.slice(1)} · waves ${formatNumber(sea.waveHeightM, 1)} m`}
              detail={[
                "Estimate",
                sea.swellHeightM !== null &&
                  [
                    `swell ${formatNumber(sea.swellHeightM, 1)} m`,
                    sea.swellPeriodS !== null && `${Math.round(sea.swellPeriodS)} s`,
                    sea.swellDirectionDeg !== null && `from ${compassPoint(sea.swellDirectionDeg)}`,
                  ]
                    .filter(Boolean)
                    .join(" "),
                formatAge(sea.fetchedAt),
              ]}
              stale={sea.stale}
            />
          )}
        </dl>
      )}
      {sea && (
        <p className="text-xs text-muted-foreground">
          Sea values are from a wave model. Follow the Department of Meteorology&apos;s sea-area
          warnings.
        </p>
      )}
    </div>
  )
}
