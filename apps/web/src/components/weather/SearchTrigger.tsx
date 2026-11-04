import type * as React from "react"
import { SearchIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Kbd } from "@/components/ui/kbd"
import { useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

const isMac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform)

interface SearchTriggerProps extends React.ComponentProps<"button"> {
  /** Shows only the icon, e.g. in a narrow header. */
  compact?: boolean
}

/** The search box that opens the palette; shows the keyboard shortcut. */
export function SearchTrigger({ compact = false, className, ...props }: SearchTriggerProps) {
  const t = useT()
  const shortcut = isMac ? "Meta+K" : "Control+K"
  if (compact) {
    return (
      <Button
        variant="outline"
        size="icon"
        aria-label={t("weather.search.triggerLabel")}
        aria-keyshortcuts={shortcut}
        className={className}
        {...props}
      >
        <SearchIcon />
      </Button>
    )
  }
  // The box sizes itself to its room: a round icon button when the row is
  // tight, the prompt from 8 rem, and the shortcut too from 14 rem. A box
  // squeezed past its words would show neither.
  return (
    <div className={cn("@container/search flex min-w-10", className)}>
      <button
        type="button"
        className="flex h-10 w-full items-center justify-center gap-2.5 rounded-full border border-border bg-card/95 text-left text-sm text-muted-foreground shadow-panel backdrop-blur-md transition-colors outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring @min-[8rem]/search:justify-start @min-[8rem]/search:pr-2 @min-[8rem]/search:pl-3.5"
        aria-label={t("weather.search.triggerLabel")}
        aria-keyshortcuts={shortcut}
        {...props}
      >
        <SearchIcon className="size-4 shrink-0" aria-hidden="true" />
        <span className="hidden min-w-0 flex-1 truncate @min-[8rem]/search:block">
          {t("weather.search.trigger")}
        </span>
        <Kbd aria-hidden="true" className="hidden shrink-0 @min-[14rem]/search:inline-flex">
          {isMac ? t("weather.search.shortcutMac") : t("weather.search.shortcutOther")}
        </Kbd>
      </button>
    </div>
  )
}
