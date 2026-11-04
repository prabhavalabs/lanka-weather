/**
 * The animated wind on the map. Loaded on demand (React.lazy) when the wind
 * layer is switched on, so the renderer is not in the first download.
 */
import { useEffect, useRef } from "react"
import { readToken } from "@/lib/tokens"
import { useMap } from "../map/MapContext"
import { slotBefore, styleReady } from "../map/slots"
import type { WindField } from "./field"
import { WindParticlesLayer } from "./layer"
import { WindRenderer } from "./renderer"
import { windBandTokens } from "./scale"

const LAYER_ID = "wind-particles"

interface WindLayerProps {
  /** The decoded field for the chosen forecast time; null while it loads. */
  field: WindField | null
  opacity?: number
}

function bandColours() {
  return windBandTokens.map((token) => readToken(token))
}

export default function WindLayer({ field, opacity = 0.9 }: WindLayerProps) {
  const map = useMap()
  const renderer = useRef<WindRenderer | null>(null)
  const fieldRef = useRef(field)

  useEffect(() => {
    if (!map) return
    let layer: WindParticlesLayer | null = null
    const instance = new WindRenderer(map, bandColours(), {
      onFrame: () => layer?.invalidate(),
    })
    layer = new WindParticlesLayer(LAYER_ID, instance.canvas, opacity)
    renderer.current = instance

    const add = () => {
      if (!map.getLayer(LAYER_ID)) map.addLayer(layer!, slotBefore(map, "particles"))
      // The theme may have changed: the particles take its colours.
      instance.setColors(bandColours())
    }
    add()
    instance.setField(fieldRef.current)
    map.on("style.load", add)
    return () => {
      map.off("style.load", add)
      instance.destroy()
      renderer.current = null
      if (styleReady(map) && map.getLayer(LAYER_ID)) map.removeLayer(LAYER_ID)
    }
    // The opacity is fixed for the layer's life.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [map])

  useEffect(() => {
    fieldRef.current = field
    renderer.current?.setField(field)
  }, [field])

  return null
}
