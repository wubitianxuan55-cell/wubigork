# 任务进度

> 最后更新: 2026-08-15（gaea 3.0 执行 · v2.40.0 发布完成 · UI 设计系统 Wave 1-2 落地中）

## UI 设计系统（3.0 补充环节，ui-ux-pro-max skill 驱动，2026-08-15）

> 需求：3.0 还差「对所有板块的 UI 进行设计」——已确认：用 ui-ux-pro-max skill、重新定义 3.0 设计语言、设计文档 + 全部板块落地实施。

| 状态 | 任务 | 提交 |
|------|------|------|
| ✅ | 设计系统定稿（docs/2026-08-15-gaea3-ui-design-system.md「智能玻璃」+ design-system/gaea/MASTER.md 定制） | 0fe3e49 |
| ✅ | 10 板块 UI 蓝图（design-system/gaea/pages/*.md，8 文件覆盖 11 板块） | d33e569 |
| ✅ | Wave 1 令牌层（colorDestructive 12 主题 + antd colorError 对齐 + .gp-panel 工具类 + appStore.test 3 用例） | ab08d2a |
| ✅ | Wave 2 死令牌治理（--ds-* 14 令牌补全 27 处消费生效 + THEME_PRESETS 单一数据源） | 9a0c374 |
| ✅ | Wave 3 壳层可访问性（theme-dot 键盘可达 + focus-visible） | fc659e4 |
| ✅ | Wave 3 modelcenter --mc-* 收敛（125 处消费别名到全局令牌，danger→destructive） | 74c20c9 |
| ✅ | Wave 3 chat 设计语言落地（侧栏激活态/气泡语义色/Composer 玻璃化/硬编码色治理 9 文件） | 1666d63 |
| ✅ | Wave 4 emoji 治理：壳层 StatusBar（antd 图标）/轻语记忆库（+测试同步）/WhisperMemoryModal/settings 4 面板/novel CommandBar+CharacterPage | e2715d5 / ca0cbff / a72f5e0 / 7c2b780 |
| ✅ | Wave 4 硬编码色令牌化：novel 状态色/theme.ts 状态角色色/character-page 警告态/imagegen 失败警告色 | 4e82064 / 760de5d / 9c1c5a7 |
| ✅ | 最终集成门禁（tsc/eslint/vitest 全量 + vite build） | 验证中 |

关键事实（盘点报告 docs/gaea3-review/10-ui-audit.md）：4 种视觉方言并行（M3 玻璃霓虹 / 办公扁平 Tailwind / modelcenter --mc-* 私有 / 记忆中枢科幻风）；--ds-* 仅 2/14 定义（已修）；主题色值 3 处重复（已单源化）；emoji 当图标 36 文件 177 处（已治理壳层/轻语/设置/novel 等，剩余为内容语义保留）；硬编码 hex 数百处（主要板块已令牌化）；11px 级字号 321 处（保留，未在本次范围）；antd 弹层动画因 WebView2 永久禁用（已知耦合）。
剩余（下一会话候选）：记忆中枢 home 科幻风已走令牌（确认即可）；办公系玻璃过渡（gaea styles.css 兜底已改 --ds-* 补全，blur 策略因 WebKitGTK 性能保留）；11px 字号治理；侧栏会话项键盘可达（a11y）。

## 阶段 7 收官 + 3.0 Step 0/1 完成（5 子代理并行，全部提交）

| 状态 | 任务 | 提交 |
|------|------|------|
| ✅ | Step 0 修债（office 注册 GaeaSend + MainBrainChat 测试 8/8 + 版本同步脚本） | 4dbba0c |
| ✅ | Step 1 会话事件日志（append-only 日志 + 投影 + checkpoint + 迁移 + 派生 + GaeaHistory 黄金测试逐字节） | 72fae6c |
| ✅ | T7-2 可见性收口（吞错清零 + 成本凭据 + 批量事务，41 测试） | 0a9fb6f |
| ✅ | T7-3 名实相符（PDF FlateDecode + OCR 容错 + 检索索引 + dashboard 真实，约 33 测试） | d7934eb |
| ✅ | T7-4 前端性能收尾（写路径清零 + 三态错误 + 渲染收敛，41 用例） | 5364281 |

## 发布

- **v2.37.0**（2026-08-15，统一构建发布，含三刀 + Step 0/1）：gaea-v2.37.0.exe（34.5MB）
  - SHA256=37A56F54DF653E3D9E8A5751EA282CEB34BF5BBCA2672D26439BF7BAEBA7A62B
  - 冒烟通过（/api/health 200）；详见 releases/v2.37.0.md

## 门禁快照（父代理实测）

- go build ./... 干净；go vet 干净
- 逐包测试全绿：session(67)/event/boot/config/search/memory/stats/bm25/auth/characterlib/channels/weixin
- internal/app：仅 1 个既有 flaky（TestDrainAndPersistAll_FinalRoundLands，单独 3/3 通过）；docmd 4 个 GBK 环境失败为基线
- 前端 tsc 0 / eslint 0 / vite build 通过；vitest 新增 41 全过（27 个 jsdom localStorage 环境失败为基线）

## Wave 2 完成：Step 1 接线 + Step 2 Manifest + Step 3a Image（4 子代理核验 + 父代理收尾，独立提交）

| 状态 | 任务 | 提交 |
|------|------|------|
| ✅ | Step 2 后端板块 Manifest（board 包 10 板块 + module_registry manifest 驱动 + GetBoardManifests + gen_bindings 464 方法） | f5ddf62 |
| ✅ | Step 1 app 层接线（事件日志「日志即真相」运行时闭环：Resume→Restore / Save→日志 / 模型调用前 fail-closed 检查点 / 压缩→checkpoint） | a9254c8 |
| ✅ | Step 2 前端（PageRegistry + MainLayout 附 B 12 硬编码点清单化 + ModuleLauncher 清单驱动 + events.ts 常量表 + bindingNames 464 + bridge 双向守卫） | 4b5af82 |
| ✅ | Step 2 label 单一来源对齐菜单文案（用户决策） | 2ba821b |
| ✅ | Step 3a Image Seam（图片后端注册表化：openai/comfyui 自注册 + 401 单次重试守卫 + fail-closed） | 9d9716d |

### 发布

- **v2.38.0**（2026-08-15，Wave 2 统一构建发布）：gaea-v2.38.0.exe（34.6MB）
  - SHA256=a9eeb837462109f8aa599213558ce6b3bd798eafbd25783fefa2edb6ca0b29fa
  - 冒烟通过（/api/health 200）；详见 releases/v2.38.0.md

### 门禁快照（父代理实测）

- go build ./... 干净；go vet 干净；test-all.ps1 **110/110 包**（首跑 6 包失败为并发抖动 + 状态文件残留，单独重跑全绿）
- internal/app 26.5s ok / internal/ai 5.2s ok / session/control/boot/search/memory/stats 全 ok
- 前端 tsc 0 errors / eslint 0 errors（350 存量 warnings）/ vite build 15.7s / vitest 404 过（27 个 jsdom localStorage 基线失败，与 v2.37.0 一致零回归）
- TestBindingsCompleteness PASS（464 方法 → 10 门面）；check-bindings-drift OK

## Wave 4 完成：Step 3 收官清理（2 子代理并行 + 父代理集成，独立提交）

| 状态 | 任务 | 提交 |
|------|------|------|
| ✅ | semantic_search 工具注册（决策纳入：实现完整 + E2E 测试，死代码恢复为办公 agent 可用工具；gaeaSpecialistTools 集中注册 ocr + semantic_search） | 89d9fae |
| ✅ | BalanceKind 从 ProviderEntry 贯通（config `balance_kind` → boot → control.Options → controller.Balance 改走 FetchByKind，空=deepseek 默认，未知 kind fail-closed；config/control/boot 三层 7 测试 + render.go 渲染） | 89d9fae |
| ✅ | ModuleLauncher 清单化（launcher.ts 纯函数 deriveLauncherModules + LAUNCHER_DESC；useSyncExternalStore 订阅活动清单，后端并入 knowledge 后首页启动器自动多出「知识库」卡；launcher.test.ts 7 用例） | 4a6033a |

### 发布

- **v2.40.0**（2026-08-15，Wave 4 统一构建发布）：gaea-v2.40.0.exe（33.1MB）
  - SHA256=f7c5fd1bc1859b025a742dcb78b26065a5718d8aa4374ef5c8cd90d7aaaff317
  - 冒烟通过（/api/health 200）；详见 releases/v2.40.0.md

### 门禁快照（父代理实测）

- go build/vet 干净；**test-all.ps1 110/110 包**
- 前端 tsc 0 errors / eslint 0 errors（350 存量 warnings）/ vite build 17s / vitest 427 过（27 个 jsdom localStorage 基线失败，与 v2.39.0 一致零回归）
- TestBindingsCompleteness PASS（464 方法，无新绑定）；check-bindings-drift OK

## Wave 3 完成：Step 3b/c/d Provider Seam + 前端接线（4 子代理实现 + 父代理集成，独立提交）

| 状态 | 任务 | 提交 |
|------|------|------|
| ✅ | Step 3b LLM Seam（LLMProvider{Provider;Chat} + ChatFromStream + NewLLM 配置驱动 + 消费者切接口，19 测试） | d183af7 |
| ✅ | Step 3c OCR/ASR/TTS Seam（OCRProvider ovis/tesseract + TTSProvider edge/sapi/herdsman/xai + ASRProvider herdsman 接口注入，GAEA_OCR_ENGINE 驱动） | 078ce1d |
| ✅ | Step 3d 分类单源化（ClassifyModelKind/ClassifyModelByName）+ 8 处注册表化（websearch/embed/rerank/vision/image/ocr/markitdown/billing） | 9a535c4 |
| ✅ | 前端 GetBoardManifests 接线（loadBoardManifests 远端 + normalize 差集 + KnowledgePage 注册，45/45 测试） | b1cc2fd |
| ✅ | 父集成（gaea.toml [retrieval]/[vision]/[markdown_converter] + [search] engine_order + boot 装配 + app 层 NewLLM 迁移） | 048768c |

### 发布

- **v2.39.0**（2026-08-15，Wave 3 统一构建发布）：gaea-v2.39.0.exe（34.7MB）
  - SHA256=fac19c50accc8310210bd768be51f1f519ba28cbce80d2a32a96ada271fb31fb
  - 冒烟通过（/api/health 200）；详见 releases/v2.39.0.md

### 门禁快照（父代理实测）

- go build ./... 干净；go vet 干净；**test-all.ps1 110/110 包**
- 前端 tsc 0 errors / eslint 0 errors / vite build 42.9s / vitest 420 过（27 个 jsdom localStorage 基线失败零回归）
- TestBindingsCompleteness PASS（464 方法，无新绑定）；check-bindings-drift OK

## 遗留（下一会话）

- **Step 0-3 全部落地完成**：3.0.0 发布条件（§3.0.0）仅剩统一回归与版本发布；下一步候选 = 3.0.0 正式发布（chat 可恢复可回放为里程碑判据，Step 1 运行时闭环已在 v2.38.0 接线）
- pickHerdsmanModel 能力标签挑选保留（非分类重复，无动作）
- 回退保障硬要求不变（每 Step 独立提交 / 旧数据只读兼容 / 二进制保留 5 版 / 运行时开关 / 回退演练）
