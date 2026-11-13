import re, os
p = os.path.join(os.environ["APPDATA"], r"WPS 灵犀\app_server_5.24.0\service\lingxi.exe")
d = open(p, "rb").read().decode("utf-8", "replace")
probes = ["起始行", "结束行", "要写入的内容", "要替换的精确文本", "查询关键词", "记忆路径",
          "读取范围", "写入内容", "旧文本", "新文本", "查询词"]
for k in probes:
    hits = [m.start() for m in re.finditer(re.escape(k), d)][:3]
    print(f"{k:12} {len(hits):2} {hits}")
    for h in hits[:1]:
        print("   ", re.sub(r"\s+", " ", d[max(0, h-90):h+90]))
