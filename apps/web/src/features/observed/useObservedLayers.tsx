/**
 * The rainfall, weather-stations, air-quality and sea-conditions layers on the
 * map, in the list and in the legends. Kept apart from App, which only asks
 * for what goes in each slot.
 */
import type { MapKeySection } from "@/components/weather/MapKey"
import type { ExpressionSpecification } from "maplibre-gl"
import { useCallback, type ReactNode } from "react"
import { Button } from "@/components/ui/button"
import { LegendSwatch } from "@/components/LegendSwatch"
import { MapLegend } from "@/components/weather/MapLegend"
import { RainfallLegend } from "@/components/weather/RainfallLegend"
import { RampLegend } from "@/components/weather/RampLegend"
import { StepLegend } from "@/components/weather/StepLegend"
import { TemperatureLegend } from "@/components/weather/TemperatureLegend"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { isNotReady } from "@/lib/api"
import { rampFromScale, type RampStop } from "@/lib/colour"
import { formatAge, formatNumber } from "@/lib/format"
import { useT } from "@/lib/i18n"
import type { Layer } from "@/lib/layers"
import type { ResolvedTheme } from "@/lib/theme"
import { aqiScale, pm25Scale, readToken, waveScale } from "@/lib/tokens"
import { useLocalStorage } from "@/lib/useLocalStorage"
import { aqiBands, aqiBandOf, aqiLabel, seaStateOf, steppedRamp } from "../environment/present"
import { FieldLayer } from "../environment/FieldLayer"
import { useAirGrid, useSeaGrid } from "../environment/queries"
import type { AirKey } from "../environment/types"
import { layerStatus, type LayerStatus } from "../layers/status"
import { countBands, gaugesSummary, withoutLiveDuplicates } from "../raingauges/present"
import { useRainGauges } from "../raingauges/queries"
import { RainGaugeLegend } from "../raingauges/RainGaugeLegend"
import { RainGaugesLayer } from "../raingauges/RainGaugesLayer"
import { ObservedRainLayer } from "../rainfall/ObservedRainLayer"
import { useObservedRain, useRainSteps } from "../rainfall/queries"
import { RainImageLayer } from "../rainfall/RainImageLayer"
import { RainImageScale, RainTimeline } from "../rainfall/RainTimeline"
import { useRainTimeline } from "../rainfall/useRainTimeline"
import { MetStationsLayer } from "../stations/MetStationsLayer"
import { feedNotice, markerTemperature } from "../stations/present"
import { useMetStations } from "../stations/queries"
import { sampleGrid } from "../weather-grid/field"

export const RAINFALL = "rainfall"
export const RAIN_GAUGES = "rain-gauges"
export const WEATHER_STATIONS = "weather-stations"
export const AIR_QUALITY = "air-quality"
export const SEA_CONDITIONS = "sea-conditions"

type AirVariable = "usAqi" | "pm25"

/**
 * The wave field fades as the map zooms in: its grid is about 55 km across and
 * its coast simplified to about 1 km, so close up it says little and its edge
 * no longer meets the basemap's shore.
 */
const seaOpacity: ExpressionSpecification = [
  "interpolate",
  ["linear"],
  ["zoom"],
  8,
  0.78,
  11,
  0.3,
  13,
  0,
]

interface ObservedLayersOptions {
  available: (id: string) => Layer | undefined
  enabled: string[]
  /** The selected Met station, if any. */
  selectedMetId: string | null
  /** The selected live rain gauge, if any. */
  selectedRainGaugeId: string | null
  theme: ResolvedTheme
  /** Shows the weather field's forecast rain. */
  onShowModelRain: () => void
}

/** The US AQI range of each band, in the order of `aqiBands`. */
const aqiRanges = ["0–50", "51–100", "101–150", "151–200", "201–300", "301+"]

function airRamp(variable: AirKey): RampStop[] {
  const scale = variable === "usAqi" ? aqiScale : pm25Scale
  return steppedRamp(
    scale,
    scale.map((step) => readToken(step.token)),
    500,
  )
}

function seaRamp(): RampStop[] {
  return rampFromScale(
    waveScale,
    waveScale.map((step) => readToken(step.token)),
    0.5,
  )
}

