@echo off
REM wubigork 桌面端构建脚本
REM 产物固定输出到 build\bin\wubigork.exe
cd /d C:\AI\wubigrok
echo === wubigork 桌面端构建 ===
wails build
echo.
echo 产物: C:\AI\wubigrok\build\bin\wubigork.exe
