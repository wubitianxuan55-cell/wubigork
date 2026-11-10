# gaea 工具面注册表地图（P0）

> 2026-11-09 众神殿 pan_1794234593182 裁决 action-1 交付物。唯一注册表（`boot.Build` 单一 `tool.Registry`）全部 reg.Add 行的契约清单：来源 / 空间 / 副作用 / 证据档 / 验收谓词。本文件由 `go run scripts/registry_map_doc.go` 从 `internal/gaea/tool.RegistryMap` 单源生成，勿手改；防漂移守卫见 `tool/registrymap_test.go`、`boot/registrymap_census_test.go`、`app/registrymap_extra_test.go`。

/mcp__ 前缀是 prompt 旁路（controller_submit.go → input.MCPPrompt 取回 prompt 文本走 runTurnWithRaw），不进工具注册表、不构成第五工具面。

| 工具 | 来源 | 空间 | 副作用 | 证据档 | 验收谓词 |
|---|---|---|---|---|---|
| `ask` | build-internal | shared | interactive | none | 用户答案回传当前席 |
| `author_memory` | extra-app | work | read | none | 作者记忆读取一致 |
| `bash` | builtin-workspace | work+shared | write | publish-guard | durable 写被发布边界拦截/隔离（P1b）；前台+后台双路径 |
| `bash_output` | builtin-domain | shared | read | none | 输出增量与 job 缓冲一致 |
| `browser_click` | builtin-domain | shared | interactive | none | 点击后页面快照反映变化 |
| `browser_close` | builtin-domain | shared | interactive | none | 目标页/浏览器关闭 |
| `browser_navigate` | builtin-domain | shared | interactive | none | 到达目标 URL 快照 |
| `browser_new_tab` | builtin-domain | shared | interactive | none | 新 tab 快照 |
| `browser_press` | builtin-domain | shared | interactive | none | 按键后快照反映变化 |
| `browser_read` | builtin-domain | shared | read | none | 读取内容与页面一致 |
| `browser_scroll` | builtin-domain | shared | interactive | none | 滚动后快照反映变化 |
| `browser_snapshot` | builtin-domain | shared | read | none | 快照与 DOM 一致 |
| `browser_switch_tab` | builtin-domain | shared | interactive | none | 激活 tab 切换 |
| `browser_tabs` | builtin-domain | shared | read | none | tab 清单与实况一致 |
| `browser_type` | builtin-domain | shared | interactive | none | 输入后快照反映文本 |
| `cad_ask` | builtin-domain | work | read | none | 图框质量闸+vision 读图结论；T0 effect 已复核豁免 |
| `chart_gen` | builtin-workspace | shared | produce | agent-produce | 图片落点存在（单系列图回归钉） |
| `complete_step` | builtin-domain | shared | meta | none | 步骤裁决与计划一致 |
| `cost_compose` | builtin-domain | work | produce | none | 组价结果与定额飞轮口径一致 |
| `cost_indicators` | builtin-domain | work | read | none | 指标与 SchemaV34 快照一致 |
| `cost_save` | builtin-domain | work | write | none | PersistWrite：库写入可查 |
| `cost_search` | builtin-domain | work | read | none | 检索命中与库一致 |
| `dag_plan` | extra-app | work | meta | none | write_paths/readonly schema 闸（六闸刀口径） |
| `diagram` | extra-app | work | produce | none | 无边 nil 切片显式 null 契约（回归钉） |
| `diagram_gen` | builtin-workspace | shared | produce | agent-produce | 图片落点存在（无边树图回归钉） |
| `doc_render` | builtin-workspace | shared | read | none | 只读渲染 PNG；覆盖率字段 P2 |
| `docx_accept` | extra-app | work | write | office-journal | 仅 accept=true；基线快照+证据卡；通道 A=无本作者待处理修订引用级（P2） |
| `docx_apply` | extra-app | work | write | office-journal | 修订制写入；基线快照+证据卡；通道 A=文本流含替换+修订在册引用级（P2） |
| `edit_file` | builtin-workspace | shared | write | agent-write | 通道 A 摘要比对（新在旧亡）；expected_revision CAS 可选 |
| `edit_lines` | builtin-workspace | shared | write | agent-write | 通道 A 摘要比对（新在旧亡）；expected_revision CAS 可选 |
| `exit_plan_mode` | build-internal | shared | interactive | none | 计划呈现与批准流转 |
| `fact_add` | extra-app | work | write | none | PersistWrite：事实库写入可查 |
| `fact_clear` | extra-app | work | write | none | PersistWrite：清空可查 |
| `fact_list` | extra-app | work | read | none | 清单与库一致 |
| `forget` | build-internal | shared | write | none | PersistWrite：记忆删除可查 |
| `format_convert` | builtin-workspace | shared | produce | agent-produce | output 落点存在且格式正确 |
| `genui_validate` | builtin-domain | shared | read | none | 校验结论与产物一致 |
| `grep` | builtin-workspace | shared | read | none | 命中行与实况一致；offset 续页游标+凭据围栏跳过记账 |
| `image_gen` | extra-app | work | produce | none | 图片落 .gaea/uploads 系统路径可查 |
| `install_skill` | build-internal | shared | write | none | PersistWrite：技能落盘可查 |
| `interrupt_agent` | build-internal | shared | meta | none | 中断信号送达 |
| `kill_shell` | builtin-domain | shared | write | none | 目标 job 终止（级联子树） |
| `knowledge_add` | builtin-domain | work | write | none | PersistWrite：知识库写入可查 |
| `knowledge_search` | builtin-domain | work | read | none | 召回与库一致 |
| `ls` | builtin-workspace | shared | read | none | 清单与实况一致 |
| `mcp__<server>__<tool>` | plugin-mcp | 随插件 | 随工具 | 随工具 | 动态热插（plugins.go/controller_mcp.go reg.Add 同账）；名单凭证边界不枚举，P0 按非凭证方式标注 |
| `memory_get` | build-internal | shared | read | none | 记忆读取与库一致 |
| `memory_search` | builtin-domain | shared | read | none | 召回与记忆库一致 |
| `move_file` | builtin-workspace | shared | write | agent-write | 目标存在且源已迁移 |
| `multi_edit` | builtin-workspace | shared | write | agent-write | 通道 A 摘要比对（新在旧亡）；expected_revision CAS 可选 |
| `ocr` | extra-app | work | read | none | 全文提取与图片内容一致（seam 测试防真拉起） |
| `office_batch` | extra-app | work | meta | task-center | 批量转 PDF 进持久调度器：断窗续跑/可取消/可重试/进度逐文件；out_dir 声明=产物目录；Result 逐文件报告（P4） |
| `office_generate` | extra-app | work | produce | office-journal | 声明路径与实际落点逐字节一致；证据卡落账；通道 A=OOXML 容器头引用级（P1a+P2） |
| `pptx_apply` | extra-app | work | write | office-journal | 页级替换；基线快照+证据卡；通道 A=页文本替换在目标不在引用级（P2） |
| `promote_session_facts` | build-internal | shared | write | none | PersistWrite：事实晋升可查 |
| `read_file` | builtin-workspace | shared | read | none | 内容与目标文件一致；尾标 revision（CAS 源）+凭据围栏 deny |
| `read_skill` | builtin-domain | shared | read | none | 返回技能正文 |
| `read_spill` | builtin-domain | shared | read | none | 溢出读与落盘内容一致 |
| `read_subagent_result` | build-internal | shared | read | none | 子代理结果读取一致 |
| `recall_archived` | builtin-domain | shared | read | none | 归档召回与存档一致（显式 SpaceShared） |
| `remember` | build-internal | shared | write | none | PersistWrite：记忆写入可查 |
| `replace` | builtin-workspace | work | write | agent-write | 全计划预检零变更（任一文件超 32MiB 整次拒）；逐文件原子发布非事务（回执列已发布/失败/pending）；changed=替换次数；发布前锁内哈希重核 |
| `request_permission` | build-internal | shared | interactive | none | 审批请求与裁决回传 |
| `routine_llm` | extra-app | work | read | none | 子调用回执与 feature 绑定一致 |
| `run_skill` | build-internal | shared | produce | none | 技能子会话产物按其工具各自登记 |
| `schedule_analyze` | builtin-workspace | shared | read | none | 分析结论可复算 |
| `schedule_apply` | builtin-workspace | shared | write | agent-write | 声明路径包含校验 + 日程库变更可查 |
| `schedule_get` | builtin-workspace | shared | read | none | 日程读取与库一致 |
| `screen_capture` | builtin-workspace | shared | read | none | 截图落 .gaea/uploads 系统路径 |
| `semantic_search` | extra-app | work | read | none | 召回与向量库一致 |
| `session_search` | build-internal | shared | read | none | 检索命中与会话存档一致 |
| `sidebar_open` | builtin-workspace | shared | interactive | none | 侧栏打开对应文件 |
| `slash_command` | build-internal | shared | meta | none | 斜杠命令展开与注册一致 |
| `subagent_list` | build-internal | shared | read | none | 子代理清单与存档一致 |
| `summarize_file` | build-internal | shared | read | none | 摘要覆盖大文件截断语义 |
| `task` | build-internal | shared | meta | none | 子代理白名单/作用域闸（六闸刀口径） |
| `todo_write` | builtin-domain | shared | meta | none | 待办清单与库一致 |
| `translate_text` | extra-app | work | read | none | 译文可回读 |
| `vision` | builtin-workspace | shared | read | none | 描述对应图片内容 |
| `wait` | builtin-domain | shared | read | none | 等待结果与 job 终态一致 |
| `web_fetch` | builtin-workspace | shared | read | none | 抓取内容与源一致（注明 URL+时间） |
| `web_search` | builtin-domain | shared | read | none | 结果注明来源 URL+检索时间 |
| `write_file` | builtin-workspace | shared | write | agent-write | 落盘后文件含 AfterSummary；快照可回滚；expected_revision CAS 可选（dsh-ops 蒸馏） |
| `xlsx_apply` | extra-app | work | write | office-journal | 公式重算零错误+引用级声明比对（P1a 升 agent 工具） |

