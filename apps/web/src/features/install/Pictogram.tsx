import type * as React from "react"
import { AppMark } from "@/components/AppLogo"
import { cn } from "@/lib/utils"

/**
 * Small drawings of the real buttons a step asks for, so people can match
 * what they see on screen: Apple's share square, Android's ⋮, the add
 * square and so on. Drawn on a 24-unit grid with round strokes.
 */
export type Glyph =
  | "share"
  | "add-square"
  | "kebab"
  | "ellipsis"
  | "menu"
  | "install-desktop"
  | "app-available"
  | "app"
  | "dock"
  | "download"
  | "bookmark"
  | "browser"

const paths: Record<Exclude<Glyph, "app">, React.ReactNode> = {
  // iOS: an open-topped square with an arrow rising out of it.
  share: (
    <>
      <path d="M12 3v11" />
      <path d="M8.5 6.5 12 3l3.5 3.5" />
      <path d="M8 10H6.5A1.5 1.5 0 0 0 5 11.5v8A1.5 1.5 0 0 0 6.5 21h11a1.5 1.5 0 0 0 1.5-1.5v-8a1.5 1.5 0 0 0-1.5-1.5H16" />
    </>
  ),
  "add-square": (
    <>
      <rect x="4" y="4" width="16" height="16" rx="4" />
      <path d="M12 8.5v7M8.5 12h7" />
    </>
  ),
  kebab: (
    <>
      <circle cx="12" cy="5.5" r="1.4" fill="currentColor" stroke="none" />
      <circle cx="12" cy="12" r="1.4" fill="currentColor" stroke="none" />
      <circle cx="12" cy="18.5" r="1.4" fill="currentColor" stroke="none" />
    </>
  ),
  ellipsis: (
    <>
      <circle cx="5.5" cy="12" r="1.4" fill="currentColor" stroke="none" />
      <circle cx="12" cy="12" r="1.4" fill="currentColor" stroke="none" />
      <circle cx="18.5" cy="12" r="1.4" fill="currentColor" stroke="none" />
    </>
  ),
  menu: <path d="M4.5 7h15M4.5 12h15M4.5 17h15" />,
  // Chrome's address-bar install icon: a screen with an arrow coming down into it.
  "install-desktop": (
    <>
      <path d="M9 4H5.5A1.5 1.5 0 0 0 4 5.5v9A1.5 1.5 0 0 0 5.5 16h13a1.5 1.5 0 0 0 1.5-1.5v-9A1.5 1.5 0 0 0 18.5 4H15" />
      <path d="M12 3v8M9 8.5l3 3 3-3" />
      <path d="M9 20h6M12 16v4" />
    </>
  ),
  // Edge's "App available": three squares and a plus.
  "app-available": (
    <>
      <rect x="4" y="4" width="6.5" height="6.5" rx="1.5" />
      <rect x="4" y="13.5" width="6.5" height="6.5" rx="1.5" />
      <rect x="13.5" y="13.5" width="6.5" height="6.5" rx="1.5" />
      <path d="M16.75 4v6.5M13.5 7.25H20" />
    </>
  ),
  dock: (
    <>
      <rect x="3" y="4" width="18" height="12" rx="2" />
      <rect x="6" y="17.5" width="12" height="3" rx="1.5" />
    </>
  ),
  download: (
    <>
      <path d="M12 4v11M7.5 10.5 12 15l4.5-4.5" />
      <path d="M5 19.5h14" />
    </>
  ),
  bookmark: <path d="M7 4h10a1 1 0 0 1 1 1v15l-6-4-6 4V5a1 1 0 0 1 1-1Z" />,
  browser: (
    <>
      <rect x="3" y="4.5" width="18" height="15" rx="2.5" />
      <path d="M3 9h18" />
      <circle cx="6" cy="6.8" r="0.6" fill="currentColor" />
      <circle cx="8.2" cy="6.8" r="0.6" fill="currentColor" />
    </>
  ),
}

/** A step's picture: a drawn button, or the button's own word (e.g. "Add"). */
export type StepPicture = { glyph: Glyph } | { label: string }

export function Pictogram({ picture, className }: { picture: StepPicture; className?: string }) {
  if ("label" in picture) {
    return (
      <span
        aria-hidden
        className={cn(
          "inline-flex h-11 min-w-11 shrink-0 items-center justify-center rounded-xl border border-border bg-background px-2.5 text-[0.9375rem] font-semibold text-[#0a84ff] dark:text-[#4da3ff]",
          className,
        )}
      >
        {picture.label}
      </span>
    )
  }
  return (
    <span
      aria-hidden
      className={cn(
        "inline-flex size-11 shrink-0 items-center justify-center rounded-xl border border-border bg-background text-foreground",
        className,
      )}
    >
      {picture.glyph === "app" ? (
        <span className="flex size-8 items-center justify-center rounded-[0.55rem] bg-[#0a0a0a] text-[#f2f2f2]">
          <AppMark className="h-5" />
        </span>
      ) : (
        <svg
          viewBox="0 0 24 24"
          className="size-6"
          fill="none"
          stroke="currentColor"
          strokeWidth={1.8}
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          {paths[picture.glyph]}
        </svg>
      )}
    </span>
  )
}
