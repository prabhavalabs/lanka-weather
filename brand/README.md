# Lanka Weather brand

## The mark

Radar watch: the island of Sri Lanka at the centre of three radar
rings, with the sweep passing through the north-east. It says what the
portal does: the whole country watched at once, so that warnings reach
people in time. The outline is traced from the provincial boundaries
and smoothed.

The island takes the text colour of wherever it sits: near-black on
light backgrounds, near-white on dark ones. The rings and the sweep are
always `#FF5A1F`.

## Files

All artwork is on a 64-unit grid. The `-light` files are for light
backgrounds and the `-dark` files for dark backgrounds.

| File                                                        | Use                                                                     |
| ----------------------------------------------------------- | ----------------------------------------------------------------------- |
| `mark-light.svg`, `mark-dark.svg`                           | The mark alone, transparent background                                  |
| `mark-small-light.svg`, `mark-small-dark.svg`               | One bold ring around a larger island, for 24 px and smaller             |
| `wordmark-light.svg`, `wordmark-dark.svg`                   | "Lanka Weather" in Geist SemiBold, outlined                             |
| `lockup-horizontal-light.svg`, `lockup-horizontal-dark.svg` | Mark and wordmark side by side                                          |
| `app-icon-dark.svg`, `app-icon-light.svg`                   | The mark on a rounded tile; the dark tile is the app icon               |
| `app-icon-small.svg`                                        | The small mark on the dark tile, for the 16 px favicon                  |
| `app-icon-maskable.svg`                                     | Full-bleed tile with the mark inside the maskable safe zone             |
| `badge.svg`                                                 | Island and two rings in solid white, for the Android notification badge |
| `favicon.svg`                                               | The small mark on the dark tile, which reads on light and dark tabs     |

The favicons, touch icon, PWA icons and web manifest in
`apps/web/public/` are generated from these files:

```bash
pnpm --filter @lanka-weather/web icons
```

In the app, `AppLogo` and `AppMark` (`apps/web/src/components/AppLogo.tsx`)
draw the same geometry inline and set the name in live Geist text.

## Usage

- Keep clear space around the mark of at least a quarter of its width.
- Use the full mark from 24 px up and the small mark below that; at
  16 px the faint rings and the sweep blur together.
- In the lockup the wordmark's cap height is about half the mark's
  height, and the caps are centred on the island.
- Do not recolour the rings, outline the island, rotate the mark or set
  the name in another typeface.
- On photographs or busy maps, place the mark on its tile.

## Colours

| Name          | Hex       | Use                                                                                                 |
| ------------- | --------- | --------------------------------------------------------------------------------------------------- |
| Ink           | `#0A0A0A` | Island on light backgrounds; dark tile; dark-theme cards                                            |
| Paper         | `#F2F2F2` | Island on dark backgrounds                                                                          |
| Signal orange | `#FF5A1F` | The rings and sweep; the accent colour of the interface (`#EC4F16` in the light theme for contrast) |

## Typefaces

- **Geist** for words and the wordmark.
- **Geist Mono** for every number, date and value; its zero is slashed.
- **Noto Sans Sinhala** and **Noto Sans Tamil** for Sinhala and Tamil
  text.

All four are open-source (SIL Open Font Licence) and self-hosted from
the `@fontsource-variable` packages.
