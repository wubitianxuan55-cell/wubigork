# package_voice_bundle.py — 语音 V2 bundle 打包脚本（P3 产品化收口 刀一）。
#
# stage 出可重定位布局（gaea 进程经 GAEA_VOICE_V2_BASE/GAEA_VOICE_V2_ROOT 指向）：
#   bundle/
#     venv/                            # sidecar Python 环境（整目录复制）
#     models/Qwen3-4B-Q4_K_M.gguf      # 4B 快模型（+sherpa 若在）
#     bin/llama-server.exe             # llama.cpp 服务端（现役二进制）
#     scripts/voice_v2/                # launcher + headers + gaea-spike.yaml
#
# 已知边界（如实记录）：
#   - venv 整目录复制仅保证本机重定位（venv 内嵌解释器绝对路径，跨机脆弱）；
#     跨机分发走安装器内 venv bootstrap 或 §16.3 提取线——本脚本不解决。
#   - breeze TTS(8012) 是用户本地引擎，不入 bundle（breeze 权重非商业红线）。
#
# 用法：
#   python package_voice_bundle.py --out <bundle-dir> [--skip-venv] [--smoke]
#   --smoke：从 bundle 布局冷启 sidecar（等 8791 → TCP 可连 → 杀掉），验证
#   布局自洽（yaml 仅含 URL+端口本就可重定位）。冒烟会短暂占用 8791。
from __future__ import annotations

import argparse
import os
import shutil
import socket
import subprocess
import sys
import time
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
LEGACY_BASE = Path(r"C:\AI\gaea-voice-v2")
LEGACY_LLAMA_BIN = Path(r"C:\AI\uocr2-llama\bin\llama-server.exe")
SCRIPT_FILES = ["gaea_sidecar_launcher.py", "gaea_sidecar_headers.py", "gaea-spike.yaml"]
MODEL_FILES = ["models/Qwen3-4B-Q4_K_M.gguf"]
MODEL_DIRS = ["models/sherpa-zh-14m"]


def copytree(src: Path, dst: Path) -> None:
    print(f"  copy {src} -> {dst}")
    shutil.copytree(src, dst, dirs_exist_ok=True)


def stage(out: Path, skip_venv: bool) -> list[str]:
    problems: list[str] = []
    (out / "scripts" / "voice_v2").mkdir(parents=True, exist_ok=True)
    (out / "bin").mkdir(parents=True, exist_ok=True)
    (out / "models").mkdir(parents=True, exist_ok=True)

    # venv：优先历史装机根
    venv_src = LEGACY_BASE / "venv"
    if not skip_venv:
        if venv_src.is_dir():
            copytree(venv_src, out / "venv")
        else:
            problems.append(f"venv 缺失: {venv_src}")

    # models
    for rel in MODEL_FILES:
        src = LEGACY_BASE / rel
        if src.is_file():
            shutil.copy2(src, out / rel)
            print(f"  copy {src}")
        else:
            problems.append(f"模型缺失: {src}")
    for rel in MODEL_DIRS:
        src = LEGACY_BASE / rel
        if src.is_dir():
            copytree(src, out / rel)
        else:
            print(f"  (可选跳过) {src}")

    # bin：llama-server 现役二进制
    if LEGACY_LLAMA_BIN.is_file():
        shutil.copy2(LEGACY_LLAMA_BIN, out / "bin" / "llama-server.exe")
        print(f"  copy {LEGACY_LLAMA_BIN}")
    else:
        problems.append(f"llama-server 缺失: {LEGACY_LLAMA_BIN}")

    # scripts
    for name in SCRIPT_FILES:
        src = REPO / "scripts" / "voice_v2" / name
        if src.is_file():
            shutil.copy2(src, out / "scripts" / "voice_v2" / name)
        else:
            problems.append(f"脚本缺失: {src}")
    return problems


def wait_port(port: int, seconds: float) -> bool:
    deadline = time.time() + seconds
    while time.time() < deadline:
        with socket.socket() as s:
            s.settimeout(0.5)
            try:
                s.connect(("127.0.0.1", port))
                return True
            except OSError:
                time.sleep(1)
    return False


def smoke(bundle: Path) -> int:
    # 冒烟前提：8791 空闲（真实 sidecar 在跑则先让位）
    with socket.socket() as s:
        try:
            s.bind(("127.0.0.1", 8791))
        except OSError:
            print("SMOKE SKIP: 8791 被占用（真实 sidecar 在跑？先停再冒烟）")
            return 2
    python = bundle / "venv" / "Scripts" / "python.exe"
    launcher = bundle / "scripts" / "voice_v2" / "gaea_sidecar_launcher.py"
    config = bundle / "scripts" / "voice_v2" / "gaea-spike.yaml"
    if not python.is_file():
        print("SMOKE FAIL: bundle venv 不完整")
        return 1
    env = os.environ.copy()
    env["GAEA_VOICE_V2_BASE"] = str(bundle)
    env["GAEA_VOICE_V2_ROOT"] = str(bundle)
    proc = subprocess.Popen(
        [str(python), str(launcher), "serve", "-f", str(config)],
        cwd=str(bundle), env=env,
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
    )
    try:
        if wait_port(8791, 180):
            print("SMOKE PASS: bundle 布局冷启 sidecar 成功（8791 可连）")
            return 0
        print("SMOKE FAIL: 180s 内 8791 未就绪")
        return 1
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=10)
        except subprocess.TimeoutExpired:
            proc.kill()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True, help="bundle 输出目录")
    ap.add_argument("--skip-venv", action="store_true", help="跳过 venv 复制（已有布局补齐用）")
    ap.add_argument("--smoke", action="store_true", help="stage 后从 bundle 冷启 sidecar 冒烟")
    args = ap.parse_args()
    out = Path(args.out).resolve()
    print(f"staging voice bundle -> {out}")
    problems = stage(out, args.skip_venv)
    for p in problems:
        print(f"  [缺] {p}")
    if problems:
        print("STAGE INCOMPLETE")
        return 1
    print("STAGE OK")
    if args.smoke:
        return smoke(out)
    return 0


if __name__ == "__main__":
    sys.exit(main())
