"""从 lingxi.exe 抽取 Go text/template 提示词模板区（templates/local/*.tmpl）。
   模板以明文内嵌，模板名清单见 templates/local/ 与 templates/tools/。
用法: python extract_tmpl.py dump|split [outdir]
"""
import os
import re
import sys

EXE = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service\lingxi.exe")
raw = open(EXE, "rb").read().decode("latin-1")

ANCHORS = ["<im_scene>", "<agent_loop>", "<tool_use>", "<tool_call_style>", "<format>",
           "<skills", "<skill_", "templates/local/agents_main_v2.tmpl", "<memory_system>",
           "<persona", "<project", "<deliver", "deliver_result", "<connector"]
print("== 锚点位置 ==")
for a in ANCHORS:
    hits = [m.start() for m in re.finditer(re.escape(a), raw)][:5]
    print(f"{a:42} {hits}")

if len(sys.argv) > 1 and sys.argv[1] == "dump":
    start = raw.find("<im_scene>")
    lo = max(0, start - 60000)
    seg = raw[lo:start + 90000]
    out = sys.argv[2] if len(sys.argv) > 2 else "tmpl-dump.txt"
    open(out, "w", encoding="utf-8", errors="replace").write(seg)
    print(f"\n窗口写入 {out}: {len(seg)} 字符，起点 {lo}")
