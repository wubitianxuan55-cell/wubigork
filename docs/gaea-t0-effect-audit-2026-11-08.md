# 议事 T0 effect 机械审计·第一遍（2026-11-08）

> **来源**：众神殿结论 pan_1794098356822 第二步（「T0 准入 = 命名白名单 ∩ ReadOnly() ∩ effect 的动作与位置声明 ∩ 运行时路径约束」）。
> **方法**：机械 sweep `internal/gaea/tool/builtin`（48 工具）——按文件级 ReadOnly 标记 + 副作用标记（起子进程/写/删/建目录/改名/系统临时写/网络出站）分类。**这是第一遍机械审计，不构成安全结论**：标记是文件级的（一文件多工具会互相污染），命中≠有罪（可能是自清理临时产物），未命中≠无罪。结论自己的口径：按 effect 表逐项人工审计 + 注册测试，预计 1-2 个工作日。
> **漂移锁**：`TestPantheonGrokT0ExactSets`（两名单逐字钉死）+ `TestPantheonGrokT0ExcludesDocRender`（doc_render 专项）已随本批落地。

## 审计发现（按优先级）

1. **⚠ cad_ask（议事 T0 在册！）带删文件标记**——CAD 渲染链的临时 PNG 清理还是越界删？需人工核 `cad_ask.go` 的删除路径是否限自身临时产物。复核 exoneration 前，按结论条件 3 同款处理：或移出 T0，或补 effect 声明+路径约束。
2. **⚠ read_file（议事 T0 在册！）带起子进程标记**——readfile.go 起什么进程（疑似 PDF/格式抽取外部工具）？同上需复核。
3. doc_render（已移出 T0，本批生效）带建目录标记——与 codex-r4/zcode-r3 取证一致（工作区删/建+系统临时写），维持除名。
4. todo_write（非 T0）带写文件标记——写面工具，纪律上禁入议事 T0，现状合规。
5. web_fetch/web_search（T0 办公只读在册）网络出站=本职；结论风险第 6 条的敏感数据出境/注入防护（域白名单+凭据禁出+来源标记）是独立工程，列后续池。
6. bash_output/kill_shell/wait（非 T0）起子进程标记——后台作业管理族（观察/终止既有作业），复核后大概率豁免。

## 全量分类表

| 工具 | 文件 | effect 标记 | 分类 |
|---|---|---|---|
| bash | bash.go | 起子进程,删文件 | 写面(禁入T0) |
| bash_output | bgjobs.go | 起子进程 | ⚠可疑 |
| kill_shell | bgjobs.go | 起子进程 | ⚠可疑 |
| wait | bgjobs.go | 起子进程 | ⚠可疑 |
| browser_navigate | browser_tools.go | - | 写面(禁入T0) |
| browser_read | browser_tools.go | - | 写面(禁入T0) |
| browser_snapshot | browser_tools.go | - | 写面(禁入T0) |
| browser_click | browser_tools.go | - | 写面(禁入T0) |
| browser_type | browser_tools.go | - | 写面(禁入T0) |
| browser_press | browser_tools.go | - | 写面(禁入T0) |
| browser_scroll | browser_tools.go | - | 写面(禁入T0) |
| browser_close | browser_tools.go | - | 写面(禁入T0) |
| browser_tabs | browser_tools.go | - | 写面(禁入T0) |
| browser_new_tab | browser_tools.go | - | 写面(禁入T0) |
| browser_switch_tab | browser_tools.go | - | 写面(禁入T0) |
| cad_ask | cad_ask.go | 删文件 | ⚠可疑 |
| chart_gen | chart_gen.go | 起子进程,建目录 | 写面(禁入T0) |
| complete_step | completestep.go | - | 纯读 |
| cost_compose | cost_compose.go | - | 写面(禁入T0) |
| cost_indicators | cost_indicators.go | - | 纯读 |
| cost_search | cost_tools.go | - | 写面(禁入T0) |
| cost_save | cost_tools.go | - | 写面(禁入T0) |
| diagram_gen | diagram_gen.go | 起子进程,建目录 | 写面(禁入T0) |
| doc_render | docrender.go | 建目录 | ⚠可疑 |
| edit_file | editfile.go | - | 写面(禁入T0) |
| edit_lines | editlines.go | - | 写面(禁入T0) |
| format_convert | format_convert.go | 写文件,建目录 | 写面(禁入T0) |
| genui_validate | genui_validate.go | - | 纯读 |
| grep | grep.go | - | 纯读 |
| knowledge_add | knowledge_add.go | - | 写面(禁入T0) |
| knowledge_search | knowledge_search.go | - | 写面(禁入T0) |
| ls | ls.go | - | 纯读 |
| memory_search | memory_search.go | - | 纯读 |
| move_file | movefile.go | 删文件,建目录,改名/移动 | 写面(禁入T0) |
| multi_edit | multiedit.go | - | 写面(禁入T0) |
| read_file | readfile.go | 起子进程 | ⚠可疑 |
| read_skill | readskill.go | - | 纯读 |
| read_spill | readspill.go | - | 纯读 |
| schedule_get | schedule_tools.go | - | 纯读 |
| schedule_apply | schedule_tools.go | - | 纯读 |
| schedule_analyze | schedule_tools.go | - | 纯读 |
| screen_capture | screenshot.go | 写文件,建目录 | 写面(禁入T0) |
| sidebar_open | sidebar_open.go | - | 写面(禁入T0) |
| todo_write | todo.go | 写文件 | ⚠可疑 |
| vision | vision.go | - | 纯读 |
| web_fetch | webfetch.go | 网络出站 | ⚠可疑 |
| web_search | websearch.go | - | 纯读 |
| write_file | writefile.go | 建目录 | 写面(禁入T0) |