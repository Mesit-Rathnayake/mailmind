@echo off
title Stop MailMind
echo ========================================================
echo               Stopping MailMind Services...
echo ========================================================
echo.

echo Stopping Frontend (Port 3001)...
for /f "tokens=5" %%a in ('netstat -aon ^| findstr ":3001" ^| findstr "LISTENING"') do (
    taskkill /F /PID %%a >nul 2>&1
)

echo Stopping Backend (Port 8080)...
for /f "tokens=5" %%a in ('netstat -aon ^| findstr ":8080" ^| findstr "LISTENING"') do (
    taskkill /F /PID %%a >nul 2>&1
)

echo.
echo ========================================================
echo   MailMind has been stopped successfully!
echo ========================================================
echo.
timeout /t 2 >nul
