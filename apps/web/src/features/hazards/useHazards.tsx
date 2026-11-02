/**
 * Official warnings and the hazard layers, gathered in one place: what they
 * add to the map, the layer list, the legends, the detail panel and the click
 * and hover handling. App.tsx composes the result with everything else.
 */
import type { MapGeoJSONFeature, Map as MapLibreMap, MapMouseEvent, PointLike } from "maplibre-gl"
import { ActivityIcon, TornadoIcon } from "lucide-react"
import { useCallback, useEffect, useMemo, useRef, type ReactNode } from "react"
import { DetailPlaceholder } from "@/components/DetailPlaceholder"
import { Button } from "@/components/ui/button"
import { GaugeList, type GaugeListItem } from "@/components/weather/GaugeList"
import { formatAge } from "@/lib/format"
import { isStale, type Layer } from "@/lib/layers"
import { count, layerStatus, type LayerStatus } from "../layers/status"
import type { Selection } from "../location/selection"
import { CycloneLegend } from "../cyclones/CycloneLegend"
import { CyclonePanel } from "../cyclones/CyclonePanel"
import {
  boundsWithSriLanka,
  cycloneSummary,
  evaluationNote,
  isCalm,
  positions,
} from "../cyclones/present"
import { useCyclones } from "../cyclones/queries"
import { CYCLONE_POSITIONS, CyclonesLayer } from "../cyclones/CyclonesLayer"
import type { CyclonePosition } from "../cyclones/types"
import { EarthquakeLegend } from "../earthquakes/EarthquakeLegend"
import { EarthquakePanel } from "../earthquakes/EarthquakePanel"
import { earthquakeParts, formatDistance } from "../earthquakes/present"
import { useEarthquakes } from "../earthquakes/queries"
import { EARTHQUAKES, EarthquakesLayer } from "../earthquakes/EarthquakesLayer"
import { LandslidePanel } from "../warnings/LandslidePanel"
import { LANDSLIDE_FILL, LandslideLayer } from "../warnings/LandslideLayer"
import { OFFICIAL_FILL, OfficialDistrictsLayer } from "../warnings/OfficialDistrictsLayer"
import {
  highlightLines,
  landslideSummary,
  levelActions,
  levelBadge,
  levelLabels,
  type LandslideLevel,
} from "../warnings/present"
import { useLandslides, useWarnings } from "../warnings/queries"
import { LandslideLegend, OfficialDistrictsLegend } from "../warnings/WarningLegends"
import { WarningBanner } from "../warnings/WarningBanner"
import { WarningsButton } from "../warnings/WarningsButton"
import { WarningsPanel } from "../warnings/WarningsPanel"
import type { HoverLayers } from "../map/HoverChip"
import { regionLevelOf } from "../regions/present"

export const LANDSLIDE_WARNINGS = "landslide-warnings"
export const OFFICIAL_WARNINGS = "official-warnings"
export const EARTHQUAKE_LAYER = "earthquakes"
export const CYCLONE_LAYER = "cyclones"

/** Hazard layers on at the first visit. */
export const hazardDefaults = [
  LANDSLIDE_WARNINGS,
  OFFICIAL_WARNINGS,
  EARTHQUAKE_LAYER,
  CYCLONE_LAYER,
]

interface HazardsOptions {
  map: MapLibreMap | null
  /** The catalogue entry of a layer, if it is available. */
  available: (id: string) => Layer | undefined
  /** Whether a layer is switched on and available. */
  isOn: (id: string) => boolean
  enable: (id: string) => void
  selection: Selection | null
  setSelection: (selection: Selection | null) => void
  /** Moves the map to show an area, clear of the cards over it. */
  fitBox: (bbox: [number, number, number, number]) => void
  /** Moves the map, if needed, so a point is clear of the cards over it. */
  reveal: (lngLat: [number, number]) => void
}

const pointSlack = 6

