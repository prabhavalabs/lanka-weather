import { ChevronRightIcon, RadioTowerIcon } from "lucide-react"
import type * as React from "react"
import { LegendSwatch } from "@/components/LegendSwatch"
import { Badge } from "@/components/ui/badge"
import { compassPoint, formatAge, formatNumber } from "@/lib/format"
import { useT, type Translate } from "@/lib/i18n"
import { aqiBand, aqiLabel, seaStateLabel } from "../environment/present"
import type { AirQualityPoint, NearbyObservation, SeaPoint } from "../location/types"
import { markerTemperature } from "./present"

function OutOfDate() {
  const t = useT()
  return (
    <Badge variant="chip" className="h-5 px-1.5 text-[0.6875rem] text-foreground">
      {t("common.outOfDate")}
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
/** The swell in words: height, then period and direction where known. */
function swellText(t: Translate, sea: SeaPoint): string | false {
  if (sea.swellHeightM === null) return false
  const values = {
    height: formatNumber(sea.swellHeightM, 1),
    period: sea.swellPeriodS === null ? "" : String(Math.round(sea.swellPeriodS)),
    direction: sea.swellDirectionDeg === null ? "" : compassPoint(sea.swellDirectionDeg),
  }
  const period = sea.swellPeriodS !== null
  const from = sea.swellDirectionDeg !== null
  if (period && from) return t("stations.summary.swellPeriodFrom", values)
  if (period) return t("stations.summary.swellPeriod", values)
  if (from) return t("stations.summary.swellFrom", values)
  return t("stations.summary.swell", values)
}

export function ObservedSummary({ observed, air, sea, onSelectStation }: ObservedSummaryProps) {
  const t = useT()
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
            <span className="flex min-w-0 items-center gap-1.5 text-sm text-foreground">
              <span className="shrink-0 font-medium">{t("stations.summary.observed")}</span>
              <span className="min-w-0 truncate" title={observed.name}>
                {observed.name}
              </span>
              <span className="numeric shrink-0 text-xs text-muted-foreground">
                {observed.distanceKm < 1
                  ? t("stations.summary.underKm")
                  : t("stations.summary.km", { km: observed.distanceKm.toFixed(0) })}
              </span>
            </span>
            <span className="numeric block truncate text-xs text-muted-foreground">
              {[
                shown !== null && `${formatNumber(shown, 1)} °C`,
                observed.temperatureC === null && shown !== null && t("stations.summary.dailyMax"),
                observed.humidityPct !== null && `${Math.round(observed.humidityPct)}%`,
                observed.rainMm !== null &&
                  t("stations.summary.rainIn", {
                    rain: formatNumber(observed.rainMm, 1),
                    hours: observed.rainHours,
                  }),
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
              label={t("stations.summary.airQuality")}
              leading={<LegendSwatch color={`var(${aqiBand(air.category).token})`} />}
              value={t("stations.summary.aqiValue", {
                band: aqiLabel(air.category),
                aqi: Math.round(air.usAqi),
              })}
              detail={[
                t("terms.estimate"),
                air.pm25 !== null && `PM2.5 ${formatNumber(air.pm25, 1)} µg/m³`,
                formatAge(air.fetchedAt),
              ]}
              stale={air.stale}
            />
          )}
          {sea && (
            <SummaryRow
              label={t("stations.summary.sea")}
              value={t("stations.summary.seaValue", {
                state: seaStateLabel(sea.state, sea.waveHeightM),
                height: formatNumber(sea.waveHeightM, 1),
              })}
              detail={[t("terms.estimate"), swellText(t, sea), formatAge(sea.fetchedAt)]}
              stale={sea.stale}
            />
          )}
        </dl>
      )}
      {sea && <p className="text-xs text-muted-foreground">{t("stations.summary.seaNote")}</p>}
    </div>
  )
}
