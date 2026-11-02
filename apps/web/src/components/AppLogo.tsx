import * as React from "react"

import { cn } from "@/lib/utils"

/* Geometry of the mark, from brand/mark-*.svg (64-unit grid). */
const island =
  "M23.5 5.5C24.05 5.33 24.08 5.92 25.5 7.5C26.92 9.08 29.5 12.08 32 15C34.5 17.92 38.17 21.5 40.5 25C42.83 28.5 45.17 32.33 46 36C46.83 39.67 46.58 43.87 45.5 47C44.42 50.13 42 53 39.5 54.8C37 56.6 33.33 57.85 30.5 57.8C27.67 57.75 24.48 56.55 22.5 54.5C20.52 52.45 19.25 49.08 18.6 45.5C17.95 41.92 18.32 37.33 18.6 33C18.88 28.67 19.7 23.58 20.3 19.5C20.9 15.42 21.67 10.83 22.2 8.5C22.73 6.17 22.95 5.67 23.5 5.5Z"
const wave = "M12 42.5C17 39.5 21 39.5 26 42.5S35 45.5 40 42.5 49 39.5 54 42.5"
/** The sun is the brand orange in both themes, as in the brand files. */
const sunColor = "#FF5A1F"

interface AppMarkProps extends Omit<React.ComponentProps<"svg">, "children"> {
  /** Leaves out the wave, for sizes under about 20 px. */
  simple?: boolean
  /** Accessible name; omit when a visible name sits beside the mark. */
  title?: string
}

/**
 * The Lanka Weather mark: the island as a raindrop, a wave of water across
 * it and the sun rising to the north-east. The drop takes the text colour.
 */
export function AppMark({ simple = false, title, className, ...props }: AppMarkProps) {
  const id = React.useId()
  const mask = `${id}-wave`
  return (
    <svg
      viewBox="17 4 39 55"
      className={cn("h-7 w-auto shrink-0", className)}
      role={title ? "img" : undefined}
      aria-hidden={title ? undefined : true}
      aria-label={title}
      {...props}
    >
      {!simple && (
        <mask id={mask} maskUnits="userSpaceOnUse" x="0" y="0" width="64" height="64">
          <rect width="64" height="64" fill="#fff" />
          <path d={wave} fill="none" stroke="#000" strokeWidth={3.4} strokeLinecap="round" />
        </mask>
      )}
      <path d={island} fill="currentColor" mask={simple ? undefined : `url(#${mask})`} />
      <circle cx={47.5} cy={13.5} r={7} fill={sunColor} />
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
        <AppMark title="Lanka Weather" className="h-6" />
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
      <AppMark className="h-8" />
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
