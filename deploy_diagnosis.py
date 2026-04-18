#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""为生产机生成一个带调试信息的迷你诊断页面"""

import paramiko

SERVER = "192.168.50.94"
SSH_USER = "root"
SSH_PASSWORD = "ckp800810"

# 生成诊断页面 HTML
diagnosis_html = '''<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <title>前端诊断工具</title>
  <style>
    body { font-family: monospace; background: #f5f5f5; margin: 20px; }
    .section { background: white; padding: 15px; margin: 10px 0; border-radius: 5px; border-left: 3px solid #1862f7; }
    button { padding: 8px 12px; margin: 5px; cursor: pointer; background: #1862f7; color: white; border: none; border-radius: 3px; }
    .result { background: #f0f0f0; padding: 10px; margin: 10px 0; border-radius: 3px; max-height: 300px; overflow-y: auto; }
    .success { color: green; }
    .error { color: red; }
    h2 { margin-top: 0; color: #1862f7; }
  </style>
</head>
<body>
  <h1>管理台前端诊断工具</h1>
  
  <div class="section">
    <h2>1️⃣ 当前状态检查</h2>
    <button onclick="checkStatus()">检查状态</button>
    <div class="result" id="statusResult"></div>
  </div>
  
  <div class="section">
    <h2>2️⃣ 测试 API 连接</h2>
    <button onclick="testAPIHealth()">测试健康检查</button>
    <div class="result" id="healthResult"></div>
  </div>
  
  <div class="section">
    <h2>3️⃣ 测试用户列表 API</h2>
    <button onclick="testUserList()">获取用户列表</button>
    <div class="result" id="userListResult"></div>
  </div>
  
  <div class="section">
    <h2>4️⃣ 测试按钮点击（模拟禁用操作）</h2>
    <p>用户名: <input id="testUsername" type="text" value="admin" style="width: 200px;"></p>
    <button onclick="testToggleUser()">测试禁用</button>
    <div class="result" id="toggleResult"></div>
  </div>
  
  <div class="section">
    <h2>5️⃣ 检查 JavaScript 控制台</h2>
    <button onclick="checkConsole()">查看错误日志</button>
    <div class="result" id="consoleResult"></div>
  </div>

  <script>
    // 记录所有错误
    const errors = [];
    window.addEventListener('error', function(e) {
      errors.push('[Error] ' + e.message + ' at ' + e.filename + ':' + e.lineno);
    });

    function $(id) { return document.getElementById(id); }
    function log(id, text, isError) {
      const el = $(id);
      el.innerHTML += (isError ? '<div class="error">' : '<div>') + text + '</div>';
    }

    async function checkStatus() {
      const result = $('statusResult');
      result.innerHTML = '';
      log('statusResult', '✓ 当前 URL: ' + window.location.href);
      log('statusResult', '✓ 当前文档加载状态: ' + document.readyState);
      
      // 检查是否有全局函数
      const functions = ['toggleUser', 'editUserDevices', 'resetUserUsage', 'deleteUser'];
      functions.forEach(fn => {
        const exists = typeof window[fn] === 'function';
        log('statusResult', (exists ? '✓' : '✗') + ' 函数 window.' + fn + ': ' + (exists ? '存在' : '不存在'), !exists);
      });
      
      // 检查 DOM 元素
      const elements = ['loginView', 'appView', 'usersTableWrap', 'userPageSize', 'userSearch'];
      elements.forEach(id => {
        const el = $(id);
        log('statusResult', (el ? '✓' : '✗') + ' DOM 元素 #' + id + ': ' + (el ? '找到' : '未找到'), !el);
      });
    }

    async function testAPIHealth() {
      const result = $('healthResult');
      result.innerHTML = '';
      try {
        const res = await fetch('/api/admin/healthz', { credentials: 'same-origin' });
        const data = await res.json();
        log('healthResult', '✓ API 健康检查成功: ' + JSON.stringify(data));
      } catch (err) {
        log('healthResult', '✗ API 健康检查失败: ' + err.message, true);
      }
    }

    async function testUserList() {
      const result = $('userListResult');
      result.innerHTML = '';
      try {
        const res = await fetch('/api/admin/users?offset=0&limit=20', { credentials: 'same-origin' });
        if (!res.ok) {
          log('userListResult', '✗ HTTP ' + res.status + ': ' + res.statusText, true);
          return;
        }
        const data = await res.json();
        log('userListResult', '✓ 用户列表获取成功');
        log('userListResult', '  总数: ' + data.total);
        log('userListResult', '  返回数: ' + data.items.length);
        if (data.items.length > 0) {
          log('userListResult', '  第一个用户: ' + JSON.stringify(data.items[0]));
        }
      } catch (err) {
        log('userListResult', '✗ 获取用户列表失败: ' + err.message, true);
      }
    }

    async function testToggleUser() {
      const result = $('toggleResult');
      result.innerHTML = '';
      const username = $('testUsername').value.trim();
      if (!username) {
        log('toggleResult', '✗ 请输入用户名', true);
        return;
      }
      
      try {
        log('toggleResult', '⏳ 正在禁用用户 ' + username + '...');
        const res = await fetch('/api/admin/users/' + encodeURIComponent(username) + '/disable', {
          method: 'POST',
          credentials: 'same-origin'
        });
        
        const text = await res.text();
        log('toggleResult', 'HTTP ' + res.status + ': ' + res.statusText);
        log('toggleResult', '响应体: ' + text);
        
        if (res.ok) {
          log('toggleResult', '✓ 禁用操作成功');
        } else {
          log('toggleResult', '✗ 禁用操作失败', true);
        }
      } catch (err) {
        log('toggleResult', '✗ 请求异常: ' + err.message, true);
      }
    }

    function checkConsole() {
      const result = $('consoleResult');
      result.innerHTML = '';
      if (errors.length === 0) {
        log('consoleResult', '✓ 没有检测到 JavaScript 错误');
      } else {
        log('consoleResult', '✗ 发现 ' + errors.length + ' 个错误:', true);
        errors.forEach(err => {
          log('consoleResult', err, true);
        });
      }
    }
  </script>
</body>
</html>'''

print("="*60)
print("部署诊断页面到生产机")
print("="*60)

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

try:
    client.connect(SERVER, username=SSH_USER, password=SSH_PASSWORD, timeout=30)
    print("✓ SSH 连接成功")
    
    # 将 HTML 写入容器内的文件
    diagnosis_path = "/app/static/diagnosis.html"
    
    # 创建 /app/static 目录
    stdin, stdout, stderr = client.exec_command("mkdir -p /app/static")
    stdout.read()
    
    # 通过 SSH 写入文件（使用 cat 和 EOF）
    sftp = client.open_sftp()
    
    with sftp.file(diagnosis_path, 'w') as f:
        f.write(diagnosis_html)
    
    print(f"✓ 诊断页面已写入: {diagnosis_path}")
    print(f"\n访问地址: http://{SERVER}:8088/static/diagnosis.html")
    
    sftp.close()
    
finally:
    client.close()

print("\n" + "="*60)
print("现在访问上面的 URL，用诊断工具测试前端问题")
print("="*60)
