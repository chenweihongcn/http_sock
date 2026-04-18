#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""验证 expire_filter 查询结果。"""

import argparse
import getpass
import json
import os
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
    p = argparse.ArgumentParser(description="expire_filter 功能验证")
    p.add_argument("--base", default=os.getenv("ADMIN_BASE", "http://127.0.0.1:8088"), help="管理地址")
    p.add_argument("--username", default=os.getenv("ADMIN_USER", "admin"), help="管理员用户名")
    p.add_argument("--password", default=os.getenv("ADMIN_PASS", ""), help="管理员密码（留空则交互输入）")
    p.add_argument("--limit", type=int, default=200, help="每次查询 limit")
    return p.parse_args()


def main():
    args = parse_args()
    base = normalize_base(args.base)
    password = args.password or getpass.getpass("管理员密码: ")

    cj = http.cookiejar.CookieJar()
    op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))

    req = urllib.request.Request(
        base + "/api/admin/login",
        data=json.dumps({"username": args.username, "password": password}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    op.open(req).read()

    for f in ["", "permanent", "expired", "expiring7"]:
        url = base + f"/api/admin/users?offset=0&limit={args.limit}" + (("&expire_filter=" + f) if f else "")
        data = json.loads(op.open(url).read())
        print((f or "all"), data.get("total"), len(data.get("items", [])))


if __name__ == "__main__":
    main()
