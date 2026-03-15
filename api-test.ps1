# Test 1: Health check
$response = Invoke-RestMethod -Uri "http://localhost/api/v1/health"
Write-Host "Test 1: Health check - $($response.status)"

# Test 2: Login with correct credentials
$response = Invoke-RestMethod -Uri "http://localhost/api/v1/auth/login" -Method POST -ContentType "application/json" -Body '{"username":"admin","password":"Admin@123456"}'
$token = $response.token
Write-Host "Test 2: Token received: $($token.Substring(0,20))..."

# Test 3: Get devices list
$devices = Invoke-RestMethod -Uri "http://localhost/api/v1/devices" -Headers @{Authorization="Bearer $token"}
Write-Host "Test 3: Devices count - $($devices.Count)"

# Test 4: Get alerts
$alerts = Invoke-RestMethod -Uri "http://localhost/api/v1/alerts" -Headers @{Authorization="Bearer $token"}
Write-Host "Test 4: Alerts count - $($alerts.Count)"

# Test 5: Wrong password returns 401
try {
  Invoke-RestMethod -Uri "http://localhost/api/v1/auth/login" -Method POST -ContentType "application/json" -Body '{"username":"admin","password":"wrongpassword"}'
  Write-Host "Test 5: Failed - expected 401"
} catch {
  Write-Host "Test 5: Correctly returned error: $($_.Exception.Message)"
}

# Test 6: No token returns 401
try {
  Invoke-RestMethod -Uri "http://localhost/api/v1/devices"
  Write-Host "Test 6: Failed - expected 401"
} catch {
  Write-Host "Test 6: Correctly returned 401: $($_.Exception.Message)"
}
