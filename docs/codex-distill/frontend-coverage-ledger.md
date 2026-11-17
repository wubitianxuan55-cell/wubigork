# codex 0.165.0 前端**全量覆盖台账**（tui/src 逐文件 → 覆盖篇 / 判定）

> 目的：把 `codex-rs/tui/src` 的**每一个前端文件**都钉到"哪一篇覆盖、判定是什么"，
> 让"前端已逆完"这句话可核对，而不是凭感觉。
>
> 口径
> - 证据根：`C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs\tui\src`（`tui` 是唯一前端 crate，371 个 `.rs`；
>   其中顶层非测试文件 **160 个**，另有 24 个子目录）；
> - 覆盖篇：波 1 = `asset-27…33`（消息流与渲染面），波 2 = `asset-34…37`，波 3 = `asset-38…46`；
> - 判定三档：**覆盖**（已出资产文档）/ **旁证**（在别的篇里作为对比或前提出现，无独立文档）/ **不搬**（终端专属，附理由分类）。

---

## 1. 覆盖总览

| 波次 | 篇数 | 主题 | 文件 |
|---|---|---|---|
| 波 1 | 7 | 流式渲染 / 转录视图 / markdown / diff / 输入面板 / 键位 / 状态动效剪贴板 | `asset-27…33` |
| 波 2 | 4 | 工具执行卡 / 输入弹层 / 会话面 / 回退与提示 | `asset-34…37` |
| 波 3 | 9 | 多智能体 / 设置主题 / 用量统计 / 首启信任 / 集成面 / 工作树环境 / 浮层度量 / 动效视觉 / 更新运行 | `asset-38…46` |
| 合计 | **20 篇** | 覆盖 `tui` crate 全部前端模块面 | — |

**160 个顶层文件 + 24 个子目录**的归属见 §2（按主题）与 §4（不搬清单）。

---

## 2. 主题台账（逐文件）

> 括号内为行数；**粗体**为有独立资产文档的模块。

### A. 会话主链与版本/会话状态 → 28 / 36 / 41 / 43

- **核心状态机**：`app.rs`(1237)、`app_event.rs`(1545)、`app_command.rs`(339)、`app_event_sender.rs`(147) → 「agent 型事件总线」旁证（波 1 §1）
- **会话记录**：`session_state.rs`(71)、`session_log.rs`(242)、`session_resume.rs`(221)、`session_start.rs`(157)、
  `session_archive_commands.rs`(395)、`session_queue_commands.rs`(138) → 36 §五、46 §五
- **命名与恢复**：`named_session_lookup.rs`(165) → 36 §四；`resume_permissions.rs`(52) → 旁证（权限继承）
- **分支与 Git 摘要**：`branch_summary.rs`(688) → 43 §五；`get_git_diff.rs`(875)、`git_action_directives.rs`(279) → 旁证（30 diff 的输入侧）
- **目标**：`goal_display.rs`(98)、`goal_files.rs`(227) → 43 §四
- **协作模式**：`collaboration_modes.rs`(94) → 38 §七

### B. 转录与渲染 → 27 / 28 / 29 / 30 / 34 / 44

- **转录**：`transcript_view.rs`(667)、`thread_transcript.rs`(328) → 28、38 §七；`transcript_reflow.rs`(274)、`resize_reflow_cap.rs`(165) → 44 §二
- **行与区间**：`rendered_selection.rs`(182)、`text_selection.rs`(72)、`vim_search.rs`(39) → 28（选区/搜索）、44 §三
- **文本度量**：`width.rs`(99)、`wrapping.rs`(1888)、`live_wrap.rs`(270)、`line_truncation.rs`(89)、`text_formatting.rs`(497) → 44（**判定不搬**，除 `line_truncation` 语义）
- **Markdown / diff**：`markdown.rs`(716)、`markdown_render.rs`(3056)、`markdown_stream.rs`(830)、`markdown_copy.rs`(601)、`markdown_text_merge.rs`(43)、
  `diff_render.rs`(2540)、`diff_model.rs`(18)、`table_detect.rs`(435) → 29、30
