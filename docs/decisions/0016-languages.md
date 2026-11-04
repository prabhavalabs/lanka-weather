# ADR 0016: The interface in Sinhala, Tamil and English

- Status: Accepted
- Date: 2026-11-03

## Context

Sri Lanka's official languages are Sinhala and Tamil, with English as the link language. The
Disaster Management Centre (DMC), the National Building Research Organisation (NBRO), the Department
of Meteorology and the Irrigation Department publish in all three. A warning map that speaks only
English leaves out most of the people it is for.

Commit 13d5c8a added a small language layer (`src/lib/i18n`) and catalogues in
`src/locales/{en,si,ta}`, with nothing translated yet. This record covers the rest: what is
translated and how, the words used, place names, dates and numbers, the map's own labels and fonts,
official bulletins, and what still reads in English.

## Decision

### Catalogues and rendering

- English is the source of keys. Each feature has one namespace file per language
  (`locales/en/floods.ts`, `locales/si/floods.ts`, `locales/ta/floods.ts`), registered in each
  `index.ts`. Sinhala and Tamil are typed `Translation<typeof en>`, so a missing key fails the type
  check, and `i18n.test.ts` checks that keys and `{placeholder}`s match and that no message is empty.
- Shared namespaces: `common` (buttons and small words), `format` (dates, ages, compass points),
  `terms` (the agencies' own words: river and landslide levels, warning colours, bulletin kinds,
  weather systems, reservoir status, administrative divisions, agency names, "estimate" and
  "official") and `layers` (the layer list and the catalogue's titles and descriptions, which the API
  sends in English and the app looks up by layer id, falling back to the API's text for a layer it
  does not know).
- Messages are whole sentences with named placeholders, because word order differs: Sinhala and
  Tamil put the verb last and the age before "updated". Plurals are `{ one, other, zero? }` with
  `count`, picked by `Intl.PluralRules`.
- Components call `useT()` (or `useLocale()`), which subscribes them to the language, so a switch
  re-renders everything that shows text with no reload. Plain helpers (`present.ts`) call
  `translate()` while a subscribed component renders. Memoised values and map layers that bake text
  in depend on the language.
- The choice is saved on the device (`lanka-weather:locale`). The first visit takes the device's
  language when it is Sinhala or Tamil, else English. `<html lang>` is set before first paint.

### Switching

A segmented control, සිංහල / தமிழ் / English, sits in the desktop sidebar footer under Appearance
and in the phone layer sheet's footer. Each language is named in itself so a reader finds theirs
whatever the interface is in. While Sinhala or Tamil is chosen, one line says the translations are
being reviewed (`common.languageNote`), until a native review is done.

### Words

- Official terms follow the issuers' trilingual bulletins and sites; `docs/i18n-glossary.md` lists
  each with its source and how sure we are. Changing a term in `terms` means changing the glossary.
- Derived figures read as estimates in every language (river risk, region and point alerts, model
  air quality, satellite rain), as the project's rules require, and official warnings as official.
- Acronyms stay as written: NBRO, DMC, USGS, GDACS, NOAA, GFS, NASA, IMERG, CAMS, Google, Flood Hub.
  Agencies are named by their official Sinhala and Tamil names where they have them. Units stay as
  they are (mm, km/h, m³/s, cusecs). The emergency number 117 is the same in every language.
- Sinhala follows written government usage with polite imperatives (…න්න). Tamil follows Sri Lankan
  usage (மாவட்டம், பிரதேச செயலகப் பிரிவு, அனர்த்தம், தாழமுக்கம், நீர்த்தேக்கம்) and Sri Lankan
  month spellings (பெப்ரவரி, ஓகஸ்ட், ஒக்டோபர்).

### Dates, times and numbers

- `lib/format.ts` reads the current language, always in Sri Lanka time (Asia/Colombo) and always in
  Western digits (`numberingSystem: "latn"`), as Sri Lankan Sinhala and Tamil papers print them.
- Month names come from the catalogues, not `Intl`: the Sinhala data in `Intl` names the Gregorian
  months after the lunar ones (ඉල් for November), which would mislead. Weekdays come from `Intl`.
