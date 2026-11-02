# API contract: the situation overview

`GET /v1/overview` is "Sri Lanka now": what the other services say, composed by fixed
rules for a first look. ADR 0014 records the decisions. No request reaches a source, and
no language model is involved.

Times are RFC 3339 in UTC. The result is wrapped in `{"data": …}` and cached for 60 seconds.
A part that cannot be read does not fail the request: it is listed in `unavailable` and its
section says so. The request answers 503 `unavailable` only if the overview is not configured.

```json
{
  "data": {
    "generatedAt": "2026-11-02T21:30:00Z",
    "status": "1 official warning or advisory in force. 1 cyclone active north of Sri Lanka. 1 river gauge at or above alert level. 10 reservoirs spilling.",
    "stale": [],
    "unavailable": [],
    "sections": [
      {
        "id": "warnings",
        "group": "official",
        "title": "Warnings in force",
        "summary": "1 warning or advisory in force, in the issuers' words.",
        "note": "Bulletins reach us through nuuuwan/lk_dmc, usually within hours of being issued. The PDF on dmc.gov.lk is the official version.",
        "official": true,
        "estimate": false,
        "available": true,
        "stale": false,
        "at": "2026-11-02T21:07:10Z",
        "source": "Disaster Management Centre",
        "counts": [{ "key": "inForce", "label": "In force", "value": 1 }],
        "items": [
          {
            "id": "2026-11-02-10-30-warning-for-stron-22c40aa8",
            "title": "Warning for strong winds and rough seas",
            "detail": "Weather",
            "tone": "severe",
            "badge": "Warning: Red",
            "estimate": false,
            "stale": false,
            "at": "2026-11-02T05:00:00Z",
            "source": "Disaster Management Centre",
            "target": { "kind": "warnings", "id": "weather" },
            "href": "https://www.dmc.gov.lk/…pdf"
          }
        ],
        "target": { "kind": "warnings" }
      }
    ]
  }
}
```

## The status

`status` is one sentence, most serious first: landslide levels and warnings in force; cyclones;
gauges at or above alert level; estimated high water, labelled as an estimate; a large earthquake
in the last day; reservoirs spilling; then the one calm thing it may say, "No river gauge above its
alert level", and only while the gauges are current. It ends by naming what is out of date
(`stale`) or unread (`unavailable`). A stale feed is left out of what it counts. It never says
"safe" or "all clear".

## Sections

In this order, `group` in brackets:

| `id` | Group | What it holds |
|---|---|---|
| `landslides` | `official` | Levels by count, the worst divisions. `note` is what the feed cannot tell |
| `warnings` | `official` | Warnings and advisories in force, in the issuers' words, with `href` |
| `gauges` | `water` | Gauges at alert or above, measured against the department's own levels |
| `rivers` | `water` | Stretches at each risk level and the worst named rivers. `estimate: true` |
| `reservoirs` | `water` | Spilling, then releasing, largest flow first, current readings only |
| `cyclones` | `hazards` | Systems in the North Indian Ocean, with distance and direction |
| `earthquakes` | `hazards` | The last 7 days' significant events: see ADR 0014 |
| `weather` | `now` | Hottest, coolest, three wettest and windiest districts. `estimate: true` |
| `rain` | `now` | The five wettest live rain gauges over 24 hours, measured, once at least 20 report; otherwise the wettest river gauges |
| `air` | `now` | The worst index on the grid. `estimate: true` |
| `sea` | `now` | The highest wave on the grid. `estimate: true` |

- `official` marks the issuers' words and levels. They are unchanged here.
- `estimate` marks what is ours or modelled, on the section and on each item.
- `available: false` means the part could not be read, and `stale: true` that what was read is too
  old to speak for today. Neither reads as calm: the summary says so.
- `at` is the age to show. Each item carries its own `at` and `source`.
- `tone` is `neutral`, `watch`, `high` or `severe`, and always comes with words.
- `counts` and `items` are empty lists, never null.

## Targets

`target` says what a click opens. `kind` is one of `warnings` (with a type as `id`), `landslide`,
`river` (with `reach`), `station`, `raingauge`, `reservoir`, `reservoirs`, `cyclone`, `quake`, `region` (with
`fill`: `temperature`, `rain` or `wind`), `place`, or `layer`. `layer` names a layer to switch on,
and `lon` and `lat` a place to show.