- **高亮与渲染件**：`render/`(9 文件：`highlight.rs` 63KB 等)、`style.rs`(380)、`color.rs`(63)、`thread_color.rs`(74)、`ui_consts.rs`(16) → 旁证（29/34）
- **终端载体**：`tui.rs`(1425)、`insert_history.rs`(1066)、`custom_terminal.rs`(1292)、`terminal_hyperlinks.rs`(1262)、
  `terminal_palette.rs`(518)、`terminal_probe.rs`(877)、`terminal_program_status.rs`(271)、`terminal_title.rs`(252)、
  `terminal_size_telemetry.rs`(58)、`terminal_visualization_instructions.rs`(26)、`live_wrap.rs` → **不搬**（§4）
- **浮层**：`pager_overlay.rs`(534) → 44 §一；`transcript_mode.rs`(21) → 旁证
- **图片与可视化**：`transcript_image.rs`(159)、`inline_visualization.rs`(291) → 45 §四

### C. 输入面与命令 → 31 / 32 / 35 / 37

- **底部面板**：`bottom_pane/`(83 文件)：`mod.rs`(162KB)、`chat_composer.rs`(504KB)+`chat_composer/*`(37 文件)、`textarea/*`(14)、
  `mentions_v2/*`(9)、`popup_state.rs`、`footer_state.rs`、`paste_burst.rs`、`list_selection_view.rs`、`selection_popup_common.rs`、
  `command_popup.rs`、`file_search_popup.rs`、`skill_popup.rs`、`scroll_state.rs`、`picker_*.rs` → 31、35
- **弹层族**：`bottom_pane/{approval_overlay(request_user_input/async_questions/multi_select_picker/mcp_server_elicitation/hooks_browser_view/
  skills_toggle_view/memories_settings_view/app_link_view/experimental_features_view/user_verification/feedback_view/feedback_note_view/
  custom_prompt_view)/*.rs` → 31（审批/追问）、42（集成面）、41（验证）
- **命令与提及**：`slash_command.rs`(361)、`mention_codec.rs`(615)、`task_mentions.rs`(350)、`file_search.rs`(119) → 35
- **键位与提示**：`keymap.rs`(3872)、`keymap_setup.rs`(1999)、`keymap/`(8 文件，含 `chords.rs` 32KB)、`key_hint.rs`(435)、
  `shortcut_help.rs`(91)、`footer_hint.rs`(57) → 32、37 §四
- **剪贴板**：`clipboard_copy.rs`(798)、`clipboard_paste.rs`(520)、`clipboard_html.rs`(46)、`copy_input_guard.rs`(10)、`markdown_copy.rs` → 33
- **提示与帮助**：`tooltips.rs`(601) → 45 §四；`turn_tip.rs`(27)、`status_indicator_widget.rs`(531) → 33（旁证）、45

### D. 审批与权限 → 31 / 41 / 42

- `approval_events.rs`(115)、`auto_review_denials.rs`(132)、`app_server_approval_conversions.rs`(175) → 31 §七、13（后端）
- `permission_discovery.rs`(168)、`security_setup.rs`(162) → 41 §三
- `windows_sandbox.rs`(159)、`activation_key_guard.rs`(61)、`ide_context.rs`(105) → 旁证（沙箱/守卫/IDE 上下文）
- `user_prompt.rs`(70)、`resume_permissions.rs`(52) → 旁证

### E. 多智能体 → 38

- `multi_agents.rs`(910) → 38 §七
- `app/`(134 文件)：`agents_overview*.rs`（`agents_overview.rs` 52.9KB + view 30.5KB + actions/details/discovery/grouping/threads/usage/new）、
  `agent_center/*`、`agent_picker.rs`、`agent_navigation.rs`、`agent_status_feed.rs`、`agent_message_consolidation.rs` → 38 §一…§七
- `collaboration_modes.rs`(94) → 38 §七