- Clocks are 24-hour with a colon (14:00) in every language; Sinhala `Intl` writes 14.00, which reads
  like a decimal next to measurements.
- Numbers are grouped in threes (12,345) in every language. Tamil `Intl` data uses the Indian lakh
  grouping (12,34,567), which Sri Lankan Tamil text does not.
- Relative ages ("12 min ago") and compass points are catalogue messages; Sinhala and Tamil write
  directions as words (නිරිත, தென்மேற்கு), since they have no letter abbreviations.

### Names of places and features

- Proper names are chosen, never translated. The API gives `nameSi`/`nameTa` (or `names`) for
  administrative regions, places and the hierarchy above them, and `nameSi` for named rivers. The app
  shows the name in the reader's language and falls back to English (`localName`, `nameIn`,
  `localNameExpression` for map labels). Gauges, reservoirs, stations and canals have English names
  only in their sources, so they stay in English.
- Search matches Sinhala and Tamil: the API's place search already matches every name, and the
  client-side search of rivers, gauges and reservoirs folds case and joiners and compares every name a
  feature has.

### The basemap

- OpenFreeMap's OpenMapTiles tiles carry `name:ta` but not `name:si` (checked against the tile schema
  and decoded tiles of Colombo, Kandy and Jaffna on 3 November 2026: most Sri Lankan towns and cities
  have `name:ta`; none has `name:si`, and `name` is in Latin script). In Tamil, place, water, road and
  airport labels read `name:ta` and fall back to the style's own label; in Sinhala they would read
  `name:si` the same way and so show English today. Region, river and division names from our own
  data do show in Sinhala, in panels, search and the hover chip, and river and landslide labels on
  the map follow the language.
- MapLibre's own buttons (zoom, compass, locate, attribution) are labelled in the language when the
  map is made and relabelled in place on a switch.
- A switch relabels the style in place with `setLayoutProperty`, from the style's own `text-field`
  kept in each layer's metadata; the map is not reloaded.
- OpenFreeMap's glyph files have no Sinhala or Tamil, and a glyphs URL serves one codepoint at a time,
  where both scripts need a letter and its marks drawn together. The style is given `font-faces` that
  point every font it names at the app's own Noto Sans Sinhala and Noto Sans Tamil files (bundled
  with the app, same origin) for those Unicode ranges. MapLibre draws text covered by `font-faces` a
  grapheme cluster at a time with the browser's text engine, so vowel signs and conjuncts come out as
  written.

### Official bulletins

The DMC publishes separate English, Sinhala and Tamil editions of many bulletins, and the warnings
API carries each with its `language`. The warnings panel shows the edition in the reader's language
when there is one, else the published edition with a short note. Text extracted from Sinhala and
Tamil PDFs is usually unusable (legacy fonts), so where the reader's edition has no usable text the
English text is shown, marked as English, with the reader's edition's PDF. Titles and text are the
issuer's words and are never machine-translated.

### Layout

Tamil runs about half as long again as English, and both scripts need more line height. Line heights
rise for Sinhala and Tamil (`index.css`, keyed on `<html lang>`), one-line labels truncate with the
full text in a tooltip, and chip rows and segmented controls wrap or scroll rather than overlap. The
self-hosted Noto Sans Sinhala and Noto Sans Tamil variable fonts follow Geist in the font stack, so
each script is drawn in its own face.

## Consequences

- Every new user-visible string needs a key and three messages; the type check and the catalogue
  test stop a missing one.
- Some text still comes from the server in English and is shown as such, marked `lang="en"`: river
  risk reasons, notes on landslide reports and cyclones, overview lines the server writes, and data
  credits. Translating them needs the API to send structured reasons rather than sentences, a later
  change.
- Basemap labels stay English in Sinhala until the tiles carry `name:si`. Drawing our own place labels
  from the places table would fix that and is left for later.
- Sinhala and Tamil wording has been checked against official sources where they could be found but
  not yet by native reviewers; the note under the switcher stays until that review is done.
