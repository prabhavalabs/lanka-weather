/**
 * The design kit: every component of the design system with placeholder
 * data, in the current theme. Development only, at /?kit.
 */
import * as React from "react"
import {
  BellIcon,
  CloudRainIcon,
  EllipsisIcon,
  GaugeIcon,
  LayersIcon,
  PlusIcon,
  WindIcon,
  WavesIcon,
} from "lucide-react"
import { toast } from "sonner"

import { AppLogo, AppMark } from "@/components/AppLogo"
import { DottedGrid } from "@/components/DottedGrid"
import { IconTile } from "@/components/IconTile"
import { LabeledRow, LabeledRows } from "@/components/LabeledRow"
import { LegendSwatch } from "@/components/LegendSwatch"
import { Panel } from "@/components/Panel"
import { ProgressBar } from "@/components/ProgressBar"
import { SegmentedBar } from "@/components/SegmentedBar"
import { StatValue } from "@/components/StatValue"
import { ThemeToggle } from "@/components/ThemeToggle"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Kbd } from "@/components/ui/kbd"
import { Label } from "@/components/ui/label"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Separator } from "@/components/ui/separator"
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"
import { Slider } from "@/components/ui/slider"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { Toaster } from "@/components/ui/sonner"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { AdminAreaPicker } from "@/components/weather/AdminAreaPicker"
import { AdminLevelSelector } from "@/components/weather/AdminLevelSelector"
import { AlertList } from "@/components/weather/AlertList"
import {
  sampleAlerts,
  sampleCurrent,
  sampleDaily,
  sampleDistricts,
  sampleEstimates,
  sampleForecast,
  samplePlace,
  sampleProvinces,
  sampleSearchResults,
  sampleWind,
} from "@/components/weather/fixtures"
import { DailyForecast } from "@/components/weather/DailyForecast"
import { ForecastStrip } from "@/components/weather/ForecastStrip"
import { GaugeList } from "@/components/weather/GaugeList"
import { LocationWeatherPanel } from "@/components/weather/LocationWeatherPanel"
import { RainfallLegend } from "@/components/weather/RainfallLegend"
import { SearchCommand, SearchTrigger } from "@/components/weather/SearchCommand"
import { TemperatureLegend } from "@/components/weather/TemperatureLegend"
import type { AdminLevel } from "@/components/weather/types"
import { WindCompass } from "@/components/weather/WindCompass"
import { WindLegend } from "@/components/weather/WindLegend"
import { RiverLegend } from "@/features/rivers/RiverLegend"

const swatches = [
  "background",
  "card",
  "muted",
  "secondary",
  "accent",
  "border",
  "foreground",
  "muted-foreground",
  "primary",
  "brand",
  "brand-soft",
  "brand-text",
  "destructive",
  "ring",
]
const statuses = ["normal", "alert", "minor-flood", "major-flood", "unknown"]

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">{title}</h2>
      {children}
    </section>
  )
}

