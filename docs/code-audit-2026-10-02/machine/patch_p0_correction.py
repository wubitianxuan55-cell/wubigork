import io

p = r'C:\AI\wubigrok\docs\code-audit-2026-10-02\README.md'
s = io.open(p, encoding='utf-8', newline='').read()

old_row = '| 21 | FE7 | `assignment` 数值字段 TS 可选 / Go 零值：一次往返后 0 与缺失不可区分 | `internal/schedule/types.go:107-111` | 改指针或引入三态（缺失/0/值）+ 往返断言 | — |'
new_row = ('| 21 | FE7 | `assignment` 的 `Quantity`/`Amount` 是裸 `float64` + `omitempty`：**显式 0 经一次往返被抹成缺键**'
           '〔复核更正：方向是「0 → 缺失」，不是「缺失 → 0」；`Units` 早已是 `*float64`〕 | `internal/schedule/types.go` | '
           '改 `*float64` 三态 + 绑定层往返断言（本轮已修） | 线5 实测更正 |')

note = ('> **本轮（2026-10-02 第一轮止血刀）已修**：#1/#2（配置保存丢段）、#3（任务收件箱无锁）、#18（GenUI 校验误判）、'
        '#20（多基线被抹）、#21（数值三态，方向已更正）、#22（`ExtractJSON` 多对象）；'
        '逐条改动与红→绿证据见 [round-1-fixes.md](round-1-fixes.md)，25 条在**当前工作树**上的复验结论见 '
        '[round-1-p0-status.md](round-1-p0-status.md)。\n\n')

changed = 0
if old_row in s:
    s = s.replace(old_row, new_row, 1)
    changed += 1
else:
    print('MISS row21')
if '**P0 的性质分布**' in s and '本轮（2026-10-02 第一轮止血刀）已修' not in s:
    s = s.replace('**P0 的性质分布**', note + '**P0 的性质分布**', 1)
    changed += 1
else:
    print('MISS note anchor')

io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('patched', changed)
