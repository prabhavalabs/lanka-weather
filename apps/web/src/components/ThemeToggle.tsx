import { MonitorIcon, MoonIcon, SunIcon } from "lucide-react"

import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useT, type MessageKey } from "@/lib/i18n"
import { isThemeMode, useTheme, type ThemeMode } from "@/lib/theme"
import { cn } from "@/lib/utils"

const options: { mode: ThemeMode; label: MessageKey; Icon: typeof SunIcon }[] = [
  { mode: "light", label: "app.theme.light", Icon: SunIcon },
  { mode: "dark", label: "app.theme.dark", Icon: MoonIcon },
  { mode: "system", label: "app.theme.system", Icon: MonitorIcon },
]

/** Light, dark or the device's setting; the choice is saved in this browser. */
export function ThemeToggle({
  className,
  showLabels = false,
}: {
  className?: string
  /** Shows words beside the icons, e.g. in a settings sheet. */
  showLabels?: boolean
}) {
  const { mode, setMode } = useTheme()
  const t = useT()

  return (
    <ToggleGroup
      type="single"
      size="sm"
      value={mode}
      onValueChange={(value) => {
        // Radix sends "" when the pressed item is pressed again; keep the choice.
        if (isThemeMode(value)) setMode(value)
      }}
      aria-label={t("common.appearance")}
      className={cn(showLabels && "w-full", className)}
    >
      {options.map(({ mode: option, label: key, Icon }) => {
        const label = t(key)
        return showLabels ? (
          <ToggleGroupItem
            key={option}
            value={option}
            className="min-w-0 flex-1"
            aria-label={label}
            title={label}
          >
            <Icon />
            <span className="truncate">{option === "system" ? t("app.theme.auto") : label}</span>
          </ToggleGroupItem>
        ) : (
          <Tooltip key={option}>
            <TooltipTrigger asChild>
              <ToggleGroupItem value={option} aria-label={label}>
                <Icon />
              </ToggleGroupItem>
            </TooltipTrigger>
            <TooltipContent side="bottom">{label}</TooltipContent>
          </Tooltip>
        )
      })}
    </ToggleGroup>
  )
}
