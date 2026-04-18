package control

import "net/http"

const adminUIVersion = "2026-04-04 10:20"

const adminLoginHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>代理管理台登录</title>
  <style>
    :root {
      --bg: #eef2f7;
      --panel: rgba(255, 255, 255, 0.96);
      --line: #d3dbe7;
      --text: #172235;
      --muted: #5f6f86;
      --primary: #0b5fff;
      --danger: #cc3b3b;
      --shadow: 0 14px 36px rgba(33, 56, 94, 0.09);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      padding: 24px;
      font-family: "Bahnschrift", "DIN Alternate", "PingFang SC", "Microsoft YaHei", sans-serif;
      background:
        radial-gradient(circle at top left, rgba(11, 95, 255, 0.1), transparent 30%),
        linear-gradient(180deg, #e8edf5 0, var(--bg) 320px);
      color: var(--text);
    }
    .auth-card {
      width: min(460px, 100%);
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 26px;
      box-shadow: var(--shadow);
      backdrop-filter: blur(12px);
    }
    .badge {
      display: inline-flex;
      align-items: center;
      padding: 5px 10px;
      border-radius: 999px;
      background: #eef4ff;
      color: var(--primary);
      font-size: 12px;
      margin-bottom: 14px;
    }
    h1 { margin: 0 0 8px; font-size: 28px; }
    p { color: var(--muted); margin: 0 0 18px; line-height: 1.6; }
    .field { display: grid; gap: 6px; margin-bottom: 12px; }
    .field label { font-size: 13px; color: var(--muted); }
    .field input {
      width: 100%;
      border-radius: 14px;
      border: 1px solid var(--line);
      padding: 11px 13px;
      background: rgba(255, 255, 255, 0.98);
      color: var(--text);
      font-size: 14px;
    }
    button {
      border: 0;
      border-radius: 14px;
      padding: 10px 14px;
      font-size: 14px;
      cursor: pointer;
      background: var(--primary);
      color: #fff;
    }
    .status { min-height: 20px; font-size: 13px; color: var(--muted); margin-top: 12px; }
    .status.error { color: var(--danger); }
    .status.success { color: #137847; }
    .version {
      margin-top: 16px;
      font-size: 12px;
      color: var(--muted);
    }
  </style>
</head>
<body>
  <div class="auth-card">
    <div class="badge">安全登录</div>
    <h1>管理员登录</h1>
    <p>未登录时不展示管理台内容。登录成功后会建立浏览器会话。</p>
    <div class="field">
      <label for="loginUsername">账号</label>
      <input id="loginUsername" type="text" placeholder="输入管理员账号" autocomplete="username">
    </div>
    <div class="field">
      <label for="loginPassword">密码</label>
      <input id="loginPassword" type="password" placeholder="输入管理员密码" autocomplete="current-password">
    </div>
    <button id="loginButton">登录</button>
    <div class="status" id="loginStatus"></div>
    <div class="version">UI 版本：` + adminUIVersion + `</div>
  </div>

  <script>
    function $(id) { return document.getElementById(id); }
    function setStatus(text, kind) {
      var el = $('loginStatus');
      el.textContent = text || '';
      el.className = 'status' + (kind ? ' ' + kind : '');
    }
    async function login() {
      var username = $('loginUsername').value.trim();
      var password = $('loginPassword').value.trim();
      if (!username || !password) {
        setStatus('账号和密码不能为空', 'error');
        return;
      }
      setStatus('正在登录...');
      try {
        var res = await fetch('/api/admin/login', {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username: username, password: password })
        });
        var text = await res.text();
        var data = {};
        try { data = text ? JSON.parse(text) : {}; } catch (_) { data = {}; }
        if (!res.ok) {
          throw new Error(data.error || data.message || '登录失败');
        }
        setStatus('登录成功，正在进入管理台...', 'success');
        window.location.reload();
      } catch (err) {
        setStatus('登录失败：' + (err && err.message ? err.message : '未知错误'), 'error');
      }
    }
    $('loginButton').addEventListener('click', login);
    $('loginPassword').addEventListener('keydown', function(event) {
      if (event.key === 'Enter') login();
    });
  </script>