export function useHazards({
  reveal,
  map,
  available,
  isOn,
  enable,
  selection,
  setSelection,
  fitBox,
}: HazardsOptions) {
  const landslideLayer = available(LANDSLIDE_WARNINGS)
  const officialLayer = available(OFFICIAL_WARNINGS)
  const earthquakeLayer = available(EARTHQUAKE_LAYER)
  const cycloneLayer = available(CYCLONE_LAYER)
  const tilesLayer = available("admin-regions")
  const landslidesOn = isOn(LANDSLIDE_WARNINGS)
  const officialOn = isOn(OFFICIAL_WARNINGS)
  const earthquakesOn = isOn(EARTHQUAKE_LAYER)
  const cyclonesOn = isOn(CYCLONE_LAYER)

  // The banner and the header's count do not depend on any layer being on.
  const warnings = useWarnings(officialLayer)
  const landslides = useLandslides(landslideLayer, landslidesOn)
  const earthquakes = useEarthquakes(earthquakeLayer, earthquakesOn || selection?.kind === "quake")
  const cyclones = useCyclones(cycloneLayer, cyclonesOn || selection?.kind === "cyclone")

  const systems = useMemo(() => positions(cyclones.data), [cyclones.data])
  const quake =
    selection?.kind === "quake"
      ? earthquakes.data?.features.find((feature) => feature.properties.id === selection.id)
          ?.properties
      : undefined
  const system: CyclonePosition | undefined =
    selection?.kind === "cyclone"
      ? systems.find((item) => item.eventId === selection.id)
      : undefined

  const openWarnings = useCallback(
    (filter: { type?: string; district?: string } = {}) =>
      setSelection({ kind: "warnings", ...filter }),
    [setSelection],
  )

  // A shared link to a storm or an earthquake centres on it, with Sri Lanka in view.
  const centred = useRef<string | null>(null)
  useEffect(() => {
    if (!map) return
    const target =
      selection?.kind === "cyclone" && system
        ? { key: `cyclone:${system.eventId}`, lon: system.lon, lat: system.lat }
        : selection?.kind === "quake" && quake && earthquakes.data
          ? {
              key: `quake:${quake.id}`,
              ...lngLat(
                earthquakes.data.features.find((feature) => feature.properties.id === quake.id),
              ),
            }
          : null
    if (!target || centred.current === target.key) return
    centred.current = target.key
    fitBox(boundsWithSriLanka(target))
  }, [map, selection, system, quake, earthquakes.data, fitBox])

  // An opened division stays in view, on phones above the sheet.
  const division =
    selection?.kind === "landslide"
      ? landslides.data?.features.find((feature) => feature.properties.id === selection.id)
          ?.properties
      : undefined
  const divisionKey = division ? division.id : null
  const divisionAt = useRef(division)
  useEffect(() => {
    divisionAt.current = division
  })
  useEffect(() => {
    if (map && divisionKey && divisionAt.current) {
      reveal([divisionAt.current.labelLon, divisionAt.current.labelLat])
    }
  }, [map, divisionKey, reveal])

  // Clicks: a storm or an earthquake, then a division, then a highlighted district.
  const handleClick = useCallback(
    (event: MapMouseEvent): boolean => {
      if (!map) return false
      const { x, y } = event.point
      const box: [PointLike, PointLike] = [
        [x - pointSlack, y - pointSlack],
        [x + pointSlack, y + pointSlack],
      ]
      const points = [CYCLONE_POSITIONS, EARTHQUAKES].filter((id) => map.getLayer(id))
      const hit = points.length ? map.queryRenderedFeatures(box, { layers: points })[0] : undefined
      if (hit?.layer.id === CYCLONE_POSITIONS) {
        setSelection({ kind: "cyclone", id: String(hit.properties.eventId) })
        return true
      }
      if (hit?.layer.id === EARTHQUAKES) {
        setSelection({ kind: "quake", id: String(hit.properties.id) })
        return true
      }
      const named = map.getLayer(LANDSLIDE_FILL)
        ? map.queryRenderedFeatures(event.point, { layers: [LANDSLIDE_FILL] })[0]
        : undefined
      if (named) {
        setSelection({ kind: "landslide", id: String(named.properties.id) })
        return true
      }
      const district = map.getLayer(OFFICIAL_FILL)
        ? map.queryRenderedFeatures(event.point, { layers: [OFFICIAL_FILL] })[0]
        : undefined
      if (
        district &&
        typeof district.properties.id === "string" &&
        regionLevelOf(district.properties.id) === "district"
      ) {
        openWarnings({ district: district.properties.id })
        return true
      }
      return false
    },
    [map, setSelection, openWarnings],
  )

  const highlighted = warnings.data?.districts
  const hover = useMemo<HoverLayers>(
    () => ({
      points: [CYCLONE_POSITIONS, EARTHQUAKES],
      areas: [LANDSLIDE_FILL, OFFICIAL_FILL],
      describe: (feature: MapGeoJSONFeature) => {
        const properties = feature.properties
        if (feature.layer.id === CYCLONE_POSITIONS) {
          return {
            title: String(properties.name),
            lines: [
              `GDACS ${properties.alertLevel} alert`,
              `${formatDistance(Number(properties.distanceKm))} from Sri Lanka`,
            ],
          }
        }
        if (feature.layer.id === EARTHQUAKES) {
          return {
            title: `M${Number(properties.magnitude).toFixed(1)} earthquake`,
            lines: [
              String(properties.place),
              `${formatDistance(Number(properties.distanceKm))} from Sri Lanka`,
            ],
          }
        }
        if (feature.layer.id === LANDSLIDE_FILL) {
          const level = Number(properties.level) as LandslideLevel
          return {
            title: String(properties.name),
            local: properties.nameSi ? String(properties.nameSi) : undefined,
            lines: [
              `${levelLabels[level]} · ${levelActions[level]}`,
              `${properties.district} District`,
            ],
          }
        }
        const district = highlighted?.find((item) => item.id === properties.id)
        return district
          ? { title: `${district.name} District`, lines: highlightLines(district) }
          : null
      },
    }),
    [highlighted],
  )

  // Layer list.
  const statuses: Record<string, LayerStatus> = {}
  const details: Record<string, ReactNode> = {}

  if (landslideLayer) {
    const meta = landslides.data?.meta
    statuses[LANDSLIDE_WARNINGS] = layerStatus({
      layer: landslideLayer,
      parts: meta ? [landslideSummary(meta, landslides.data?.features.length ?? 0)] : [],
      observedAt: meta?.fetchedAt,
      error: landslides.error,
      pending: landslides.isPending,
    })
    if (meta) {
      // The divisions are on the map; this list reaches them from the keyboard too.
      const divisionItems: GaugeListItem[] = (landslides.data?.features ?? [])
        .slice(0, 8)
        .map(({ properties }) => ({
          id: properties.id,
          title: properties.name,
          subtitle: `${properties.district} District`,
          badge: { variant: levelBadge[properties.level], label: `Level ${properties.level}` },
          onSelect: () => setSelection({ kind: "landslide", id: properties.id }),
        }))
      details[LANDSLIDE_WARNINGS] = (
        <div className="flex flex-col gap-2.5">
          <p className="text-xs text-foreground">{meta.note}</p>
          <LandslideLegend
            bare
            counts={landslides.data?.features.length ? meta.counts : undefined}
          />
          <GaugeList title="Most serious divisions" items={divisionItems} />
          {meta.pdfUrl && (
            <a
              href={meta.pdfUrl}
              target="_blank"
              rel="noreferrer"
              className="self-start text-xs font-medium text-brand-text underline underline-offset-2 hover:text-foreground"
            >
              Latest official report (PDF)
            </a>
          )}
          <p className="text-xs text-muted-foreground">
            Reports state a validity of about 24 hours and are reissued while a risk lasts. A place
            not named is not a place said to be safe.
          </p>
        </div>
      )
    }
  }

  if (officialLayer) {
    const data = warnings.data
    const mentioned = data?.districts.length ?? 0
    statuses[OFFICIAL_WARNINGS] = layerStatus({
      layer: officialLayer,
      parts: data
        ? [
            count(data.activeCount, "in force", "in force"),
            mentioned > 0 ? count(mentioned, "district") : "no district highlighted",
          ]
        : [],
      observedAt: data?.meta.fetchedAt,
      error: warnings.error,
      pending: warnings.isPending,
    })
    details[OFFICIAL_WARNINGS] = (
      <div className="flex flex-col gap-2.5">
        <OfficialDistrictsLegend />
        <p className="text-xs text-muted-foreground">
          Highlights come only from landslide levels and from warnings or advisories whose clean
          English text names a district by exact word. A bulletin whose text cannot be read cleanly
          highlights nothing, so an unmarked district is not a clear one.
        </p>
        <Button variant="secondary" size="sm" className="self-start" onClick={() => openWarnings()}>
          Read official warnings
        </Button>
      </div>
    )
  }

  if (earthquakeLayer) {
    const data = earthquakes.data
    statuses[EARTHQUAKE_LAYER] = layerStatus({
      layer: earthquakeLayer,
      parts: data ? earthquakeParts(data) : [],
      observedAt: data?.meta.fetchedAt,
      error: earthquakes.error,
      pending: earthquakes.isPending,
    })
    details[EARTHQUAKE_LAYER] = (
      <div className="flex flex-col gap-2.5">
        <EarthquakeLegend bare />
        <GaugeList
          title="Latest"
          items={(data?.features ?? []).slice(0, 5).map(({ properties }) => ({
            id: properties.id,
            title: `M${properties.magnitude.toFixed(1)} ${properties.place}`,
            subtitle: (
              <span className="numeric">
                {formatAge(properties.time)} · {formatDistance(properties.distanceKm)} from Sri
                Lanka
              </span>
            ),
            onSelect: () => setSelection({ kind: "quake", id: properties.id }),
          }))}
        />
        <Button
          variant="secondary"
          size="sm"
          className="self-start"
          onClick={() => {
            enable(EARTHQUAKE_LAYER)
            fitBox([60, -15, 105, 25])
          }}
        >
          Show the Indian Ocean
        </Button>
        <p className="text-xs text-muted-foreground">
          M4.5 and above from 15° S to 25° N and 60° to 105° E, and whatever smaller USGS lists near
          Sri Lanka. Lanka Weather generates no tsunami warnings.
        </p>
      </div>
    )
  }

  if (cycloneLayer) {
    const data = cyclones.data
    statuses[CYCLONE_LAYER] = layerStatus({
      layer: cycloneLayer,
      parts: data ? cycloneSummary(data) : [],
      observedAt: data?.meta.fetchedAt,
      error: cyclones.error,
      pending: cyclones.isPending,
    })
    details[CYCLONE_LAYER] = (
      <div className="flex flex-col gap-2.5">
        {data && systems.length === 0 && (
          <p
            className="rounded-xl border border-border bg-inset px-3 py-2 text-xs text-foreground"
            data-calm={isCalm(data.meta.status) ? "true" : "false"}
          >
            {data.meta.note}
            {data.meta.fetchedAt && (
              <>
                {" · checked "}
                <span className="numeric">{formatAge(data.meta.fetchedAt)}</span>
              </>
            )}
          </p>
        )}
        {systems.map((item) => (
          <Button
            key={item.eventId}
            variant="secondary"
            size="sm"
            className="self-start"
            onClick={() => setSelection({ kind: "cyclone", id: item.eventId })}
          >
            {item.name} · {formatDistance(item.distanceKm)} from Sri Lanka
          </Button>
        ))}
        {systems.length > 0 && <CycloneLegend bare />}
        <p className="text-xs text-muted-foreground">{evaluationNote}</p>
      </div>
    )
  }

  // Legends over the map on wide screens.
  const legends = (
    <>
      {landslidesOn && Boolean(landslides.data?.features.length) && (
        <LandslideLegend counts={landslides.data?.meta.counts} />
      )}
      {earthquakesOn && Boolean(earthquakes.data?.features.length) && <EarthquakeLegend />}
      {cyclonesOn && systems.length > 0 && <CycloneLegend />}
    </>
  )

  // Map layers.
  const fills = (
    <>
      {officialOn && tilesLayer && (
        <OfficialDistrictsLayer tiles={tilesLayer} districts={warnings.data?.districts ?? []} />
      )}
      {landslidesOn && (
        <LandslideLayer
          data={landslides.data}
          selectedId={selection?.kind === "landslide" ? selection.id : null}
        />
      )}
    </>
  )
  const points = (
    <>
      {cyclonesOn && (
        <CyclonesLayer
          data={cyclones.data}
          selectedId={selection?.kind === "cyclone" ? selection.id : null}
        />
      )}
      {earthquakesOn && (
        <EarthquakesLayer
          data={earthquakes.data}
          selectedId={selection?.kind === "quake" ? selection.id : null}
        />
      )}
    </>
  )

  // Detail panel.
  const close = () => setSelection(null)
  let detail: ReactNode = null
  let detailLabel: string | null = null
  if (selection?.kind === "warnings") {
    detailLabel = "Official warnings"
    detail = (
      <WarningsPanel
        layer={officialLayer}
        type={selection.type}
        district={selection.district}
        districtName={warnings.data?.districts.find((item) => item.id === selection.district)?.name}
        onFilter={(filter) => setSelection({ kind: "warnings", ...filter })}
        onClose={close}
      />
    )
  } else if (selection?.kind === "landslide") {
    detailLabel = "Landslide warning"
    detail = <LandslidePanel id={selection.id} onClose={close} />
  } else if (selection?.kind === "quake") {
    detailLabel = "Earthquake"
    detail = quake ? (
      <EarthquakePanel event={quake} stale={earthquakes.data?.meta.stale} onClose={close} />
    ) : (
      <DetailPlaceholder
        icon={<ActivityIcon />}
        title="Earthquake"
        message={
          earthquakes.isPending
            ? undefined
            : "This event is not in the last 30 days of USGS's list. It may have been revised or deleted."
        }
        onClose={close}
      />
    )
  } else if (selection?.kind === "cyclone") {
    detailLabel = "Cyclone"
    detail = system ? (
      <CyclonePanel system={system} stale={cyclones.data?.meta.stale} onClose={close} />
    ) : (
      <DetailPlaceholder
        icon={<TornadoIcon />}
        title="Cyclone"
        message={cyclones.isPending ? undefined : "This system is no longer listed as active."}
        onClose={close}
      />
    )
  }

  // Not having been able to look is never a calm answer: a failed request, an
  // old copy from the browser's cache and an old upstream all count.
  const stale =
    warnings.isError ||
    (warnings.data?.meta.stale ?? false) ||
    (officialLayer !== undefined &&
      warnings.dataUpdatedAt > 0 &&
      isStale(officialLayer, new Date(warnings.dataUpdatedAt)))
  const activeCount = warnings.data?.activeCount ?? 0
  const banner = warnings.data ? (
    <WarningBanner items={warnings.data.banner} stale={stale} onOpen={() => openWarnings()} />
  ) : null
  const button = (compact: boolean) => (
    <WarningsButton
      compact={compact}
      count={activeCount}
      stale={stale}
      onClick={() => openWarnings()}
      className={compact ? undefined : "pointer-events-auto"}
    />
  )

  return {
    statuses,
    details,
    legends,
    fills,
    points,
    detail,
    detailLabel,
    detailFull: selection?.kind === "warnings",
    /** The map is showing something far from Sri Lanka, on purpose. */
    farAway: selection?.kind === "cyclone" || selection?.kind === "quake",
    banner,
    button,
    hover,
    handleClick,
  }
}

function lngLat(feature: { geometry: { coordinates: number[] } } | undefined) {
  const [lon = 0, lat = 0] = feature?.geometry.coordinates ?? []
  return { lon, lat }
}
