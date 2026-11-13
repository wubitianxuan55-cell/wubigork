"""D 批探针：服务目录四件工具链识别 + lingxi.exe 里 deliver_result / tools/*.tmpl 区域定位。"""
import os
import re

BASE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service")
print("== 工具链识别 ==")
for n in ["lingxi-native-host.exe", "lingxi-sandbox-runner.exe", "lingxi-sandbox-setup.exe",
          "lingxi-wpscli-proxy.exe", "lingxi.exe"]:
    p = os.path.join(BASE, n)
    d = open(p, "rb").read()
    go = b"google.golang.org" in d or b"Go build" in d
    rust = b"rustc" in d or b"core::panicking" in d or b"library/core/src" in d
    mod = ""
    m = re.search(rb"path\t([A-Za-z0-9_./\-]{10,140})\n", d)
    if m:
        mod = m.group(1).decode("ascii", "replace")
    print(f"{n:28} {len(d)/1048576:6.1f}MB  MZ={d[:2]==b'MZ'}  go={go}  rust={rust}  module={mod or '-'}")

EXE = os.path.join(BASE, "lingxi.exe")
txt = open(EXE, "rb").read().decode("utf-8", "replace")

print("\n== deliver_result 命中位置 ==")
hits = [m.start() for m in re.finditer("deliver_result", txt)]
print(hits)
for i, s in enumerate(hits):
    seg = txt[max(0, s - 400):s + 700]
    print(f"\n---- hit {i} @ {s} ----")
    print(seg)
