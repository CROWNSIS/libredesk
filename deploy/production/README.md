# LibreDesk Production Deployment

This deployment runs LibreDesk v2.8.0 with PostgreSQL 17 and Redis 7. All
routine deployment work is performed by `onionlake-admin`. LibreDesk is
reachable only through the bundled reverse proxy.

## Security model

- Application image is pinned to the v2.8.0 multi-platform manifest digest.
- PostgreSQL and Redis have no host-published ports.
- Database, Redis, encryption, and initial system-user credentials are
  generated on the server and stored in `~/libredesk/.env` with mode 0600.
- Redis authentication and persistent append-only storage are enabled.
- Containers use resource limits, bounded logs, and `no-new-privileges`.
- LibreDesk has no host-published port. A pinned Nginx container publishes
  port 9000 only on loopback and the VM's Tailscale address.
- Nginx forwards Cloudflare's authenticated client address as `X-Client-IP`,
  as required for LibreDesk rate limiting and audit logs.
- `onionlake-admin` is the only non-root account in the `docker` group. Docker
  group membership is equivalent to root-level host access and must not be
  granted to application or untrusted accounts.

## Server installation

Copy this directory to the server. Run the OS prerequisite step with sudo,
then log out and back in so the new group membership applies and run the
application setup as `onionlake-admin`:

```bash
sudo ./bootstrap.sh
./setup.sh
```

The initial `System` password is in `~/libredesk/.env`. Change it after the
first login. It is passed only to the one-time idempotent installer and is not
present in the long-running application container environment.

## Backups

The `libredesk-backup.timer` user timer runs `~/libredesk/backup.sh` nightly.
It writes a PostgreSQL custom-format
dump, uploads archive, environment/encryption configuration, and checksums to
`~/libredesk/backups`. Local backups are retained for 30 days. Proxmox
snapshots provide a second recovery layer, but application-consistent database
dumps remain required before every LibreDesk upgrade.

## Deploying an OLEDU image

The hosted GitHub Actions pipeline publishes commit-tagged images to GHCR. Get
the digest for the tested image, then deploy it as `onionlake-admin`:

```bash
~/libredesk/deploy-image.sh ghcr.io/crownsis/libredesk@sha256:<digest>
```

The script accepts only immutable CROWNSIS image digests. It backs up the
database, uploads, and encryption configuration before changing the image,
then rolls back automatically if the application does not become healthy.

## Public ingress

Cloudflare Tunnel routes `support.olcepks.ca` to
`http://100.105.41.61:9000`. The origin is reachable only over Tailscale and
passes through the bundled Nginx proxy. No router port forwarding or public
origin IP is required.
