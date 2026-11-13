"""D 批探针 2：找 tools/*.tmpl 正文、kdocs-cli 工具链、native-host 轮廓。"""
import os
import re

BASE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service")
EXE = os.path.join(BASE, "lingxi.exe")
txt = open(EXE, "rb").read().decode("utf-8", "replace")

print("== 找 tools/*.tmpl 正文的锚点 ==")
for k in ["c_deliver_result", "c_text", "c_tool_result", "交付结果", "交付物", "ref=", "deliver",
          "share_file", "c_speech_to_text", "generate_image"]:
    print(f"  {k:20} {txt.count(k):5}")

for k in ["c_deliver_result", "交付结果文件", "将文件交付"]:
    for m in list(re.finditer(re.escape(k), txt))[:2]:
        s = m.start()
        print(f"\n---- {k} @ {s} ----")
        print(txt[max(0, s - 500):s + 800])

print("\n== kdocs-cli 轮廓 ==")
kd = os.path.join(os.environ["LOCALAPPDATA"], r"Programs\lingxi-desktop\resources\kdocs-cli\kdocs-cli.exe")
d = open(kd, "rb").read()
go = b"google.golang.org" in d or b"Go build" in d
rust = b"core::panicking" in d or b"library/core/src" in d
mm = re.search(rb"path\t([A-Za-z0-9_./\-]{10,140})\n", d)
print(f"kdocs-cli.exe {len(d)/1048576:.1f}MB go={go} rust={rust} module={(mm.group(1).decode() if mm else '-')}")

print("\n== native-host 轮廓 ==")
nh = os.path.join(BASE, "lingxi-native-host.exe")
n = open(nh, "rb").read().decode("utf-8", "replace")
for k in ["[NH-MSG]", "cdp", "CDP", "Target.attachToTarget", "Runtime.evaluate", "Page.navigate",
          "webSocketDebuggerUrl", "\\\\.\\pipe", "mcp", "Content-Length", "jsonrpc"]:
    print(f"  {k:26} {n.count(k)}")
