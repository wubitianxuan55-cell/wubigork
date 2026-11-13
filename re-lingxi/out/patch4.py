import re
# 1) extract_all.py 不再写 contracts/routes（交给 extract_contracts.py）
p="extract_all.py"; s=open(p,encoding="utf-8").read()
s=s.replace('''# ---------------------------------------------------------------- 3) HTTP 路由''','''# ---------------------------------------------------------------- (3/4 交由 extract_contracts.py 生成)
if False:
    pass
# ---------------------------------------------------------------- 3) HTTP 路由''',1)
# 直接把 3)/4) 两段整体注释掉：用标记切分
start=s.find("# ---------------------------------------------------------------- (3/4 交由")
end=s.find("# ---------------------------------------------------------------- 5) wpscli JS 分组")
assert start>0 and end>start
s=s[:start]+"# (3) 路由 与 (4) 契约 由 extract_contracts.py 生成，此处跳过\n\n"+s[end:]
open(p,"w",encoding="utf-8").write(s)

# 2) snake 修正（处理 ID/URL 缩写）
q="extract_contracts.py"; t=open(q,encoding="utf-8").read()
t=t.replace('''def snake(s):
    s = re.sub(r"([A-Z]+)([A-Z][a-z])", r"\\1_\\2", s)
    s = re.sub(r"([a-z0-9])([A-Z])", r"\\1_\\2", s)
    return s.lower()''',
'''def snake(s):
    parts = re.findall(r"[A-Z]+(?![a-z])|[A-Z][a-z]+|[a-z0-9]+", s)
    out = "_".join(p.lower() for p in parts)
    for bad, good in (("task_i_d", "task_id"), ("todo_i_d", "todo_id"), ("i_d", "id"),
                      ("input_u_r_l", "input_url"), ("max_tokens", "max_tokens")):
        out = out.replace(bad, good)
    return out''')
open(q,"w",encoding="utf-8").write(t)
print("patched")
