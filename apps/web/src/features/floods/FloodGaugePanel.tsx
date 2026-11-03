import { WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { LegendSwatch } from "@/components/LegendSwatch"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { formatAge, formatDayTime, formatNumber } from "@/lib/format"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import {
  floodAreaLabel,
  floodSeverityBadge,
  floodSeverityLabel,
  floodTrendArrows,
  floodTrendLabel,
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
  return `${formatNumber(value, value >= 100 ? 0 : 1)} ${shown}`
}

/** One Google flood forecast gauge: its forecast severity and trend, window and thresholds. */
export function FloodGaugePanel({ gauge, areas = [], onClose, className }: FloodGaugePanelProps) {
  const t = useT()
  const thresholds = gauge.thresholds
  return (
    <Panel
      icon={<WavesIcon />}
      title={gaugeName(gauge)}
      subtitle={t("floods.gauge.subtitle", { source: gauge.gaugeSource || "Google" })}
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("floods.gauge.close")}
        >
          <XIcon />
        </Button>
      }
      footer={<p>{t("floods.forecastFooter")}</p>}
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={floodSeverityBadge[gauge.severity]}>
          {floodSeverityLabel(gauge.severity)}
        </Badge>
        <Badge variant="chip" className="text-foreground">
          {floodTrendArrows[gauge.trend]} {floodTrendLabel(gauge.trend)}
        </Badge>
        {gauge.stale && (
          <Badge variant="chip" className="text-foreground">
            {t("common.outOfDate")}
          </Badge>
        )}
        {!gauge.qualityVerified && (
          <Badge variant="chip" className="text-foreground">
            {t("floods.lowerConfidence")}
          </Badge>
        )}
      </div>

      {!gauge.qualityVerified && (
        <p className="rounded-xl border border-border bg-inset px-3 py-2 text-xs text-foreground">
          {t("floods.gauge.lowerConfidenceHelp")}
        </p>
      )}

      <LabeledRows>
        <LabeledRow
          label={t("floods.gauge.forecastFor")}
          value={formatWindow(gauge.forecastStart, gauge.forecastEnd)}
        />
        <LabeledRow
          label={t("common.issued")}
          value={`${formatDayTime(gauge.issuedAt)}${t("common.separator")}${formatAge(gauge.issuedAt)}`}
          tone={gauge.stale ? "warning" : "default"}
        />
        <LabeledRow
          label={t("floods.gauge.quality")}
          value={gauge.qualityVerified ? t("floods.gauge.verified") : t("floods.gauge.notVerified")}
          numeric={false}
        />
      </LabeledRows>

      {thresholds && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">
            {t("floods.gauge.thresholds")}
          </h3>
          <LabeledRows>
            <LabeledRow
              leading={<LegendSwatch color="var(--flood-warning)" />}
              label={floodSeverityLabel("above_normal")}
              value={formatThreshold(thresholds.warning, thresholds.unit)}
            />
            <LabeledRow
              leading={<LegendSwatch color="var(--flood-danger)" />}
              label={floodSeverityLabel("severe")}
              value={formatThreshold(thresholds.danger, thresholds.unit)}
            />
            <LabeledRow
              leading={<LegendSwatch color="var(--flood-extreme)" />}
              label={floodSeverityLabel("extreme")}
              value={formatThreshold(thresholds.extremeDanger, thresholds.unit)}
            />
          </LabeledRows>
          <p className="text-xs text-muted-foreground">{t("floods.gauge.thresholdsHelp")}</p>
        </section>
      )}

      {areas.length > 0 && (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-xs font-medium text-muted-foreground">{t("floods.gauge.areas")}</h3>
          <LabeledRows>
            {areas.map((area) => (
              <LabeledRow
                key={area.id}
                label={floodAreaLabel(area.properties.kind)}
                value={formatWindow(area.properties.validFrom, area.properties.validUntil)}
              />
            ))}
          </LabeledRows>
        </section>
      )}
    </Panel>
  )
}
