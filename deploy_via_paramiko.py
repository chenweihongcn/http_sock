#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
部署脚本：通过 paramiko SSH 连接上传 Docker 镜像并启动容器
"""
import paramiko
import os
import sys
from pathlib import Path

# 配置
SERVER = "192.168.50.94"
SSH_USER = "root"
SSH_KEY = os.path.expanduser("~/.ssh/id_ed25519")
TAR_FILE = r"i:\HTTP\http_sock_arm64.tar"
REMOTE_TAR_PATH = "/mnt/mmc0-4/docker/http_sock_arm64.tar"
REMOTE_DATA_DIR = "/mnt/mmc0-4/docker/http_sock_data"

def upload_file(sftp, local_path, remote_path):
    """上传文件"""
    print(f"上传 {local_path} 到 {remote_path}...")
    sftp.put(local_path, remote_path)
    print("✓ 上传完成")

def run_command(ssh, command, print_output=True):
    """执行远程命令"""
    if print_output:
        print(f"\n> {command}")
    stdin, stdout, stderr = ssh.exec_command(command)
    output = stdout.read().decode('utf-8')
    error = stderr.read().decode('utf-8')
    
    if print_output:
        if output:
            print(output)
        if error:
            print(f"[ERR] {error}", file=sys.stderr)
    
    return output, error

def main():
    print("="*60)
    print("Docker 镜像部署脚本 (Paramiko SSH)")
    print("="*60)
    
    # 检查本地文件
    if not os.path.exists(TAR_FILE):
        print(f"ERROR: {TAR_FILE} 不存在")
        return False
    
    file_size = os.path.getsize(TAR_FILE)
    print(f"✓ 源文件存在: {TAR_FILE} ({file_size} bytes)")
    
    # 建立 SSH 连接
    print(f"\n连接到 {SERVER}...")
    client = paramiko.SSHClient()
    client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    
    try:
        # 尝试密钥认证
        client.connect(SERVER, username=SSH_USER, key_filename=SSH_KEY, timeout=30)
        print("✓ SSH 连接成功")
    except paramiko.AuthenticationException as e:
        print(f"✗ 密钥认证失败: {e}")
        print("尝试密码认证...")
        try:
            # 备选：密码认证
            password = input("输入 SSH 密码: ")
            client.connect(SERVER, username=SSH_USER, password=password, timeout=30)
            print("✓ SSH 密码认证成功")
        except Exception as e:
            print(f"✗ 连接失败: {e}")
            return False
    except Exception as e:
        print(f"✗ SSH 连接失败: {e}")
        return False
    
    # 建立 SFTP 连接
    try:
        sftp = client.open_sftp()
        print("✓ SFTP 连接成功")
    except Exception as e:
        print(f"✗ SFTP 连接失败: {e}")
        client.close()
        return False
    
    try:
        # 第一步：停止旧容器
        print("\n[步骤 1] 停止旧容器...")
        run_command(client, "docker stop lowres-proxy 2>/dev/null || true")
        run_command(client, "docker rm lowres-proxy 2>/dev/null || true")
        
        # 第二步：上传文件
        print("\n[步骤 2] 上传 Docker 镜像...")
        upload_file(sftp, TAR_FILE, REMOTE_TAR_PATH)
        
        # 第三步：验证上传
        print("\n[步骤 3] 验证上传文件...")
        cmd = f"ls -lh {REMOTE_TAR_PATH} && md5sum {REMOTE_TAR_PATH}"
        run_command(client, cmd)
        
        # 第四步：加载镜像
        print("\n[步骤 4] 加载 Docker 镜像...")
        run_command(client, f"docker rmi chenweihongcn/http_sock:arm64 2>/dev/null || true")
        run_command(client, f"docker load -i {REMOTE_TAR_PATH}")
        
        # 第五步：检查镜像
        print("\n[步骤 5] 检查镜像...")
        run_command(client, "docker images | grep http_sock")
        
        # 第六步：修复数据权限
        print("\n[步骤 6] 修复数据目录权限...")
        run_command(client, f"chmod 777 {REMOTE_DATA_DIR} 2>/dev/null || true")
        run_command(client, f"chmod 666 {REMOTE_DATA_DIR}/*.db 2>/dev/null || true")
        
        # 第七步：启动容器
        print("\n[步骤 7] 启动新容器...")
        docker_cmd = (
            "docker run -d --name lowres-proxy "
            "--restart=always "
            "-p 8899:8899 "
            "-p 1080:1080 "
            "-p 8088:8088 "
            f"-v {REMOTE_DATA_DIR}:/app/data "
            "-e BOOTSTRAP_ADMIN_PASS=Admin2026Strong9X "
            "-e BOOTSTRAP_READONLY=ops "
            "-e BOOTSTRAP_READONLY_PASS=OpsPass123 "
            "chenweihongcn/http_sock:arm64"
        )
        output, _ = run_command(client, docker_cmd)
        container_id = output.strip()
        print(f"✓ 容器启动成功: {container_id[:12]}")
        
        # 第八步：等待容器启动并检查状态
        print("\n[步骤 8] 检查容器状态...")
        import time
        time.sleep(3)
        run_command(client, "docker ps | grep lowres-proxy")
        
        # 第九步：检查 healthz 端点
        print("\n[步骤 9] 检查健康检查端点...")
        run_command(client, "curl -s http://localhost:8088/api/admin/healthz | head -n 1 || echo 'healthz endpoint not yet ready'")
        
        print("\n" + "="*60)
        print("✓ 部署完成！")
        print("="*60)
        print(f"\n访问管理界面: http://{SERVER}:8088/")
        print(f"默认管理员: admin / Admin2026Strong9X")
        print(f"只读账户: ops / OpsPass123")
        
        return True
        
    finally:
        sftp.close()
        client.close()

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
