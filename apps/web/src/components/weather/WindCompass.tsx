import { compassPoint, formatNumber } from "@/lib/format"
import { cn } from "@/lib/utils"

interface WindCompassProps {
  speedKmh: number
  gustKmh?: number
  /** Direction the wind blows from, in degrees clockwise from north. */
  fromDegrees: number
  /** Diameter in pixels. */
  size?: number
  className?: string
}

const ticks = Array.from({ length: 72 }, (_, index) => index * 5)
const cardinal = [
  { label: "N", angle: 0 },
  { label: "E", angle: 90 },
  { label: "S", angle: 180 },
  { label: "W", angle: 270 },
]

/**
 * A compass dial: the arrow points where the wind is going, the speed sits
 * in the centre. The accessible label states where it comes from.
 */
export function WindCompass({
  speedKmh,
  gustKmh,
  fromDegrees,
  size = 168,
  className,
}: WindCompassProps) {
  const point = compassPoint(fromDegrees)
  const towards = (fromDegrees + 180) % 360
  const label =
    `Wind from the ${point} (${Math.round(fromDegrees)}°) at ${formatNumber(speedKmh)} km/h` +
    (gustKmh !== undefined ? `, gusting ${formatNumber(gustKmh)} km/h` : "")

  return (
    <figure
      className={cn("relative inline-grid place-items-center", className)}
      style={{ width: size, height: size }}
    >
      <svg
        viewBox="0 0 200 200"
        className="absolute inset-0 size-full"
        role="img"
        aria-label={label}
      >
        <circle cx={100} cy={100} r={92} fill="none" className="stroke-border" strokeWidth={1} />
        {ticks.map((angle) => {
          const major = angle % 45 === 0
          return (
            <line
              key={angle}
              x1={100}
              x2={100}
              y1={14}
              y2={major ? 26 : 20}
              transform={`rotate(${angle} 100 100)`}
              className={major ? "stroke-muted-foreground" : "stroke-border"}
              strokeWidth={major ? 1.5 : 1}
              strokeLinecap="round"
            />
          )
        })}
        {cardinal.map(({ label: letter, angle }) => {
          const radians = ((angle - 90) * Math.PI) / 180
          return (
            <text
              key={letter}
              x={100 + Math.cos(radians) * 64}
              y={100 + Math.sin(radians) * 64}
              textAnchor="middle"
              dominantBaseline="central"
              className={cn(
                "font-mono text-[11px]",
                letter === "N" ? "fill-foreground" : "fill-muted-foreground",
              )}
            >
              {letter}
            </text>
          )
        })}
        <g
          style={{ transform: `rotate(${towards}deg)`, transformOrigin: "100px 100px" }}
          className="transition-transform duration-500 ease-out"
        >
          {/* Tail at the side the wind comes from, head where it goes. */}
          <line
            x1={100}
            y1={170}
            x2={100}
            y2={46}
            className="stroke-brand"
            strokeWidth={2.5}
            strokeLinecap="round"
          />
          <path d="M100 30 L110 50 L100 45 L90 50 Z" className="fill-brand" />
          <circle cx={100} cy={170} r={4} className="fill-card stroke-brand" strokeWidth={2.5} />
        </g>
        <circle cx={100} cy={100} r={34} className="fill-card stroke-border" strokeWidth={1} />
      </svg>
      <figcaption className="relative flex flex-col items-center leading-none" aria-hidden="true">
        <span className="numeric text-2xl font-medium text-foreground">
          {formatNumber(speedKmh)}
        </span>
        <span className="mt-1 text-[0.6875rem] text-muted-foreground">km/h</span>
      </figcaption>
    </figure>
  )
}
