# stt_compare.py — M4 STT 对照：sherpa streaming zipformer zh-14m vs paraformer-zh（离线同语料）。
#
# sherpa 走流式识别器（逐 16k 帧喂，模拟实时），paraformer 结果直接取自
# corpus_run.json（管线实录转写）。WER 用字级编辑距离（中文，去标点同口径）。
# 用法：python -I stt_compare.py <corpus_dir> <corpus_run.json> <sherpa_model_dir>
import json
import sys
import time
from pathlib import Path

import sherpa_onnx
import soundfile as sf


def wer(ref: str, hyp: str) -> float:
    r, h = list(ref), list(hyp)
    if not r:
        return 0.0 if not h else 1.0
    d = list(range(len(h) + 1))
    for i in range(1, len(r) + 1):
        prev, d[0] = d[0], i
        for j in range(1, len(h) + 1):
            cur = d[j]
            d[j] = min(d[j] + 1, d[j - 1] + 1, prev + (r[i - 1] != h[j - 1]))
            prev = cur
    return d[len(h)] / len(r)


def main() -> None:
    corpus_dir, run_path, model_dir = Path(sys.argv[1]), Path(sys.argv[2]), Path(sys.argv[3])
    built = json.loads((corpus_dir / 'corpus_built.json').read_text(encoding='utf-8'))
    runs = {r['name']: r for r in json.loads(run_path.read_text(encoding='utf-8'))}

    recognizer = sherpa_onnx.OnlineRecognizer.from_transducer(
        tokens=str(model_dir / 'tokens.txt'),
        encoder=str(model_dir / 'encoder-epoch-99-avg-1.onnx'),
        decoder=str(model_dir / 'decoder-epoch-99-avg-1.onnx'),
        joiner=str(model_dir / 'joiner-epoch-99-avg-1.onnx'),
        num_threads=2,
        sample_rate=16000,
        feature_dim=80,
        decoding_method='greedy_search',
        provider='cpu',
    )

    # 期望全轮文本（与 corpus 构建同口径：去标点拼接）
    import re
    def norm(s):
        return re.sub(r'[，。？！,. ?!、；：…]', '', s)

    print(f"{'case':22s} {'sherpa_wer':>9s} {'para_wer':>9s} {'sherpa_rtf':>10s}")
    sher_wers, para_wers, rtfs = [], [], []
    for case in built['cases']:
        # 期望文本：整案例各段拼一口径（轮级对齐取 run 的转写拼接）
        expected = norm(''.join(json.loads((corpus_dir.parent / 'corpus_manifest.json').read_text(encoding='utf-8'))[
            [c['name'] for c in json.loads((corpus_dir.parent / 'corpus_manifest.json').read_text(encoding='utf-8'))].index(case['name'])
        ]['segments'][i]['text'] for i in range(len(case['boundaries']))))
        pcm, sr = sf.read(str(corpus_dir / case['wav']), dtype='float32', always_2d=False)
        assert sr == 16000
        stream = recognizer.create_stream()
        t0 = time.perf_counter()
        chunk = 1600  # 100ms
        for off in range(0, len(pcm), chunk):
            part = pcm[off:off + chunk]
            stream.accept_waveform(sr, part)
        stream.input_finished()
        while recognizer.is_ready(stream):
            recognizer.decode_stream(stream)
        text = recognizer.get_result(stream)
        dur = time.perf_counter() - t0
        audio_dur = len(pcm) / sr
        rtf = dur / audio_dur

        run = runs.get(case['name'])
        para_text = norm(''.join(run['transcripts'])) if run else ''
        sw = wer(expected, norm(text))
        pw = wer(expected, para_text)
        sher_wers.append(sw)
        para_wers.append(pw)
        rtfs.append(rtf)
        print(f"{case['name']:22s} {sw:9.1%} {pw:9.1%} {rtf:10.2f}")

    n = len(sher_wers)
    print(f"\n均值 WER：sherpa={sum(sher_wers)/n:.1%} paraformer={sum(para_wers)/n:.1%}")
    print(f"sherpa 流式 RTF：均值 {sum(rtfs)/n:.2f} 最大 {max(rtfs):.2f}（含模型首帧）")


if __name__ == '__main__':
    main()
