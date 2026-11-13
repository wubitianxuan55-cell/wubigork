import re
p="extract_contracts.py"
s=open(p,encoding="utf-8").read()
s=s.replace('''        body = defs.get(f"{sn}_arg") or defs.get(sn) or ""
        # 少数参数名与 define 名不同形，做一次兜底
        if not body and sn.endswith("_id"):
            body = defs.get(f"{sn[:-3]}_id_arg", "")''',
'''        src = f"簇{cno}" if cno else "簇?"
        body = defs.get(f"{sn}_arg") or defs.get(sn) or ""
        if not body and sn.endswith("_id"):
            body = defs.get(f"{sn[:-3]}_id_arg", "")
        if not body:                     # 簇内没有 → 退回全局并标注
            body = GLOBAL.get(f"{sn}_arg") or GLOBAL.get(sn) or ""
            if body:
                src = "全局*"''')
s=s.replace('''hit = miss = 0''','''GLOBAL = {}
for _, _, d in clusters:
    for k, v in d.items():
        if len(v) > len(GLOBAL.get(k, "")):
            GLOBAL[k] = v
hit = miss = 0''')
s=s.replace('''        md.append(f"| `{t}` | `{sn}` | {cno or '—'} | {(body[:130].replace('|','/') or '—')} |")''',
'''        md.append(f"| `{t}` | `{sn}` | {src} | {(body[:130].replace('|','/') or '—')} |")''')
open(p,"w",encoding="utf-8").write(s)
print("patched")
