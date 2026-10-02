import io

p = r'C:\AI\wubigrok\docs\code-audit-2026-10-02\README.md'
s = io.open(p, encoding='utf-8', newline='').read()

edits = []

# 1) 标题：说明 25 行里有重复
a1 = '## 3. 屎山 TOP 榜（25 条 P0 全表 + 复核状态）'
b1 = '## 3. 屎山 TOP 榜（表内 25 行 = **22 条独立 P0** + 3 行重复 + 复核状态）'
edits.append((a1, b1))

# 2) #17 撤销（复核不成立）
a2 = ('| 17 | FE2 | `CostLibraryView` 表格视图「全选」调用**未定义函数**：点击抛未捕获 ReferenceError、无反馈、无测试覆盖 '
      '| `frontend/src/gaea/components/CostLibraryView.tsx:1084-1090` | 补实现或删功能；加「点表头复选框 → 全部行进 selected」用例 | — |')
b2 = ('| 17 | FE2 | ~~表格视图「全选」调用未定义函数~~ **〔复核撤销：判据不成立〕** `setAllSelected`/`clearSelection` 定义在同文件 '
      '`1126-1131`（函数声明提升，点击不会抛 ReferenceError，且该文件自 v4.386.0 未改过）。**真实缺口降级为 P2**：表头全选没有测试覆盖 '
      '| `frontend/src/gaea/components/CostLibraryView.tsx:1126-1131`（定义处） | 补「点表头复选框 → 全部行进 selected」用例即可 | 主代理 + 复核线亲核 |')
edits.append((a2, b2))

# 3) 在「本轮已修」注记后追加复核更正
a3 = ('[round-1-p0-status.md](round-1-p0-status.md)。')
b3 = ('[round-1-p0-status.md](round-1-p0-status.md)。\n>\n'
      '> **复核更正（2026-10-02 复核线 + 主代理亲核）**：① 上表 25 行中 **#19/#20/#21 与 #1/#2/#8 是同一条**（GA5-01/02/03 重复行），'
      '故**独立 P0 实为 22 条**；② **#17 撤销**（判据不成立，见该行）；③ 复核后仍未开放的独立条目集中在：'
      '后台链登记纪律（#4/#5）、检索与知识库吞错/全局索引（#6/#7）、崩溃面（#8）、三套编排与并发闸（#9/#10/#11）、'
      '微信语音并发（#12/#13）、绘梦单槽进度（#14）、三类复发型缺陷无守卫（#23/#24/#25）。')
edits.append((a3, b3))

n = 0
for a, b in edits:
    if a in s:
        s = s.replace(a, b, 1)
        n += 1
    else:
        print('MISS:', a[:70])
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('patched', n, 'of', len(edits))
