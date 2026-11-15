#requires -Version 5.1
<#
.SYNOPSIS
  Launcher FixPhone untuk Windows: menjalankan database, API, dan frontend
  tanpa Docker.

.DESCRIPTION
  Perintah:
    start    - pastikan .env, PostgreSQL portable, database, migrasi, seed,
               frontend, dan proses API/Vite berjalan lalu buka browser.
    stop     - hentikan API, frontend, dan PostgreSQL portable.
    restart  - stop lalu start.
    status   - tampilkan status database, API, dan frontend.
    logs     - tampilkan log terakhir (API, frontend, PostgreSQL).
    shortcut - buat shortcut "FixPhone" di Desktop.

  Proses dijalankan di background dengan PID dan log di .dev/run sehingga
  jendela terminal boleh ditutup.

.EXAMPLE
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts/fixphone.ps1 start
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts/fixphone.ps1 status
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts/fixphone.ps1 start -TenantName "Toko Saya" -OwnerEmail owner@toko.id -OwnerPassword "Rahasia123"
#>
[CmdletBinding()]
param(
  [ValidateSet('start', 'stop', 'restart', 'status', 'logs', 'shortcut')]
  [string]$Action = 'start',

  [switch]$NoBrowser,
  [switch]$Follow,

  [string]$TenantName,
  [string]$TenantSlug,
  [string]$OwnerName,
  [string]$OwnerEmail,
  [string]$OwnerPassword
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$RepoRoot = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RepoRoot 'backend'
$FrontendDir = Join-Path $RepoRoot 'frontend'
$DevDir = Join-Path $RepoRoot '.dev'
$RunDir = Join-Path $DevDir 'run'
$BinDir = Join-Path $DevDir 'bin'
$EnvFile = Join-Path $RepoRoot '.env'
$EnvExample = Join-Path $RepoRoot '.env.example'
$DevDbScript = Join-Path $PSScriptRoot 'dev-db.ps1'

$ApiPidFile = Join-Path $RunDir 'api.pid'
$WebPidFile = Join-Path $RunDir 'web.pid'
$ApiLogFile = Join-Path $RunDir 'api.log'
$WebLogFile = Join-Path $RunDir 'web.log'
$ApiErrFile = Join-Path $RunDir 'api.err.log'
$WebErrFile = Join-Path $RunDir 'web.err.log'
$PostgresLogFile = Join-Path $DevDir 'postgres.log'

function Write-Step([string]$Message) { Write-Host "==> $Message" -ForegroundColor Cyan }
function Write-Ok([string]$Message) { Write-Host "  OK  $Message" -ForegroundColor Green }
function Write-Warn([string]$Message) { Write-Host "  !!  $Message" -ForegroundColor Yellow }
function Write-Err([string]$Message) { Write-Host "  XX  $Message" -ForegroundColor Red }

function Ensure-RunDirs {
  New-Item -ItemType Directory -Force -Path $RunDir | Out-Null
  New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
}

function New-RandomToken([int]$Length) {
  $chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  $bytes = New-Object byte[] $Length
  $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
  try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
  $sb = New-Object System.Text.StringBuilder
  foreach ($b in $bytes) { [void]$sb.Append($chars[$b % $chars.Length]) }
  return $sb.ToString()
}

function New-RandomBase64Key {
  $bytes = New-Object byte[] 32
  $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
  try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
  return [Convert]::ToBase64String($bytes)
}

function Read-DotEnv {
  $map = @{}
  if (-not (Test-Path $EnvFile)) { return $map }
  foreach ($line in Get-Content $EnvFile) {
    $trimmed = $line.Trim()
    if ($trimmed -eq '' -or $trimmed.StartsWith('#')) { continue }
    $idx = $trimmed.IndexOf('=')
    if ($idx -lt 1) { continue }
    $key = $trimmed.Substring(0, $idx).Trim()
    $value = $trimmed.Substring($idx + 1).Trim()
    if ($value.Length -ge 2) {
      $first = $value.Substring(0, 1)
      $last = $value.Substring($value.Length - 1, 1)
      if (($first -eq '"' -and $last -eq '"') -or ($first -eq "'" -and $last -eq "'")) {
        $value = $value.Substring(1, $value.Length - 2)
      }
    }
    $map[$key] = $value
  }
  return $map
}

function Set-DotEnvValue([string]$Key, [string]$Value) {
  $lines = @()
  if (Test-Path $EnvFile) { $lines = @(Get-Content $EnvFile) }
  $found = $false
  for ($i = 0; $i -lt $lines.Count; $i++) {
    if ($lines[$i] -match "^$([regex]::Escape($Key))=") {
      $lines[$i] = "$Key=$Value"
      $found = $true
      break
    }
  }
  if (-not $found) { $lines += "$Key=$Value" }
  # UTF-8 without BOM so the first key is still parsed by godotenv.
  [System.IO.File]::WriteAllLines($EnvFile, [string[]]$lines)
}

function Initialize-EnvFile {
  if (Test-Path $EnvFile) { return }
  if (-not (Test-Path $EnvExample)) { throw "Templat .env tidak ditemukan: $EnvExample" }
  Write-Step 'Membuat .env dari .env.example dengan secret acak'
  Copy-Item $EnvExample $EnvFile
  Set-DotEnvValue 'JWT_SECRET' (New-RandomToken 48)
  Set-DotEnvValue 'APP_ENC_KEY' (New-RandomBase64Key)
  Set-DotEnvValue 'SEED_SUPERADMIN_PASSWORD' (New-RandomToken 20)
  Write-Ok "File .env dibuat: $EnvFile"
}

function Test-PortListening([int]$Port) {
  try {
    return (Test-NetConnection -ComputerName '127.0.0.1' -Port $Port -InformationLevel Quiet -WarningAction SilentlyContinue)
  } catch {
    return $false
  }
}

function Wait-Http([string]$Url, [int]$TimeoutSeconds) {
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 5
      if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) { return $true }
    } catch {
      # Not ready yet.
    }
    Start-Sleep -Milliseconds 700
  }
  return $false
}

