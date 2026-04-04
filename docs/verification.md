# Verification Guide

当前版本使用管理员账号密码登录和 HttpOnly Cookie 会话，不再使用 Bearer Token。

## Quick Verify on iStoreOS / Linux

在仓库根目录执行：

```sh
sh scripts/verify-control-plane.sh
```

脚本会验证：

1. Docker daemon 已就绪。
2. 容器栈可正常启动。
3. 管理健康检查可达。
4. 超级管理员可登录并建立 Cookie 会话。
5. 可创建测试用户，且用户可出现在分页列表中。
6. HTTP 代理与 SOCKS5 代理均可正常认证转发。
7. 禁用用户后，HTTP 代理返回 407。
8. 只读管理员可读不可写。
9. 超级管理员可创建新的平台管理员。
10. 新管理员可通过账号密码登录并访问管理接口。

## Optional Overrides

可按需覆盖环境变量：

```sh
BOOTSTRAP_ADMIN_USER=admin \
BOOTSTRAP_ADMIN_PASS=StrongAdmin123 \
BOOTSTRAP_READONLY=ops \
BOOTSTRAP_READONLY_PASS=OpsPass123 \
HTTP_PORT=8899 \
SOCKS5_PORT=1080 \
ADMIN_PORT=8088 \
sh scripts/verify-control-plane.sh
```

## Manual Checks

管理员登录：

```sh
curl -i -c cookie.txt -X POST http://127.0.0.1:8088/api/admin/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"StrongAdmin123"}'
```

当前身份：

```sh
curl -b cookie.txt http://127.0.0.1:8088/api/admin/me
```

用户分页搜索：

```sh
curl -b cookie.txt "http://127.0.0.1:8088/api/admin/users?q=vpn&offset=0&limit=20"
```

活跃设备列表：

```sh
curl -b cookie.txt http://127.0.0.1:8088/api/admin/users/vpn/devices
```

## If Verification Fails

检查服务日志：

```sh
docker compose -f deploy/docker-compose.yml logs --tail=200
```

检查容器状态：

```sh
docker compose -f deploy/docker-compose.yml ps
```
