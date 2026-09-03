# Sub2API Custom Container Auto-Update

This deployment option checks `ghcr.io/anti2077/sub2api:custom` every five
minutes and recreates only the `sub2api` Compose service when the image digest
changes. PostgreSQL, Redis, named volumes, bind mounts, and Compose environment
values are not recreated.

The updater runs as a host-side systemd service. It does not mount the Docker
socket into the Sub2API container or into a third-party updater container.

## Support boundary

- Linux host with systemd, Docker Engine, Docker Compose v2, and `flock`.
- A running Compose service named `sub2api` with the repository health check.
- The public `ghcr.io/anti2077/sub2api:custom` image. If the package becomes
  private, authenticate the root Docker client with a read-only package token
  before enabling the timer.
- Apple `container`, Kubernetes, Swarm, rootless Docker with a nonstandard
  socket, and multi-replica zero-downtime rollout are outside this template.

Automatic container rollback cannot undo a database migration that already
ran during application startup. Keep current PostgreSQL backups and test schema
changes before merging them into `Anti2077/custom`.

## Install

Run the installer from a checkout of the custom branch. Point
`--compose-dir` at the directory containing the Compose file and `.env` that
are actually used by the deployment:

```bash
cd /path/to/sub2api/deploy
sudo ./install-container-autoupdate.sh \
  --compose-dir /absolute/path/to/current/deployment
```

For a nondefault Compose or environment filename:

```bash
sudo ./install-container-autoupdate.sh \
  --compose-dir /absolute/path/to/current/deployment \
  --compose-file docker-compose.local.yml \
  --env-file .env
```

The installer performs a read-only Compose configuration validation before it
writes anything. It then installs:

- `/usr/local/libexec/sub2api-container-update`
- `/etc/sub2api-container-update/docker-compose.auto-update.yml`
- `/etc/sub2api-container-update.env`
- `sub2api-container-update.service` and `.timer`

Existing updater configuration is backed up with a UTC timestamp before an
installer rerun replaces it. Deployment data and the deployment `.env` are
never copied or modified.

## Update flow

1. A push to `Anti2077/custom` builds and publishes the moving `custom` image.
2. The systemd timer pulls that image without stopping the running container.
3. If the digest is unchanged, the updater exits without recreating anything.
4. If it changed, the current image is tagged locally as `rollback-local`.
5. Compose recreates only `sub2api`; PostgreSQL and Redis keep running.
6. The updater waits up to 180 seconds for the application health check.
7. An unhealthy replacement is reverted to the previous image. The failed
   digest is quarantined until a different image is published.

There is a brief application interruption while the single Sub2API container
is recreated. The timer never starts a service that an operator stopped.

## Operate and verify

Trigger a check immediately:

```bash
sudo systemctl start sub2api-container-update.service
```

Inspect the schedule and recent result:

```bash
sudo systemctl status sub2api-container-update.timer
sudo systemctl status sub2api-container-update.service
```

Follow updater logs:

```bash
sudo journalctl -u sub2api-container-update.service -f
```

Confirm the running application image and health:

```bash
docker inspect --format '{{.Config.Image}} {{.State.Health.Status}}' sub2api
docker compose ps sub2api
```

`systemctl start` runs the one-shot check now. `status` shows the last exit
result and next timer activation. `journalctl -f` follows new log entries.

## Pause and resume

Pause checks without touching the running application:

```bash
sudo systemctl disable --now sub2api-container-update.timer
```

Resume the five-minute schedule:

```bash
sudo systemctl enable --now sub2api-container-update.timer
```

## Remove

Removing the updater does not stop or recreate the running Sub2API container:

```bash
sudo systemctl disable --now sub2api-container-update.timer
sudo rm -f \
  /etc/systemd/system/sub2api-container-update.service \
  /etc/systemd/system/sub2api-container-update.timer \
  /etc/sub2api-container-update.env \
  /etc/sub2api-container-update/docker-compose.auto-update.yml \
  /usr/local/libexec/sub2api-container-update
sudo systemctl daemon-reload
```

The last failed-image marker remains under
`/var/lib/sub2api-container-update` for audit purposes and can be removed
separately when it is no longer needed.

## Rollback

An unhealthy update is rolled back automatically. To pin a known image
manually, stop the timer first and edit `/etc/sub2api-container-update.env`:

```text
SUB2API_IMAGE=ghcr.io/anti2077/sub2api:custom-<7-character-commit>
```

Then run one update check:

```bash
sudo systemctl start sub2api-container-update.service
```

Keep the timer disabled while pinned. After a corrected image is published,
change `SUB2API_IMAGE` back to `ghcr.io/anti2077/sub2api:custom`, trigger one
manual check, verify health, and then re-enable the timer.

The automatic rollback marker is stored at
`/var/lib/sub2api-container-update/failed-image-id`. Do not delete it merely to
retry an image that is still known to be unhealthy.

## Security notes

- The systemd unit runs as root because Docker Engine's control socket is
  root-owned, but the script accepts only the Anti2077 GHCR repository and the
  `sub2api` service name.
- The service receives a read-only view of the host filesystem except for its
  lock and state directories.
- No Docker socket is exposed inside the web application.
- No registry credential is stored in the repository or updater environment
  template.
