@echo off
REM 1Panel.edu scheduler: 安装开机自启
REM 部署: 复制 1panel-monitor.vbs 到 Start Menu 启动文件夹
REM 不需 admin
setlocal

set "VBS_SRC=%~dp01panel-monitor-hidden.vbs"
set "STARTUP_DIR=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "VBS_DST=%STARTUP_DIR%\1panel-monitor.vbs"

if not exist "%VBS_SRC%" (
    echo ERROR: %VBS_SRC% not found
    exit /b 1
)

if not exist "%STARTUP_DIR%" (
    echo ERROR: %STARTUP_DIR% not found
    exit /b 1
)

copy /Y "%VBS_SRC%" "%VBS_DST%" >nul
if errorlevel 1 (
    echo ERROR: copy failed
    exit /b 1
)

echo Installed: %VBS_DST%
echo Reboot to test, or run now:
echo     wscript "%VBS_DST%"
endlocal
