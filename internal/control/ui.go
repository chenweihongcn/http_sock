package control

import "net/http"

const adminUIHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>代理管理台</title>
  <style>
    :root {
      --bg: #edf3fb;
      --panel: rgba(255, 255, 255, 0.94);
      --line: #d9e3f1;
      --text: #1b2636;
      --muted: #6b778a;
      --primary: #1862f7;
      --primary-soft: #eaf1ff;
      --danger: #d04343;
      --danger-soft: #fff1f1;
      --success: #16814e;
      --shadow: 0 18px 44px rgba(43, 72, 124, 0.08);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      background:
        radial-gradient(circle at top left, rgba(24, 98, 247, 0.14), transparent 28%),
        linear-gradient(180deg, #e7f0ff 0, var(--bg) 320px);
      color: var(--text);
    }
    .wrap { max-width: 1380px; margin: 0 auto; padding: 24px; }
    .hero {
      background: linear-gradient(135deg, #175df1, #5ca8ff);
      color: #fff;
      border-radius: 22px;
      padding: 24px 26px;
      box-shadow: 0 22px 56px rgba(23, 93, 241, 0.22);
      margin-bottom: 18px;
    }
    .hero h1 { margin: 0 0 8px; font-size: 30px; }
    .hero p { margin: 0; opacity: .92; }
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
      grid-template-columns: repeat(4, minmax(0, 1fr));
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
    th { color: var(--muted); font-weight: 600; background: #fbfcff; }
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
    <div class="hero">
      <h1>HTTP + SOCKS5 管理台</h1>
      <p>管理员使用账号密码登录。支持用户搜索分页、批量启停、批量删除、延期、配额充值、改密、删号、流量重置、活跃设备查看与踢出。</p>
    </div>

    <div id="loginView" class="auth-shell">
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
              <div class="group"><span class="badge">当前账号在线会话</span></div>
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
              <div class="group"><button class="secondary" id="refreshAuditsBtn">刷新审计</button></div>
            </div>
            <div id="auditsTableWrap"></div>
          </div>
        </div>

        <div class="stack">
          <div class="panel">
            <h3>操作台：创建代理用户</h3>
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
            <div class="actions">
              <button id="createUserBtn">创建用户</button>
            </div>
            <div class="status" id="createUserStatus"></div>
          </div>

          <div class="panel">
            <h3>操作台：创建管理员</h3>
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

          <div class="panel">
            <h3>操作台：修改当前管理员密码</h3>
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

  <script>
    const state = {
      me: null,
      users: [],
      admins: [],
      sessions: [],
      audits: [],
      userTags: [],
      selectedUsers: new Set(),
      selectedUser: '',
      lastHealthSignature: '',
      healthReport: { expired: 0, expiring3: 0, expiring7: 0, nearQuota: 0, overDevices: 0 },
      offset: 0,
      limit: 20,
      total: 0,
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
    async function api(path, options) {
      const res = await fetch(path, Object.assign({ credentials: 'same-origin' }, options || {}));
      const text = await res.text();
      let data = {};
      try { data = text ? JSON.parse(text) : {}; } catch (_) { data = { raw: text }; }
      if (res.status === 401) {
        showLogin();
        throw new Error('请重新登录');
      }
      if (!res.ok) {
        throw new Error(data.error || data.message || res.statusText || 'request_failed');
      }
      return data;
    }
    function showLogin() {
      $('loginView').classList.remove('hidden');
      $('appView').classList.add('hidden');
    }
    function showApp() {
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
    }

    async function login() {
      const username = $('loginUsername').value.trim();
      const password = $('loginPassword').value.trim();
      if (!username || !password) {
        setStatus('loginStatus', '账号和密码不能为空', 'error');
        return;
      }
      try {
        await api('/api/admin/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password })
        });
        setStatus('loginStatus', '登录成功', 'success');
        await bootstrapApp();
      } catch (err) {
        setStatus('loginStatus', '登录失败：' + err.message, 'error');
      }
    }

    async function logout() {
      try { await api('/api/admin/logout', { method: 'POST' }); } catch (_) {}
      state.me = null;
      state.sessions = [];
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
          '<button class="ghost" onclick="selectUserDetail(\'' + escapeHTML(user.username) + '\')">详情</button>' +
          '<button class="ghost" onclick="toggleUser(\'' + escapeHTML(user.username) + '\', \'' + action + '\')">' + actionLabel + '</button>' +
          '<button class="secondary" onclick="toggleUserOps(\'' + panelId + '\', this)">更多</button>' +
          '</div>' +
          '<div class="user-op-more hidden" id="' + panelId + '">' +
          '<button class="ghost" onclick="setUserTag(\'' + escapeHTML(user.username) + '\', \'' + escapeHTML(user.tag || '') + '\')">标签</button>' +
          '<button class="ghost" onclick="extendUserDays(\'' + escapeHTML(user.username) + '\')">延期</button>' +
          '<button class="ghost" onclick="topupUserQuota(\'' + escapeHTML(user.username) + '\')">充值</button>' +
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
        '<div class="note">标签：<span class="badge">' + escapeHTML(user.tag || '-') + '</span></div>' +
        '<div class="note">创建时间：' + fmtTime(user.created_at) + '</div>' +
        '<div class="note">到期时间：' + fmtTime(user.expires_at) + '，更新时间：' + fmtTime(user.updated_at) + '</div>' +
        '<div class="note">IP来源说明：' + ipMode + '</div>';
      html += '<h4>活跃设备</h4>';
      if (!devices.length) {
        html += '<p class="note">当前没有活跃设备。</p>';
      } else {
        html += '<table><thead><tr><th>客户端IP</th><th>最后活跃</th><th>操作</th></tr></thead><tbody>';
        devices.forEach(function(device) {
          html += '<tr><td class="mono">' + escapeHTML(device.ip) + '</td><td>' + fmtTime(device.last_seen) + '</td>' +
            '<td><button class="danger" onclick="kickUserDevice(\'' + escapeHTML(user.username) + '\', \'' + escapeHTML(device.ip) + '\')">踢出</button></td></tr>';
        });
        html += '</tbody></table>';
      }
      html += '<h4>最近操作记录</h4>';
      if (!audits || !audits.length) {
        html += '<p class="note">暂无与该用户相关的操作日志。</p>';
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
      if (!state.sessions.length) {
        $('sessionsTableWrap').innerHTML = '<p class="note">暂无在线会话。</p>';
        return;
      }
      let html = '<table><thead><tr><th>会话ID</th><th>设备类型</th><th>设备名称</th><th>登录IP</th><th>原始登录IP</th><th>创建时间</th><th>最后活跃</th><th>到期</th><th>操作</th></tr></thead><tbody>';
      state.sessions.forEach(function(item) {
        const deviceInfo = detectDeviceInfo(item.user_agent);
        const displayIP = item.ip_address || '-';
        const originalIP = item.original_ip || item.ip_address || '-';
        html += '<tr>' +
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
      if (!username || !password) {
        setStatus('createUserStatus', '用户名和密码不能为空', 'error');
        return;
      }
      try {
        await api('/api/admin/users', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password, max_devices: maxDevices, quota_bytes: quotaMB * 1024 * 1024, expires_at: 0 })
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
      await Promise.all([loadProfile(), loadUserTags(), loadUsers(), loadAdmins(), loadSessions(), loadAudits()]);
      showApp();
      await runHealthCheck(false);
      syncModulePanel();
    }

    document.querySelectorAll('.tab').forEach(function(button) {
      button.addEventListener('click', function() {
        setActiveTab(button.getAttribute('data-tab'));
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
      await Promise.all([loadProfile(), loadUserTags(), loadUsers(), loadAdmins(), loadSessions(), loadAudits()]);
      await runHealthCheck(false);
      toast('模块数据已刷新', 'success');
    });
    $('moduleExportUsersBtn').addEventListener('click', function() { exportUsersCSV().catch(function(e){toast(e.message, 'error');}); });
    $('moduleFocusCreateUserBtn').addEventListener('click', function() {
      $('createUserName').focus();
      $('createUserName').scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
    $('moduleFocusCreateAdminBtn').addEventListener('click', function() {
      $('createAdminName').focus();
      $('createAdminName').scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
    $('moduleFocusChangePwdBtn').addEventListener('click', function() {
      $('oldAdminPassword').focus();
      $('oldAdminPassword').scrollIntoView({ behavior: 'smooth', block: 'center' });
    });
    $('moduleLogoutBtn').addEventListener('click', logout);
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
    $('refreshAuditsBtn').addEventListener('click', loadAudits);
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

        (function(_toggleUser, _setUserTag, _extendUserDays, _topupUserQuota, _editUserDevices, _resetUserUsage, _resetUserPassword,
            _deleteUser, _viewUserAudit, _selectUserDetail, _kickUserDevice,
          _toggleAdmin, _changeAdminRole, _resetAdminPassword, _deleteSession, _toggleUserOps) {
      window.toggleUser        = function(u, a)   { _toggleUser(u, a).catch(function(e){toast(e.message,'error');}); };
      window.setUserTag        = function(u, t)   { _setUserTag(u, t).catch(function(e){toast(e.message,'error');}); };
      window.extendUserDays    = function(u)      { _extendUserDays(u).catch(function(e){toast(e.message,'error');}); };
      window.topupUserQuota    = function(u)      { _topupUserQuota(u).catch(function(e){toast(e.message,'error');}); };
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
    })(toggleUser, setUserTag, extendUserDays, topupUserQuota, editUserDevices, resetUserUsage, resetUserPassword,
       deleteUser, viewUserAudit, selectUserDetail, kickUserDevice,
       toggleAdmin, changeAdminRole, resetAdminPassword, deleteSession, toggleUserOps);

    (async function init() {
      try {
        await bootstrapApp();
        setInterval(function() {
          if ($('appView').classList.contains('hidden')) return;
          runHealthCheck(true).catch(function() {});
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
	_, _ = w.Write([]byte(adminUIHTML))
}
