import re
p = "extract_tool_tmpl.py"
s = open(p, encoding="utf-8").read()
s = s.replace(
 '''DEF = re.compile(r\'\\{\\{\\s*define\\s+"([A-Za-z0-9_]+)"\\s*\\}\\}(.{0,4000}?)\\{\\{\\s*end\\s*\\}\\}\', re.S)''',
 '''# 允许 trim 标记：{{ define "x" -}} … {{- end }}
DEF = re.compile(r\'\\{\\{-?\\s*define\\s+"([A-Za-z0-9_]+)"\\s*-?\\}\\}(.{0,6000}?)\\{\\{-?\\s*end\\s*-?\\}\\}\', re.S)''')
open(p, "w", encoding="utf-8").write(s)
print("regex patched:", "DEF =" in s)
