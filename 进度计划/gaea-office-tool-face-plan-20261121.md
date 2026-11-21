# 办公工具面双轴瘦身施工单（精简/标准）
2026-11-21 · 依据众神殿 pan_1795223234328（正议 3 轮）+ pan_1795224680124（续议 3 轮），用户三次确认
基准 2e7a8ac3 · 实施席开工

## 一、终案（四席+主席收口，用户拍板逐字）

- **两轴正交**：`run_mode`（standard/cost/proposal，既有）× 新键 `tool_face = "lean"|"standard"`
  （UI 显示「精简/标准」）。**新建会话默认 lean（精简）**；一键「标准」恢复 browser 开、产图开、
  COM 跟随本机；**CAD+进度在两档都关**。
- **组覆盖**：`[office.tool_groups]` 保存 `group_id = true|false`（高级控制）；改动=登记默认值，
  新会话生效（复用面板 hidden 现成文案）；computer 沿用 `[computer_use].enabled` 注册闸不复制开关。
- **语义=Hide**（schema 不可见、协议层按名可调），非 deny；deny 层为独立备选任务。

## 二、功能组目录（tool 包单源，守卫钉死）

| 组 | 成员 | 备注 |
|---|---|---|
| cad | cad_ask(1) | **造价档默认开**（2026-11-21 用户拍板改排：CAD 并入造价）；标准/方案档关 |
| schedule | schedule_get/apply/analyze(3) | **方案档默认开**（用户拍板：进度并入方案）；标准/造价档关 |
| browser | browser_*×12 | lean 关 / standard 开 |
| com | office COM 十件 | lean 关 / standard 跟随本机（注册闸 officeComAvailable） |
| computer | computer_*×6 | 不参与 Hide——注册闸 [computer_use].enabled |
| cost | cost_search/save/compose/indicators/cost_project/apply/audit(7) | 按 run_mode（standard 档藏/cost 档开） |
| longdoc | longdoc_manifest/build(2) | 按 run_mode |
| media | image_gen/chart_gen/diagram/diagram_gen(4) | **两档默认开**（用户签注：产图不改默认） |
| core | 其余全部 | 不可配置 |

守卫三断言：组成员 ∈ RegistryMap；一个可配置工具只属一组；未知组配置 fail-closed。

## 三、装配顺序（确定性）

run_mode → tool_face 默认 → 用户组覆盖 → 汇总 Hide 名单（排序固定）→ skillFilter 跟随最终启用组
（拒绝名单 {schedule-edit, cad-takeoff}，schedule/cad 组开启时恢复）→ 实际隐藏组排序 → ModeLine
确定性拼接（「运行模式：X｜工具面：Y」+未载入组+按名可调指引；相同配置逐字节相同；不写时间/计数）。

## 四、验收六断言 + 台账

断言：①精简档 Schemas() 无关组名（computer 走注册闸不装）；②组关时 L1 技能索引无
schedule-edit/cad-takeoff；③ModeLine 点名隐藏组；④切标准档 browser×12 回、CAD/进度仍关；
⑤开组新建会话全回归；⑥关组工具按名受控探针可调（computer 除外）。
台账：census 三数基线（len(Schemas)/schema 字节/COM 开关差——70/92 为静态规划值不得写实测）；
30 天频次走后端 tool_dispatch 聚合（toolCounts 是会话内计数不可替代）；装机 [computer_use]
未核；第二刀（→≤55）触发=尾部 20 件调用占比<1%；fact_* 合并与 multi_edit 软退役 A/B 后置。

## 五、落点

- tool 组目录 `internal/gaea/tool/toolgroups.go`；配置 `internal/gaea/config/config.go` [office]
  扩 tool_face+tool_groups；组合器 `internal/app/office_runmode.go`（composeHideList）；装配
  `gaea_handler.go`；绑定 GaeaSetToolFace/GaeaToolFace/GaeaSetToolGroup/GaeaResetToolGroups；
  面板 `ToolsSection.tsx` 组开关+预设钮（ToolSurfaceEntry.Group 数据源）。
- 面态记录：ga.toolFace+组覆盖快照，GaeaNewSession 判面重建（复用会话创建时定档路径）。