### F. 设置、主题、身份与版本 → 39 / 46

- **设置**：`config_update.rs`(440)、`local_settings.rs`(266)、`debug_config.rs`(1352)、`experimental_features.rs`(156) → 39 §一/§四/§五
- **主题与配色**：`theme_picker.rs`(571)、`terminal_palette.rs`(518)、`color.rs`(63) → 39 §三
- **模型与档位**：`model_catalog.rs`(40)、`model_migration.rs`(549)、`service_tier_resolution.rs`(95) → 旁证（39/40 的上游数据）
- **身份与版本**：`backend_id.rs`(10)、`app_info.rs`(119)、`version.rs`(2)、`local_chatgpt_auth.rs`(206)、`npm_registry.rs`(112) → 41（身份）、46（版本）
- **更新**：`update_action.rs`(188)、`update_versions.rs`(167)、`update_prompt.rs`(335)、`updates.rs`(150)、`updates_cache.rs`(46) → 46 §一/§二
- **其它偏好**：`daybreak.rs`(101)、`clock_format.rs`(182) → 旁证（主题时段/时间格式）

### G. 集成面 → 42

- `dynamic_tools.rs`(1582)、`dynamic_tools_mcp.rs`(385)、`dynamic_tools_response.rs`(113) → 42 §一（schema 驱动）
- `hooks_rpc.rs`(94)、`bottom_pane/hooks_browser_view.rs`(1689) → 42 §三
- `skills_helpers.rs`(41)、`bottom_pane/skills_toggle_view.rs`(533) → 42 §三
- `bottom_pane/{mcp_server_elicitation.rs,app_link_view.rs,memories_settings_view.rs}` → 42 §一/§二/§四/§五
- `subscription.rs`(38)、`backend_banners.rs`(69) → 旁证（后端公告带）

### H. 用量与统计 → 40

- `analytics.rs`(520)、`analytics/`(54 文件：`dashboard/summary_panel/plan_panel/chat_panel/task_panel/report_data/chart/activity_chart/
  plot/normalize/controls/navigation/hints/data/chrome/tokens/styles/mouse/recovery/client` 等) → 40 §四
- `status/`(10 文件：`card.rs` 37KB、`rate_limits.rs` 20KB、`thread_usage.rs`、`helpers.rs`、`remote_connection.rs`、`format.rs`、`account.rs`) → 40 §一…§三
- `token_usage.rs`(77) → 40 §三

### I. 首启与启动链 → 41

- `onboarding/`(9 文件：`onboarding_screen.rs` 43KB、`welcome.rs`、`trust_directory.rs`、`directory_trust.rs`、`auth.rs` 61KB、`bedrock.rs` 35KB) → 41 §一/§二
- 启动链：`startup_preflight.rs`(69)、`startup_presentation.rs`(90)、`startup_recovery.rs`(163)、`startup_orchestration.rs`(954)、
  `startup_hooks_review.rs`(493)、`startup_draft.rs`(502)、`startup_draft_input.rs`(255)、`startup_draft_layout.rs`(109)、`startup_error.rs`(34) → 41 §四
- `cwd_prompt.rs`(542)、`external_editor.rs`(292) → 39 §六、43

### J. 工作树与环境 → 43

- `worktree_browser.rs`(159)、`worktree_startup.rs`(292)、`managed_worktree_tools.rs`(325)、`managed_worktree_tool_specs.rs`(17)、
  `projectless.rs`(107)、`workspace_command.rs`(190)、`additional_dirs.rs`(135)、`oss_selection.rs`(418) → 43
- `workspace_messages.rs`(24)、`managed_new_thread_defaults.rs`(48) → 旁证

### K. 动效与视觉件 → 45

- `motion.rs`(136)、`system_motion.rs`(73)、`shimmer.rs`(72)、`summary_shimmer.rs`(58)、`empty_state_animation.rs`(273)、
  `empty_state_animation/`(6 文件，含 `policy.rs`) → 45 §一…§三
