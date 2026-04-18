#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""检查容器状态"""

import paramiko
import json

SERVER = "192.168.50.94"
SSH_USER = "root"
SSH_PASSWORD = "ckp800810"

print("="*60)
print("检查容器状态和启动日志")
print("="*60)

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    client.connect(SERVER, username=SSH_USER, password=SSH_PASSWORD, timeout=30)
    print("✓ SSH 连接成功\n")
    
    # 1. 检查容器状态
    print("[1] 容器状态:")
    stdin, stdout, stderr = client.exec_command("docker ps | grep lowres-proxy")
    output = stdout.read().decode('utf-8')
    if output:
        print(output)
    else:
        print("  ✗ 容器未找到，可能已停止或删除")
    
    # 2. 检查最近的日志
    print("\n[2] 最近 30 行启动日志:")
    stdin, stdout, stderr = client.exec_command("docker logs lowres-proxy 2>&1 | tail -30")
    output = stdout.read().decode('utf-8')
    if output:
        for line in output.split('\n'):
            if line.strip():
                print(f"  {line}")
    
    # 3. 检查数据库是否存在
    print("\n[3] 检查数据库文件:")
    stdin, stdout, stderr = client.exec_command("ls -lh /mnt/mmc0-4/docker/http_sock_data/proxy.db 2>&1")
    output = stdout.read().decode('utf-8')
    if output:
        print(f"  {output.strip()}")
    
    # 4. 检查容器内的 healthz
    print("\n[4] 健康检查:")
    stdin, stdout, stderr = client.exec_command('docker exec lowres-proxy sh -c "curl -s http://localhost:8088/api/admin/healthz" 2>&1')
    output = stdout.read().decode('utf-8')
    if output:
        try:
            data = json.loads(output)
            print(f"  ✓ 服务正常: {data}")
        except:
            print(f"  响应: {output}")
    
    # 5. 检查环境变量
    print("\n[5] 容器环境变量:")
    stdin, stdout, stderr = client.exec_command("docker inspect lowres-proxy --format='{{.Config.Env}}' 2>&1")
    output = stdout.read().decode('utf-8')
    if output:
        # 格式化输出
        env_str = output.strip().replace('[', '').replace(']', '')
        for item in env_str.split():
            if 'BOOTSTRAP' in item:
                print(f"  {item}")
    
finally:
    client.close()

print("\n" + "="*60)