function Invoke-DevDb([string]$DbAction) {
  & powershell -NoProfile -ExecutionPolicy Bypass -File $DevDbScript $DbAction
  if ($LASTEXITCODE -ne 0) { throw "scripts/dev-db.ps1 $DbAction gagal (exit $LASTEXITCODE)" }
}

function Ensure-PostgresBinaries {
  $pgCtl = Join-Path $DevDir 'pgsql\bin\pg_ctl.exe'
  if (Test-Path $pgCtl) { return }
  Write-Step 'PostgreSQL portable belum ada; mengunduh (sekali saja, ~22 MB)'
  Invoke-DevDb 'download'
}

function Invoke-GoBuild([string]$Output, [string]$Package) {
  Write-Step "Membangun $Package"
  Push-Location $BackendDir
  try {
    & go build -o $Output $Package
    if ($LASTEXITCODE -ne 0) { throw "go build $Package gagal" }
  } finally {
    Pop-Location
  }
}

function Invoke-Cli([string]$Command, [string[]]$CommandArgs) {
  $cliExe = Join-Path $BinDir 'fixphone-cli.exe'
  # Out-Host keeps native stdout out of the function's return value.
  & $cliExe $Command @CommandArgs | Out-Host
  return $LASTEXITCODE
}

function Get-PortablePsql {
  $candidates = @(
    (Join-Path $DevDir 'pgsql\bin\psql.exe'),
    (Join-Path $DevDir 'tools\pgsql-client')
  )
  if (Test-Path $candidates[0]) { return $candidates[0] }
  $clientDir = $candidates[1]
  if (Test-Path $clientDir) {
    $found = Get-ChildItem $clientDir -Recurse -Filter 'psql.exe' -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($found) { return $found.FullName }
  }
  return $null
}

function Get-TenantCount([string]$Dsn, [string]$Psql) {
  if (-not $Psql) { return -1 }
  try {
    $result = & $Psql $Dsn -tAc 'SELECT count(*) FROM tenants' 2>$null
    if ($LASTEXITCODE -ne 0) { return -1 }
    return [int]($result | Select-Object -First 1)
  } catch {
    return -1
  }
}

function ConvertTo-Slug([string]$Text) {
  $slug = $Text.ToLowerInvariant()
  $slug = [regex]::Replace($slug, '[^a-z0-9]+', '-')
  $slug = $slug.Trim('-')
  if ($slug -eq '') { $slug = 'toko-' + (Get-Date -Format 'yyyyMMddHHmmss') }
  return $slug
}

