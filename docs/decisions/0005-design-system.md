# ADR 0005: shadcn/ui with a token-based theme, light and dark

- Status: Accepted
- Date: 2026-11-02

## Context

The interface is a full-screen map with floating cards: a layer list,
station details and, soon, place weather, search, administrative
boundaries and legends. It has to read well in daylight on a phone and
at night on a dark screen, and hazard colours must mean the same thing
on the map, in panels and in legends. Several features will be built in
parallel, so they need shared components and one source of colours.

## Decision

- **Components:** shadcn/ui (Radix primitives, `radix-nova` style) in
  `apps/web/src/components/ui`, restyled: pill buttons, large-radius
  cards with hairline borders, segmented controls with an orange
  selected state, orange switches, dark tooltip chips. App composites
  (Panel, StatValue, LabeledRow, ProgressBar, SegmentedBar,
  LegendSwatch, DottedGrid, ThemeToggle, AppLogo) live beside them in
  `src/components`, and presentational weather shells in
  `src/components/weather`.
- **Tokens:** every colour is a CSS custom property in
  `src/styles/theme.css`, in hex, for `:root` (light) and `.dark`. The
  shadcn variables keep their meaning (`accent` is the quiet hover
  surface); the orange signal colour is `brand-*`. App tokens cover
  flood status, temperature, rainfall, wind and the basemap.
- **Contrast:** a unit test computes WCAG 2 contrast for text (4.5:1)
  and non-text (3:1) pairs in both themes, so a token change that breaks
  AA fails CI.
- **Theme:** light, dark or the device setting, saved in local storage
  and applied by an inline script before first paint. The map loads the
  matching OpenFreeMap style and retunes land, water and labels to the
  tokens as it loads (`transformStyle`); data layers read their colours
  from the same tokens on every style load.
- **Type:** Geist for text, Geist Mono for numbers, dates and values,
  Noto Sans Sinhala and Tamil for those scripts, all self-hosted.
- **Brand:** source artwork in `/brand`; icons and the web manifest are
  generated from it by a script and committed.
- **Design kit:** in development, `/?kit` renders every component with
  placeholder data in the current theme. Production builds leave it out.

## Consequences

- Features compose existing components instead of styling from scratch,
  and hazard colours change in one place.
- MapLibre cannot read CSS variables, so map layers must read tokens
  through `readToken`/`readTokens` and re-add themselves on
  `style.load`; a theme change reloads the basemap rather than diffing
  it.
- shadcn components are copied into the repository, so upstream fixes
  are pulled in by hand.
