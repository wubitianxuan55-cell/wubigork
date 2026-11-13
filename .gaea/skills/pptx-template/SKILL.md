---
name: pptx-template
description: "Use this skill when the user wants to replicate an existing PowerPoint template: 母版复刻 / 按模板做 PPT / 沿用这份模板的版式 / template deck / 模板复刻。五步流水线 flatten → shell-report → make-skeleton → apply-flat → assemble，保母版版式换内容，16:9 硬约束，字体白名单∩系统字体。新建（无模板）走 pptx 技能；局部编辑已打开的文件走 office_edit。"
license: MIT
---

# PPTX 模板复刻流水线

## 入口互斥（先判路线，选错就退化）

- **用户给了模板 pptx，要求沿用其版式** → 本技能（五步流水线）。
- 无模板从零生成 → `pptx` 技能（create_pptx.py）。
- 用户开着的实时文档改几处 → `office_edit`（COM 实时档）。
- 三条路线不得混用：模板复刻的产物由母版决定样式，不手拼格式。

## 五步流水线（失败即停，任何一步 success:false 就停下报给用户）

```bash
S=scripts           # run_skill 后相对技能根
T=<任务目录>        # 产物在任务目录根部，不散落

# 1. 摊平：模板 → flat.json（每页版式/占位符/字体/尺寸）+ assets/ 图片
python $S/flatten.py 模板.pptx -o $T/flat.json --assets $T/assets

# 2. 母版体检：16:9 复核（非 16:9 直接判败）+ 字体清单×系统∩ + 版式结构
python $S/shell-report.py 模板.pptx

# 3. 空骨架：保母版/版式/主题，删内容页
python $S/make-skeleton.py 模板.pptx -o $T/skeleton.pptx

# 4. 填充：结构化文案按版式占位符填入（页→版式→占位符映射即时记录）
python $S/apply-flat.py $T/skeleton.pptx $T/spec.json -o $T/final.pptx

# 5. 终检守门：16:9/字体合规/无占位符残留/页数对账
python $S/assemble.py $T/final.pptx --min-pages 5

# 6. 视觉验收（交付门，必过）：assemble 通过 ≠ 排版没跑——
#    有 office_ 工具族：office_render {doc:$T/final.pptx, overview:true} 渲染
#    总览 PNG 逐页目检（溢出/错版/占位错位）；
#    无 COM：soffice --headless --convert-to pdf + pdftoppm 出图抽查。
#    目检不过 → 修 spec（精简要点/拆页）重跑第 4~6 步；不许带病交付。
```

## spec.json 怎么写

```json
{
  "font": "微软雅黑",
  "slides": [
    {"layout": "标题幻灯片", "title": "季度经营汇报", "subtitle": "2026 Q4", "notes": "开场 30 秒"},
    {"layout": "标题和内容", "title": "总体业绩", "bullets": ["营收同比 +18%", "毛利率 42.3%"]}
  ]
}
```

- `layout` 名**必须**来自第 1 步 flat.json 的 `layouts` 清单（或第 3 步返回值），不存在即失败。
- `title`/`subtitle`/`bullets`（或 `body`）按占位符角色匹配；版式没有对应占位符时**失败即停**，不要换版式硬塞。
- `font` 省则用体检的 fallback（微软雅黑）；溢出只是粗估 warning，交付前用 `office_render {overview:true}` 目检。

## 硬约束与纪律

1. **仅 16:9**（13.33×7.5in）；非 16:9 模板第一步体检就判败，不改造母版。
2. **字体白名单 ∩ 系统字体**（GDI 枚举实测），空交集回退微软雅黑；模板里带出的白名单外字体在终检会被点名。
3. **失败即停**：任何一步 success:false 终止后续写操作，把 message 原样报给用户。
4. 产物在任务目录根部；`flat.json`/`spec.json`/`skeleton.pptx`/`final.pptx` 命名固定，方便对账。

## 依赖

python-pptx（本机已装）；无云依赖。图片沿用模板内嵌资源（flatten --assets 导出），
spec 暂不支持插图——需要插图时在 apply-flat 之后用 python-pptx 手工补或 office_edit。
