import { WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { LegendSwatch } from "@/components/LegendSwatch"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { formatAge, formatDayTime } from "@/lib/format"
import { cn } from "@/lib/utils"
import {
  floodAreaLabels,
  floodSeverityBadge,
  floodSeverityLabels,
  floodTrendArrows,
  floodTrendLabels,
  formatWindow,
  gaugeName,
} from "./present"
import type { FloodAreaFeature, FloodGaugeProperties } from "./types"

interface FloodGaugePanelProps {
  gauge: FloodGaugeProperties
  /** Forecast flood areas belonging to this gauge. */
  areas?: FloodAreaFeature[]
  onClose: () => void
  className?: string
}

function formatThreshold(value: number | null, unit: string) {
  if (value === null) return "–"
  const shown = unit === "m3/s" ? "m³/s" : unit
  return `${value.toFixed(value >= 100 ? 0 : 1)} ${shown}`
}

/** One Google flood forecast gauge: its forecast severity and trend, window and thresholds. */
export function FloodGaugePanel({ gauge, areas = [], onClose, className }: FloodGaugePanelProps) {
  const thresholds = gauge.thresholds
  return (
    <Panel
      icon={<WavesIcon />}
      title={gaugeName(gauge)}
      subtitle={`Flood forecast gauge · ${gauge.gaugeSource || "Google"}`}
      className={cn("w-full", className)}
      action={
        <Button variant="outline" size="icon" onClick={onClose} aria-label="Close gauge details">
          <XIcon />
        </Button>
      }
      footer={
        <p>
          Google Flood Hub (Flood Forecasting API). A forecast, not an official warning. Official
          flood warnings come from the Irrigation Department and the DMC.
        </p>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={floodSeverityBadge[gauge.severity]}>
          {floodSeverityLabels[gauge.severity]}
        </Badge>
        <Badge variant="chip" className="text-foreground">
          {floodTrendArrows[gauge.trend]} {floodTrendLabels[gauge.trend]}
        </Badge>
        {gauge.stale && (
          <Badge variant="chip" className="text-foreground">
            Out of date
          </Badge>
        )}
        {!gauge.qualityVerified && (
          <Badge variant="chip" className="text-foreground">
            Lower confidence
          </Badge>
        )}
      </div>

      {!gauge.qualityVerified && (
        <p className="rounded-xl border border-border bg-inset px-3 py-2 text-xs text-foreground">
          Google has not quality-verified this gauge&apos;s model, and asks that it be used with
          caution. Flood Hub calls this Extended coverage. It does not raise river risk, alerts or
          the overview here.
        </p>
      )}

      <LabeledRows>
        <LabeledRow
          label="Forecast for"
          value={formatWindow(gauge.forecastStart, gauge.forecastEnd)}
        />
        <LabeledRow
          label="Issued"
          value={`${formatDayTime(gauge.issuedAt)} · ${formatAge(gauge.issuedAt)}`}
          tone={gauge.stale ? "warning" : "default"}
        />
        <LabeledRow
          label="Quality"
          value={gauge.qualityVerified ? "Verified by Google" : "Lower confidence, not verified"}
          numeric={false}
        />
      </LabeledRows>

      {thresholds && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">Gauge model thresholds</h3>
          <LabeledRows>
            <LabeledRow
              leading={<LegendSwatch color="var(--flood-warning)" />}
              label="Warning"
              value={formatThreshold(thresholds.warning, thresholds.unit)}
            />
            <LabeledRow
              leading={<LegendSwatch color="var(--flood-danger)" />}
              label="Danger"
              value={formatThreshold(thresholds.danger, thresholds.unit)}
            />
            <LabeledRow
              leading={<LegendSwatch color="var(--flood-extreme)" />}
              label="Extreme"
              value={formatThreshold(thresholds.extremeDanger, thresholds.unit)}
            />
          </LabeledRows>
          <p className="text-xs text-muted-foreground">
            Severity is Google&apos;s forecast of the flow against these levels in the forecast
            window.
          </p>
        </section>
      )}

      {areas.length > 0 && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">Forecast flood areas</h3>
          <LabeledRows>
            {areas.map((area) => (
              <LabeledRow
                key={area.id}
                label={floodAreaLabels[area.properties.kind] ?? area.properties.kind}
                value={formatWindow(area.properties.validFrom, area.properties.validUntil)}
              />
            ))}
          </LabeledRows>
        </section>
      )}
    </Panel>
  )
}
