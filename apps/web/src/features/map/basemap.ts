import type {
  ExpressionSpecification,
  FontFacesSpecification,
  LayerSpecification,
  StyleSpecification,
} from "maplibre-gl"
import type { Locale } from "@/lib/i18n"

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

/** Where a label layer keeps the style's own `text-field`, to localise it again later. */
export const labelFieldKey = "lanka-weather:text-field"

/** Files for the scripts the style's glyphs URL cannot draw: Sinhala and Tamil. */
export interface ScriptFonts {
  sinhala: string
  tamil: string
}

const sinhalaRange = ["U+0D80-0DFF", "U+111E0-111FF"]
const tamilRange = ["U+0B80-0BFF", "U+11FC0-11FFF"]
/** Joiners Sinhala uses for conjuncts (yansaya, rakaransaya, touching letters). */
const joinerRange = ["U+200C-200D"]

/**
 * `font-faces` for every font the style names, pointing Sinhala and Tamil
 * at the app's own Noto files. OpenFreeMap's glyph files have neither
 * script, and a glyph URL serves one codepoint at a time where these
 * scripts need a letter and its marks drawn together; MapLibre draws text
 * covered by `font-faces` a grapheme cluster at a time with the browser's
 * text engine.
 */
export function scriptFontFaces(stacks: Iterable<string>, fonts: ScriptFonts) {
  const faces: FontFacesSpecification = {}
  for (const stack of stacks) {
    faces[stack] = [
      { url: fonts.sinhala, "unicode-range": [...sinhalaRange, ...joinerRange] },
      { url: fonts.tamil, "unicode-range": tamilRange },
    ]
  }
  return faces
}

/** Every font stack the style's text layers name, plus the one our own layers use. */
function fontStacks(layers: LayerSpecification[]): Set<string> {
  const stacks = new Set(["Noto Sans Regular", "Noto Sans Bold", "Noto Sans Italic"])
  for (const layer of layers) {
    if (layer.type !== "symbol") continue
    const font = layer.layout?.["text-font"]
    if (Array.isArray(font) && font.every((name) => typeof name === "string")) {
      for (const name of font as string[]) stacks.add(name)
    }
  }
  return stacks
}

/** Whether a `text-field` shows a name (rather than, say, a road number). */
function showsName(field: unknown): boolean {
  return JSON.stringify(field ?? null).includes('"name')
}

/**
 * A label's text in the reader's language: the tiles' `name:ta` (or
 * `name:si`, which OpenFreeMap does not carry today) where a place has one,
 * else the style's own label.
 */
export function localisedLabel(original: unknown, locale: Locale): unknown {
  if (locale === "en") return original
  return ["coalesce", ["get", `name:${locale}`], original]
}

/** The label layers to localise, with the style's own `text-field` of each. */
export function labelFields(layers: LayerSpecification[]): { id: string; original: unknown }[] {
  return layers.flatMap((layer) => {
    const original = (layer.metadata as Record<string, unknown> | undefined)?.[labelFieldKey]
    return original === undefined ? [] : [{ id: layer.id, original }]
  })
}

/**
 * Tunes an OpenFreeMap style to the app's theme: land and water in the
 * palette's near-neutral tones, muted labels with a halo in the land colour,
 * quiet boundaries. Roads and buildings keep the style's own colours. Layers
 * the palette does not cover are returned unchanged; the input is not mutated.
 *
 * With `locale`, name labels read in that language where the tiles have it,
 * and keep the style's own field in their metadata so a later language change
 * can be applied without reloading the style. With `fonts`, Sinhala and Tamil
 * are drawn from those files.
 */
export function tuneBasemap(
  style: StyleSpecification,
  palette: BasemapPalette,
  options: { locale?: Locale; fonts?: ScriptFonts } = {},
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
  const localised = options.locale
    ? layers.map((layer): LayerSpecification => {
        if (layer.type !== "symbol" || !showsName(layer.layout?.["text-field"])) return layer
        const original = layer.layout!["text-field"]
        return {
          ...layer,
          metadata: {
            ...((layer.metadata as object | undefined) ?? {}),
            [labelFieldKey]: original,
          },
          layout: {
            ...layer.layout,
            "text-field": localisedLabel(original, options.locale!),
          },
        } as LayerSpecification
      })
    : layers
  return {
    ...style,
    layers: localised,
    ...(options.fonts
      ? {
          "font-faces": {
            ...(style["font-faces"] ?? {}),
            ...scriptFontFaces(fontStacks(localised), options.fonts),
          },
        }
      : {}),
  }
}
