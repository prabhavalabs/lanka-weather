import { useEffect, useState } from "react"
import { reducedMotionQuery, useMediaQuery } from "@/lib/useMediaQuery"
import type { RainStep } from "./types"

/** How long each half hour shows while playing. */
const PLAY_STEP_MS = 900

/**
 * Which half hour the satellite image shows. It is held as a distance from the
 * newest image, so when a new image arrives the view stays on "latest" or on
 * the time being looked at. Playing is not offered under reduced motion.
 */
export function useRainTimeline(steps: RainStep[], enabled: boolean) {
  const [back, setBack] = useState(0)
  const [playing, setPlaying] = useState(false)
  const reducedMotion = useMediaQuery(reducedMotionQuery)
  const last = steps.length - 1
  const index = Math.max(0, last - back)
  const canPlay = !reducedMotion && steps.length > 1
  const running = playing && canPlay && enabled

  useEffect(() => {
    if (!running) return
    const timer = window.setTimeout(() => {
      setBack((value) => (value <= 0 ? last : value - 1))
    }, PLAY_STEP_MS)
    return () => window.clearTimeout(timer)
  }, [running, back, last])

  return {
    index,
    step: steps[index],
    nextStep: steps[index === last ? 0 : index + 1],
    setIndex: (next: number) => {
      setPlaying(false)
      setBack(Math.max(0, last - next))
    },
    playing: running,
    setPlaying,
    canPlay,
  }
}
