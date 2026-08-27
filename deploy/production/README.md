# LibreDesk Production Deployment

This deployment runs LibreDesk v2.8.0 with PostgreSQL 17 and Redis 7. All
routine deployment work is performed by `onionlake-admin`. The application is
bound to localhost so it must be accessed through a trusted HTTPS reverse
proxy or Tailscale Serve.

## Security model

- Application image is pinned to the v2.8.0 multi-platform manifest digest.
- PostgreSQL and Redis have no host-published ports.
- Database, Redis, encryption, and initial system-user credentials are
  generated on the server and stored in `~/libredesk/.env` with mode 0600.
- Redis authentication and persistent append-only storage are enabled.
- Containers use resource limits, bounded logs, and `no-new-privileges`.
- The application port is available only at `127.0.0.1:9000`.
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

## Public ingress

The official LibreDesk documentation recommends Nginx and requires the proxy
to set `X-Client-IP`. Start from `nginx.conf.example`, add the production
hostname and TLS certificate, and expose ports 80/443 only after DNS is ready.
