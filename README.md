# Low Resource HTTP + SOCKS5 Proxy

A minimal Go proxy service for ARMv8 devices (for example iStoreOS on NanoPi R2S Plus).

## Features in current implementation

- HTTP proxy with CONNECT tunneling
- SOCKS5 proxy with CONNECT support
- Username/password authentication
- SQLite-backed control plane with admin API
- User policies: disable/enable, expiry, quota, max active IP devices
- Default HTTP port is `8899` and can be changed
- Docker-first deployment

## Quick Start (Docker)

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

## Environment variables

- `LISTEN_HOST`: listen host, default `0.0.0.0`
- `HTTP_PORT`: HTTP proxy port, default `8899`
- `SOCKS5_PORT`: SOCKS5 proxy port, default `1080`
- `ADMIN_PORT`: admin API port, default `8088`
- `DIAL_TIMEOUT`: outbound dial timeout, default `15s`
- `CONTROL_PLANE_ENABLED`: set `true` to use SQLite control plane
- `DB_PATH`: SQLite path, default `./data/proxy.db`
- `ADMIN_TOKEN`: bearer token for admin API
- `READONLY_ADMIN_TOKEN`: read-only bearer token for admin API
- `DEVICE_WINDOW`: active IP window for device counting, default `10m`
- `BOOTSTRAP_USER`: first user auto-created when DB is empty
- `BOOTSTRAP_PASS`: bootstrap password
- `BOOTSTRAP_READONLY`: bootstrap readonly admin username

If `CONTROL_PLANE_ENABLED=false`, service falls back to static users from:

- `PROXY_USERS`: comma-separated user list in `user:pass` format

Static user example:

```bash
PROXY_USERS="alice:pass1,bob:pass2"
```

Control plane example:

```bash
CONTROL_PLANE_ENABLED=true
ADMIN_TOKEN="replace-with-strong-token"
READONLY_ADMIN_TOKEN="replace-with-readonly-token"
DB_PATH="./data/proxy.db"
BOOTSTRAP_USER="admin"
BOOTSTRAP_PASS="admin123"
BOOTSTRAP_READONLY="ops"
```

## Admin Roles

- `super`: can read and write all admin APIs.
- `readonly`: can only call `GET` APIs.

Readonly write attempts return:

```json
{"error":"forbidden","reason":"readonly_cannot_write"}
```

## Local Run

```bash
go run ./cmd/server
```

## Test HTTP proxy

```bash
curl -x http://admin:admin123@127.0.0.1:8899 https://example.com -I
```

## Test SOCKS5 proxy

```bash
curl --socks5 admin:admin123@127.0.0.1:1080 https://example.com -I
```

## Admin API

Set header:

```bash
Authorization: Bearer <ADMIN_TOKEN>
```

Current actor info:

```bash
curl http://127.0.0.1:8088/api/admin/me \
	-H "Authorization: Bearer change-me"
```

Audit logs:

```bash
curl "http://127.0.0.1:8088/api/admin/audits?limit=50" \
	-H "Authorization: Bearer change-me"
```

Audit filters (`actor`, `action`, `target`, `from`, `to`):

```bash
curl "http://127.0.0.1:8088/api/admin/audits?actor=admin&action=create_user&limit=20" \
	-H "Authorization: Bearer change-me"
```

List admins:

```bash
curl http://127.0.0.1:8088/api/admin/admins \
	-H "Authorization: Bearer change-me"
```

Create admin (super only):

```bash
curl -X POST http://127.0.0.1:8088/api/admin/admins \
	-H "Authorization: Bearer change-me" \
	-H "Content-Type: application/json" \
	-d '{"username":"ops2","token":"ops2-token","role":"readonly"}'
```

Set admin role (super only):

```bash
curl -X POST http://127.0.0.1:8088/api/admin/admins/ops2/set-role \
	-H "Authorization: Bearer change-me" \
	-H "Content-Type: application/json" \
	-d '{"role":"super"}'
```

Rotate admin token (super only):

```bash
curl -X POST http://127.0.0.1:8088/api/admin/admins/ops2/rotate-token \
	-H "Authorization: Bearer change-me" \
	-H "Content-Type: application/json" \
	-d '{"token":"ops2-token-new"}'
```

Health check:

```bash
curl http://127.0.0.1:8088/api/admin/healthz
```

Create user:

```bash
curl -X POST http://127.0.0.1:8088/api/admin/users \
	-H "Authorization: Bearer change-me" \
	-H "Content-Type: application/json" \
	-d '{"username":"u1","password":"p1","max_devices":2,"quota_bytes":1073741824,"expires_at":0}'
```

List users:

```bash
curl http://127.0.0.1:8088/api/admin/users \
	-H "Authorization: Bearer change-me"
```

Disable user:

```bash
curl -X POST http://127.0.0.1:8088/api/admin/users/u1/disable \
	-H "Authorization: Bearer change-me"
```

Extend expiry by days:

```bash
curl -X POST http://127.0.0.1:8088/api/admin/users/u1/extend \
	-H "Authorization: Bearer change-me" \
	-H "Content-Type: application/json" \
	-d '{"days":30}'
```

Top up quota:

```bash
curl -X POST http://127.0.0.1:8088/api/admin/users/u1/topup \
	-H "Authorization: Bearer change-me" \
	-H "Content-Type: application/json" \
	-d '{"bytes":536870912}'
```

## Next Steps

- Add role-based admin accounts (super admin and read-only operator)
- Add session logs and richer audit query endpoints
- Add lightweight admin web UI
- Add optional PostgreSQL backend for multi-node growth
