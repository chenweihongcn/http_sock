#!/usr/bin/env python
# -*- coding: utf-8 -*-
import paramiko, json

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect("192.168.50.94", username="root", password="ckp800810", timeout=30)

# 检查审计日志 + 当前代理用户 + webhook 活动
cmds = {
    "audit_logs": "docker exec lowres-proxy sh -c \"cat /app/data/proxy.db | strings | grep -a 'action\\|actor\\|toggle\\|disable\\|enable' | tail -30\" 2>&1 || echo 'strings not available'",
    "access_log_tail": "docker logs --tail=40 lowres-proxy 2>&1",
    "check_users_table": "docker exec lowres-proxy sh -c \"ls -la /app/data/\" 2>&1",
}

for name, cmd in cmds.items():
    print(f"\n=== {name} ===")
    stdin, stdout, stderr = client.exec_command(cmd)
    out = stdout.read().decode('utf-8', errors='replace')
    err = stderr.read().decode('utf-8', errors='replace')
    if out.strip():
        print(out[:1500])
    if err.strip():
        print(f"[err] {err[:200]}")

client.close()
