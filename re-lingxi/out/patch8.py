p="build_tool_js_map.py"; s=open(p,encoding="utf-8").read()
s=s.replace('def nearest_tool(off, window=6000):','def nearest_tool(off, window=10_000_000):')
s=s.replace('"- 归属规则：片段 offset 向前/后 6 KB 内**最近的工具名**；`?` 表示 6 KB 内没有工具名（游离片段）"',
            '"- 归属规则：逐片段找**最近的工具名**（不设窗口），距离列给出可靠性；距离越大越可疑"')
open(p,"w",encoding="utf-8").write(s)
print("patched")
