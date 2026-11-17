# API contract: push alerts

Web Push subscriptions and their preferences, under `/api/push`. ADR 0018 records the decisions.
There are no accounts: a subscription's endpoint is its identity, and its `auth` key proves
ownership for changes.

Every response is wrapped in `{"data": …}` or `{"error": {"code", "message"}}` and sent with
`Cache-Control: no-store`. Bodies are JSON, at most 8 kB. A client may make 30 requests a minute
to these routes and send 5 test notices in 10 minutes; beyond that the answer is 429
`rate_limited` with `Retry-After`.

## `GET /api/push/key`

Whether alerts can be turned on, and the key to subscribe with (`applicationServerKey`).

```json
{ "data": { "enabled": true, "publicKey": "BMjlRtJjlpRYU4k6nOUt9Po3BIYuC-JhMe9ho04mBkEs…" } }
```

Without VAPID keys on the server: `{ "data": { "enabled": false } }`.

## Preferences

```json
{
  "topics": ["official", "landslide", "river", "flood_forecast", "cyclone", "earthquake"],
  "minSeverity": "warning",
  "areas": {
    "country": false,
    "districts": ["LK-91", "LK-92"],
    "near": { "lat": 6.68, "lon": 80.4, "radiusKm": 10 }
  }
}
```

- `topics`: any of `official`, `landslide` (levels 2 and 3), `landslide_level1`, `river`,
  `heavy_rain`, `flood_risk`, `flood_forecast`, `cyclone`, `earthquake`, `reservoir`; at least one. `landslide_level1` and
  `reservoir` are opt-in and are sent whatever `minSeverity` says.
- `minSeverity`: `info`, `watch`, `warning` (default) or `danger`.
- `areas`: `country`, or districts (codes from `/v1/regions?level=district`, at most 25) and/or
  `near`, a point in Sri Lanka with a radius of 1 to 100 km. The server rounds the point to two
  decimals (about 1.1 km) and works out the districts within the radius. National events
  (cyclones, earthquakes, bulletins naming no district) go to every area.

## `POST /api/push/subscriptions`

Creates a subscription, or updates one with the same endpoint and auth key.

```json
{
  "subscription": {
    "endpoint": "https://fcm.googleapis.com/fcm/send/…",
    "expirationTime": null,
    "keys": { "p256dh": "BK…", "auth": "Q2…" }
  },
  "preferences": { "…": "as above" },
  "locale": "si"
}
```

`subscription` is `PushSubscription.toJSON()`. The endpoint must be https on a known push service:
`fcm.googleapis.com`, `*.push.services.mozilla.com`, `web.push.apple.com` or `*.notify.windows.com`.
`locale` is `en`, `si` or `ta`: notices are written in it. Answers 201 when created, 200 when
updated:

```json
{
  "data": {
    "locale": "si",
    "preferences": { "…": "as stored" },
    "createdAt": "2026-11-03T08:38:41Z",
    "updatedAt": "2026-11-03T08:38:41Z",
    "lastSuccessAt": null
  }
}
```

The endpoint and keys are never returned.

## `PUT /api/push/subscriptions`

Changes preferences and language.

```json
{ "endpoint": "https://fcm.googleapis.com/fcm/send/…", "auth": "Q2…", "preferences": { "…": "" }, "locale": "en" }
```

## `DELETE /api/push/subscriptions`

```json
{ "endpoint": "https://fcm.googleapis.com/fcm/send/…", "auth": "Q2…" }
```

Answers `{ "data": { "deleted": true } }`. It works even when push is switched off on the server.

## `POST /api/push/subscriptions/renew`

Called by the service worker on `pushsubscriptionchange`.

```json
{ "oldEndpoint": "https://fcm.googleapis.com/fcm/send/old…", "subscription": { "endpoint": "…", "keys": { "p256dh": "…", "auth": "…" } } }
```

The subscription stored under `oldEndpoint` moves to the new endpoint and keys, keeping its
preferences. With `oldEndpoint` null or unknown, a subscription already stored under the new
endpoint has its keys refreshed; otherwise 404 `subscription_not_found`, and the app subscribes
again with its saved preferences when next opened.

## `POST /api/push/test`

Sends a test notice (severity `info`) to the subscription, at most once a minute.

```json
{ "endpoint": "https://fcm.googleapis.com/fcm/send/…", "auth": "Q2…" }
```

