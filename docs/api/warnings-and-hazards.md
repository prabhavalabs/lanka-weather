# API contract: official warnings, landslides, earthquakes and cyclones

ADR 0011 records the decisions behind these endpoints. Every response comes
from data the worker has collected; no request reaches a source.

Times are RFC 3339 in UTC. Report dates are Sri Lanka local dates,
`YYYY-MM-DD`. Map layers are bare GeoJSON; other endpoints wrap their result in
`{"data": …}`. Errors are `{"error": {"code": "…", "message": "…"}}`. All cache
for 60 seconds. A service that is not configured answers 503 `unavailable`.

Before a first collection these answer 200 with empty lists and a `waiting`
status or a null `fetchedAt`, not an error.

## GET /v1/warnings

Bulletins and what is in force.

| Parameter | Meaning |
|---|---|
| `type` | `landslide`, `water_level`, `weather` or `situation`; else 400 `invalid_type` |
| `district` | A district code such as `LK-21`: bulletins whose clean text names it; else 400 `invalid_district` |
| `limit` | 1 to 200, default 60; else 400 `invalid_limit` |

Only `bulletins` is filtered. The list reaches back 45 days, newest first.

```json
{
  "data": {
    "bulletins": [
      {
        "id": "2026-11-02-10-30-warning-for-stron-22c40aa8",
        "type": "weather",
        "typeLabel": "Weather",
        "title": "Warning for strong winds and rough seas",
        "severity": "warning",
        "colour": "Red",
        "messageId": "07 MSW /26/11/02/01",
        "issuedAt": "2026-11-02T05:00:00Z",
        "validFrom": null,
        "validUntil": "2026-11-03T05:00:00Z",
        "current": true,
        "pdfUrl": "https://www.dmc.gov.lk/images/dmcreports/…pdf",
        "sourcePageUrl": "https://www.dmc.gov.lk/index.php?option=com_dmcreports…",
        "language": "en",
        "text": "The Severe Cyclone Murasu is very likely to move northwestwards…",
        "textStatus": "clean",
        "districts": [{ "id": "LK-42", "name": "Mannar" }],
        "source": "dmc-reports",
        "sourceTime": "2026-11-02T18:48:12Z",
        "fetchedAt": "2026-11-02T19:53:04Z"
      }
    ],
    "inForce": [],
    "activeCount": 1,
    "banner": [
      {
        "key": "bulletin:2026-11-02-10-30-warning-for-stron-22c40aa8",
        "kind": "bulletin",
        "title": "Warning for strong winds and rough seas",
        "detail": "Weather",
        "colour": "Red",
        "issuedAt": "2026-11-02T05:00:00Z",
        "pdfUrl": "https://www.dmc.gov.lk/images/dmcreports/…pdf"
      }
    ],
    "districts": [
      {
        "id": "LK-42",
        "name": "Mannar",
        "landslideLevel": 0,
        "mentioned": [{ "id": "2026-11-02-10-30-…", "title": "Warning for strong winds and rough seas" }]
      }
    ],
    "landslide": {},
    "meta": {
      "source": "dmc-reports",
      "attribution": "Disaster Management Centre, Sri Lanka, via nuuuwan/lk_dmc",
      "note": "Bulletins reach us through nuuuwan/lk_dmc, usually within hours of being issued. The PDF on dmc.gov.lk is the official version.",
      "fetchedAt": "2026-11-02T19:53:04Z",
      "upstreamAt": "2026-11-02T18:48:12Z",
      "generatedAt": "2026-11-02T20:03:14Z",
      "stale": false
    }
  }
}
```

- `title` and `colour` are the issuer's. `severity` is `warning`, `advisory`,
  `withdrawal` or `routine`, read from the title and colour. Landslide reports
  are `routine`: their seriousness is each division's level.
- `text` is null unless `textStatus` is `clean`: sentences copied unchanged,
  separated by blank lines, and possibly incomplete. Other statuses are
  `pending`, `unusable` and `unavailable`.
- `validUntil` is null if the bulletin states no period.
- `current` is true for a warning or advisory in force.
- `inForce` is unfiltered, one per warning. `activeCount` adds one for a
  landslide report showing. `landslide` is the `meta` below.
- `sourceTime` is when the upstream index was rebuilt.
- `stale` is true if our fetch or the upstream's rebuild is older than the
  layer's limit.

## GET /v1/warnings/landslides

