# Verification Guide

This project includes one-click verification for control plane and proxy data plane.

## Quick Verify on iStoreOS / Linux

Run from repository root:

```sh
sh scripts/verify-control-plane.sh
```

The script validates:

1. Docker daemon is ready.
2. Stack can be started with Docker Compose.
3. Admin API health endpoint is reachable.
4. Admin token can create a user.
5. User appears in list API.
6. HTTP proxy authentication and forwarding work.
7. SOCKS5 proxy authentication and forwarding work.
8. Disabled user is blocked (HTTP 407 expected).
9. Readonly admin can read but cannot write (403 expected on write).
10. Super admin can create and rotate admin tokens; old token is rejected after rotation.

## Optional Overrides

You can override environment variables when running the script:

```sh
ADMIN_TOKEN=change-me READONLY_ADMIN_TOKEN=readonly-change-me HTTP_PORT=8899 SOCKS5_PORT=1080 ADMIN_PORT=8088 sh scripts/verify-control-plane.sh
```

## If Verification Fails

Inspect service logs:

```sh
docker compose -f deploy/docker-compose.yml logs --tail=200
```

Check running containers:

```sh
docker compose -f deploy/docker-compose.yml ps
```
