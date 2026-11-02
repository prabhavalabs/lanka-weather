import { ActivityIcon, ExternalLinkIcon, InfoIcon, XIcon } from "lucide-react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { StatValue } from "@/components/StatValue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { formatAge, formatDayTime, formatNumber } from "@/lib/format"
import { formatDistance, tsunamiNote } from "./present"
import type { Earthquake } from "./types"

interface EarthquakePanelProps {
  event: Earthquake
  stale?: boolean
  onClose: () => void
}

/** One earthquake: magnitude, place, time, depth and distance from Sri Lanka. */
export function EarthquakePanel({ event, stale = false, onClose }: EarthquakePanelProps) {
  return (
    <Panel
      icon={<ActivityIcon />}
      title={event.place || "Earthquake"}
      subtitle="Earthquake · USGS"
      className="w-full"
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label="Close earthquake details"
        >
          <XIcon />
        </Button>
      }
      footer={
        <p>
          Source: U.S. Geological Survey. Revised{" "}
          <span className="numeric">{formatAge(event.updatedAt)}</span>, fetched{" "}
          <span className="numeric">{formatAge(event.fetchedAt)}</span>.
        </p>
      }
    >
      <div className="flex items-start justify-between gap-3">
        <StatValue
          value={formatNumber(event.magnitude, 1)}
          unit={event.magType}
          caption="Magnitude"
        />
        <div className="flex flex-col items-end gap-1.5">
          <Badge variant="chip" className="text-foreground capitalize">
            {event.status}
          </Badge>
          {stale && (
            <Badge variant="chip" className="text-foreground">
              Out of date
            </Badge>
          )}
        </div>
      </div>
      {event.tsunami && (
        <div
          role="note"
          className="flex gap-2.5 rounded-xl border-2 border-status-major-flood bg-inset px-3 py-2.5 text-sm text-foreground"
        >
          <InfoIcon className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <p>
            {tsunamiNote}{" "}
            <a
              className="underline underline-offset-2"
              href="https://www.dmc.gov.lk/"
              target="_blank"
              rel="noreferrer"
            >
              dmc.gov.lk
            </a>
          </p>
        </div>
      )}
      <LabeledRows>
        <LabeledRow
          label="Time"
          value={`${formatDayTime(event.time)} · ${formatAge(event.time)}`}
        />
        <LabeledRow label="Depth" value={`${formatNumber(event.depthKm)} km`} />
        <LabeledRow label="From Sri Lanka's coast" value={formatDistance(event.distanceKm)} />
        {event.alert && <LabeledRow label="USGS PAGER alert" value={event.alert} numeric={false} />}
      </LabeledRows>
      {event.url && (
        <a
          href={event.url}
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1.5 self-start rounded-sm text-sm font-medium text-brand-text underline underline-offset-2 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
        >
          USGS event page
          <ExternalLinkIcon className="size-3.5" aria-hidden="true" />
          <span className="sr-only">(opens in a new tab)</span>
        </a>
      )}
      <p className="text-xs text-muted-foreground">
        Lanka Weather generates no tsunami warnings. Official warnings come from the DMC and the
        Department of Meteorology.
      </p>
    </Panel>
  )
}
