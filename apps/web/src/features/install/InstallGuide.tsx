import {
  BellRingIcon,
  CheckCircle2Icon,
  ChevronDownIcon,
  CopyIcon,
  DownloadIcon,
  ExternalLinkIcon,
  FeatherIcon,
  MaximizeIcon,
  WifiOffIcon,
} from "lucide-react"
import { useId, useMemo, useState, type ReactNode } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer"
import { ScrollArea } from "@/components/ui/scroll-area"
import { useT, type MessageKey, type Translate } from "@/lib/i18n"
import { desktopQuery, useMediaQuery } from "@/lib/useMediaQuery"
import { cn } from "@/lib/utils"
import { guides, splitGuide } from "./guides"
import { installLink } from "./links"
import { Pictogram } from "./Pictogram"
import {
  currentPlatform,
  deviceOf,
  guideBrowsers,
  guideStart,
  type Device,
  type GuideId,
  type InAppBrowser,
  type Platform,
} from "./platform"
import {
  closeInstallGuide,
  openAlertSettings,
  promptInstall,
  useInstallState,
  type InstallState,
} from "./state"

const appNames: Record<Exclude<InAppBrowser, "webview">, string> = {
  facebook: "Facebook",
  messenger: "Messenger",
  instagram: "Instagram",
  tiktok: "TikTok",
  line: "LINE",
  whatsapp: "WhatsApp",
  snapchat: "Snapchat",
  wechat: "WeChat",
  twitter: "X",
  linkedin: "LinkedIn",
  viber: "Viber",
  google: "Google",
}

const devices: Device[] = ["iphone", "android", "computer"]

/**
 * "Install Lanka Weather": a dialog on wide screens and a bottom drawer on
 * phones. Mounted once (InstallHost); opened with openInstallGuide().
 */
export function InstallGuide() {
  const state = useInstallState()
  const desktop = useMediaQuery(desktopQuery, true)
  const t = useT()
  const onOpenChange = (open: boolean) => {
    if (!open) closeInstallGuide()
  }
  const body = state.open ? <GuideBody state={state} /> : null

  if (desktop) {
    return (
      <Dialog open={state.open} onOpenChange={onOpenChange}>
        <DialogContent className="flex max-h-[min(46rem,calc(100dvh-2rem))] flex-col gap-0 p-0 sm:max-w-md">
          <DialogHeader className="gap-1 px-6 pt-6 pb-3 text-left">
            <DialogTitle className="text-lg">{t("install.title")}</DialogTitle>
            <DialogDescription>{t("install.description")}</DialogDescription>
          </DialogHeader>
          <ScrollArea type="auto" className="flex min-h-0 flex-col" viewportClassName="min-h-0">
            <div className="px-6 pb-6">{body}</div>
          </ScrollArea>
        </DialogContent>
      </Dialog>
    )
  }

  return (
    <Drawer open={state.open} onOpenChange={onOpenChange}>
      <DrawerContent className="max-h-[92dvh]">
        <DrawerHeader className="gap-1 px-5 text-left group-data-[vaul-drawer-direction=bottom]/drawer-content:text-left">
          <DrawerTitle className="text-lg">{t("install.title")}</DrawerTitle>
          <DrawerDescription>{t("install.description")}</DrawerDescription>
        </DrawerHeader>
        <ScrollArea className="flex min-h-0 flex-1 flex-col" viewportClassName="min-h-0 flex-1">
          <div className="px-5 pb-6">{body}</div>
        </ScrollArea>
      </DrawerContent>
    </Drawer>
  )
}

function GuideBody({ state }: { state: InstallState }) {
  const t = useT()
  const platform = useMemo(() => currentPlatform(), [])
  const start = guideStart(platform)
  const detected = start.kind === "steps" ? start.guide : null
  const [chosen, setChosen] = useState<GuideId | null>(null)
  const [chooserOpen, setChooserOpen] = useState(start.kind === "steps" && !start.exact)

  if (start.kind === "installed" || state.installed) {
    return (
      <div className="flex flex-col gap-5">
        <Installed state={state} />
        <WhyInstall />
      </div>
    )
  }

  const shown: GuideId = chosen ?? detected ?? `${deviceOf(platform)}-chrome`
  const isDetected = shown === detected
  const alertsFirst = state.reason === "alerts" && platform.os === "ios"

  return (
    <div className="flex flex-col gap-5">
      {alertsFirst && (
        <Callout icon={<BellRingIcon />} tone="brand">
          {t("install.alertsReason")}
        </Callout>
      )}
      {start.kind === "in-app" && <InApp app={start.app} platform={platform} />}
      {start.kind === "in-app" && !chosen ? null : (
        <Steps
          guide={shown}
          canPrompt={state.canPrompt && isDetected && splitGuide(shown).device !== "iphone"}
          nearest={isDetected && start.kind === "steps" && !start.exact}
        />
      )}
      <Chooser
        open={chooserOpen || start.kind === "in-app"}
        onOpenChange={setChooserOpen}
        shown={chosen ?? (start.kind === "in-app" ? null : shown)}
        defaultDevice={deviceOf(platform)}
        onChoose={setChosen}
      />
      <WhyInstall />
    </div>
  )
}

