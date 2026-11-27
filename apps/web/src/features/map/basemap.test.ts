import type { ExpressionSpecification, StyleSpecification } from "maplibre-gl"
import { describe, expect, it } from "vitest"
import {
  groupedLayers,
  labelFields,
  localisedLabel,
  tuneBasemap,
  type BasemapPalette,
} from "./basemap"

const palette: BasemapPalette = {
  land: "#111111",
  water: "#000000",
  waterway: "#222222",
  park: "#151515",
  boundary: "#333333",
  label: "#888888",
  labelStrong: "#cccccc",
  labelHalo: "#0a0a0a",
  waterLabel: "#778899",
}

const style: StyleSpecification = {
  version: 8,
  sources: { openmaptiles: { type: "vector", url: "https://example.org/tiles.json" } },
  layers: [
    { id: "background", type: "background", paint: { "background-color": "#f2f3f0" } },
    { id: "water", type: "fill", source: "openmaptiles", "source-layer": "water" },
    { id: "landuse_park", type: "fill", source: "openmaptiles", "source-layer": "landuse" },
    { id: "building", type: "fill", source: "openmaptiles", "source-layer": "building" },
    {
      id: "highway_major_inner",
      type: "line",
      source: "openmaptiles",
      "source-layer": "transportation",
      paint: { "line-color": "#fff", "line-width": 2 },
    },
    { id: "boundary_2", type: "line", source: "openmaptiles", "source-layer": "boundary" },
    { id: "label_city", type: "symbol", source: "openmaptiles", "source-layer": "place" },
    { id: "label_village", type: "symbol", source: "openmaptiles", "source-layer": "place" },
    { id: "water_name", type: "symbol", source: "openmaptiles", "source-layer": "water_name" },
  ],
}

const paint = (result: StyleSpecification, id: string) =>
  result.layers.find((layer) => layer.id === id)?.paint as Record<string, unknown> | undefined

describe("tuneBasemap", () => {
  const result = tuneBasemap(style, palette)

  it("leaves Sri Lanka's own name off the map, and keeps its neighbours'", () => {
    const tuned = tuneBasemap(
      {
        ...style,
        layers: [
          {
            id: "label_country_2",
            type: "symbol",
            source: "openmaptiles",
            "source-layer": "place",
            filter: ["==", ["get", "class"], "country"],
          },
        ],
      },
      palette,
    )
    const layer = tuned.layers[0] as { filter?: unknown }
    expect(layer.filter).toEqual([
      "all",
      ["==", ["get", "class"], "country"],
      ["!=", ["get", "name:en"], "Sri Lanka"],
    ])
  })

  it("paints land, water, parks and boundaries from the palette", () => {
    expect(paint(result, "background")?.["background-color"]).toBe("#111111")
    expect(paint(result, "water")?.["fill-color"]).toBe("#000000")
    expect(paint(result, "landuse_park")?.["fill-color"]).toBe("#151515")
    expect(paint(result, "boundary_2")?.["line-color"]).toBe("#333333")
  })

  it("mutes labels and names larger places more strongly", () => {
    expect(paint(result, "label_city")?.["text-color"]).toBe("#cccccc")
    expect(paint(result, "label_village")?.["text-color"]).toBe("#888888")
    expect(paint(result, "label_village")?.["text-halo-color"]).toBe("#0a0a0a")
    expect(paint(result, "water_name")?.["text-color"]).toBe("#778899")
  })

  it("leaves roads and buildings alone and does not mutate the input", () => {
    expect(paint(result, "highway_major_inner")).toEqual({ "line-color": "#fff", "line-width": 2 })
    expect(paint(result, "building")).toBeUndefined()
    expect(paint(style, "background")?.["background-color"]).toBe("#f2f3f0")
  })
})