- `pets/`(10 文件 ~130KB) → **不搬**（彩蛋层，见 §4）

### L. 剪贴板/系统集成/运行时（终端专属）→ **不搬**

`screen_reader.rs`(181)、`screen_reader_windows.rs`(60)、`daemon_startup.rs`(241)、`daemon_recovery.rs`(212)、
`daemon_telemetry.rs`(62)、`app_server_session.rs`(4278)、`app_server_connection.rs`(47)、`terminal_*`、`custom_terminal.rs`、
`insert_history.rs`、`tui.rs`、`terminal_probe.rs`、`terminal_program_status.rs`、`terminal_size_telemetry.rs`、
`terminal_visualization_instructions.rs`、`transcript_image.rs`(协议部分)、`test_backend.rs`(112)、`test_support.rs`(62)、
`total`(不计) — 详见 §4

### M. 文档与常量 → 旁证

`lib.rs`(2369)、`main.rs`(57)、`cli.rs`(124)、`ui_consts.rs`(16) — crate 装配与入口，无独立 UI 语义。

---

## 3. 判定分布（160 个顶层文件）

| 判定 | 数量 | 说明 |
|---|---|---|
| 覆盖（有独立资产文档） | ≈44 | 波 1–3 的主体模块（含子目录内文件） |
| 旁证（在别篇作为对比/前提） | ≈72 | 如 `app.rs`、`app_event.rs`、`model_catalog.rs`、`render/*`、`session_*` 的存储侧 |
| 不搬（终端专属，§4 给理由） | ≈44 | 终端渲染/协议/输入设备/守护进程/彩蛋 |

> 注：子目录（`app/`134、`bottom_pane/`83、`chatwidget/`104、`transcript_view/`44、`history_cell/`38、`analytics/`54、
> `streaming/`16、`status/`10、`onboarding/`9、`keymap/`8、`render/`9、`tui/`31、`pets/`10、`empty_state_animation/`6 …）
> 已按其所属主题并入 §2 对应行；`chatwidget/*` 与 `streaming/*` 原文已由波 1 覆盖。

---

## 4. "不搬"清单与理由（终端专属的六类）

1. **终端渲染与回放**：`custom_terminal.rs`、`insert_history.rs`、`tui.rs`、`terminal_probe.rs`、`terminal_program_status.rs`、
   `terminal_size_telemetry.rs`、`resize_reflow_cap.rs` 的具体数值 → Web 由 DOM/Blink 负责；
   可迁移的只有"重建量不得超过宿主保留量"的**决策**（已在 44 §二保留）。
2. **终端协议**：`terminal_hyperlinks.rs`（OSC 8）、`terminal_title.rs`、`notifications/{osc9,bel}.rs`、`pets/{sixel,image_protocol}.rs`、
   `transcript_image.rs` 的图片协议 → 无对应 DOM 能力（Web 用 `<a>`/`document.title`/Notification API/`<img>`）。
3. **字符级度量与换行**：`wrapping.rs`、`live_wrap.rs`、`width.rs` 的 grapheme/sentinel 机制 → 浏览器换行；
   仅"按显示宽度截断"这一条语义被 44 §三 采用。
4. **输入设备与 vim**：`bottom_pane/textarea/vim*.rs`、`vim_search.rs`、`mouse.rs`（终端鼠标协议） → Web 用原生选区/编辑器。
5. **进程与守护**：`daemon_*`、`app_server_session.rs`、`screen_reader_*`、`windows_sandbox.rs` → 由 Wails/Go 侧承担或不存在。
6. **彩蛋层**：`pets/*` → 产品判断（非机制）。

---

## 5. 维护约定

- 新增一篇资产时，**必须同时更新本台账**（§2 对应行 + §1 计数），否则视为未完成。
- 判定从"不搬"改为"覆盖"时，要写清楚**为什么现在值得搬**（否则台账会漂）。
- 上游版本变化（如 0.166）只需重跑：`tui/src` 文件清单 → 与本台账 diff → 差异项重新归档。