## P3 四类验收谓词（非文档工具）

每类至少 1 金标例 + 1 已知缺陷例，harness 见 `builtin/p3_predicates_test.go`（缺类=守卫红）。

| 类 | 统一验收口径 | 代表工具 |
|---|---|---|---|
| deliver-write（交付写） | 声明路径=实际落点；落账可查（Journal/库）；越界与非法输入正确失败 | office_generate / docx_apply / pptx_apply / xlsx_apply / schedule_apply / write_file |
| domain-compute（领域计算） | 计算结果可复算（重算/对照公式/库快照一致）；约束违反 fail-closed | schedule_analyze / cost_indicators / cost_compose |
| read-retrieval（读取检索） | 命中内容与库/文件实况一致；参数闸 fail-closed（检索为模糊匹配语义，诚实缺省=不冒充精确率） | grep / read_file / knowledge_search / memory_search / cost_search |
| task-orchestrate（任务编排） | 编排结构与约束闸一致（节点数/写路径/只读作用域六闸刀）；进度落盘可恢复 | dag_plan / task / todo_write |

适用条件：办公路径继续走 `boot.Build` 唯一注册表（四外部入口：builtin / ExtraTools / 插件 / Build 内部）；若出现第五装配面（如按 `boot.go` 注释绕过 Build 的自定义装配器）则本清单失效，需重新盘点。
