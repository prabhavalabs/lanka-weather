import { ContainerIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { ProgressBar } from "@/components/ProgressBar"
import { StatValue } from "@/components/StatValue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { cn } from "@/lib/utils"
import {
  flowNote,
  flowShort,
  formatReadingDay,
  readingDayAge,
  reservoirStatusLabels,
  reservoirStatusMeanings,
  reservoirSubtitle,
  storageView,
} from "./present"
import { useReservoirHistory } from "./queries"
import { StorageChart } from "./StorageChart"
import type { ReservoirProperties } from "./types"

const historyDays = 60

/** The small print every reservoir panel carries. */
export function ReservoirSources({ reservoir }: { reservoir?: ReservoirProperties }) {
  return (
    <div className="flex flex-col gap-1">
      <p>
        Source: {reservoir?.agency ?? "Irrigation Department and Mahaweli Authority"}, compiled by{" "}
        <a
          href={reservoir?.sourceUrl ?? "https://waterlevels.lk/reservoirs"}
          target="_blank"
          rel="noreferrer"
          className="underline underline-offset-2 hover:text-foreground"
        >
          Sri Lanka Water Levels (waterlevels.lk)
        </a>
        .{reservoir?.mapped && " Position © OpenStreetMap contributors."}
      </p>
      <p>Not an official warning. In an emergency call 117.</p>
    </div>
  )
}

interface ReservoirPanelProps {
  reservoir: ReservoirProperties
  onClose: () => void
  className?: string
}

/** One reservoir: status in the source's words, storage, flow, age and history. */
export function ReservoirPanel({ reservoir, onClose, className }: ReservoirPanelProps) {
  const history = useReservoirHistory(reservoir.id, historyDays)
  const storage = storageView(reservoir)
  const note = reservoir.flow ? flowNote(reservoir.flow) : null

  return (
    <Panel
      icon={<ContainerIcon />}
      title={reservoir.name}
      subtitle={reservoirSubtitle(reservoir)}
      className={cn("w-full", className)}
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label="Close reservoir details"
        >
          <XIcon />
        </Button>
      }
      footer={<ReservoirSources reservoir={reservoir} />}
    >
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Badge variant={reservoir.status}>{reservoirStatusLabels[reservoir.status]}</Badge>
          {reservoir.stale && (
            <Badge variant="chip" className="text-foreground">
              Out of date
            </Badge>
          )}
        </div>
        <p className="text-sm text-muted-foreground">{reservoirStatusMeanings[reservoir.status]}</p>
        <StatValue
          label="Storage"
          value={storage.value}
          unit={storage.value === "–" ? undefined : "%"}
          caption={storage.caption}
        />
        {storage.current && reservoir.storagePercent !== null && (
          <ProgressBar
            value={Math.min(1, reservoir.storagePercent / 100)}
            label="Storage as a share of capacity"
          />
        )}
      </div>

      <LabeledRows>
        {reservoir.river && <LabeledRow label="River" value={reservoir.river} numeric={false} />}
        <LabeledRow
          label="Flow"
          value={reservoir.flow ? flowShort(reservoir.flow) : "Not reported"}
          numeric={Boolean(reservoir.flow)}
        />
        {reservoir.flow?.agency && (
          <LabeledRow label="Flow reported by" value={reservoir.flow.agency} numeric={false} />
        )}
        <LabeledRow
          label="Reading day"
          value={
            reservoir.readingDay
              ? `${formatReadingDay(reservoir.readingDay)} · ${readingDayAge(reservoir.readingDay)}`
              : "Not known"
          }
          tone={reservoir.stale ? "warning" : "default"}
        />
        {reservoir.agency && (
          <LabeledRow label="Reported by" value={reservoir.agency} numeric={false} />
        )}
        {!reservoir.mapped && (
          <LabeledRow label="On the map" value="Location not mapped" numeric={false} />
        )}
      </LabeledRows>
      {note && <p className="text-xs text-muted-foreground">{note}</p>}

      <div>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground">
          Storage, last {historyDays} days
        </h3>
        {history.isPending && (
          <Skeleton
            className="h-32 w-full rounded-2xl"
            role="status"
            aria-label="Loading history"
          />
        )}
        {history.error && (
          <p
            className="rounded-2xl border border-border bg-inset py-6 text-center text-xs text-muted-foreground"
            role="alert"
          >
            History is unavailable.
          </p>
        )}
        {history.data && <StorageChart readings={history.data.readings} days={historyDays} />}
      </div>
    </Panel>
  )
}
