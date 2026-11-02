import type { StyleSpecification } from "maplibre-gl"
import { describe, expect, it } from "vitest"
import { tuneBasemap, type BasemapPalette } from "./basemap"

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
