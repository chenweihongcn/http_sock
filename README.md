# Low Resource HTTP + SOCKS5 Proxy

面向 ARMv8 低资源设备的 HTTP + SOCKS5 代理平台，内置 SQLite 控制平面和 Web 管理台。

## 当前实现

- HTTP 代理，支持 CONNECT 隧道
- SOCKS5 代理，支持用户名密码认证
- SQLite 控制平面
- 管理员账号密码登录
- HttpOnly Cookie 会话
- 用户搜索分页
- 用户启用/禁用、删除、改密、重置流量
- 用户设备数限制、活跃设备查看与踢出
- 批量启用/禁用用户
- 管理员账号管理、会话列表、审计日志
- 真实客户端 IP 识别（支持可信反向代理头）

## 认证模型

- 管理员：账号密码登录，登录后由浏览器保存 HttpOnly Cookie 会话
- 代理用户：用户名密码认证，可用于 HTTP 代理和 SOCKS5 代理
- 旧 Bearer Token 管理登录：已停用

## 部署模式

- 推荐：OpenWrt 原生进程 + procd
- 备选：Docker

## OpenWrt 一键部署（推荐）

在 Windows PowerShell 中执行：

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\scripts\deploy-openwrt-native.ps1 -HostIP 192.168.50.94 -RootPassword ckp800810
```

脚本会自动完成：

1. 编译 linux/arm64 二进制
2. 上传到 OpenWrt
3. 安装到 `/usr/local/bin/proxy-server`
4. 创建环境文件 `/etc/proxy-platform.env`
5. 创建 procd 服务 `/etc/init.d/proxy-platform`
6. 从旧 Docker 目录迁移数据库到 `/mnt/mmc0-4/proxy-platform/proxy.db`（如果目标库不存在）
7. 停止旧 Docker 容器并启用原生服务

## Docker 快速启动（备选）

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

## 环境变量

- `LISTEN_HOST`：监听地址，默认 `0.0.0.0`
- `HTTP_PORT`：HTTP 代理端口，默认 `8899`
- `SOCKS5_PORT`：SOCKS5 端口，默认 `1080`
- `ADMIN_PORT`：管理端口，默认 `8088`
- `TRUST_PROXY_HEADERS`：是否信任代理头获取真实客户端 IP，默认 `false`
- `REAL_IP_HEADER`：真实 IP 头名，默认 `X-Forwarded-For`
- `DIAL_TIMEOUT`：出站拨号超时，默认 `15s`
- `CONTROL_PLANE_ENABLED`：是否启用 SQLite 控制平面，默认 `true`
- `DB_PATH`：SQLite 文件路径，默认 `./data/proxy.db`
- `SMB_ROOT_DIR`：SMB 用户目录根路径，默认 `/mnt/mmc0-4/proxy-platform/smb`
- `DEVICE_WINDOW`：设备数统计窗口，默认 `10m`
- `BOOTSTRAP_USER`：数据库为空时自动创建的第一个代理用户
- `BOOTSTRAP_PASS`：第一个代理用户密码
- `BOOTSTRAP_ADMIN_USER`：超级管理员账号，默认回退到 `BOOTSTRAP_USER`
- `BOOTSTRAP_ADMIN_PASS`：超级管理员密码，默认回退到 `BOOTSTRAP_PASS`
- `BOOTSTRAP_READONLY`：只读管理员账号，默认 `ops`
- `BOOTSTRAP_READONLY_PASS`：只读管理员密码，默认 `ops123456`
- `ADMIN_SESSION_TTL`：管理员会话时长，默认 `12h`
- `ADMIN_COOKIE_SECURE`：是否给管理 Cookie 打上 `Secure` 标记，默认 `false`
- `PASSWORD_MIN_LENGTH`：后台设置密码时的最小长度，默认 `8`
- `ADMIN_RATE_LIMIT_RPS`：管理 API 每个客户端 IP 的速率上限（请求/秒），默认 `20`
- `ADMIN_RATE_LIMIT_BURST`：管理 API 每个客户端 IP 的突发桶容量，默认 `60`
- `ADMIN_LOGIN_MAX_FAILS`：登录失败触发临时封禁的阈值，默认 `8`
- `ADMIN_LOGIN_FAIL_WINDOW`：登录失败计数窗口，默认 `15m`
- `ADMIN_LOGIN_BLOCK_FOR`：达到阈值后的封禁时长，默认 `15m`
- `ADMIN_UI_ALERT_RATE_LIMIT_DELTA`：管理台限流告警增量阈值，默认 `3`
- `ADMIN_UI_ALERT_LOGIN_FAIL_DELTA`：管理台登录失败告警增量阈值，默认 `5`
- `ADMIN_UI_ALERT_LOGIN_BLOCK_DELTA`：管理台登录封禁告警增量阈值，默认 `1`
- `ADMIN_UI_ALERT_COOLDOWN`：管理台告警冷却时间，默认 `5m`

如果 `CONTROL_PLANE_ENABLED=false`，服务回退到静态用户：

- `PROXY_USERS`：逗号分隔的 `user:pass` 列表

示例：

```bash
CONTROL_PLANE_ENABLED=true
DB_PATH="/mnt/mmc0-4/proxy-platform/proxy.db"
SMB_ROOT_DIR="/mnt/mmc0-4/proxy-platform/smb"
BOOTSTRAP_USER="vpn"
BOOTSTRAP_PASS="abc123456"
BOOTSTRAP_ADMIN_USER="admin"
BOOTSTRAP_ADMIN_PASS="admin123"
BOOTSTRAP_READONLY="ops"
BOOTSTRAP_READONLY_PASS="OpsPass123"
ADMIN_SESSION_TTL="12h"
PASSWORD_MIN_LENGTH="8"
TRUST_PROXY_HEADERS="true"
REAL_IP_HEADER="X-Forwarded-For"
ADMIN_RATE_LIMIT_RPS="20"
ADMIN_RATE_LIMIT_BURST="60"
ADMIN_LOGIN_MAX_FAILS="8"
ADMIN_LOGIN_FAIL_WINDOW="15m"
ADMIN_LOGIN_BLOCK_FOR="15m"
ADMIN_UI_ALERT_RATE_LIMIT_DELTA="3"
ADMIN_UI_ALERT_LOGIN_FAIL_DELTA="5"
ADMIN_UI_ALERT_LOGIN_BLOCK_DELTA="1"
ADMIN_UI_ALERT_COOLDOWN="5m"
```

### SMB 目录约定

- 每个用户对应目录：`/mnt/mmc0-4/proxy-platform/smb/<username>`
- 创建用户默认不创建目录，管理员“开通SMB”后才创建目录
- 管理员“关闭SMB”或删除用户时会清理目录
- 管理台用户详情会显示 SMB 目录路径

### WebDAV 兼容访问

- 地址：`http://<HOST>:8088/webdav/`
- 认证：使用代理用户名/密码（Basic Auth）
- 前提：该用户已由管理员开通 SMB
- 目录映射：每个用户仅能访问自己的目录 `SMB_ROOT_DIR/<username>`

