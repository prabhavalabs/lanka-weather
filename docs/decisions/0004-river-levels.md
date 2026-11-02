# ADR 0004: River levels from the Irrigation Department gauge service

- Status: Accepted
- Date: 2026-11-02

## Context

River levels are the earliest local sign of flooding. The Irrigation
Department's Hydrology and Disaster Management Division publishes
readings from about 40 gauging stations through a public ArcGIS feature
service behind its gauge dashboard. Each feature is one reading with the
station's name, basin, position, water level, rainfall, and its alert,
minor-flood and major-flood thresholds. Stations report about hourly.

The same readings are scraped into `nuuuwan/lk_irrigation`, and DMC
bulletins republish them every three hours as PDFs. Both are later and
less complete than the feature service.

## Decision

- **Source:** query the feature service directly every 15 minutes,
  selecting readings created since three hours before the newest stored
  reading (a week on the first run), oldest first, 1,000 per page.
- **Identity:** a station is identified by a slug of its gauge name; a
  reading by station and observation time, so overlapping fetches are
  idempotent.
- **Units:** levels and thresholds are compared in the unit the service
  publishes them in, so classification never converts units. Stations
  are labelled metres except Nagalagam Street, which the department
  reads in feet.
- **Thresholds:** a station's thresholds are used only when all three are
  present, positive and increasing; otherwise its status is "unknown".
- **Status:** normal, alert, minor flood or major flood, from the latest
  reading against the thresholds.
- **Trend:** the change per hour between the latest reading and the
  oldest reading within the previous three hours (at least 30 minutes
  apart); rising or falling at 0.05 units per hour or more.
- **Staleness:** a station whose latest reading is older than the layer's
  stale limit (three hours) is drawn faded and labelled out of date.
- **Retention:** readings are kept for 400 days; fetch records for 30.
- **Scheduling:** `lankaweather worker` runs collectors in-process on
  their intervals. One worker leads through a PostgreSQL advisory lock;
  every fetch is recorded in `source_runs`. A job queue is not needed
  until collectors must run in parallel or be retried individually.

## Consequences

- The map shows readings within about 15 minutes of publication.
- Our status repeats the department's own thresholds; it is not a
  forecast and is labelled with the department as the source.
- If the department changes the feature service, the collector fails
  visibly in `source_runs` and the layer turns stale rather than showing
  old data as current.
- Unit labels for stations other than Nagalagam Street should be
  confirmed with the department's published bulletins.
