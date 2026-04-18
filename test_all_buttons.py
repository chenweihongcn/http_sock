#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""全面测试管理台所有按钮功能"""

import urllib.request
import urllib.error
import urllib.parse
import json
import http.cookiejar
import sys
import time

SERVER = "192.168.50.94:8088"
BASE = f"http://{SERVER}"
PASS = "Admin2026Strong9X"

cookie_jar = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookie_jar))

ok = 0
fail = 0

def req(method, path, body=None, expect_status=200):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    headers = {'Content-Type': 'application/json'} if data else {}
    r = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with opener.open(r) as res:
            resp_body = res.read().decode()
            status = res.status
            try:
                resp = json.loads(resp_body)
            except:
                resp = resp_body
            return status, resp
    except urllib.error.HTTPError as e:
        try:
            resp = json.loads(e.read().decode())
        except:
            resp = {}
        return e.code, resp

def check(label, status, body, expected=200):
    global ok, fail
    passed = (status == expected)
    mark = "✓" if passed else "✗"
    detail = json.dumps(body) if isinstance(body, dict) else str(body)[:80]
    print(f"  {mark} [{status}] {label}: {detail}")
    if passed:
        ok += 1
    else:
        fail += 1
    return passed

print("="*65)
print("管理台按钮功能全面测试")
print("="*65)

# ─── 1. 登陆 ───────────────────────────────────────────────────────
print("\n【1】登陆")
s, b = req("POST", "/api/admin/login", {"username": "admin", "password": PASS})
if not check("admin 登陆", s, b, 200):
    print("  ✗ 登陆失败，终止测试")
    sys.exit(1)

# ─── 2. 获取当前用户列表 ────────────────────────────────────────────
print("\n【2】获取用户列表")
s, b = req("GET", "/api/admin/users?offset=0&limit=20")
check("GET /api/admin/users", s, b, 200)
if isinstance(b, dict):
    users = b.get("items", [])
    print(f"     共 {b.get('total', 0)} 个用户: {[u['username'] for u in users]}")
else:
    users = []

# ─── 3. 创建测试用户 ────────────────────────────────────────────────
print("\n【3】创建测试用户 _btntest_")
s, b = req("POST", "/api/admin/users", {
    "username": "_btntest_",
    "password": "BtnTest1234",
    "expires_at": 0,
    "quota_bytes": 0,
    "max_devices": 2
})
if s == 201:
    check("创建用户", s, b, 201)
else:
    # 可能已存在
    check("创建用户（已有则忽略）", s, b, 400)

# ─── 4. 逐个测试按钮对应的 API ────────────────────────────────────
TEST_USER = "_btntest_"
print(f"\n【4】对 {TEST_USER} 测试各按钮 API")

s, b = req("POST", f"/api/admin/users/{TEST_USER}/disable")
check("禁用按钮 → POST /disable", s, b, 200)

# 验证状态已改变
s, b = req("GET", f"/api/admin/users?q={TEST_USER}")
if s == 200 and b.get("items"):
    u = b["items"][0]
    changed = u["status"] == 0
    mark = "✓" if changed else "✗"
    print(f"  {mark}  验证：禁用后 status={u['status']} (期望 0)")
    if changed: ok += 1
    else: fail += 1

s, b = req("POST", f"/api/admin/users/{TEST_USER}/enable")
check("启用按钮 → POST /enable", s, b, 200)

s, b = req("POST", f"/api/admin/users/{TEST_USER}/set-devices",
           {"max_devices": 3})
check("设备按钮 → POST /set-devices", s, b, 200)

s, b = req("POST", f"/api/admin/users/{TEST_USER}/usage-reset")
check("清流量按钮 → POST /usage-reset", s, b, 200)

s, b = req("POST", f"/api/admin/users/{TEST_USER}/password",
           {"password": "NewPass9999"})
check("改密码按钮 → POST /password", s, b, 200)

s, b = req("GET", f"/api/admin/users/{TEST_USER}/devices")
check("设备列表 → GET /devices", s, b, 200)

# ─── 5. 测试管理员标签的按钮 ─────────────────────────────────────
print("\n【5】管理员页按钮")
s, b = req("GET", "/api/admin/admins")
check("GET /api/admin/admins", s, b, 200)
if isinstance(b, dict):
    admins = b.get("items", [])
    print(f"     共 {len(admins)} 个管理员: {[a['username'] for a in admins]}")

# 找一个非自身的管理员来测试
other_admin = next((a for a in (admins if isinstance(b, dict) and b.get("items") else [])
                    if a["username"] != "admin"), None)
if other_admin:
    uname = other_admin["username"]
    print(f"  对 {uname} 测试操作...")
    s, b = req("POST", f"/api/admin/admins/{uname}/disable")
    check(f"禁用管理员 {uname}", s, b, 200)
    s, b = req("POST", f"/api/admin/admins/{uname}/enable")
    check(f"启用管理员 {uname}", s, b, 200)

# ─── 6. 批量操作 ───────────────────────────────────────────────────
print("\n【6】批量操作按钮")
s, b = req("POST", "/api/admin/users/batch-status",
           {"usernames": [TEST_USER], "enabled": False})
check("批量禁用 → POST /batch-status", s, b, 200)
s, b = req("POST", "/api/admin/users/batch-status",
           {"usernames": [TEST_USER], "enabled": True})
check("批量启用 → POST /batch-status", s, b, 200)

# ─── 7. 只读账号操作权限 ──────────────────────────────────────────
print("\n【7】只读账号 ops 权限验证")
s, b = req("POST", "/api/admin/login", {"username": "ops", "password": "OpsPass123"})
check("ops 登陆", s, b, 200)

s, b = req("POST", f"/api/admin/users/{TEST_USER}/disable")
check("ops 禁用用户（期望 403）", s, b, 403)

# ─── 8. 切回 admin，清理测试用户 ─────────────────────────────────
print("\n【8】清理测试数据")
s, b = req("POST", "/api/admin/login", {"username": "admin", "password": PASS})
check("重新以 admin 登陆", s, b, 200)

s, b = req("DELETE", f"/api/admin/users/{TEST_USER}")
check("删除按钮 → DELETE /users/_btntest_", s, b, 200)

# ─── 汇总 ─────────────────────────────────────────────────────────
print("\n" + "="*65)
total = ok + fail
print(f"测试结果：{ok}/{total} 通过，{fail} 失败")
if fail == 0:
    print("✅ 所有 API 功能正常！")
    print("\n如果浏览器按钮仍无反应，问题在前端——")
    print("  → 按 F12 查看 Console 有无红色错误")
    print("  → 检查 Network 标签中点击按钮后是否有 HTTP 请求发出")
else:
    print("❌ 部分 API 有问题，见上方 ✗ 标记")
print("="*65)
