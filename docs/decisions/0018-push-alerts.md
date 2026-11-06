# ADR 0018: Push alerts

- Status: Accepted
- Date: 2026-11-03

## Context

The map tells people what is happening only while they look at it. Warnings matter most at night
and on the move, when nobody has the map open. The first release listed "alert subscriptions and
push notifications" as a non-goal; this record makes them a goal and sets their rules.

The constraints come from the earlier records:

- Official DMC, NBRO, Irrigation Department and Department of Meteorology notices are
  authoritative; anything derived or forecast must say it is an estimate or a forecast (ADR 0011,
  0014, 0015).
- A stale feed must never read as calm, and no place is ever called "safe" (ADR 0011, 0014).
- One service, one database, no per-visitor calls to third parties (ADR 0001).

## Decision

### Web Push, with no accounts

- **Web Push with VAPID** (RFC 8030, 8291, 8292). It reaches Android, desktop browsers and, since
  iOS 16.4, iPhones and iPads with the app on the Home Screen. No app store, no SMS gateway, no
  vendor account: the push services (FCM, Mozilla autopush, Apple, WNS) carry encrypted payloads
  they cannot read.
- **Our own implementation on the standard library** (`internal/push/webpush`): `crypto/ecdh`,
  `crypto/hkdf` and raw ECDSA keys make RFC 8291 and 8292 about 300 lines. `webpush-go` (MIT, last
  release January 2025) was considered: it uses deprecated `crypto/elliptic` calls and turns a
  `mailto:` subject into `mailto:mailto:`. Ours is tested against RFC 8291's Appendix A vector byte
  for byte.
- **Keys from the environment**: `LANKAWEATHER_VAPID_PUBLIC_KEY`, `_PRIVATE_KEY` and `_SUBJECT`
  (`lankaweather vapid-keys` prints a pair). Without them push is off: `GET /api/push/key` says
  `enabled: false`, the worker runs no engine, and deleting a subscription still works.
- **The endpoint is the identity** and the subscription's auth secret the proof of ownership for
  changes. The service worker's `pushsubscriptionchange` can prove only that it knew the old
  endpoint, a secret URL, which is accepted for renewal.

### What is stored, and for how long

| Kept | Not kept |
|---|---|
| The endpoint and its two keys | IP addresses (not in the database, not in logs) |
| Browser family and platform ("chrome-android") | The full User-Agent |
| Language (en, si, ta) | Any account, name, phone number or e-mail |
| Topics, minimum severity, districts | A precise location |
| A "near me" point rounded to two decimals (about 1.1 km) and its radius | Where the person is later: the point is read once, when they ask |
| Delivery health: last success, failure counts, back-off | Endpoints in logs: runs log counts only |

Events and the record of what was sent to whom are kept **30 days**, then pruned with the other
retention jobs. A subscription lives until its owner turns alerts off, its push service answers 404
or 410, or it fails 20 times in a row with no success for 7 days.

### What triggers an alert

The engine (`internal/alerts`) reads the services that serve the map, so a notice never disagrees
with the panel it opens. Each subject has a level from 0; a rise above the level last alerted
raises an event.

| Source | Subject | Levels | Severity |
|---|---|---|---|
| DMC warnings and advisories newly in force (not landslide reports) | One warning, its Tamil edition and numbered updates included | advisory 1; warning or amber advisory 2; red 3 | watch, warning, danger |
| NBRO landslide levels | A district: the highest level of its DS divisions in the report in force | 1, 2, 3 | watch (opt-in topic), warning, danger |
| River gauges, Irrigation Department and waterlevels.lk | A gauge, mapped to the district containing it or nearest within about 5 km | alert 1, minor flood 2, major flood 3 | watch, warning, danger |
| Heavy rain measured at the HMIS live gauges (topic `heavy_rain`) | A district, at its wettest current gauge | at least 30 mm in an hour, 50 in 3 hours or 100 in 24: 1; 50, 100 or 150: 2; 75, 150 or 250: 3 | watch, warning, warning |
| Google flood forecasts, verified gauges only | A gauge | Warning 1, Danger 2, Extreme 3 | warning, warning, danger |
| Google flash floods, read every 10 minutes | A district: the highest of the areas over it (Google reissues an area under a new ID every few hours, which had alerted again each time) | likely 1, highly likely 2 | warning |
| GDACS cyclones | A system | forecast track within 1,000 km 1; centre within 1,000 km 2; within 300 km 3 | watch, warning, danger |
| USGS earthquakes of the last 6 hours | An event | see below | watch, warning |
| Reservoirs spilling, current readings only | A reservoir with a known district | spilling 1 | watch (opt-in topic) |

