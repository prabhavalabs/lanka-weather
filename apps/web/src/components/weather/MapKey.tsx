import { ListIcon } from "lucide-react"
import { useId, type ReactNode } from "react"
import { Button } from "@/components/ui/button"
import { useLocalStorage } from "@/lib/useLocalStorage"

/**
 * The key to what is on the map: one button over the map, opening the legends
 * of the layers that are on in a column that scrolls. A stack of open legend
 * cards would cover the map it explains.
 */
export function MapKey({ children }: { children: ReactNode }) {
  const [open, setOpen] = useLocalStorage("lanka-weather:map-key-open", false)
  const id = useId()
  return (
    <div className="pointer-events-auto flex flex-col items-start gap-2">
      {open && (
        <div
          id={id}
          className="flex max-h-[min(62dvh,34rem)] flex-col gap-2 overflow-y-auto overscroll-contain rounded-2xl [scrollbar-width:thin] *:shrink-0"
        >
          {children}
        </div>
      )}
      <Button
        variant="secondary"
        size="sm"
        aria-expanded={open}
        aria-controls={open ? id : undefined}
        className="rounded-full bg-card/95 shadow-panel backdrop-blur-md"
        onClick={() => setOpen((current) => !current)}
      >
        <ListIcon />
        {open ? "Hide map key" : "Map key"}
      </Button>
    </div>
  )
}
