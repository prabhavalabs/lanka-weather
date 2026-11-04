import { PauseIcon, PlayIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { formatDayTime } from "@/lib/format"
import { useLocale, useT } from "@/lib/i18n"
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
  const t = useT()
  const locale = useLocale()
  const time = steps[Math.min(index, steps.length - 1)] ?? now
  const iso = new Date(time).toISOString()
  const single = steps.length <= 1
  const lead = leadLabel(time, now, locale)
  const when = formatDayTime(iso, locale)
  return (
    <section
      aria-label={t("grid.timeline.section")}
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
          aria-label={playing ? t("grid.timeline.pause") : t("grid.timeline.play")}
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
          thumbLabel={t("grid.timeline.thumb")}
          thumbValueText={t("grid.timeline.thumbValue", { time: when, lead })}
        />
        <p className="max-w-[9rem] min-w-[6.5rem] shrink-0 text-right leading-tight">
          <span className="numeric block text-sm text-foreground">{lead}</span>
          <span className="numeric block text-[0.6875rem] text-muted-foreground">{when}</span>
        </p>
      </div>
      <div className="flex items-center gap-2" aria-label={t("grid.timeline.scale")}>
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
      <p className="flex flex-wrap justify-between gap-x-3 gap-y-0.5 text-[0.6875rem] text-muted-foreground">
        <span>{loading ? t("grid.timeline.sourceLoading") : t("grid.timeline.source")}</span>
        {runAt && (
          <span className="numeric">
            {t("grid.timeline.run", { time: formatDayTime(runAt, locale) })}
          </span>
        )}
      </p>
    </section>
  )
}
