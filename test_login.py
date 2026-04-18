#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""测试管理员登陆"""

import urllib.request
import urllib.error
import json
import sys

SERVER = "192.168.50.94:8088"
URL = f"http://{SERVER}/api/admin/login"

# 测试数据
credentials = [
    {"username": "admin", "password": "Admin2026Strong9X"},
    {"username": "ops", "password": "OpsPass123"},
    {"username": "admin", "password": "wrongpassword"},  # 测试失败情况
]

print("="*60)
print("管理员登陆测试")
print("="*60)

for cred in credentials:
    print(f"\n测试: {cred['username']} / {cred['password']}")
    try:
        data = json.dumps(cred).encode('utf-8')
        req = urllib.request.Request(
            URL,
            data=data,
            headers={'Content-Type': 'application/json'},
            method='POST'
        )
        
        with urllib.request.urlopen(req, timeout=5) as response:
            status_code = response.status
            headers = dict(response.headers)
            body = response.read().decode('utf-8')
            
            print(f"状态码: {status_code}")
            print(f"响应体: {body[:200]}")
            
            if status_code == 200:
                print("✓ 登陆成功")
                # 检查 Set-Cookie
                if 'Set-Cookie' in headers:
                    print(f"✓ Cookie 已设置: {headers['Set-Cookie'][:50]}...")
            else:
                print(f"✗ 登陆失败 (HTTP {status_code})")
    except urllib.error.HTTPError as e:
        print(f"HTTP 错误 {e.code}: {e.reason}")
        try:
            error_body = e.read().decode('utf-8')
            print(f"错误响应: {error_body[:200]}")
        except:
            pass
    except Exception as e:
        print(f"✗ 请求失败: {e}")

print("\n" + "="*60)
