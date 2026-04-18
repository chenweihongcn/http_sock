param(
    [string]$HostIP = "192.168.50.94",
    [string]$RootUser = "root",
    [string]$RootPassword = "",
    [string]$SshKeyPath = "",
  [string]$AdminUser = "admin",
    [string]$AdminPassword = "Admin2026Strong9X",
    [string]$ReadonlyUser = "ops",
    [string]$ReadonlyPassword = "OpsPass123",
    [switch]$SkipDockerStop
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) {
    Write-Host "`n=== $msg ===" -ForegroundColor Cyan
}

function Invoke-External {
  param(
    [Parameter(Mandatory = $true)][string]$FilePath,
    [Parameter(Mandatory = $false)][string[]]$Arguments = @(),
    [string]$FailMessage = "external command failed"
  )

  & $FilePath @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "$FailMessage (exit=$LASTEXITCODE): $FilePath $($Arguments -join ' ')"
  }
}

$workspace = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$localBin = Join-Path $workspace "proxy-server-openwrt-arm64"
$localImageTar = Join-Path $workspace "http_sock_arm64.tar"
$localImageName = "chenweihongcn/http_sock:arm64"
$remoteTmpBin = "/tmp/proxy-server"
$remoteBin = "/usr/local/bin/proxy-server"
$remoteEnv = "/etc/proxy-platform.env"
$remoteInit = "/etc/init.d/proxy-platform"
$remoteDbDir = "/mnt/mmc0-4/proxy-platform"
$remoteDb = "/mnt/mmc0-4/proxy-platform/proxy.db"
$oldDockerDb = "/mnt/mmc0-4/docker/http_sock_data/proxy.db"

