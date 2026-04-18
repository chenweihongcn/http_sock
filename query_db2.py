#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""通过 Docker exec 查询数据库"""

import paramiko
import os

SERVER = "192.168.50.94"
SSH_USER = "root"
SSH_PASSWORD = "ckp800810"
CONTAINER = "lowres-proxy"
DB_PATH = "/app/data/proxy.db"

print("="*60)
print("通过 Docker 容器查询数据库")
print("="*60)

# 建立 SSH 连接
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    client.connect(SERVER, username=SSH_USER, password=SSH_PASSWORD, timeout=30)
    print(f"✓ SSH 连接成功\n")
    
    # 使用 docker exec 在容器内执行查询
    cmd = f"""docker exec {CONTAINER} /app/proxy-server -db-query "SELECT username, role, password_set, length(password_hash) as hash_len FROM admins ORDER BY username;" """
    
    print(f"执行: docker exec {CONTAINER} [查询管理员表]\n")
    stdin, stdout, stderr = client.exec_command(cmd)
    output = stdout.read().decode('utf-8')
    error = stderr.read().decode('utf-8')
    
    if output:
        print("查询结果:")
        print(output)
    
    if error:
        print(f"错误: {error}")
    
    # 如果 -db-query 不支持，尝试直接用 SQL 连接
    print("\n" + "="*60)
    print("直接 HTTP API 验证:")
    print("="*60)
    
    # 检查健康状态
    curl_cmd = "docker exec lowres-proxy sh -c 'curl -s http://localhost:8088/api/admin/healthz'"
    stdin, stdout, stderr = client.exec_command(curl_cmd)
    output = stdout.read().decode('utf-8')
    print(f"Healthz: {output.strip()}")
    
finally:
    client.close()

print("\n" + "="*60)
