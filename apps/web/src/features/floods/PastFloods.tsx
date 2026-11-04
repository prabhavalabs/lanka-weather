import { ScrollArea } from "@/components/ui/scroll-area"
import type { ExpressionSpecification, Map as MapLibreMap } from "maplibre-gl"
import { HistoryIcon, XIcon } from "lucide-react"
import { useEffect, useRef } from "react"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { Panel } from "@/components/Panel"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import type { Layer } from "@/lib/layers"
import { formatDay, formatInteger } from "@/lib/format"
import { useT } from "@/lib/i18n"
import { readTokens } from "@/lib/tokens"
import { cn } from "@/lib/utils"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import type { PastFlood, PastFloodsDocument } from "../river-risk/types"
import { pastFloodMagnitude, pastFloodName } from "./present"

const SOURCE = "past-floods"
/** The extents: what clicks and hovers hit. */
export const PAST_FLOOD_FILL = "past-flood-fill"
const LINE = "past-flood-line"
const SELECTED = "past-flood-selected"
const HATCH = "past-flood-hatch"

/** Diagonal stripes, so past extents read as a different kind of thing from a forecast's solid fill. */
function hatchImage(colour: string) {
  const size = 12
  const canvas = document.createElement("canvas")
  canvas.width = canvas.height = size
  const context = canvas.getContext("2d")!
  context.strokeStyle = colour
  context.lineWidth = 1.5
  context.beginPath()
  for (const offset of [-size, 0, size]) {
    context.moveTo(offset, size)
    context.lineTo(offset + size, 0)
  }
  context.stroke()
  return context.getImageData(0, 0, size, size)
}

function absolute(path: string) {
  return /^https?:/.test(path) ? path : `${window.location.origin}${path}`
}

function eventFilter(selected: number | null): ExpressionSpecification {
  return selected === null ? ["literal", true] : ["==", ["id"], selected]
}

function addLayers(map: MapLibreMap, layer: Layer, selected: number | null) {
  const tokens = readTokens({ past: "--past-flood", selected: "--status-stroke-selected" })
  if (map.hasImage(HATCH)) map.removeImage(HATCH)
  map.addImage(HATCH, hatchImage(tokens.past), { pixelRatio: 2 })
  if (!map.getSource(SOURCE)) {
    map.addSource(SOURCE, {
      type: "vector",
      tiles: [absolute(layer.tilePath ?? "/v1/tiles/past-floods/{z}/{x}/{y}.pbf")],
      minzoom: 0,
      maxzoom: layer.maxZoom ?? 14,
      attribution: "Past flood extents: Irrigation Department of Sri Lanka",
    })
  }
  const before = slotBefore(map, "past")
  if (!map.getLayer(PAST_FLOOD_FILL)) {
    map.addLayer(
      {
        id: PAST_FLOOD_FILL,
        type: "fill",
        source: SOURCE,
        "source-layer": "events",
        filter: eventFilter(selected),
        paint: { "fill-pattern": HATCH, "fill-opacity": 0.9 },
      },
      before,
    )
  }
  if (!map.getLayer(LINE)) {
    map.addLayer(
      {
        id: LINE,
        type: "line",
        source: SOURCE,
        "source-layer": "events",
        filter: eventFilter(selected),
        paint: {
          "line-color": tokens.past,
          "line-width": ["interpolate", ["linear"], ["zoom"], 7, 0.5, 12, 1.2],
          "line-opacity": 0.85,
        },
      },
      before,
    )
  }
  if (!map.getLayer(SELECTED)) {
    map.addLayer(
      {
        id: SELECTED,
        type: "line",
        source: SOURCE,
        "source-layer": "events",
        filter: ["==", ["id"], selected ?? -1],
        paint: { "line-color": tokens.selected, "line-width": 2.25 },
      },
      before,
    )
  }
}

function removeLayers(map: MapLibreMap) {
  if (!styleReady(map)) return
  for (const id of [SELECTED, LINE, PAST_FLOOD_FILL]) if (map.getLayer(id)) map.removeLayer(id)
  if (map.getSource(SOURCE)) map.removeSource(SOURCE)
}

interface PastFloodsLayerProps {
  layer: Layer
  /** Shows only this event; null shows every event. */
  selectedFid: number | null
}

/**
 * Where past floods reached, hatched in violet: a colour no forecast uses and
 * a pattern no forecast has, so an old extent is never read as a current
 * warning. It draws under every live layer.
 */
export function PastFloodsLayer({ layer, selectedFid }: PastFloodsLayerProps) {
  const map = useMap()
  const layerRef = useRef(layer)
  const selectedRef = useRef(selectedFid)

  useEffect(() => {
    if (!map) return
    const add = () => addLayers(map, layerRef.current, selectedRef.current)
    add()
    map.on("style.load", add)
    return () => {
      map.off("style.load", add)
      removeLayers(map)
    }
  }, [map])

  useEffect(() => {
    selectedRef.current = selectedFid
    if (!map || !map.getLayer(PAST_FLOOD_FILL)) return
    map.setFilter(PAST_FLOOD_FILL, eventFilter(selectedFid))
    map.setFilter(LINE, eventFilter(selectedFid))
    map.setFilter(SELECTED, ["==", ["id"], selectedFid ?? -1])
  }, [map, selectedFid])

  return null
}

