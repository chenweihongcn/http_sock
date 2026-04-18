#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""检查当前管理员身份和权限"""

import urllib.request
import urllib.error
import json
import sys

SERVER = "192.168.50.94:8088"

# 先以 admin 身份登陆
login_url = f"http://{SERVER}/api/admin/login"
login_data = json.dumps({"username": "admin", "password": "Admin2026Strong9X"}).encode('utf-8')

print("="*60)
print("1. 以 admin 身份登陆...")
print("="*60)

try:
    req = urllib.request.Request(
        login_url,
        data=login_data,
        headers={'Content-Type': 'application/json'},
        method='POST'
    )
    with urllib.request.urlopen(req, timeout=5) as response:
        response_data = json.loads(response.read().decode('utf-8'))
        print(f"✓ 登陆成功")
        print(f"  用户名: {response_data.get('username')}")
        print(f"  角色: {response_data.get('role')}")
        
        # 获取 cookie
        cookies = response.headers.get('Set-Cookie')
        print(f"  Cookie: {cookies[:50]}..." if cookies else "  未设置 Cookie")
        
except Exception as e:
    print(f"✗ 登陆失败: {e}")
    sys.exit(1)

print("\n" + "="*60)
print("2. 查询当前会话和权限...")
print("="*60)

# 现在用浏览器 cookie 方式查询（模拟浏览器会话）
# 注：因为没有实际的 cookie jar，我们测试一个需要权限的操作

# 测试创建用户（需要 super 权限）
create_user_url = f"http://{SERVER}/api/admin/users"
create_user_data = json.dumps({
    "username": "test_user_" + str(int(__import__('time').time()) % 10000),
    "password": "TestPass12345",
    "expires_at": 0,
    "quota_bytes": 0,
    "max_devices": 1
}).encode('utf-8')

print("\n3. 测试创建用户（需要 super 权限）...")

# 注意：这里我们无法传递 cookie，所以测试只是为了看 API 是否存在
print("  (跳过此测试，因为需要实际的浏览器会话)")

print("\n" + "="*60)
print("4. 列出现有用户...")
print("="*60)

# 测试列出用户（只读操作）
list_users_url = f"http://{SERVER}/api/admin/users?offset=0&limit=20"

try:
    req = urllib.request.Request(
        list_users_url,
        method='GET'
    )
    with urllib.request.urlopen(req, timeout=5) as response:
        result = json.loads(response.read().decode('utf-8'))
        print(f"✓ 获取用户列表成功")
        print(f"  总数: {result.get('total', 0)}")
        print(f"  项数: {len(result.get('items', []))}")
        
        if result.get('items'):
            print(f"\n  用户列表:")
            for user in result.get('items', [])[:3]:
                print(f"    - {user.get('username')}: 状态={user.get('status')} (1=启用, 0=禁用)")
        
except urllib.error.HTTPError as e:
    print(f"✗ 获取用户列表失败 (HTTP {e.code})")
    try:
        error_body = json.loads(e.read().decode('utf-8'))
        print(f"  错误: {error_body.get('error')}")
    except:
        pass
except Exception as e:
    print(f"✗ 请求失败: {e}")

print("\n" + "="*60)
print("分析:")
print("="*60)
print("""
可能的问题：
1. 检查浏览器开发者工具 (F12) → Network 标签
   - 点击按钮后查看请求是否发送
   - 查看响应状态码（200 表示成功，403 表示权限不足，401 表示未登陆）

2. 如果返回 403 Forbidden，说明当前用户权限不足（可能是 readonly 角色）
   - 解决方案：用 super 管理员账号重新登陆

3. 如果请求根本没有发送，说明前端 JavaScript 有问题
   - 检查浏览器控制台有无错误信息

4. 检查是否允许修改操作的权限
   - admin 账号应该是 super 角色，应该有全部权限
   - ops 账号是 readonly 角色，只能查看
""")
