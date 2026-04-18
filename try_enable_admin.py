#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""验证 readonly 管理员无法执行写操作（含 CSRF 头）。"""

import argparse
import getpass
import json
import os
import sys
import urllib.error
import urllib.request
import http.cookiejar


def normalize_base(base):
    b = base.strip()
    if not b:
        b = "http://127.0.0.1:8088"
    if not b.startswith("http://") and not b.startswith("https://"):
        b = "http://" + b
    return b.rstrip("/")


def parse_args():
    p = argparse.ArgumentParser(description="readonly 管理员写操作拒绝验证")
    p.add_argument("--base", default=os.getenv("ADMIN_BASE", "http://127.0.0.1:8088"), help="管理地址，例如 http://192.168.50.94:8088")
    p.add_argument("--username", default=os.getenv("ADMIN_USER", "ops"), help="管理员用户名")
    p.add_argument("--password", default=os.getenv("ADMIN_PASS", ""), help="管理员密码（留空则交互输入）")
    p.add_argument("--target-user", default="admin", help="目标用户")
    return p.parse_args()


def load_json(resp):
    raw = resp.read().decode("utf-8")
    return json.loads(raw) if raw else {}


def main():
    args = parse_args()
    base = normalize_base(args.base)
    password = args.password or getpass.getpass("管理员密码: ")

    print("=" * 60)
    print("验证 readonly 管理员写操作拒绝")
    print("=" * 60)

    cj = http.cookiejar.CookieJar()
    op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))

    print("\n[1] 登录管理员...")
    login_req = urllib.request.Request(
        base + "/api/admin/login",
        data=json.dumps({"username": args.username, "password": password}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    try:
        with op.open(login_req, timeout=8) as resp:
            login_data = load_json(resp)
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8")
        print(f"✗ 登录失败: HTTP {e.code} {body}")
        return 1

    csrf = str(login_data.get("csrf_token", "")).strip()
    print(f"✓ 登录成功: role={login_data.get('role')}")
    print("✓ 会话已建立")

    print("\n[2] 发起写请求（应被拒绝）...")
    enable_req = urllib.request.Request(
        base + f"/api/admin/users/{args.target_user}/enable",
        headers={"X-CSRF-Token": csrf},
        method="POST",
    )
    try:
        with op.open(enable_req, timeout=8) as resp:
            data = load_json(resp)
            print(f"✗ 未预期成功: {data}")
            return 2
    except urllib.error.HTTPError as e:
        body_text = e.read().decode("utf-8")
        try:
            body = json.loads(body_text)
        except Exception:
            body = {"raw": body_text}

        reason = body.get("reason", "")
        if e.code == 403 and reason == "readonly_cannot_write":
            print("✓ 验证通过: readonly 管理员无法执行写操作")
            return 0
        if e.code == 403 and reason == "csrf_check_failed":
            print("✗ 请求被 CSRF 拒绝，未进入权限判断")
            return 3
        print(f"✗ 非预期响应: HTTP {e.code} {body}")
        return 4


if __name__ == "__main__":
    sys.exit(main())
