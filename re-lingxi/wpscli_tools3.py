"""从 wpscli.exe 静态抽取 MCP 工具面（[描述][工具名] 成对）。
   动态 tools/list 被 parent_verify 拦住（非金山父进程直接退出），只能静态还原。
用法: python wpscli_tools3.py [out.md]
"""
import re
import sys

EXE = r"C:\Users\wubi\AppData\Local\Programs\lingxi-desktop\resources\wps-cli\wpscli.exe"
raw = open(EXE, "rb").read().decode("latin-1")

ALL = sorted(set(re.findall(r"wps_[a-z]+(?:_[a-z]+)*", raw)))
# 丢弃粘连 token / 版本串
ALL = [n for n in ALL if not re.search(r"(exewps|_vers$|^wps_w$)", n)]
# 丢弃是更长名字前缀的短名（分组名或粘连），避免重复匹配
ALL = [n for n in ALL if not any(o != n and o.startswith(n) for o in ALL)]

start = raw.find("Search for files on disk by name or content keyword")
blob = raw[start:start + 150000]

occ = [(m.start(), m.end(), n) for n in ALL for m in re.finditer(rf"(?<![a-z_]){re.escape(n)}(?![a-z_])", blob)]
occ.sort()
best = {}
prev_end = 0
for s, e, n in occ:
    desc = re.sub(r"[\x00-\x1f\x7f]", " ", blob[prev_end:s])
    desc = re.sub(r"\s{2,}", " ", desc).strip()
    if len(desc) > len(best.get(n, "")):
        best[n] = desc
    prev_end = e

GROUPS = {"wps_doc", "wps_pdf", "wps_ppt", "wps_sheet"}
LEGACY = {"wps_writer", "wps_spreadsheet", "wps_presentation"}
sem = [n for n in sorted(best) if n not in GROUPS and n not in LEGACY]
groups = [n for n in sorted(best) if n in GROUPS]
legacy = [n for n in sorted(best) if n in LEGACY]


def cat(n):
    return {"doc": "Writer/docx", "pdf": "PDF", "ppt": "Presentation/pptx",
            "sheet": "Spreadsheet/xlsx"}.get(n.split("_")[1], "通用/其它")


md = ["# wpscli.exe 工具面（静态抽取 · 续批）", "",
      f"- 二进制: `{EXE}`（8.0 MB，Rust + rmcp）",
      "- 抽取方式: 二进制内串联字符串 `[描述][工具名]` 成对切段（静态，未执行目标）",
      "- 动态校验: `wpscli.exe --help` → `错误：不支持非金山应用调用本程序。`（`src\\parent_verify.rs`，exit=1）"
      " → **tools/list 拿不到，schema 只能静态还原**",
      f"- 语义工具 **{len(sem)}** 个（首批按 30 记，本批实测 {len(sem)}）；legacy 逃生舱 {len(legacy)} 个；分组名 {len(groups)} 个",
      "", "## 工具清单与描述原文", "", "| 工具 | 类别 | 描述（原文节选） |", "|---|---|---|"]
for n in sem:
    d = best[n].replace("|", "\\|")
    md.append(f"| `{n}` | {cat(n)} | {d[:320]}{'…' if len(d) > 320 else ''} |")
md += ["", "## legacy 逃生舱（任意 JSAPI 脚本）", "", "| 工具 | 描述原文 |", "|---|---|"]
for n in legacy:
    md.append(f"| `{n}` | {best[n][:320].replace('|', chr(92) + '|')} |")
if groups:
    md += ["", "## 分组名（非工具）", "", "、".join(f"`{g}`" for g in groups)]

md += ["", "## 参数词汇（工具间复用，来自 schema 字段串）", "",
       "`file` 目标文档绝对路径 · `output` 输出文件/目录 · `script`/`script_file` JSAPI 逃生舱 · "
       "`ui` headed 真界面（默认 headless） · `format` 导出格式 · "
       "`what` 读取粒度（full/page/metadata/bookmarks/fields/tables/outline） · "
       "`target` 定位（find/paragraph/range/slide/shape/placeholder） · "
       "`data`/`from_text`/`from_json`/`from_csv` 创建与填充 · `range`/`chart_type`/`operations` 表格。", "",
       "## 与 gaea 对照", "",
       "- 覆盖面 ≈ gaea `office_*`（word 21 / excel 24 / ppt 19 类 op）。灵犀**只多**：`wps_doc_check`（校对）、"
       "`wps_file_info`、`wps_search`、`wps_image_extract`（OCR）。",
       "- 差异不在 op 数量，在**通道形态**：MCP 协议 + `script/script_file` 逃生舱 + `ui` 开关（headless↔headed 同一工具双模）。",
       "- 可照抄的最小形态：任何高层工具都配 `script` + `script_file` 双入参与 `ui` 布尔，模型不必因缺 op 而放弃。"]

out = sys.argv[1] if len(sys.argv) > 1 else "WPSCLI-tools.md"
open(out, "w", encoding="utf-8").write("\n".join(md))
print(f"semantic={len(sem)} legacy={len(legacy)} groups={len(groups)} total_named={len(best)}")
print("\n".join(md[:14]))
