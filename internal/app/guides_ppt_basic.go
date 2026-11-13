package app

// guides_ppt_basic.go — ppt 的 office_edit 操作清单基础版（2026-11-13 波2 拍板：
// PPT 成品版式引擎取消不做——helper 的 designed slides 实现随 OfficeHelper.cs
// 逐字节保真保留（升级 diff 能力不破坏），但教学面只发基础 ops 清单（上游
// GUIDES ppt 的「touching up a designed slide」段机械提取，提取脚本
// .tmp/extract_guides_ppt_basic.mjs），不引导模型用 slide kind/theme 成品版式。
// word/excel 两清单仍见 guides.go（全量上游对齐）。

// officeGuidesPptBasic = ppt 应用基础 ops 清单全文（上游过渡句「The operations
// below are for touching up...」在本清单独立发放时无上文，已摘除）。
var officeGuidesPptBasic = `PowerPoint ops. slide = slide number; shape = a shape name or "#id" from office_read, or "title" / "body". Positions and sizes are in points. Formulas: write LaTeX between dollar signs in any slide text (title, body, text box), as in Word; they become native PowerPoint equations, inline with the text. A formula on its own gets a text box of its own.
- add_slide {layout?: title|title_content|two_content|title_only|blank|section, title?, body?, notes?, at?} — body: lines separated by newlines become bullets.
- set_text {slide, shape, text, size?, bold?, color?, align?} — a line that starts with a tab (or two spaces) is a bullet one level down.
- format_text {slide, shape, find?, font?, size?, bold?, italic?, underline?, color?, align?, bullets?, lineSpacing?, fit?: "shrink"|"grow"|"none"} — the whole text of the shape, or only the part that reads find.
- add_textbox {slide, text, left, top, width, height, size?, bold?, color?, align?, fill?, name?}
- add_image {slide, path, left, top, width?, height?, name?}
- add_smartart {slide, layout, items, left?, top?, width?, height?, colors?, look?, name?} — a native SmartArt diagram. layout: list | bullet_list | horizontal_list | process | chevron | arrows | timeline | steps | cycle | radial | hierarchy | org_chart | tree | pyramid | venn | matrix | funnel | target | balance | gear, or the name a layout has in PowerPoint. items: texts, or {text, children:[..]} for the levels below (the branches of a hierarchy, the bullets under a step). Prefer it to hand-drawn boxes and arrows for steps, cycles, structures and comparisons.
- add_table {slide, data:[[cell,..],..], left?, top?, width?, height?, size?, font?, align?, header?, name?} — a native table; formulas in its cells are set as text with real subscripts and powers.
- add_shape {slide, kind, left, top, width, height, text?, fill?, line?: colour|"none", lineWidth?, size?, bold?, color?, name?} — kind: rectangle | rounded | ellipse | diamond | triangle | arrow | arrow_left | arrow_up | arrow_down | chevron | pentagon | hexagon | star | callout | cloud | line | arrow_line (for a line, width and height are how far it runs).
- set_shape {slide, shape, left?, top?, width?, height?, fill?, name?}; delete_shape {slide, shape}
- reuse_slide {from, texts?, images?, delete?, at?, notes?}; delete_slides {from, to}; delete_slide {slide}; move_slide {slide, to} (to = the number the slide has afterwards; page numbers follow); duplicate_slide {slide, to?}; set_layout {slide, layout}; set_background {slide, color}; set_notes {slide, text}
`
