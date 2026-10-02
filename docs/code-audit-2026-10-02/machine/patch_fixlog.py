import io

p = r'C:\AI\wubigrok\docs\code-audit-2026-10-02\round-1-fixes.md'
s = io.open(p, encoding='utf-8', newline='').read()

dao1 = '''### 刀1 · 配置保存丢段（审计 P0-1 / P0-2）

- **根因**：`RenderTOML` 是**有损归一化渲染**（只覆盖渲染器建模的段与键）；`Config.SaveTo` 直接用它整文件重写 → 一次「始终允许」审批（`boot.go:623-632` 的 `PersistAllowRule`）**或任何一次设置面板保存**（`app/gaea_ui_extra.go:147 gaeaApplyCfg → gaeaConfig.Save`）即把未建模的顶层段与被漏渲染的段内键从磁盘静默删除。
- **改动**：新增 `internal/gaea/config/render_preserve.go`（`RenderTOMLPreserving(c, existing)`，三层保留：根标量**前置** / 渲染器负责段做**键级合并** / 未知顶层段逐字追加）；`edit.go` 的 `SaveTo` 与 `config.go` 的 `WriteFile` 改走它（签名、原子写、权限位不变）；`render.go` 函数体零改动，仅注释标注「落盘请用 Preserving」。`rendererOwnedTopLevel = agent/providers/tools/permissions/space_profiles/sandbox/plugins`。
- **实测踩到的坑（已修）**：① 根标量按原位置写会被 BurntSushi 吞进上一张表（`[sandbox]` 之后的 `workspace = …` 读回空串）→ 根标量必须前置到第一张表之前；② `Load()` 对 `Providers` 是**整片替换**，故 `[[providers]]` 必须**按位次**配对（按 name 配对会把用户的 thinking/effort 补到别的条目）；③ 遗留 provider 条目必须连表头一起保留，否则掉进上一张表；④ `sandbox`/`plugins` 也在渲染器职责内（原清单漏列），当未知段整块保留会写出**重复表头 = 非法 TOML**。
- **验收（三重独立）**：
  - 线1 红证（临时探针 `GAEA_ZZ_LOSSY=1` 退化为旧路径）：`16 段丢 11 段`、`workspace 读回空串`、`known 段内 7 个键全丢`；绿证 4 条测试全 PASS（含钱测试「13 段逐字段零漂移」、真实配置形状、我追加的键级测试、连存 3 次收敛）。
  - 主代理端到端探针（`.tmp/configprobe`，生产同源链 `Load → AddPermissionRuleForSpace → Save`，**用真实配置的副本**、`APPDATA` 指向临时目录）：**修复前 16 段 → 5 段（丢 11）、`Session.Space` work→空**；**修复后 16 → 16、字段零漂移**。
  - 主代理注入非空值的合成夹具（`effort="high"`、`subagent_effort="low"`、`approval_timeout_secs=42`、`compact=true`、provider `thinking="adaptive"`/`effort="max"`）：**8 个字段往返全部保真、0 漂移**。
- **未覆盖**：`[space_profiles.X.permissions/.guardrails]` 与 `[[plugins]]` 的键级合并无专项用例（代码路径支持、既有测试仍绿）；用户注释**内容逐字保留但位置会挪到文件末尾保留区**；`internal/config` 经核实**不是**同类有损（read-modify-write + 全字段 marshal），未动。
- **用户数据保护**：原件一字未动；同目录留下 `config.toml.bak-20261002-audit-pre-RenderToML-fix`（5995 B）。

'''

dao7 = '''
### 刀7 · `bookimport` 的第二份 JSON 提取实现（P0-22 同类）

- **问题**：`internal/bookimport/reconstruct.go` 自带一份 `strings.IndexAny(s,"{[")` + `LastIndexByte(s, closeCh)` 的提取器，与 `util.ExtractJSON` 同属一个 bug 类（线3 越界上报）。
- **改动**：改为 `util.ExtractJSON` 的**薄包装**（无 import 环：util 只依赖 stdlib），逐字保留本包原契约「找不到即报错、标量/null 不算结构」；顺手删掉只为旧实现服务的私有 `stripCodeFence`。调用点（`CallJSON` → `app/novel_import_ai.go:354/379`）零改动。
- **红 → 绿**：红 4 个子用例（两对象拼接 / 正文+对象+后文 `{}` / 字符串内花括号 / 嵌套后噪声）→ 绿全包 ok；两个「护栏」用例（对象数组先取、裸围栏容错）改前改后均 PASS（证明没丢原容错面）；端到端 `TestNovelOutlineReconstruct` ok。
- **残留**：第三份同类实现 `scripts/test_herdsman_models.go:173`（脚本件）仍在；未处理「正文先出现合法 JSON 噪声」；`util` 的「对象优先于纯标量数组」在 `[1,2] {"a":1}` 这类回复上与旧实现结果不同（`ExpectArray` 下旧成功 / 新重试）。
'''

sec2_old_start = '## 二、进行中 / 待收口'
sec2_new = '''## 二、25 条 P0 的当前状态（只读复核）

复核线（HEAD `3d0210d1`，只读、不跑测试）逐条读**当前**代码后产出 `round-1-p0-status.md`；主代理另亲核撤销/更正了其中 3 条判据。

| 口径 | 数量 |
|---|---|
| 表内行数 | 25 行（其中 #19/#20/#21 与 #1/#2/#8 是同一条） |
| **独立 P0** | **22 条** |
| 本轮已修 | 7 条独立（#1/#2 配置、#3 收件箱、#15/#16 进度契约、#18 GenUI、#22 ExtractJSON）＋ 非编号内的 bookimport 副本 |
| 部分修 | 1 条（#5 逐场景协程：登记已补、WG 仍缺） |
| 撤销 | 1 条（#17 前端全选：`setAllSelected`/`clearSelection` 定义在同文件 1126-1131，声明提升，原判据不成立；降级 P2「缺表头全选用例」） |
| 更正 | 方向 1 条（#21 真实方向是「显式 0 → 缺键」）；计数 1 条（#22 生产调用点 **21** 处，非 25） |
| **仍开放** | **10 条独立**：#4/#5 后台链登记纪律 · #6/#7 检索与知识库吞错/全局索引 · #8 崩溃面 · #9/#10/#11 三套编排与并发闸 · #12/#13 微信语音并发 · #14 绘梦单槽进度 · #23/#24/#25 三类复发型缺陷无守卫 |

**复核线给出的两条「差一行就闭环」高性价比项**：

1. #5 `internal/app/scene_cards_handler.go:218` 前补 `a.chapterGenWG.Add(1)` + 协程首行 `defer a.chapterGenWG.Done()`；
2. #4 `internal/app/converge_handler.go:177` 改走既有 `registerChapterGen`（并入取消登记表 + WG）。

> 完整逐条证据（当前行号 + 修复后新位置）见 [round-1-p0-status.md](round-1-p0-status.md)。
'''

n = 0
if '## 一、已交付（5 刀）' in s:
    s = s.replace('## 一、已交付（5 刀）', '## 一、已交付（7 刀）', 1)
    n += 1
if '### 刀2 · 任务收件箱' in s:
    s = s.replace('### 刀2 · 任务收件箱', dao1 + '### 刀2 · 任务收件箱', 1)
    n += 1
if '\n---\n\n## 二、进行中 / 待收口' in s:
    s = s.replace('\n---\n\n## 二、进行中 / 待收口', dao7 + '\n---\n\n' + sec2_new, 1)
    n += 1
else:
    print('MISS section2 anchor')

io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('patched', n)