function Callout({
  icon,
  tone = "muted",
  children,
}: {
  icon: ReactNode
  tone?: "muted" | "brand"
  children: ReactNode
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-3 rounded-2xl border px-4 py-3 text-sm [&>svg]:mt-0.5 [&>svg]:size-4 [&>svg]:shrink-0",
        tone === "brand"
          ? "border-brand/40 bg-brand-soft text-brand-soft-foreground [&>svg]:text-brand-text"
          : "border-border bg-muted/50 text-muted-foreground",
      )}
    >
      {icon}
      <p className="min-w-0">{children}</p>
    </div>
  )
}

function Steps({
  guide,
  canPrompt,
  nearest,
}: {
  guide: GuideId
  canPrompt: boolean
  nearest: boolean
}) {
  const t = useT()
  const { device, browser } = splitGuide(guide)
  const { steps, note } = guides[guide]
  const headingId = useId()

  return (
    <section aria-labelledby={headingId} className="flex flex-col gap-3">
      {canPrompt && (
        <div className="flex flex-col gap-2 rounded-2xl border border-border bg-muted/40 p-4">
          <Button
            size="lg"
            variant="brand"
            className="h-12 w-full text-base"
            onClick={() => void promptInstall()}
          >
            <DownloadIcon className="size-5" />
            {t("install.installNow")}
          </Button>
          <p className="text-center text-xs text-muted-foreground">{t("install.installNowHint")}</p>
        </div>
      )}
      <h3 id={headingId} className="text-sm font-semibold">
        {canPrompt
          ? t("install.orSteps")
          : t("install.stepsFor", {
              device: deviceLabel(t, device),
              browser: browserLabel(t, browser),
            })}
      </h3>
      {nearest && <p className="text-xs text-muted-foreground">{t("install.nearest")}</p>}
      <ol className="flex flex-col gap-2">
        {steps.map((step, index) => (
          <li
            key={step.text}
            className="flex items-center gap-3 rounded-2xl border border-border bg-card px-3 py-2.5"
          >
            <span
              aria-hidden
              className="numeric flex size-7 shrink-0 items-center justify-center rounded-full bg-primary text-[0.8125rem] font-semibold text-primary-foreground"
            >
              {index + 1}
            </span>
            <p className="min-w-0 flex-1 text-[0.9375rem] leading-snug break-words">
              <span className="sr-only">{t("install.step", { number: index + 1 })}: </span>
              {t(step.text)}
            </p>
            <Pictogram picture={step.picture} />
          </li>
        ))}
      </ol>
      {note && (
        <Callout icon={<BellRingIcon />}>
          <span>{t(note)}</span>
        </Callout>
      )}
    </section>
  )
}

function deviceLabel(t: Translate, device: Device): string {
  return t(`install.devices.${device}`)
}

function browserLabel(t: Translate, browser: string): string {
  return t(`install.browsers.${browser}` as MessageKey)
}

function Chooser({
  open,
  onOpenChange,
  shown,
  defaultDevice,
  onChoose,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  shown: GuideId | null
  defaultDevice: Device
  onChoose: (guide: GuideId) => void
}) {
  const t = useT()
  const panelId = useId()
  const current = shown ? splitGuide(shown) : null
  const [device, setDevice] = useState<Device>(current?.device ?? defaultDevice)
  const activeDevice = current?.device ?? device

  return (
    <section className="flex flex-col gap-3">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => onOpenChange(!open)}
        className="-mx-1 flex min-h-11 items-center justify-between gap-3 rounded-xl px-1 text-left text-sm font-medium text-foreground outline-none hover:text-brand-text focus-visible:ring-2 focus-visible:ring-ring"
      >
        <span className="min-w-0">{t("install.otherDevice")}</span>
        <ChevronDownIcon
          aria-hidden
          className={cn("size-4 shrink-0 transition-transform", open && "rotate-180")}
        />
      </button>
      {open && (
        <div id={panelId} className="flex flex-col gap-3">
          <ChipGroup label={t("install.device")}>
            {devices.map((option) => (
              <Chip
                key={option}
                pressed={activeDevice === option}
                onClick={() => {
                  setDevice(option)
                  onChoose(`${option}-${guideBrowsers[option][0]}` as GuideId)
                }}
              >
                {deviceLabel(t, option)}
              </Chip>
            ))}
          </ChipGroup>
          <ChipGroup label={t("install.browser")}>
            {guideBrowsers[activeDevice].map((browser) => {
              const id = `${activeDevice}-${browser}` as GuideId
              return (
                <Chip key={id} pressed={shown === id} onClick={() => onChoose(id)}>
                  {browserLabel(t, browser)}
                </Chip>
              )
            })}
          </ChipGroup>
        </div>
      )}
    </section>
  )
}