## 本地运行

```bash
go run ./cmd/server
```

## 代理测试

HTTP 代理：

```bash
curl -x http://vpn:abc123456@127.0.0.1:8899 https://example.com -I
```

SOCKS5 代理：

```bash
curl --socks5 vpn:abc123456@127.0.0.1:1080 https://example.com -I
```

## 管理 API

### 登录

```bash
curl -i -X POST http://127.0.0.1:8088/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

返回头中会带 `Set-Cookie: admin_session=...`。
响应 JSON 会返回 `csrf_token`，后续所有写操作（POST/PUT/DELETE）都需要通过请求头 `X-CSRF-Token` 传回该值。

Linux/macOS 下可直接保存 Cookie：

```bash
curl -c cookie.txt -X POST http://127.0.0.1:8088/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

之后所有管理请求带上：

```bash
-b cookie.txt
```

同时请在写请求中增加：

```bash
-H "X-CSRF-Token: <csrf_token>"
```

### 当前身份

```bash
curl http://127.0.0.1:8088/api/admin/me -b cookie.txt
```

### 用户列表（支持搜索分页）

```bash
curl "http://127.0.0.1:8088/api/admin/users?q=vpn&status=1&offset=0&limit=20" -b cookie.txt
```

### 创建用户

```bash
curl -X POST http://127.0.0.1:8088/api/admin/users \
  -b cookie.txt \
  -H "Content-Type: application/json" \
  -d '{"username":"u1","password":"p12345678","max_devices":2,"quota_bytes":1073741824,"expires_at":0}'
```

### 删除用户

