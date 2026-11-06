# clean_polluted_memory_20261106.py — 2026-11-06 语音翻车（转写滚雪球+空格撕裂）
# 污染记忆定向清理。默认 DRY-RUN（只列不动）；--apply 才执行（先 .bak 备份）。
#
# 判级依据（docs/gaea-voice-port-distill-20261107.md 污染审计节）：
#   删=蒸馏伪影入册——雪球串被编码成用户风格特质/回声文本被当真/撕裂文本入档/
#      污染串推断出的关系事实（含"明确表达了爱意"=「我在等你爱」回声伪影）
#   留=用户真实挫败信号（好好说话/困惑/说错话）与中性事实——那是诚实记忆
#
# 用法：python -I clean_polluted_memory_20261106.py [--apply] [--db <hermes.db>]
# 前置：建议先关 gaea 壳（脚本检测到运行中会提示但不阻止，WAL 多写者可并发）。
import json
import os
import shutil
import sqlite3
import sys
import datetime

DEFAULT_DB = os.path.join(os.environ.get("APPDATA", ""), "gaea", "whisper_data", "hermes.db")

# 伪影事实 ID（2026-11-07 只读审计定级，见上；全 ID 自库核对）
BAD_FACT_IDS = [
    "8c1293dbade4a1e0",  # gaea回复：空格撕裂文本入档（根因2 伪影）
    "aec297c9174d1b4f",  # 五一路推断（从撕裂诗文本推出，抽取自认弱推断）
    "c9da4e3f53c9ae08",  # 用户风格=无标点连字符短句（雪球串当特质，conf 0.9）
    "a4682e09689d8024",  # 用户喜欢省略标点短句堆叠（同上，conf 0.9）
    "e8a55abe0455aa4d",  # 「我在看我在看你」回声文本当用户耐心（conf 0.8）
    "034778e31d32abce",  # 用户明确表达爱意（「我在等你爱」回声伪影，conf 0.9）⚠高危
    "4a49b04d5239e4f2",  # 持续倾听注视（回声文本当行为，conf 0.8）
]
BAD_EPISODE_IDS = ["6b1372333c143719"]   # 污染叙事情节
DIRTY_ROW_SESSION = "whisper_gaea"
DIRTY_TURN_RANGE = range(123, 129)  # t123-t128 污染轮（t130+ 为用户真实轮/修复后干净轮，留）


def main() -> int:
    apply = "--apply" in sys.argv
    db_path = DEFAULT_DB
    if "--db" in sys.argv:
        db_path = sys.argv[sys.argv.index("--db") + 1]
    print(f"库: {db_path}\n模式: {'APPLY（先备份再删）' if apply else 'DRY-RUN（只列不动）'}")

    if apply:
        stamp = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
        bak = db_path + ".bak-pollution-clean-" + stamp
        shutil.copy2(db_path, bak)
        for suf in ("-wal", "-shm"):
            if os.path.exists(db_path + suf):
                shutil.copy2(db_path + suf, bak + suf)
        print(f"备份: {bak}")

    con = sqlite3.connect(db_path, timeout=5)
    cur = con.cursor()
    total = 0

    idmarks = ",".join("?" * len(BAD_FACT_IDS))
    cur.execute(f"SELECT id, substr(summary,1,60), source_turn_index FROM memory_facts WHERE id IN ({idmarks})", BAD_FACT_IDS)
    found = cur.fetchall()
    print(f"\n[事实] 命中 {len(found)}/{len(BAD_FACT_IDS)}：")
    for fid, s, t in found:
        print(f"  - [{fid[:8]}] turn={t} :: {s}")
    total += len(found)
    epmarks = ",".join("?" * len(BAD_EPISODE_IDS))
    cur.execute(f"SELECT id FROM episodes WHERE id IN ({epmarks})", BAD_EPISODE_IDS)
    eps = cur.fetchall()
    print(f"[情节] 命中 {len(eps)}/{len(BAD_EPISODE_IDS)}")
    total += len(eps)

    cur.execute("SELECT rows_json FROM chat_history WHERE session_id=?", (DIRTY_ROW_SESSION,))
    row = cur.fetchone()
    if row:
        rows = json.loads(row[0])
        dirty = [r for r in rows if r.get("turnIndex") in DIRTY_TURN_RANGE]
        print(f"[chat_history] {DIRTY_ROW_SESSION} 污染轮 {DIRTY_TURN_RANGE.start}-{DIRTY_TURN_RANGE.stop-1}: "
              f"命中 {len(dirty)}/{len(rows)} 行 turnIndex={[r.get('turnIndex') for r in dirty]}")
        total += 1  # 整段 rows_json 重写计 1
    else:
        print("[chat_history] 会话不存在")
        dirty = []

    if not apply:
        print(f"\nDRY-RUN 结束：将删 {total} 项。确认无误后加 --apply 执行。")
        return 0

    cur.execute(f"DELETE FROM memory_facts WHERE id IN ({idmarks})", BAD_FACT_IDS)
    print(f"已删事实 {cur.rowcount}")
    cur.execute(f"DELETE FROM episodes WHERE id IN ({epmarks})", BAD_EPISODE_IDS)
    print(f"已删情节 {cur.rowcount}")
    if row and dirty:
        kept = [r for r in rows if r.get("turnIndex") not in DIRTY_TURN_RANGE]
        cur.execute("UPDATE chat_history SET rows_json=? WHERE session_id=?",
                    (json.dumps(kept, ensure_ascii=False), DIRTY_ROW_SESSION))
        print(f"chat_history 已重写：{len(rows)}→{len(kept)} 行")
    con.commit()
    con.close()
    print("完成。人格状态机（companion_state）的提示历史窗口 ≤4 行会随新干净轮自然滚出，不做手术。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
