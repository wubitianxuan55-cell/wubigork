# false_interrupt_bound.py — 误打断率上界测量（§13.3 记分卡第 5 项工具）。
#
# 方法（如实口径）：误打断率 ≤ 取消响应占比 —— 把每一次 response.cancelled 都
# 归因为打断（无论用户有意插话还是噪声/回声误触发），得到保守上界；上界 ≤3%
# 即门绿，上界 >3% 仅说明「需要区分有意/无意」，不构成红判。
# 数据源：sidecar 日志（%TEMP%/gaea-voice-v2/python.exe.log，跨重启追加）。
# 注意：自动化测试流量（循环 wav=故意 bargein 机器）会污染占比——按日核对时
# 排除已知测试窗口。
import re
import sys
from collections import defaultdict
from pathlib import Path


def main() -> int:
    log = Path(sys.argv[1]) if len(sys.argv) > 1 else None
    if log is None:
        import os
        log = Path(os.environ.get("TEMP", ".")) / "gaea-voice-v2" / "python.exe.log"
    days: dict[str, dict[str, int]] = defaultdict(lambda: defaultdict(int))
    day_pat = re.compile(r"^(\d{4}-\d{2}-\d{2})")
    for line in open(log, encoding="utf-8", errors="replace"):
        m = day_pat.match(line)
        if not m:
            continue
        d = days[m.group(1)]
        if "Response done (status=" in line:
            for st in ("completed", "cancelled", "failed"):
                if f"status={st}" in line:
                    d[st] += 1
        if "speech during pending response: cancelled" in line:
            d["bargein"] += 1
    print(f"{'date':<12}{'done':>6}{'cxl':>6}{'fail':>6}{'bgin':>6}{'cancel-ratio(UB)':>18}")
    for day in sorted(days):
        d = days[day]
        total = d["completed"] + d["cancelled"]
        ub = d["cancelled"] / total * 100 if total else 0.0
        print(f"{day:<12}{d['completed']:>6}{d['cancelled']:>6}{d['failed']:>6}{d['bargein']:>6}{ub:>17.1f}%")
    print("\n口径：cancel-ratio=误打断率上界（全部取消归因打断）；测试流量日需人工剔除。")
    return 0

if __name__ == "__main__":
    import os
    sys.exit(main())
