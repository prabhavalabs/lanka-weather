# Deploying weather.prabhavalabs.com

This is how the maintainers deploy their instance. To run your own, adapt
the host name, paths and database set-up to your server.

The site runs on an Ubuntu host as one Docker Compose project,
`lanka-weather`, behind the host's nginx. The database is a PostgreSQL
container shared by other apps on that host, reached on its `platform`
Docker network.

```
browser ─TLS─▶ host nginx :443 ─▶ 127.0.0.1:18750 web (nginx, built app)
                                         ├─ /v1, /api, /healthz ─▶ api (lankaweather serve)
                                         └─ everything else: the app's files
worker (lankaweather worker) ─▶ collectors, alerts and push
api, worker, migrate ─▶ platform-postgres:5432, database lankaweather (platform network)
```

- `migrate` applies the migrations on every start; `api` and `worker`
  wait for it.
- Only one worker collects at a time: it holds a PostgreSQL advisory
  lock.
- Every stored row comes from the collectors, so the containers hold no
  state. A new database fills itself within a few hours of the first
  start.

## Files on the server

| Path | What |
|---|---|
| `/opt/lanka-weather/` | The committed tree, replaced on each deploy |
| `/opt/lanka-weather/.env.production` | Settings and secrets (mode 0600; never in Git) |
| `/opt/lanka-weather/deploy/.env` | The deployed revision, written by `deploy.sh` |
| `/etc/nginx/sites-available/lanka-weather` | From `deploy/nginx/weather.prabhavalabs.com.conf`, plus Certbot's TLS block |
| `/opt/platform/env/apps/lankaweather.pg.env` | The database login, from `/opt/platform/bin/pg-new-app.sh` |

`.env.production` holds the settings in `.env.example`:

- `LANKAWEATHER_DATABASE_URL`, which points at
  `platform-postgres:5432/lankaweather?sslmode=disable`;
- `LANKAWEATHER_CONTACT`;
- `LANKAWEATHER_GOOGLE_API_KEY`;
- the three `LANKAWEATHER_VAPID_*` settings.

Leave `LANKAWEATHER_ADDR` out: the image sets `:8080`.

## Deploy

From a clean checkout of the revision to ship:

```bash
DEPLOY_HOST=deploy@your-server deploy/deploy.sh
```

The script archives `HEAD` and copies it to the server. It then builds
both images there and restarts the stack.

Check the result on the server:

```bash
cd /opt/lanka-weather/deploy && docker compose ps && docker compose logs --tail 50 api worker
curl -fsS https://weather.prabhavalabs.com/healthz
```

To roll back, check out the earlier revision and deploy again.

## First install (done once)

1. **DNS:** in Cloudflare, add an `A` record for `weather` pointing at
   the host's address. Set it to DNS only, so Certbot's HTTP challenge
   reaches nginx.
2. **Database:** run `/opt/platform/bin/pg-new-app.sh lankaweather`. Then,
   as the superuser, run `CREATE EXTENSION postgis; CREATE EXTENSION
   pg_trgm;` in `lankaweather`. The app's role is not a superuser, so it
   cannot create them.
3. **Settings:** write `/opt/lanka-weather/.env.production`. Make a
   VAPID pair with
   `docker run --rm lanka-weather-api:<rev> vapid-keys`. Changing the
   pair later drops every push subscription.
4. **Deploy:** run `deploy.sh`.
5. **nginx:** copy the site file to
   `/etc/nginx/sites-available/lanka-weather` and link it from
   `sites-enabled`. Run `nginx -t && systemctl reload nginx`, then
   `certbot --nginx -d weather.prabhavalabs.com`.

If the Cloudflare proxy is turned on later, set SSL/TLS to Full
(strict). The API's rate limits would then see Cloudflare's addresses,
unless nginx is taught `set_real_ip_from` for Cloudflare's ranges.
