import io, os

p = r'C:\AI\wubigrok\docs\code-audit-2026-10-02\04-内部领域包与whisper分册.md'
s = io.open(p, encoding='utf-8', newline='').read()
fixes = [
    ('### duplication（复制粘贴与平行实现，19 条）', '### duplication（复制粘贴与平行实现，16 条）'),
    ('### concurrency（并发与锁，1 条）', '### concurrency（并发与锁，2 条）'),
    ('### error-swallow（吞错与静默失败，7 条）', '### error-swallow（吞错与静默失败，6 条）'),
    ('### legacy-residue（兼容壳与历史残留，4 条）', '### legacy-residue（兼容壳与历史残留，5 条）'),
    ('### complexity（复杂度，5 条）', '### complexity（复杂度，4 条）'),
    ('### doc-drift（注释与实现漂移，3 条）', '### doc-drift（注释与实现漂移，2 条）'),
]
n = 0
for a, b in fixes:
    if a in s:
        s = s.replace(a, b, 1)
        n += 1
    else:
        print('MISS:', a)
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('patched', n, 'headings')
