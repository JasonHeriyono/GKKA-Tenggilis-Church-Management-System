$ErrorActionPreference = "Stop"

$BaseUrl = if ($env:SIM_G_API_URL) { $env:SIM_G_API_URL.TrimEnd("/") } else { "http://localhost:8080/gkki/api" }
$AdminEmail = if ($env:SIM_G_ADMIN_IDENTIFIER) { $env:SIM_G_ADMIN_IDENTIFIER } else { "admin.test@example.com" }
$AdminPassword = if ($env:SIM_G_ADMIN_PASSWORD) { $env:SIM_G_ADMIN_PASSWORD } else { "TestAdmin2026!" }
$JemaatEmail = if ($env:SIM_G_JEMAAT_IDENTIFIER) { $env:SIM_G_JEMAAT_IDENTIFIER } else { "jemaat.skill.test@example.com" }
$JemaatPassword = if ($env:SIM_G_JEMAAT_PASSWORD) { $env:SIM_G_JEMAAT_PASSWORD } else { "TestJemaat2026!" }

$JemaatCode = if ($env:SIM_G_TEST_JEMAAT_CODE) { $env:SIM_G_TEST_JEMAAT_CODE } else { "TEST-API-JEMAAT" }
$JemaatName = if ($env:SIM_G_TEST_JEMAAT_NAME) { $env:SIM_G_TEST_JEMAAT_NAME } else { "SIM-G API Test Jemaat" }

Write-Host "`n=== SIM-G API Test Account Setup (Docker-free) ===" -ForegroundColor Cyan
Write-Host "API URL: $BaseUrl"

function Invoke-ApiRequest {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [Parameter(Mandatory = $true)][string]$Method,
        [string]$Token,
        [object]$Body
    )

    $params = @{
        Uri = $Uri
        Method = $Method
        ContentType = "application/json"
        ErrorAction = "Stop"
        TimeoutSec = 10
    }

    if ($Token) {
        $params.Headers = @{ Authorization = "Bearer $Token" }
    }
    if ($null -ne $Body) {
        $params.Body = $Body | ConvertTo-Json -Depth 10
    }

    Invoke-RestMethod @params
}

function Get-AccessToken {
    param([Parameter(Mandatory = $true)][string]$Identifier, [Parameter(Mandatory = $true)][string]$Password)

    $result = Invoke-ApiRequest -Uri "$BaseUrl/auth/login" -Method Post -Body @{
        identifier = $Identifier
        password = $Password
    }

    $token = $result.data.access_token
    if (-not $token) { $token = $result.access_token }
    return $token
}

# Check the local API. This script prepares accounts; the local test runner starts the API.
try {
    $health = Invoke-RestMethod -Uri "$BaseUrl/health" -Method Get -TimeoutSec 5 -ErrorAction Stop
    if ($health.data.status -and $health.data.status -ne "ok") {
        throw "API health status is '$($health.data.status)'."
    }
    if ($health.data.db -and $health.data.db -ne "up") {
        throw "Database health status is '$($health.data.db)'."
    }
} catch {
    throw @"
Cannot reach a healthy API at $BaseUrl.
This script does not start Docker or the backend. Start the local MySQL service and backend first, or run scripts\test-api-local.ps1 to start the Go backend automatically.
Details: $($_.Exception.Message)
"@
}
Write-Host "API and database health checks passed." -ForegroundColor Green

# Authenticate as Super Admin. The script deliberately does not create or reset this account.
try {
    $adminToken = Get-AccessToken -Identifier $AdminEmail -Password $AdminPassword
    if (-not $adminToken) { throw "Login response did not contain an access token." }
    Write-Host "Super Admin login succeeded." -ForegroundColor Green
} catch {
    throw @"
Super Admin login failed. The account-setup script requires a working Super Admin account and will not change its password.
Expected administrator: $AdminEmail
Check that this account exists in the local database and that the credentials are correct.
Details: $($_.Exception.Message)
"@
}

# Reuse the test Jemaat account if the configured password works.
try {
    $jemaatToken = Get-AccessToken -Identifier $JemaatEmail -Password $JemaatPassword
    if ($jemaatToken) {
        Write-Host "Existing Jemaat test account works; reusing it." -ForegroundColor Green
        Write-Host "`nTest-account setup completed." -ForegroundColor Green
        return
    }
} catch {
    # Continue to inspect/create the account below.
}

# Determine whether the account exists before attempting creation.
$existingUser = $null
try {
    $usersResult = Invoke-ApiRequest -Uri "$BaseUrl/users?limit=100" -Method Get -Token $adminToken
    $items = $usersResult.data.items
    if ($null -eq $items) { $items = $usersResult.data }
    if ($items -is [System.Collections.IEnumerable] -and $items -isnot [string]) {
        $existingUser = @($items) | Where-Object { $_.email -eq $JemaatEmail } | Select-Object -First 1
    }
} catch {
    Write-Warning "Could not list users to check whether the Jemaat account exists. The create request will determine whether creation is possible."
}

if ($existingUser) {
    throw @"
The Jemaat test account already exists, but login with the configured password failed.
The script will not reset an existing account password.
Email: $JemaatEmail
User ID: $($existingUser.id)
Check the password or update SIM_G_JEMAAT_PASSWORD before rerunning.
"@
}

Write-Host "Creating the Jemaat test account..."
try {
    $null = Invoke-ApiRequest -Uri "$BaseUrl/users" -Method Post -Token $adminToken -Body @{
        kode_jemaat = $JemaatCode
        nama = $JemaatName
        email = $JemaatEmail
        role = "Jemaat"
        password = $JemaatPassword
    }
} catch {
    throw @"
Could not create the Jemaat test account.
Email: $JemaatEmail
Kode jemaat: $JemaatCode
Check the API error for duplicate email/kode jemaat, validation, or permissions.
No existing account was deleted and no password was reset.
Details: $($_.Exception.Message)
"@
}

try {
    $verifiedToken = Get-AccessToken -Identifier $JemaatEmail -Password $JemaatPassword
    if (-not $verifiedToken) { throw "Login response did not contain an access token." }
} catch {
    throw "The account creation request returned successfully, but login verification failed. Check the account status and API response. Details: $($_.Exception.Message)"
}

Write-Host "Jemaat account created and login verified." -ForegroundColor Green
Write-Host "`nTest-account setup completed." -ForegroundColor Green