import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { isLocale, localeOrder, locales, setLocale, useLocale, useT } from "@/lib/i18n"
import { cn } from "@/lib/utils"

/**
 * සිංහල / தமிழ் / English: each language named in itself, so a reader finds
 * theirs whatever the interface is in. The choice is saved on this device.
 */
export function LanguageSwitcher({ className }: { className?: string }) {
  const locale = useLocale()
  const t = useT()
  return (
    <ToggleGroup
      type="single"
      size="sm"
      value={locale}
      onValueChange={(value) => {
        // Radix sends "" when the pressed item is pressed again; keep the choice.
        if (isLocale(value)) setLocale(value)
      }}
      aria-label={t("common.language")}
      className={cn("w-full", className)}
    >
      {localeOrder.map((option) => (
        <ToggleGroupItem
          key={option}
          value={option}
          lang={locales[option].tag}
          // The English name helps someone who reads neither script.
          title={
            option === "en"
              ? locales[option].nativeName
              : `${locales[option].nativeName} (${locales[option].englishName})`
          }
          className="min-w-0 flex-1 px-2"
        >
          <span className="truncate">{locales[option].nativeName}</span>
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  )
}

/** One line while Sinhala or Tamil is chosen: the translations are still being reviewed. */
export function LanguageNote({ className }: { className?: string }) {
  const locale = useLocale()
  const t = useT()
  if (locale === "en") return null
  return (
    <p className={cn("text-xs text-muted-foreground", className)}>{t("common.languageNote")}</p>
  )
}
