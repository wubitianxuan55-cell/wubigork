# playback_tap_analyze.py — 播放面诊断声纹解析（2026-11-07 重叠线复开）。
#
# 输入：diag/voice-playback-tap-<file>.pcm（设备率 Int16 LE，服务端 /v1/diag/playback
# 落盘，rate 在伴生 .meta.json）——worklet process() 实际外放样本（含欠载补零）。
# 用法：python -I playback_tap_analyze.py <tap.pcm> [--rate 48000] [--wav out.wav] [--csv out.csv]
#
# 分窗（0.5s）指标与异常判据（花栗鼠=时间轴压缩，重叠=能量/频谱异常堆积）：
#   rms        窗内均方根（有声/静音）
#   peak       窗内峰值
#   zcr        过零率（音调代理：加速→基频抬升→zcr 系统性上移）
#   jump       相邻样本差 |Δ|>0.45 满幅 的样本对数（掉样本/逐样本顶替的
#              不连续特征；正常语音起音也会有少量，看**密度**与**聚集**）
# 标记：
#   破碎       jump 密度 ≥ 8/窗 且 rms 有声（顶替/杂音高发形态）
#   加速嫌疑   zcr ≥ 全有声窗中位数 × 1.5 且 rms 有声（音调漂移）
#   静音洞     有声→静音→有声 的短静音窗（跳段/欠载嫌疑）
#   异常能量   rms ≥ 全有声窗中位数 × 2（重叠堆积嫌疑，须人耳复听确认）
# 汇总末尾给「重点复听区间」（异常窗 ±2 窗合并），WAV 导出供人耳校验。
import argparse
import array
import json
import math
import statistics
import struct
import sys
import wave
from pathlib import Path

WINDOW_SEC = 0.5
JUMP_THRESH = 0.45          # |Δ|/32768 阈值（满幅比例）
BROKEN_JUMP_MIN = 8         # 破碎窗：jump 计数下限
ZCR_BOOST = 1.5             # 加速嫌疑：zcr 相对中位数倍数
RMS_BOOST = 2.0             # 能量异常：rms 相对中位数倍数
SPEECH_RMS_MIN = 0.01       # 有声判定


def load_pcm(path: Path, rate: int | None) -> tuple[array.array, int]:
    if rate is None:
        meta = path.with_suffix('.meta.json')
        if meta.exists():
            rate = int(json.loads(meta.read_text(encoding='utf-8'))['rate'])
        else:
            sys.exit(f'--rate 未给且无 sidecar: {meta}')
    raw = path.read_bytes()
    if len(raw) % 2:
        raw = raw[:-1]
    samples = array.array('h')
    samples.frombytes(raw)
    return samples, rate


def analyze(samples: array.array, rate: int) -> list[dict]:
    win = int(rate * WINDOW_SEC)
    rows = []
    for w in range(0, len(samples) - 1, win):
        seg = samples[w:w + win]
        n = len(seg)
        sq = 0.0
        peak = 0
        zc = 0
        jumps = 0
        prev = seg[0]
        for s in seg:
            sq += s * s
            if abs(s) > peak:
                peak = abs(s)
            if (s >= 0) != (prev >= 0):
                zc += 1
            if abs(s - prev) > JUMP_THRESH * 32768:
                jumps += 1
            prev = s
        rows.append({
            'idx': len(rows),
            't0': w / rate,
            't1': (w + n) / rate,
            'rms': math.sqrt(sq / n) / 32768,
            'peak': peak / 32768,
            'zcr': zc / n,
            'jump': jumps,
            'speech': False, 'broken': False, 'fast': False, 'loud': False, 'hole': False,
        })
    speech_rows = [r for r in rows if r['rms'] >= SPEECH_RMS_MIN]
    for r in rows:
        r['speech'] = r['rms'] >= SPEECH_RMS_MIN
    if speech_rows:
        rms_med = statistics.median(r['rms'] for r in speech_rows)
        # zcr 基线取前 20% 有声窗（「到了后面才加速」的用户场景=漂移段占比大，
        # 全文件中位数会被自抬漏判；用户症状的锚=开头正常段）
        base_n = max(1, len(speech_rows) // 5)
        zcr_base = statistics.median(r['zcr'] for r in speech_rows[:base_n])
        for r in rows:
            if r['speech']:
                r['broken'] = r['jump'] >= BROKEN_JUMP_MIN
                r['fast'] = r['zcr'] >= zcr_base * ZCR_BOOST
                r['loud'] = r['rms'] >= rms_med * RMS_BOOST
    # 静音洞：夹在有声窗之间的静音窗
    for i in range(1, len(rows) - 1):
        if not rows[i]['speech'] and rows[i - 1]['speech'] and rows[i + 1]['speech']:
            rows[i]['hole'] = True
    return rows


def merge_ranges(flags: list[int], pad: int = 2) -> list[tuple[int, int]]:
    if not flags:
        return []
    out = []
    s = p = flags[0]
    for f in flags[1:]:
        if f - p <= pad + 1:
            p = f
        else:
            out.append((s, p))
            s = p = f
    out.append((s, p))
    return out


def export_wav(samples: array.array, rate: int, out: Path) -> None:
    with wave.open(str(out), 'wb') as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)
        wf.setframerate(rate)
        wf.writeframes(samples.tobytes())


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument('pcm')
    ap.add_argument('--rate', type=int, default=None)
    ap.add_argument('--wav', default=None)
    ap.add_argument('--csv', default=None)
    args = ap.parse_args()

    path = Path(args.pcm)
    samples, rate = load_pcm(path, args.rate)
    dur = len(samples) / rate
    rows = analyze(samples, rate)

    print(f'文件: {path.name}  rate={rate}  时长={dur:.1f}s  窗数={len(rows)}')
    print(f'{"窗":>4} {"时间":>12} {"rms":>7} {"peak":>6} {"zcr":>6} {"jump":>5}  标记')
    for r in rows:
        tags = []
        if r['broken']: tags.append('破碎')
        if r['fast']: tags.append('加速嫌疑')
        if r['hole']: tags.append('静音洞')
        if r['loud']: tags.append('异常能量')
        if not r['speech']: tags.append('静')
        if r['broken'] or r['fast'] or r['loud'] or r['hole']:
            print(f"{r['idx']:>4} {r['t0']:>5.1f}-{r['t1']:<5.1f} {r['rms']:>7.4f} {r['peak']:>6.3f} {r['zcr']:>6.3f} {r['jump']:>5}  {' '.join(tags)}")
    anomalies = sorted({r['idx'] for r in rows if r['broken'] or r['fast'] or r['loud'] or r['hole']})
    print(f'\n异常窗 {len(anomalies)}/{len(rows)}')
    if anomalies:
        print('重点复听区间（异常窗±2 合并）:')
        for s, e in merge_ranges(anomalies):
            print(f'  {rows[s]["t0"]:6.1f}s - {rows[e]["t1"]:6.1f}s')
    else:
        print('全程无异常窗——污染不在已落盘的播放输出侧（查声学环节/设备）')

    if args.wav:
        export_wav(samples, rate, Path(args.wav))
        print(f'\nWAV 导出: {args.wav}（人耳复听用）')
    if args.csv:
        import csv
        with open(args.csv, 'w', newline='', encoding='utf-8') as f:
            wr = csv.DictWriter(f, fieldnames=list(rows[0].keys()))
            wr.writeheader()
            wr.writerows(rows)
        print(f'CSV 导出: {args.csv}')


if __name__ == '__main__':
    main()
