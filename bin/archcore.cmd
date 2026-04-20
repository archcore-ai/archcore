@echo off
rem archcore shim launcher (Windows cmd). Delegates to archcore.ps1.
powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "%~dp0archcore.ps1" %*
exit /b %ERRORLEVEL%
