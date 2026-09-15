@echo off
title MailMind Launcher
echo ========================================================
echo               Starting MailMind Pipeline...
echo ========================================================
echo.

cd /d "%~dp0"

echo [1/3] Starting Go Backend API Server (Port 8080)...
start "MailMind Backend" /min cmd /c "cd /d ""%~dp0"" && go run ./cmd/server"

echo [2/3] Starting Next.js Web Client (Port 3001)...
start "MailMind Frontend" /min cmd /c "cd /d ""%~dp0frontend"" && npm run dev"

echo [3/3] Waiting for servers to initialize...
set "BACKEND_READY="
set "FRONTEND_READY="
set /a RETRIES=0

:wait_loop
if not defined BACKEND_READY (
	powershell -NoProfile -Command "try { if ((Invoke-WebRequest -UseBasicParsing -Uri 'http://127.0.0.1:8080/health' -TimeoutSec 1).StatusCode -eq 200) { exit 0 } } catch {}; exit 1" >nul 2>&1
	if not errorlevel 1 set "BACKEND_READY=1"
)
if not defined FRONTEND_READY (
	powershell -NoProfile -Command "try { if ((Invoke-WebRequest -UseBasicParsing -Uri 'http://127.0.0.1:3001' -TimeoutSec 1).StatusCode -eq 200) { exit 0 } } catch {}; exit 1" >nul 2>&1
	if not errorlevel 1 set "FRONTEND_READY=1"
)

if defined BACKEND_READY if defined FRONTEND_READY goto services_ready

set /a RETRIES+=1
if %RETRIES% GEQ 30 goto wait_timeout

timeout /t 1 /nobreak >nul
goto wait_loop

:wait_timeout
echo.
if not defined BACKEND_READY echo Backend did not become ready at http://localhost:8080
if not defined FRONTEND_READY echo Frontend did not become ready at http://localhost:3001
goto startup_failed

:services_ready

echo.
echo ========================================================
echo   MailMind is running!
echo   Frontend: http://localhost:3001
echo   Backend:  http://localhost:8080
echo ========================================================
echo.
echo Opening MailMind in your browser...
start http://localhost:3001

exit /b 0

:startup_failed
echo.
echo MailMind could not start completely. Check the backend and frontend windows for details.
pause
exit /b 1
