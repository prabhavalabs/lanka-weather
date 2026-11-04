/**
 * Installing Lanka Weather as an app (ADR 0017).
 *
 * - `InstallHost`: mount once; it holds the guide and opens it for `/?install`.
 * - `InstallButton`: the "Install app" entry for menus.
 * - `openInstallGuide({ reason })`: opens the guide from anywhere, e.g. the
 *   push alerts flow with `reason: "alerts"` on an iPhone browser tab. Code
 *   that would rather not import this module can dispatch
 *   `new CustomEvent("lanka-weather:open-install-guide", { detail: { reason: "alerts" } })`
 *   on `window`.
 * - `registerAlertSettingsOpener(fn)`: lets the "installed" view link to the
 *   alert settings.
 */
export { InstallButton } from "./InstallButton"
export { InstallHost } from "./InstallHost"
export {
  closeInstallGuide,
  openInstallGuide,
  openInstallGuideEvent,
  promptInstall,
  registerAlertSettingsOpener,
  useInstallState,
  type GuideReason,
} from "./state"
export { currentPlatform, isStandalone, type Platform } from "./platform"
export { installLink, installParam } from "./links"