interface PastFloodsListProps {
  data: PastFloodsDocument | undefined
  selectedId: string | null
  onSelect: (id: string) => void
}

/** The mapped events, newest first, under the layer's switch. */
export function PastFloodsList({ data, selectedId, onSelect }: PastFloodsListProps) {
  const t = useT()
  return (
    <div className="flex flex-col gap-2">
      <p className="flex items-start gap-2 text-xs text-muted-foreground">
        <span
          aria-hidden="true"
          className="mt-0.5 inline-block h-2.5 w-3.5 shrink-0 rounded-[2px] border border-past-flood bg-[repeating-linear-gradient(135deg,var(--past-flood)_0,var(--past-flood)_1px,transparent_1px,transparent_3px)]"
        />
        {t("floods.past.legend", {
          attribution: data?.meta.attribution ?? t("terms.agency.irrigation"),
        })}
      </p>
      {data ? (
        <ScrollArea
          type="auto"
          className="rounded-xl border border-border"
          viewportClassName="max-h-56"
        >
          <ul className="flex flex-col divide-y divide-border">
            {data.events.map((event) => (
              <li key={event.id}>
                <button
                  type="button"
                  aria-pressed={selectedId === event.id}
                  onClick={() => onSelect(event.id)}
                  className={cn(
                    "flex w-full items-center justify-between gap-2 px-3 py-1.5 text-left text-xs hover:bg-accent",
                    selectedId === event.id && "bg-accent",
                  )}
                >
                  <span className="min-w-0 truncate text-foreground" title={pastFloodName(event)}>
                    {pastFloodName(event)}
                  </span>
                  <span className="numeric shrink-0 text-muted-foreground">
                    {formatInteger(Math.round(event.areaKm2))} km²
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </ScrollArea>
      ) : (
        <Skeleton
          className="h-24 w-full rounded-xl"
          role="status"
          aria-label={t("floods.past.loadingList")}
        />
      )}
    </div>
  )
}

/** "As published: {name}", the source's own title set in italics, in English as published. */
function AsPublished({ template, name }: { template: string; name: string }) {
  const [before = "", after = ""] = template.split("{name}")
  return (
    <>
      {before}
      <span className="italic" lang="en">
        {name}
      </span>
      {after}
    </>
  )
}

interface PastFloodPanelProps {
  event: PastFlood | undefined
  loading: boolean
  /**
   * The server's English note. The panel shows its own words for it, in the
   * current language, so this is kept only for callers that pass it.
   */
  note?: string | undefined
  attribution: string | undefined
  onClose: () => void
  className?: string
}

/** One past flood: which event it was and where it reached. Never a forecast. */
export function PastFloodPanel({
  event,
  loading,
  attribution,
  onClose,
  className,
}: PastFloodPanelProps) {
  const t = useT()
  const close = (
    <Button variant="outline" size="icon" onClick={onClose} aria-label={t("floods.past.close")}>
      <XIcon />
    </Button>
  )
  if (!event) {
    return (
      <Panel
        icon={<HistoryIcon />}
        title={t("floods.past.title")}
        className={cn("w-full", className)}
        action={close}
      >
        {loading ? (
          <Skeleton
            className="h-24 w-full rounded-2xl"
            role="status"
            aria-label={t("common.loading")}
          />
        ) : (
          <p className="text-sm text-muted-foreground">{t("floods.past.missing")}</p>
        )}
      </Panel>
    )
  }
  return (
    <Panel
      icon={<HistoryIcon />}
      title={pastFloodName(event)}
      subtitle={t("floods.past.title")}
      className={cn("w-full", className)}
      action={close}
      footer={
        <p>
          {t("floods.past.source", {
            attribution: attribution ?? t("terms.agency.irrigation"),
          })}
        </p>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <Badge className="border-past-flood bg-transparent text-foreground">
          {t("floods.past.pastEvent")}
        </Badge>
        <Badge variant="chip" className="text-foreground">
          {t("floods.past.notForecast")}
        </Badge>
      </div>
      <p className="text-sm text-foreground">{t("floods.past.note")}</p>
      <LabeledRows>
        <LabeledRow
          label={t("floods.past.when")}
          value={event.date ? formatDay(event.date) : String(event.year)}
        />
        {event.magnitude && (
          <LabeledRow
            label={t("floods.past.mappedAs")}
            value={pastFloodMagnitude(event.magnitude)}
            numeric={false}
          />
        )}
        <LabeledRow
          label={t("floods.past.extent")}
          value={`${formatInteger(Math.round(event.areaKm2))} km²`}
        />
        {event.districts.length > 0 && (
          <LabeledRow
            label={t("floods.past.districts")}
            value={<span lang="en">{event.districts.join(", ")}</span>}
            numeric={false}
          />
        )}
      </LabeledRows>
      <p className="break-words text-xs text-muted-foreground">
        <AsPublished template={t("floods.past.asPublished")} name={event.sourceName} />
      </p>
    </Panel>
  )
}
