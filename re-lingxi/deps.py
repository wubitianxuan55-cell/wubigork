"""技能脚本外部依赖矩阵：扫 skills/ + builtin_skill/，产出每个技能的
   imports / 外部域名 / 外部可执行 / 环境变量 / 脚本清单，并对照 python-env 预装包。
用法: python deps.py [out.json]
"""
import json
import os
import re
import sys
from collections import defaultdict

ROOT = os.path.join(os.environ["APPDATA"], "WPS 灵犀", "app_server_5.24.0")
SKILL_DIRS = {"skills": os.path.join(ROOT, "skills"), "builtin": os.path.join(ROOT, "builtin_skill")}
PREINSTALLED = set()
SP = os.path.join(os.environ["APPDATA"], "WPS 灵犀", "python-env", "Lib", "site-packages")
for n in os.listdir(SP):
    if n.endswith(".dist-info"):
        PREINSTALLED.add(n.rsplit("-", 1)[0].lower().replace("_", "-"))
    elif n.endswith(".egg-info"):
        PREINSTALLED.add(n.split("-")[0].lower().replace("_", "-"))
STDLIB_HINT = {"os", "sys", "re", "json", "time", "pathlib", "subprocess", "argparse", "shutil",
               "tempfile", "typing", "datetime", "collections", "io", "math", "csv", "base64",
               "hashlib", "textwrap", "random", "itertools", "functools", "glob", "zipfile",
               "urllib", "uuid", "logging", "copy", "string", "struct", "asyncio", "socket",
               "http", "xml", "decimal", "statistics", "unicodedata", "traceback", "dataclasses",
               "enum", "contextlib", "warnings", "platform", "shlex", "binascii", "codecs",
               "concurrent", "queue", "signal", "threading", "email", "mimetypes", "calendar",
               "__future__", "abc", "difflib", "html", "importlib", "inspect", "operator",
               "pickle", "pprint", "secrets", "textwrap", "unittest", "venv", "webbrowser",
               "colorsys", "errno", "filecmp", "fnmatch", "getpass", "gzip", "hmac", "ipaddress",
               "locale", "numbers", "pdb", "pkgutil", "posixpath", "shutil", "ssl", "stat",
               "subprocess", "tarfile", "tempfile", "time", "types", "unittest", "warnings"}

IMPORT_RE = re.compile(r"^\s*(?:from\s+([A-Za-z_][\w.]*)|import\s+([A-Za-z_][\w.,\s]*))", re.M)
HOST_RE = re.compile(r"https?://([A-Za-z0-9._~:\-]+)")
ENV_RE = re.compile(r"os\.environ(?:\.get)?\(?\[?['\"]([A-Z][A-Z0-9_]{2,})['\"]")
BIN_RE = re.compile(r"['\"]([a-z][a-z0-9_\-]{2,})\b", re.I)


def top_mods(path):
    mods = set()
    try:
        src = open(path, encoding="utf-8", errors="ignore").read()
    except OSError:
        return mods, set(), set(), set()
    for m in IMPORT_RE.finditer(src):
        raw = m.group(1) or m.group(2) or ""
        for part in raw.split(","):
            part = part.strip().split(" ")[0]
            if not part:
                continue
            mods.add(part.split(".")[0])
    hosts = {h.split("/")[0] for h in HOST_RE.findall(src)}
    envs = set(ENV_RE.findall(src))
    return mods, hosts, envs, src


result = {}
for label, base in SKILL_DIRS.items():
    if not os.path.isdir(base):
        continue
    for skill in sorted(os.listdir(base)):
        sdir = os.path.join(base, skill)
        if not os.path.isdir(sdir):
            continue
        entry = {"kind": label, "py": [], "md": [], "third_party": [], "hosts": [],
                 "env": [], "frontmatter": {}}
        tin, hosts, envs = set(), set(), set()
        for dirpath, _, files in os.walk(sdir):
            for f in files:
                p = os.path.join(dirpath, f)
                rel = os.path.relpath(p, sdir).replace("\\", "/")
                if f.endswith(".py"):
                    entry["py"].append(rel)
                    m, h, e, _ = top_mods(p)
                    tin |= {x for x in m if x not in STDLIB_HINT}
                    hosts |= h
                    envs |= e
                elif f.endswith(".md"):
                    entry["md"].append(rel)
                    if rel.upper().startswith("SKILL"):
                        txt = open(p, encoding="utf-8", errors="ignore").read()
                        fm = re.match(r"^---\n(.*?)\n---", txt, re.S)
                        if fm:
                            for line in fm.group(1).splitlines():
                                if ":" in line:
                                    k, v = line.split(":", 1)
                                    entry["frontmatter"][k.strip()] = v.strip()
        entry["third_party"] = sorted(tin)
        entry["missing_from_preinstall"] = sorted(
            x for x in tin if x.lower().replace("_", "-") not in PREINSTALLED
        )
        entry["hosts"] = sorted(hosts)
        entry["env"] = sorted(envs)
        result[f"{label}/{skill}"] = entry

out = {"preinstalled": sorted(PREINSTALLED), "skills": result}
path = sys.argv[1] if len(sys.argv) > 1 else "deps.json"
with open(path, "w", encoding="utf-8") as fh:
    json.dump(out, fh, ensure_ascii=False, indent=1)

print(f"{'skill':22} {'py':>3} {'md':>3}  third-party imports (★=非预装)")
for name, e in result.items():
    tp = ", ".join(("*" if m in e["missing_from_preinstall"] else "") + m for m in e["third_party"])
    print(f"{name:22} {len(e['py']):3} {len(e['md']):3}  {tp}")
print("\n--- 外部域名 ---")
for name, e in result.items():
    if e["hosts"]:
        print(f"{name:22} {', '.join(e['hosts'])}")
print("\n--- 环境变量 ---")
for name, e in result.items():
    if e["env"]:
        print(f"{name:22} {', '.join(e['env'])}")
