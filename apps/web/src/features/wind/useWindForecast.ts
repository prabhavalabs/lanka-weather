import { useQueryClient } from "@tanstack/react-query"
import { useEffect, useMemo, useState } from "react"
import type { Layer } from "@/lib/layers"
import { forecastSteps, nearestStep, stepAt } from "./forecast"
import { prefetchWindField, useWindField } from "./queries"

/** How long each forecast step shows while playing. */
const PLAY_STEP_MS = 1100

/** "Now", refreshed every few minutes so the steps follow the clock. */
function useNow(intervalMs = 5 * 60_000) {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), intervalMs)
    return () => window.clearInterval(timer)
  }, [intervalMs])
  return now
}

/**
 * The wind layer's forecast time and field: the 3-hourly steps the newest
 * run offers, the chosen step, playback, and the next step loaded ahead.
 */
export function useWindForecast(layer: Layer | undefined, enabled: boolean) {
  const client = useQueryClient()
  const now = useNow()
  const [index, setIndex] = useState(0)
  const [playing, setPlaying] = useState(false)
  const [reach, setReach] = useState<string | null>(null)

  const steps = useMemo(() => forecastSteps(now, reach), [now, reach])
  const time = steps.length ? stepAt(steps, index) : nearestStep(now)
  const query = useWindField(layer, time, enabled)
  const field = query.data ?? null

  // The first field says how far its run reaches; the steps follow it.
  const lastValidAt = field?.lastValidAt ?? null
  if (lastValidAt && lastValidAt !== reach) setReach(lastValidAt)

  const current = !query.isPlaceholderData && query.isSuccess

  // Play: move on once the shown step has loaded.
  useEffect(() => {
    if (!playing || !enabled || !current || steps.length < 2) return
    const timer = window.setTimeout(
      () => setIndex((value) => (value + 1) % steps.length),
      PLAY_STEP_MS,
    )
    return () => window.clearTimeout(timer)
  }, [playing, enabled, current, index, steps.length])

  // Load the next step ahead, so playing and stepping do not wait.
  useEffect(() => {
    if (!enabled || steps.length < 2) return
    const next = steps[(index + 1) % steps.length]
    if (next !== undefined) void prefetchWindField(client, layer, next)
  }, [client, layer, enabled, steps, index])

  return {
    steps,
    index: Math.min(index, steps.length - 1),
    setIndex,
    playing,
    setPlaying,
    now,
    field,
    query,
    loading: query.isFetching && !current,
  }
}
