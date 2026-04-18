#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""重新启用 admin 用户"""

import urllib.request
import http.cookiejar
import json

SERVER = "192.168.50.94:8088"

cookie_jar = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookie_jar))

# 登陆
login_url = f"http://{SERVER}/api/admin/login"
login_data = json.dumps({"username": "admin", "password": "Admin2026Strong9X"}).encode('utf-8')

req = urllib.request.Request(login_url, data=login_data, headers={'Content-Type': 'application/json'}, method='POST')
with opener.open(req) as response:
    print("✓ 登陆成功")

# 重新启用 admin
enable_url = f"http://{SERVER}/api/admin/users/admin/enable"
req = urllib.request.Request(enable_url, method='POST')

try:
    with opener.open(req) as response:
        result = json.loads(response.read().decode('utf-8'))
        print(f"✓ admin 用户已启用: {result}")
except Exception as e:
    print(f"✗ 启用失败: {e}")