describe("basemap labels and scripts", () => {
  const nameField: ExpressionSpecification = ["coalesce", ["get", "name_en"], ["get", "name"]]
  const labelled: StyleSpecification = {
    ...style,
    layers: [
      {
        id: "label_town",
        type: "symbol",
        source: "openmaptiles",
        "source-layer": "place",
        layout: { "text-field": nameField, "text-font": ["Noto Sans Bold"] },
      },
      {
        id: "road_shield",
        type: "symbol",
        source: "openmaptiles",
        "source-layer": "transportation_name",
        layout: { "text-field": ["to-string", ["get", "ref"]] },
      },
    ],
  }
  const layout = (result: StyleSpecification, id: string) =>
    result.layers.find((layer) => layer.id === id)?.layout as Record<string, unknown>

  it("reads place names in Tamil where the tiles have them, else as the style does", () => {
    const tuned = tuneBasemap(labelled, palette, { locale: "ta" })
    expect(layout(tuned, "label_town")["text-field"]).toEqual([
      "coalesce",
      ["get", "name:ta"],
      nameField,
    ])
    expect(layout(tuned, "road_shield")["text-field"]).toEqual(["to-string", ["get", "ref"]])
  })

  it("keeps the style's own label in English and remembers it for a later switch", () => {
    const tuned = tuneBasemap(labelled, palette, { locale: "en" })
    expect(layout(tuned, "label_town")["text-field"]).toEqual(nameField)
    expect(labelFields(tuned.layers)).toEqual([{ id: "label_town", original: nameField }])
    expect(localisedLabel(nameField, "si")).toEqual(["coalesce", ["get", "name:si"], nameField])
  })

  it("draws Sinhala and Tamil from the app's fonts for every font the style names", () => {
    const tuned = tuneBasemap(labelled, palette, {
      fonts: { sinhala: "/si.woff2", tamil: "/ta.woff2" },
    })
    const faces = tuned["font-faces"] as Record<
      string,
      { url: string; "unicode-range": string[] }[]
    >
    expect(Object.keys(faces)).toEqual(
      expect.arrayContaining(["Noto Sans Regular", "Noto Sans Bold", "Noto Sans Italic"]),
    )
    expect(faces["Noto Sans Bold"]).toEqual([
      { url: "/si.woff2", "unicode-range": ["U+0D80-0DFF", "U+111E0-111FF", "U+200C-200D"] },
      { url: "/ta.woff2", "unicode-range": ["U+0B80-0BFF", "U+11FC0-11FFF"] },
    ])
  })
})

describe("basemap groups", () => {
  const visibility = (result: StyleSpecification, id: string) =>
    (result.layers.find((layer) => layer.id === id)?.layout as Record<string, unknown> | undefined)
      ?.visibility

  it("marks each detail layer with its group, and leaves land and water out", () => {
    expect(groupedLayers(tuneBasemap(style, palette).layers)).toEqual([
      { id: "landuse_park", group: "landcover" },
      { id: "building", group: "landcover" },
      { id: "highway_major_inner", group: "roads" },
      { id: "boundary_2", group: "boundaries" },
      { id: "label_city", group: "places" },
      { id: "label_village", group: "places" },
      { id: "water_name", group: "waterways" },
    ])
  })

  it("hides the groups that are not shown, and only those", () => {
    const result = tuneBasemap(style, palette, { shown: ["places"] })
    expect(visibility(result, "highway_major_inner")).toBe("none")
    expect(visibility(result, "building")).toBe("none")
    expect(visibility(result, "water_name")).toBe("none")
    expect(visibility(result, "label_city")).toBeUndefined()
    expect(visibility(result, "water")).toBeUndefined()
    expect(visibility(result, "background")).toBeUndefined()
  })

  it("never groups a layer the style itself hides", () => {
    const hidden = tuneBasemap(
      {
        ...style,
        layers: [
          {
            id: "road_hidden",
            type: "line",
            source: "openmaptiles",
            "source-layer": "transportation",
            layout: { visibility: "none" },
          },
        ],
      },
      palette,
      { shown: ["roads"] },
    )
    expect(groupedLayers(hidden.layers)).toEqual([])
    expect(visibility(hidden, "road_hidden")).toBe("none")
  })
})
