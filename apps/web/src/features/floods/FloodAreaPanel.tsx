import { WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { formatAge, formatDayTime } from "@/lib/format"
import { cn } from "@/lib/utils"
import { areaBadge, areaLikelihood, floodAreaLabels } from "./present"
import type { FloodAreaFeature } from "./types"

interface FloodAreaPanelProps {
  area: FloodAreaFeature
  seeded?: boolean
  onClose: () => void
  onSelectGauge: (id: string) => void
  className?: string
}

const kindNames: Record<FloodAreaFeature["properties"]["kind"], string> = {
  notification: "Riverine flood notification",
  inundation_high: "Inundation",
  inundation_medium: "Inundation",
  inundation_low: "Inundation",
  flash_likely: "Flash flood",
  flash_highly_likely: "Flash flood",
  significant_event: "Significant flood event",
}

/** One of Google's flood polygons: what kind it is, how likely, and when it applies. */
export function FloodAreaPanel({
  area,
  seeded,
  onClose,
  onSelectGauge,
  className,
}: FloodAreaPanelProps) {
  const { kind, validFrom, validUntil, issuedAt, gaugeId, fetchedAt, lowerConfidence } =
    area.properties
  const likelihood = areaLikelihood(area.properties)
  return (
    <Panel
      icon={<WavesIcon />}
      title={floodAreaLabels[kind] ?? "Flood area"}
      subtitle="Google flood forecast"
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label="Close flood area details"
        >
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
        <Badge variant={areaBadge(area.properties)}>{likelihood.label}</Badge>
        {lowerConfidence && (
          <Badge variant="chip" className="text-foreground">
            Lower confidence
          </Badge>
        )}
        {seeded && (
          <Badge variant="destructive" className="uppercase">
            Test data
          </Badge>
        )}
      </div>
      <LabeledRows>
        <LabeledRow label="Kind" value={kindNames[kind] ?? "Flood area"} numeric={false} />
        <LabeledRow label="Likelihood" value={likelihood.label} numeric={false} />
        <LabeledRow
          label="Forecast from"
          value={validFrom ? formatDayTime(validFrom) : "Not given"}
        />
        <LabeledRow
          label="Forecast until"
          value={validUntil ? formatDayTime(validUntil) : "Not given"}
        />
        {issuedAt && (
          <LabeledRow
            label="Issued"
            value={`${formatDayTime(issuedAt)} · ${formatAge(issuedAt)}`}
          />
        )}
        {fetchedAt && <LabeledRow label="Read" value={formatAge(fetchedAt)} />}
      </LabeledRows>
      {likelihood.note && <p className="text-xs text-muted-foreground">{likelihood.note}</p>}
      {gaugeId && (
        <Button
          variant="secondary"
          size="sm"
          className="w-fit"
          onClick={() => onSelectGauge(gaugeId)}
        >
          Open its forecast gauge
        </Button>
      )}
    </Panel>
  )
}
