---
name: pptx
description: "Use this skill whenever the user wants to create or edit PowerPoint presentations (.pptx), or asks for 演示文稿 / PPT / 幻灯片 / 汇报材料 / 汇报PPT / slides. Generates a clean 16:9 deck from structured content (封面 + 章节 + 要点 + 演讲备注) with python-pptx."
license: MIT
---

# PPTX 演示文稿生成

## 概述

用 python-pptx 从结构化内容生成排版规整的 16:9 .pptx 演示文稿。适合汇报材料、项目介绍、培训课件、方案讲解等场景。生成的文件可被 PowerPoint / WPS 直接打开继续编辑。

## 快速开始

### 1. 准备内容大纲

先确认或自行规划大纲，通常包含：封面 → 目录/概述 → 章节页（2-5 页）→ 总结。每页内容遵循「标题 + 3-6 个要点」结构，要点短句化，避免整段粘贴。

### 2. 生成结构化 JSON

创建 spec 文件（UTF-8），结构如下：

```json
{
  "title": "月度经营分析汇报",
  "subtitle": "2026 年 7 月",
  "author": "gaea",
  "slides": [
    { "title": "总体业绩", "points": ["营收同比 +18%，环比 +4%", "毛利率 42.3%，提升 1.8pct", "新签合同额创近 12 个月新高"] },
    { "title": "问题与风险", "points": ["华东区回款周期拉长至 65 天", "A 产品线库存周转放缓", "Q3 人员招聘缺口 3 人"], "notes": "重点讲风险排序与应对" }
  ]
}
```

- `points` 中的条目可以是字符串（一级要点）或 `{"text": "...", "level": 1}`（二级要点，level 0 或省略为一级）
- `notes` 可选，写入演讲者备注
- 封面信息缺省时自动使用 `title` 作为封面标题

### 3. 运行生成脚本

```bash
python scripts/create_pptx.py spec.json 输出文件.pptx
```

脚本会自动检测 python-pptx；缺失时先 `pip install python-pptx`（失败则提示用户安装）。

### 4. 验证

- 脚本返回 JSON：`{"ok": true, "path": "...", "slides": N}`
- 用 `ls -l` 确认文件大小；必要时用 bash + python-pptx 重新打开检查页数
- 如用户要求调整（改标题/增删页/换内容），修改 spec 后重新生成

## 设计约定

- 画布 16:9（13.333 × 7.5 英寸）
- 封面：深蓝标题（#1F3864）+ 蓝色强调条 + 浅灰副标题/作者
- 内容页：顶部深蓝标题 + 蓝色下划线 + 正文要点（深灰 #404040，一级要点带蓝色 ▪ 标记）
- 中文字体优先 Microsoft YaHei，无则回退默认字体
- 每页要点建议 3-6 条，超长内容拆分多页

## 注意

- 只生成 .pptx（勿伪造 .ppt）
- 复杂图表先交给 chart_gen 生成图片，在 notes 里注明插入位置，或后续人工插入
- 文件路径含中文/空格时正常处理，用参数传递即可