export function useObservedLayers({
  available,
  enabled,
  selectedMetId,
  selectedRainGaugeId,
  theme,
  onShowModelRain,
}: ObservedLayersOptions) {
  // Subscribes the caller to the language: everything below is rebuilt on a switch.
  const t = useT()
  const [airVariable, setAirVariable] = useLocalStorage<AirVariable>(
    "lanka-weather:air-variable",
    "usAqi",
  )
  const on = (id: string) => enabled.includes(id) && available(id) !== undefined
  const rainLayer = available(RAINFALL)
  const gaugeLayer = available(RAIN_GAUGES)
  const stationLayer = available(WEATHER_STATIONS)
  const airLayer = available(AIR_QUALITY)
  const seaLayer = available(SEA_CONDITIONS)
  const rainOn = on(RAINFALL)
  const gaugesOn = on(RAIN_GAUGES)
  const stationsOn = on(WEATHER_STATIONS)
  const airOn = on(AIR_QUALITY)
  const seaOn = on(SEA_CONDITIONS)

  const rain = useRainSteps(rainLayer, rainOn)
  const observedRain = useObservedRain(rainOn)
  const rainGauges = useRainGauges(gaugeLayer, gaugesOn || selectedRainGaugeId !== null)
  const stations = useMetStations(stationLayer, stationsOn || selectedMetId !== null)
  const air = useAirGrid(airLayer, airOn)
  const sea = useSeaGrid(seaLayer, seaOn)
  const timeline = useRainTimeline(rain.data?.steps ?? [], rainOn)

  const statuses: Record<string, LayerStatus> = {}
  const details: Record<string, ReactNode> = {}

  // Rainfall: the satellite estimate, measured totals, and a way to the forecast.
  if (rainLayer) {
    const counts = observedRain.data?.meta
    statuses[RAINFALL] = {
      ...layerStatus({
        layer: rainLayer,
        parts: rain.data
          ? [
              t("rain.layer.satellite"),
              ...(counts?.fresh ? [t("rain.layer.points", { count: counts.fresh })] : []),
            ]
          : [],
        observedAt: rain.data?.newestValidAt,
        ageLabel: "image",
        error: rain.error,
        pending: rain.isPending,
      }),
    }
  }
  details[RAINFALL] = (
    <div className="flex flex-col gap-3 text-xs text-muted-foreground">
      <section className="flex flex-col gap-1.5">
        <h3 className="font-medium text-foreground">{t("rain.layer.satellite")}</h3>
        <p>
          {rain.data?.newestValidAt
            ? t("rain.layer.satelliteTextNewest", { age: formatAge(rain.data.newestValidAt) })
            : t("rain.layer.satelliteText")}
        </p>
        {rain.data && <RainImageScale data={rain.data} theme={theme} />}
      </section>
      <section className="flex flex-col gap-1.5">
        <h3 className="font-medium text-foreground">{t("rain.layer.measuredTitle")}</h3>
        <p>
          {t("rain.layer.measuredText")}
          {observedRain.data &&
            ` ${t("rain.layer.measuredCounts", {
              fresh: observedRain.data.meta.fresh,
              stale: observedRain.data.meta.stale,
            })}`}
        </p>
        <RainfallLegend bare />
      </section>
      <section className="flex flex-col gap-1.5">
        <h3 className="font-medium text-foreground">{t("rain.layer.forecastTitle")}</h3>
        <p>{t("rain.layer.forecastText")}</p>
        <Button
          size="sm"
          variant="secondary"
          className="h-auto min-h-8 self-start text-left whitespace-normal"
          onClick={onShowModelRain}
        >
          {t("rain.layer.showForecast")}
        </Button>
      </section>
    </div>
  )

  // Live rain gauges: what was measured, wetter ones showing first.
  const gaugeMeta = rainGauges.data?.meta
  if (gaugeLayer) {
    statuses[RAIN_GAUGES] = layerStatus({
      layer: gaugeLayer,
      parts: gaugeMeta ? [gaugesSummary(gaugeMeta)] : [],
      observedAt: gaugeMeta?.sourceTime,
      error: rainGauges.error,
      pending: rainGauges.isPending,
    })
  }
  const gaugeCounts = rainGauges.data ? countBands(rainGauges.data.features) : undefined
  details[RAIN_GAUGES] = (
    <div className="flex flex-col gap-3 text-xs text-muted-foreground">
      <RainGaugeLegend bare counts={gaugeCounts} />
      <p>{t("rain.gauges.layerText")}</p>
      {gaugeMeta?.institutions && (
        <p>
          {Object.entries(gaugeMeta.institutions)
            .sort(([, a], [, b]) => b - a)
            .map(([name, value]) => `${name} ${value}`)
            .join(" · ")}
          .
        </p>
      )}
      <p>
        {gaugeLayer?.attribution && <span lang="en">{gaugeLayer.attribution}. </span>}
        {t("terms.notOfficial")}
      </p>
    </div>
  )

  // Stations: measured readings, with a plain warning when the copy has stopped.
  const stationMeta = stations.data?.meta
  const notice = stationMeta ? feedNotice(stationMeta) : null
  if (stationLayer) {
    const newest = [stationMeta?.threeHourly.newestObservedAt, stationMeta?.daily.newestObservedAt]
      .filter((value): value is string => Boolean(value))
      .sort()
      .at(-1)
    const shown = stations.data?.features.filter(
      (feature) => markerTemperature(feature.properties) !== null,
    )
    statuses[WEATHER_STATIONS] = {
      ...layerStatus({
        layer: stationLayer,
        parts: shown
          ? [t("stations.layer.observed"), t("stations.layer.count", { count: shown.length })]
          : [],
        observedAt: newest,
        ageLabel: "newest",
        error: stations.error,
        pending: stations.isPending,
      }),
      ...(stationMeta ? { stale: stationMeta.stale } : {}),
    }
  }
  details[WEATHER_STATIONS] = (
    <div className="flex flex-col gap-3 text-xs text-muted-foreground">
      {notice && (
        <p
          role="status"
          className="rounded-xl border border-border bg-card px-3 py-2 text-foreground"
        >
          {notice}
        </p>
      )}
      <p className="flex items-center gap-2">
        <LegendSwatch color="var(--temp-4)" />
        {t("stations.layer.legendText")}
      </p>
      <TemperatureLegend title={t("stations.layer.legendTitle")} bare className="md:hidden" />
    </div>
  )

  // Air quality.
  if (airLayer) {
    const values = air.data?.values[airVariable]
    statuses[AIR_QUALITY] = layerStatus({
      layer: airLayer,
      parts: values
        ? [
            t("terms.estimate"),
            airVariable === "usAqi" ? t("environment.air.usAqi") : t("environment.air.pm25"),
          ]
        : [],
      observedAt: air.data?.fetchedAt,
      error: air.error,
      pending: air.isPending,
    })
  }
  const airLegend = (bare: boolean, className?: string) =>
    airVariable === "usAqi" ? (
      <MapLegend
        title={t("environment.air.legendTitle")}
        unit={t("terms.estimate")}
        bare={bare}
        className={className ?? "w-64"}
      >
        <ul className="flex flex-col gap-1">
          {aqiBands.map((band, index) => (
            <li key={band.token} className="flex items-center justify-between gap-2">
              <LegendSwatch
                color={`var(${band.token})`}
                label={aqiLabel(band.category)}
                className="min-w-0"
              />
              <span className="numeric shrink-0 text-[0.6875rem] whitespace-nowrap text-muted-foreground">
                {aqiRanges[index]}
              </span>
            </li>
          ))}
        </ul>
      </MapLegend>
    ) : (
      <StepLegend
        title={t("environment.air.pm25Title")}
        unit="µg/m³"
        scale={pm25Scale}
        names={{
          0: aqiLabel("good"),
          3: aqiLabel("unhealthy"),
          5: aqiLabel("hazardous"),
        }}
        bare={bare}
        className={className}
      />
    )
  details[AIR_QUALITY] = (
    <div className="flex flex-col gap-3">
      <ToggleGroup
        type="single"
        size="sm"
        value={airVariable}
        onValueChange={(value) => {
          if (value) setAirVariable(value as AirVariable)
        }}
        aria-label={t("environment.air.toggle")}
        className="w-full"
      >
        <ToggleGroupItem value="usAqi" className="flex-1">
          {t("environment.air.usAqi")}
        </ToggleGroupItem>
        <ToggleGroupItem value="pm25" className="flex-1">
          {t("environment.air.pm25")}
        </ToggleGroupItem>
      </ToggleGroup>
      {airLegend(true, "md:hidden")}
      <p className="text-xs text-muted-foreground">
        {t("environment.air.note")} <span lang="en">{t("environment.air.credit")}</span>
      </p>
    </div>
  )

  // Sea conditions.
  if (seaLayer) {
    statuses[SEA_CONDITIONS] = layerStatus({
      layer: seaLayer,
      parts: sea.data ? [t("environment.sea.model"), t("environment.sea.waveHeight")] : [],
      observedAt: sea.data?.fetchedAt,
      error: sea.error,
      pending: sea.isPending,
    })
  }
  const seaLegend = (bare: boolean, className?: string) => (
    <RampLegend
      title={t("environment.sea.legendTitle")}
      unit="m"
      scale={waveScale}
      openSpan={0.5}
      ends={[t("environment.seaState.smooth"), t("environment.seaState.high")]}
      bare={bare}
      className={className}
    />
  )
  details[SEA_CONDITIONS] = (
    <div className="flex flex-col gap-3">
      {seaLegend(true, "md:hidden")}
      <p className="text-xs text-muted-foreground">
        {t("environment.sea.note")} <span lang="en">{t("environment.sea.credit")}</span>
      </p>
    </div>
  )

  const rainTimeline =
    rainOn && rain.data && rain.data.steps.length > 0 ? (
      <RainTimeline
        data={rain.data}
        index={timeline.index}
        step={timeline.step}
        onIndexChange={timeline.setIndex}
        playing={timeline.playing}
        onPlayingChange={timeline.setPlaying}
        canPlay={timeline.canPlay}
        theme={theme}
      />
    ) : null

  const legends = (
    <>
      {rainTimeline}
      {stationsOn && (
        <TemperatureLegend title={t("stations.layer.legendTitleObserved")} className="w-60" />
      )}
      {airOn && airLegend(false)}
      {seaOn && seaLegend(false, "w-60")}
    </>
  )

  const airRampFor = useCallback(() => airRamp(airVariable), [airVariable])

  const mapLayers = (
    <>
      {airOn && (
        <FieldLayer
          id="air-quality"
          grid={air.data?.grid}
          values={air.data?.values[airVariable]}
          ramp={airRampFor}
          rampKey={airVariable}
          slot="fills"
        />
      )}
      {seaOn && (
        <FieldLayer
          id="sea"
          grid={sea.data?.grid}
          values={sea.data?.values.waveHeightM}
          ramp={seaRamp}
          rampKey="waves"
          slot="lines"
          land={sea.data?.land}
          opacity={seaOpacity}
        />
      )}
      {rainOn && (
        <RainImageLayer
          steps={rain.data}
          step={timeline.step}
          nextStep={timeline.playing ? timeline.nextStep : undefined}
          theme={theme}
        />
      )}
      {rainOn && (
        <ObservedRainLayer
          data={
            gaugesOn ? withoutLiveDuplicates(observedRain.data, rainGauges.data) : observedRain.data
          }
        />
      )}
      {gaugesOn && <RainGaugesLayer data={rainGauges.data} selectedId={selectedRainGaugeId} />}
      {stationsOn && <MetStationsLayer data={stations.data} selectedId={selectedMetId} />}
    </>
  )

  /** Lines under the pointer from the grids that are on. */
  const lineAt = (lon: number, lat: number): string[] => {
    const lines: string[] = []
    const airValues = air.data?.values[airVariable]
    if (airOn && air.data && airValues) {
      const value = sampleGrid(air.data.grid, airValues, lon, lat)
      if (value !== null) {
        lines.push(
          airVariable === "usAqi"
            ? t("environment.air.hoverAqi", {
                band: aqiLabel(aqiBandOf(value).category),
                aqi: Math.round(value),
              })
            : t("environment.air.hoverPm25", { value: formatNumber(value, 1) }),
        )
      }
    }
    const waves = sea.data?.values.waveHeightM
    if (seaOn && sea.data && waves) {
      const value = sampleGrid(sea.data.grid, waves, lon, lat)
      if (value !== null) {
        lines.push(
          t("environment.sea.hover", { height: formatNumber(value, 1), state: seaStateOf(value) }),
        )
      }
    }
    return lines
  }

  return {
    stationsOn,
    airOn,
    statuses,
    details,
    legends,
    keySections: (gaugesOn
      ? [
          {
            id: "rain-gauges",
            title: t("rain.gauges.legendTitle"),
            unit: t("rain.gauges.legendUnit"),
            content: <RainGaugeLegend bare counts={gaugeCounts} />,
          },
        ]
      : []) satisfies MapKeySection[],
    rainTimeline,
    mapLayers,
    lineAt,
    stationFeature: (id: string) => stations.data?.features.find((feature) => feature.id === id),
    rainGaugeFeature: (id: string) =>
      rainGauges.data?.features.find((feature) => feature.id === id),
    stationsPending: stations.isPending,
    /** Whether any layer is waiting for its first collection. */
    waiting: [
      rainOn && isNotReady(rain.error),
      airOn && isNotReady(air.error),
      seaOn && isNotReady(sea.error),
    ].some(Boolean),
  }
}
