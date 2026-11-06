# ADR 0019: A combined flood-risk estimate by district

- Status: Accepted
- Date: 2026-11-06

Extends ADR 0018 (push alerts) with a derived topic, and adds a heavy-rain alert from the live
gauges (ADR 0015).

## Context

On 6 November 2026 Colombo flooded in the evening and our flood notice arrived at 00:15 the next
day. The causes, from the production database:

- Google issued a Colombo flash-flood forecast at 06:10, which reached people at 07:09, then
  stopped returning it at 16:13, before the rain.
- From 17:50 to 18:20 the Department of Meteorology's head-office gauge measured at least 57 mm.
  Nothing turned measured rain into an alert.
- Google issued a new forecast at 20:15. Its API first returned it between 23:15 and 00:15, and our
  hourly poll stored it at 00:15. The notice did not say when Google issued it.
- Each of Google's reissues alerted again under a new area ID: five Colombo notices in four days.

Every source alone was either late (Google), silent (measured rain) or too coarse (bulletins). The
owner asked for an algorithm combining them that says what could happen.

## Decision

### Per-source fixes (alerts in their own right)

- Flash floods are read every 10 minutes by their own collector (`google-flash-floods`), beside the
  others, and a stored polygon is not read again. `flood_areas.first_seen_at` records the first
  sighting, and each one is logged with its delay from Google's issue time.
- Flash-flood alerts follow the district, not Google's area, and say when Google issued them.
- Heavy rain measured at the HMIS gauges is a new topic, `heavy_rain` (ADR 0018 has its levels).
  The gauges report rolling 24-hour totals, and the rise over a window is a lower bound on the
  rain in it.

### The combined estimate (`internal/floodrisk`)

Every district is scored from eight families of evidence, each 0 to 3 (notable, significant,
severe):

| Family | 1 | 2 | 3 |
|---|---|---|---|
| Rain measured now (largest rise at a current gauge, lower bound) | 15 mm in 1 h or 30 in 3 h | 30 or 60 | 50 or 100 |
| Ground already wet: the 24-hour total less the 3-hour rise | 50 mm | 100 | 200 |
| Rain forecast (Open-Meteo): wettest cell in 6 h, or district mean in 24 h | 30 or 75 mm | 60 or 150 | 100 or 250 |
| Google flash flood over the district | | likely | highly likely |
| Google verified river forecast in the district | Warning | Danger | Extreme |
| River gauges in the district | alert | minor flood | major flood |
| DMC or Department of Meteorology bulletin naming the district (weather, water level, situation) | advisory | warning, amber | red |
| Reservoirs spilling | 1 | 3 | 6 |

A district's level needs **two families to agree**. One source alone already has its own alert;
the estimate is for corroboration and lead time.

- **Moderate**, flooding possible: two families, 3 points between them. Severity watch.
- **High**, flooding likely: two families, 4 points, one of them at least 2. Severity warning.
- **Very high**: three families, 7 points. Still warning: danger is kept for official red warnings
  and measured major floods.

The scenario follows where the points lie: rain and flash forecasts point to flash flooding of
low-lying roads and areas within hours, and rivers and reservoirs to rivers overflowing. Both
together say both; an official bulletin on wet ground says only that low-lying areas may flood.

The thresholds are a first calibration against one event. Replayed over Colombo on 6 November with
the stored gauges and Google's areas as we held them (`internal/floodrisk/testdata`), the district
is high at 14:00, when Google's morning forecast and Kesbewa's 45 mm downpour agree. That is four
hours before the cloudburst and ten before the notice that came. The day's rain forecast is not
kept, so the replay leaves it out, which can only lower the estimate. The thresholds should be
reviewed after each flood season against the events of that season.

### Alerts, API and app

- The engine computes the estimate every run from the layers it already reads, as `flood_risk`
  events under a new `flood_risk` topic, one subject per district, held 6 hours like the
  forecasts. A run missing a source is incomplete and lowers nothing.
- A notice names the strongest signals (at most three) and the scenario: "Flood risk: High ·
  Colombo" / "Our estimate from Google's flash-flood forecast, heavy rain now and ground already
  soaked; not an official warning. Low-lying roads and areas may flood within hours. Follow the
  DMC." It opens the district.
- `GET /v1/flood-risk` serves every district's level, signals and inputs, worked out at most once a
  minute; the district panel shows them. A district at level 0 reads "no two sources point to
  flooding here now", never safe.
- Both new topics are on by default. Migrations 00014 and 00015 add them to subscriptions that
  chose flood forecasts.

## Consequences

- A measured-rain notice follows a burst by up to about an hour: readings reach the HMIS copy in
  about half an hour, the collector reads it every 30 minutes, and the engine within a minute of
  that. Reading the copy more often would gain little; reading HMIS directly would need its login.
- The estimate can call flooding likely when it does not come. It always says it is an estimate,
  is kept below danger, and needs two sources to agree.
- A faulty gauge can still raise a district: an impossible rise (over 200 mm an hour) is left out,
  and a notice names the gauge, but an implausible but possible reading counts.
- Sinhala and Tamil wording follows ADR 0018's terms and must be reviewed by native speakers.
- New families (soil moisture, radar nowcasts, canal levels in Colombo) are a field on `Inputs` and
  a row in the table.
