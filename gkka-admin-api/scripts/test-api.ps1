
$ErrorActionPreference = "Stop"

# SIM-G API test configuration
$env:SIM_G_API_TESTS = "1"
$env:SIM_G_API_URL = "http://host.docker.internal:8080/gkki/api"
$env:SIM_G_TEST_IDENTIFIER = "admin.test@example.com"
$env:SIM_G_TEST_PASSWORD = "TestAdmin2026!"
$env:SIM_G_TEST_USER_ID = "2"
$env:SIM_G_JEMAAT_IDENTIFIER = "jemaat.skill.test@example.com"
$env:SIM_G_JEMAAT_PASSWORD = "TestJemaat2026!"

Write-Host ""
Write-Host "=== SIM-G API Integration Tests ===" -ForegroundColor Cyan

# Check that the API container is running.
$apiContainer = docker compose ps --status running -q api

if ($LASTEXITCODE -ne 0 -or -not $apiContainer) {
    throw "API container is not running. Start it with: docker compose up -d"
}

Write-Host "API container is running." -ForegroundColor Green
Write-Host "Running integration tests..."
Write-Host ""

# Run the tests inside a Go container.
docker run --rm `
    --add-host=host.docker.internal:host-gateway `
    -v "${PWD}:/app" `
    -w /app `
    -e SIM_G_API_TESTS `
    -e SIM_G_API_URL `
    -e SIM_G_TEST_IDENTIFIER `
    -e SIM_G_TEST_PASSWORD `
    -e SIM_G_TEST_USER_ID `
    -e SIM_G_JEMAAT_IDENTIFIER `
    -e SIM_G_JEMAAT_PASSWORD `
    golang:1.26.6 `
    go test ./tests/api -v

if ($LASTEXITCODE -ne 0) {
    throw "API integration tests failed. Check the output above."
}

Write-Host ""
Write-Host "API integration tests passed." -ForegroundColor Green