Answers 202 `{ "data": { "sent": true } }` once the push service accepts it.

## `POST /api/push/inbox`

The notices sent to the subscription in the last 30 days, newest first (at most 100), written in
`locale`, for the app's list of notices (ADR 0020). POST because the body carries the auth key.

```json
{ "endpoint": "https://fcm.googleapis.com/fcm/send/…", "auth": "Q2…", "locale": "en" }
```

```json
{
  "data": {
    "notices": [
      {
        "id": "43",
        "sentAt": "2026-11-12T15:23:06Z",
        "kind": "bulletin",
        "severity": "warning",
        "title": "Flood warning · Matara",
        "body": "DMC, issued 18:30: “Flood warning for Nilwala River Basin”. Districts: Matara. Follow official instructions.",
        "url": "/?warnings=water_level&district=LK-32",
        "official": true,
        "source": "Disaster Management Centre",
        "national": false,
        "areas": [{ "id": "LK-32", "name": "Matara" }],
        "events": [
          {
            "key": "bulletin:b5332fc4308c:2:1794496986",
            "kind": "bulletin",
            "severity": "warning",
            "title": "Flood warning · Matara",
            "body": "…",
            "url": "/?warnings=water_level&district=LK-32",
            "detectedAt": "2026-11-12T15:23:06Z",
            "issuedAt": "2026-11-12T13:00:00Z",
            "areas": [{ "id": "LK-32", "name": "Matara" }]
          }
        ]
      }
    ]
  }
}
```

`official` is true for the DMC's bulletins, NBRO's landslide levels and the Irrigation Department's
gauges; forecasts, estimates and community data are false. `issuedAt` is when the issuer issued or
measured it, when known.

## Errors

| Status | Code | When |
|---|---|---|
| 400 | `invalid_json`, `invalid_endpoint`, `unsupported_push_service`, `invalid_keys`, `invalid_locale`, `invalid_topics`, `invalid_severity`, `invalid_districts`, `invalid_location`, `invalid_radius`, `invalid_areas`, `invalid_subscription` | The body cannot be accepted |
| 403 | `not_owner` | The auth key does not match the subscription |
| 404 | `subscription_not_found` | No subscription has this endpoint |
| 410 | `subscription_gone` | The push service no longer accepts it (it has been deleted) |
| 413 | `body_too_large` | Over 8 kB |
| 429 | `rate_limited` | Too many requests, or a test within a minute of the last |
| 502 | `delivery_failed` | The push service refused the test notice |
| 503 | `push_disabled` | No VAPID keys on the server |

## The notice

What the service worker receives, decrypted (the contract with `apps/web/src/sw.ts`):

```json
{
  "title": "Landslide Level 3 · Ratnapura",
  "body": "NBRO, issued 16:00: Level 3 (Red) in Ratnapura district, 4 divisions including Ayagama, Elapatha. Evacuate now if you are in these areas.",
  "url": "/?warnings=landslide&district=LK-91&notice=42",
  "tag": "landslide:district:LK-91",
  "severity": "danger",
  "sentAt": "2026-11-03T08:39:02Z",
  "requireInteraction": true,
  "lang": "en",
  "id": "42",
  "kind": "landslide"
}
```

- `url` is a path the app opens a panel for: `/?warnings=<type>[&district=]`,
  `/?warnings=landslide&district=`, `/?station=`, `/?gauge=`, `/?area=`, `/?cyclone=`, `/?quake=`,
  `/?reservoir=`, `/?reservoirs=active`, or `/`, followed by `notice=<id>`, which opens the
  notice's details over that map.
- `id` is the notice's id, as `POST /api/push/inbox` lists it; absent for a test notice. `kind` is
  the kind of notice.
- `tag` groups notices so a newer one replaces an older one: `bulletin:<hash>`,
  `landslide:district:<code>`, `river:station:<id>`, `forecast:gauge:<id>`, `flash:area:<id>`,
  `cyclone:<id>`, `quake:<id>`, `reservoir:<id>`, `<kind>:summary:<id>` for a gathered notice (its
  own, so it never replaces another gathered notice), `test`.
- `requireInteraction` is present, and true, only for danger.
- Request headers: `TTL` in seconds (hours, never days), `Urgency: high` for warning and danger and
  `normal` otherwise, `Topic` a 32-character hash of the tag.
