@echo off
rem gaea voice-v2 M1 Spike sidecar launcher (speech-to-speech serve).
rem Requires: C:/AI/gaea-voice-v2 venv, herdsman 8080, breeze 8012.
setlocal
set "VENV=C:\AI\gaea-voice-v2\venv"
if not exist "%VENV%\Scripts\python.exe" (
  echo [voice-v2] venv missing: %VENV% - install deps first, see M1 report
  exit /b 1
)
"%VENV%\Scripts\python.exe" -m speech_to_speech.cli serve -f "%~dp0gaea-spike.yaml" %*