function New-FirstTenant([string]$Dsn) {
  $marker = Join-Path $RunDir 'tenant.ready'
  $psql = Get-PortablePsql
  $count = Get-TenantCount $Dsn $psql
  if ($count -gt 0) {
    if (-not (Test-Path $marker)) { Set-Content -Path $marker -Value (Get-Date -Format o) }
    Write-Ok "Tenant sudah ada ($count); tidak membuat data baru"
    return
  }
  if ($count -lt 0 -and (Test-Path $marker)) {
    Write-Ok 'Tenant sudah pernah disiapkan; lewati pemeriksaan ulang'
    return
  }

  $name = $TenantName
  $email = $OwnerEmail
  $password = $OwnerPassword
  $owner = $OwnerName

  if ([string]::IsNullOrWhiteSpace($name)) {
    Write-Host ''
    Write-Host 'Belum ada tenant. Buat toko pertama Anda:' -ForegroundColor Cyan
    $name = Read-Host '  Nama toko [Toko Saya]'
    if ([string]::IsNullOrWhiteSpace($name)) { $name = 'Toko Saya' }
    $email = Read-Host '  Email owner [owner@toko.local]'
    if ([string]::IsNullOrWhiteSpace($email)) { $email = 'owner@toko.local' }
    if ([string]::IsNullOrWhiteSpace($owner)) {
      $owner = Read-Host '  Nama owner [Owner Toko]'
      if ([string]::IsNullOrWhiteSpace($owner)) { $owner = 'Owner Toko' }
    }
    if ([string]::IsNullOrWhiteSpace($password)) {
      $password = Read-Host '  Password owner (min. 8 karakter, huruf + angka; kosong = acak)'
    }
  }

  if ([string]::IsNullOrWhiteSpace($owner)) { $owner = 'Owner Toko' }
  if ([string]::IsNullOrWhiteSpace($password)) { $password = New-RandomToken 14 }
  if ([string]::IsNullOrWhiteSpace($email)) { $email = 'owner@toko.local' }
  $slug = $TenantSlug
  if ([string]::IsNullOrWhiteSpace($slug)) { $slug = ConvertTo-Slug $name }

  Write-Step "Membuat tenant '$name' ($slug) dengan owner $email"
  $code = Invoke-Cli 'create-tenant' @(
    '--name', $name,
    '--slug', $slug,
    '--owner-name', $owner,
    '--owner-email', $email,
    '--owner-password', $password,
    '--outlet-name', 'Outlet Utama',
    '--outlet-code', 'MAIN'
  )
  if ($code -ne 0) {
    Write-Warn 'create-tenant gagal; lihat pesan di atas (mungkin slug/email sudah dipakai)'
    return
  }
  Set-Content -Path $marker -Value (Get-Date -Format o)
  Write-Ok "Tenant dibuat. Login: $email / $password"
}

function Test-AlivePid([string]$PidFile) {
  if (-not (Test-Path $PidFile)) { return $false }
  $raw = (Get-Content $PidFile -ErrorAction SilentlyContinue | Select-Object -First 1)
  if (-not $raw) { Remove-Item -Force $PidFile -ErrorAction SilentlyContinue; return $false }
  $process = Get-Process -Id ([int]$raw) -ErrorAction SilentlyContinue
  if ($process) { return $true }
  Remove-Item -Force $PidFile -ErrorAction SilentlyContinue
  return $false
}

function Stop-FromPidFile([string]$PidFile, [string]$Label) {
  if (-not (Test-Path $PidFile)) {
    Write-Warn "$Label tidak berjalan (pid file tidak ada)"
    return
  }
  $raw = (Get-Content $PidFile -ErrorAction SilentlyContinue | Select-Object -First 1)
  if ($raw) {
    $process = Get-Process -Id ([int]$raw) -ErrorAction SilentlyContinue
    if ($process) {
      Stop-Process -Id $process.Id -Force
      Write-Ok "$Label dihentikan (pid $($process.Id))"
    } else {
      Write-Warn "$Label sudah tidak berjalan (pid $raw)"
    }
  }
  Remove-Item -Force $PidFile -ErrorAction SilentlyContinue
}

function Test-ApiReady([int]$Port) {
  try {
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:$Port/readyz" -UseBasicParsing -TimeoutSec 5
    return ($response.StatusCode -eq 200)
  } catch {
    return $false
  }
}

