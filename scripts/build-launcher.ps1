#requires -Version 5.1
<#
.SYNOPSIS
  Membangun FixPhone.exe dari tools/launcher di root repo.

.DESCRIPTION
  FixPhone.exe adalah launcher tipis yang memanggil scripts/fixphone.ps1.
  Bangun ulang setiap kali main.go berubah. Hasil build tidak di-commit
  (lihat .gitignore) agar tidak ada binary di repositori.

.EXAMPLE
  powershell -NoProfile -ExecutionPolicy Bypass -File scripts/build-launcher.ps1
#>
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent $PSScriptRoot
$SourceDir = Join-Path $RepoRoot 'tools\launcher'
$Output = Join-Path $RepoRoot 'FixPhone.exe'

if (-not (Test-Path (Join-Path $SourceDir 'main.go'))) {
  throw "Sumber launcher tidak ditemukan: $SourceDir"
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw 'Go tidak ditemukan di PATH. Pasang Go 1.21+ dari https://go.dev/dl/'
}

Write-Host "==> Membangun $Output" -ForegroundColor Cyan
Push-Location $SourceDir
try {
  & go build -trimpath -ldflags '-s -w' -o $Output .
  if ($LASTEXITCODE -ne 0) { throw "go build gagal (exit $LASTEXITCODE)" }
} finally {
  Pop-Location
}

$size = [math]::Round((Get-Item $Output).Length / 1KB, 1)
Write-Host "  OK  $Output ($size KB)" -ForegroundColor Green
Write-Host '      Dobel-klik file tersebut untuk menjalankan FixPhone.'
