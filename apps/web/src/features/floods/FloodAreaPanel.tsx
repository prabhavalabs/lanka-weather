import { WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { formatAge, formatDayTime } from "@/lib/format"
import { type MessageKey, useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"
import { areaBadge, areaLikelihood, floodAreaLabel } from "./present"
import type { FloodAreaFeature } from "./types"

interface FloodAreaPanelProps {
  area: FloodAreaFeature
  seeded?: boolean
  onClose: () => void
  onSelectGauge: (id: string) => void
  className?: string
}

const kindNames: Record<FloodAreaFeature["properties"]["kind"], MessageKey> = {
  notification: "floods.areaKind.notification",
  inundation_high: "floods.areaKind.inundation",
  inundation_medium: "floods.areaKind.inundation",
  inundation_low: "floods.areaKind.inundation",
  flash_likely: "floods.areaKind.flash",
  flash_highly_likely: "floods.areaKind.flash",
  significant_event: "floods.areaKind.significant_event",
}

/** One of Google's flood polygons: what kind it is, how likely, and when it applies. */
export function FloodAreaPanel({
  area,
  seeded,
  onClose,
  onSelectGauge,
  className,
}: FloodAreaPanelProps) {
  const t = useT()
  const { kind, validFrom, validUntil, issuedAt, gaugeId, fetchedAt, lowerConfidence } =
    area.properties
  const likelihood = areaLikelihood(area.properties)
  return (
    <Panel
      icon={<WavesIcon />}
      title={floodAreaLabel(kind)}
      subtitle={t("floods.areaPanel.subtitle")}
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("floods.areaPanel.close")}
        >
          <XIcon />
        </Button>
      }
      footer={<p>{t("floods.forecastFooter")}</p>}
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={areaBadge(area.properties)}>{likelihood.label}</Badge>
        {lowerConfidence && (
          <Badge variant="chip" className="text-foreground">
            {t("floods.lowerConfidence")}
          </Badge>
        )}
        {seeded && (
          <Badge variant="destructive" className="uppercase">
            {t("floods.testData")}
          </Badge>
        )}
      </div>
      <LabeledRows>
        <LabeledRow
          label={t("floods.areaPanel.kind")}
          value={t(kindNames[kind] ?? "floods.area.fallback")}
          numeric={false}
        />
        <LabeledRow
          label={t("floods.areaPanel.likelihood")}
          value={likelihood.label}
          numeric={false}
        />
        <LabeledRow
          label={t("floods.areaPanel.from")}
          value={validFrom ? formatDayTime(validFrom) : t("common.notGiven")}
        />
        <LabeledRow
          label={t("floods.areaPanel.until")}
          value={validUntil ? formatDayTime(validUntil) : t("common.notGiven")}
        />
        {issuedAt && (
          <LabeledRow
            label={t("common.issued")}
            value={`${formatDayTime(issuedAt)}${t("common.separator")}${formatAge(issuedAt)}`}
          />
        )}
        {fetchedAt && (
          <LabeledRow label={t("floods.areaPanel.read")} value={formatAge(fetchedAt)} />
        )}
      </LabeledRows>
      {likelihood.note && <p className="text-xs text-muted-foreground">{likelihood.note}</p>}
      {gaugeId && (
        <Button
          variant="secondary"
          size="sm"
          className="w-fit"
          onClick={() => onSelectGauge(gaugeId)}
        >
          {t("floods.areaPanel.openGauge")}
        </Button>
      )}
    </Panel>
  )
}
