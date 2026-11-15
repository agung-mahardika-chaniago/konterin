@echo off
setlocal
cd /d "%~dp0"
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\fixphone.ps1" stop
echo.
echo Tekan tombol apa saja untuk menutup jendela ini.
pause >nul
endlocal