#!/usr/bin/env bash
set -Eeuo pipefail

deployment_dir=${LIBREDESK_DEPLOY_DIR:-"${HOME}/libredesk"}
cd "${deployment_dir}"
umask 077

timestamp=$(date -u +%Y%m%dT%H%M%SZ)
destination="backups/${timestamp}"
mkdir -p "${destination}"

set -a
source .env
set +a

docker compose exec -T db pg_dump \
  --username "${POSTGRES_USER}" \
  --dbname "${POSTGRES_DB}" \
  --format custom \
  --no-owner \
  --no-acl > "${destination}/postgres.dump"

tar -C data -czf "${destination}/uploads.tar.gz" uploads
install -m 0600 .env "${destination}/environment.env"
sha256sum "${destination}/postgres.dump" "${destination}/uploads.tar.gz" \
  > "${destination}/SHA256SUMS"

find backups -mindepth 1 -maxdepth 1 -type d -mtime +30 -exec rm -rf -- {} +
echo "Backup completed: ${deployment_dir}/${destination}"
