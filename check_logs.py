#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""查看容器启动日志验证 bootstrap 初始化"""

import paramiko

SERVER = "192.168.50.94"
SSH_USER = "root"
SSH_PASSWORD = "ckp800810"
CONTAINER = "lowres-proxy"

print("="*60)
print("容器启动日志")
print("="*60)

# 建立 SSH 连接
client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    client.connect(SERVER, username=SSH_USER, password=SSH_PASSWORD, timeout=30)
    print(f"✓ SSH 连接成功\n")
    
    # 查看容器日志
    cmd = f"docker logs {CONTAINER}"
    print(f"执行: docker logs {CONTAINER}\n")
    
    stdin, stdout, stderr = client.exec_command(cmd)
    output = stdout.read().decode('utf-8')
    
    # 输出日志（最后 50 行）
    lines = output.strip().split('\n')
    for line in lines[-50:]:
        print(line)
    
finally:
    client.close()

print("\n" + "="*60)
