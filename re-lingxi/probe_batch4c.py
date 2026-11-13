"""D 批探针 3：native-host 消息面、sandbox 两件策略串、kdocs-cli 命令面。"""
import os
import re

BASE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service")


def load(p):
    return open(p, "rb").read().decode("utf-8", "replace")


print("== native-host [NH-MSG] 上下文 ==")
nh = load(os.path.join(BASE, "lingxi-native-host.exe"))
for m in list(re.finditer(r"\[NH-MSG\]", nh))[:9]:
    s = m.start()
    seg = re.sub(r"\s+", " ", nh[max(0, s - 160):s + 200])
    print("  -", seg)

print("\n== native-host 关键串 ==")
for k in ["Content-Length", "Attach", "Screenshot", "evaluate", "Target", "browser", "tab",
          "cdp.event", "cdp.request", "native_host", "Authorization", "127.0.0.1"]:
    print(f"  {k:18} {nh.count(k):4}")

print("\n== native-host 符号（含 cdp/nh/mcp 的） ==")
for sym in sorted(set(re.findall(r"[A-Za-z0-9_./\-]{6,120}\.[A-Za-z_][A-Za-z0-9_]{2,60}", nh))):
    if re.search(r"(?i)(cdp|nativehost|NHMsg|browser|websocket)", sym) and "/" in sym:
        print("  ", sym)

for name in ["lingxi-sandbox-runner.exe", "lingxi-sandbox-setup.exe"]:
    print(f"\n== {name} 策略串 ==")
    d = load(os.path.join(BASE, name))
    for k in ["WFP", "FwpmFilterAdd", "FwpmEngineOpen", "AppContainer", "Sid", "CreateRestrictedToken",
              "deny", "Deny", "read_root", "write_root", "proxy", "LingxiOnline", "LingxiOffline",
              "tools", "policy", "sandbox_policy", "codex", "approval"]:
        c = d.count(k)
        if c:
            print(f"  {k:22} {c}")

print("\n== kdocs-cli 命令面线索 ==")
kd = load(os.path.join(os.environ["LOCALAPPDATA"],
                       r"Programs\lingxi-desktop\resources\kdocs-cli\kdocs-cli.exe"))
for k in ["Usage:", "usage:", "action", "resource", "api.wps.cn", "365.kdocs.cn", "--app-id",
          "KDOCS_CLI_APP_ID", "WPS_SID"]:
    print(f"  {k:18} {kd.count(k):4}")
acts = sorted(set(re.findall(r"\b([a-z]+(?:_[a-z]+){1,3})\b", kd)))
print("  snake_case 候选动作数:", len(acts))
print("  样本:", ", ".join(acts[:60]))
