import { BellIcon, LandmarkIcon } from "lucide-react"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { badgeText } from "@/features/notifications/present"
import { useT } from "@/lib/i18n"

export type WarningsTab = "official" | "alerts"

interface WarningsTabsProps {
  tab: WarningsTab
  /** Official warnings and advisories in force. */
  officialCount: number
  /** Alerts this device was sent and has not seen. */
  unread: number
  onTabChange: (tab: WarningsTab) => void
}

/**
 * The two halves of one panel: what the Department of Meteorology, the DMC
 * and NBRO have issued, and the alerts this device was sent about them. One
 * place to look, with the source of each kept apart.
 */
export function WarningsTabs({ tab, officialCount, unread, onTabChange }: WarningsTabsProps) {
  const t = useT()
  const badge = badgeText(unread)
  return (
    <Tabs value={tab} onValueChange={(value) => onTabChange(value as WarningsTab)}>
      <TabsList className="w-full" aria-label={t("warnings.tabs.label")}>
        <TabsTrigger value="official">
          <LandmarkIcon />
          {t("warnings.tabs.official")}
          {officialCount > 0 && (
            <span className="numeric rounded-full bg-brand px-1.5 text-[0.625rem] leading-4 font-semibold text-brand-foreground">
              {officialCount}
            </span>
          )}
        </TabsTrigger>
        <TabsTrigger value="alerts">
          <BellIcon />
          {t("warnings.tabs.alerts")}
          {badge && (
            <span className="numeric rounded-full bg-destructive px-1.5 text-[0.625rem] leading-4 font-semibold text-white">
              {badge}
            </span>
          )}
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )
}
