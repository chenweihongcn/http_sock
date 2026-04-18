#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""直接通过 SSH 查询数据库中的管理员账号"""

import paramiko

SERVER = "192.168.50.94"
SSH_USER = "root"
SSH_PASSWORD = "ckp800810"
CONTAINER = "lowres-proxy"

print("="*60)
print("查询数据库中的管理员账号")
print("="*60)

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    client.connect(SERVER, username=SSH_USER, password=SSH_PASSWORD, timeout=30)
    print("✓ SSH 连接成功\n")
    
    # 进入容器并查询管理员表
    # Note: 容器内应该有 sqlite3 或者可以通过应用查询
    
    # 先检查容器内是否有 sqlite3
    print("[1] 检查容器内的工具:")
    stdin, stdout, stderr = client.exec_command("docker exec lowres-proxy which sqlite3")
    output = stdout.read().decode('utf-8').strip()
    if output:
        print(f"  ✓ 找到 sqlite3: {output}")
        has_sqlite = True
    else:
        print("  ✗ 容器内没有 sqlite3")
        has_sqlite = False
    
    if has_sqlite:
        # 查询管理员表
        print("\n[2] 查询管理员表:")
        sql = "SELECT username, role, LENGTH(password_hash) as hash_len, password_hash IS NOT NULL as has_hash FROM admins ORDER BY username;"
        cmd = f"docker exec lowres-proxy sqlite3 /app/data/proxy.db \"{sql}\""
        
        stdin, stdout, stderr = client.exec_command(cmd)
        output = stdout.read().decode('utf-8')
        error = stderr.read().decode('utf-8')
        
        if output:
            print(f"  {output}")
        if error:
            print(f"  错误: {error}")
    
    # 检查正在运行的应用程序的启动是否成功
    print("\n[3] 检查应用启动日志:")
    stdin, stdout, stderr = client.exec_command("docker logs lowres-proxy 2>&1 | grep -i 'bootstrap\\|admin\\|error' || echo '(无相关日志)'")
    output = stdout.read().decode('utf-8')
    print(f"  {output}")

finally:
    client.close()

print("\n" + "="*60)
