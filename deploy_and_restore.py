#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""部署新版本并恢复用户数据"""

import paramiko
import urllib.request
import urllib.parse
import http.cookiejar
import json
import time
import os

SERVER_SSH = "192.168.50.94"
SERVER_HTTP = "192.168.50.94:8088"
SSH_USER = "root"
SSH_PASSWORD = "ckp800810"
TAR_FILE = r"i:\HTTP\http_sock_arm64.tar"
REMOTE_TAR = "/mnt/mmc0-4/docker/http_sock_arm64.tar"

# ─── SSH 部署 ─────────────────────────────────────────────────────
print("="*60)
print("[1/4] 上传并部署新镜像")
print("="*60)

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(SERVER_SSH, username=SSH_USER, password=SSH_PASSWORD, timeout=30)
sftp = client.open_sftp()

def ssh(cmd, show=True):
    stdin, stdout, stderr = client.exec_command(cmd)
    out = stdout.read().decode('utf-8').strip()
    err = stderr.read().decode('utf-8').strip()
    if show and out: print("  " + out)
    if show and err: print("  [err] " + err)
    return out

print("停止旧容器...")
ssh("docker stop lowres-proxy 2>/dev/null || true")
ssh("docker rm   lowres-proxy 2>/dev/null || true")

print(f"上传镜像 ({os.path.getsize(TAR_FILE)//1024}KB)...")
sftp.put(TAR_FILE, REMOTE_TAR)
print("  ✓ 上传完成")

print("加载镜像...")
out = ssh("docker rmi chenweihongcn/http_sock:arm64 2>/dev/null || true", show=False)
ssh("docker load -i " + REMOTE_TAR)

print("启动容器...")
docker_cmd = (
    "docker run -d --name lowres-proxy "
    "--restart=always "
    "-p 8899:8899 -p 1080:1080 -p 8088:8088 "
    "-v /mnt/mmc0-4/docker/http_sock_data:/app/data "
    "-e BOOTSTRAP_ADMIN_PASS=Admin2026Strong9X "
    "-e BOOTSTRAP_READONLY=ops "
    "-e BOOTSTRAP_READONLY_PASS=OpsPass123 "
    "chenweihongcn/http_sock:arm64"
)
ssh(docker_cmd)
time.sleep(3)

status = ssh("docker ps --filter name=lowres-proxy --format '{{.Status}}'", show=False)
print(f"  容器状态: {status}")

sftp.close()
client.close()

# ─── 恢复用户数据 ─────────────────────────────────────────────────
print("\n" + "="*60)
print("[2/4] 恢复用户数据")
print("="*60)

cj = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))

def api(method, path, body=None):
    url = "http://" + SERVER_HTTP + path
    data = json.dumps(body).encode() if body else None
    hdrs = {'Content-Type':'application/json'} if data else {}
    r = urllib.request.Request(url, data=data, headers=hdrs, method=method)
    try:
        with opener.open(r) as res:
            return res.status, json.loads(res.read())
    except urllib.error.HTTPError as e:
        try: return e.code, json.loads(e.read())
        except: return e.code, {}

# 登陆
s, b = api("POST", "/api/admin/login", {"username": "admin", "password": "Admin2026Strong9X"})
if s != 200:
    print(f"✗ 登陆失败 {s}: {b}")
    exit(1)
print(f"✓ 登陆成功: {b.get('username')} / {b.get('role')}")

# 先获取当前用户列表
s, b = api("GET", "/api/admin/users?limit=50")
existing = {u['username'] for u in b.get('items', [])}
print(f"  当前已有用户: {list(existing)}")

# 需要恢复的用户（从之前截图还原）
users_to_restore = [
    {"username": "legalcoop", "password": "Legalcoop123!", "expires_at": 1752332445, "max_devices": 1, "quota_bytes": 0},
    {"username": "vpn",       "password": "Vpn12345678",   "expires_at": 0,          "max_devices": 1, "quota_bytes": 0},
]

for user in users_to_restore:
    if user['username'] in existing:
        print(f"  跳过（已存在）: {user['username']}")
        continue
    payload = {
        "username": user['username'],
        "password": user['password'],
        "expires_at": user['expires_at'],
        "quota_bytes": user['quota_bytes'],
        "max_devices": user['max_devices'],
    }
    s, b = api("POST", "/api/admin/users", payload)
    if s == 201:
        print(f"  ✓ 已恢复用户: {user['username']} (到期: {'永久' if user['expires_at']==0 else str(user['expires_at'])})")
        # 恢复 legalcoop 的过期时间
        if user['expires_at'] > 0:
            api("POST", f"/api/admin/users/{urllib.parse.quote(user['username'])}/extend", {"days": 0})
    else:
        print(f"  ✗ 恢复用户失败 {user['username']}: {s} {b}")

# 最终确认
s, b = api("GET", "/api/admin/users?limit=50")
users = b.get('items', [])
print(f"\n  ✓ 最终用户列表 ({b.get('total')} 个):")
for u in users:
    expires = '永久' if not u['expires_at'] else time.strftime('%Y/%m/%d', time.localtime(u['expires_at']))
    status = '启用' if u['status'] == 1 else '禁用'
    print(f"    - {u['username']}: {status}, 到期={expires}, 设备={u['active_ips']}/{u['max_devices']}")

print("\n" + "="*60)
print("✅ 部署完成！")
print("="*60)
print(f"\n管理界面: http://{SERVER_SSH}:8088/")
print("现在所有按钮均使用页内 Modal 确认和 Toast 提示")
print("不再依赖浏览器弹窗，不受弹窗拦截器影响")
