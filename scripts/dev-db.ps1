#requires -Version 5.1
<#
.SYNOPSIS
  PostgreSQL 16 portable untuk development FixPhone (tanpa Docker/admin).

.DESCRIPTION
  Mengunduh binary PostgreSQL 16 dari arsip Zonky embedded-postgres, lalu
  mengelola cluster lokal di folder .dev/. Data tidak masuk ke Git.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 start
  powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 status
  powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 reset
#>
[CmdletBinding()]
param(
  [ValidateSet('download', 'init', 'start', 'stop', 'status', 'reset', 'logs')]
  [string]$Action = 'start',

  [int]$Port = 55432
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$RepoRoot = Split-Path -Parent $PSScriptRoot
$DevDir = Join-Path $RepoRoot '.dev'
$PgDir = Join-Path $DevDir 'pgsql'
$BinDir = Join-Path $PgDir 'bin'
$DataDir = Join-Path $DevDir 'pgdata'
$CacheDir = Join-Path $DevDir 'cache'
$LogFile = Join-Path $DevDir 'postgres.log'

$PgVersion = '16.15.0'
$JarName = "embedded-postgres-binaries-windows-amd64-$PgVersion.jar"
$JarUrl = "https://repo1.maven.org/maven2/io/zonky/test/postgres/embedded-postgres-binaries-windows-amd64/$PgVersion/$JarName"

$InitDb = Join-Path $BinDir 'initdb.exe'
$PgCtl = Join-Path $BinDir 'pg_ctl.exe'
$Postgres = Join-Path $BinDir 'postgres.exe'

function Write-Step([string]$Message) {
  Write-Host "==> $Message" -ForegroundColor Cyan
}

function Assert-Binaries {
  if (-not (Test-Path $InitDb) -or -not (Test-Path $PgCtl) -or -not (Test-Path $Postgres)) {
    throw "Binary PostgreSQL belum ada. Jalankan: powershell -ExecutionPolicy Bypass -File scripts/dev-db.ps1 download"
  }
}

function Invoke-Download {
  New-Item -ItemType Directory -Force -Path $CacheDir | Out-Null
  $jar = Join-Path $CacheDir $JarName
  $txz = Join-Path $CacheDir 'postgres-windows-x86_64.txz'

  if (-not (Test-Path $jar)) {
    Write-Step "Mengunduh PostgreSQL $PgVersion (sekitar 22 MB)"
    Invoke-WebRequest -UseBasicParsing -Uri $JarUrl -OutFile $jar
  }

  if (-not (Test-Path $txz)) {
    Write-Step 'Mengekstrak arsip dari jar'
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [System.IO.Compression.ZipFile]::OpenRead($jar)
    try {
      $entry = $zip.Entries | Where-Object { $_.FullName -like '*.txz' } | Select-Object -First 1
      if (-not $entry) { throw 'Arsip .txz tidak ditemukan di dalam jar.' }
      [System.IO.Compression.ZipFileExtensions]::ExtractToFile($entry, $txz, $true)
    } finally {
      $zip.Dispose()
    }
  }

  if (-not (Test-Path $InitDb)) {
    Write-Step "Mengekstrak binary ke $PgDir"
    New-Item -ItemType Directory -Force -Path $PgDir | Out-Null
    tar -xf $txz -C $PgDir
    if ($LASTEXITCODE -ne 0) { throw 'Gagal mengekstrak arsip PostgreSQL.' }
  }

  & $Postgres --version
}

function Test-Initialized {
  return Test-Path (Join-Path $DataDir 'PG_VERSION')
}

function Initialize-Database {
  Assert-Binaries
  if (Test-Initialized) {
    Write-Step "Cluster sudah diinisialisasi di $DataDir"
    return
  }

  New-Item -ItemType Directory -Force -Path $DevDir | Out-Null
  Write-Step "Inisialisasi cluster di $DataDir (trust auth, khusus development)"
  & $InitDb -D $DataDir -U postgres -A trust -E UTF8 --locale=C
  if ($LASTEXITCODE -ne 0) { throw 'initdb gagal.' }

  $conf = Join-Path $DataDir 'postgresql.conf'
  $extra = @"

# ===== FixPhone portable dev =====
listen_addresses = '127.0.0.1'
port = $Port
max_connections = 100
"@
  Add-Content -Path $conf -Value $extra
}

function Test-Running {
  if (-not (Test-Path $PgCtl)) { return $false }
  & $PgCtl -D $DataDir status *> $null
  return ($LASTEXITCODE -eq 0)
}

# Win32_Process.Create menjalankan proses tanpa console induk. Tanpa ini,
# menutup jendela launcher mengirim CTRL_CLOSE_EVENT ke postgres dan
# mematikannya (gejala: API mati, login gagal "periksa koneksi").
function Start-DetachedProcess([string]$FilePath, [string[]]$Arguments, [string]$WorkingDirectory) {
  $quoted = foreach ($arg in $Arguments) {
    if ($arg -match '[\s"]') { '"' + ($arg -replace '"', '\"') + '"' } else { $arg }
  }
  $commandLine = '"' + $FilePath + '"'
  if ($quoted) { $commandLine += ' ' + ($quoted -join ' ') }
  $result = Invoke-CimMethod -ClassName Win32_Process -MethodName Create -Arguments @{
    CommandLine      = $commandLine
    CurrentDirectory = $WorkingDirectory
  }
  if ($result.ReturnValue -ne 0) {
    throw "Gagal menjalankan proses (kode $($result.ReturnValue)): $commandLine"
  }
  return [int]$result.ProcessId
}

function Start-Database {
  Assert-Binaries
  if (-not (Test-Initialized)) { Initialize-Database }

  if (Test-Running) {
    Write-Step "PostgreSQL sudah berjalan di 127.0.0.1:$Port"
    return
  }

  Write-Step "Menjalankan PostgreSQL di 127.0.0.1:$Port"
  $null = Start-DetachedProcess $PgCtl @('-D', $DataDir, '-l', $LogFile, 'start') (Split-Path -Parent $PgCtl)

  $ok = $false
  for ($i = 0; $i -lt 60; $i++) {
    Start-Sleep -Milliseconds 300
    if ((Test-NetConnection -ComputerName '127.0.0.1' -Port $Port -InformationLevel Quiet -WarningAction SilentlyContinue)) {
      $ok = $true
      break
    }
  }
  if (-not $ok) { throw "Port $Port tidak menerima koneksi. Lihat $LogFile" }
  Write-Step "Siap. DB_DSN=postgres://postgres:postgres@127.0.0.1:$Port/fixphone_dev?sslmode=disable"
}

function Stop-Database {
  if (-not (Test-Path $PgCtl)) {
    Write-Step 'Binary PostgreSQL tidak ada; tidak ada yang perlu dihentikan.'
    return
  }
  if (-not (Test-Initialized)) {
    Write-Step 'Cluster belum diinisialisasi.'
    return
  }
  if (-not (Test-Running)) {
    Write-Step 'PostgreSQL tidak sedang berjalan.'
    return
  }
  Write-Step 'Menghentikan PostgreSQL'
  & $PgCtl -D $DataDir -m fast -w stop
  if ($LASTEXITCODE -ne 0) { throw 'Gagal menghentikan PostgreSQL.' }
}

function Show-Status {
  if (-not (Test-Path $PgCtl)) {
    Write-Host 'Status: binary belum diunduh'
    return
  }
  if (-not (Test-Initialized)) {
    Write-Host 'Status: cluster belum diinisialisasi'
    return
  }
  if (Test-Running) {
    Write-Host "Status: berjalan di 127.0.0.1:$Port" -ForegroundColor Green
  } else {
    Write-Host 'Status: berhenti' -ForegroundColor Yellow
  }
  Write-Host "Data: $DataDir"
  Write-Host "Log : $LogFile"
}

switch ($Action) {
  'download' { Invoke-Download }
  'init' { Initialize-Database }
  'start' { Start-Database }
  'stop' { Stop-Database }
  'status' { Show-Status }
  'logs' {
    if (-not (Test-Path $LogFile)) { throw "Log belum ada: $LogFile" }
    Get-Content $LogFile -Tail 80
  }
  'reset' {
    Stop-Database
    if (Test-Path $DataDir) {
      Write-Step "Menghapus data lama di $DataDir"
      Remove-Item -Recurse -Force $DataDir
    }
    Initialize-Database
    Start-Database
  }
}
