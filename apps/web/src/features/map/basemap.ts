import type { ExpressionSpecification, LayerSpecification, StyleSpecification } from "maplibre-gl"

/** Basemap colours, read from the theme tokens (src/styles/theme.css). */
export interface BasemapPalette {
  land: string
  water: string
  waterway: string
  park: string
  boundary: string
  label: string
  labelStrong: string
  labelHalo: string
  waterLabel: string
}

export const basemapTokens: Record<keyof BasemapPalette, `--${string}`> = {
  land: "--map-land",
  water: "--map-water",
  waterway: "--map-waterway",
  park: "--map-park",
  boundary: "--map-boundary",
  label: "--map-label",
  labelStrong: "--map-label-strong",
  labelHalo: "--map-label-halo",
  waterLabel: "--map-water-label",
}

/** Place labels that name larger places, drawn in the stronger label colour. */
const strongPlaces = /city|capital|country|state/

const notSriLanka: ExpressionSpecification = ["!=", ["get", "name:en"], "Sri Lanka"]

function sourceLayer(layer: LayerSpecification): string | undefined {
  return "source-layer" in layer ? layer["source-layer"] : undefined
}

/** A copy of a layer with some paint properties replaced. */
function repaint<T extends LayerSpecification>(layer: T, paint: Record<string, unknown>): T {
  return { ...layer, paint: { ...(layer.paint ?? {}), ...paint } } as T
}

/**
 * Tunes an OpenFreeMap style to the app's theme: land and water in the
 * palette's near-neutral tones, muted labels with a halo in the land colour,
 * quiet boundaries. Roads and buildings keep the style's own colours. Layers
 * the palette does not cover are returned unchanged; the input is not mutated.
 */
export function tuneBasemap(
  style: StyleSpecification,
  palette: BasemapPalette,
): StyleSpecification {
  const layers = style.layers.map((layer): LayerSpecification => {
    const source = sourceLayer(layer)
    switch (layer.type) {
      case "background":
        return repaint(layer, { "background-color": palette.land })
      case "fill":
        if (source === "water") return repaint(layer, { "fill-color": palette.water })
        if (source === "park" || /park/.test(layer.id))
          return repaint(layer, { "fill-color": palette.park })
        return layer
      case "line":
        if (source === "waterway") return repaint(layer, { "line-color": palette.waterway })
        if (source === "boundary") return repaint(layer, { "line-color": palette.boundary })
        return layer
      case "symbol": {
        const halo = {
          "text-halo-color": palette.labelHalo,
          "text-halo-width": 1.25,
          "text-halo-blur": 0.5,
        }
        if (source === "water_name" || source === "waterway")
          return repaint(layer, { "text-color": palette.waterLabel, ...halo })
        if (source === "place") {
          const painted = repaint(layer, {
            "text-color": strongPlaces.test(layer.id) ? palette.labelStrong : palette.label,
            ...halo,
          })
          // The map is about Sri Lanka: its own name, in three scripts across the middle
          // of the island, would sit on top of the data. Neighbours keep theirs.
          return /country/.test(layer.id)
            ? ({
                ...painted,
                filter: ["all", ...(painted.filter ? [painted.filter] : []), notSriLanka],
              } as unknown as LayerSpecification)
            : painted
        }
        if (source === "transportation_name" || source === "aerodrome_label")
          return repaint(layer, { "text-color": palette.label, ...halo })
        return layer
      }
      default:
        return layer
    }
  })
  return { ...style, layers }
}
