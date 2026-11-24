# Security policy

## Supported versions

Lanka Weather has no versioned releases. Security fixes go to the `main`
branch and to the maintainers' deployment at
<https://weather.prabhavalabs.com>.

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Report it privately through GitHub:
[open a security advisory](https://github.com/prabhavalabs/lanka-weather/security/advisories/new).
Include what is affected, how to reproduce it, and the impact you expect.

You can expect:

- an acknowledgement within 7 days;
- an assessment and, for a confirmed problem, a plan for the fix;
- credit in the advisory when it is published, if you want it.

This is a volunteer project, so there is no bug bounty.

## Scope

In scope, for example:

- exposure of server-side secrets, such as the Google API key or the
  VAPID private key used for push notifications;
- abuse of the push endpoints under `/api/push/`: subscribing,
  changing or sending notices to someone else's subscription, or
  getting around their rate limits;
- injection through data the collectors ingest, such as DMC bulletin
  PDFs and pages, station or place names, or source attribution text,
  that reaches the API, the database or the web app (XSS, SQL
  injection);
- server-side request forgery or path traversal in the collectors;
- poisoning the service worker's cache so that the app shows false
  warnings.

Out of scope:

- wrong, late or missing data. Please open a normal issue; it is
  important, but it is not a security problem;
- vulnerabilities in the upstream sources (for example dmc.gov.lk,
  meteo.gov.lk or waterlevels.lk). Report those to their operators;
- volumetric denial of service.

## Testing

Test against your own local instance (see the README), not against
weather.prabhavalabs.com. Never send push notices to real subscribers.
