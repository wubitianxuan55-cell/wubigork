p = "extract_i_batch.py"
s = open(p, encoding="utf-8").read()

# 3) 占位符：改成在原始二进制里计数 + 取样
s = s.replace('''usage = defaultdict(list)
for m in re.finditer(r"\\| `(\\d+)` \\| \\w+ \\| `([^`]{0,120})`", stmts):
    head = m.group(2)
    for ph in re.findall(r"__[A-Z_]{3,}__", head):
        usage[ph].append(head)''',
'''usage = defaultdict(list)
for ph in ph_all:
    cnt = wp.count(ph)
    i = wp.find(ph)
    ctx = re.sub(r"\\s+", " ", wp[max(0, i - 60):i + len(ph) + 60]) if i >= 0 else ""
    usage[ph].append(f"×{cnt} {ctx}")''')

# 4) 流式事件：只数带引号的字面量，标注泛词
s = s.replace('''EV = Counter()
for k in ["content_block_delta", "content_block_start", "content_block_stop", "char_delta",
          "line_delta", "patch_delta", "message_start", "message_stop", "message_delta",
          "tool_call_start", "tool_call_delta", "tool_result", "title_updated", "reasoning_delta",
          "thinking_delta", "done", "error", "usage", "ping"]:
    c = lx.count(k)
    if c:
        EV[k] = c''',
'''EV = Counter()
GENERIC = {"error", "ping", "usage", "done"}
for k in ["content_block_delta", "content_block_start", "content_block_stop", "char_delta",
          "line_delta", "patch_delta", "message_start", "message_stop", "message_delta",
          "tool_call_start", "tool_call_delta", "tool_result", "title_updated", "reasoning_delta",
          "thinking_delta", "done", "error", "usage", "ping"]:
    c = lx.count(f'"{k}"')          # 只认带引号的字面量
    if c:
        EV[k] = c''')
s = s.replace('''for k, v in EV.most_common():
    md.append(f"| `{k}` | {v} |")''',
'''for k, v in EV.most_common():
    tag = "（泛词，含非事件用法）" if k in GENERIC else ""
    md.append(f"| `{k}` | {v} | {tag} |")''')
s = s.replace('''"## 事件名频次", "", "| 事件 | 次数 |", "|---|---|"]''',
              '''"## 事件名频次（只数带引号的字面量）", "", "| 事件 | 次数 | 备注 |", "|---|---|---|"]''')
open(p, "w", encoding="utf-8").write(s)
print("patched I")
