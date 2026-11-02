import type * as React from "react"
import { SearchIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Kbd } from "@/components/ui/kbd"
import { cn } from "@/lib/utils"

const isMac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform)

interface SearchTriggerProps extends React.ComponentProps<"button"> {
  /** Shows only the icon, e.g. in a narrow header. */
  compact?: boolean
}

/** The search box that opens the palette; shows the keyboard shortcut. */
export function SearchTrigger({ compact = false, className, ...props }: SearchTriggerProps) {
  if (compact) {
    return (
      <Button
        variant="outline"
        size="icon"
        aria-label="Search places, rivers and areas"
        className={className}
        {...props}
      >
        <SearchIcon />
      </Button>
    )
  }
  return (
    <button
      type="button"
      className={cn(
        "flex h-10 w-full items-center gap-2.5 rounded-full border border-border bg-card pr-2 pl-3.5 text-left text-sm text-muted-foreground shadow-panel transition-colors outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring",
        className,
      )}
      {...props}
    >
      <SearchIcon className="size-4 shrink-0" aria-hidden="true" />
      <span className="flex-1 truncate">Search places, rivers, areas</span>
      <Kbd aria-hidden="true">{isMac ? "⌘K" : "Ctrl K"}</Kbd>
    </button>
  )
}