The landslide-warnings layer, bare GeoJSON. Features are listed only while the
report's levels belong on the map.

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "LK-2136",
      "geometry": { "type": "MultiPolygon", "coordinates": [] },
      "properties": {
        "id": "LK-2136",
        "name": "Yatinuwara",
        "nameSi": "යටිනුවර",
        "districtId": "LK-21",
        "district": "Kandy",
        "labelLon": 80.55,
        "labelLat": 7.25,
        "level": 1,
        "label": "Level 1 (Yellow)",
        "colour": "Yellow",
        "action": "Watch"
      }
    }
  ],
  "meta": {
    "source": "nuuuwan-lk-dmc-landslides",
    "attribution": "National Building Research Organisation, Disaster Management Centre, via nuuuwan/lk_dmc_landslides",
    "status": "none",
    "note": "Last report 31 Oct 2026; none currently active as far as we know.",
    "stale": false,
    "reportDate": "2026-10-31",
    "issuedAt": "2026-10-31T10:30:00Z",
    "validUntil": "2026-11-01T10:30:00Z",
    "validityStated": true,
    "messageId": "LEWM-31-10-2026-1600-38",
    "pdfUrl": "https://www.dmc.gov.lk/images/dmcreports/…pdf",
    "sourcePageUrl": "https://www.dmc.gov.lk/index.php?option=com_dmcreports…",
    "counts": { "1": 3, "2": 0, "3": 0 },
    "unresolved": [],
    "fetchedAt": "2026-11-02T19:53:27Z",
    "upstreamAt": "2026-11-02T18:48:12Z",
    "generatedAt": "2026-11-02T20:03:14Z"
  }
}
```

`status` is `waiting`, `active`, `lapsed`, `pending`, `none` or `unknown` (ADR
0011). Show `note` as it is. `counts` describes the latest report whether it is
showing or not. `unresolved` lists its codes that match no region.

## GET /v1/warnings/landslides/{id}

One DS division in the envelope: `id`, `name`, `nameSi`, `districtId`,
`district`, `lon`, `lat`, `level` (`{level, label, colour, action}`, or null if
it is not named or no report is showing), `report` (the `meta` above) and
`history` (up to 30 reports that named it, newest first: `{date, level, label,
colour, action}`). 404 `division_not_found` for an unknown code.

## GET /v1/hazards/earthquakes

The last 30 days, newest first, bare GeoJSON.

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "us7000tty7",
      "geometry": { "type": "Point", "coordinates": [91.6593, 13.3785] },
      "properties": {
        "id": "us7000tty7",
        "magnitude": 5.1,
        "magType": "mb",
        "place": "218 km NNW of Bamboo Flat, India",
        "time": "2026-10-28T…Z",
        "depthKm": 10,
        "distanceKm": 1248,
        "url": "https://earthquake.usgs.gov/earthquakes/eventpage/us7000tty7",
        "alert": "",
        "tsunami": false,
        "significance": 400,
        "status": "reviewed",
        "updatedAt": "…",
        "fetchedAt": "…",
        "source": "usgs-earthquakes"
      }
    }
  ],
  "meta": {
    "source": "usgs-earthquakes",
    "attribution": "U.S. Geological Survey",
    "windowDays": 30,
    "fetchedAt": "…",
    "generatedAt": "…",
    "stale": false,
    "count": 18
  }
}
```

`distanceKm` is to Sri Lanka's nearest coast. `tsunami` is USGS's flag: an
indicator, not a warning for Sri Lanka.

## GET /v1/hazards/cyclones

Current systems in the North Indian Ocean, bare GeoJSON. Each system is a
`position` feature followed by features of the same `eventId` with `kind` one of
`wind-buffer` (`windKmh`, `band`, `label`), `uncertainty-cone`, `track`,
`forecast-track` (`stage`: TD, TS or HU) and `track-point` (`label`, `time`,
`forecast`).

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "1001346",
      "geometry": { "type": "Point", "coordinates": [80.5, 15.1] },
      "properties": {
        "kind": "position",
        "eventId": "1001346",
        "name": "MURASU-26",
        "alertLevel": "Red",
        "windKmh": 157,
        "severityText": "Hurricane/Typhoon > 74 mph (maximum wind speed of 157 km/h)",
        "country": "India",
        "startedAt": "2026-11-01T00:00:00Z",
        "latestAt": "2026-11-02T18:00:00Z",
        "advisorySource": "JTWC",
        "reportUrl": "https://www.gdacs.org/report.aspx?eventid=1001346&episodeid=8&eventtype=TC",
        "lon": 80.5,
        "lat": 15.1,
        "distanceKm": 587,
        "stale": false,
        "source": "gdacs",
        "sourceTime": "…",
        "fetchedAt": "…"
      }
    }
  ],
  "meta": {
    "source": "gdacs",
    "attribution": "Global Disaster Alert and Coordination System (GDACS), gdacs.org",
    "status": "active",
    "note": "",
    "fetchedAt": "…",
    "generatedAt": "…",
    "stale": false,
    "count": 1
  }
}
```

`status` is `waiting`, `active`, `none` or `unknown`. `none` carries the note "No
active cyclone in the North Indian Ocean" and is only given for a fresh fetch;
`unknown` says it cannot tell. A system's `stale` means no advisory for 18 hours.

## The `official` part of GET /v1/summary

```json
{
  "status": "available",
  "landslide": {
    "status": "none",
    "note": "Last report 31 Oct 2026; none currently active as far as we know.",
    "stale": false,
    "level": null,
    "dsId": "LK-2130",
    "ds": "Gangawata Korale",
    "district": "Kandy",
    "reportDate": "2026-10-31",
    "issuedAt": "2026-10-31T10:30:00Z",
    "validUntil": "2026-11-01T10:30:00Z",
    "pdfUrl": "https://www.dmc.gov.lk/images/dmcreports/…pdf"
  },
  "warnings": [],
  "general": [],
  "note": "Bulletins reach us through nuuuwan/lk_dmc…",
  "stale": false,
  "fetchedAt": "2026-11-02T19:53:04Z"
}
```

- `landslide.level` is the level of the DS division containing the place, as in
  `/v1/warnings/landslides`; null means not named or none showing, **not** safe.
- `warnings` are bulletins from the last 48 hours naming the place's district by
  exact word in clean text; `general` are other warnings in force and the latest
  bulletin of each other type. Both are bulletins as in `/v1/warnings`.
- `status` is `not_available`, with `stale` true, if nothing could be read, and
  `unavailable` then lists `official`.
