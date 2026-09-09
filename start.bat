@echo off
title MailMind Launcher
echo ========================================================
echo               Starting MailMind Pipeline...
echo ========================================================
echo.

cd /d "%~dp0"

echo [1/3] Starting Go Backend API Server (Port 8080)...
start "MailMind Backend" /min cmd /c "go run ./cmd/server"

echo [2/3] Starting Next.js Web Client (Port 3001)...
start "MailMind Frontend" /min cmd /c "cd frontend && npm run dev"

echo [3/3] Waiting for servers to initialize...
timeout /t 3 /nobreak >nul

echo.
echo ========================================================
echo   MailMind is running!
echo   Frontend: http://localhost:3001
echo   Backend:  http://localhost:8080
echo ========================================================
echo.
echo Opening MailMind in your browser...
start http://localhost:3001

echo.
echo Press any key to close this launcher window (servers remain running in background).
pause >nul
