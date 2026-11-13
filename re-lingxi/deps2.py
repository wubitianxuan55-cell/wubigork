"""技能脚本外部依赖矩阵 v2（ast 精确解析）：
   每个技能 -> 第三方 import（区分预装/未预装）、外部域名、外部可执行、环境变量、脚本清单。
用法: python deps2.py [out.json]
"""
import ast
import json
import os
import re
import sys

ROOT = os.path.join(os.environ["APPDATA"], "WPS 灵犀", "app_server_5.24.0")
BASES = {"skills": os.path.join(ROOT, "skills"), "builtin": os.path.join(ROOT, "builtin_skill")}
SP = os.path.join(os.environ["APPDATA"], "WPS 灵犀", "python-env", "Lib", "site-packages")

PREINSTALLED = {}
for n in os.listdir(SP):
    if n.endswith(".dist-info"):
        name, ver = n[: -len(".dist-info")].rsplit("-", 1) if n.count("-") >= 1 else (n, "")
        PREINSTALLED[name.lower().replace("_", "-")] = ver

STD = set(sys.stdlib_module_names)

HOST_RE = re.compile(r"https?://([A-Za-z0-9._\-]+)")
ENV_RE = re.compile(r"""(?:environ(?:\.get)?\(|getenv\()\s*['"]([A-Z][A-Z0-9_]{2,})['"]""")
BIN_HINTS = {"lbo-cli", "wpscli", "kdocs-cli", "kdocs", "node", "ffmpeg", "ffprobe", "git",
             "python", "python3", "soffice", "libreoffice", "tesseract", "pandoc", "chrome",
             "chromium", "msedge", "msedgewebview2", "curl", "wget", "7z", "unzip"}


def scan_py(path):
    mods, exes = set(), set()
    try:
        src = open(path, encoding="utf-8", errors="ignore").read()
        tree = ast.parse(src)
    except (OSError, SyntaxError):
        return mods, exes, set(), set(), ""
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for a in node.names:
                mods.add(a.name.split(".")[0])
        elif isinstance(node, ast.ImportFrom):
            if node.level == 0 and node.module:
                mods.add(node.module.split(".")[0])
    # 外部可执行：subprocess.* 调用里的字符串首参数
    for node in ast.walk(tree):
        if isinstance(node, ast.Call):
            fn = node.func
            name = getattr(fn, "attr", None) or getattr(fn, "id", None)
            if name in {"run", "Popen", "call", "check_output", "check_call", "execv", "spawnv"}:
                for arg in node.args[:1]:
                    if isinstance(arg, ast.Constant) and isinstance(arg.value, str):
                        exes.add(os.path.basename(arg.value))
                    elif isinstance(arg, (ast.List, ast.Tuple)) and arg.elts:
                        e0 = arg.elts[0]
                        if isinstance(e0, ast.Constant) and isinstance(e0.value, str):
                            exes.add(os.path.basename(e0.value))
    hosts = set(HOST_RE.findall(src))
    envs = set(ENV_RE.findall(src))
    return mods, exes, hosts, envs, src


def frontmatter(p):
    txt = open(p, encoding="utf-8", errors="ignore").read()
    m = re.match(r"^---\n(.*?)\n---", txt, re.S)
    out = {}
    if m:
        for line in m.group(1).splitlines():
            if ":" in line and not line.startswith((" ", "\t")):
                k, v = line.split(":", 1)
                out[k.strip()] = v.strip()
    return out


result = {}
for label, base in BASES.items():
    if not os.path.isdir(base):
        continue
    for skill in sorted(os.listdir(base)):
        sdir = os.path.join(base, skill)
        if not os.path.isdir(sdir):
            continue
        e = {"kind": label, "py": [], "md": [], "third_party": {}, "hosts": [], "exes": [],
             "env": [], "frontmatter": {}, "bytes": 0}
        mods, exes, hosts, envs = set(), set(), set(), set()
        for dirpath, _, files in os.walk(sdir):
            for f in files:
                p = os.path.join(dirpath, f)
                rel = os.path.relpath(p, sdir).replace("\\", "/")
                e["bytes"] += os.path.getsize(p)
                if f.endswith(".py"):
                    e["py"].append(rel)
                    m, x, h, v, _ = scan_py(p)
                    mods |= m
                    exes |= x
                    hosts |= h
                    envs |= v
                elif f.endswith(".md"):
                    e["md"].append(rel)
                    if rel.upper().startswith("SKILL"):
                        e["frontmatter"] = frontmatter(p)
        tp = {}
        for m in sorted(mods):
            if m in STD:
                continue
            key = m.lower().replace("_", "-")
            pre = next((v for k, v in PREINSTALLED.items() if k.replace("-", "") == key.replace("-", "")), None)
            tp[m] = "preinstalled" if pre is not None else "MISSING"
        e["third_party"] = tp
        e["hosts"] = sorted(hosts)
        e["exes"] = sorted(x for x in exes if x)
        e["env"] = sorted(envs)
        result[f"{label}/{skill}"] = e

out = {"preinstalled": PREINSTALLED, "skills": result}
with open(sys.argv[1] if len(sys.argv) > 1 else "deps2.json", "w", encoding="utf-8") as fh:
    json.dump(out, fh, ensure_ascii=False, indent=1)

print(f"{'skill':24}{'py':>3}{'md':>3}  {'KB':>6}  第三方包（★=python-env 未预装）")
for n, e in result.items():
    tp = ", ".join(("★" if s == "MISSING" else "") + m for m, s in e["third_party"].items())
    print(f"{n:24}{len(e['py']):3}{len(e['md']):3}  {e['bytes']//1024:6}  {tp}")
print("\n--- 外部域名 ---")
for n, e in result.items():
    if e["hosts"]:
        print(f"{n:24} {', '.join(e['hosts'])}")
print("\n--- 外部可执行 ---")
for n, e in result.items():
    if e["exes"]:
        print(f"{n:24} {', '.join(e['exes'])}")
