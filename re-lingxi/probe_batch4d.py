"""D 批探针 4：native-host 路由/端口文件、sandbox 两件策略原文、kdocs-cli 命令面。"""
import os
import re

BASE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service")


def load(p):
    return open(p, "rb").read().decode("utf-8", "replace")


nh = load(os.path.join(BASE, "lingxi-native-host.exe"))
print("== native-host：路由 / 端口文件 / 消息类型 ==")
for k in ["/cdp", "/msg", "/healthz", "/v1/", "port", "native-host.json", ".native-host",
          "session_start", "session_end", "cdp.request", "plugin", "Plugin", "token"]:
    print(f"  {k:18} {nh.count(k):4}")
for pat in [r'"/[a-z0-9_/\-]{2,40}"', r"native-host[\\/][a-z0-9\-_.]+"]:
    got = sorted(set(re.findall(pat, nh)))[:20]
    print(f"  pattern {pat} -> {got}")

print("\n== sandbox-setup：WFP / policy 原文 ==")
st = load(os.path.join(BASE, "lingxi-sandbox-setup.exe"))
for m in list(re.finditer(r"(FwpmFilterAdd|FwpmEngineOpen|deny_|read_root|write_root|proxy_)", st))[:14]:
    s = m.start()
    print("  -", re.sub(r"\s+", " ", st[max(0, s - 120):s + 160]))

print("\n== sandbox-runner / setup：codex 来源串 ==")
for name in ["lingxi-sandbox-runner.exe", "lingxi-sandbox-setup.exe"]:
    d = load(os.path.join(BASE, name))
    lines = [t for t in re.findall(r"[\x20-\x7e]{12,120}", d) if "codex" in t.lower()]
    print(f"  {name}: {len(lines)} 条")
    for t in sorted(set(lines))[:8]:
        print("    ", t)

print("\n== kdocs-cli 命令面 ==")
kd = load(os.path.join(os.environ["LOCALAPPDATA"],
                       r"Programs\lingxi-desktop\resources\kdocs-cli\kdocs-cli.exe"))
for k in ["Usage:", "usage:", "action", "resource", "api.wps.cn", "365.kdocs.cn",
          "KDOCS_CLI_APP_ID", "WPS_SID", "kdocs-cli"]:
    print(f"  {k:18} {kd.count(k):4}")
acts = sorted(set(re.findall(r"\b([a-z][a-z0-9]+(?:_[a-z0-9]+){1,3})\b", kd)))
print(f"  snake_case 候选 {len(acts)} 个；样本: {', '.join(acts[:70])}")
usages = [re.sub(r"\s+", " ", t) for t in re.findall(r"[A-Za-z][^\x00]{0,120}Usage:[^\x00]{0,200}", kd)]
print("  Usage 行:", usages[:4])
