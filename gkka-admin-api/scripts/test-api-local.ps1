$ErrorActionPreference = "Stop"

# Docker-free API integration test runner.
# Requirements:
# - Go installed and available on PATH
# - MySQL installed/running locally
# - Local .env configured for that MySQL instance
# - Database migrations applied
# This script starts the Go API if it is not already healthy, then runs tests.

$RepoRoot = Split-Path -Parent $PSScriptRoot
$env:SIM_G_API_TESTS = "1"

if (-not $env:SIM_G_API_URL) { $env:SIM_G_API_URL = "http://localhost:8080/gkki/api" }
if (-not $env:SIM_G_TEST_IDENTIFIER) { $env:SIM_G_TEST_IDENTIFIER = "admin.test@example.com" }
if (-not $env:SIM_G_TEST_PASSWORD) { $env:SIM_G_TEST_PASSWORD = "TestAdmin2026!" }
if (-not $env:SIM_G_TEST_USER_ID) { $env:SIM_G_TEST_USER_ID = "2" }
if (-not $env:SIM_G_JEMAAT_IDENTIFIER) { $env:SIM_G_JEMAAT_IDENTIFIER = "jemaat.skill.test@example.com" }
if (-not $env:SIM_G_JEMAAT_PASSWORD) { $env:SIM_G_JEMAAT_PASSWORD = "TestJemaat2026!" }

$BaseUrl = $env:SIM_G_API_URL.TrimEnd("/")
$HealthUrl = "$BaseUrl/health"
$StartedBackend = $false
$BackendProcess = $null
$LogPath = Join-Path $env:TEMP "sim-g-api-test-backend.log"

Write-Host "`n=== SIM-G API Tests (Docker-free / Local) ===" -ForegroundColor Cyan
Write-Host "Repository: $RepoRoot"
Write-Host "API URL:    $BaseUrl"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is not installed or is not on PATH. Install Go for Windows, reopen PowerShell, then run this script again. This runner does not use Docker."
}

Push-Location $RepoRoot
try {
    go version
    if ($LASTEXITCODE -ne 0) { throw "Could not run Go. Check the Go installation." }

    # First reuse a healthy API if one is already running.
    $ApiHealthy = $false
    try {
        $health = Invoke-RestMethod -Uri $HealthUrl -Method Get -TimeoutSec 3 -ErrorAction Stop
        $ApiHealthy = (($health.data.status -eq "ok" -or -not $health.data.status) -and
                       ($health.data.db -eq "up" -or -not $health.data.db))
    } catch {
        $ApiHealthy = $false
    }

    if (-not $ApiHealthy) {
        Write-Host "`nAPI is not healthy. Starting the local Go backend..." -ForegroundColor Yellow

        # Run in the repository root so the app can load its normal .env/config files.
        if (-not (Test-Path (Join-Path $RepoRoot ".env"))) {
            throw "No .env file found at $RepoRoot\.env. Create/configure it for your locally installed MySQL before starting the backend."
        }

        $BackendProcess = Start-Process -FilePath (Get-Command go).Source `
            -ArgumentList @("run", ".") `
            -WorkingDirectory $RepoRoot `
            -RedirectStandardOutput $LogPath `
            -RedirectStandardError ($LogPath + ".err") `
            -PassThru
        $StartedBackend = $true

        $deadline = (Get-Date).AddSeconds(60)
        while ((Get-Date) -lt $deadline) {
            if ($BackendProcess.HasExited) {
                $stdout = if (Test-Path $LogPath) { Get-Content $LogPath -Raw } else { "" }
                $stderr = if (Test-Path ($LogPath + ".err")) { Get-Content ($LogPath + ".err") -Raw } else { "" }
                throw "The local backend exited before becoming healthy. Check MySQL, .env, and migrations.`n--- Backend stdout ---`n$stdout`n--- Backend stderr ---`n$stderr"
            }

            try {
                $health = Invoke-RestMethod -Uri $HealthUrl -Method Get -TimeoutSec 2 -ErrorAction Stop
                if (($health.data.status -eq "ok" -or -not $health.data.status) -and
                    ($health.data.db -eq "up" -or -not $health.data.db)) {
                    $ApiHealthy = $true
                    break
                }
            } catch {}
            Start-Sleep -Seconds 2
        }

        if (-not $ApiHealthy) {
            $stdout = if (Test-Path $LogPath) { Get-Content $LogPath -Raw } else { "" }
            $stderr = if (Test-Path ($LogPath + ".err")) { Get-Content ($LogPath + ".err") -Raw } else { "" }
            throw "The backend did not become healthy within 60 seconds. Verify local MySQL is running, .env has correct database settings, and migrations are applied.`n--- Backend stdout ---`n$stdout`n--- Backend stderr ---`n$stderr"
        }
        Write-Host "Local backend is healthy." -ForegroundColor Green
    } else {
        Write-Host "Using the already-running local API." -ForegroundColor Green
    }

    Write-Host "`nPreparing API test accounts..." -ForegroundColor Cyan
    $setupScript = Join-Path $PSScriptRoot "setup-api-tests.ps1"
    if (-not (Test-Path $setupScript)) { throw "Setup script not found: $setupScript" }
    & $setupScript
    if ($LASTEXITCODE -and $LASTEXITCODE -ne 0) { throw "Test account setup failed." }

    Write-Host "`nRunning API integration tests..." -ForegroundColor Cyan
    go test ./tests/api -v
    if ($LASTEXITCODE -ne 0) { throw "API tests failed. Review the output above." }

    Write-Host "`nAPI integration tests passed." -ForegroundColor Green
}
finally {
    # Only stop the backend process if this script started it.
    if ($StartedBackend -and $BackendProcess -and -not $BackendProcess.HasExited) {
        Stop-Process -Id $BackendProcess.Id -Force -ErrorAction SilentlyContinue
        Write-Host "Stopped the backend process started by this test runner."
    }
    Pop-Location
}