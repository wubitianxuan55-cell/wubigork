"""E 批探针 2：kdocs-cli 的 service/action 面 + 主进程 IAB HTTP 路由。"""
import os
import re

KD = os.path.join(os.environ["LOCALAPPDATA"], r"Programs\lingxi-desktop\resources\kdocs-cli\kdocs-cli.exe")
MAIN = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out", "x", "out", "main", "index.js")

kd = open(KD, "rb").read().decode("utf-8", "replace")

print("== kdocs-cli: unknown action / service 上下文 ==")
for m in list(re.finditer(r"unknown action|unknown service|Error: unknown", kd))[:6]:
    s = m.start()
    print("  -", re.sub(r"\s+", " ", kd[max(0, s - 300):s + 220]))

print("\n== kdocs-cli: 疑似 service 名（三字母/短词 + 紧随 action 列表）==")
for cand in ["drive", "doc", "sheet", "bitable", "wiki", "calendar", "mail", "todo", "team",
             "comment", "revision", "search", "file", "folder", "share", "permission"]:
    if re.search(rf'"{cand}"', kd):
        print(f"  \"{cand}\" 命中 {kd.count(chr(34)+cand+chr(34))}")

print("\n== kdocs-cli: 子命令词（auth/call/feedback/upgrade/help 等）==")
for k in ["\"auth\"", "\"call\"", "\"feedback\"", "\"upgrade\"", "\"help\"", "set-token",
          "\"tool\"", "\"action\"", "\"service\""]:
    print(f"  {k:14} {kd.count(k)}")

print("\n== 主进程 IAB 路由 ==")
js = open(MAIN, encoding="utf-8", errors="replace").read()
for pat in [r'"(/iab/[a-z0-9/_\-]{2,60})"', r'"(/local-file[a-z0-9/_\-]*)"',
            r'`(/iab/[a-z0-9/_\-${}]{2,60})`']:
    got = sorted(set(re.findall(pat, js)))
    print(f"  {pat} -> {len(got)}")
    for g in got[:25]:
        print("    ", g)

print("\n== 主进程 HTTP 路由候选（createServer 附近）==")
idx = js.find("createServer(")
while idx > 0 and idx < len(js):
    seg = js[max(0, idx - 120):idx + 400]
    print("  ...", re.sub(r"\s+", " ", seg)[:380])
    idx = js.find("createServer(", idx + 1)
    if idx > 4_000_000:
        break
