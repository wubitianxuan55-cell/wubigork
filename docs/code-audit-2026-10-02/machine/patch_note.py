import io

p = r'C:\AI\wubigrok\docs\code-audit-2026-10-02\README.md'
s = io.open(p, encoding='utf-8', newline='').read()
note = ('**单元补审说明（AP7）**：绘梦/图像/视觉单元（AP7）首轮由工作流编排的审计员完成并落盘，'
        '但工作流层回报该 agent 失败；主代理随后**重做了该单元**，`units/AP7.json` 为第二轮结果'
        '（20 条 / 1 条 P0，每行 evidence 已按工作树当前行号逐行回验 0 处不匹配）。'
        '03 分册综合时用的是第一轮内容——两轮**结论一致**（同一条 P0：三链共用包级单槽进度/取消导致并发串台），'
        '仅在个别 P1/P2 条目上互换，故分册内计数与本报告单元级合计可能相差 ±1，'
        '**细节一律以终稿 `units/AP7.json` 与本报告为准**。\n\n---\n\n## 2. 数字总览')
old = '---\n\n## 2. 数字总览'
if old in s:
    s = s.replace(old, note, 1)
    io.open(p, 'w', encoding='utf-8', newline='').write(s)
    print('inserted AP7 note')
else:
    print('MISS anchor')
