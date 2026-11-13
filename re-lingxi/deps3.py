"""技能命令面 × 外部依赖 逐文件矩阵 v3。
对每个 skill 的每个 .py：
  - 解析本地模块（同技能内可解析，不算第三方）
  - 第三方 import：预装 / 未预装
  - add_parser("...") 子命令
  - 云依赖标记：模块名或字符串命中（transcoder/kdocs/wps_http/api.wps.cn/lingxi.wps.cn/365.kdocs.cn/WPS_SID）
输出 JSON + 控制台表。
用法: python deps3.py [out.json]
"""
import ast
import json
import os
import re
import sys

ROOT = os.path.join(os.environ["APPDATA"], "WPS 灵犀", "app_server_5.24.0")
BASES = {"skills": os.path.join(ROOT, "skills"), "builtin": os.path.join(ROOT, "builtin_skill")}
SP = os.path.join(os.environ["APPDATA"], "WPS 灵犀", "python-env", "Lib", "site-packages")
PRE = set()
for n in os.listdir(SP):
    if n.endswith(".dist-info"):
        PRE.add(n[: -len(".dist-info")].rsplit("-", 1)[0].lower().replace("_", "-"))
STD = set(sys.stdlib_module_names)
CLOUD_MOD = re.compile(r"(transcoder|kdocs|wps_http|kdocs_bridge|lbo_cli)", re.I)
CLOUD_HOST = re.compile(r"(lingxi\.wps\.cn|api\.wps\.cn|365\.kdocs\.cn|www\.kdocs\.cn|ksyuncs\.com|wpscdn\.cn|cowork\.wps\.cn)")
PARSER_RE = re.compile(r"""add_parser\(\s*["']([\w\-]+)["']""")
# import 名 -> 发行包名（用于对照 site-packages 的 dist-info）
DIST = {"PIL": "pillow", "bs4": "beautifulsoup4", "docx": "python-docx", "pptx": "python-pptx",
        "yaml": "pyyaml", "cv2": "opencv-python", "sklearn": "scikit-learn", "Xlib": "python-xlib",
        "pdfplumber": "pdfplumber", "pypdf": "pypdf", "formulas": "formulas", "mutagen": "mutagen",
        "cdp_use": "cdp-use", "httpx": "httpx", "aiohttp": "aiohttp", "jinja2": "jinja2",
        "rdkit": "rdkit", "requests": "requests", "numpy": "numpy", "openpyxl": "openpyxl",
        "lxml": "lxml", "fontTools": "fonttools", "matplotlib": "matplotlib", "pypdfium2": "pypdfium2"}


def local_modules(sdir):
    names = set()
    for dirpath, dirs, files in os.walk(sdir):
        for d in dirs:
            if os.path.exists(os.path.join(dirpath, d, "__init__.py")):
                names.add(d)
        for f in files:
            if f.endswith(".py"):
                names.add(f[:-3])
    return names


result = {}
for label, base in BASES.items():
    if not os.path.isdir(base):
        continue
    for skill in sorted(os.listdir(base)):
        sdir = os.path.join(base, skill)
        if not os.path.isdir(sdir):
            continue
        local = local_modules(sdir)
        files = {}
        for dirpath, _, fnames in os.walk(sdir):
            for f in fnames:
                if not f.endswith(".py"):
                    continue
                p = os.path.join(dirpath, f)
                rel = os.path.relpath(p, sdir).replace("\\", "/")
                src = open(p, encoding="utf-8", errors="ignore").read()
                mods = set()
                try:
                    tree = ast.parse(src)
                    for node in ast.walk(tree):
                        if isinstance(node, ast.Import):
                            mods |= {a.name.split(".")[0] for a in node.names}
                        elif isinstance(node, ast.ImportFrom) and node.level == 0 and node.module:
                            mods.add(node.module.split(".")[0])
                except SyntaxError:
                    pass
                third, miss = set(), set()
                for m in sorted(mods):
                    if m in STD or m in local:
                        continue
                    dist = DIST.get(m, m).lower().replace("_", "-")
                    third.add(m)
                    if dist in PRE or dist.replace("-", "") in {x.replace("-", "") for x in PRE}:
                        continue
                    miss.add(m)
                cm = sorted({m for m in mods if CLOUD_MOD.search(m)})
                ch = sorted(set(CLOUD_HOST.findall(src)))
                files[rel] = {
                    "third_party": sorted(third),
                    "missing_pkgs": sorted(miss),
                    "cloud_modules": cm,
                    "cloud_hosts": ch,
                    "cloud_env": bool(re.search(r"WPS_SID|WPS_API_BASE|KDOCS_CLI", src)),
                    "subcommands": sorted(set(PARSER_RE.findall(src))),
                    "kb": os.path.getsize(p) // 1024,
                }
        result[f"{label}/{skill}"] = {"local_modules": sorted(local), "files": files}

with open(sys.argv[1] if len(sys.argv) > 1 else "deps3.json", "w", encoding="utf-8") as fh:
    json.dump(result, fh, ensure_ascii=False, indent=1)

for skill, e in result.items():
    print(f"\n### {skill}")
    for rel, f in sorted(e["files"].items()):
        marks = []
        if f["missing_pkgs"]:
            marks.append("缺包:" + ",".join(f["missing_pkgs"]))
        if f["cloud_modules"]:
            marks.append("云模块:" + ",".join(f["cloud_modules"]))
        if f["cloud_hosts"]:
            marks.append("云域名:" + ",".join(f["cloud_hosts"]))
        if f["cloud_env"]:
            marks.append("云凭据env")
        subs = f["subcommands"]
        print(f"  {rel:52}{f['kb']:5}KB  {' | '.join(marks) if marks else 'local-only'}"
              + (f"  cmds={','.join(subs[:12])}{'…' if len(subs) > 12 else ''}" if subs else ""))
