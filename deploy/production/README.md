# LibreDesk Production Deployment

This deployment runs LibreDesk v2.8.0 with PostgreSQL 17, Redis 7, and a
CPU-only Ollama service. All routine deployment work is performed by
`onionlake-admin`. LibreDesk is reachable only through the bundled reverse
proxy.

## Security model

- Application image is pinned to the v2.8.0 multi-platform manifest digest.
- PostgreSQL and Redis have no host-published ports.
- Ollama has no published port and its long-running service is attached only
  to the internal backend network. Only the one-shot model initializer receives
  temporary egress so it can pull configured models.
- Database, Redis, encryption, and initial system-user credentials are
  generated on the server and stored in `~/libredesk/.env` with mode 0600.
- Redis authentication and persistent append-only storage are enabled.
- Containers use resource limits, bounded logs, and `no-new-privileges`.
- Ollama runs as non-root UID 1000 with all capabilities dropped, a read-only
  root filesystem, and models in the persistent `libredesk_ollama-models`
  volume.
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

## Local AI

The official CPU image is pinned to stable Ollama `0.32.15` and the immutable
Linux/amd64 manifest digest returned by Docker Hub on 2026-08-27. This digest
is intentionally specific to VM 106's architecture. Verify and update both the
version and digest deliberately when upgrading Ollama.

The idempotent `ollama-init` service pulls `qwen3:1.7b` for completions
and `qwen3-embedding:0.6b` for 1024-dimensional multilingual embeddings. Override `OLLAMA_COMPLETION_MODEL` and
`OLLAMA_EMBEDDING_MODEL` in `.env` before running setup to use different model
names. LibreDesk provider settings must use `http://ollama:11434/v1`; this URL
works only inside the backend Docker network and must never be routed through
Nginx, Cloudflare Tunnel, or a host-published port.

CPU inference is intentionally serialized with `OLLAMA_NUM_PARALLEL=1` and
`ai_agent.worker_count=1`. At most two models remain loaded. The defaults give
Ollama 4 CPUs, 5.5 GiB RAM, a five-minute keep-alive, and a modest 4096-token
context. Larger contexts consume more RAM and should be load-tested before
raising `OLLAMA_CONTEXT_LENGTH`. Resource limits and keep-alive are configurable
through the documented `OLLAMA_*` values in `.env`.

For Qwen3 completion models, set the completion provider's **Reasoning effort**
to `none` in LibreDesk's AI provider settings. This uses Qwen's non-thinking
mode for routine grounded support answers; leaving it blank makes the model
spend most of the response budget on hidden reasoning and can return an empty
answer when the token cap is reached.

## Backups

The `libredesk-backup.timer` user timer runs `~/libredesk/backup.sh` nightly.
It writes a PostgreSQL custom-format
dump, uploads archive, environment/encryption configuration, Ollama model
inventory, and checksums to `~/libredesk/backups`. Local backups are retained
for 30 days. Proxmox
snapshots provide a second recovery layer, but application-consistent database
dumps remain required before every LibreDesk upgrade.

The Ollama volume is a reproducible model cache and is not copied into every
nightly backup. This avoids duplicating several gigabytes of immutable model
blobs. On restore, recover `.env`, the database, and uploads first, then run
`docker compose up -d`. The initializer recreates `libredesk_ollama-models` and
re-pulls the two model names from `.env`; compare the resulting `ollama list`
with the backup's `ollama-models.txt`. A Proxmox snapshot remains the recovery
path when an exact offline copy of the model cache is required.

For a clean restore, verify `SHA256SUMS`, restore `environment.env` as `.env`,
extract `uploads.tar.gz` under `data/`, start `db` and `redis`, and load
`postgres.dump` with `pg_restore --no-owner --no-acl`. Then start the full
Compose project and confirm `app`, `db`, `redis`, `proxy`, and `ollama` are
healthy. Never delete or replace existing named volumes until the backup has
been verified and the restore is being performed intentionally.

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

Install the remotely managed tunnel connector with its scoped token:

```bash
sudo cloudflared service install <tunnel-token>
```

The LibreDesk General Settings root URL must be
`https://support.olcepks.ca` so email links, OAuth callbacks, the help center,
and live-chat assets use the public hostname.