Earthquakes: level 1 for M5 within 500 km of the coast, M6 anywhere in the region USGS is asked
about (15° S to 25° N, 60 to 105° E), or M4 within 300 km, which may be felt here; level 2 for M6.5
within 1,500 km, M7 anywhere in the region, or USGS's tsunami flag with M6. A level 2 notice says it
is **not a tsunami warning** and refers people to the DMC and the Department of Meteorology. We
never issue a tsunami warning.

Heavy rain is measured, not forecast, and reaches people before any river rises or forecast
arrives: on 6 November 2026 Colombo's head-office gauge measured at least 67.5 mm in the hour to
18:20, six hours before Google's flash-flood forecast reached us. The gauges report a rolling
24-hour total, so the rain of the last hour or three is its rise, a lower bound: a notice can
understate rain, never invent it. A gauge with an impossible rise (over 200 mm an hour) is left
out, and so is one whose newest reading is over 90 minutes old. The notice names the gauge and
says it is not an official warning; it stays at warning, as what the rain means for a place is
our reading of it. Migration 00014 adds the topic to subscriptions that chose flood forecasts.

A flash-flood notice says when Google issued the forecast ("Google issued it at 20:15"), so a
forecast that reached us late cannot pass for an early warning. A forecast that lapses sends
nothing: an "all clear" from a forecast says nothing about the rain to come.

Forecasts reach danger only at Google's Extreme: they are estimates, and danger passes the hourly
cap. Their text says "an estimate, not an official warning". Spills and Level 1 landslides are
opt-in topics: once chosen they are sent whatever the minimum severity, or choosing them would do
nothing at the default threshold.

Severity, as people choose a minimum: **info** for test notices; **watch** for Level 1, alert level
and the like; **warning** for Level 2, minor flood, Google's Warning and official warnings;
**danger** for Level 3, major flood and red notices. New subscriptions default to warning.

### Hysteresis and staleness

- **Hold times.** Once alerted at a level, a subject alerts at that level or below again only after
  staying below it for a hold time: 6 hours for gauges, forecasts and bulletins, 12 for landslide
  levels and cyclones, 24 for reservoirs. A gauge hovering at minor flood pings once. Rising above
  the alerted level always alerts: escalation is never held back.
- **No data is neither a rise nor a fall.** A stale gauge, a gauge without thresholds, a cyclone
  with an old advisory, or a landslide report that is lapsed, pending or unknown changes nothing.
  A whole feed that is stale cannot end anything: subjects it no longer mentions keep their state.
  Only an active report or a fresh "none" lowers landslide levels.
- **What was in force before subscribing is not sent.** An event raised before a subscription
  existed never reaches it; the app shows what is already in force. Escalations after subscribing
  are sent.

### Dedupe and delivery

- **Stable keys.** An event's key is its subject, level and the time that level was alerted
  (`river:nagalagam:2:1793685600`), so the same episode is stored once however often the engine
  runs. The event and the state that raised it are written in one transaction.
- **Nothing sent twice.** `push_deliveries` has a row per event per subscription. Rows are claimed
  before sending and released if the notice could not be delivered, so a crash loses a notice at
  worst and never repeats one.
- **Timeliness.** The engine is a collector with a 30-second interval, run beside the others, so a
  notice leaves within about a minute of the data landing. Only the worker holding the scheduler's
  advisory lock runs it.
- **Batching.** A subscriber's pending events become one notice per kind: "Landslide Level 2 · 3
  districts" rather than three. The most severe, then newest, event leads.
- **Cap.** Four notices an hour per subscription, except danger. Events held back go in the next
  notice while still relevant.
