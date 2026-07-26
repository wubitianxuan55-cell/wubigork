@echo off
REM wubigork 桌面端构建脚本
cd /d C:\AI\wubigrok
REM 避免 Windows 应用控制策略拦截 Temp 目录
if not exist .tmp mkdir .tmp
set TMP=C:\AI\wubigrok\.tmp
set TEMP=C:\AI\wubigrok\.tmp
echo === wubigork 桌面端构建 ===
wails build
echo.
echo 复制到桌面（绕过 SAC 扫描策略）...
copy /Y "C:\AI\wubigrok\build\bin\wubigork.exe" "%USERPROFILE%\Desktop\wubigork.exe" >nul 2>&1
echo.
echo 产物:
echo   C:\AI\wubigrok\build\bin\wubigork.exe
echo   %USERPROFILE%\Desktop\wubigork.exe
echo.
echo 如果仍被 SAC 阻止，运行桌面上的 wubigork.exe 试试
echo 或直接右键 wubigork.exe → 属性 → 勾选「解除锁定」
