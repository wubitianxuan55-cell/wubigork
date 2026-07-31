@echo off
REM ============================================================
REM Z-Image-Turbo 模型下载脚本 (gaea)
REM 需要下载 2 个文件到 ComfyUI 模型目录
REM 总计约 9.6 GB — 建议用 VPN 或代理下载
REM ============================================================

set DEST_DIR=D:\AI\ComfyUI\models

echo.
echo === Z-Image-Turbo 模型下载指南 ===
echo.

echo [1/2] Z-Image-Turbo Q5_K_M GGUF (约 5.2 GB)
echo     下载: https://huggingface.co/jayn7/Z-Image-Turbo-GGUF/resolve/main/z_image_turbo-Q5_K_M.gguf
echo     镜像: https://hf-mirror.com/jayn7/Z-Image-Turbo-GGUF/resolve/main/z_image_turbo-Q5_K_M.gguf
echo     放到: %DEST_DIR%\diffusion_models\
echo.

echo [2/2] Qwen3-4B Q4_K_M GGUF (约 2.5 GB)
echo     下载: https://huggingface.co/mradermacher/Qwen3-4B-i1-GGUF/resolve/main/Qwen3-4B.i1-Q4_K_M.gguf
echo     镜像: https://hf-mirror.com/mradermacher/Qwen3-4B-i1-GGUF/resolve/main/Qwen3-4B.i1-Q4_K_M.gguf
echo     放到: %DEST_DIR%\text_encoders\
echo.

echo === 验证清单 ===
echo [ ] %DEST_DIR%\diffusion_models\z_image_turbo-Q5_K_M.gguf
echo [ ] %DEST_DIR%\text_encoders\Qwen3-4B.i1-Q4_K_M.gguf
echo [x] %DEST_DIR%\vae\ae.safetensors (已存在)
echo.

echo 下载完成后启动 ComfyUI 即可在 gaea 中使用。
pause
