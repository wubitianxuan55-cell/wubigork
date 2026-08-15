# 任务进度

> 最后更新: 2026-08-15（gaea 3.0 执行 · v2.39.0 发布完成）

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

- Step 3 收官清理：semantic_search 工具定义未注册（#6 只含 OCR，是否启用待决策）；billing kind 未从 ProviderEntry 贯通（controller 默认 deepseek）；ModuleLauncher 仍用静态 canonicalBoards
- 回退保障硬要求不变（每 Step 独立提交 / 旧数据只读兼容 / 二进制保留 5 版 / 运行时开关 / 回退演练）
