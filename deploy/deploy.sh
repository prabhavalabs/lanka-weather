#!/usr/bin/env bash
# Ships the committed tree at HEAD to the server, builds the images there and
# restarts the stack. Uncommitted changes are not deployed.
#
#   DEPLOY_HOST=deploy@your-server deploy/deploy.sh
#
# The server keeps its settings in $DEPLOY_DIR/.env.production and the deployed
# revision in $DEPLOY_DIR/deploy/.env; both survive a deploy. Runbook: deploy/README.md.
set -euo pipefail

host=${DEPLOY_HOST:?set DEPLOY_HOST, e.g. deploy@your-server}
dir=${DEPLOY_DIR:-/opt/lanka-weather}
rev=$(git rev-parse --short=12 HEAD)

if [[ -n $(git status --porcelain --untracked-files=no) ]]; then
  echo "note: uncommitted changes are not deployed; deploying $rev" >&2
fi

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
git archive HEAD | tar -x -C "$stage"

ssh "$host" "install -d -m 0755 '$dir'"
rsync -az --delete --exclude /.env.production --exclude /deploy/.env "$stage"/ "$host:$dir/"

ssh "$host" bash -s -- "$dir" "$rev" <<'REMOTE'
set -euo pipefail
dir=$1 rev=$2
cd "$dir/deploy"
test -f ../.env.production || { echo "missing $dir/.env.production (see deploy/README.md)" >&2; exit 1; }
printf 'LANKAWEATHER_VERSION=%s\n' "$rev" > .env
docker compose build
docker compose up -d --remove-orphans
docker compose ps
REMOTE