```bash
curl -X DELETE http://127.0.0.1:8088/api/admin/users/u1 -b cookie.txt
```

### 修改用户密码

```bash
curl -X POST http://127.0.0.1:8088/api/admin/users/u1/password \
  -b cookie.txt \
  -H "Content-Type: application/json" \
  -d '{"password":"newpass123"}'
```

### 重置用户流量

```bash
curl -X POST http://127.0.0.1:8088/api/admin/users/u1/usage-reset -b cookie.txt
```

### 设置用户设备数

```bash
curl -X POST http://127.0.0.1:8088/api/admin/users/u1/set-devices \
  -b cookie.txt \
  -H "Content-Type: application/json" \
  -d '{"max_devices":3}'
```

### 查看用户活跃设备

```bash
curl http://127.0.0.1:8088/api/admin/users/u1/devices -b cookie.txt
```

### 踢出用户设备

```bash
curl -X DELETE http://127.0.0.1:8088/api/admin/users/u1/devices/1.2.3.4 -b cookie.txt
```

### 批量启用/禁用

```bash
curl -X POST http://127.0.0.1:8088/api/admin/users/batch-status \
  -b cookie.txt \
  -H "Content-Type: application/json" \
  -d '{"usernames":["u1","u2"],"enabled":false}'
```

### 创建管理员

```bash
curl -X POST http://127.0.0.1:8088/api/admin/admins \
  -b cookie.txt \
  -H "Content-Type: application/json" \
  -d '{"username":"ops2","password":"ops2pass88","role":"readonly"}'
```

### 修改管理员密码

```bash
curl -X POST http://127.0.0.1:8088/api/admin/admins/ops2/password \
  -b cookie.txt \
  -H "Content-Type: application/json" \
  -d '{"password":"ops2pass99"}'
```

### 当前管理员修改自己的密码

```bash
curl -X POST http://127.0.0.1:8088/api/admin/profile/password \
  -b cookie.txt \
  -H "Content-Type: application/json" \
  -d '{"old_password":"admin123","new_password":"NewAdminPass456"}'
```

### 会话列表与下线

```bash
curl http://127.0.0.1:8088/api/admin/sessions -b cookie.txt
```

```bash
curl -X DELETE http://127.0.0.1:8088/api/admin/sessions/SESSION_ID -b cookie.txt
```

### 审计日志

```bash
curl "http://127.0.0.1:8088/api/admin/audits?actor=admin&limit=50" -b cookie.txt
```

### 安全统计（限流/防爆破）

```bash
curl http://127.0.0.1:8088/api/admin/security-stats -b cookie.txt
```

返回示例字段：

- `admin_rate_limited_total`：管理 API 被限流总次数
- `admin_login_failed_total`：管理员登录失败总次数
- `admin_login_blocked_total`：管理员登录被封禁命中总次数
- `blocked_active`：当前仍在封禁窗口内的键数量

当触发限流或防爆破时，接口会返回 `429`：

- 管理 API 限流：`{"error":"too_many_requests","reason":"admin_api_rate_limited"}`
- 登录封禁：`{"error":"too_many_attempts","retry_after_seconds":123}`

审计日志可按以下动作筛选安全事件：

- `admin_api_rate_limited`：命中管理 API 限流
- `admin_login_failed`：管理员登录失败
- `admin_login_blocked`：管理员登录被临时封禁

管理台内置了安全统计自动告警（前端每 60 秒轮询，可由环境变量调整）：

- 仅当增量达到阈值才弹窗：限流 `+3`、登录失败 `+5`、登录封禁 `+1`
- 告警冷却时间：5 分钟（冷却内不重复弹窗）
- 审计页会显示当前生效的阈值与冷却时间（以服务端返回配置为准）

## 管理角色

- `super`：允许全部管理操作
- `readonly`：仅允许读取数据

只读管理员对写请求会收到：

```json
{"error":"forbidden","reason":"readonly_cannot_write"}
```

## 说明

- 管理 Cookie 为 HttpOnly，前端 JavaScript 无法直接读取
- 用户和管理员密码会以哈希形式保存；旧的明文代理用户密码会在首次成功登录时自动升级为哈希
- 建议在公网暴露管理口时启用反向代理 HTTPS，并将 `ADMIN_COOKIE_SECURE=true`
- 仅在可信反代链路下开启 `TRUST_PROXY_HEADERS=true`，避免伪造请求头导致来源 IP 污染

