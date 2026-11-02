import { PauseIcon, PlayIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { formatClock, formatDay } from "@/lib/format"
import { windScale } from "@/lib/tokens"
import { cn } from "@/lib/utils"
import { leadLabel } from "./forecast"

interface WindTimelineProps {
  /** Forecast times offered, epoch milliseconds, the first nearest now. */
  steps: number[]
  index: number
  onIndexChange: (index: number) => void
  playing: boolean
  onPlayingChange: (playing: boolean) => void
  /** The model run the field comes from (ISO 8601). */
  runAt?: string
  /** True while the chosen step's field loads. */
  loading?: boolean
  /** The current time, epoch milliseconds, for "+3 h" labels. */
  now: number
  className?: string
}

/**
 * The wind forecast's time: a slider over the 3-hourly steps from now to
 * three days ahead, with play and pause. Times are Sri Lanka time.
 */
export function WindTimeline({
  steps,
  index,
  onIndexChange,
  playing,
  onPlayingChange,
  runAt,
  loading = false,
  now,
  className,
}: WindTimelineProps) {
  const time = steps[Math.min(index, steps.length - 1)] ?? now
  const iso = new Date(time).toISOString()
  const single = steps.length <= 1
  return (
    <section
      aria-label="Wind forecast time"
      className={cn(
        "pointer-events-auto flex w-full max-w-[23rem] flex-col gap-2 rounded-2xl border border-border bg-card/95 px-3 py-2.5 text-card-foreground shadow-panel backdrop-blur-md",
        className,
      )}
    >
      <div className="flex items-center gap-3">
        <Button
          variant="outline"
          size="icon"
          className="size-8 shrink-0"
          onClick={() => onPlayingChange(!playing)}
          disabled={single}
          aria-label={playing ? "Pause the wind forecast" : "Play the wind forecast"}
        >
          {playing ? <PauseIcon /> : <PlayIcon />}
        </Button>
        <Slider
          className="flex-1"
          min={0}
          max={Math.max(0, steps.length - 1)}
          step={1}
          value={[Math.min(index, Math.max(0, steps.length - 1))]}
          onValueChange={([value]) => {
            onPlayingChange(false)
            if (value !== undefined) onIndexChange(value)
          }}
          disabled={single}
          thumbLabel="Forecast time"
          thumbValueText={`${formatDay(iso)} ${formatClock(iso)}, ${leadLabel(time, now)}`}
        />
        <p className="w-[6.5rem] shrink-0 text-right leading-tight">
          <span className="numeric block text-sm text-foreground">{leadLabel(time, now)}</span>
          <span className="numeric block text-[0.6875rem] text-muted-foreground">
            {formatDay(iso)} {formatClock(iso)}
          </span>
        </p>
      </div>
      <div className="flex items-center gap-2" aria-label="Particle colours by wind speed in km/h">
        <ol className="grid flex-1 grid-cols-7 gap-0.5">
          {windScale.map((step, position) => (
            <li key={step.token} className="flex flex-col gap-0.5">
              <span
                aria-hidden="true"
                className="h-1.5 rounded-[2px]"
                style={{ backgroundColor: `var(${step.token})` }}
              />
              <span className="numeric text-[0.625rem] leading-none text-muted-foreground">
                {step.from === null ? `<${windScale[position + 1]?.from}` : step.from}
              </span>
            </li>
          ))}
        </ol>
        <span className="numeric text-[0.625rem] text-muted-foreground">km/h</span>
      </div>
      <p className="flex justify-between gap-3 text-[0.6875rem] text-muted-foreground">
        <span>Wind 10 m · NOAA GFS{loading ? " · loading…" : ""}</span>
        {runAt && (
          <span className="numeric">
            run {formatDay(runAt)} {formatClock(runAt)}
          </span>
        )}
      </p>
    </section>
  )
}
