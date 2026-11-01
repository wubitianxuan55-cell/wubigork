# eot_analyze.py — M3 评测汇总：服务端日志按连接切片 → Smart Turn 逐边界判决表 + 指标。
#
# 切片依据：每个 corpus 案例一条 WS 连接，日志按「Client connected ... (session X)」
# 到下一次「Client connected」为一片。判决行形如：
#   Smart Turn: complete (p=0.855, 30.7ms); using 800ms speculative reopen grace
# 轮提交依据：Turn turn_N rev=R latency: ... smart_turn_status=... status=...
# 用法：python -I eot_analyze.py <server.log> <corpus_built.json> [firered_scores.json]
import json
import re
import sys
from collections import defaultdict
from pathlib import Path

ST_RE = re.compile(r'Smart Turn: (complete|incomplete) \(p=([\d.]+), ([\d.]+)ms\)')
TURN_RE = re.compile(r'Turn (turn_\d+) rev=(\d+) latency: .*?smart_turn_status=(\w+) status=(\w+)')
SOFT_RE = re.compile(r'Speech soft-ended \(segment=(\d+)ms, active=(\d+)ms, turn=(\w+) rev=(\d+)\)')


def main() -> None:
    log_path, built_path = sys.argv[1], sys.argv[2]
    firered_path = sys.argv[3] if len(sys.argv) > 3 else None
    built = json.loads(Path(built_path).read_text(encoding='utf-8'))

    # 按连接切片
    chunks = []
    cur = None
    for line in Path(log_path).read_text(encoding='utf-8', errors='replace').splitlines():
        if 'Client connected to pipeline' in line:
            if cur is not None:
                chunks.append(cur)
            cur = {'lines': [], 'session': line.split('session ')[-1].split(')')[0]}
        if cur is not None:
            cur['lines'].append(line)
    if cur is not None:
        chunks.append(cur)

    # 每片的 smart turn 判决 / 软结束 / 轮提交
    per_conn = []
    for ch in chunks:
        sts = [(m.group(1), float(m.group(2)), float(m.group(3))) for l in ch['lines'] if (m := ST_RE.search(l))]
        turns = [(m.group(1), int(m.group(2)), m.group(3), m.group(4)) for l in ch['lines'] if (m := TURN_RE.search(l))]
        softs = len([l for l in ch['lines'] if 'Speech soft-ended' in l])
        per_conn.append({'session': ch['session'], 'smart_turns': sts, 'turns': turns, 'soft_ends': softs})

    # 只取尾部与 corpus 案例数等长的连接段（本日志含此前探针/回放连接）
    n = len(built['cases'])
    tail = per_conn[-n:]
    print(f'日志连接片 {len(per_conn)} 条，取尾部 {len(tail)} 条对齐语料\n')

    firered = {}
    if firered_path:
        for r in json.loads(Path(firered_path).read_text(encoding='utf-8')):
            firered[(r['case'], r['boundary'])] = r

    # 汇总表：案例 × 边界 → smart turn 判决 vs 期望 vs FireRed 文本判决
    st_ok = st_total = 0
    infer_ms = []
    rows = []
    for case, conn in zip(built['cases'], tail):
        sts = conn['smart_turns']
        # 末边界（tail silence 触发）通常也有一次判决；内部边界数 = len(boundaries)-1
        for i, b in enumerate(case['boundaries']):
            st = sts[i] if i < len(sts) else None
            fr = firered.get((case['name'], b['after_segment']))
            label, p, ms = (st if st else ('-', -1, -1))
            expected = b['expected']
            ok = label == expected
            if not case['observe']:
                st_ok += ok
                st_total += 1
            if ms > 0:
                infer_ms.append(ms)
            rows.append({
                'case': case['name'], 'boundary': i, 'expected': expected,
                'smart_turn': label, 'p': p, 'ms': ms, 'ok': ok,
                'fire_p': fr['eou_prob'] if fr else None,
                'fire_05': fr['p05_decision'] if fr else None,
                'observe': case['observe'],
            })
        commits = ','.join(t[3] for t in conn['turns'])
        revs = ','.join(str(t[1]) for t in conn['turns'])
        rows.append({'case': case['name'],
                     'meta': f"soft_ends={conn['soft_ends']} commits=[{commits}] revs=[{revs}]"})

    for r in rows:
        if 'meta' in r:
            print(f"  {r['case']:22s} {r['meta']}")
        else:
            mark = '-' if r['observe'] else ('OK ' if r['ok'] else 'MISS')
            fr = f" fr={r['fire_05']}(p={r['fire_p']:.2f})" if r['fire_p'] is not None else ''
            print(f"  {r['case']:22s} b{r['boundary']} exp={r['expected']:10s} st={r['smart_turn']:10s} p={r['p']:.3f} {r['ms']:.0f}ms {mark}{fr}")

    if st_total:
        print(f"\nSmart Turn 边界准确率（剔除观察项）：{st_ok}/{st_total} = {st_ok/st_total:.0%}")
    if infer_ms:
        infer_ms.sort()
        p50 = infer_ms[len(infer_ms)//2]
        p95 = infer_ms[min(len(infer_ms)-1, int(len(infer_ms)*0.95))]
        print(f"Smart Turn 推理：p50={p50:.1f}ms p95={p95:.1f}ms max={infer_ms[-1]:.1f}ms（门禁 p95≤80ms）")


if __name__ == '__main__':
    main()
