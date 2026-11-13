p="extract_i_batch.py"; s=open(p,encoding="utf-8").read()
s=s.replace('''      "| 占位符 | 推断类型 | 出现片段数 | 样例 |", "|---|---|---|---|"]
for ph in ph_all:
    u = usage.get(ph, [])
    md.append(f"| `{ph}` | {kind(ph)} | {len(u)} | {(u[0][:70].replace('|','/') if u else '—')} |")''',
'''      "| 占位符 | 推断类型 | 二进制内出现次数 | 上下文样例 |", "|---|---|---|---|"]
for ph in ph_all:
    u = usage.get(ph) or [""]
    m = re.match(r"×(\\d+)\\s*(.*)", u[0])
    cnt = m.group(1) if m else "0"
    ctx = (m.group(2) if m else "")[:70].replace("|", "/")
    md.append(f"| `{ph}` | {kind(ph)} | {cnt} | `{ctx or '—'}` |")''')
open(p,"w",encoding="utf-8").write(s); print("patched")
