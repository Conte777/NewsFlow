#!/usr/bin/env pwsh
# QR Authentication Script for Account Service
# Usage: ./qr-auth.ps1 [-SessionName "my-account"] [-ServiceUrl "http://localhost:8084"]

param(
    [string]$SessionName = "telegram-account",
    [string]$ServiceUrl = "http://localhost:8084",
    [string]$OutputPath = "$PSScriptRoot\..\qr_code.png"
)

$ErrorActionPreference = "Stop"

Write-Host "Starting QR authentication..." -ForegroundColor Cyan

# Start QR auth session
$body = @{ session_name = $SessionName } | ConvertTo-Json
$response = Invoke-RestMethod -Uri "$ServiceUrl/api/v1/auth/qr/start" -Method POST -Body $body -ContentType "application/json"

$sessionId = $response.session_id
Write-Host "Session ID: $sessionId" -ForegroundColor Yellow

# Save QR code image
$qrBytes = [System.Convert]::FromBase64String($response.qr_code_base64)
[System.IO.File]::WriteAllBytes($OutputPath, $qrBytes)
Write-Host "QR code saved to: $OutputPath" -ForegroundColor Green

# Open QR code image
Start-Process $OutputPath

Write-Host "`nScan the QR code in Telegram: Settings -> Devices -> Link Desktop Device" -ForegroundColor Cyan
Write-Host "Polling for status..." -ForegroundColor Gray

# Poll for status
$maxAttempts = 150  # 5 minutes with 2 second intervals
$attempt = 0

while ($attempt -lt $maxAttempts) {
    Start-Sleep -Seconds 2
    $attempt++

    try {
        $status = Invoke-RestMethod -Uri "$ServiceUrl/api/v1/auth/qr/$sessionId/status" -Method GET

        switch ($status.status) {
            "success" {
                Write-Host "`nAuthentication successful!" -ForegroundColor Green
                Write-Host "Phone: $($status.phone_number)" -ForegroundColor Green
                Write-Host "Account ID: $($status.account_id)" -ForegroundColor Green
                exit 0
            }
            "waiting_password" {
                Write-Host "`n2FA password required!" -ForegroundColor Yellow
                $password = Read-Host -Prompt "Enter 2FA password" -AsSecureString
                $plainPassword = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
                    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($password)
                )

                $pwBody = @{ password = $plainPassword } | ConvertTo-Json
                $pwResponse = Invoke-RestMethod -Uri "$ServiceUrl/api/v1/auth/qr/$sessionId/password" -Method POST -Body $pwBody -ContentType "application/json"
                Write-Host "Password submitted, waiting for result..." -ForegroundColor Gray
            }
            "failed" {
                Write-Host "`nAuthentication failed: $($status.error)" -ForegroundColor Red
                exit 1
            }
            "expired" {
                Write-Host "`nSession expired" -ForegroundColor Red
                exit 1
            }
            "cancelled" {
                Write-Host "`nSession cancelled" -ForegroundColor Yellow
                exit 1
            }
            "pending" {
                # Update QR code if available
                if ($status.qr_code_base64) {
                    $newQrBytes = [System.Convert]::FromBase64String($status.qr_code_base64)
                    [System.IO.File]::WriteAllBytes($OutputPath, $newQrBytes)
                }
                Write-Host "." -NoNewline
            }
        }
    }
    catch {
        Write-Host "`nError checking status: $_" -ForegroundColor Red
    }
}

Write-Host "`nTimeout waiting for authentication" -ForegroundColor Red
exit 1
