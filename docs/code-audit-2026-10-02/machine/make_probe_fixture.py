import io, os, re

ROOT = r'C:\AI\wubigrok'
SRC = os.path.join(os.environ['APPDATA'], 'gaea', 'config.toml.bak-20261002-audit-pre-RenderToML-fix')
DST = os.path.join(ROOT, '.tmp', 'cfgprobe-synth', 'gaea', 'config.toml')
os.makedirs(os.path.dirname(DST), exist_ok=True)

# 段内改值（不是加键——真实配置里这些键已存在，值是空/零，导致键级保留"压不到"）
WANT = {
    'agent': {'effort': '"high"', 'subagent_effort': '"low"', 'approval_timeout_secs': '42'},
    'tools': {'compact': 'true'},
    'providers': {'thinking': '"adaptive"', 'effort': '"max"'},
    'sandbox': {'paths': '["/x", "/y"]'},
}

lines = io.open(SRC, encoding='utf-8').read().split('\n')
out, cur, changed, seen = [], None, [], set()
for ln in lines:
    m = re.match(r'^\s*\[\[?([A-Za-z_][A-Za-z0-9_.]*)\]\]?\s*$', ln)
    if m:
        cur = m.group(1)
    km = re.match(r'^(\s*)([A-Za-z_][A-Za-z0-9_]*)(\s*)=.*$', ln)
    if km and cur in WANT and km.group(2) in WANT[cur]:
        key = (cur, km.group(2))
        if key not in seen:  # 同段重复键只改第一次（旧文件里可能是重复的）
            seen.add(key)
            new = '%s%s = %s' % (km.group(1), km.group(2), WANT[cur][km.group(2)])
            changed.append('%s.%s: %r -> %s' % (cur, km.group(2), ln.strip()[:40], new.strip()))
            ln = new
    out.append(ln)

io.open(DST, 'w', encoding='utf-8', newline='').write('\n'.join(out))
print('synth:', DST)
print('改值条数:', len(changed))
for c in changed:
    print('  ', c)
for sec, keys in WANT.items():
    for k in keys:
        if (sec, k) not in seen:
            print('  未命中（键在该段不存在）: %s.%s' % (sec, k))