Push-Location $workspace
try {
  if (Get-Command go -ErrorAction SilentlyContinue) {
    Write-Step "Build linux/arm64 binary"
    $env:CGO_ENABLED = "1"
    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    Invoke-External -FilePath "go" -Arguments @("build", "-o", $localBin, "./cmd/server") -FailMessage "go build failed"
  }
  elseif (Test-Path $localBin) {
    Write-Step "Use existing local binary (go not found)"
  }
  else {
    Write-Step "Extract binary from Docker image (go not found)"

    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
      throw "go not found and docker not available; cannot produce arm64 binary"
    }

    if (Test-Path $localImageTar) {
      Invoke-External -FilePath "docker" -Arguments @("load", "-i", $localImageTar) -FailMessage "docker load failed"
    }

    $containerId = (& docker create $localImageName).Trim()
    if ($LASTEXITCODE -ne 0) {
      throw "docker create failed for image: $localImageName"
    }
    if ([string]::IsNullOrWhiteSpace($containerId)) {
      throw "failed to create temp container from image: $localImageName"
    }

    try {
      Invoke-External -FilePath "docker" -Arguments @("cp", "${containerId}:/app/proxy-server", $localBin) -FailMessage "docker cp failed"
    }
    finally {
      & docker rm -f $containerId | Out-Host
    }
  }

    if (-not (Test-Path $localBin)) {
        throw "build failed: $localBin not found"
    }

    $baseScpArgs = @()
    $baseSshArgs = @()
    if ($SshKeyPath -ne "") {
        $baseScpArgs += @("-i", $SshKeyPath)
        $baseSshArgs += @("-i", $SshKeyPath)
    }

    if ($RootPassword -ne "") {
        $pscp = Join-Path $workspace "tools\pscp.exe"
        $plink = Join-Path $workspace "tools\plink.exe"
        if (-not (Test-Path $pscp) -or -not (Test-Path $plink)) {
          throw "password mode requires local tools\\pscp.exe and tools\\plink.exe (download from official PuTTY release and keep them out of git)"
        }

        Write-Step "Upload binary (password mode)"
        Invoke-External -FilePath $pscp -Arguments @("-batch", "-scp", "-pw", $RootPassword, $localBin, "${RootUser}@${HostIP}:${remoteTmpBin}") -FailMessage "upload binary failed"

        $tmpEnvLocal = Join-Path $env:TEMP ("proxy-platform.env.{0}.tmp" -f ([guid]::NewGuid().ToString("N")))
        $tmpInitLocal = Join-Path $env:TEMP ("proxy-platform.init.{0}.tmp" -f ([guid]::NewGuid().ToString("N")))
        $utf8NoBom = New-Object System.Text.UTF8Encoding($false)

        $envContent = @"
LISTEN_HOST=0.0.0.0
HTTP_PORT=8899
SOCKS5_PORT=1080
ADMIN_PORT=8088
CONTROL_PLANE_ENABLED=true
DB_PATH=${remoteDb}
SMB_ROOT_DIR=/mnt/mmc0-4/proxy-platform/smb
DEVICE_WINDOW=10m
BOOTSTRAP_ADMIN_USER=${AdminUser}
BOOTSTRAP_ADMIN_PASS=${AdminPassword}
BOOTSTRAP_READONLY=${ReadonlyUser}
BOOTSTRAP_READONLY_PASS=${ReadonlyPassword}
ADMIN_SESSION_TTL=12h
PASSWORD_MIN_LENGTH=8
TRUST_PROXY_HEADERS=true
REAL_IP_HEADER=X-Forwarded-For
"@

        $initContent = @'
#!/bin/sh /etc/rc.common
START=99
STOP=10
USE_PROCD=1
NAME=proxy-platform
BIN=/usr/local/bin/proxy-server
ENVFILE=/etc/proxy-platform.env

start_service() {
  [ -x $BIN ] || return 1
  [ -f $ENVFILE ] || return 1
  [ -d /mnt/mmc0-4/proxy-platform ] || return 1

  procd_open_instance
  procd_set_param command /bin/sh -c "set -a; . $ENVFILE; set +a; exec $BIN"
  procd_set_param respawn 3600 5 5
  procd_set_param limits nofile=65535
  procd_close_instance
}
'@

        [System.IO.File]::WriteAllText($tmpEnvLocal, $envContent.Replace("`r`n", "`n"), $utf8NoBom)
        [System.IO.File]::WriteAllText($tmpInitLocal, $initContent.Replace("`r`n", "`n"), $utf8NoBom)

        Write-Step "Upload service files (password mode)"
        Invoke-External -FilePath $pscp -Arguments @("-batch", "-scp", "-pw", $RootPassword, $tmpEnvLocal, "${RootUser}@${HostIP}:/tmp/proxy-platform.env") -FailMessage "upload env failed"
        Invoke-External -FilePath $pscp -Arguments @("-batch", "-scp", "-pw", $RootPassword, $tmpInitLocal, "${RootUser}@${HostIP}:/tmp/proxy-platform.init") -FailMessage "upload init script failed"

        $setupCmd = "mkdir -p ${remoteDbDir}; mkdir -p /usr/local/bin; cp -f ${remoteTmpBin} ${remoteBin}; chmod 755 ${remoteBin}; if [ -f ${oldDockerDb} ] && [ ! -f ${remoteDb} ]; then cp -a ${oldDockerDb} ${remoteDb}; fi; cp -f /tmp/proxy-platform.env ${remoteEnv}; cp -f /tmp/proxy-platform.init ${remoteInit}; chmod 755 ${remoteInit}"

        Write-Step "Install binary and configure service"
        Invoke-External -FilePath $plink -Arguments @("-batch", "-pw", $RootPassword, "${RootUser}@${HostIP}", $setupCmd) -FailMessage "remote setup failed"

        Remove-Item $tmpEnvLocal, $tmpInitLocal -ErrorAction SilentlyContinue

        $dockerCutover = if ($SkipDockerStop) { "true" } else { "docker stop lowres-proxy 2>/dev/null || true; docker rm lowres-proxy 2>/dev/null || true" }
        $cutoverCmd = "$dockerCutover; ${remoteInit} enable; ${remoteInit} restart; sleep 2; echo '--- listening ports ---'; netstat -lntp 2>/dev/null | grep -E ':8899|:1080|:8088' || true; echo '--- service status ---'; ${remoteInit} status || true; echo '--- db file ---'; ls -lh ${remoteDb} || true"

        Write-Step "Cut over to native service"
        Invoke-External -FilePath $plink -Arguments @("-batch", "-pw", $RootPassword, "${RootUser}@${HostIP}", $cutoverCmd) -FailMessage "remote cutover failed"
    }
    else {
      Write-Step "Upload binary"
      Invoke-External -FilePath "scp" -Arguments @($baseScpArgs + @($localBin, "${RootUser}@${HostIP}:${remoteTmpBin}")) -FailMessage "upload binary failed"

        $setupCmd = @"
mkdir -p ${remoteDbDir}
mkdir -p /usr/local/bin
install -m 0755 ${remoteTmpBin} ${remoteBin}

if [ -f ${oldDockerDb} ] && [ ! -f ${remoteDb} ]; then
  cp -a ${oldDockerDb} ${remoteDb}
fi

cat > ${remoteEnv} <<'EOF'
LISTEN_HOST=0.0.0.0
HTTP_PORT=8899
SOCKS5_PORT=1080
ADMIN_PORT=8088
CONTROL_PLANE_ENABLED=true
DB_PATH=${remoteDb}
SMB_ROOT_DIR=/mnt/mmc0-4/proxy-platform/smb
DEVICE_WINDOW=10m
BOOTSTRAP_ADMIN_USER=${AdminUser}
BOOTSTRAP_ADMIN_PASS=${AdminPassword}
BOOTSTRAP_READONLY=${ReadonlyUser}
BOOTSTRAP_READONLY_PASS=${ReadonlyPassword}
ADMIN_SESSION_TTL=12h
PASSWORD_MIN_LENGTH=8
TRUST_PROXY_HEADERS=true
REAL_IP_HEADER=X-Forwarded-For
EOF

cat > ${remoteInit} <<'EOF'
#!/bin/sh /etc/rc.common
START=99
STOP=10
USE_PROCD=1
NAME=proxy-platform
BIN=/usr/local/bin/proxy-server
ENVFILE=/etc/proxy-platform.env

start_service() {
  [ -x `$BIN ] || return 1
  [ -f `$ENVFILE ] || return 1
  [ -d /mnt/mmc0-4/proxy-platform ] || return 1

  procd_open_instance
  procd_set_param command /bin/sh -c "set -a; . `$ENVFILE; set +a; exec `$BIN"
  procd_set_param respawn 3600 5 5
  procd_set_param limits nofile=65535
  procd_close_instance
}
EOF

chmod 755 ${remoteInit}
"@

        Write-Step "Install binary and configure service"
        Invoke-External -FilePath "ssh" -Arguments @($baseSshArgs + @("${RootUser}@${HostIP}", $setupCmd)) -FailMessage "remote setup failed"

        $dockerCutover = if ($SkipDockerStop) { "true" } else { "docker stop lowres-proxy 2>/dev/null || true; docker rm lowres-proxy 2>/dev/null || true" }
        $cutoverCmd = "$dockerCutover; ${remoteInit} enable; ${remoteInit} restart; sleep 2; echo '--- listening ports ---'; netstat -lntp 2>/dev/null | grep -E ':8899|:1080|:8088' || true; echo '--- service status ---'; ${remoteInit} status || true; echo '--- db file ---'; ls -lh ${remoteDb} || true"

        Write-Step "Cut over to native service"
        Invoke-External -FilePath "ssh" -Arguments @($baseSshArgs + @("${RootUser}@${HostIP}", $cutoverCmd)) -FailMessage "remote cutover failed"
    }

    Write-Step "Done"
    Write-Host "Native deployment finished. Validate with: python .\\test_realip_layout.py" -ForegroundColor Green
}
finally {
    Pop-Location
}
