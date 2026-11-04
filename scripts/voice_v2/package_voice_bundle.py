# package_voice_bundle.py — 语音 V2 bundle 打包脚本（bundle v2，2026-11-05）。
#
# sidecar/venv/4B 已随 V1 语音链退役——bundle v2 = 全新净布局：
#   bundle/
#     gaea.exe                        # V2-only 构建（语音引擎进程内：realtime/EOT/STT/VAD）
#     onnxruntime.dll                 # 语音引擎三 DLL（铁律：必须与 exe 同目录，System32 劫持前科）
#     sherpa-onnx-c-api.dll
#     sherpa-onnx-cxx-api.dll
#     models/voice/                   # silero_vad_v5_16k / smart-turn-v3.2 / paraformer int8 / tokens
#     scripts/voice_v2/               # 手动回滚物（保留一版期，非运行依赖）
#     gaea_install.ps1                # 装机/更新（复制到 C:\AI\gaea-app + 桌面快捷方式）
#     BUNDLE.md                       # 布局说明 + SHA256 清单
#
# 模型解析：进程按 GAEA_VOICE_V2_BASE -> <base>/models/voice/ 优先，回落
# C:\AI\models\voice\（统一根）——bundle 自带模型即离线完整。
#
# 已知边界（如实记录）：
#   - breeze TTS(8012) 是用户本地引擎，不入 bundle（breeze 权重非商业红线）；
#     TTS 腿按需拉起（ensure 链）。
#   - 跨机分发需目标机 WebView2 运行时（Wails 前提）。
#
# 用法：
#   python package_voice_bundle.py --out <bundle-dir> [--exe <gaea.exe>] [--smoke]
#   --smoke：布局自洽校验（四件+模型+脚本齐全，DLL 与 exe 同目录）。
from __future__ import annotations

import argparse
import hashlib
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
BUILD_BIN = REPO / "build" / "bin"
UNIFIED_MODELS = Path(r"C:\AI\models\voice")
VOICE_MODEL_FILES = [
    "silero_vad_v5_16k.onnx",
    "smart-turn-v3.2-cpu.onnx",
    "model.int8.onnx",
    "tokens.txt",
]
DLL_FILES = ["onnxruntime.dll", "sherpa-onnx-c-api.dll", "sherpa-onnx-cxx-api.dll"]


def sha256_of(p: Path) -> str:
    h = hashlib.sha256()
    with p.open("rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def stage(out: Path, exe: Path) -> list[str]:
    problems: list[str] = []
    out.mkdir(parents=True, exist_ok=True)
    (out / "models" / "voice").mkdir(parents=True, exist_ok=True)

    # exe + 三 DLL（同目录铁律）
    shutil.copy2(exe, out / "gaea.exe")
    print(f"  copy {exe}")
    for name in DLL_FILES:
        src = exe.parent / name
        if src.is_file():
            shutil.copy2(src, out / name)
            print(f"  copy {src}")
        else:
            problems.append(f"DLL 缺失: {src}（cgo 构建须随 exe 派发）")

    # 语音模型（统一根 -> bundle）
    for name in VOICE_MODEL_FILES:
        src = UNIFIED_MODELS / name
        if src.is_file():
            shutil.copy2(src, out / "models" / "voice" / name)
            print(f"  copy {src}")
        else:
            problems.append(f"语音模型缺失: {src}")

    # 手动回滚物（保留一版期，非运行依赖）
    roll = REPO / "scripts" / "voice_v2"
    dst = out / "scripts" / "voice_v2"
    dst.mkdir(parents=True, exist_ok=True)
    for name in ["gaea-spike.yaml", "gaea_sidecar_launcher.py", "gaea_sidecar_headers.py"]:
        src = roll / name
        if src.is_file():
            shutil.copy2(src, dst / name)

    # 装机脚本
    installer = REPO / "scripts" / "gaea_install.ps1"
    if installer.is_file():
        shutil.copy2(installer, out / "gaea_install.ps1")
    else:
        problems.append(f"装机脚本缺失: {installer}")

    # 布局说明 + SHA256 清单
    lines = [
        "# gaea voice bundle v2",
        "",
        "- 语音对话唯一路径 = V2 Go 原生管线（进程内 realtime 8791 + breeze TTS 腿 + 聊天绑定 LLM 腿）。",
        "- 铁律：三只 DLL 必须与 gaea.exe 同目录（Windows DLL 搜索序 + System32 旧版 ORT 劫持前科）。",
        "- 模型解析：GAEA_VOICE_V2_BASE=<本目录> 时优先 <base>/models/voice/，回落统一根 C:/AI/models/voice。",
        "- 装机：运行 gaea_install.ps1 -Launch（复制到 C:/AI/gaea-app + 桌面快捷方式）。",
        "",
        "## SHA256",
        "",
    ]
    for f in sorted(out.rglob("*")):
        if f.is_file() and f.suffix in {".exe", ".dll", ".onnx", ".ps1"}:
            lines.append(f"- {f.relative_to(out)}: {sha256_of(f)[:16]}")
    (out / "BUNDLE.md").write_text("\n".join(lines), encoding="utf-8")
    return problems


def smoke(out: Path) -> int:
    failures = 0
    must = ["gaea.exe", "BUNDLE.md", "gaea_install.ps1", *DLL_FILES,
            *(f"models/voice/{n}" for n in VOICE_MODEL_FILES)]
    for rel in must:
        p = out / rel
        ok = p.is_file()
        print(f"  {'PASS' if ok else 'FAIL'} {rel}")
        failures += 0 if ok else 1
    return failures


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--exe", default=str(BUILD_BIN / "gaea.exe"))
    ap.add_argument("--smoke", action="store_true")
    args = ap.parse_args()
    out = Path(args.out)
    exe = Path(args.exe)
    if not exe.is_file():
        print(f"FAIL exe 缺失: {exe}", file=sys.stderr)
        return 2
    problems = stage(out, exe)
    if args.smoke:
        failures = smoke(out)
        if failures or problems:
            for pr in problems:
                print(f"FAIL {pr}", file=sys.stderr)
            return 1
    elif problems:
        for pr in problems:
            print(f"FAIL {pr}", file=sys.stderr)
        return 1
    print(f"OK bundle -> {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
