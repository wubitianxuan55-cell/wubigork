# 第十二轮修复记录（P1 批次十：机械收敛·并行子代理波次）· 2026-10-02

> **承接**：批次一~九（b6f092f7 → 4534bc11）。
> **范式**：5 个并行子代理（G1 project / G2 app / G3 docmd+whisper / G4 FE lib / G5 FE 组件），按「同 Go 包不并行、文件互斥」切口；主代理收线缝合+裁决+全量验证。
> **验证**：`go build`/`go vet` 0；六触达包 `go test -count=1` 全绿（app 全包 105s）；`tsc -b` exit 0、eslint（16 触达文件）0、vitest 触达面 99 文件 959 测试全绿。净 −129 行（36 改 + 5 新 + 1 删）。

## 已落地（7 条 P1）

### IN1-02（G1）· project/narrative 原子写收敛
删 project.writeFileAtomic（fsync 版）与 narrative 同名实现，6 调用点归一 `fileutil.AtomicWrite`（perm 对齐各旧实际位 0o600/0o644）。**审计证据纠错**：AtomicWrite 现场已含 MkdirAll+RenameWithRetry，fileutil 本体零改动。**fsync 差值裁决**：如实收敛（两旧实现本就非严格持久化保证——单文件 sync 无 dir sync / best-effort 忽略错误），全仓统一 fsync 属独立决策不夹带。行为变化 1 处有测试锁：AtomicWrite 自动创建缺失父目录（`TestWriteFileAtomicCreatesMissingParent` 改写 + fileutil 新增多层父目录用例）。**新发现 wart（仅上报）**：AtomicWrite 的 perm 参数收下但从未生效（无 chmod，实际恒 0600）。

### AP5-02（G2）· app 包原子写 5 份收敛
gaea_skill_distill / gaea_prompt_store / gaea_route_suggestions / gaea_skill_stats / gaea_task_inbox 五处同构样板（约 −120 行）归一 fileutil.AtomicWrite。非同构站点（os.Remove 旧文件版、scratch 临时文件版、epub 固定前缀版共 8 处）**保留未收**并留清单——它们不是本模式的副本，强收是无收益抽象。

### AP1-12（G2）· 摘要回退链三处收敛
create_chapter_handler 内联闭包删除、buildStageDigest 取文改经 `prevSummaryResolver(pm)`，三处归一单一来源（novel_rewrite_handler 的 resolver 实现零改动，注释升格单源）。

### IN2-03（G2）· 图片后端解析 5 遍 → resolveImageBackend
新 `internal/app/image_backend_resolve.go`：结构化不可用原因（engine-disabled/glm-key-missing/comfyui-url-missing）+ 规范 kind，5 调用点（initImageBackend/SetImageBackend/buildImageClientFor/restoreImageBackend/GetComfyUILoras）各留站点语义（原文案报错/静默跳过/glm→xai 回退），7 子测纯函数覆盖。**未走 ai 注册表**：注册表工厂严格校验会让「引擎启用但 BaseURL 空」从构造成功变报错——零行为变化优先，注释在案。**主代理裁决（三处并集语义行为变化，全保留）**：① restore 的 comfyui→herdsman 嵌套覆盖删除（真漂移 bug：开项目静默改绑共享 client，其余四副本均无此行为）；② restore 补齐 herdsman/glm 显式重绑（幂等，+1 条 info 日志）；③ SetImageBackend comfyui 空地址从「构造死后端生成时才爆」改 fail-fast 报「未配置 ComfyUI 地址」（与其余四副本口径一致，诚实方向）。

### IN3-10（G3）· pdf.go stream 扫描两份 → streamScanner 迭代器
decode/strip 两份同构扫描收敛为 streamSpan 迭代器（isWS/trim 参数化）。**差分测试逮到两处审计未载的行为差异并原样保留**：空白类（decode 含 `\f`，strip 不含）、伪命中关键字（strip 连前导文本丢弃，decode 原样写出）——旧实现对拍 2 万条 fuzz 输入逐字节一致后才收，13 子测钉边界。

### IN4-03（G3）· whisper 情绪标签七张散表 → emotionLabelTable 单表
9 标签 × 7 取值面（zhName/innerFeeling/tendency/maxLength/prohibitions/openers/imperfection）单表化，psyche.go 的 labelZH 一并收编。**golden 矩阵钉法**：收敛前 9 面×10 输入 dump 到仓外快照，改后 diff 逐字节一致（100 行），随后硬编码为正式测试（7 函数×9 标签+未知标签默认值）。**审计纠错**：「json 词池」不存在，实际散表 7 张（中文名表在 psyche.go）。

### FE4-01 + FE4-03（G4）· FE lib 收敛
`lcsDiff<T>` 泛型收编四份 LCS（diff 行/docx 段/pptx 页签名/diffRender 字符），回溯取向原样保留钉输出确定性，新增 diff.test.ts 7 条+既有渲染测试群全绿。相对时间两套收敛到 `time.ts` 一份，文案走 i18n（新键 4 枚 × en/zh/zh-TW 三份字典，tsc 键集校验即锁），删 `src/utils/time.ts`，ProjectCardItem 迁移（新旧行为对照：2-29 天从「N 天/周前」变 M-D、日历日差 1 变「昨天」、en 环境不再出中文——审计认可口径）。审计纠错：relativeTime 消费方多一处（App.tsx）。

### FE6-06 + FE7-04（G5）· FE 组件复用收敛
CharacterLibEditor + SinCastPanel 两处手写 ComfyUI 轮询迁 `useComfyTaskProgress`（能力差双向核对：hook 为超集，document.hidden 跳帧/失败保帧增益随迁；SinCastPanel 的 v4.427「排队不轮询」语义精确保留，取消链路核对解耦）。TaskInspector 删本地日期换算四函数改 `calendar.ts`（两处语义差异——脏日历清洗、扫描越界返回第 N 个工作日——统一到 Go 镜像口径）。新增 4 用例。

## 跨波次缝合（主代理收线）
1. **G1×G2 交互**：project.WriteChapter 获得 MkdirAll 增益后，app 侧 `TestWriteBackRewritten_WriteFailureReturnsError` 的写失败注入失效——G2 用 HEAD 临时 worktree 实证归因后把注入改为「chapters 位置占普通文件」（对两代实现都成立）。
2. **陈旧注释三处**（G1 删函数后遗留引用）：create_chapter_handler.go:404、plot_branch_handler.go:318、plan_store.go:19-20（含已不成立的 fsync 表述）——收线时统一修正。

## 本批结论与余量
- 并行子代理波次实证：五代理 ~43 分钟全绿收线，切口纪律（同包不并行/文件互斥/禁 git 写操作）零冲突；两个跨波次交互都被代理按规程上报或自证归因。
- 审计证据纠错 4 处（AtomicWrite 能力、json 词池不存在、image_handler:1020 无 switch、relativeTime 消费方多一处）——审计行号/证据仍需现场核实的老口径成立。
- 上报未动：AtomicWrite perm 摆设参数（fileutil）、全仓 fsync 决策、app 包 8 处非同构 CreateTemp、scene/style/export 裸 os.WriteFile（IN1-02 相邻问题清单）。
- 留池（需拍板）不变：AP1-10、AP5-01、AP6-05、AP2-03、FE2-05/06/08、FE7-11、AP4-01/AP4-04、#25。
