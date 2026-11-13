"""E 批探针：① tools/*.tmpl 正文候选 ② kdocs-cli service/action 枚举 ③ IAB 路由（index.js）。"""
import os
import re

BASE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service")
MAIN = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", "x", "out", "main", "index.js")


def load(p):
    return open(p, "rb").read().decode("utf-8", "replace")


lx = load(os.path.join(BASE, "lingxi.exe"))

print("== ① '交付' 串分布（去重后前 40，排除已入库段）==")
seen = set()
for m in re.finditer(r"[\x20-\x7e\u4e00-\u9fff\uff00-\uffef]{18,200}", lx):
    t = re.sub(r"\s+", " ", m.group(0)).strip()
    if "交付" not in t or t in seen:
        continue
    seen.add(t)
    if len(seen) > 40:
        break
    print(f"  {m.start():>9} {t[:150]!r}")

print("\n== ① tool 模板语法候选（含 {{ . 且含 ref/paths/format 的串）==")
for m in re.finditer(r"\{\{[^\x00]{0,160}?(?:ref|paths|format|c_text)[^\x00]{0,160}?\}\}", lx):
    print("  ", re.sub(r"\s+", " ", m.group(0))[:200])

print("\n== ② kdocs-cli service/action 线索 ==")
kd = load(os.path.join(os.environ["LOCALAPPDATA"],
                       r"Programs\lingxi-desktop\resources\kdocs-cli\kdocs-cli.exe"))
for m in list(re.finditer(r"unknown action", kd))[:4]:
    s = m.start()
    print("  -", re.sub(r"\s+", " ", kd[max(0, s - 260):s + 200]))
for k in ["service", "Service", "action", "Action", "flag:", "Unknown action", "available"]:
    print(f"    {k:16} {kd.count(k)}")

print("\n== ③ IAB 路由（主进程 index.js）==")
js = open(MAIN, encoding="utf-8", errors="replace").read() if os.path.exists(MAIN) else ""
print("  index.js found:", bool(js), len(js))
if js:
    routes = sorted(set(re.findall(r'["\'`](/iab/[a-z0-9/_\-]{2,50})["\'`]', js)))
    print("  /iab/* 路由:", routes[:40])
    for k in ['"/local-file', "56305", "httpServer", "createServer", "listen(", "/screenshot",
              "/tabs", "/navigate", "/evaluate", "/files"]:
        print(f"    {k:16} {js.count(k)}")
    for m in list(re.finditer(r'createServer\(', js))[:2]:
        s = m.start()
        print("    ctx:", re.sub(r"\s+", " ", js[max(0, s - 200):s + 240])[:420])
