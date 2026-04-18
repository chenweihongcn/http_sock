#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""查询数据库验证管理员账号状态"""

import paramiko
import os

SERVER = "192.168.50.94"
SSH_USER = "root"
SSH_PASSWORD = "ckp800810"
DB_PATH = "/mnt/mmc0-4/docker/http_sock_data/proxy.db"

print("="*60)
print("查询管理员账号数据库状态")
print("="*60)

# 建立 SSH 连接
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    client.connect(SERVER, username=SSH_USER, password=SSH_PASSWORD, timeout=30)
    print(f"✓ SSH 连接成功")
    
    # 查询 admins 表
    query = f"""sqlite3 {DB_PATH} "SELECT username, role, password_set, password_hash IS NOT NULL as has_hash FROM admins ORDER BY username;" """
    print(f"\n执行查询: {query}\n")
    
    stdin, stdout, stderr = client.exec_command(query)
    output = stdout.read().decode('utf-8')
    error = stderr.read().decode('utf-8')
    
    if output:
        print("查询结果:")
        print("-" * 60)
        print(output)
        print("-" * 60)
    
    if error:
        print(f"错误: {error}")
    
    # 尝试用 sqlite3 .schema 输出表结构
    print("\n" + "="*60)
    print("查询表结构:")
    print("="*60)
    schema_query = f"""sqlite3 {DB_PATH} ".schema admins" """
    stdin, stdout, stderr = client.exec_command(schema_query)
    schema = stdout.read().decode('utf-8')
    if schema:
        print(schema)
    
finally:
    client.close()

print("\n" + "="*60)
print("✓ 查询完成")
print("="*60)
