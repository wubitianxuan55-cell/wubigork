import io

p = r'C:\AI\wubigrok\docs\code-audit-2026-10-02\README.md'
s = io.open(p, encoding='utf-8', newline='').read()
pairs = [
    ('| 办公文档、造价、进度/DAG/任务、微信语音、绘梦、绑定面、记忆中枢 | 130 | 9 | 19 |',
     '| 办公文档、造价、进度/DAG/任务、微信语音、绘梦、绑定面、记忆中枢 | 131 | 9 | 19 |'),
    ('本报告 511 条**没有一条**是 lint 能抓的', '本报告 512 条**没有一条**是 lint 能抓的'),
]
for a, b in pairs:
    if a in s:
        s = s.replace(a, b, 1)
        print('ok:', a[:40])
    else:
        print('MISS:', a[:60])
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('done')