function Start-FixPhone {
  Ensure-RunDirs
  Initialize-EnvFile
  $envMap = Read-DotEnv

  $apiPort = 8080
  if ($envMap.ContainsKey('APP_PORT') -and $envMap['APP_PORT']) { $apiPort = [int]$envMap['APP_PORT'] }
  $webPort = 5173
  $dsn = $envMap['DB_DSN']
  if ([string]::IsNullOrWhiteSpace($dsn)) { throw 'DB_DSN tidak ada di .env' }

  if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go tidak ditemukan di PATH. Pasang Go 1.26+ dari https://go.dev/dl/'
  }
  if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    throw 'Node.js tidak ditemukan di PATH. Pasang Node.js 20+ dari https://nodejs.org/'
  }

  Write-Step 'Menyiapkan database portable'
  Ensure-PostgresBinaries
  Invoke-DevDb 'start'

  Write-Step 'Menyiapkan skema dan data sistem'
  Invoke-GoBuild (Join-Path $BinDir 'fixphone-cli.exe') './cmd/cli'
  if ((Invoke-Cli 'db-setup' @()) -ne 0) { throw 'db-setup gagal' }
  if ((Invoke-Cli 'migrate' @('up')) -ne 0) { throw 'migrate up gagal' }
  if ((Invoke-Cli 'seed' @()) -ne 0) { throw 'seed gagal' }
  Write-Ok 'Database, migrasi, dan seed siap'

  New-FirstTenant $dsn

  if (Test-AlivePid $ApiPidFile) {
    Write-Ok "API sudah berjalan (pid $(Get-Content $ApiPidFile | Select-Object -First 1))"
  } else {
    if (Test-PortListening $apiPort) {
      throw "Port $apiPort sudah dipakai proses lain. Hentikan proses itu atau jalankan: scripts/fixphone.ps1 stop"
    }
    $apiExe = Join-Path $BinDir 'fixphone-api.exe'
    Invoke-GoBuild $apiExe './cmd/api'
    Write-Step "Menjalankan API di http://127.0.0.1:$apiPort"
    $apiProcess = Start-Process -FilePath $apiExe -WorkingDirectory $BackendDir `
      -RedirectStandardOutput $ApiLogFile -RedirectStandardError $ApiErrFile `
      -WindowStyle Hidden -PassThru
    Set-Content -Path $ApiPidFile -Value $apiProcess.Id
    if (-not (Wait-Http "http://127.0.0.1:$apiPort/readyz" 90)) {
      Write-Warn "API belum siap. Lihat log: $ApiLogFile"
    } else {
      Write-Ok "API siap (pid $($apiProcess.Id))"
    }
  }

  if (Test-AlivePid $WebPidFile) {
    Write-Ok "Frontend sudah berjalan (pid $(Get-Content $WebPidFile | Select-Object -First 1))"
  } else {
    if (Test-PortListening $webPort) {
      throw "Port $webPort sudah dipakai proses lain. Hentikan proses itu atau jalankan: scripts/fixphone.ps1 stop"
    }
    $viteEntry = Join-Path $FrontendDir 'node_modules\vite\bin\vite.js'
    if (-not (Test-Path $viteEntry)) {
      Write-Step 'Memasang dependency frontend (npm ci, sekali saja)'
      Push-Location $FrontendDir
      try {
        & npm ci --no-audit --no-fund
        if ($LASTEXITCODE -ne 0) { throw 'npm ci gagal' }
      } finally {
        Pop-Location
      }
    }
    Write-Step "Menjalankan frontend di http://127.0.0.1:$webPort"
    $webProcess = Start-Process -FilePath 'node' `
      -ArgumentList @($viteEntry, '--host', '127.0.0.1', '--port', "$webPort", '--strictPort') `
      -WorkingDirectory $FrontendDir `
      -RedirectStandardOutput $WebLogFile -RedirectStandardError $WebErrFile `
      -WindowStyle Hidden -PassThru
    Set-Content -Path $WebPidFile -Value $webProcess.Id
    if (-not (Wait-Http "http://127.0.0.1:$webPort/" 90)) {
      Write-Warn "Frontend belum siap. Lihat log: $WebLogFile"
    } else {
      Write-Ok "Frontend siap (pid $($webProcess.Id))"
    }
  }

  $url = "http://127.0.0.1:$webPort"
  Write-Host ''
  Write-Host 'FixPhone berjalan' -ForegroundColor Green
  Write-Host "  Aplikasi : $url"
  Write-Host "  API      : http://127.0.0.1:$apiPort"
  Write-Host "  Log API  : $ApiLogFile"
  Write-Host "  Log web  : $WebLogFile"
  Write-Host "  Hentikan : powershell -NoProfile -ExecutionPolicy Bypass -File scripts/fixphone.ps1 stop"
  Write-Host ''

  if (-not $NoBrowser) {
    Start-Process $url
  }
}

