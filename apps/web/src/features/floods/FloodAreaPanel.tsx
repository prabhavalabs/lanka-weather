import { WavesIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { formatDayTime } from "@/lib/format"
import { cn } from "@/lib/utils"
import { floodAreaLabels, floodSeverityBadge, floodSeverityLabels } from "./present"
import type { FloodAreaFeature } from "./types"

interface FloodAreaPanelProps {
  area: FloodAreaFeature
  seeded?: boolean
  onClose: () => void
  onSelectGauge: (id: string) => void
  className?: string
}

/** One of Google's flood polygons: what it means and when it applies. */
export function FloodAreaPanel({
  area,
  seeded,
  onClose,
  onSelectGauge,
  className,
}: FloodAreaPanelProps) {
  const { kind, severity, validFrom, validUntil, issuedAt, gaugeId } = area.properties
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
          Source: Google Flood Hub (Flood Forecasting API). Official flood warnings come from the
          Irrigation Department and the DMC.
        </p>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={floodSeverityBadge[severity]}>{floodSeverityLabels[severity]}</Badge>
        {seeded && (
          <Badge variant="destructive" className="uppercase">
            Test data
          </Badge>
        )}
      </div>
      <LabeledRows>
        <LabeledRow label="Valid from" value={validFrom ? formatDayTime(validFrom) : "–"} />
        <LabeledRow label="Valid until" value={validUntil ? formatDayTime(validUntil) : "–"} />
        {issuedAt && <LabeledRow label="Issued" value={formatDayTime(issuedAt)} />}
      </LabeledRows>
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