</body>
</html>`

const adminUIHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>代理管理台</title>
  <style>
    :root {
      --bg: #eef2f7;
      --panel: rgba(255, 255, 255, 0.96);
      --line: #d3dbe7;
      --text: #172235;
      --muted: #5f6f86;
      --primary: #0b5fff;
      --primary-soft: #e7efff;
      --danger: #cc3b3b;
      --danger-soft: #fff0f0;
      --success: #137847;
      --warn: #9a6400;
      --warn-soft: #fff7e6;
      --shadow: 0 14px 36px rgba(33, 56, 94, 0.09);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Bahnschrift", "DIN Alternate", "PingFang SC", "Microsoft YaHei", sans-serif;
      background:
        radial-gradient(circle at top left, rgba(11, 95, 255, 0.1), transparent 30%),
        linear-gradient(180deg, #e8edf5 0, var(--bg) 320px);
      color: var(--text);
    }
        .sysbar {
          display: flex;
          flex-wrap: wrap;
          align-items: center;
          justify-content: space-between;
          gap: 10px;
          background: #ffffff;
          border: 1px solid var(--line);
          border-radius: 16px;
          padding: 10px 12px;
          margin-bottom: 12px;
          box-shadow: var(--shadow);
        }
        .sysbar-left, .sysbar-right {
          display: flex;
          align-items: center;
          gap: 8px;
          flex-wrap: wrap;
        }
        .chip {
          display: inline-flex;
          align-items: center;
          border: 1px solid var(--line);
          border-radius: 999px;
          background: #f8fbff;
          color: var(--muted);
          padding: 5px 10px;
          font-size: 12px;
          line-height: 1;
        }
        .chip.ok { background: #edf9f2; color: var(--success); border-color: #bfe5cc; }
        .chip.warn { background: var(--warn-soft); color: var(--warn); border-color: #f0d8a3; }
        .chip.busy { background: var(--primary-soft); color: var(--primary); border-color: #bcd0ff; }
        .global-notice {
          display: none;
          margin: 0 0 12px;
          border-radius: 14px;
          border: 1px solid #f3b7b7;
          background: #fff4f4;
          color: #a32828;
          padding: 10px 12px;
          font-size: 13px;
        }
        .global-notice.show { display: block; }
        body.simple .overview-grid { display: none; }
        body.simple .workspace-grid { grid-template-columns: minmax(0, 1fr); }
        body.simple .stack { display: none; }
        body.simple .hero p { display: none; }
        body.simple .tabs [data-tab="modulesView"] { display: none; }
        body.simple .panel { padding: 14px; border-radius: 16px; }
        body.simple .module-grid { grid-template-columns: 1fr; }
    .wrap { max-width: 1380px; margin: 0 auto; padding: 24px; }
    .hero {
      background: linear-gradient(135deg, #175df1, #5ca8ff);
      color: #fff;
      border-radius: 22px;
      padding: 24px 26px;
      box-shadow: 0 22px 56px rgba(23, 93, 241, 0.22);
      margin-bottom: 18px;
      position: relative;
    }
    .hero h1 { margin: 0 0 8px; font-size: 30px; }
    .hero p { margin: 0; opacity: .92; }
    .hero-version {
      position: absolute;
      top: 18px;
      right: 18px;
      padding: 6px 10px;
      border-radius: 999px;
      background: rgba(255,255,255,.16);
      border: 1px solid rgba(255,255,255,.24);
      color: rgba(255,255,255,.94);
      font-size: 12px;
      backdrop-filter: blur(10px);
    }
    .auth-shell {
      min-height: calc(100vh - 150px);
      display: grid;
      place-items: center;
    }
    .auth-card {
      width: min(460px, 100%);
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 24px;
      padding: 26px;
      box-shadow: var(--shadow);
      backdrop-filter: blur(12px);
    }
    .auth-card h2 { margin: 0 0 10px; }
    .auth-card p { color: var(--muted); margin: 0 0 18px; }
    .overview-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 14px;
      margin-bottom: 16px;
      align-items: stretch;
    }
    .workspace-grid {
      display: grid;
      grid-template-columns: minmax(0, 1fr) 360px;
      gap: 18px;
      align-items: start;
    }
    .panel {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 22px;
      padding: 18px;
      box-shadow: var(--shadow);
      backdrop-filter: blur(12px);
    }
    .panel h2, .panel h3 { margin: 0 0 12px; }
    .stack { display: grid; gap: 14px; }
    .ops-shell {
      display: grid;
      gap: 12px;
    }
    .ops-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 8px;
    }
    .ops-header h3 {
      margin: 0;
      font-size: 18px;
    }
    .ops-subtitle {
      margin: 4px 0 0;
      color: var(--muted);
      font-size: 12px;
      line-height: 1.5;
    }
    .ops-tabs {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 8px;
    }
    .op-tab {
      border: 1px solid var(--line);
      background: #f7f9fe;
      color: var(--muted);
      border-radius: 12px;
      padding: 9px 8px;
      font-size: 13px;
      text-align: center;
    }
    .op-tab.active {
      background: var(--primary-soft);
      color: var(--primary);
      border-color: #c9daff;
      font-weight: 600;
    }
    .ops-body {
      border: 1px dashed #d6e2fb;
      border-radius: 16px;
      background: #fbfdff;
      padding: 14px;
    }
    .op-pane { display: none; }
    .op-pane.active { display: block; }
    .op-pane .actions { margin-top: 4px; }
    .op-pane .status { margin-top: 8px; }
    .field { display: grid; gap: 6px; margin-bottom: 12px; }
    .field label { font-size: 13px; color: var(--muted); }
    .field input, .field select, textarea {
      width: 100%;
      border-radius: 14px;
      border: 1px solid var(--line);
      padding: 11px 13px;
      background: rgba(255, 255, 255, 0.98);
      color: var(--text);
      font-size: 14px;
    }
    .row { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
    .row-3 { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
    .row-4 { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
    .row-5 { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 10px; }
    .actions, .toolbar, .tabs { display: flex; flex-wrap: wrap; gap: 8px; }
    button {
      border: 0;
      border-radius: 14px;
      padding: 10px 14px;
      font-size: 14px;
      cursor: pointer;
      background: var(--primary);
      color: #fff;
    }
    button.secondary { background: var(--primary-soft); color: var(--primary); }
    button.ghost { background: #f3f6fb; color: var(--text); }
    button.danger { background: var(--danger-soft); color: var(--danger); }
    button:disabled { opacity: .55; cursor: not-allowed; }
    .tabs { margin-bottom: 14px; }
    .tab {
      border: 1px solid var(--line);
      background: rgba(255,255,255,.86);
      color: var(--muted);
      border-radius: 999px;
    }
    .tab.active { background: var(--primary); color: #fff; border-color: var(--primary); }
    .view { display: none; }
    .view.active { display: block; }
    .module-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 12px;
    }
    .module-card {
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 14px;
      background: #f8fbff;
    }
    .module-card h4 {
      margin: 0 0 8px;
      font-size: 15px;
    }
    .module-card p {
      margin: 0 0 12px;
      color: var(--muted);
      font-size: 12px;
      line-height: 1.6;
    }
    .module-stats {
      display: grid;
      grid-template-columns: repeat(5, minmax(0, 1fr));
      gap: 10px;
      margin-bottom: 12px;
    }
    .module-stat {
      border: 1px solid var(--line);
      border-radius: 12px;
      background: #fff;
      padding: 10px;
      font-size: 12px;
      color: var(--muted);
    }
    .module-stat b {
      display: block;
      margin-bottom: 4px;
      font-size: 20px;
      color: var(--text);
    }
    .toolbar {
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;
    }
    .toolbar .group { display: flex; flex-wrap: wrap; gap: 8px; }
    .status { min-height: 20px; font-size: 13px; color: var(--muted); }
    .status.success { color: var(--success); }
    .status.error { color: var(--danger); }
    .summary {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 10px;
      margin-bottom: 14px;
    }
    .metric {
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 12px;
      background: #f8fbff;
    }
    .metric b { display: block; font-size: 22px; margin-bottom: 4px; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td {
      text-align: left;
      padding: 10px 8px;
      border-bottom: 1px solid #edf1f6;
      vertical-align: top;
    }
    th { color: var(--muted); font-weight: 700; background: #f6f9ff; position: sticky; top: 0; z-index: 1; }
    .mono { font-family: Consolas, "SFMono-Regular", monospace; word-break: break-all; }
    .note { color: var(--muted); font-size: 12px; line-height: 1.6; }
    .hidden { display: none !important; }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 5px 10px;
      border-radius: 999px;
      background: #eef4ff;
      color: var(--primary);
      font-size: 12px;
    }
    .danger-text { color: var(--danger); }
    .detail-card {
      margin-top: 14px;
      border: 1px solid var(--line);
      border-radius: 18px;
      padding: 16px;
      background: #fbfdff;
    }
    .user-op {
      display: grid;
      gap: 6px;
      min-width: 280px;
    }
    .user-op-main {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
    }
    .user-op-more {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
      border: 1px dashed var(--line);
      border-radius: 12px;
      padding: 8px;
      background: #f8fbff;
    }
    .user-op button {
      padding: 6px 10px;
      font-size: 12px;
      border-radius: 10px;
    }
    .health-list {
      margin: 10px 0 0;
      padding: 0;
      list-style: none;
      display: grid;
      gap: 8px;
    }
    .health-list li {
      border: 1px solid var(--line);
      border-radius: 14px;
      padding: 8px 10px;
      background: #f8fbff;
      font-size: 12px;
      color: var(--muted);
    }
    .health-list b { color: var(--text); }
    .pager {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-top: 12px;
    }
    @media (max-width: 1180px) {
      .overview-grid { grid-template-columns: 1fr; }
      .workspace-grid { grid-template-columns: 1fr; }
    }
    @media (max-width: 720px) {
      .row, .row-3, .row-4, .row-5, .summary, .module-grid, .module-stats { grid-template-columns: 1fr; }
      .wrap { padding: 14px; }
    }
    /* Toast */
    #toastArea {
      position: fixed; top: 18px; right: 18px; z-index: 9999;
      display: flex; flex-direction: column; gap: 8px;
    }
    .toast {
      padding: 11px 18px; border-radius: 14px; font-size: 13px; font-weight: 500;
      box-shadow: 0 6px 24px rgba(0,0,0,.12); animation: fadeInUp .2s ease;
      max-width: 340px; word-break: break-word;
    }
    .toast.success { background: #edfaf2; color: var(--success); border: 1px solid #b6e8c9; }
    .toast.error   { background: var(--danger-soft); color: var(--danger); border: 1px solid #f7cbcb; }
    .toast.info    { background: #eaf1ff; color: var(--primary); border: 1px solid #c5d9ff; }
    @keyframes fadeInUp { from { opacity:0; transform:translateY(-8px); } to { opacity:1; transform:none; } }
    /* Modal */
    .modal-overlay {
      position: fixed; inset: 0; background: rgba(0,0,0,.35); z-index: 8888;
      display: flex; align-items: center; justify-content: center;
    }
    .modal-box {
      background: #fff; border-radius: 22px; padding: 28px 26px;
      width: min(420px, 94vw); box-shadow: 0 24px 64px rgba(43,72,124,.18);
    }
    .modal-box h3 { margin: 0 0 10px; font-size: 17px; }
    .modal-box p  { margin: 0 0 18px; color: var(--muted); font-size: 14px; }
    .modal-box input { margin-bottom: 16px; }
    .modal-actions { display: flex; gap: 10px; justify-content: flex-end; }
  </style>
</head>
<body>
  <div id="toastArea"></div>
  <div class="wrap">
    <div id="authedShell" class="hidden">
    <div class="sysbar">
      <div class="sysbar-left">
        <span class="chip ok" id="chipServiceState">服务在线</span>
        <span class="chip" id="chipDataState">数据已同步</span>
        <span class="chip" id="chipLastRefresh">最近刷新：-</span>
      </div>
      <div class="sysbar-right">
        <button class="ghost" id="toggleSimplifyBtn">切换简洁模式</button>
        <span class="chip" id="chipViewMode">视图：简洁</span>
        <span class="chip" id="chipEnv">模式：控制平面</span>
        <span class="chip" id="chipNetState">网络空闲</span>
      </div>
    </div>
    <div class="global-notice" id="globalNotice"></div>
    <div class="hero">
      <div class="hero-version">UI 版本：` + adminUIVersion + `</div>
      <h1>HTTP + SOCKS5 管理台</h1>
      <p>管理员使用账号密码登录。支持用户搜索分页、批量启停、批量删除、延期、配额充值、改密、删号、流量重置、活跃设备查看与踢出。</p>
    </div>
    </div>

    <div id="loginView" class="auth-shell hidden">
      <div class="auth-card">
        <h2>管理员登录</h2>
        <p>登录成功后会建立浏览器会话，不再使用 Token。</p>
        <div class="field">
          <label for="loginUsername">账号</label>
          <input id="loginUsername" type="text" placeholder="输入管理员账号">
        </div>
        <div class="field">
          <label for="loginPassword">密码</label>
          <input id="loginPassword" type="password" placeholder="输入管理员密码">
        </div>
        <div class="actions">
          <button id="loginButton">登录</button>
        </div>
        <div class="status" id="loginStatus"></div>
      </div>
    </div>

    <div id="appView" class="hidden">
      <div class="overview-grid">
        <div class="panel">
          <h2>当前管理员与来源</h2>
          <div class="summary">
            <div class="metric"><b id="meName">-</b><span>账号</span></div>
            <div class="metric"><b id="meRole">-</b><span>角色</span></div>
            <div class="metric"><b id="meSessionCount">0</b><span>会话数</span></div>
          </div>
          <div class="summary">
            <div class="metric"><b id="meClientIP" class="mono">-</b><span>客户端IP</span></div>
            <div class="metric"><b id="meRemoteIP" class="mono">-</b><span>网关/直连IP</span></div>
            <div class="metric"><b id="meIPMode">-</b><span>IP识别方式</span></div>
          </div>
          <div class="actions">
            <button class="secondary" id="refreshProfileBtn">刷新身份</button>
            <button class="danger" id="logoutBtn">退出登录</button>
          </div>
        </div>

        <div class="panel">
          <h3>到期提醒与巡检</h3>
          <div class="summary">
            <div class="metric"><b id="expiredCount">0</b><span>已过期</span></div>
            <div class="metric"><b id="expiring7Count">0</b><span>7天内到期</span></div>
            <div class="metric"><b id="nearQuotaCount">0</b><span>流量告警</span></div>
          </div>
          <div class="actions">
            <button class="ghost" id="filterExpiredBtn">看已过期</button>
            <button class="ghost" id="filterExpiring7Btn">看7天内到期</button>
            <button class="ghost" id="filterPermanentBtn">看永久</button>
            <button class="secondary" id="refreshHealthBtn">巡检</button>
          </div>
          <ul class="health-list" id="healthList"></ul>
          <div class="note" id="healthLastScan">最近巡检：-</div>
        </div>
      </div>

      <div class="workspace-grid">
        <div class="panel">
          <div class="tabs">
            <button class="tab" data-tab="modulesView">模块面板</button>
            <button class="tab active" data-tab="usersView">用户</button>
            <button class="tab" data-tab="adminsView">管理员</button>
            <button class="tab" data-tab="sessionsView">会话</button>
            <button class="tab" data-tab="auditsView">审计</button>
          </div>

          <div id="modulesView" class="view">
            <div class="toolbar">
              <div class="group"><span class="badge">服务模块总览</span></div>
              <div class="group">
                <button class="secondary" id="moduleRefreshAllBtn">一键刷新</button>
                <button class="secondary" id="moduleRunHealthBtn">立即巡检</button>
              </div>
            </div>
            <div class="module-stats">
              <div class="module-stat"><b id="moduleUsersTotal">0</b><span>用户总数</span></div>
              <div class="module-stat"><b id="moduleSessionsTotal">0</b><span>在线会话</span></div>
              <div class="module-stat"><b id="moduleAuditsTotal">0</b><span>审计条数</span></div>
              <div class="module-stat"><b id="moduleExpiredTotal">0</b><span>已过期用户</span></div>
              <div class="module-stat"><b id="moduleBlockedActive">0</b><span>封禁中键数</span></div>
            </div>
            <div class="module-grid">
              <div class="module-card">
                <h4>用户运营</h4>
                <p>快速进入用户管理，支持搜索、批量启停、导入导出、延期与流量调整。</p>
                <div class="actions">
                  <button class="secondary" id="moduleGotoUsersBtn">打开用户列表</button>
                  <button class="ghost" id="moduleExportUsersBtn">导出用户 CSV</button>
                  <button class="ghost" id="moduleFocusCreateUserBtn">新建代理用户</button>
                </div>
              </div>
              <div class="module-card">
                <h4>账号与权限</h4>
                <p>快速跳转管理员管理，处理角色调整、禁启用和密码重置。</p>
                <div class="actions">
                  <button class="secondary" id="moduleGotoAdminsBtn">打开管理员模块</button>
                  <button class="ghost" id="moduleFocusCreateAdminBtn">新建管理员</button>
                  <button class="ghost" id="moduleFocusChangePwdBtn">修改我的密码</button>
                </div>
              </div>
              <div class="module-card">
                <h4>到期与巡检</h4>
                <p>按到期状态快速筛选用户，结合巡检项处理过期、流量告警与设备超限。</p>
                <div class="actions">
                  <button class="secondary" id="moduleFilterExpiredBtn">查看已过期</button>
                  <button class="ghost" id="moduleFilterExpiring7Btn">查看 7 天内到期</button>
                  <button class="ghost" id="moduleFilterPermanentBtn">查看永久用户</button>
                </div>
              </div>
              <div class="module-card">
                <h4>会话与审计</h4>
                <p>快速定位在线会话与关键操作日志，便于排查风控与变更问题。</p>
                <div class="actions">
                  <button class="secondary" id="moduleGotoSessionsBtn">打开会话模块</button>
                  <button class="ghost" id="moduleGotoAuditsBtn">打开审计模块</button>
                  <button class="danger" id="moduleLogoutBtn">退出登录</button>
                </div>
              </div>
            </div>
          </div>

          <div id="usersView" class="view active">
            <div class="toolbar">
              <div class="group row-5" style="flex:1; min-width:280px;">
                <input id="userSearch" type="text" placeholder="搜索用户名">
                <select id="userStatusFilter">
                  <option value="">全部状态</option>
                  <option value="1">仅启用</option>
                  <option value="0">仅禁用</option>
                </select>
                <select id="userTagFilter">
                  <option value="">全部标签</option>
                </select>
                <select id="userExpiryFilter">
                  <option value="">全部到期</option>
                  <option value="expiring7">7 天内到期</option>
                  <option value="expired">已过期</option>
                  <option value="permanent">永久</option>
                </select>
                <select id="userPageSize">
                  <option value="10">每页 10 条</option>
                  <option value="20" selected>每页 20 条</option>
                  <option value="50">每页 50 条</option>
                </select>
              </div>
              <div class="group">
                <button class="secondary" id="searchUsersBtn">搜索</button>
                <button class="secondary" id="resetUsersBtn">重置</button>
                <button class="secondary" id="refreshUsersBtn">刷新</button>
              </div>
            </div>
            <div class="actions" style="margin-bottom:10px;">
              <button class="secondary" id="batchEnableBtn">批量启用</button>
              <button class="secondary" id="batchDisableBtn">批量禁用</button>
              <button class="secondary" id="batchTagBtn">批量打标</button>
              <button class="secondary" id="batchExtendBtn">批量延期</button>
              <button class="secondary" id="batchTopupBtn">批量充值</button>
              <button class="danger" id="batchDeleteBtn">批量删除</button>
              <button class="ghost" id="clearSelectedBtn">清空选择</button>
              <button class="ghost" id="exportUsersBtn">导出 CSV</button>
              <button class="ghost" id="importUsersBtn">导入 CSV</button>
              <input id="importUsersFile" type="file" accept=".csv,text/csv" style="display:none;">
              <span class="badge">已选 <span id="selectedUserCount">0</span> 项</span>
            </div>
            <div class="note" id="userDetailHint">点击用户名或“详情/记录”可查看该用户的登录/使用记录、设备信息和管理员后台操作记录。</div>
            <div id="usersTableWrap"></div>
            <div class="pager">
              <button class="ghost" id="prevPageBtn">上一页</button>
              <div class="note" id="pageInfo">-</div>
              <button class="ghost" id="nextPageBtn">下一页</button>
            </div>
            <div id="userDetail" class="detail-card hidden"></div>
          </div>

          <div id="adminsView" class="view">
            <div class="toolbar">
              <div class="group"><span class="badge">平台账号管理</span></div>
              <div class="group"><button class="secondary" id="refreshAdminsBtn">刷新管理员</button></div>
            </div>
            <div id="adminsTableWrap"></div>
          </div>

          <div id="sessionsView" class="view">
            <div class="toolbar">
              <div class="group"><span class="badge" id="sessionsOwnerBadge">当前账号在线会话</span></div>
              <div class="group"><button class="secondary" id="refreshSessionsBtn">刷新会话</button></div>
            </div>
            <div id="sessionsTableWrap"></div>
          </div>

          <div id="auditsView" class="view">
            <div class="toolbar">
              <div class="group row-3" style="flex:1; min-width:280px;">
                <input id="auditActor" type="text" placeholder="操作者">
                <input id="auditAction" type="text" placeholder="动作">
                <input id="auditTarget" type="text" placeholder="目标">
              </div>
              <div class="group">
                <button class="ghost" id="auditPresetBlockedBtn">登录封禁</button>
                <button class="ghost" id="auditPresetLoginFailBtn">登录失败</button>
                <button class="ghost" id="auditPresetRateLimitBtn">命中限流</button>
                <button class="ghost" id="auditPresetClearBtn">清空筛选</button>
                <button class="secondary" id="refreshAuditsBtn">刷新审计</button>
              </div>
            </div>
            <div class="summary" style="margin-bottom:10px;">
              <div class="metric"><b id="securityRateLimitedTotal">0</b><span>管理 API 限流次数</span></div>
              <div class="metric"><b id="securityLoginFailedTotal">0</b><span>管理员登录失败次数</span></div>
              <div class="metric"><b id="securityLoginBlockedTotal">0</b><span>管理员登录封禁命中</span></div>
            </div>
            <div class="note" id="securityThresholdHint">告警阈值：限流 +3、登录失败 +5、登录封禁 +1；冷却 5 分钟</div>
            <div id="auditsTableWrap"></div>
          </div>
        </div>

        <div class="stack">
          <div class="panel ops-shell">
            <div class="ops-header">
              <div>
                <h3>操作中心</h3>
                <p class="ops-subtitle">在一个区域完成账号创建与密码维护，减少页面割裂和滚动成本。</p>
              </div>
              <span class="badge">快捷操作</span>
            </div>

            <div class="ops-tabs">
              <button class="op-tab active" data-op="user">代理用户</button>
              <button class="op-tab" data-op="admin">管理员</button>
              <button class="op-tab" data-op="password">修改密码</button>
            </div>

            <div class="ops-body">
              <div id="opPaneUser" class="op-pane active">
                <div class="field">
                  <label for="createUserName">用户名</label>
                  <input id="createUserName" type="text" placeholder="例如 vpn02">
                </div>
                <div class="field">
                  <label for="createUserPassword">密码</label>
                  <input id="createUserPassword" type="password" placeholder="至少 8 位">
                </div>
                <div class="row">
                  <div class="field">
                    <label for="createUserDevices">最大设备数</label>
                    <input id="createUserDevices" type="number" min="1" value="1">
                  </div>
                  <div class="field">
                    <label for="createUserDays">延长天数</label>
                    <input id="createUserDays" type="number" min="0" value="0">
                  </div>
                </div>
                <div class="field">
                  <label for="createUserQuota">流量上限 MB，0 表示不限</label>
                  <input id="createUserQuota" type="number" min="0" value="0">
                </div>
                <div class="field">
                  <label for="createUserSMBQuota">SMB 文件空间 MB，0 表示不限</label>
                  <input id="createUserSMBQuota" type="number" min="0" value="0">
                </div>
                <div class="field">
                  <label for="createUserSpeed">代理限速 KB/s，0 表示不限</label>
                  <input id="createUserSpeed" type="number" min="0" value="0">
                </div>
                <div class="field">
                  <label><input id="createUserSMBEnabled" type="checkbox"> 创建时开通 SMB 目录</label>
                </div>
                <div class="actions">
                  <button id="createUserBtn">创建用户</button>
                </div>
                <div class="status" id="createUserStatus"></div>
              </div>

              <div id="opPaneAdmin" class="op-pane">
                <div class="field">
                  <label for="createAdminName">管理员账号</label>
                  <input id="createAdminName" type="text" placeholder="例如 ops2">
                </div>
                <div class="field">
                  <label for="createAdminPassword">管理员密码</label>
                  <input id="createAdminPassword" type="password" placeholder="至少 8 位">
                </div>
                <div class="field">
                  <label for="createAdminRole">角色</label>
                  <select id="createAdminRole">
                    <option value="readonly">readonly</option>
                    <option value="super">super</option>
                  </select>
                </div>
                <div class="actions">
                  <button id="createAdminBtn">创建管理员</button>
                </div>
                <div class="status" id="createAdminStatus"></div>
              </div>

              <div id="opPanePassword" class="op-pane">
                <div class="field">
                  <label for="oldAdminPassword">旧密码</label>
                  <input id="oldAdminPassword" type="password">
                </div>
                <div class="field">
                  <label for="newAdminPassword">新密码</label>
                  <input id="newAdminPassword" type="password">
                </div>
                <div class="actions">
                  <button id="changeMyPasswordBtn">修改密码</button>
                </div>
                <div class="status" id="changeMyPasswordStatus"></div>
              </div>
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>

  <script>
    const state = {
      me: null,
      csrfToken: '',
      users: [],
      admins: [],
      sessions: [],
      audits: [],
      securityStats: null,
      lastSecuritySnapshot: null,
      lastSecurityAlertAt: 0,
      securityAlertThreshold: {
        rateLimitedDelta: 3,
        loginFailedDelta: 5,
        loginBlockedDelta: 1,
        cooldownMs: 5 * 60 * 1000
      },
      userTags: [],
      selectedUsers: new Set(),
      selectedUser: '',
      lastHealthSignature: '',
      healthReport: { expired: 0, expiring3: 0, expiring7: 0, nearQuota: 0, overDevices: 0 },
      offset: 0,
      limit: 20,
      total: 0,
      pendingRequests: 0,
      simpleMode: true,
    };

    function $(id) { return document.getElementById(id); }

    // ─── Toast & Modal helpers ──────────────────────────────────
    function toast(msg, type) {
      type = type || 'info';
      var el = document.createElement('div');
      el.className = 'toast ' + type;
      el.textContent = msg;
      $('toastArea').appendChild(el);
      setTimeout(function() { el.remove(); }, type === 'error' ? 5000 : 3000);
    }
    // Returns a Promise<boolean>
    function modalConfirm(title, message) {
      return new Promise(function(resolve) {
        var overlay = document.createElement('div');
        overlay.className = 'modal-overlay';
        overlay.innerHTML =
          '<div class="modal-box">' +
          '<h3>' + escapeHTML(title) + '</h3>' +
          '<p>' + escapeHTML(message) + '</p>' +
          '<div class="modal-actions">' +
          '<button class="ghost" id="_mNo">取消</button>' +
          '<button class="danger" id="_mYes">确认</button>' +
          '</div></div>';
        document.body.appendChild(overlay);
        overlay.querySelector('#_mYes').addEventListener('click', function() { overlay.remove(); resolve(true); });
        overlay.querySelector('#_mNo').addEventListener('click',  function() { overlay.remove(); resolve(false); });
      });
    }
    // Returns a Promise<string|null>
    function modalPrompt(title, placeholder, defaultVal) {
      return new Promise(function(resolve) {
        var overlay = document.createElement('div');
        overlay.className = 'modal-overlay';
        overlay.innerHTML =
          '<div class="modal-box">' +
          '<h3>' + escapeHTML(title) + '</h3>' +
          '<input id="_mInput" type="text" placeholder="' + escapeHTML(placeholder || '') + '" value="' + escapeHTML(defaultVal != null ? String(defaultVal) : '') + '">' +
          '<div class="modal-actions">' +
          '<button class="ghost" id="_mNo">取消</button>' +
          '<button id="_mYes">确认</button>' +
          '</div></div>';
        document.body.appendChild(overlay);
        var input = overlay.querySelector('#_mInput');
        input.focus(); input.select();
        overlay.querySelector('#_mYes').addEventListener('click', function() { overlay.remove(); resolve(input.value); });
        overlay.querySelector('#_mNo').addEventListener('click',  function() { overlay.remove(); resolve(null); });
        input.addEventListener('keydown', function(e) {
          if (e.key === 'Enter') { overlay.remove(); resolve(input.value); }
          if (e.key === 'Escape') { overlay.remove(); resolve(null); }
        });
      });
    }
    // password prompt
    function modalPassword(title) {
      return new Promise(function(resolve) {
        var overlay = document.createElement('div');
        overlay.className = 'modal-overlay';
        overlay.innerHTML =
          '<div class="modal-box">' +
          '<h3>' + escapeHTML(title) + '</h3>' +
          '<input id="_mInput" type="password" placeholder="新密码（至少 8 位）">' +
          '<div class="modal-actions">' +
          '<button class="ghost" id="_mNo">取消</button>' +
          '<button id="_mYes">确认</button>' +
          '</div></div>';
        document.body.appendChild(overlay);
        var input = overlay.querySelector('#_mInput');
        input.focus();
        overlay.querySelector('#_mYes').addEventListener('click', function() { overlay.remove(); resolve(input.value || null); });
        overlay.querySelector('#_mNo').addEventListener('click',  function() { overlay.remove(); resolve(null); });
        input.addEventListener('keydown', function(e) {
          if (e.key === 'Enter') { overlay.remove(); resolve(input.value || null); }
          if (e.key === 'Escape') { overlay.remove(); resolve(null); }
        });
      });
    }

    function escapeHTML(value) {
      return String(value || '').replace(/[&<>"']/g, function(ch) {
        return ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[ch];
      });
    }
    function fmtTime(ts) {
      if (!ts) return '永久';
      return new Date(ts * 1000).toLocaleString();
    }
    function fmtBytes(bytes) {
      if (!bytes) return '不限';
      return (bytes / 1024 / 1024).toFixed(0) + ' MB';
    }
    function detectDeviceInfo(userAgent) {
      const ua = String(userAgent || '').toLowerCase();
      if (!ua) {
        return { type: '未知', name: '未知设备' };
      }
      if (ua.indexOf('socks5') >= 0) {
        return { type: '代理客户端', name: 'SOCKS5 客户端' };
      }
      if (ua.indexOf('http connect') >= 0 || ua.indexOf('http 代理客户端') >= 0) {
        return { type: '代理客户端', name: 'HTTP 代理客户端' };
      }

      let deviceType = '桌面';
      if (ua.indexOf('bot') >= 0 || ua.indexOf('spider') >= 0 || ua.indexOf('crawler') >= 0) {
        deviceType = '爬虫/机器人';
      } else if (ua.indexOf('ipad') >= 0 || ua.indexOf('tablet') >= 0) {
        deviceType = '平板';
      } else if (ua.indexOf('iphone') >= 0 || ua.indexOf('android') >= 0 || ua.indexOf('mobile') >= 0) {
        deviceType = '手机';
      }

      let os = '未知系统';
      if (ua.indexOf('windows') >= 0) os = 'Windows';
      else if (ua.indexOf('android') >= 0) os = 'Android';
      else if (ua.indexOf('iphone') >= 0 || ua.indexOf('ipad') >= 0 || ua.indexOf('ios') >= 0) os = 'iOS';
      else if (ua.indexOf('mac os x') >= 0 || ua.indexOf('macintosh') >= 0) os = 'macOS';
      else if (ua.indexOf('linux') >= 0) os = 'Linux';

      let browser = '浏览器';
      if (ua.indexOf('micromessenger/') >= 0) browser = '微信内置浏览器';
      else if (ua.indexOf('curl/') >= 0) browser = 'curl';
      else if (ua.indexOf('postmanruntime/') >= 0) browser = 'Postman';
      else if (ua.indexOf('edg/') >= 0) browser = 'Edge';
      else if (ua.indexOf('chrome/') >= 0 && ua.indexOf('edg/') < 0) browser = 'Chrome';
      else if (ua.indexOf('safari/') >= 0 && ua.indexOf('chrome/') < 0) browser = 'Safari';
      else if (ua.indexOf('firefox/') >= 0) browser = 'Firefox';

      return {
        type: deviceType,
        name: browser + ' (' + os + ')'
      };
    }
    function setStatus(id, text, kind) {
      const el = $(id);
      el.textContent = text || '';
      el.className = 'status' + (kind ? ' ' + kind : '');
    }
    function setGlobalNotice(text) {
      var el = $('globalNotice');
      if (!el) return;
      var msg = String(text || '').trim();
      if (!msg) {
        el.classList.remove('show');
        el.textContent = '';
        return;
      }
      el.textContent = '系统提示：' + msg;
      el.classList.add('show');
    }
    function setNetworkBusy(isBusy) {
      var chip = $('chipNetState');
      if (!chip) return;
      chip.className = 'chip' + (isBusy ? ' busy' : '');
      chip.textContent = isBusy ? '网络请求中...' : '网络空闲';
    }
    function markDataRefreshed() {
      var now = new Date();
      if ($('chipLastRefresh')) {
        $('chipLastRefresh').textContent = '最近刷新：' + now.toLocaleTimeString();
      }
      if ($('chipDataState')) {
        $('chipDataState').textContent = '数据已同步';
        $('chipDataState').className = 'chip ok';
      }
    }
    function applySimpleMode(enable) {
      state.simpleMode = Boolean(enable);
      document.body.classList.toggle('simple', state.simpleMode);
      if ($('chipViewMode')) {
        $('chipViewMode').textContent = state.simpleMode ? '视图：简洁' : '视图：完整';
      }
      if (state.simpleMode && $('modulesView') && $('modulesView').classList.contains('active')) {
        setActiveTab('usersView');
      }
    }
    async function api(path, options) {
      const opts = Object.assign({ credentials: 'same-origin' }, options || {});
      const method = String((opts && opts.method) || 'GET').toUpperCase();
      if (method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS') {
        opts.headers = Object.assign({}, opts.headers || {});
        if (state.csrfToken) {
          opts.headers['X-CSRF-Token'] = state.csrfToken;
        }
      }
      state.pendingRequests += 1;
      setNetworkBusy(true);
      try {
        const res = await fetch(path, opts);
        const text = await res.text();
        let data = {};
        try { data = text ? JSON.parse(text) : {}; } catch (_) { data = { raw: text }; }
        if (res.status === 401) {
          showLogin();
          throw new Error('请重新登录');
        }
        if (!res.ok) {
          if (res.status === 403 && data && data.reason === 'csrf_check_failed') {
            state.csrfToken = '';
            showLogin();
            throw new Error('登录状态校验失败，请重新登录');
          }
          throw new Error(data.error || data.message || res.statusText || 'request_failed');
        }
        return data;
      } finally {
        state.pendingRequests = Math.max(0, state.pendingRequests - 1);
        setNetworkBusy(state.pendingRequests > 0);
      }
    }
    function showLogin() {
      $('authedShell').classList.add('hidden');
      $('loginView').classList.remove('hidden');
      $('appView').classList.add('hidden');
    }
    function showApp() {
      $('authedShell').classList.remove('hidden');
      $('loginView').classList.add('hidden');
      $('appView').classList.remove('hidden');
    }
    function syncProfile() {
      $('meName').textContent = state.me ? state.me.username : '-';
      $('meRole').textContent = state.me ? state.me.role : '-';
      $('meSessionCount').textContent = String(state.sessions.length || 0);
      $('meClientIP').textContent = state.me ? (state.me.client_ip || '-') : '-';
      $('meRemoteIP').textContent = state.me ? (state.me.remote_ip || '-') : '-';
      $('meIPMode').textContent = state.me ? (state.me.trust_proxy_headers ? ('可信头: ' + (state.me.real_ip_header || 'X-Forwarded-For')) : 'RemoteAddr') : '-';
    }

    function syncModulePanel() {
      if (!$('moduleUsersTotal')) return;
      $('moduleUsersTotal').textContent = String(state.total || state.users.length || 0);
      $('moduleSessionsTotal').textContent = String(state.sessions.length || 0);
      $('moduleAuditsTotal').textContent = String(state.audits.length || 0);
      $('moduleExpiredTotal').textContent = String((state.healthReport && state.healthReport.expired) || 0);
      $('moduleBlockedActive').textContent = String((state.securityStats && state.securityStats.blocked_active) || 0);
      $('securityRateLimitedTotal').textContent = String((state.securityStats && state.securityStats.admin_rate_limited_total) || 0);
      $('securityLoginFailedTotal').textContent = String((state.securityStats && state.securityStats.admin_login_failed_total) || 0);
      $('securityLoginBlockedTotal').textContent = String((state.securityStats && state.securityStats.admin_login_blocked_total) || 0);
      var threshold = state.securityAlertThreshold || { rateLimitedDelta: 3, loginFailedDelta: 5, loginBlockedDelta: 1, cooldownMs: 5 * 60 * 1000 };
      var cooldownMinutes = Math.max(1, Math.round(Number(threshold.cooldownMs || 0) / 60000));
      if ($('securityThresholdHint')) {
        $('securityThresholdHint').textContent = '告警阈值：限流 +' + threshold.rateLimitedDelta + '、登录失败 +' + threshold.loginFailedDelta + '、登录封禁 +' + threshold.loginBlockedDelta + '；冷却 ' + cooldownMinutes + ' 分钟';
      }
    }

    async function login() {
      const username = $('loginUsername').value.trim();
      const password = $('loginPassword').value.trim();
      if (!username || !password) {
        setStatus('loginStatus', '账号和密码不能为空', 'error');
        return;
      }
      try {
        const data = await api('/api/admin/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password })
        });
        state.csrfToken = data && data.csrf_token ? String(data.csrf_token) : '';
        setStatus('loginStatus', '登录成功', 'success');
        await bootstrapApp();
      } catch (err) {
        setGlobalNotice(err.message || '登录失败');
        setStatus('loginStatus', '登录失败：' + err.message, 'error');
      }
    }

    async function logout() {
      try { await api('/api/admin/logout', { method: 'POST' }); } catch (_) {}
      state.me = null;
      state.csrfToken = '';
      state.sessions = [];
      setGlobalNotice('');
      showLogin();
    }

    async function loadProfile() {
      state.me = await api('/api/admin/me');
      syncProfile();
    }

    async function loadSessions() {
      const data = await api('/api/admin/sessions');
      state.sessions = data.items || [];
      syncProfile();
      renderSessions();
      syncModulePanel();
    }

    async function loadUsers() {
      state.limit = Number($('userPageSize').value || '20');
      const params = new URLSearchParams({ offset: String(state.offset), limit: String(state.limit) });
      const q = $('userSearch').value.trim();
      const status = $('userStatusFilter').value;
      const tag = $('userTagFilter').value;
      const expiry = $('userExpiryFilter').value;
      if (q) params.set('q', q);
      if (status !== '') params.set('status', status);
      if (tag) params.set('tag', tag);
      if (expiry) params.set('expire_filter', expiry);
      const data = await api('/api/admin/users?' + params.toString());
      state.users = data.items || [];
      state.total = data.total || 0;
      renderUsers();
      renderPager();
      if (state.selectedUser) {
        await loadUserDetail(state.selectedUser);
      }
      syncModulePanel();
    }

    async function loadUserTags() {
      const selected = $('userTagFilter').value;
      const data = await api('/api/admin/users/tags');
      state.userTags = data.items || [];
      const options = ['<option value="">全部标签</option>'];
      state.userTags.forEach(function(tag) {
        options.push('<option value="' + escapeHTML(tag) + '">' + escapeHTML(tag) + '</option>');
      });
      $('userTagFilter').innerHTML = options.join('');
      if (selected && state.userTags.indexOf(selected) >= 0) {
        $('userTagFilter').value = selected;
      }
    }

    async function loadUserDetail(username) {
      state.selectedUser = username;
      try {
        const overview = await api('/api/admin/users/' + encodeURIComponent(username) + '/overview');
        renderUserDetail(overview.user || null, overview.devices || [], overview.audits || [], overview);
      } catch (err) {
        $('userDetail').classList.remove('hidden');
        $('userDetail').innerHTML = '<div class="danger-text">用户详情加载失败：' + escapeHTML(err.message) + '</div>';
      }
    }

    async function loadAdmins() {
      const data = await api('/api/admin/admins');
      state.admins = data.items || [];
      renderAdmins();
      syncModulePanel();
    }

    async function loadAudits() {
      const params = new URLSearchParams({ limit: '100' });
      if ($('auditActor').value.trim()) params.set('actor', $('auditActor').value.trim());
      if ($('auditAction').value.trim()) params.set('action', $('auditAction').value.trim());
      if ($('auditTarget').value.trim()) params.set('target', $('auditTarget').value.trim());
      const data = await api('/api/admin/audits?' + params.toString());
      state.audits = data.items || [];
      renderAudits();
      syncModulePanel();
    }

    function evaluateSecurityAlerts(nextStats, withNotify) {
      var next = {
        rateLimited: Number((nextStats && nextStats.admin_rate_limited_total) || 0),
        loginFailed: Number((nextStats && nextStats.admin_login_failed_total) || 0),
        loginBlocked: Number((nextStats && nextStats.admin_login_blocked_total) || 0)
      };
      var prev = state.lastSecuritySnapshot;
      state.lastSecuritySnapshot = next;
      if (!withNotify || !prev) return;

      var threshold = state.securityAlertThreshold || {
        rateLimitedDelta: 3,
        loginFailedDelta: 5,
        loginBlockedDelta: 1,
        cooldownMs: 5 * 60 * 1000
      };

      var deltaRate = next.rateLimited - prev.rateLimited;
      var deltaFail = next.loginFailed - prev.loginFailed;
      var deltaBlocked = next.loginBlocked - prev.loginBlocked;
      var now = Date.now();
      var parts = [];
      if (deltaRate >= threshold.rateLimitedDelta) parts.push('限流 +' + deltaRate);
      if (deltaFail >= threshold.loginFailedDelta) parts.push('登录失败 +' + deltaFail);
      if (deltaBlocked >= threshold.loginBlockedDelta) parts.push('登录封禁 +' + deltaBlocked);
      if (parts.length) {
        if (now - state.lastSecurityAlertAt < threshold.cooldownMs) return;
        state.lastSecurityAlertAt = now;
        toast('安全告警：' + parts.join('，'), 'error');
      }
    }

    async function loadSecurityStats(withNotify) {
      const data = await api('/api/admin/security-stats');
      state.securityStats = data || null;
      if (state.securityStats) {
        state.securityAlertThreshold = {
          rateLimitedDelta: Math.max(1, Number(state.securityStats.ui_alert_rate_limited_delta) || 3),
          loginFailedDelta: Math.max(1, Number(state.securityStats.ui_alert_login_failed_delta) || 5),
          loginBlockedDelta: Math.max(1, Number(state.securityStats.ui_alert_login_blocked_delta) || 1),
          cooldownMs: Math.max(1000, (Number(state.securityStats.ui_alert_cooldown_seconds) || 300) * 1000)
        };
      }
      evaluateSecurityAlerts(state.securityStats, Boolean(withNotify));
      syncModulePanel();
    }

    function renderUsers() {
      if (!state.users.length) {
        $('usersTableWrap').innerHTML = '<p class="note">没有匹配的用户。</p>';
        $('selectedUserCount').textContent = String(state.selectedUsers.size);
        return;
      }
      let html = '<table><thead><tr>' +
        '<th><input type="checkbox" id="selectAllUsers"></th>' +
        '<th>用户名</th><th>标签</th><th>状态</th><th>到期</th><th>流量</th><th>设备</th><th>更新时间</th><th>操作</th></tr></thead><tbody>';
      state.users.forEach(function(user, idx) {
        const checked = state.selectedUsers.has(user.username) ? ' checked' : '';
        const action = user.status === 1 ? 'disable' : 'enable';
        const actionLabel = user.status === 1 ? '禁用' : '启用';
        const panelId = 'userOpsMore_' + idx;
        html += '<tr>' +
          '<td><input type="checkbox" class="user-check" data-username="' + escapeHTML(user.username) + '"' + checked + '></td>' +
          '<td class="mono"><button class="ghost" onclick="selectUserDetail(\'' + escapeHTML(user.username) + '\')">' + escapeHTML(user.username) + '</button></td>' +
          '<td><span class="badge">' + escapeHTML(user.tag || '-') + '</span></td>' +
          '<td>' + (user.status === 1 ? '启用' : '禁用') + '</td>' +
          '<td>' + fmtTime(user.expires_at) + '</td>' +
          '<td>' + (user.quota_bytes > 0 ? ((user.used_bytes / 1024 / 1024).toFixed(0) + ' / ' + (user.quota_bytes / 1024 / 1024).toFixed(0) + ' MB') : '不限') + '</td>' +
          '<td>' + user.active_ips + ' / ' + user.max_devices + '</td>' +
          '<td>' + fmtTime(user.updated_at) + '</td>' +
          '<td><div class="user-op">' +
          '<div class="user-op-main">' +
          '<button class="ghost" onclick="selectUserDetail(\'' + escapeHTML(user.username) + '\')">详情/记录</button>' +
          '<button class="ghost" onclick="toggleUser(\'' + escapeHTML(user.username) + '\', \'' + action + '\')">' + actionLabel + '</button>' +
          '<button class="secondary" onclick="toggleUserOps(\'' + panelId + '\', this)">更多</button>' +
          '</div>' +
          '<div class="user-op-more hidden" id="' + panelId + '">' +
          '<button class="ghost" onclick="setUserTag(\'' + escapeHTML(user.username) + '\', \'' + escapeHTML(user.tag || '') + '\')">标签</button>' +
          '<button class="ghost" onclick="extendUserDays(\'' + escapeHTML(user.username) + '\')">延期</button>' +
          '<button class="ghost" onclick="topupUserQuota(\'' + escapeHTML(user.username) + '\')">充值</button>' +
          '<button class="ghost" onclick="setUserQuota(\'' + escapeHTML(user.username) + '\', ' + (user.quota_bytes || 0) + ')">设流量</button>' +
          (user.smb_enabled === 1
            ? '<button class="ghost" onclick="topupUserSMBQuota(\'' + escapeHTML(user.username) + '\')">SMB扩容</button><button class="ghost" onclick="setUserSMBQuota(\'' + escapeHTML(user.username) + '\', ' + (user.smb_quota_bytes || 0) + ')">设SMB</button><button class="ghost" onclick="setUserSMB(\'' + escapeHTML(user.username) + '\', false)">关闭SMB</button>'
            : '<button class="ghost" onclick="setUserSMB(\'' + escapeHTML(user.username) + '\', true)">开通SMB</button>') +
          '<button class="ghost" onclick="setUserSpeedLimit(\'' + escapeHTML(user.username) + '\', ' + (user.speed_limit_kbps || 0) + ')">限速</button>' +
          '<button class="ghost" onclick="editUserDevices(\'' + escapeHTML(user.username) + '\', ' + user.max_devices + ')">设备</button>' +
          '<button class="ghost" onclick="resetUserUsage(\'' + escapeHTML(user.username) + '\')">清流量</button>' +
          '<button class="ghost" onclick="resetUserPassword(\'' + escapeHTML(user.username) + '\')">改密码</button>' +
          '<button class="ghost" onclick="viewUserAudit(\'' + escapeHTML(user.username) + '\')">审计</button>' +
          '<button class="danger" onclick="deleteUser(\'' + escapeHTML(user.username) + '\')">删除</button>' +
          '</div></div>' +
          '</td>' +
          '</tr>';
      });
      html += '</tbody></table>';
      $('usersTableWrap').innerHTML = html;
      $('selectedUserCount').textContent = String(state.selectedUsers.size);
      const selectAll = $('selectAllUsers');
      if (selectAll) {
        selectAll.checked = state.users.length > 0 && state.users.every(function(user) { return state.selectedUsers.has(user.username); });
        selectAll.addEventListener('change', function() {
          if (this.checked) {
            state.users.forEach(function(user) { state.selectedUsers.add(user.username); });
          } else {
            state.users.forEach(function(user) { state.selectedUsers.delete(user.username); });
          }
          renderUsers();
        });
      }
      document.querySelectorAll('.user-check').forEach(function(input) {
        input.addEventListener('change', function() {
          const username = this.getAttribute('data-username');
          if (this.checked) state.selectedUsers.add(username); else state.selectedUsers.delete(username);
          $('selectedUserCount').textContent = String(state.selectedUsers.size);
        });
      });
    }

    function renderPager() {
      const from = state.total === 0 ? 0 : state.offset + 1;
      const to = Math.min(state.offset + state.limit, state.total);
      $('pageInfo').textContent = '显示 ' + from + '-' + to + ' / ' + state.total;
      $('prevPageBtn').disabled = state.offset <= 0;
      $('nextPageBtn').disabled = state.offset + state.limit >= state.total;
    }

    function renderUserDetail(user, devices, audits, deviceMeta) {
      if (!user) {
        $('userDetail').classList.remove('hidden');
        $('userDetail').innerHTML = '<p class="note">用户信息不存在或已被删除。</p>';
        return;
      }
      var ipMode = deviceMeta && deviceMeta.trust_proxy_headers ? ('已启用可信头解析（' + escapeHTML(deviceMeta.real_ip_header || 'X-Forwarded-For') + '）') : '使用连接源地址（RemoteAddr）';
      let html = '<h3>用户详情：<span class="mono">' + escapeHTML(user.username) + '</span></h3>' +
        '<div class="summary">' +
        '<div class="metric"><b>' + (user.status === 1 ? '启用' : '禁用') + '</b><span>状态</span></div>' +
        '<div class="metric"><b>' + user.active_ips + ' / ' + user.max_devices + '</b><span>活跃设备</span></div>' +
        '<div class="metric"><b>' + (user.quota_bytes > 0 ? ((Math.max(user.quota_bytes - user.used_bytes, 0) / 1024 / 1024).toFixed(0) + ' MB') : '不限') + '</b><span>剩余流量</span></div>' +
        '</div>' +
        '<div class="note">SMB 状态：' + (user.smb_enabled === 1 ? '已开通' : '未开通') + '</div>' +
        '<div class="note">SMB 空间：' + (user.smb_quota_bytes > 0 ? ((user.smb_used_bytes / 1024 / 1024).toFixed(0) + ' / ' + (user.smb_quota_bytes / 1024 / 1024).toFixed(0) + ' MB') : '不限') + '</div>' +
        '<div class="note">代理速度：' + (user.speed_limit_kbps > 0 ? (user.speed_limit_kbps + ' KB/s') : '不限') + '</div>' +
        '<div class="note">SMB 目录：<span class="mono">' + escapeHTML(user.smb_enabled === 1 ? ((deviceMeta && deviceMeta.smb_path) || '-') : '-') + '</span></div>' +
        '<div class="note">WebDAV：<span class="mono">' + (user.smb_enabled === 1 ? escapeHTML(location.origin + '/webdav/') : '-') + '</span>（账号：' + escapeHTML(user.username) + '）</div>' +
        '<div class="note">标签：<span class="badge">' + escapeHTML(user.tag || '-') + '</span></div>' +
        '<div class="note">创建时间：' + fmtTime(user.created_at) + '</div>' +
        '<div class="note">到期时间：' + fmtTime(user.expires_at) + '，更新时间：' + fmtTime(user.updated_at) + '</div>' +
        '<div class="note">IP来源说明：' + ipMode + '</div>';
      html += '<h4>用户登录/使用记录</h4>';
      html += '<p class="note">这里显示的是该用户最近的登录与使用设备记录，包括设备类型、首次登录时间和最近使用时间。</p>';
      if (!devices.length) {
        html += '<p class="note">当前没有最近登录/使用记录。下方表格如果有内容，那是管理员后台操作记录，不是用户登录记录。</p>';
      } else {
        html += '<table><thead><tr><th>客户端IP</th><th>设备类型</th><th>设备名称</th><th>首次登录</th><th>最近使用</th><th>操作</th></tr></thead><tbody>';
        devices.forEach(function(device) {
          const deviceInfo = detectDeviceInfo(device.user_agent);
          const firstSeen = device.first_seen || device.last_seen;
          html += '<tr><td class="mono">' + escapeHTML(device.ip) + '</td><td>' + escapeHTML(deviceInfo.type) + '</td>' +
            '<td title="' + escapeHTML(device.user_agent || '-') + '">' + escapeHTML(deviceInfo.name) + '</td><td>' + fmtTime(firstSeen) + '</td><td>' + fmtTime(device.last_seen) + '</td>' +
            '<td><button class="danger" onclick="kickUserDevice(\'' + escapeHTML(user.username) + '\', \'' + escapeHTML(device.ip) + '\')">踢出</button></td></tr>';
        });
        html += '</tbody></table>';
      }
      html += '<h4>管理员后台操作记录</h4>';
      html += '<p class="note">下面显示的是管理员对该用户做过的后台操作，例如改密、延期、禁用、踢设备，不是用户自己的登录记录。</p>';
      if (!audits || !audits.length) {
        html += '<p class="note">暂无管理员后台操作日志。</p>';
      } else {
        html += '<table><thead><tr><th>时间</th><th>操作者</th><th>动作</th><th>详情</th></tr></thead><tbody>';
        audits.forEach(function(item) {
          html += '<tr>' +
            '<td>' + fmtTime(item.created_at) + '</td>' +
            '<td class="mono">' + escapeHTML(item.actor || '-') + '</td>' +
            '<td>' + escapeHTML(item.action || '-') + '</td>' +
            '<td class="mono">' + escapeHTML(item.detail || '-') + '</td>' +
            '</tr>';
        });
        html += '</tbody></table>';
      }
      $('userDetail').classList.remove('hidden');
      $('userDetail').innerHTML = html;
    }

    function renderAdmins() {
      if (!state.admins.length) {
        $('adminsTableWrap').innerHTML = '<p class="note">暂无管理员。</p>';
        return;
      }
      let html = '<table><thead><tr><th>账号</th><th>角色</th><th>状态</th><th>密码</th><th>创建时间</th><th>操作</th></tr></thead><tbody>';
      state.admins.forEach(function(admin) {
        const toggleAction = admin.status === 1 ? 'disable' : 'enable';
        const toggleLabel = admin.status === 1 ? '禁用' : '启用';
        const nextRole = admin.role === 'super' ? 'readonly' : 'super';
        const nextRoleLabel = admin.role === 'super' ? '降为只读' : '升为 super';
        html += '<tr>' +
          '<td class="mono">' + escapeHTML(admin.username) + '</td>' +
          '<td>' + escapeHTML(admin.role) + '</td>' +
          '<td>' + (admin.status === 1 ? '启用' : '禁用') + '</td>' +
          '<td>' + (admin.password_set ? '已设置' : '未设置') + '</td>' +
          '<td>' + fmtTime(admin.created_at) + '</td>' +
          '<td>' +
          '<button class="ghost" onclick="toggleAdmin(\'' + escapeHTML(admin.username) + '\', \'' + toggleAction + '\')">' + toggleLabel + '</button> ' +
          '<button class="ghost" onclick="changeAdminRole(\'' + escapeHTML(admin.username) + '\', \'' + nextRole + '\')">' + nextRoleLabel + '</button> ' +
          '<button class="ghost" onclick="resetAdminPassword(\'' + escapeHTML(admin.username) + '\')">重置密码</button>' +
          '</td></tr>';
      });
      html += '</tbody></table>';
      $('adminsTableWrap').innerHTML = html;
    }

    function renderSessions() {
      if ($('sessionsOwnerBadge')) {
        $('sessionsOwnerBadge').textContent = '当前账号在线会话' + (state.me && state.me.username ? '：' + state.me.username : '');
      }
      if (!state.sessions.length) {
        $('sessionsTableWrap').innerHTML = '<p class="note">暂无在线会话。</p>';
        return;
      }
      let html = '<table><thead><tr><th>账号</th><th>会话ID</th><th>设备类型</th><th>设备名称</th><th>登录IP</th><th>原始登录IP</th><th>创建时间</th><th>最后活跃</th><th>到期</th><th>操作</th></tr></thead><tbody>';
      state.sessions.forEach(function(item) {
        const deviceInfo = detectDeviceInfo(item.user_agent);
        const displayIP = item.ip_address || '-';
        const originalIP = item.original_ip || item.ip_address || '-';
        html += '<tr>' +
          '<td><span class="badge">' + escapeHTML(item.username || (state.me && state.me.username) || '-') + '</span></td>' +
          '<td class="mono">' + escapeHTML(item.session_id) + '</td>' +
          '<td>' + escapeHTML(deviceInfo.type) + '</td>' +
          '<td title="' + escapeHTML(item.user_agent || '-') + '">' + escapeHTML(deviceInfo.name) + '</td>' +
          '<td class="mono">' + escapeHTML(displayIP) + '</td>' +
          '<td class="mono">' + escapeHTML(originalIP) + '</td>' +
          '<td>' + fmtTime(item.created_at) + '</td>' +
          '<td>' + fmtTime(item.last_activity) + '</td>' +
          '<td>' + fmtTime(item.expires_at) + '</td>' +
          '<td><button class="danger" onclick="deleteSession(\'' + escapeHTML(item.session_id) + '\')">下线</button></td>' +
          '</tr>';
      });
      html += '</tbody></table>';
      $('sessionsTableWrap').innerHTML = html;
    }

    function renderAudits() {
      if (!state.audits.length) {
        $('auditsTableWrap').innerHTML = '<p class="note">暂无审计日志。</p>';
        return;
      }
      let html = '<table><thead><tr><th>ID</th><th>操作者</th><th>动作</th><th>目标</th><th>详情</th><th>时间</th></tr></thead><tbody>';
      state.audits.forEach(function(item) {
        html += '<tr><td>' + item.id + '</td><td class="mono">' + escapeHTML(item.actor) + '</td><td>' + escapeHTML(item.action) + '</td><td class="mono">' + escapeHTML(item.target) + '</td><td class="mono">' + escapeHTML(item.detail || '-') + '</td><td>' + fmtTime(item.created_at) + '</td></tr>';
      });
      html += '</tbody></table>';
      $('auditsTableWrap').innerHTML = html;
    }

    async function createUser() {
      const username = $('createUserName').value.trim();
      const password = $('createUserPassword').value.trim();
      const maxDevices = Number($('createUserDevices').value || '1');
      const days = Number($('createUserDays').value || '0');
      const quotaMB = Number($('createUserQuota').value || '0');
      const smbQuotaMB = Number($('createUserSMBQuota').value || '0');
      const speedKbps = Number($('createUserSpeed').value || '0');
      const smbEnabled = $('createUserSMBEnabled').checked;
      if (!username || !password) {
        setStatus('createUserStatus', '用户名和密码不能为空', 'error');
        return;
      }
      try {
        await api('/api/admin/users', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password, max_devices: maxDevices, quota_bytes: quotaMB * 1024 * 1024, smb_enabled: smbEnabled, smb_quota_bytes: smbQuotaMB * 1024 * 1024, speed_limit_kbps: speedKbps, expires_at: 0 })
        });
        if (days > 0) {
          await api('/api/admin/users/' + encodeURIComponent(username) + '/extend', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ days: days })
          });
        }
        setStatus('createUserStatus', '创建成功', 'success');
        $('createUserName').value = '';
        $('createUserPassword').value = '';
        $('createUserSMBQuota').value = '0';
        $('createUserSpeed').value = '0';
        $('createUserSMBEnabled').checked = false;
        await loadUsers();
        await loadAudits();
      } catch (err) {
        setStatus('createUserStatus', '创建失败：' + err.message, 'error');
      }
    }

    async function createAdmin() {
      const username = $('createAdminName').value.trim();
      const password = $('createAdminPassword').value.trim();
      const role = $('createAdminRole').value;
      if (!username || !password) {
        setStatus('createAdminStatus', '管理员账号和密码不能为空', 'error');
        return;
      }
      try {
        await api('/api/admin/admins', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password, role })
        });
        setStatus('createAdminStatus', '管理员创建成功', 'success');
        $('createAdminName').value = '';
        $('createAdminPassword').value = '';
        await loadAdmins();
        await loadAudits();
      } catch (err) {
        setStatus('createAdminStatus', '创建失败：' + err.message, 'error');
      }
    }

    async function changeMyPassword() {
      const oldPassword = $('oldAdminPassword').value.trim();
      const newPassword = $('newAdminPassword').value.trim();
      if (!oldPassword || !newPassword) {
        setStatus('changeMyPasswordStatus', '旧密码和新密码不能为空', 'error');
        return;
      }
      try {
        await api('/api/admin/profile/password', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ old_password: oldPassword, new_password: newPassword })
        });
        setStatus('changeMyPasswordStatus', '密码修改成功', 'success');
        $('oldAdminPassword').value = '';
        $('newAdminPassword').value = '';
        await loadAudits();
      } catch (err) {
        setStatus('changeMyPasswordStatus', '修改失败：' + err.message, 'error');
      }
    }

    async function toggleUser(username, action) {
      await api('/api/admin/users/' + encodeURIComponent(username) + '/' + action, { method: 'POST' });
      toast((action === 'disable' ? '已禁用' : '已启用') + ' ' + username, 'success');
      await loadUsers();
      await loadAudits();
    }
    async function editUserDevices(username, current) {
      const nextValue = await modalPrompt('修改最大设备数', '请输入整数', current || 1);
      if (nextValue === null) return;
      const maxDevices = Number(nextValue);
      if (!Number.isInteger(maxDevices) || maxDevices < 1) {
        toast('请输入大于等于 1 的整数', 'error');
        return;
      }
      await api('/api/admin/users/' + encodeURIComponent(username) + '/set-devices', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ max_devices: maxDevices })
      });
      toast('设备数已更新：' + username + ' → ' + maxDevices, 'success');
      await loadUsers();
      await loadAudits();
    }
    async function resetUserUsage(username) {
      if (!await modalConfirm('清零流量', '确认把用户 ' + username + ' 的已用流量清零？')) return;
      await api('/api/admin/users/' + encodeURIComponent(username) + '/usage-reset', { method: 'POST' });
      toast('流量已清零：' + username, 'success');
      await loadUsers();
      await loadAudits();
      if (state.selectedUser === username) await loadUserDetail(username);
    }
    async function resetUserPassword(username) {
      const password = await modalPassword('修改用户密码：' + username);
      if (password === null) return;
      await api('/api/admin/users/' + encodeURIComponent(username) + '/password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password })
      });
      toast('密码已更新：' + username, 'success');
      await loadAudits();
    }
    async function deleteUser(username) {
      if (!await modalConfirm('删除用户', '确认删除用户 ' + username + '？该操作不可恢复。')) return;
      await api('/api/admin/users/' + encodeURIComponent(username), { method: 'DELETE' });
      state.selectedUsers.delete(username);
      if (state.selectedUser === username) {
        state.selectedUser = '';
        $('userDetail').classList.add('hidden');
        $('userDetail').innerHTML = '';
      }
      toast('已删除用户：' + username, 'info');
      await loadUsers();
      await loadAudits();
    }
    async function selectUserDetail(username) {
      await loadUserDetail(username);
    }
    async function kickUserDevice(username, ip) {
      if (!await modalConfirm('踢出设备', '确认踢出设备 ' + ip + '？')) return;
      await api('/api/admin/users/' + encodeURIComponent(username) + '/devices/' + encodeURIComponent(ip), { method: 'DELETE' });
      toast('已踢出设备 ' + ip, 'info');
      await loadUsers();
      await loadUserDetail(username);
      await loadAudits();
    }
    async function extendUserDays(username) {
      const input = await modalPrompt('延长到期时间', '输入天数（可为负数）', '30');
      if (input === null) return;
      const days = Number(input);
      if (!Number.isInteger(days)) {
        toast('请输入整数天数', 'error');
        return;
      }
      await api('/api/admin/users/' + encodeURIComponent(username) + '/extend', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ days: days })
      });
      toast('已调整到期时间：' + username + '（' + (days >= 0 ? '+' : '') + days + ' 天）', 'success');
      await loadUsers();
      await loadAudits();
      if (state.selectedUser === username) await loadUserDetail(username);
    }
    async function setUserTag(username, currentTag) {
      const value = await modalPrompt('设置用户标签', '例如：业务A/测试组（留空=清除）', currentTag || '');
      if (value === null) return;
      const tag = String(value || '').trim();
      await api('/api/admin/users/' + encodeURIComponent(username) + '/set-tag', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tag: tag })
      });
      toast('标签已更新：' + username + ' → ' + (tag || '空'), 'success');
      await loadUserTags();
      await loadUsers();
      await loadAudits();
    }
    async function topupUserQuota(username) {
      const input = await modalPrompt('流量充值', '输入要增加的 MB（可为负数）', '1024');
      if (input === null) return;
      const mb = Number(input);
      if (!Number.isFinite(mb)) {
        toast('请输入有效数字', 'error');
        return;
      }
      const bytes = Math.round(mb * 1024 * 1024);
      await api('/api/admin/users/' + encodeURIComponent(username) + '/topup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ bytes: bytes })
      });
      toast('已充值流量：' + username + '（' + mb + ' MB）', 'success');
      await loadUsers();
      await loadAudits();
      if (state.selectedUser === username) await loadUserDetail(username);
    }
    async function topupUserSMBQuota(username) {
      const input = await modalPrompt('SMB 扩容', '输入要增加的 MB', '1024');
      if (input === null) return;
      const mb = Number(input);
      if (!Number.isFinite(mb) || mb <= 0) {
        toast('请输入大于 0 的数字', 'error');
        return;
      }
      const bytes = Math.round(mb * 1024 * 1024);
      await api('/api/admin/users/' + encodeURIComponent(username) + '/smb-topup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ bytes: bytes })
      });
      toast('SMB 扩容完成：' + username + '（+' + mb + ' MB）', 'success');
      await loadUsers();
      await loadAudits();
      if (state.selectedUser === username) await loadUserDetail(username);
    }
    async function setUserSMB(username, enable) {
      const action = enable ? 'smb-enable' : 'smb-disable';
      await api('/api/admin/users/' + encodeURIComponent(username) + '/' + action, { method: 'POST' });
      toast((enable ? '已开通 SMB：' : '已关闭 SMB：') + username, 'success');
      await loadUsers();
      await loadAudits();
      if (state.selectedUser === username) await loadUserDetail(username);
    }
    async function setUserQuota(username, currentBytes) {
      const currentMB = Math.round((Number(currentBytes) || 0) / 1024 / 1024);
      const input = await modalPrompt('设置总流量额度', '输入总额度 MB（0=不限）', String(currentMB));
      if (input === null) return;
      const mb = Number(input);
      if (!Number.isFinite(mb) || mb < 0) {
        toast('请输入大于等于 0 的数字', 'error');
        return;
      }
      await api('/api/admin/users/' + encodeURIComponent(username) + '/set-quota', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ bytes: Math.round(mb * 1024 * 1024) })
      });
      toast('总流量额度已设置：' + username + '（' + mb + ' MB）', 'success');
      await loadUsers(); await loadAudits(); if (state.selectedUser === username) await loadUserDetail(username);
    }
    async function setUserSMBQuota(username, currentBytes) {
      const currentMB = Math.round((Number(currentBytes) || 0) / 1024 / 1024);
      const input = await modalPrompt('设置 SMB 总空间', '输入总空间 MB（0=不限）', String(currentMB));
      if (input === null) return;
      const mb = Number(input);
      if (!Number.isFinite(mb) || mb < 0) {
        toast('请输入大于等于 0 的数字', 'error');
        return;
      }
      await api('/api/admin/users/' + encodeURIComponent(username) + '/set-smb-quota', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ bytes: Math.round(mb * 1024 * 1024) })
      });
      toast('SMB 总空间已设置：' + username + '（' + mb + ' MB）', 'success');
      await loadUsers(); await loadAudits(); if (state.selectedUser === username) await loadUserDetail(username);
    }
    async function setUserSpeedLimit(username, currentKbps) {
      const input = await modalPrompt('设置代理速度', '输入 KB/s（0=不限）', String(Number(currentKbps) || 0));
      if (input === null) return;
      const kbps = Number(input);
      if (!Number.isFinite(kbps) || kbps < 0) {
        toast('请输入大于等于 0 的数字', 'error');
        return;
      }
      await api('/api/admin/users/' + encodeURIComponent(username) + '/set-speed', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ kbps: Math.round(kbps) })
      });
      toast('代理速度已设置：' + username + '（' + Math.round(kbps) + ' KB/s）', 'success');
      await loadUsers(); await loadAudits(); if (state.selectedUser === username) await loadUserDetail(username);
    }
    function toggleUserOps(panelId, btn) {
      var panel = document.getElementById(panelId);
      if (!panel) return;
      var willShow = panel.classList.contains('hidden');
      document.querySelectorAll('.user-op-more').forEach(function(el) {
        el.classList.add('hidden');
      });
      document.querySelectorAll('.user-op-main .secondary').forEach(function(el) {
        if (el.textContent === '收起') el.textContent = '更多';
      });
      if (willShow) {
        panel.classList.remove('hidden');
        if (btn) btn.textContent = '收起';
      }
    }
    function setActiveTab(tabId) {
      document.querySelectorAll('.tab').forEach(function(item) { item.classList.remove('active'); });
      document.querySelectorAll('.view').forEach(function(item) { item.classList.remove('active'); });
      var tabButton = document.querySelector('.tab[data-tab="' + tabId + '"]');
      if (tabButton) tabButton.classList.add('active');
      $(tabId).classList.add('active');
    }
    function setActiveOpPane(opKey) {
      var normalized = (opKey || 'user').toLowerCase();
      document.querySelectorAll('.op-tab').forEach(function(btn) {
        btn.classList.toggle('active', btn.getAttribute('data-op') === normalized);
      });
      document.querySelectorAll('.op-pane').forEach(function(pane) {
        pane.classList.remove('active');
      });
      var paneMap = {
        user: 'opPaneUser',
        admin: 'opPaneAdmin',
        password: 'opPanePassword'
      };
      var target = $(paneMap[normalized] || 'opPaneUser');
      if (target) target.classList.add('active');
    }
    async function viewUserAudit(username) {
      $('auditActor').value = '';
      $('auditAction').value = '';
      $('auditTarget').value = username;
      await loadAudits();
      setActiveTab('auditsView');
      toast('已切换到审计视图：' + username, 'info');
    }
    async function batchSetUsers(enabled) {
      const usernames = Array.from(state.selectedUsers);
      if (!usernames.length) {
        toast('请先选择用户', 'error');
        return;
      }
      if (!await modalConfirm('批量操作', '确认' + (enabled ? '启用' : '禁用') + '选中的 ' + usernames.length + ' 个用户？')) return;
      await api('/api/admin/users/batch-status', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ usernames, enabled })
      });
      toast('批量操作已完成，共 ' + usernames.length + ' 个用户', 'success');
      state.selectedUsers.clear();
      await loadUsers();
      await loadAudits();
    }
    async function batchExtendUsers() {
      const usernames = Array.from(state.selectedUsers);
      if (!usernames.length) {
        toast('请先选择用户', 'error');
        return;
      }
      const input = await modalPrompt('批量延期', '输入天数（可为负数）', '30');
      if (input === null) return;
      const days = Number(input);
      if (!Number.isInteger(days)) {
        toast('请输入整数天数', 'error');
        return;
      }
      if (!await modalConfirm('批量延期', '确认对 ' + usernames.length + ' 个用户调整 ' + (days >= 0 ? '+' : '') + days + ' 天？')) return;
      let success = 0;
      const failed = [];
      for (var i = 0; i < usernames.length; i++) {
        try {
          await api('/api/admin/users/' + encodeURIComponent(usernames[i]) + '/extend', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ days: days })
          });
          success++;
        } catch (err) {
          failed.push(usernames[i] + ': ' + err.message);
        }
      }
      await loadUsers();
      await loadAudits();
      toast('批量延期完成：成功 ' + success + '，失败 ' + failed.length, failed.length ? 'error' : 'success');
    }
    async function batchSetTagUsers() {
      const usernames = Array.from(state.selectedUsers);
      if (!usernames.length) {
        toast('请先选择用户', 'error');
        return;
      }
      const value = await modalPrompt('批量打标', '输入标签（留空=清除）', '');
      if (value === null) return;
      const tag = String(value || '').trim();
      if (!await modalConfirm('批量打标', '确认对 ' + usernames.length + ' 个用户设置标签为「' + (tag || '空') + '」？')) return;
      let success = 0;
      const failed = [];
      for (var i = 0; i < usernames.length; i++) {
        try {
          await api('/api/admin/users/' + encodeURIComponent(usernames[i]) + '/set-tag', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ tag: tag })
          });
          success++;
        } catch (err) {
          failed.push(usernames[i] + ': ' + err.message);
        }
      }
      await loadUserTags();
      await loadUsers();
      await loadAudits();
      toast('批量打标完成：成功 ' + success + '，失败 ' + failed.length, failed.length ? 'error' : 'success');
    }
    async function batchTopupUsers() {
      const usernames = Array.from(state.selectedUsers);
      if (!usernames.length) {
        toast('请先选择用户', 'error');
        return;
      }
      const input = await modalPrompt('批量充值', '输入 MB（可为负数）', '1024');
      if (input === null) return;
      const mb = Number(input);
      if (!Number.isFinite(mb)) {
        toast('请输入有效数字', 'error');
        return;
      }
      const bytes = Math.round(mb * 1024 * 1024);
      if (!await modalConfirm('批量充值', '确认对 ' + usernames.length + ' 个用户调整 ' + mb + ' MB 流量？')) return;
      let success = 0;
      const failed = [];
      for (var i = 0; i < usernames.length; i++) {
        try {
          await api('/api/admin/users/' + encodeURIComponent(usernames[i]) + '/topup', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ bytes: bytes })
          });
          success++;
        } catch (err) {
          failed.push(usernames[i] + ': ' + err.message);
        }
      }
      await loadUsers();
      await loadAudits();
      toast('批量充值完成：成功 ' + success + '，失败 ' + failed.length, failed.length ? 'error' : 'success');
    }
    async function fetchAllUsersForExport() {
      var offset = 0;
      var limit = 200;
      var all = [];
      var q = $('userSearch').value.trim();
      var status = $('userStatusFilter').value;
      var tag = $('userTagFilter').value;
      var expiry = $('userExpiryFilter').value;
      while (true) {
        var params = new URLSearchParams({ offset: String(offset), limit: String(limit) });
        if (q) params.set('q', q);
        if (status !== '') params.set('status', status);
        if (tag) params.set('tag', tag);
        if (expiry) params.set('expire_filter', expiry);
        var data = await api('/api/admin/users?' + params.toString());
        var items = data.items || [];
        all = all.concat(items);
        if (items.length < limit || all.length >= (data.total || 0)) break;
        offset += limit;
      }
      return all;
    }
    async function fetchAllUsersRaw() {
      var offset = 0;
      var limit = 200;
      var all = [];
      while (true) {
        var data = await api('/api/admin/users?offset=' + offset + '&limit=' + limit);
        var items = data.items || [];
        all = all.concat(items);
        if (items.length < limit || all.length >= (data.total || 0)) break;
        offset += limit;
      }
      return all;
    }
    function renderHealthPanel(report) {
      $('expiredCount').textContent = String(report.expired);
      $('expiring7Count').textContent = String(report.expiring7);
      $('nearQuotaCount').textContent = String(report.nearQuota);

      var lines = [];
      if (report.expired > 0) lines.push('<li><b>已过期用户：</b>' + report.expired + ' 个，建议优先处理。</li>');
      if (report.expiring3 > 0) lines.push('<li><b>3天内到期：</b>' + report.expiring3 + ' 个，建议提前续期。</li>');
      if (report.nearQuota > 0) lines.push('<li><b>流量超90%：</b>' + report.nearQuota + ' 个，建议提醒或充值。</li>');
      if (report.overDevices > 0) lines.push('<li><b>设备超限：</b>' + report.overDevices + ' 个，请检查共享账号风险。</li>');
      if (!lines.length) lines.push('<li><b>巡检通过：</b>当前无高风险项。</li>');

      $('healthList').innerHTML = lines.join('');
      $('healthLastScan').textContent = '最近巡检：' + new Date().toLocaleString();
    }
    async function runHealthCheck(withNotify) {
      var users = await fetchAllUsersRaw();
      var now = Math.floor(Date.now() / 1000);
      var in3 = now + 3 * 24 * 3600;
      var in7 = now + 7 * 24 * 3600;
      var expired = 0;
      var expiring3 = 0;
      var expiring7 = 0;
      var nearQuota = 0;
      var overDevices = 0;
      users.forEach(function(user) {
        if (user.expires_at > 0 && user.expires_at < now) expired++;
        if (user.expires_at > 0 && user.expires_at >= now && user.expires_at <= in3) expiring3++;
        if (user.expires_at > 0 && user.expires_at >= now && user.expires_at <= in7) expiring7++;
        if (user.quota_bytes > 0 && user.used_bytes >= user.quota_bytes * 0.9) nearQuota++;
        if (user.max_devices > 0 && user.active_ips > user.max_devices) overDevices++;
      });
      var report = {
        expired: expired,
        expiring3: expiring3,
        expiring7: expiring7,
        nearQuota: nearQuota,
        overDevices: overDevices
      };
      state.healthReport = report;
      renderHealthPanel(report);
      syncModulePanel();

      var signature = [expired, expiring3, expiring7, nearQuota, overDevices].join('|');
      if (withNotify && signature !== state.lastHealthSignature && (expired > 0 || nearQuota > 0 || overDevices > 0)) {
        toast('巡检告警：已过期 ' + expired + '，流量告警 ' + nearQuota + '，设备超限 ' + overDevices, 'error');
      }
      state.lastHealthSignature = signature;
    }
    async function applyExpiryQuickFilter(mode) {
      $('userExpiryFilter').value = mode;
      state.offset = 0;
      setActiveTab('usersView');
      await loadUsers();
    }
    async function applyAuditPreset(mode) {
      $('auditActor').value = '';
      $('auditTarget').value = '';
      if (mode === 'blocked') {
        $('auditAction').value = 'admin_login_blocked';
      } else if (mode === 'login_failed') {
        $('auditAction').value = 'admin_login_failed';
      } else if (mode === 'rate_limit') {
        $('auditAction').value = 'admin_api_rate_limited';
      } else {
        $('auditAction').value = '';
      }
      setActiveTab('auditsView');
      await loadAudits();
      await loadSecurityStats();
    }
    function csvEscape(value) {
      var text = String(value == null ? '' : value);
      if (text.indexOf('"') >= 0 || text.indexOf(',') >= 0 || text.indexOf('\n') >= 0) {
        return '"' + text.replace(/"/g, '""') + '"';
      }
      return text;
    }
    async function exportUsersCSV() {
      var users = await fetchAllUsersForExport();
      var lines = ['username,password,tag,expires_at,quota_mb,max_devices,status'];
      users.forEach(function(user) {
        lines.push([
          csvEscape(user.username),
          '',
          csvEscape(user.tag || ''),
          csvEscape(user.expires_at || 0),
          csvEscape((user.quota_bytes || 0) / 1024 / 1024),
          csvEscape(user.max_devices || 1),
          csvEscape(user.status || 0)
        ].join(','));
      });
      var blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8' });
      var url = URL.createObjectURL(blob);
      var a = document.createElement('a');
      a.href = url;
      a.download = 'users_' + Date.now() + '.csv';
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
      toast('已导出 ' + users.length + ' 条用户记录', 'success');
    }
    function parseCSVLine(line) {
      var arr = [];
      var cur = '';
      var inQuote = false;
      for (var i = 0; i < line.length; i++) {
        var ch = line[i];
        if (inQuote) {
          if (ch === '"') {
            if (i + 1 < line.length && line[i + 1] === '"') {
              cur += '"';
              i++;
            } else {
              inQuote = false;
            }
          } else {
            cur += ch;
          }
        } else {
          if (ch === ',') {
            arr.push(cur);
            cur = '';
          } else if (ch === '"') {
            inQuote = true;
          } else {
            cur += ch;
          }
        }
      }
      arr.push(cur);
      return arr;
    }
    async function importUsersCSVText(text) {
      var rows = text.replace(/\r/g, '').split('\n').filter(function(line) { return line.trim() !== ''; });
      if (rows.length < 2) throw new Error('CSV 内容为空');
      var header = parseCSVLine(rows[0]).map(function(h) { return h.trim(); });
      var idx = function(name) { return header.indexOf(name); };
      if (idx('username') < 0 || idx('password') < 0) throw new Error('CSV 必须包含 username,password 列');

      var created = 0;
      var skipped = 0;
      var failed = 0;
      for (var i = 1; i < rows.length; i++) {
        var cols = parseCSVLine(rows[i]);
        var username = (cols[idx('username')] || '').trim();
        var password = (cols[idx('password')] || '').trim();
        if (!username || !password) {
          skipped++;
          continue;
        }
        var expiresAt = Number((idx('expires_at') >= 0 ? cols[idx('expires_at')] : '0') || '0');
        var tag = (idx('tag') >= 0 ? cols[idx('tag')] : '').trim();
        var quotaMB = Number((idx('quota_mb') >= 0 ? cols[idx('quota_mb')] : '0') || '0');
        var maxDevices = Number((idx('max_devices') >= 0 ? cols[idx('max_devices')] : '1') || '1');
        var status = Number((idx('status') >= 0 ? cols[idx('status')] : '1') || '1');

        try {
          await api('/api/admin/users', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              username: username,
              password: password,
              tag: tag,
              expires_at: Number.isFinite(expiresAt) ? expiresAt : 0,
              quota_bytes: Number.isFinite(quotaMB) ? Math.round(quotaMB * 1024 * 1024) : 0,
              max_devices: Number.isInteger(maxDevices) && maxDevices > 0 ? maxDevices : 1
            })
          });
          if (tag) {
            await api('/api/admin/users/' + encodeURIComponent(username) + '/set-tag', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ tag: tag })
            });
          }
          if (status === 0) {
            await api('/api/admin/users/' + encodeURIComponent(username) + '/disable', { method: 'POST' });
          }
          created++;
        } catch (err) {
          failed++;
        }
      }
      await loadUsers();
      await loadAudits();
      toast('导入完成：新增 ' + created + '，跳过 ' + skipped + '，失败 ' + failed, failed ? 'error' : 'success');
    }
    async function batchDeleteUsers() {
      const usernames = Array.from(state.selectedUsers);
      if (!usernames.length) {
        toast('请先选择用户', 'error');
        return;
      }
      if (!await modalConfirm('批量删除', '确认删除选中的 ' + usernames.length + ' 个用户？该操作不可恢复。')) return;
      let success = 0;
      const failed = [];
      for (var i = 0; i < usernames.length; i++) {
        try {
          await api('/api/admin/users/' + encodeURIComponent(usernames[i]), { method: 'DELETE' });
          success++;
        } catch (err) {
          failed.push(usernames[i] + ': ' + err.message);
        }
      }
      state.selectedUsers.clear();
      await loadUsers();
      await loadAudits();
      if (failed.length) {
        toast('删除完成：成功 ' + success + '，失败 ' + failed.length, 'error');
      } else {
        toast('已删除 ' + success + ' 个用户', 'success');
      }
    }

    async function toggleAdmin(username, action) {
      await api('/api/admin/admins/' + encodeURIComponent(username) + '/' + action, { method: 'POST' });
      toast((action === 'disable' ? '已禁用' : '已启用') + '管理员 ' + username, 'success');
      await loadAdmins();
      await loadAudits();
    }
    async function changeAdminRole(username, role) {
      if (!await modalConfirm('调整角色', '确认把管理员 ' + username + ' 调整为 ' + role + '？')) return;
      await api('/api/admin/admins/' + encodeURIComponent(username) + '/set-role', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role })
      });
      toast('角色已更新：' + username + ' → ' + role, 'success');
      await loadAdmins();
      await loadAudits();
    }
    async function resetAdminPassword(username) {
      const password = await modalPassword('重置管理员密码：' + username);
      if (password === null) return;
      await api('/api/admin/admins/' + encodeURIComponent(username) + '/password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password })
      });
      toast('管理员密码已更新：' + username, 'success');
      await loadAdmins();
      await loadAudits();
    }
    async function deleteSession(sessionID) {
      if (!await modalConfirm('下线会话', '确认下线这个会话？')) return;
      await api('/api/admin/sessions/' + encodeURIComponent(sessionID), { method: 'DELETE' });
      toast('会话已下线', 'info');
      await loadSessions();
      await loadAudits();
    }

    async function bootstrapApp() {
      await Promise.all([loadProfile(), loadUserTags(), loadUsers(), loadAdmins(), loadSessions(), loadAudits(), loadSecurityStats(false)]);
      showApp();
      applySimpleMode(state.simpleMode);
      await runHealthCheck(false);
      syncModulePanel();
      markDataRefreshed();
      setGlobalNotice('');
    }

    document.querySelectorAll('.tab').forEach(function(button) {
      button.addEventListener('click', function() {
        setActiveTab(button.getAttribute('data-tab'));
      });
    });
    document.querySelectorAll('.op-tab').forEach(function(button) {
      button.addEventListener('click', function() {
        setActiveOpPane(button.getAttribute('data-op'));
      });
    });
    $('loginButton').addEventListener('click', login);
    $('logoutBtn').addEventListener('click', logout);
    $('refreshProfileBtn').addEventListener('click', async function() { await loadProfile(); await loadSessions(); });
    $('refreshHealthBtn').addEventListener('click', function() { runHealthCheck(false).catch(function(e){toast(e.message, 'error');}); });
    $('filterExpiredBtn').addEventListener('click', function() { applyExpiryQuickFilter('expired').catch(function(e){toast(e.message, 'error');}); });
    $('filterExpiring7Btn').addEventListener('click', function() { applyExpiryQuickFilter('expiring7').catch(function(e){toast(e.message, 'error');}); });
    $('filterPermanentBtn').addEventListener('click', function() { applyExpiryQuickFilter('permanent').catch(function(e){toast(e.message, 'error');}); });
    $('moduleGotoUsersBtn').addEventListener('click', function() { setActiveTab('usersView'); });
    $('moduleGotoAdminsBtn').addEventListener('click', function() { setActiveTab('adminsView'); });
    $('moduleGotoSessionsBtn').addEventListener('click', function() { setActiveTab('sessionsView'); });
    $('moduleGotoAuditsBtn').addEventListener('click', function() { setActiveTab('auditsView'); });
    $('moduleFilterExpiredBtn').addEventListener('click', function() { applyExpiryQuickFilter('expired').catch(function(e){toast(e.message, 'error');}); });
    $('moduleFilterExpiring7Btn').addEventListener('click', function() { applyExpiryQuickFilter('expiring7').catch(function(e){toast(e.message, 'error');}); });
    $('moduleFilterPermanentBtn').addEventListener('click', function() { applyExpiryQuickFilter('permanent').catch(function(e){toast(e.message, 'error');}); });
    $('moduleRunHealthBtn').addEventListener('click', function() { runHealthCheck(false).catch(function(e){toast(e.message, 'error');}); });
    $('moduleRefreshAllBtn').addEventListener('click', async function() {
      await Promise.all([loadProfile(), loadUserTags(), loadUsers(), loadAdmins(), loadSessions(), loadAudits(), loadSecurityStats(false)]);
      await runHealthCheck(false);
      markDataRefreshed();
      toast('模块数据已刷新', 'success');
    });
    $('moduleExportUsersBtn').addEventListener('click', function() { exportUsersCSV().catch(function(e){toast(e.message, 'error');}); });
    $('moduleFocusCreateUserBtn').addEventListener('click', function() {
      setActiveOpPane('user');
      $('createUserName').focus();
      $('createUserName').scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
    $('moduleFocusCreateAdminBtn').addEventListener('click', function() {
      setActiveOpPane('admin');
      $('createAdminName').focus();
      $('createAdminName').scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
    $('moduleFocusChangePwdBtn').addEventListener('click', function() {
      setActiveOpPane('password');
      $('oldAdminPassword').focus();
      $('oldAdminPassword').scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
    $('moduleLogoutBtn').addEventListener('click', logout);
    $('toggleSimplifyBtn').addEventListener('click', function() {
      applySimpleMode(!state.simpleMode);
      toast(state.simpleMode ? '已切换到简洁视图' : '已切换到完整视图', 'info');
    });
    $('createUserBtn').addEventListener('click', createUser);
    $('createAdminBtn').addEventListener('click', createAdmin);
    $('changeMyPasswordBtn').addEventListener('click', changeMyPassword);
    $('searchUsersBtn').addEventListener('click', async function() { state.offset = 0; await loadUsers(); });
    $('resetUsersBtn').addEventListener('click', async function() {
      $('userSearch').value = '';
      $('userStatusFilter').value = '';
      $('userTagFilter').value = '';
      $('userExpiryFilter').value = '';
      state.offset = 0;
      await loadUsers();
    });
    $('refreshUsersBtn').addEventListener('click', loadUsers);
    $('refreshAdminsBtn').addEventListener('click', loadAdmins);
    $('refreshSessionsBtn').addEventListener('click', loadSessions);
    $('refreshAuditsBtn').addEventListener('click', async function() { await loadAudits(); await loadSecurityStats(false); });
    $('auditPresetBlockedBtn').addEventListener('click', function() { applyAuditPreset('blocked').catch(function(e){toast(e.message, 'error');}); });
    $('auditPresetLoginFailBtn').addEventListener('click', function() { applyAuditPreset('login_failed').catch(function(e){toast(e.message, 'error');}); });
    $('auditPresetRateLimitBtn').addEventListener('click', function() { applyAuditPreset('rate_limit').catch(function(e){toast(e.message, 'error');}); });
    $('auditPresetClearBtn').addEventListener('click', function() { applyAuditPreset('all').catch(function(e){toast(e.message, 'error');}); });
    $('batchEnableBtn').addEventListener('click', function() { batchSetUsers(true); });
    $('batchDisableBtn').addEventListener('click', function() { batchSetUsers(false); });
    $('batchTagBtn').addEventListener('click', function() { batchSetTagUsers(); });
    $('batchExtendBtn').addEventListener('click', function() { batchExtendUsers(); });
    $('batchTopupBtn').addEventListener('click', function() { batchTopupUsers(); });
    $('batchDeleteBtn').addEventListener('click', function() { batchDeleteUsers(); });
    $('exportUsersBtn').addEventListener('click', function() { exportUsersCSV().catch(function(e){toast(e.message, 'error');}); });
    $('importUsersBtn').addEventListener('click', function() { $('importUsersFile').click(); });
    $('importUsersFile').addEventListener('change', function() {
      var file = this.files && this.files[0];
      if (!file) return;
      var reader = new FileReader();
      reader.onload = function() {
        importUsersCSVText(String(reader.result || '')).catch(function(e){toast('导入失败：' + e.message, 'error');});
      };
      reader.readAsText(file, 'utf-8');
      this.value = '';
    });
    $('clearSelectedBtn').addEventListener('click', function() {
      state.selectedUsers.clear();
      renderUsers();
      toast('已清空选择', 'info');
    });
    $('prevPageBtn').addEventListener('click', async function() {
      state.offset = Math.max(0, state.offset - state.limit);
      await loadUsers();
    });
    $('nextPageBtn').addEventListener('click', async function() {
      state.offset += state.limit;
      await loadUsers();
    });
    $('userPageSize').addEventListener('change', async function() {
      state.offset = 0;
      await loadUsers();
    });
    $('userExpiryFilter').addEventListener('change', async function() {
      state.offset = 0;
      await loadUsers();
    });
    $('userTagFilter').addEventListener('change', async function() {
      state.offset = 0;
      await loadUsers();
    });
    $('loginPassword').addEventListener('keydown', function(event) {
      if (event.key === 'Enter') login();
    });

        (function(_toggleUser, _setUserTag, _extendUserDays, _topupUserQuota, _setUserQuota, _topupUserSMBQuota, _setUserSMBQuota, _setUserSMB, _setUserSpeedLimit, _editUserDevices, _resetUserUsage, _resetUserPassword,
            _deleteUser, _viewUserAudit, _selectUserDetail, _kickUserDevice,
          _toggleAdmin, _changeAdminRole, _resetAdminPassword, _deleteSession, _toggleUserOps) {
      window.toggleUser        = function(u, a)   { _toggleUser(u, a).catch(function(e){toast(e.message,'error');}); };
      window.setUserTag        = function(u, t)   { _setUserTag(u, t).catch(function(e){toast(e.message,'error');}); };
      window.extendUserDays    = function(u)      { _extendUserDays(u).catch(function(e){toast(e.message,'error');}); };
      window.topupUserQuota    = function(u)      { _topupUserQuota(u).catch(function(e){toast(e.message,'error');}); };
      window.setUserQuota      = function(u, b)   { _setUserQuota(u, b).catch(function(e){toast(e.message,'error');}); };
          window.topupUserSMBQuota = function(u)      { _topupUserSMBQuota(u).catch(function(e){toast(e.message,'error');}); };
      window.setUserSMBQuota   = function(u, b)   { _setUserSMBQuota(u, b).catch(function(e){toast(e.message,'error');}); };
      window.setUserSMB        = function(u, e)   { _setUserSMB(u, e).catch(function(err){toast(err.message,'error');}); };
      window.setUserSpeedLimit = function(u, k)   { _setUserSpeedLimit(u, k).catch(function(e){toast(e.message,'error');}); };
      window.editUserDevices   = function(u, c)   { _editUserDevices(u, c).catch(function(e){toast(e.message,'error');}); };
      window.resetUserUsage    = function(u)      { _resetUserUsage(u).catch(function(e){toast(e.message,'error');}); };
      window.resetUserPassword = function(u)      { _resetUserPassword(u).catch(function(e){toast(e.message,'error');}); };
      window.deleteUser        = function(u)      { _deleteUser(u).catch(function(e){toast(e.message,'error');}); };
      window.viewUserAudit     = function(u)      { _viewUserAudit(u).catch(function(e){toast(e.message,'error');}); };
      window.selectUserDetail  = function(u)      { _selectUserDetail(u).catch(function(e){toast(e.message,'error');}); };
      window.kickUserDevice    = function(u, ip)  { _kickUserDevice(u, ip).catch(function(e){toast(e.message,'error');}); };
      window.toggleUserOps     = function(id, b)  { _toggleUserOps(id, b); };
      window.toggleAdmin       = function(u, a)   { _toggleAdmin(u, a).catch(function(e){toast(e.message,'error');}); };
      window.changeAdminRole   = function(u, r)   { _changeAdminRole(u, r).catch(function(e){toast(e.message,'error');}); };
      window.resetAdminPassword = function(u)     { _resetAdminPassword(u).catch(function(e){toast(e.message,'error');}); };
      window.deleteSession     = function(id)     { _deleteSession(id).catch(function(e){toast(e.message,'error');}); };
    })(toggleUser, setUserTag, extendUserDays, topupUserQuota, setUserQuota, topupUserSMBQuota, setUserSMBQuota, setUserSMB, setUserSpeedLimit, editUserDevices, resetUserUsage, resetUserPassword,
       deleteUser, viewUserAudit, selectUserDetail, kickUserDevice,
       toggleAdmin, changeAdminRole, resetAdminPassword, deleteSession, toggleUserOps);

    (async function init() {
      try {
        await bootstrapApp();
        setInterval(function() {
          if ($('appView').classList.contains('hidden')) return;
          runHealthCheck(true).catch(function() {});
          loadSecurityStats(true).catch(function() {});
        }, 60000);
      } catch (_) {
        showLogin();
      }
    })();
  </script>
</body>
</html>`

func (h *APIServer) handleAdminUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
  w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
  w.Header().Set("Pragma", "no-cache")
  w.Header().Set("Expires", "0")
  w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'; object-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
  w.Header().Set("X-Admin-UI-Version", adminUIVersion)
  if _, ok := h.authenticateAdmin(w, r); !ok {
    _, _ = w.Write([]byte(adminLoginHTML))
    return
  }
  _, _ = w.Write([]byte(adminUIHTML))
}
