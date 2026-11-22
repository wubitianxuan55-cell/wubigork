@echo off
rem cgo_test_dll_prep.cmd - go test -exec wrapper (System32 onnxruntime.dll hijack fix).
rem Since 2026-11-11 Windows ships ORT 1.17.1 in System32 (OS component). go test
rem binaries live in a temp dir with no DLLs, so their load-time imports resolve
rem to System32 and the voice cgo tests crash (0xc0000005, "requested API version
rem [28] not available"). This wrapper copies the DLL trio next to the test
rem binary (exe dir is first in the search order), then runs the test binary.
rem DLL source dir can be overridden with GAEA_TEST_DLL_DIR; default build\bin.
setlocal
set "DLLDIR=%GAEA_TEST_DLL_DIR%"
if "%DLLDIR%"=="" set "DLLDIR=C:\AI\wubigrok\build\bin"
if exist "%DLLDIR%\onnxruntime.dll" copy /Y "%DLLDIR%\onnxruntime.dll" "%~dp1" >nul
if exist "%DLLDIR%\sherpa-onnx-c-api.dll" copy /Y "%DLLDIR%\sherpa-onnx-c-api.dll" "%~dp1" >nul
if exist "%DLLDIR%\sherpa-onnx-cxx-api.dll" copy /Y "%DLLDIR%\sherpa-onnx-cxx-api.dll" "%~dp1" >nul
endlocal & %*