function Stop-FixPhone {
  Write-Step 'Menghentikan FixPhone'
  Stop-FromPidFile $WebPidFile 'Frontend'
  Stop-FromPidFile $ApiPidFile 'API'
  if (Test-Path $DevDbScript) { Invoke-DevDb 'stop' }
  Write-Ok 'Semua proses dihentikan'
}

function Show-Status {
  $envMap = Read-DotEnv
  $apiPort = 8080
  if ($envMap.ContainsKey('APP_PORT') -and $envMap['APP_PORT']) { $apiPort = [int]$envMap['APP_PORT'] }
  $webPort = 5173

  Write-Host 'Status FixPhone' -ForegroundColor Cyan

  $pgCtl = Join-Path $DevDir 'pgsql\bin\pg_ctl.exe'
  if (Test-Path $pgCtl) {
    $null = & $pgCtl -D (Join-Path $DevDir 'pgdata') status 2>&1
    if ($LASTEXITCODE -eq 0) { Write-Ok 'PostgreSQL berjalan (127.0.0.1:55432)' } else { Write-Warn 'PostgreSQL berhenti' }
  } else {
    Write-Warn 'PostgreSQL portable belum diunduh'
  }

  if (Test-ApiReady $apiPort) {
    Write-Ok "API siap di http://127.0.0.1:$apiPort"
  } else {
    Write-Warn "API tidak merespons /readyz di port $apiPort"
  }

  try {
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:$webPort/" -UseBasicParsing -TimeoutSec 5
    if ($response.StatusCode -eq 200) { Write-Ok "Frontend berjalan di http://127.0.0.1:$webPort" }
    else { Write-Warn "Frontend merespons status $($response.StatusCode)" }
  } catch {
    Write-Warn "Frontend tidak merespons di port $webPort"
  }
}

function Show-Logs {
  foreach ($item in @(
      @{ Label = 'API'; Path = $ApiLogFile },
      @{ Label = 'API error'; Path = $ApiErrFile },
      @{ Label = 'Frontend'; Path = $WebLogFile },
      @{ Label = 'Frontend error'; Path = $WebErrFile },
      @{ Label = 'PostgreSQL'; Path = $PostgresLogFile }
    )) {
    if (-not (Test-Path $item.Path)) { continue }
    Write-Host ''
    Write-Host "--- $($item.Label) ($($item.Path)) ---" -ForegroundColor Cyan
    Get-Content $item.Path -Tail 40
  }
  if ($Follow) {
    Write-Host ''
    Write-Host "--- mengikuti $ApiLogFile (Ctrl+C untuk berhenti) ---" -ForegroundColor Cyan
    Get-Content $ApiLogFile -Tail 10 -Wait
  }
}

function New-DesktopShortcut {
  $cmdPath = Join-Path $RepoRoot 'FixPhone-Mulai.cmd'
  if (-not (Test-Path $cmdPath)) { throw "File launcher tidak ditemukan: $cmdPath" }
  $desktop = [Environment]::GetFolderPath('Desktop')
  $lnkPath = Join-Path $desktop 'FixPhone.lnk'
  $shell = New-Object -ComObject WScript.Shell
  $shortcut = $shell.CreateShortcut($lnkPath)
  $shortcut.TargetPath = $cmdPath
  $shortcut.WorkingDirectory = $RepoRoot
  $shortcut.Description = 'Jalankan FixPhone (lokal)'
  $shortcut.IconLocation = "$env:SystemRoot\System32\shell32.dll,13"
  $shortcut.Save()
  Write-Ok "Shortcut dibuat: $lnkPath"
}

switch ($Action) {
  'start' { Start-FixPhone }
  'stop' { Stop-FixPhone }
  'restart' { Stop-FixPhone; Start-FixPhone }
  'status' { Show-Status }
  'logs' { Show-Logs }
  'shortcut' { New-DesktopShortcut }
}