- **Headers.** `TTL` in hours, never past the events' relevance (danger 6, warning 4, watch 2, info
  1). `Urgency: high` for warning and danger, normal otherwise. `Topic` is a hash of the notice's
  tag, so a newer notice replaces an undelivered older one at the push service; the tag does the
  same on the device.
- **Failures.** 404 or 410 deletes the subscription. 429 waits for its `Retry-After` and backs off.
  Other server errors and network failures are retried twice in the run, then back off from a
  minute, doubling to an hour. A 400 or 413 is not retried (it would be refused again); a 401 or 403
  is released, as an operator can fix our keys. A pool of four sends in parallel.

### Text

Notices are written on the server in the subscriber's language (`internal/push/messages`), so the
service worker shows them as they arrive. Titles are short, bodies say what to do and who said so:
"Landslide Level 3 · Ratnapura" / "NBRO: evacuate if you are in an affected area. Tap for details."
District names come from the regions table's `name_si` and `name_ta`, without the trailing word
"district", which the source cuts at a fixed length.

Terms follow the issuers' trilingual notices as far as could be checked:

| English | Sinhala | Tamil |
|---|---|---|
| Landslide Level 1 / 2 / 3 (Watch, Alert, Evacuate) | නායයෑම් මට්ටම 1/2/3; අවධානයෙන් සිටින්න, සූදානම් වන්න, ඉවත් වන්න | மண்சரிவு நிலை 1/2/3; விழிப்புடன் இருங்கள், அவதானமாக இருங்கள், வெளியேறுங்கள் |
| Alert, minor flood, major flood level | අනතුරු ඇඟවීමේ, සුළු ගංවතුර, මහා ගංවතුර මට්ටම | எச்சரிக்கை, சிறிய வெள்ள, பாரிய வெள்ள மட்டம் |
| Warning, advisory, red warning | අනතුරු ඇඟවීම, උපදේශනය, රතු අනතුරු ඇඟවීම | எச்சரிக்கை, ஆலோசனை, சிவப்பு எச்சரிக்கை |
| Cyclone, earthquake, tsunami | සුළි කුණාටුව, භූ කම්පනය, සුනාමි | சூறாவளி, நில அதிர்வு, சுனாமி |
| Reservoir spilling | ජලාශය වාන් දමයි | நீர்த்தேக்கம் வான் பாய்கிறது |
| DMC, Department of Meteorology, Irrigation Department | ආපදා කළමනාකරණ මධ්‍යස්ථානය, කාලගුණ විද්‍යා දෙපාර්තමේන්තුව, වාරිමාර්ග දෙපාර්තමේන්තුව | அனர்த்த முகாமைத்துவ மத்திய நிலையம், வளிமண்டலவியல் திணைக்களம், நீர்ப்பாசனத் திணைக்களம் |

The sources searched on 2026-11-03 were mostly English press reports; the Sinhala and Tamil
wording above is the common usage of these institutions and must be reviewed by native speakers
against current DMC and NBRO notices before launch, as the rest of the interface is. Tests check
that every kind has text in all three languages, that forecasts say they are estimates, that
earthquakes say they are not tsunami warnings, and that nothing calls a place safe.

### The settings

A bell in the toolbar and the phone bar opens them. Permission is asked only when someone turns
alerts on. Each case has its own state: an unsupported browser; notifications blocked, with how to
allow them per browser; push off on the server; no service worker; and an iPhone or iPad tab, which
is told to add the app to the Home Screen first (iOS 16.4 or later) and offered the install guide.

## Consequences

- Alerts depend on the feeds. They are as late as the data: hours for bulletins and landslide
  levels (ADR 0011), 15 to 30 minutes for gauges. The settings say alerts can arrive late or not at
  all and give 117.
- Push needs HTTPS in production (browsers allow it on localhost only), and on iOS the installed
  app. Push services may delay or drop messages, especially on phones in power saving.
- Changing the VAPID keys invalidates every subscription; the app subscribes afresh when next
  opened.
- Matching runs in Go over all subscriptions each run. That is fine for tens of thousands; beyond
  that, matching should move into SQL by topic and district.
- A rate limit per client is in memory, per process; a proxy should limit too.
- `pushsubscriptionchange` with no known old endpoint returns 404, and the subscription is
  restored when the app is next opened, which re-sends its preferences.
