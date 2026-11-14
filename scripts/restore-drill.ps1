#requires -Version 5.1
<#
.SYNOPSIS
  Latihan restore PostgreSQL FixPhone (backup/restore drill).

.DESCRIPTION
  Membuktikan bahwa backup penuh database dapat dipulihkan. Alur:
    1. pg_dump -Fc database sumber (default: fixphone_test)
    2. buat database scratch baru
    3. pg_restore ke database scratch
    4. bandingkan jumlah tabel/baris, status RLS, dan versi migrasi
    5. hapus database scratch (kecuali -KeepScratch)

  Script mencari pg_dump/pg_restore/psql di PATH, instalasi PostgreSQL
  standar, .dev/pgsql/bin (server portable), lalu .dev/tools/pgsql-client.
  Binary client dapat diunduh dari:
    https://github.com/theseus-rs/postgresql-binaries/releases

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts/restore-drill.ps1
  powershell -ExecutionPolicy Bypass -File scripts/restore-drill.ps1 -SourceDSN "postgres://postgres:postgres@127.0.0.1:55432/fixphone_test?sslmode=disable"
  powershell -ExecutionPolicy Bypass -File scripts/restore-drill.ps1 -KeepDump -KeepScratch
#>
[CmdletBinding()]
param(
  [string]$SourceDSN = 'postgres://postgres:postgres@127.0.0.1:55432/fixphone_test?sslmode=disable',
  [string]$ScratchDB = 'fixphone_restore_drill',
  [string]$DumpPath = '',
  [switch]$KeepDump,
  [switch]$KeepScratch
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$RepoRoot = Split-Path -Parent $PSScriptRoot
$DevDir = Join-Path $RepoRoot '.dev'
$DrillDir = Join-Path $DevDir 'restore-drill'

function Write-Step([string]$Message) {
  Write-Host "==> $Message" -ForegroundColor Cyan
}

function Write-Ok([string]$Message) {
  Write-Host "  OK  $Message" -ForegroundColor Green
}

function Fail([string]$Message) {
  Write-Host "  FAIL $Message" -ForegroundColor Red
  throw $Message
}

function Find-PgTool([string]$Name) {
  $cmd = Get-Command $Name -ErrorAction SilentlyContinue
  if ($cmd) { return $cmd.Source }

  # Windows client binaries carry an .exe suffix; Linux/macOS runners do not.
  $isWindowsHost = $env:OS -eq 'Windows_NT'
  $exe = if ($isWindowsHost) { "$Name.exe" } else { $Name }

  $candidates = @()
  if ($isWindowsHost) {
    if ($env:ProgramFiles) {
      $candidates += Get-ChildItem (Join-Path $env:ProgramFiles 'PostgreSQL') -Directory -ErrorAction SilentlyContinue |
        Sort-Object Name -Descending |
        ForEach-Object { Join-Path $_.FullName "bin\$exe" }
    }
  } else {
    foreach ($root in @('/usr/lib/postgresql', '/usr/local/pgsql', '/opt/homebrew/opt')) {
      $candidates += Get-ChildItem $root -Directory -ErrorAction SilentlyContinue |
        Sort-Object Name -Descending |
        ForEach-Object { Join-Path $_.FullName "bin\$exe" }
    }
  }
  $pgBin = Join-Path (Join-Path $DevDir 'pgsql') 'bin'
  $clientDir = Join-Path (Join-Path $DevDir 'tools') 'pgsql-client'
  $candidates += Join-Path $pgBin $exe
  $candidates += Join-Path $pgBin $Name
  $candidates += Get-ChildItem $clientDir -Recurse -Filter $exe -ErrorAction SilentlyContinue |
    Select-Object -ExpandProperty FullName

  foreach ($candidate in $candidates) {
    if ($candidate -and (Test-Path $candidate)) { return $candidate }
  }
  return $null
}

function ConvertFrom-Dsn([string]$Dsn) {
  $uri = [System.Uri]$Dsn
  if (-not $uri.IsAbsoluteUri) { Fail "DSN tidak valid: $Dsn" }
  $userInfo = $uri.UserInfo.Split(':', 2)
  $database = $uri.AbsolutePath.TrimStart('/')
  if (-not $database) { Fail "DSN tidak memuat nama database: $Dsn" }
  $query = @{}
  foreach ($pair in $uri.Query.TrimStart('?').Split('&')) {
    if (-not $pair) { continue }
    $kv = $pair.Split('=', 2)
    $query[$kv[0]] = if ($kv.Length -gt 1) { [System.Uri]::UnescapeDataString($kv[1]) } else { '' }
  }
  return [pscustomobject]@{
    Host     = $uri.Host
    Port     = if ($uri.Port -gt 0) { $uri.Port } else { 5432 }
    User     = if ($userInfo.Length -ge 1 -and $userInfo[0]) { [System.Uri]::UnescapeDataString($userInfo[0]) } else { 'postgres' }
    Password = if ($userInfo.Length -ge 2) { [System.Uri]::UnescapeDataString($userInfo[1]) } else { '' }
    Database = $database
    SslMode  = if ($query.ContainsKey('sslmode')) { $query['sslmode'] } else { 'prefer' }
  }
}

function Invoke-Psql([string]$Psql, $Conn, [string]$Database, [string]$Sql) {
  $env:PGPASSWORD = $Conn.Password
  # NOTICE messages on stderr must not be treated as errors by
  # $ErrorActionPreference = 'Stop'; only the exit code decides success.
  $previous = $ErrorActionPreference
  $ErrorActionPreference = 'Continue'
  try {
    $output = & $Psql --no-psqlrc -h $Conn.Host -p $Conn.Port -U $Conn.User -d $Database -v ON_ERROR_STOP=1 -t -A -c $Sql
  } finally {
    $ErrorActionPreference = $previous
  }
  if ($LASTEXITCODE -ne 0) {
    throw "psql gagal (db=$Database): $($output -join "`n")"
  }
  return ($output | Where-Object { $_ -ne '' })
}

function Get-TableStats([string]$Psql, $Conn, [string]$Database) {
  $tables = Invoke-Psql $Psql $Conn $Database @"
SELECT c.relname
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'r'
ORDER BY c.relname
"@

  $stats = [ordered]@{}
  foreach ($table in $tables) {
    $row = Invoke-Psql $Psql $Conn $Database "SELECT count(*) FROM public.$table"
    $stats[$table] = [int64]$row
  }
  return $stats
}

function Get-SecurityState([string]$Psql, $Conn, [string]$Database) {
  $rls = Invoke-Psql $Psql $Conn $Database @"
SELECT c.relname
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public' AND c.relkind = 'r' AND c.relrowsecurity
ORDER BY c.relname
"@
  $policies = Invoke-Psql $Psql $Conn $Database @"
SELECT count(*)
FROM pg_policies
WHERE schemaname = 'public'
"@
  $version = Invoke-Psql $Psql $Conn $Database "SELECT version::text FROM schema_migrations LIMIT 1"
  return [pscustomobject]@{
    RlsTables = @($rls)
    Policies  = [int]$policies
    Version   = ($version | Select-Object -First 1)
  }
}

# --- Locate tools -----------------------------------------------------------
$pgDump = Find-PgTool 'pg_dump'
$pgRestore = Find-PgTool 'pg_restore'
$psql = Find-PgTool 'psql'
if (-not $pgDump) { Fail 'pg_dump tidak ditemukan. Set PATH atau unduh client tools ke .dev/tools/pgsql-client.' }
if (-not $pgRestore) { Fail 'pg_restore tidak ditemukan. Set PATH atau unduh client tools ke .dev/tools/pgsql-client.' }
if (-not $psql) { Fail 'psql tidak ditemukan. Set PATH atau unduh client tools ke .dev/tools/pgsql-client.' }

Write-Step "Menggunakan client tools PostgreSQL"
Write-Host "  pg_dump   : $pgDump"
Write-Host "  pg_restore: $pgRestore"
Write-Host "  psql      : $psql"

$conn = ConvertFrom-Dsn $SourceDSN
Write-Step "Sumber: $($conn.User)@$($conn.Host):$($conn.Port)/$($conn.Database)"

# The dump is verified against itself; refuse to run against the scratch DB.
if ($conn.Database -eq $ScratchDB) { Fail "SourceDSN dan ScratchDB tidak boleh sama ($ScratchDB)." }

New-Item -ItemType Directory -Force -Path $DrillDir | Out-Null
if (-not $DumpPath) {
  $DumpPath = Join-Path $DrillDir ("backup-{0}.dump" -f (Get-Date -Format 'yyyyMMdd-HHmmss'))
}

$env:PGPASSWORD = $conn.Password
$env:PGCONNECT_TIMEOUT = '10'
# Keep psql NOTICE output off stderr so callers do not mistake it for failure.
$env:PGOPTIONS = '-c client_min_messages=warning'

# --- 1. Dump ----------------------------------------------------------------
Write-Step "Membuat backup penuh (custom format) ke $DumpPath"
& $pgDump -h $conn.Host -p $conn.Port -U $conn.User -d $conn.Database -Fc --no-owner --no-privileges -f $DumpPath
if ($LASTEXITCODE -ne 0) { Fail 'pg_dump gagal.' }
$dumpBytes = (Get-Item $DumpPath).Length
if ($dumpBytes -le 0) { Fail 'File dump kosong.' }
Write-Ok ("dump {0:N0} byte" -f $dumpBytes) 

# --- 2. Recreate scratch database ------------------------------------------
Write-Step "Menyiapkan database scratch: $ScratchDB"
Invoke-Psql $psql $conn 'postgres' "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$ScratchDB' AND pid <> pg_backend_pid()" | Out-Null
Invoke-Psql $psql $conn 'postgres' "DROP DATABASE IF EXISTS `"$ScratchDB`"" | Out-Null
Invoke-Psql $psql $conn 'postgres' "CREATE DATABASE `"$ScratchDB`" TEMPLATE template0" | Out-Null

# Policies reference the cluster-wide application role, so it must exist even
# though privileges are skipped in the dump.
Invoke-Psql $psql $conn $ScratchDB @"
DO `$`$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fixphone_app') THEN
    CREATE ROLE fixphone_app NOLOGIN NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;
  END IF;
END
`$`$
"@ | Out-Null

# --- 3. Restore -------------------------------------------------------------
Write-Step 'Memulihkan backup ke database scratch'
& $pgRestore -h $conn.Host -p $conn.Port -U $conn.User -d $ScratchDB --no-owner --exit-on-error -j 2 $DumpPath
if ($LASTEXITCODE -ne 0) { Fail 'pg_restore gagal.' }
Write-Ok 'pg_restore selesai'

# --- 4. Verify --------------------------------------------------------------
Write-Step 'Memverifikasi hasil restore'
$sourceStats = Get-TableStats $psql $conn $conn.Database
$restoreStats = Get-TableStats $psql $conn $ScratchDB

if ($sourceStats.Count -ne $restoreStats.Count) {
  Fail "jumlah tabel berbeda: sumber=$($sourceStats.Count) restore=$($restoreStats.Count)"
}
$rowMismatch = @()
foreach ($table in $sourceStats.Keys) {
  if (-not $restoreStats.Contains($table)) {
    $rowMismatch += "tabel $table hilang"
    continue
  }
  if ($sourceStats[$table] -ne $restoreStats[$table]) {
    $rowMismatch += "$table sumber=$($sourceStats[$table]) restore=$($restoreStats[$table])"
  }
}
if ($rowMismatch.Count -gt 0) { Fail ("perbedaan data:`n  " + ($rowMismatch -join "`n  ")) }
Write-Ok "$($sourceStats.Count) tabel dan seluruh jumlah baris identik"

$sourceSec = Get-SecurityState $psql $conn $conn.Database
$restoreSec = Get-SecurityState $psql $conn $ScratchDB
if ($sourceSec.Version -ne $restoreSec.Version) {
  Fail "versi migrasi berbeda: sumber=$($sourceSec.Version) restore=$($restoreSec.Version)"
}
Write-Ok "versi migrasi schema_migrations = $($restoreSec.Version)"
if ($sourceSec.Policies -ne $restoreSec.Policies) {
  Fail "jumlah policy berbeda: sumber=$($sourceSec.Policies) restore=$($restoreSec.Policies)"
}
Write-Ok "$($restoreSec.Policies) policy RLS ikut terpulihkan"
if ($sourceSec.RlsTables.Count -ne $restoreSec.RlsTables.Count) {
  Fail "tabel dengan RLS berbeda: sumber=$($sourceSec.RlsTables.Count) restore=$($restoreSec.RlsTables.Count)"
}
Write-Ok "$($restoreSec.RlsTables.Count) tabel berstatus RLS aktif"

# --- 5. Cleanup -------------------------------------------------------------
if ($KeepScratch) {
  Write-Step "Database scratch dipertahankan: $ScratchDB"
} else {
  Write-Step 'Menghapus database scratch'
  Invoke-Psql $psql $conn 'postgres' "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$ScratchDB' AND pid <> pg_backend_pid()" | Out-Null
  Invoke-Psql $psql $conn 'postgres' "DROP DATABASE IF EXISTS `"$ScratchDB`"" | Out-Null
}
if ($KeepDump) {
  Write-Step "File dump dipertahankan: $DumpPath"
} elseif (Test-Path $DumpPath) {
  Remove-Item -Force $DumpPath
}

Write-Host ''
Write-Host 'Restore drill LULUS' -ForegroundColor Green
Write-Host "  dump     : $dumpBytes byte"
Write-Host "  tabel    : $($sourceStats.Count)"
Write-Host "  policy   : $($restoreSec.Policies)"
Write-Host "  migrasi  : $($restoreSec.Version)"
