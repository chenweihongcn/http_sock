#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""修复 admin 账号 - 重启容器并清理数据库"""

import paramiko
import time

SERVER = "192.168.50.94"
SSH_USER = "root"
SSH_PASSWORD = "ckp800810"
CONTAINER = "lowres-proxy"
DB_PATH = "/mnt/mmc0-4/docker/http_sock_data/proxy.db"

print("="*60)
print("修复 admin 账号初始化")
print("="*60)

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    client.connect(SERVER, username=SSH_USER, password=SSH_PASSWORD, timeout=30)
    print("✓ SSH 连接成功\n")
    
    # 1. 停止容器
    print("[1] 停止容器...")
    stdin, stdout, stderr = client.exec_command(f"docker stop {CONTAINER}")
    output = stdout.read().decode('utf-8').strip()
    print(f"  {output}")
    time.sleep(2)
    
    # 2. 删除旧数据库（让它重新初始化）
    print("\n[2] 删除旧数据库文件...")
    stdin, stdout, stderr = client.exec_command(f"rm -f {DB_PATH} && ls -lh {DB_PATH} 2>&1 || echo '✓ 数据库已删除'")
    output = stdout.read().decode('utf-8').strip()
    print(f"  {output}")
    
    # 3. 重启容器
    print("\n[3] 重启容器...")
    stdin, stdout, stderr = client.exec_command(f"docker start {CONTAINER}")
    output = stdout.read().decode('utf-8').strip()
    print(f"  {output}")
    time.sleep(3)
    
    # 4. 检查容器状态
    print("\n[4] 检查容器状态...")
    stdin, stdout, stderr = client.exec_command(f"docker ps | grep {CONTAINER}")
    output = stdout.read().decode('utf-8').strip()
    if 'Up' in output:
        print("  ✓ 容器已启动")
    else:
        print("  ✗ 容器启动失败")
    
    # 5. 等待应用启动
    print("\n[5] 等待应用初始化...")
    time.sleep(3)
    
    # 6. 检查数据库文件
    print("\n[6] 检查新数据库文件...")
    stdin, stdout, stderr = client.exec_command(f"ls -lh {DB_PATH}")
    output = stdout.read().decode('utf-8').strip()
    print(f"  {output}")
    
    # 7. 检查最新日志
    print("\n[7] 最近启动日志:")
    stdin, stdout, stderr = client.exec_command(f"docker logs {CONTAINER} 2>&1 | tail -5")
    output = stdout.read().decode('utf-8')
    for line in output.split('\n'):
        if line.strip():
            print(f"  {line}")
    
finally:
    client.close()

print("\n" + "="*60)
print("✓ 容器重启完成")
print("="*60)
print("""
现在尝试重新登陆：
- admin / Admin2026Strong9X （super 管理员）
- ops / OpsPass123 （readonly 账号）

如果 admin 仍然无法登陆，说明 bootstrap 逻辑有问题。
""")