export default function KitPage() {
  const [range, setRange] = React.useState("6h")
  const [level, setLevel] = React.useState<AdminLevel>("district")
  const [area, setArea] = React.useState<string | null>("district-ratnapura")
  const [searchOpen, setSearchOpen] = React.useState(false)
  const [query, setQuery] = React.useState("")
  const [alerts, setAlerts] = React.useState(true)
  const [threshold, setThreshold] = React.useState([60])

  const results = sampleSearchResults.filter((result) =>
    `${result.title} ${result.subtitle ?? ""}`.toLowerCase().includes(query.toLowerCase()),
  )

  return (
    <TooltipProvider>
      <Toaster position="top-center" />
      <div className="min-h-dvh bg-background text-foreground">
        <header className="sticky top-0 z-20 flex items-center gap-4 border-b border-border bg-background/90 px-6 py-3 backdrop-blur">
          <AppLogo tagline="Design kit" className="flex-1" />
          <ThemeToggle />
        </header>

        <div className="mx-auto flex max-w-6xl flex-col gap-12 px-6 py-10">
          <Section title="Brand">
            <div className="flex flex-wrap items-center gap-8">
              <AppLogo />
              <AppLogo variant="tile" />
              <AppMark title="Mark" className="size-16" />
              <AppMark title="Mark, simple" simple className="size-4" />
              <AppMark title="Mark" className="size-4" />
            </div>
          </Section>

          <Section title="Colour">
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-7">
              {swatches.map((token) => (
                <div key={token} className="flex flex-col gap-1.5">
                  <span
                    className="h-12 rounded-xl border border-border"
                    style={{ backgroundColor: `var(--${token})` }}
                  />
                  <span className="numeric text-[0.6875rem] text-muted-foreground">{token}</span>
                </div>
              ))}
            </div>
            <div className="flex flex-wrap gap-3">
              {statuses.map((status) => (
                <span
                  key={status}
                  className="numeric rounded-full px-3 py-1 text-xs"
                  style={{
                    backgroundColor: `var(--status-${status})`,
                    color: `var(--status-${status}-foreground)`,
                  }}
                >
                  {status}
                </span>
              ))}
            </div>
            <div className="flex flex-wrap gap-6">
              {(["temp", "rain", "wind"] as const).map((ramp) => (
                <div key={ramp} className="flex items-center gap-1">
                  <span className="numeric mr-2 text-xs text-muted-foreground">{ramp}</span>
                  {Array.from({ length: ramp === "rain" ? 6 : 7 }, (_, index) => (
                    <span
                      key={index}
                      className="size-6 rounded-md"
                      style={{ backgroundColor: `var(--${ramp}-${index + 1})` }}
                    />
                  ))}
                </div>
              ))}
            </div>
          </Section>

          <Section title="Type">
            <div className="flex flex-col gap-2">
              <p className="text-3xl font-medium tracking-tight">Geist for words</p>
              <p className="numeric text-3xl">Geist Mono 0123456789 · 18,240.60</p>
              <p className="text-base">Sinhala and Tamil fall back to Noto: ශ්‍රී ලංකාව · இலங்கை</p>
            </div>
          </Section>

          <Section title="Buttons and controls">
            <div className="flex flex-wrap items-center gap-3">
              <Button>
                <PlusIcon data-icon="inline-start" />
                Add station
              </Button>
              <Button variant="secondary">Not now</Button>
              <Button variant="outline">Outline</Button>
              <Button variant="ghost">Ghost</Button>
              <Button variant="brand">Brand</Button>
              <Button variant="destructive">Remove</Button>
              <Button variant="link">Link</Button>
              <Button variant="outline" size="icon" aria-label="More">
                <EllipsisIcon />
              </Button>
              <Button size="sm">Small</Button>
              <Button size="lg">Large</Button>
              <Button disabled>Disabled</Button>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Badge>Default</Badge>
              <Badge variant="secondary">Secondary</Badge>
              <Badge variant="outline">Outline</Badge>
              <Badge variant="chip">0 of 10</Badge>
              <Badge variant="brand">Selected</Badge>
              <Badge variant="normal">Normal</Badge>
              <Badge variant="alert">Alert</Badge>
              <Badge variant="minor_flood">Minor flood</Badge>
              <Badge variant="major_flood">Major flood</Badge>
              <Badge variant="unknown">No thresholds</Badge>
            </div>
            <div className="flex flex-wrap items-center gap-6">
              <ToggleGroup
                type="single"
                value={range}
                onValueChange={(value) => value && setRange(value)}
                aria-label="Time range"
              >
                {["1h", "6h", "24h", "72h"].map((option) => (
                  <ToggleGroupItem key={option} value={option} className="numeric">
                    {option}
                  </ToggleGroupItem>
                ))}
              </ToggleGroup>
              <div className="flex items-center gap-2">
                <Switch id="kit-switch" checked={alerts} onCheckedChange={setAlerts} />
                <Label htmlFor="kit-switch">Alerts</Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch id="kit-switch-off" defaultChecked={false} size="sm" />
                <Label htmlFor="kit-switch-off">Small</Label>
              </div>
              <div className="flex items-center gap-2">
                <Checkbox id="kit-check" defaultChecked />
                <Label htmlFor="kit-check">Checkbox</Label>
              </div>
              <div className="flex w-56 flex-col gap-2">
                <Slider
                  value={threshold}
                  onValueChange={setThreshold}
                  max={100}
                  aria-label="Threshold"
                />
                <span className="numeric text-xs text-muted-foreground">{threshold[0]}%</span>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Input placeholder="Goal" className="w-56" />
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="outline">Tooltip</Button>
                </TooltipTrigger>
                <TooltipContent>
                  <span className="numeric">Oct · 1,500.00</span>
                </TooltipContent>
              </Tooltip>
              <Popover>
                <PopoverTrigger asChild>
                  <Button variant="outline">Popover</Button>
                </PopoverTrigger>
                <PopoverContent>
                  <p className="text-sm">Popovers use the popover surface and a hairline.</p>
                </PopoverContent>
              </Popover>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline">Menu</Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent>
                  <DropdownMenuLabel>Station</DropdownMenuLabel>
                  <DropdownMenuItem>Zoom to station</DropdownMenuItem>
                  <DropdownMenuItem>Copy link</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem variant="destructive">Hide</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <Dialog>
                <DialogTrigger asChild>
                  <Button variant="outline">Dialog</Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Share this view</DialogTitle>
                    <DialogDescription>
                      Anyone with the link sees the same layers.
                    </DialogDescription>
                  </DialogHeader>
                  <DialogFooter>
                    <Button variant="secondary">Cancel</Button>
                    <Button>Copy link</Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
              <Sheet>
                <SheetTrigger asChild>
                  <Button variant="outline">Sheet</Button>
                </SheetTrigger>
                <SheetContent>
                  <SheetHeader>
                    <SheetTitle>Settings</SheetTitle>
                    <SheetDescription>A side sheet.</SheetDescription>
                  </SheetHeader>
                  <div className="px-4">
                    <ThemeToggle showLabels />
                  </div>
                </SheetContent>
              </Sheet>
              <Button
                variant="outline"
                onClick={() =>
                  toast("River levels updated", { description: "40 stations · 14:00" })
                }
              >
                Toast
              </Button>
              <span className="flex items-center gap-1 text-sm text-muted-foreground">
                Press <Kbd>⌘K</Kbd>
              </span>
            </div>
          </Section>

          <Section title="Cards">
            <div className="grid gap-6 md:grid-cols-2">
              <Panel
                icon={<WavesIcon />}
                title="Ratnapura"
                subtitle="Kalu Ganga · river gauge"
                action={
                  <Button variant="outline" size="icon" aria-label="More">
                    <EllipsisIcon />
                  </Button>
                }
                footer={
                  <>
                    <span>Next reading</span>
                    <span className="numeric text-foreground">Mon 3 Nov · 15:00</span>
                  </>
                }
              >
                <StatValue
                  label="Water level"
                  value="7.82"
                  unit="m"
                  delta="↑ +0.13 m/h"
                  deltaLabel="rising"
                  caption="0.68 m below major flood level"
                  aside="64%"
                />
                <ProgressBar value={0.64} label="Towards major flood" />
                <LabeledRows>
                  <LabeledRow
                    leading={<LegendSwatch color="var(--status-alert)" />}
                    label="Alert"
                    value="5.20 m"
                  />
                  <LabeledRow
                    leading={<LegendSwatch color="var(--status-minor-flood)" />}
                    label="Minor flood"
                    value="7.50 m"
                  />
                  <LabeledRow
                    leading={<LegendSwatch color="var(--status-major-flood)" />}
                    label="Major flood"
                    value="8.50 m"
                  />
                  <LabeledRow label="Rate change" value="+0.15 m" tone="signal" />
                </LabeledRows>
              </Panel>

              <Panel
                icon={<GaugeIcon />}
                title="River gauges"
                subtitle="40 stations · 2 above alert"
                action={
                  <Button variant="outline" size="icon" aria-label="Add">
                    <PlusIcon />
                  </Button>
                }
              >
                <StatValue value="40" unit="stations" delta="+2" deltaLabel="since 06:00" />
                <SegmentedBar
                  label="Stations by status"
                  total={44}
                  segments={[
                    { label: "Major flood", value: 1, color: "var(--status-major-flood)" },
                    { label: "Minor flood", value: 1, color: "var(--status-minor-flood)" },
                    { label: "Alert", value: 3, color: "var(--status-alert)" },
                    { label: "Normal", value: 34, color: "var(--chart-2)" },
                    { label: "No thresholds", value: 1, color: "var(--chart-4)" },
                  ]}
                />
                <div className="flex flex-wrap gap-x-4 gap-y-1.5">
                  <LegendSwatch color="var(--status-major-flood)" label="Major" value={1} />
                  <LegendSwatch color="var(--status-minor-flood)" label="Minor" value={1} />
                  <LegendSwatch color="var(--status-alert)" label="Alert" value={3} />
                  <LegendSwatch color="var(--chart-2)" label="Normal" value={34} />
                </div>
                <DottedGrid className="flex h-36 items-end justify-between gap-1.5 px-4 pb-4">
                  {[3, 5, 4, 7, 9, 6, 12, 15, 11, 8, 5, 4].map((value, index) => (
                    <span
                      key={index}
                      className="flex-1 rounded-full"
                      style={{
                        height: `${value * 6}%`,
                        backgroundColor: index === 7 ? "var(--brand)" : "var(--chart-3)",
                      }}
                    />
                  ))}
                </DottedGrid>
              </Panel>

              <Panel
                icon={<BellIcon />}
                title="Alerts"
                subtitle="2 rules · this device"
                action={<Switch checked={alerts} onCheckedChange={setAlerts} aria-label="Alerts" />}
              >
                <div className="flex flex-col gap-2 rounded-2xl bg-muted p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium">Kalu Ganga</p>
                      <p className="text-sm text-muted-foreground">When above alert level</p>
                    </div>
                    <Switch defaultChecked aria-label="Kalu Ganga alerts" />
                  </div>
                </div>
                <div className="flex gap-3">
                  <Button className="flex-1">
                    <PlusIcon data-icon="inline-start" />
                    Add rule
                  </Button>
                  <Button variant="secondary" className="flex-1">
                    Not now
                  </Button>
                </div>
              </Panel>

              <Panel icon={<LayersIcon />} title="Loading" subtitle="Skeletons">
                <Skeleton className="h-9 w-40" />
                <Skeleton className="h-2.5 w-full rounded-full" />
                <Skeleton className="h-24 w-full rounded-2xl" />
              </Panel>
            </div>
          </Section>

          <Section title="Weather shells">
            <div className="flex flex-wrap items-center gap-4">
              <SearchTrigger className="max-w-sm" onClick={() => setSearchOpen(true)} />
              <SearchTrigger compact onClick={() => setSearchOpen(true)} />
              <div className="flex flex-wrap items-center gap-2">
                <AdminLevelSelector
                  level={level}
                  onLevelChange={setLevel}
                  disabledLevels={["gn"]}
                />
                <AdminAreaPicker
                  level={level}
                  areas={level === "province" ? sampleProvinces : sampleDistricts}
                  selectedAreaId={area}
                  onAreaChange={setArea}
                />
              </div>
            </div>
            <SearchCommand
              open={searchOpen}
              onOpenChange={setSearchOpen}
              query={query}
              onQueryChange={setQuery}
              results={results}
              recent={sampleSearchResults.slice(0, 3)}
              onSelect={(result) => {
                setSearchOpen(false)
                toast(`Selected ${result.title}`)
              }}
            />
            <div className="grid items-start gap-6 lg:grid-cols-[24rem_1fr]">
              <LocationWeatherPanel
                place={samplePlace}
                current={sampleCurrent}
                forecast={sampleForecast}
                daily={sampleDaily}
                today="2026-11-03"
                wind={sampleWind}
                alerts={sampleEstimates}
                officialAlerts={sampleAlerts}
                sources="Placeholder data"
                onClose={() => toast("Closed")}
              >
                <GaugeList
                  title="River gauges nearby"
                  items={[
                    {
                      id: "ratnapura",
                      title: "Ratnapura",
                      subtitle: "Kalu Ganga · 2.1 km",
                      value: "7.82 m ↑",
                      badge: { variant: "minor_flood", label: "Minor flood" },
                      onSelect: () => toast("Station"),
                    },
                  ]}
                />
              </LocationWeatherPanel>
              <div className="flex flex-col gap-6">
                <Panel icon={<CloudRainIcon />} title="Next 48 hours" subtitle="Ratnapura">
                  <ForecastStrip hours={sampleForecast} />
                </Panel>
                <Panel icon={<CloudRainIcon />} title="7 days" subtitle="Ratnapura">
                  <DailyForecast days={sampleDaily} today="2026-11-03" />
                </Panel>
                <div className="flex flex-wrap items-start gap-6">
                  <Panel icon={<WindIcon />} title="Wind" subtitle="From the south-west">
                    <WindCompass speedKmh={24} gustKmh={46} fromDegrees={225} />
                  </Panel>
                  <div className="flex min-w-72 flex-1 flex-col gap-3">
                    <AlertList alerts={sampleAlerts} />
                    <AlertList alerts={sampleEstimates} label="Estimates" />
                    <AlertList alerts={[]} />
                  </div>
                </div>
                <DottedGrid className="flex flex-wrap items-end gap-3 p-4">
                  <TemperatureLegend />
                  <WindLegend />
                  <RainfallLegend />
                  <RiverLegend
                    counts={{ normal: 34, alert: 3, minor_flood: 1, major_flood: 1, unknown: 1 }}
                  />
                </DottedGrid>
              </div>
            </div>
          </Section>

          <Section title="Tabs and separators">
            <Tabs defaultValue="a" className="max-w-md">
              <TabsList>
                <TabsTrigger value="a">Today</TabsTrigger>
                <TabsTrigger value="b">Week</TabsTrigger>
              </TabsList>
              <TabsContent value="a">
                <IconTile tone="brand">
                  <WavesIcon />
                </IconTile>
              </TabsContent>
              <TabsContent value="b">Week view</TabsContent>
            </Tabs>
            <Tabs defaultValue="a" className="max-w-md">
              <TabsList variant="line">
                <TabsTrigger value="a">Line</TabsTrigger>
                <TabsTrigger value="b">Tabs</TabsTrigger>
              </TabsList>
            </Tabs>
            <Separator />
          </Section>
        </div>
      </div>
    </TooltipProvider>
  )
}
