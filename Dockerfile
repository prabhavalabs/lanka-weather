# syntax=docker/dockerfile:1
#
# Production images, built by deploy/compose.yaml:
#   api  the Go binary, for `migrate`, `serve` and `worker`
#   web  the built web app behind nginx, which also forwards /v1, /api and
#        /healthz to the api service
# Runbook: deploy/README.md.

FROM golang:1.26-alpine AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/lankaweather ./cmd/lankaweather

FROM node:24-alpine AS web-build
WORKDIR /src
RUN corepack enable
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc* ./
COPY apps/web/package.json apps/web/
RUN pnpm install --frozen-lockfile --filter @lanka-weather/web...
COPY apps/web apps/web
RUN pnpm --filter @lanka-weather/web build

FROM alpine:3.22 AS api
RUN apk add --no-cache ca-certificates tzdata poppler-utils \
  && adduser -D -H -u 10001 lankaweather
COPY --from=go-build /out/lankaweather /usr/local/bin/lankaweather
USER lankaweather
ENV LANKAWEATHER_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["lankaweather"]
CMD ["serve"]

FROM nginxinc/nginx-unprivileged:1.31-alpine AS web
COPY deploy/nginx/web.conf /etc/nginx/conf.d/default.conf
COPY --from=web-build /src/apps/web/dist /usr/share/nginx/html
EXPOSE 8080
