import json, re, os, collections

SPILL = r"C:\Users\wubi\AppData\Local\Temp\dsh-spill-oJYhav\session-beea35b6bc23\15f3c7b34fcc-job_output.txt"
t = open(SPILL, encoding='utf-8', errors='replace').read()
d = None
try:
    d = json.loads(t[t.find('{'):], strict=False)
except Exception as e:
    print('json parse failed (spill 被截断): %s' % str(e)[:80])
if d:
    print('units_total', d.get('units_total'), 'ok', d.get('units_ok'), 'failed', d.get('failed'))
    for s in d.get('synth', []):
        print('--- %s' % os.path.basename(s.get('file') or ''))
        print('    findings=%s p0=%s p1=%s cutters=%s' % (s.get('total_findings'), s.get('p0'), s.get('p1'), s.get('cutter_count')))
        for x in (s.get('top10') or [])[:4]:
            print('    %s %s | %s:%s' % (x.get('severity'), x.get('title'), x.get('file'), x.get('line')))
else:
    for m in re.finditer(r'"total_findings":\s*(\d+)[^}]*?"p0":\s*(\d+)[^}]*?"p1":\s*(\d+)[^}]*?"verdict_short":\s*"([^"]{0,80})', t, re.S):
        print('synth:', m.group(1), 'p0', m.group(2), 'p1', m.group(3), '|', m.group(4)[:60])

# 04 分册小节标题里括号标注的条数 vs 实际条目数
p4 = r'C:\AI\wubigrok\docs\code-audit-2026-10-02\04-内部领域包与whisper分册.md'
lines = open(p4, encoding='utf-8', errors='replace').read().split('\n')
cur = None
counts = collections.OrderedDict()
claimed = {}
for ln in lines:
    if ln.startswith('### '):
        cur = ln
        counts[cur] = 0
        m = re.search(r'[（(]\s*(\d+)\s*条', ln)
        claimed[cur] = int(m.group(1)) if m else None
    elif cur and re.match(r'^\*\*[A-Z]+\d+ ', ln):
        counts[cur] += 1
print()
print('=== 04 分册 分类清单 小节计数核验 ===')
for k, v in counts.items():
    if claimed[k] is not None and claimed[k] != v:
        print('MISMATCH 实际=%d 标注=%s | %s' % (v, claimed[k], k))
    else:
        print('ok       实际=%d 标注=%s | %s' % (v, claimed[k], k))
