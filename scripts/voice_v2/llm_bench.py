# llm_bench.py — LLM 裸速基准（绕过语音链直打端点）：TTFT + 总耗时。
# 用法：python llm_bench.py（目标清单在 TARGETS，按需增删）
# 目标：herdsman 35B (8080) vs 4B (8123) vs 自管 35B (8124)；prompt：tiny(~50tok) vs voice-sized(~1100tok)。
#
# 2026-11-05 实测结论（35B MTP-r2 同 gguf 三种服务形态）：
#   herdsman 全链   TTFT 3.32s / total 4.61s   ← 管理层 +2.1s 固定
#   后端直连(19173) TTFT 1.33s / total 2.45s   ← herdsman 拉起旗标的固定成本
#   自管 llama(8124, b11980+cache-reuse+ctx8192) TTFT 0.08s / total 1.22s
#   → 35B 模型本身不慢；慢在 herdsman 服务栈。语音 e2e 投影 ≈1.9s（与 4B 同档）。
import json
import time
import urllib.request

TARGETS = [
    ("35B-herdsman", "http://localhost:8080/v1/chat/completions",
     "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2"),
    ("4B-voice", "http://127.0.0.1:8123/v1/chat/completions", "qwen3-4b"),
]

SYSTEM_SMALL = "你是语音助手，用一句话回答。"
SYSTEM_VOICE = ("你是 gaea 的语音对话助手。" * 200)  # ~5K chars ≈ 语音 system 体量
USER_SMALL = "数到三"
USER_VOICE = ("【本轮上下文】\n情绪与状态块……\n" * 40) + "今晚我想听你讲一个很短的故事"


def bench(name, url, model, system, user, n=3):
    ttfts, totals = [], []
    body = json.dumps({
        "model": model, "stream": True, "max_tokens": 64,
        "messages": [{"role": "system", "content": system}, {"role": "user", "content": user}],
    }).encode()
    for _ in range(n):
        t0 = time.perf_counter()
        req = urllib.request.Request(url, data=body, headers={"Content-Type": "application/json"})
        ttft = None
        with urllib.request.urlopen(req, timeout=120) as resp:
            for raw in resp:
                line = raw.decode("utf-8", "replace").strip()
                if not line.startswith("data:"):
                    continue
                payload = line[5:].strip()
                if payload == "[DONE]":
                    break
                try:
                    j = json.loads(payload)
                except ValueError:
                    continue
                delta = (j.get("choices") or [{}])[0].get("delta", {})
                if delta.get("content") and ttft is None:
                    ttft = time.perf_counter() - t0
        total = time.perf_counter() - t0
        ttfts.append(ttft if ttft is not None else total)
        totals.append(total)
    ttfts.sort()
    totals.sort()
    print(f"  {name:<14} TTFT中位={ttfts[len(ttfts)//2]:.2f}s  总耗时中位={totals[len(totals)//2]:.2f}s (n={n})")


for label, url, model in TARGETS:
    print(f"== {label} ({url}) ==")
    bench("tiny", url, model, SYSTEM_SMALL, USER_SMALL)
    bench("voice-sized", url, model, SYSTEM_VOICE, USER_VOICE)
