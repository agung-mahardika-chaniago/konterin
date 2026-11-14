# FixPhone production smoke test (PowerShell 7).
#
# Verifies the public edge, login, refresh cookie, tenant-scoped reads, and
# fail-closed public tracking. Run after deploy and after every restore.
#
#   $env:SMOKE_BASE_URL="https://fixphone.example.com"
#   $env:SMOKE_EMAIL="owner@toko.example.com"
#   $env:SMOKE_PASSWORD="..."
#   pwsh -NoProfile -File scripts/smoke-prod.ps1
#
# Exit code 0 = all checks passed.
param(
    [string]$BaseUrl = $env:SMOKE_BASE_URL,
    [string]$Email = $env:SMOKE_EMAIL,
    [string]$Password = $env:SMOKE_PASSWORD,
    [switch]$SkipCertificateCheck
)

$ErrorActionPreference = 'Stop'
if (-not $BaseUrl) { throw 'SMOKE_BASE_URL wajib diisi' }
if (-not $Email -or -not $Password) { throw 'SMOKE_EMAIL dan SMOKE_PASSWORD wajib diisi' }
$BaseUrl = $BaseUrl.TrimEnd('/')

$script:Failures = 0
$script:Token = $null
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession

function Invoke-Smoke {
    param(
        [string]$Name,
        [string]$Method,
        [string]$Path,
        [int[]]$Expected = @(200),
        [hashtable]$Headers = @{},
        [string]$Body = $null
    )
    $request = @{
        Method             = $Method
        Uri                = "$BaseUrl$Path"
        SkipHttpErrorCheck = $true
        TimeoutSec         = 20
        WebSession         = $session
        Headers            = $Headers
        UserAgent          = 'fixphone-smoke/1.0'
    }
    if ($SkipCertificateCheck) { $request.SkipCertificateCheck = $true }
    if ($null -ne $Body) {
        $request.Body = $Body
        $request.ContentType = 'application/json'
    }

    try {
        $response = Invoke-WebRequest @request
    } catch {
        Write-Host "[FAIL] $Name - request error: $($_.Exception.Message)" -ForegroundColor Red
        $script:Failures++
        return $null
    }

    $code = [int]$response.StatusCode
    if ($Expected -notcontains $code) {
        Write-Host "[FAIL] $Name - status $code, expected $($Expected -join '/')" -ForegroundColor Red
        Write-Host "       $($response.Content)" -ForegroundColor DarkGray
        $script:Failures++
        return $null
    }
    Write-Host "[ OK ] $Name ($code)" -ForegroundColor Green
    return $response
}

Write-Host "FixPhone smoke test -> $BaseUrl"

$health = Invoke-Smoke -Name 'liveness /healthz' -Method GET -Path '/healthz'
if ($health) {
    try {
        $healthJson = $health.Content | ConvertFrom-Json
        if ($healthJson.status -eq 'ok') {
            Write-Host '[ OK ] liveness reports status ok' -ForegroundColor Green
        } else {
            Write-Host '[FAIL] /healthz bukan JSON status ok (SPA fallback?)' -ForegroundColor Red
            $script:Failures++
        }
    } catch {
        Write-Host '[FAIL] /healthz bukan JSON valid (SPA fallback?)' -ForegroundColor Red
        $script:Failures++
    }
}
$ready = Invoke-Smoke -Name 'readiness /readyz' -Method GET -Path '/readyz'
if ($ready) {
    try {
        $readyJson = $ready.Content | ConvertFrom-Json
        if ($readyJson.status -eq 'ready' -and $readyJson.database -eq 'up') {
            Write-Host '[ OK ] readiness reports database up' -ForegroundColor Green
        } else {
            Write-Host '[FAIL] /readyz bukan JSON siap dari API (SPA fallback?)' -ForegroundColor Red
            $script:Failures++
        }
    } catch {
        Write-Host '[FAIL] /readyz bukan JSON valid (SPA fallback?)' -ForegroundColor Red
        $script:Failures++
    }
}
$versionInfo = Invoke-Smoke -Name 'build metadata /version' -Method GET -Path '/version'
if ($versionInfo) {
    try {
        $buildVersion = ($versionInfo.Content | ConvertFrom-Json).version
        if ($buildVersion) {
            Write-Host "[ OK ] versi API $buildVersion" -ForegroundColor Green
        } else {
            Write-Host '[FAIL] /version bukan metadata build dari API (SPA fallback?)' -ForegroundColor Red
            $script:Failures++
        }
    } catch {
        Write-Host '[FAIL] /version bukan JSON valid (SPA fallback?)' -ForegroundColor Red
        $script:Failures++
    }
}

