@echo off
REM gaea desktop build script (ASCII-only messages; Chinese notes live in docs)
REM Usage: build.bat [skip-smoke]
REM   default: wails build -> artifact freshness check -> smoke test -> desktop copy
REM   skip-smoke: skip smoke test for fast iteration (do NOT release without smoke)

cd /d C:\AI\wubigrok

REM Redirect temp dirs to .tmp to avoid Windows SAC/AV scanning policies
if not exist .tmp mkdir .tmp
set TMP=C:\AI\wubigrok\.tmp
set TEMP=C:\AI\wubigrok\.tmp

echo === gaea desktop build ===

REM Delete the previous artifact first: a successful build MUST recreate it.
REM (v4.8.3 lesson: wails build used to print a success block unconditionally;
REM  the only trustworthy proof is a fresh artifact + a real exit code.)
if exist build\bin\gaea.exe (
    del /q build\bin\gaea.exe
    if exist build\bin\gaea.exe (
        echo [FAIL] build\bin\gaea.exe is locked by a running instance - close it and retry
        exit /b 1
    )
)

call wails build
if errorlevel 1 (
    echo [FAIL] wails build failed
    exit /b 1
)
echo [OK] wails build completed

if not exist build\bin\gaea.exe (
    echo [FAIL] build\bin\gaea.exe was not produced by this build
    exit /b 1
)

set ARG=%~1
if /i "%ARG%"=="skip-smoke" goto :skip_smoke

echo === smoke test ===
copy /Y build\bin\gaea.exe .tmp\smoke-gaea.exe >nul
if errorlevel 1 (
    echo [FAIL] cannot stage smoke copy at .tmp\smoke-gaea.exe
    exit /b 1
)
REM pwsh (PowerShell 7) is not on PATH on every dev box; fall back to
REM Windows PowerShell 5.1 (smoke.ps1 uses no PS7-only syntax).
set SMOKE_SHELL=pwsh
where pwsh >nul 2>&1
if errorlevel 1 set SMOKE_SHELL=powershell
%SMOKE_SHELL% -NoProfile -ExecutionPolicy Bypass -File scripts\smoke.ps1 -ExePath .tmp\smoke-gaea.exe
if errorlevel 1 (
    echo [FAIL] smoke test failed - do NOT release this build
    exit /b 1
)
echo [OK] smoke test passed

:skip_smoke

echo === desktop copy ===
copy /Y "build\bin\gaea.exe" "%USERPROFILE%\Desktop\gaea.exe" >nul 2>&1
if errorlevel 1 (
    echo [WARN] desktop copy failed - SAC may block; artifact is still at build\bin\gaea.exe
) else (
    echo [OK] desktop copy done
)

echo.
echo [DONE] artifact: C:\AI\wubigrok\build\bin\gaea.exe
exit /b 0
