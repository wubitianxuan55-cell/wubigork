# firered_eot.py — FireRedChat 中文 EOT 离线评分（M3 A/B 对照）。
#
# 权重：HuggingFace FireRedTeam/FireRedChat-turn-detector（chinese_best_model_q8.onnx，
# Apache-2.0），推理口径复刻插件 base.py：拼当前轮用户文本→去标点→tokenizer
# （128 左截断）→ONNX→softmax 末类=eou 概率。逐边界输入=该边界前的段拼接。
#
# 用法：python scripts/voice_v2/firered_eot.py <corpus_manifest.json> <firered_dir> <out.json>
import json
import re
import sys
import time
from pathlib import Path

import numpy as np
import onnxruntime as ort
from transformers import AutoTokenizer


def softmax(x):
    e = np.exp(x - np.max(x))
    return e / e.sum()


def main() -> None:
    manifest_path, model_dir, out_path = sys.argv[1], Path(sys.argv[2]), sys.argv[3]
    cases = json.loads(Path(manifest_path).read_text(encoding='utf-8'))

    tok = AutoTokenizer.from_pretrained(model_dir / 'tokenizer', local_files_only=True, truncation_side='left')
    sess = ort.InferenceSession(str(model_dir / 'chinese_best_model_q8.onnx'), providers=['CPUExecutionProvider'])

    results = []
    for case in cases:
        segs = case['segments']
        text_so_far = ''
        for i, seg in enumerate(segs):
            text_so_far += seg['text']
            last = i == len(segs) - 1
            pause = seg.get('pause_ms_after', 0) if not last else 0
            expected = 'complete' if (last or pause >= 2000) else 'incomplete'
            clean = re.sub('[，。？！,. ?!]', '', text_so_far)
            inputs = tok(
                clean,
                truncation=True,
                padding='max_length',
                add_special_tokens=False,
                return_tensors='np',
                max_length=128,
            )
            t0 = time.perf_counter()
            outputs = sess.run(
                None,
                {
                    'input_ids': inputs['input_ids'].astype('int64'),
                    'attention_mask': inputs['attention_mask'].astype('int64'),
                },
            )
            ms = (time.perf_counter() - t0) * 1000
            eou = float(softmax(outputs[0]).flatten()[-1])
            results.append({
                'case': case['name'],
                'boundary': i,
                'expected': expected,
                'text': clean,
                'eou_prob': round(eou, 4),
                'p05_decision': 'complete' if eou >= 0.5 else 'incomplete',
                'unlikely_008': eou < 0.08,
                'infer_ms': round(ms, 1),
            })

    Path(out_path).write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding='utf-8')

    # 汇总：0.5 阈值与 0.08 unlikely 档的边界准确率
    judged = [r for r in results if r['case'] not in ('trailing_incomplete', 'backchannel_um')]
    for thr_label, pred in (('p>=0.5', lambda r: r['p05_decision']), ('p>=0.08', lambda r: 'complete' if not r['unlikely_008'] else 'incomplete')):
        ok = sum(1 for r in judged if pred(r) == r['expected'])
        print(f'{thr_label}: {ok}/{len(judged)} = {ok / len(judged):.0%}')
    ms = [r['infer_ms'] for r in results]
    print(f'infer ms: p50={sorted(ms)[len(ms)//2]:.1f} max={max(ms):.1f} (n={len(ms)})')


if __name__ == '__main__':
    main()