if ($BaseUrl -like 'https://*') {
    if ($health -and -not $health.Headers['Strict-Transport-Security']) {
        Write-Host '[FAIL] HSTS header missing on HTTPS response' -ForegroundColor Red
        $script:Failures++
    } else {
        Write-Host '[ OK ] HSTS header present' -ForegroundColor Green
    }
}

Invoke-Smoke -Name 'anonymous request rejected' -Method GET -Path '/api/v1/customers' `
    -Expected 401 | Out-Null
Invoke-Smoke -Name 'unknown tracking ticket fails closed' -Method GET `
    -Path '/api/v1/public/track?ticket=NONEXISTENT&phone_last4=0000' -Expected 404 | Out-Null

$loginBody = @{ email = $Email; password = $Password } | ConvertTo-Json -Compress
$login = Invoke-Smoke -Name 'login' -Method POST -Path '/api/v1/auth/login' -Body $loginBody
if (-not $login) {
    Write-Host "Smoke test gagal: login tidak berhasil." -ForegroundColor Red
    exit 1
}
$loginJson = $login.Content | ConvertFrom-Json
$script:Token = $loginJson.data.access_token
if (-not $script:Token) {
    Write-Host '[FAIL] access_token tidak ada pada respons login' -ForegroundColor Red
    $script:Failures++
    exit 1
}

$authHeaders = @{ Authorization = "Bearer $script:Token" }
$me = Invoke-Smoke -Name 'profile /auth/me' -Method GET -Path '/api/v1/auth/me' -Headers $authHeaders
if ($me) {
    $role = ($me.Content | ConvertFrom-Json).data.user.role
    Write-Host "       role=$role"
}

Invoke-Smoke -Name 'dashboard summary' -Method GET -Path '/api/v1/dashboard/summary' `
    -Headers $authHeaders | Out-Null
Invoke-Smoke -Name 'service order list' -Method GET -Path '/api/v1/service-orders?per_page=1' `
    -Headers $authHeaders | Out-Null
Invoke-Smoke -Name 'customer list' -Method GET -Path '/api/v1/customers?per_page=1' `
    -Headers $authHeaders | Out-Null

$refresh = Invoke-Smoke -Name 'refresh cookie rotation' -Method POST -Path '/api/v1/auth/refresh' `
    -Body '{}'
if ($refresh) {
    $newToken = ($refresh.Content | ConvertFrom-Json).data.access_token
    if (-not $newToken) {
        Write-Host '[FAIL] refresh tidak mengembalikan access_token' -ForegroundColor Red
        $script:Failures++
    } else {
        $script:Token = $newToken
        $authHeaders = @{ Authorization = "Bearer $script:Token" }
    }
}

Invoke-Smoke -Name 'logout' -Method POST -Path '/api/v1/auth/logout' -Headers $authHeaders | Out-Null
Invoke-Smoke -Name 'refresh cookie revoked after logout' -Method POST -Path '/api/v1/auth/refresh' `
    -Body '{}' -Expected 401 | Out-Null

if ($script:Failures -gt 0) {
    Write-Host "Smoke test GAGAL: $($script:Failures) pemeriksaan." -ForegroundColor Red
    exit 1
}
Write-Host 'Smoke test LULUS.' -ForegroundColor Green
