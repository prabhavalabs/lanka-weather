import { ContainerIcon, XIcon } from "lucide-react"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { formatAge } from "@/lib/format"
import {
  activeReservoirs,
  flowShort,
  readingDayAge,
  reservoirStatusLabels,
  storageView,
} from "./present"
import { ReservoirSources } from "./ReservoirPanel"
import type { ReservoirFeature, ReservoirsCollection } from "./types"

interface ReservoirListPanelProps {
  data: ReservoirsCollection | undefined
  pending: boolean
  /** No figure is current, so nothing can be said about today. */
  stale: boolean
  onSelect: (id: string) => void
  onClose: () => void
}

function item(feature: ReservoirFeature, onSelect: (id: string) => void): GaugeListItem {
  const { id, name, district, status, flow, readingDay, mapped, stale } = feature.properties
  const storage = storageView(feature.properties)
  return {
    id,
    title: name,
    subtitle: [
      district,
      flow ? flowShort(flow) : null,
      readingDay ? readingDayAge(readingDay) : null,
      stale ? "out of date" : null,
      mapped ? null : "not on the map",
    ]
      .filter(Boolean)
      .join(" · "),
    value: storage.value === "–" ? undefined : `${storage.value}%`,
    badge: { variant: status, label: reservoirStatusLabels[status] },
    onSelect: () => onSelect(id),
  }
}

/**
 * Reservoirs spilling or releasing, with a calm line when there are none, and
 * every other reservoir below, so those with no place on the map can be reached.
 */
export function ReservoirListPanel({
  data,
  pending,
  stale,
  onSelect,
  onClose,
}: ReservoirListPanelProps) {
  const features = data?.features ?? []
  const active = activeReservoirs(features)
  const activeIds = new Set(active.map((feature) => feature.id))
  const others = features.filter((feature) => !activeIds.has(feature.id))

  return (
    <Panel
      icon={<ContainerIcon />}
      title="Reservoirs spilling or releasing"
      subtitle={
        data ? `Daily figures · fetched ${formatAge(data.meta.fetchedAt)}` : "Daily figures"
      }
      className="w-full"
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label="Close the reservoir list"
        >
          <XIcon />
        </Button>
      }
      footer={<ReservoirSources />}
    >
      {stale && (
        <div className="flex items-center gap-2">
          <Badge variant="chip" className="text-foreground">
            Out of date
          </Badge>
          <p className="text-xs text-muted-foreground">
            These figures have not been updated recently.
          </p>
        </div>
      )}
      {pending && (
        <Skeleton
          className="h-32 w-full rounded-2xl"
          role="status"
          aria-label="Loading reservoirs"
        />
      )}
      {data && (
        <GaugeList
          title={`Now (${active.length})`}
          items={active.map((feature) => item(feature, onSelect))}
          emptyText={
            features.length === 0
              ? "Reservoir figures have not been collected yet."
              : stale
                ? "No current figures to say whether any reservoir is spilling or releasing."
                : "None spilling or releasing on the latest readings."
          }
        />
      )}
      {data && others.length > 0 && (
        <details className="group rounded-xl">
          <summary className="cursor-pointer rounded-lg py-1 text-xs font-medium text-muted-foreground hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none">
            All other reservoirs ({others.length}), including those not on the map
          </summary>
          <GaugeList
            className="mt-2"
            title="Holding, no data or out of date"
            items={others.map((feature) => item(feature, onSelect))}
          />
        </details>
      )}
    </Panel>
  )
}
