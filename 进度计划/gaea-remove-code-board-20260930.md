# 删除编程板块（code / DeepSeek Harness 工作台）· 2026-09-30

> 用户指令（原文）：「删除编程板块」。整块退役：前端 manifest/页面/入口 + Go 侧
> DSH Web 进程管理五绑定 + 两侧 mock/类型/测试/文档同步；零替代品、零保留兜底页。

## 1. 删除范围（足迹清单）

| 层 | 文件 | 处理 |
|---|---|---|
| 后端·功能 | `internal/app/programming_web.go`（状态/前置检查/启停/日志尾，8.9KB） | 删除 |
| 后端·测试 | `internal/app/programming_web_test.go`（16 例） | 删除 |
| 后端·清单 | `internal/app/board/builtins.go`（`code` 条目 + `CanonicalIDs`） | 移除 code；canonical 13→12 |
| 后端·测试 | `internal/app/board/manifest_test.go`（space 表）、`internal/app/board_manifest_test.go`（14→13 + code 断言反转） | 更新 |
| 绑定门面 | `internal/app/bindings_core.go`（5 个委托方法；生成物） | **手删 5 方法**（整表再生会带来全库格式化噪声，见 §3 坑①） |
| 绑定清单 | `frontend/src/gaea/lib/bindingNames.ts`（生成物） | 再生：744 → 739 |
| 绑定类型 | `frontend/src/gaea/lib/bridge/core.ts`（5 声明 + 类型 import）、`frontend/wailsjs/go/app/CoreB.{js,d.ts}`（生成物 5 包装） | 移除 |
| 类型域 | `frontend/src/gaea/lib/types/programming.ts` + `types.ts` re-export | 删除 |
| 空间分面 | `frontend/src/gaea/lib/spaceBindings.ts`（independent 5 条）+ 测试数量锁 | 567 → 562，independent 归零（机制保留） |
| 前端页面 | `frontend/src/pages/ProgrammingPage.tsx`（538 行）+ `.test.tsx`（12 例）+ `programming-page.css`（17KB） | 删除 |
| 前端注册 | `frontend/src/main.tsx`（`registerPage('ProgrammingPage')`） | 移除 |
| 前端清单 | `frontend/src/boards/manifests.ts`（code 条目 + 图标注册表 CodeOutlined）+ `launcher.ts`（LAUNCHER_DESC.code） | 移除 |
| 前端测试 | `boards/manifests.test.ts` / `boards/launcher.test.ts` / `boards/space.test.ts` / `layouts/CommandRail.test.tsx` / `gaea/lib/spaceBindings.test.ts` | 夹具/计数/断言同步 |
| 前端壳层 | `MainLayout.tsx`（`v3-prog-host` 工具栏宿主）+ `v3/foundation.css`（`.v3-strip-prog`） | 删除（宿主无消费方） |
| mock | `frontend/src/gaea/lib/mock/core.ts`（5 桩 + 联合类型） | 移除 |
| 文档 | `design-system`（无编程页）、`.gaea/AGENTS.md`（工位构成句）、`docs/*`（历史调研/架构档） | 仅 AGENTS 现状句更新；历史档保留原文 |

**保留项（刻意不动）**：`independent` 独立窗口**机制**（`BoardSpace='independent'` /
`isIndependentBoard` / rail foot 分栏 / spaceBindings 分面）——它是通用能力，删板块不等于删
机制；删除后该分面为空集，rail foot 段自动隐藏（`.v3-rail-divider` 不渲染）。

## 2. 数据面变化

| 计数 | 前 | 后 | 说明 |
|---|---|---|---|
| Go 绑定面（bindingNames） | 744 | **739** | `-5`：GetProgrammingWebStatus / Start / Stop / GetProgrammingWebPreflight / ProgrammingWebLogTail |
| spaceBindings 分面表 | 567 | **562** | 同上 `-5`（independent 分面归零） |
| canonical 板块（含 knowledge） | 14 | **13** | `internal/app/board` + 前端静态清单一致 |
| 菜单项 | 12 | **11** | 编程原为 `menuOrder 6 / space independent` |
| 首页启动器卡 | 12 | **11** | 编程原不在双空间首页（independent），此计数为全量口径 |

## 3. 坑与教训

① **生成物再生粒度**：`go run ./scripts/gen_bindings` 全量再生会把 11 个门面文件从
多行体压成单行体（生成器已迭代、入库件未同步）——一旦全跑，diff 里混入 ~1000 行格式化
噪声。处置：先 `git restore` 全部再生文件，再**手删** `bindings_core.go` 的 5 个委托方法
（保持与入库件同风格），只再生 `bindingNames.ts`（名字清单，格式稳定）。漂移闸按**方法名
集合**比对，与格式无关，故手删不破坏该闸。
② **测试夹具是删除的第二战场**：板块删除在测试里表现为「夹具表 -1 / 计数 -1 / 断言反转」，
共 6 个文件（Go 2 + 前端 4）。漏一处在 CI 里就是红——本轮顺序=先改实现 → 跑定向 → 按失败
清单逐个改夹具（避免盲改）。
③ **契约 testid 与泛机制要分清**：`independent` 机制在 rail / spaceBindings / manifest
校验三处都有落点，全部保留；只有「编程」这一**实例**被移除。

## 4. 验收

| 判据 | 证据 |
|---|---|
| Go 侧 | `go build ./...` 0 错、`go vet ./...` 0 警；`go test ./internal/app/... ./internal/gaea/... -count=1` 全绿（app 122s 包级绿） |
| 绑定漂移 | `scripts/check-bindings-drift.ps1` → `OK：bindingNames.ts 与 Go 绑定面一致（739 个方法）` |
| 前端静态 | `tsc -b` 0 错；`eslint .` 0 error（2 条既有 warning 在无关文件） |
| 前端测试 | `pnpm vitest run` **410 文件 / 3564 例全绿**（首轮 2 例 PptxEditPanel 负载 flake，隔离复跑绿=非回归） |
| 守卫 | `frontend-e-check.mjs` OK、`check-docs.mjs` OK |
| 真机 DOM 走查 | 无头 Edge CDP：`body.innerText` 不含「编程」；rail 菜单 = 首页/办公/造价数据库/记忆中枢/模型中心/青鸟 + 深浅色切换；`.v3-rail-divider` 不存在；首页「文书台」正常渲染 |

## 5. 后续

- 桌面版 exe 需重建才会同步（当前运行的 4.438.0 内嵌旧资产，仍含编程入口）。
- `docs/2026-08-15-gaea3-architecture-design.md` §9 等历史档按「历史不改写」纪律保留，
  引用时以本档为准。
