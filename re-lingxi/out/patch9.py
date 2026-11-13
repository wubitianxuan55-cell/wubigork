import io
p = "docs/gaea-lingxi-re-distill-2026-11.md"
s = io.open(p, encoding="utf-8").read()
old = "> 原始产物（G 批，全量清单，2026-11-13 续）：`re-lingxi/out/GO-types.md`（15 984 类型 / 1 365 包）、`GO-error-codes.md`（425 个码）、`HTTP-routes.md`（129 条）、`PROMPT-tool-contracts.md`（25 工具 / 102 参数 / 86 条中文原文）、`WPSCLI-tool-js.md`。脚本：`extract_all.py`、`extract_contracts.py`。"
new = old + "\n> 原始产物（H 批，片段归属尝试，2026-11-13 续）：`WPSCLI-js-statements.md`（213 条真 JS 语句 + 对象模型清点）、`WPSCLI-tool-js-map.md`（邻近法反例）、`WPSCLI-module-map.md`（模块法结果）、`WPSCLI-js-api-inventory.md`。脚本：`rebuild_js.py`、`build_tool_js_map.py`、`build_module_map.py`、`build_js_api.py`。"
assert old in s
io.open(p, "w", encoding="utf-8").write(s.replace(old, new, 1))
print("header updated")
