#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""测试用户操作按钮的完整流程"""

import urllib.request
import urllib.error
import urllib.parse
import json
import sys
import http.cookiejar

SERVER = "192.168.50.94:8088"

print("="*60)
print("测试管理员操作流程（完整模拟浏览器）")
print("="*60)

# 使用 CookieJar 来自动管理 Cookie
cookie_jar = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookie_jar))
urllib.request.install_opener(opener)

# 第一步：登陆
print("\n[1] 登陆 admin...")
login_url = f"http://{SERVER}/api/admin/login"
login_data = json.dumps({"username": "admin", "password": "Admin2026Strong9X"}).encode('utf-8')

try:
    req = urllib.request.Request(
        login_url,
        data=login_data,
        headers={'Content-Type': 'application/json'},
        method='POST'
    )
    with opener.open(req) as response:
        result = json.loads(response.read().decode('utf-8'))
        print(f"✓ 登陆成功: username={result.get('username')}, role={result.get('role')}")
except Exception as e:
    print(f"✗ 登陆失败: {e}")
    sys.exit(1)

# 第二步：获取用户列表
print("\n[2] 获取用户列表...")
list_url = f"http://{SERVER}/api/admin/users?offset=0&limit=20"

try:
    req = urllib.request.Request(list_url, method='GET')
    with opener.open(req) as response:
        result = json.loads(response.read().decode('utf-8'))
        users = result.get('items', [])
        print(f"✓ 获取用户列表成功: 共 {result.get('total')} 个用户")
        
        if users:
            print(f"\n  用户列表:")
            for user in users[:5]:
                print(f"    - {user.get('username')}: status={user.get('status')} (1=启用, 0=禁用)")
            
            # 选择第一个用户来测试禁用操作
            target_user = users[0]
            target_username = target_user.get('username')
            current_status = target_user.get('status')
            
            # 确定要执行的操作
            action = 'disable' if current_status == 1 else 'enable'
            print(f"\n  选定测试用户: {target_username}")
            print(f"  当前状态: {current_status} (1=启用, 0=禁用)")
            print(f"  将执行操作: {action}")
            
            # 第三步：执行禁用/启用操作
            print(f"\n[3] 执行 {action} 操作...")
            action_url = f"http://{SERVER}/api/admin/users/{urllib.parse.quote(target_username)}/{action}"
            
            try:
                req = urllib.request.Request(action_url, method='POST')
                with opener.open(req) as response:
                    result = json.loads(response.read().decode('utf-8'))
                    print(f"✓ {action} 操作成功: {result}")
                    
                    # 刷新用户列表，验证状态是否改变
                    print(f"\n[4] 刷新用户列表验证改变...")
                    req = urllib.request.Request(list_url, method='GET')
                    with opener.open(req) as response:
                        result = json.loads(response.read().decode('utf-8'))
                        updated_user = next((u for u in result.get('items', []) if u.get('username') == target_username), None)
                        if updated_user:
                            new_status = updated_user.get('status')
                            print(f"✓ 用户 {target_username} 的新状态: {new_status}")
                            if new_status != current_status:
                                print(f"✓✓ 状态已成功改变！")
                            else:
                                print(f"✗ 状态未改变（可能是其他问题）")
                        else:
                            print(f"✗ 未找到用户 {target_username}")
            except urllib.error.HTTPError as e:
                error_body = json.loads(e.read().decode('utf-8'))
                print(f"✗ {action} 操作失败 (HTTP {e.code}): {error_body.get('error')}")
            
except Exception as e:
    print(f"✗ 请求失败: {e}")
    sys.exit(1)

print("\n" + "="*60)
print("✓ 测试完成")
print("="*60)
