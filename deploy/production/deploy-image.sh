#!/usr/bin/env bash
set -Eeuo pipefail

image=${1:-}
if [[ ! ${image} =~ ^ghcr\.io/crownsis/libredesk@sha256:[0-9a-f]{64}$ ]]; then
  echo "Usage: $0 ghcr.io/crownsis/libredesk@sha256:<64-hex-digest>" >&2
  exit 2
fi

deployment_dir=${LIBREDESK_DEPLOY_DIR:-"${HOME}/libredesk"}
cd "${deployment_dir}"

docker pull "${image}"
./backup.sh

previous_env=$(mktemp "${deployment_dir}/.env.rollback.XXXXXX")
chmod 0600 "${previous_env}"
cp .env "${previous_env}"

temporary_env=$(mktemp "${deployment_dir}/.env.XXXXXX")
chmod 0600 "${temporary_env}"
awk -F= -v image="${image}" '
  BEGIN { OFS = "="; found = 0 }
  $1 == "LIBREDESK_IMAGE" { $2 = image; found = 1 }
  { print }
  END { if (!found) print "LIBREDESK_IMAGE", image }
' .env > "${temporary_env}"
mv "${temporary_env}" .env

rollback() {
  echo "Deployment failed; restoring the previous image." >&2
  mv "${previous_env}" .env
  docker compose up -d --no-deps app
}
trap rollback ERR

docker compose config --quiet
docker compose up -d --no-deps app

for attempt in $(seq 1 30); do
  if [[ $(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' libredesk_app) == healthy ]] && \
    curl --fail --silent --show-error --output /dev/null http://127.0.0.1:9000/; then
    trap - ERR
    rm -f "${previous_env}"
    echo "Deployed ${image}"
    exit 0
  fi
  sleep 5
done

false
