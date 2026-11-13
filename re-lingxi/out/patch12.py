import io
p = "docs/gaea-lingxi-re-distill-2026-11.md"
s = io.open(p, encoding="utf-8").read()
old = "> 原始产物（H 批，片段归属尝试，2026-11-13 续）：`WPSCLI-js-statements.md`（213 条真 JS 语句 + 对象模型清点）、`WPSCLI-tool-js-map.md`（邻近法反例）、`WPSCLI-module-map.md`（模块法结果）、`WPSCLI-js-api-inventory.md`。脚本：`rebuild_js.py`、`build_tool_js_map.py`、`build_module_map.py`、`build_js_api.py`。"
new = old + "\n> 原始产物（I 批，骨架/记忆/占位符/流式，2026-11-13 续）：`PROMPT-skeleton.md`（130 节点装配顺序）、`MEMORY-spec.md`（72 KB）、`WPSCLI-placeholders.md`（66 个）、`STREAM-events.md`。脚本：`extract_i_batch.py`。"
assert old in s
io.open(p, "w", encoding="utf-8").write(s.replace(old, new, 1))
print("header updated")
