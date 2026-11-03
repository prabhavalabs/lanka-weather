import { ContainerIcon, XIcon } from "lucide-react"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { formatAge } from "@/lib/format"
import { translate, useT } from "@/lib/i18n"
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
      stale ? translate("reservoirs.list.outOfDate") : null,
      mapped ? null : translate("reservoirs.list.notOnMap"),
    ]
      .filter(Boolean)
      .join(translate("common.separator")),
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
  const t = useT()
  const features = data?.features ?? []
  const active = activeReservoirs(features)
  const activeIds = new Set(active.map((feature) => feature.id))
  const others = features.filter((feature) => !activeIds.has(feature.id))

  return (
    <Panel
      icon={<ContainerIcon />}
      title={t("reservoirs.list.title")}
      subtitle={
        data
          ? t("reservoirs.list.subtitle", { age: formatAge(data.meta.fetchedAt) })
          : t("reservoirs.list.subtitleEmpty")
      }
      className="w-full"
      action={
        <Button
          variant="outline"
          size="icon"
          onClick={onClose}
          aria-label={t("reservoirs.list.close")}
        >
          <XIcon />
        </Button>
      }
      footer={<ReservoirSources />}
    >
      {stale && (
        <div className="flex items-center gap-2">
          <Badge variant="chip" className="shrink-0 text-foreground">
            {t("common.outOfDate")}
          </Badge>
          <p className="min-w-0 text-xs text-muted-foreground">{t("reservoirs.list.staleNote")}</p>
        </div>
      )}
      {pending && (
        <Skeleton
          className="h-32 w-full rounded-2xl"
          role="status"
          aria-label={t("reservoirs.list.loading")}
        />
      )}
      {data && (
        <GaugeList
          title={t("reservoirs.list.now", { count: active.length })}
          items={active.map((feature) => item(feature, onSelect))}
          emptyText={
            features.length === 0
              ? t("reservoirs.list.emptyNone")
              : stale
                ? t("reservoirs.list.emptyStale")
                : t("reservoirs.list.emptyCalm")
          }
        />
      )}
      {data && others.length > 0 && (
        <details className="group rounded-xl">
          <summary className="cursor-pointer rounded-lg py-1 text-xs font-medium text-muted-foreground hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none">
            {t("reservoirs.list.others", { count: others.length })}
          </summary>
          <GaugeList
            className="mt-2"
            title={t("reservoirs.list.othersTitle")}
            items={others.map((feature) => item(feature, onSelect))}
          />
        </details>
      )}
    </Panel>
  )
}
