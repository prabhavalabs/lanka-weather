import { PauseIcon, PlayIcon } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Slider } from "@/components/ui/slider"
import { formatAge } from "@/lib/format"
import { useT } from "@/lib/i18n"
import type { ResolvedTheme } from "@/lib/theme"
import { cn } from "@/lib/utils"
import { liftedColour, rainImageLift, stepLabel } from "./present"
import type { RainStep, RainSteps } from "./types"

interface RainTimelineProps {
  data: RainSteps
  index: number
  step: RainStep | undefined
  onIndexChange: (index: number) => void
  playing: boolean
  onPlayingChange: (playing: boolean) => void
  canPlay: boolean
  theme: ResolvedTheme
  className?: string
}

/** NASA's colours at round rates, drawn as the map draws them. */
export function RainImageScale({ data, theme }: { data: RainSteps; theme: ResolvedTheme }) {
  const t = useT()
  const lift = rainImageLift(theme)
  const stops = data.legend.stops
  return (
    <div
      className="flex items-center gap-2"
      aria-label={t("rain.timeline.scale", {
        from: stops[0]?.rateMmH ?? 0,
        to: stops.at(-1)?.rateMmH ?? 0,
        unit: data.legend.unit,
      })}
      role="img"
    >
      <ol
        className="grid flex-1 gap-0.5"
        style={{ gridTemplateColumns: `repeat(${stops.length}, 1fr)` }}
      >
        {stops.map((stop) => (
          <li key={stop.rateMmH} className="flex flex-col gap-0.5">
            <span
              aria-hidden="true"
              className="h-1.5 rounded-[2px]"
              style={{ backgroundColor: liftedColour(stop.color, lift) }}
            />
            <span className="numeric text-[0.625rem] leading-none text-muted-foreground">
              {stop.rateMmH}
            </span>
          </li>
        ))}
      </ol>
      <span className="numeric text-[0.625rem] text-muted-foreground">{data.legend.unit}</span>
    </div>
  )
}

/**
 * The satellite image's time: a slider over the half hours stored, newest at
 * the right. It always says how old the image is and that it is an estimate.
 */
export function RainTimeline({
  data,
  index,
  step,
  onIndexChange,
  playing,
  onPlayingChange,
  canPlay,
  theme,
  className,
}: RainTimelineProps) {
  const t = useT()
  const count = data.steps.length
  const label = step ? stepLabel(step) : ""
  const age = step ? formatAge(step.endsAt) : ""
  return (
    <section
      aria-label={t("rain.timeline.section")}
      className={cn(
        "pointer-events-auto flex w-full max-w-[23rem] flex-col gap-2 rounded-2xl md:w-[23rem] border border-border bg-card/95 px-3 py-2.5 text-card-foreground shadow-panel backdrop-blur-md",
        className,
      )}
    >
      <div className="flex items-center gap-3">
        {canPlay && (
          <Button
            variant="outline"
            size="icon"
            className="size-8 shrink-0"
            onClick={() => onPlayingChange(!playing)}
            aria-label={playing ? t("rain.timeline.pause") : t("rain.timeline.play")}
          >
            {playing ? <PauseIcon /> : <PlayIcon />}
          </Button>
        )}
        <Slider
          className="flex-1"
          min={0}
          max={Math.max(0, count - 1)}
          step={1}
          value={[index]}
          onValueChange={([value]) => {
            if (value !== undefined) onIndexChange(value)
          }}
          disabled={count < 2}
          thumbLabel={t("rain.timeline.thumb")}
          thumbValueText={t("rain.timeline.thumbValue", { time: label, age })}
        />
        <p className="max-w-[45%] min-w-[9.25rem] shrink-0 text-right leading-tight">
          <span className="numeric block text-sm text-foreground">{age}</span>
          <span className="numeric block text-[0.6875rem] text-muted-foreground">{label}</span>
        </p>
      </div>
      <RainImageScale data={data} theme={theme} />
      <p className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-[0.6875rem] text-muted-foreground">
        <span>{t("rain.timeline.note")}</span>
        {data.stale && (
          <Badge variant="chip" className="h-5 px-1.5 text-[0.6875rem] text-foreground">
            {t("rain.timeline.outOfDate")}
          </Badge>
        )}
      </p>
    </section>
  )
}
