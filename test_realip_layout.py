#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
验证真实IP与管理台布局改动：
1. /api/admin/me 返回 client_ip, remote_ip, trust_proxy_headers, real_ip_header
2. 设备列表 API 返回 trust_proxy_headers, real_ip_header 元字段
3. HTML 包含新布局标志 (overview-grid, workspace-grid, meClientIP)
4. 旧功能（用户列表、批量操作、审计等）不受破坏
"""
import json
import urllib.request
import urllib.error
import http.cookiejar
import sys

BASE = "http://192.168.50.94:8088"
ADMIN_USER = "admin"
ADMIN_PASS = "Admin2026Strong9X"

ok_count = 0
fail_count = 0

def ok(name):
    global ok_count
    ok_count += 1
    print(f"  ✓  {name}")

def fail(name, reason=""):
    global fail_count
    fail_count += 1
    print(f"  ✗  {name}" + (f": {reason}" if reason else ""))

# Setup session
cj = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))

def api(path, method="GET", body=None, headers=None):
    url = BASE + path
    h = {"Content-Type": "application/json"}
    if headers:
        h.update(headers)
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, headers=h, method=method)
    try:
        with opener.open(req, timeout=10) as resp:
            return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        try:
            body = json.loads(e.read().decode())
        except Exception:
            body = {}
        return e.code, body

print("\n====== Phase 1: 登录 ======")
status, data = api("/api/admin/login", "POST", {"username": ADMIN_USER, "password": ADMIN_PASS})
if status == 200 and data.get("ok"):
    ok("admin login")
else:
    fail("admin login", f"status={status} data={data}")
    sys.exit(1)

print("\n====== Phase 2: /api/admin/me 新字段 ======")
status, me = api("/api/admin/me")
if status == 200:
    ok(f"/api/admin/me status=200")
else:
    fail("/api/admin/me", f"status={status}")

for field in ("username", "role", "client_ip", "remote_ip", "trust_proxy_headers", "real_ip_header", "forwarded_for"):
    if field in me:
        ok(f"me.{field} = {repr(me[field])}")
    else:
        fail(f"me.{field} missing", f"keys={list(me.keys())}")

trust = me.get("trust_proxy_headers")
if isinstance(trust, bool):
    ok(f"trust_proxy_headers is bool ({trust})")
else:
    fail("trust_proxy_headers type invalid", f"value={trust!r}")

if me.get("client_ip") and me.get("remote_ip"):
    if trust is False:
        if me.get("client_ip") == me.get("remote_ip"):
            ok("client_ip == remote_ip when trust=false")
        else:
            fail("client_ip != remote_ip when trust=false", f"{me.get('client_ip')} vs {me.get('remote_ip')}")
    elif trust is True:
        if me.get("real_ip_header"):
            ok("real_ip_header present when trust=true")
        else:
            fail("real_ip_header missing when trust=true")
        ok("client_ip/remote_ip present when trust=true")
else:
    fail("client_ip or remote_ip is empty", str(me))

print("\n====== Phase 3: 设备列表元字段 ======")
# 使用 vpn 用户（已知有设备）或任意已存在用户
status, users_data = api("/api/admin/users?limit=5")
test_username = None
if status == 200 and users_data.get("items"):
    test_username = users_data["items"][0]["username"]

if test_username:
    status, dev_data = api(f"/api/admin/users/{test_username}/devices")
    if status == 200:
        ok(f"devices for '{test_username}' status=200")
        if "trust_proxy_headers" in dev_data:
            ok(f"devices.trust_proxy_headers = {dev_data['trust_proxy_headers']}")
        else:
            fail("devices.trust_proxy_headers missing", str(list(dev_data.keys())))
        if "real_ip_header" in dev_data:
            ok(f"devices.real_ip_header = {dev_data['real_ip_header']}")
        else:
            fail("devices.real_ip_header missing")
    else:
        fail(f"devices for '{test_username}'", f"status={status}")
else:
    fail("no user found to test devices")

print("\n====== Phase 4: HTML 布局标志 ======")
try:
    with urllib.request.urlopen(BASE + "/", timeout=10) as resp:
        html = resp.read().decode("utf-8")
    for marker in ("overview-grid", "workspace-grid", "meClientIP", "meRemoteIP", "meIPMode",
                   "IP识别方式", "客户端IP", "IP来源说明", "操作台：创建代理用户"):
        if marker in html:
            ok(f"HTML contains '{marker}'")
        else:
            fail(f"HTML missing '{marker}'")
except Exception as e:
    fail("fetch UI HTML", str(e))

print("\n====== Phase 5: 旧功能回归 ======")
# user list
status, data = api("/api/admin/users?limit=10")
if status == 200 and "items" in data and "total" in data:
    ok(f"user list total={data['total']}")
else:
    fail("user list", f"status={status}")

# tags
status, data = api("/api/admin/users/tags")
if status == 200 and "items" in data:
    ok("user tags ok")
else:
    fail("user tags", f"status={status}")

# sessions
status, data = api("/api/admin/sessions")
if status == 200 and "items" in data:
    ok(f"sessions ok count={len(data['items'])}")
else:
    fail("sessions", f"status={status}")

# user overview aggregate
status, data = api("/api/admin/users/legalcoop/overview")
if status == 200 and "user" in data and "devices" in data and "audits" in data:
    ok("user overview aggregate")
else:
    fail("user overview aggregate", f"status={status}")

# audits
status, data = api("/api/admin/audits?limit=10")
if status == 200 and "items" in data:
    ok(f"audits ok count={len(data['items'])}")
else:
    fail("audits", f"status={status}")

# healthz
status, data = api("/api/admin/healthz")
if status == 200 and data.get("ok"):
    ok("healthz")
else:
    fail("healthz", f"status={status}")

print(f"\n{'='*40}")
print(f"结果: ✓ {ok_count} 通过 / ✗ {fail_count} 失败")
print('='*40)
sys.exit(0 if fail_count == 0 else 1)