function ChipGroup({ label, children }: { label: string; children: ReactNode }) {
  const id = useId()
  return (
    <div role="group" aria-labelledby={id} className="flex flex-col gap-1.5">
      <span id={id} className="text-xs text-muted-foreground">
        {label}
      </span>
      <div className="flex flex-wrap gap-2">{children}</div>
    </div>
  )
}

function Chip({
  pressed,
  onClick,
  children,
}: {
  pressed: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      aria-pressed={pressed}
      onClick={onClick}
      className={cn(
        "min-h-11 rounded-full border px-4 text-sm font-medium transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring",
        pressed
          ? "border-primary bg-primary text-primary-foreground"
          : "border-border bg-card text-foreground hover:bg-accent",
      )}
    >
      {children}
    </button>
  )
}

async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    // In-app browsers often refuse the clipboard API; a selected text area still copies.
    const area = document.createElement("textarea")
    area.value = text
    area.setAttribute("readonly", "")
    area.style.position = "fixed"
    area.style.opacity = "0"
    document.body.appendChild(area)
    area.select()
    const copied = document.execCommand("copy")
    area.remove()
    return copied
  }
}

function InApp({ app, platform }: { app: InAppBrowser; platform: Platform }) {
  const t = useT()
  const link = installLink()
  const android = platform.os === "android"
  // Android opens an intent: link in Chrome, from most in-app browsers.
  const chromeIntent = `intent://${window.location.host}/?install#Intent;scheme=https;package=com.android.chrome;end`

  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-status-alert/50 bg-status-alert/8 p-4">
      <h3 className="text-[0.9375rem] font-semibold">{t("install.inApp.title")}</h3>
      <p className="text-sm text-muted-foreground">
        {app === "webview"
          ? t("install.inApp.bodyUnknown")
          : t("install.inApp.body", { app: appNames[app] })}
      </p>
      <div className="flex items-center gap-3">
        <p className="min-w-0 flex-1 text-[0.9375rem] leading-snug">
          {t(android ? "install.inApp.android" : "install.inApp.ios")}
        </p>
        <Pictogram picture={{ glyph: android ? "kebab" : "ellipsis" }} />
      </div>
      <div className="flex flex-wrap gap-2">
        <Button
          size="lg"
          className="h-12 flex-1"
          onClick={async () => {
            const copied = await copyText(link)
            if (copied) toast.success(t("install.inApp.copied"), { id: "install-copy" })
            else toast.error(t("install.inApp.copyFailed"), { id: "install-copy" })
          }}
        >
          <CopyIcon />
          {t("install.inApp.copy")}
        </Button>
        {android && (
          <Button asChild size="lg" variant="secondary" className="h-12 flex-1">
            <a href={chromeIntent}>
              <ExternalLinkIcon />
              {t("install.inApp.openChrome")}
            </a>
          </Button>
        )}
      </div>
      <p className="numeric truncate text-xs text-muted-foreground select-all">{link}</p>
    </section>
  )
}

function Installed({ state }: { state: InstallState }) {
  const t = useT()
  return (
    <section className="flex flex-col gap-3 rounded-2xl border border-border bg-muted/40 p-4">
      <div className="flex items-center gap-2.5">
        <CheckCircle2Icon aria-hidden className="size-5 shrink-0 text-status-normal" />
        <h3 className="text-[0.9375rem] font-semibold">{t("install.installed.title")}</h3>
      </div>
      <p className="text-sm text-muted-foreground">{t("install.installed.body")}</p>
      <p className="text-sm">{t("install.installed.alerts")}</p>
      {state.hasAlertSettings && (
        <Button size="lg" variant="brand" className="h-12" onClick={openAlertSettings}>
          <BellRingIcon />
          {t("install.installed.alertSettings")}
        </Button>
      )}
    </section>
  )
}

function WhyInstall() {
  const t = useT()
  const reasons = [
    { icon: <BellRingIcon />, text: t("install.why.alerts") },
    { icon: <MaximizeIcon />, text: t("install.why.fullScreen") },
    { icon: <WifiOffIcon />, text: t("install.why.offline") },
    { icon: <FeatherIcon />, text: t("install.why.small") },
  ]
  return (
    <section className="flex flex-col gap-2.5 border-t border-border pt-4">
      <h3 className="text-sm font-semibold">{t("install.why.title")}</h3>
      <ul className="grid gap-2 sm:grid-cols-2">
        {reasons.map((reason) => (
          <li
            key={reason.text}
            className="flex items-start gap-2.5 text-sm text-muted-foreground [&>svg]:mt-0.5 [&>svg]:size-4 [&>svg]:shrink-0 [&>svg]:text-foreground"
          >
            {reason.icon}
            <span className="min-w-0">{reason.text}</span>
          </li>
        ))}
      </ul>
    </section>
  )
}
