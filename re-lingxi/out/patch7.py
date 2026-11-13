import io
p = "docs/gaea-lingxi-re-distill-2026-11.md"
s = io.open(p, encoding="utf-8").read()
old = "> 原始产物（F 批，线协议与 IPC，2026-11-13 续）：`re-lingxi/out/WIRE-json-tags.md`（2303 个字段名）、`re-lingxi/out/IPC-channels.md`（510 个通道）。脚本：`extract_wire.py`、`extract_ipc2.py`。"
new = old + "\n> 原始产物（G 批，全量清单，2026-11-13 续）：`re-lingxi/out/GO-types.md`（15 984 类型 / 1 365 包）、`GO-error-codes.md`（425 个码）、`HTTP-routes.md`（129 条）、`PROMPT-tool-contracts.md`（25 工具 / 102 参数 / 86 条中文原文）、`WPSCLI-tool-js.md`。脚本：`extract_all.py`、`extract_contracts.py`。"
assert old in s
s = s.replace(old, new, 1)
io.open(p, "w", encoding="utf-8").write(s)
print("header updated")
