import * as React from "react"

import { cn } from "@/lib/utils"

/* Geometry of the mark, from brand/mark-*.svg (64-unit grid, centred on 32,32). */
const island =
  "M23.52 28.83Q23.22 29.29 23.82 33.09Q24.41 36.88 24.31 37.18Q24.21 37.48 24.44 38.3Q24.67 39.13 24.48 39.39Q24.28 39.66 25.23 42.63Q26.19 45.6 26.65 46.19Q27.12 46.78 28.77 47.31Q30.42 47.84 31.34 47.34Q32.26 46.85 34.28 46.22Q36.29 45.6 37.77 44.41Q39.26 43.22 39.79 42.1Q40.32 40.98 40.55 39.95Q40.78 38.93 40.71 37.71Q40.65 36.49 40.28 35.27Q39.92 34.05 39.16 33.16Q38.4 32.26 38.27 31.67Q38.14 31.08 37.81 30.84Q37.48 30.61 37.02 28.8Q36.55 26.98 36.42 26.88Q36.29 26.79 36.06 27.02Q35.83 27.25 35.63 27.21Q35.43 27.18 35.47 26.46Q35.5 25.73 34.38 24.24Q33.25 22.76 32.73 21.64Q32.2 20.52 30.42 19.03Q28.63 17.55 28.17 16.89Q27.71 16.23 26.82 16.26Q25.93 16.29 25.23 16.59Q24.54 16.89 24.51 17.02Q24.48 17.15 25.04 17.28Q25.6 17.41 25.53 17.71Q25.47 18.01 25.76 18.01Q26.06 18.01 26.59 18.5Q27.12 19 26.62 19.36Q26.13 19.72 26.39 20.42Q26.65 21.11 26.46 21.74Q26.26 22.36 25.63 22.89Q25 23.42 25.14 24.67Q25.27 25.93 24.97 26.29Q24.67 26.65 24.51 27.64Q24.34 28.63 24.15 28.73Q23.95 28.83 23.88 28.6Q23.82 28.37 23.52 28.83Z"
/** The island larger, for the small mark. */
const islandSmall =
  "M21.72 28.16Q21.36 28.72 22.08 33.32Q22.8 37.92 22.68 38.28Q22.56 38.64 22.84 39.64Q23.12 40.64 22.88 40.96Q22.64 41.28 23.8 44.88Q24.96 48.48 25.52 49.2Q26.08 49.92 28.08 50.56Q30.08 51.2 31.2 50.6Q32.32 50 34.76 49.24Q37.2 48.48 39 47.04Q40.8 45.6 41.44 44.24Q42.08 42.88 42.36 41.64Q42.64 40.4 42.56 38.92Q42.48 37.44 42.04 35.96Q41.6 34.48 40.68 33.4Q39.76 32.32 39.6 31.6Q39.44 30.88 39.04 30.6Q38.64 30.32 38.08 28.12Q37.52 25.92 37.36 25.8Q37.2 25.68 36.92 25.96Q36.64 26.24 36.4 26.2Q36.16 26.16 36.2 25.28Q36.24 24.4 34.88 22.6Q33.52 20.8 32.88 19.44Q32.24 18.08 30.08 16.28Q27.92 14.48 27.36 13.68Q26.8 12.88 25.72 12.92Q24.64 12.96 23.8 13.32Q22.96 13.68 22.92 13.84Q22.88 14 23.56 14.16Q24.24 14.32 24.16 14.68Q24.08 15.04 24.44 15.04Q24.8 15.04 25.44 15.64Q26.08 16.24 25.48 16.68Q24.88 17.12 25.2 17.96Q25.52 18.8 25.28 19.56Q25.04 20.32 24.28 20.96Q23.52 21.6 23.68 23.12Q23.84 24.64 23.48 25.08Q23.12 25.52 22.92 26.72Q22.72 27.92 22.48 28.04Q22.24 28.16 22.16 27.88Q22.08 27.6 21.72 28.16Z"
/** The radar's sweep, from north through 60 degrees. */
const sweep = "M32 32V4A28 28 0 0 1 56.25 18Z"
/** The rings and sweep are the brand orange in both themes, as in the brand files. */
const signal = "#FF5A1F"

interface AppMarkProps extends Omit<React.ComponentProps<"svg">, "children"> {
  /** One bold ring around a larger island, for sizes under about 24 px. */
  simple?: boolean
  /** Accessible name; omit when a visible name sits beside the mark. */
  title?: string
}

/**
 * The Lanka Weather mark: the island at the centre of radar rings, the
 * whole country watched at once. The island takes the text colour.
 */
export function AppMark({ simple = false, title, className, ...props }: AppMarkProps) {
  return (
    <svg
      viewBox={simple ? "4.6 4.6 54.8 54.8" : "4 4 56 56"}
      className={cn("size-7 shrink-0", className)}
      role={title ? "img" : undefined}
      aria-hidden={title ? undefined : true}
      aria-label={title}
      {...props}
    >
      {simple ? (
        <>
          <circle cx={32} cy={32} r={25.5} fill="none" stroke={signal} strokeWidth={3.4} />
          <path d={islandSmall} fill="currentColor" />
        </>
      ) : (
        <>
          <path d={sweep} fill={signal} fillOpacity={0.28} />
          <g fill="none" stroke={signal} strokeWidth={1.6}>
            <circle cx={32} cy={32} r={13} strokeOpacity={0.9} />
            <circle cx={32} cy={32} r={20.5} strokeOpacity={0.55} />
            <circle cx={32} cy={32} r={28} strokeOpacity={0.3} />
          </g>
          <path d={island} fill="currentColor" />
        </>
      )}
    </svg>
  )
}

interface AppLogoProps extends React.ComponentProps<"div"> {
  /** `lockup` adds the name beside the mark; `tile` sets the mark on a rounded tile. */
  variant?: "lockup" | "mark" | "tile"
  /** Under the name, e.g. a one-line description. */
  tagline?: React.ReactNode
}

/** The app's logo: mark and name in Geist, laid out as in brand/lockup-horizontal-*.svg. */
export function AppLogo({ variant = "lockup", tagline, className, ...props }: AppLogoProps) {
  if (variant === "tile") {
    return (
      <div
        className={cn(
          "inline-flex size-10 items-center justify-center rounded-xl bg-[#0a0a0a] text-[#f2f2f2] ring-1 ring-white/10",
          className,
        )}
        {...props}
      >
        <AppMark title="Lanka Weather" className="size-8" />
      </div>
    )
  }
  if (variant === "mark") {
    return (
      <div className={cn("inline-flex text-foreground", className)} {...props}>
        <AppMark title="Lanka Weather" />
      </div>
    )
  }
  return (
    <div className={cn("flex items-center gap-2.5 text-foreground", className)} {...props}>
      <AppMark className="size-9" />
      <div className="min-w-0 leading-none">
        <p className="text-[1.0625rem] font-semibold tracking-[-0.02em] whitespace-nowrap">
          Lanka Weather
        </p>
        {tagline && (
          <p className="mt-1 truncate text-xs font-normal text-muted-foreground">{tagline}</p>
        )}
      </div>
    </div>
  )
}
