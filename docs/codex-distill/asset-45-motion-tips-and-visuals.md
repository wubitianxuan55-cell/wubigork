# 资产 45 · 动效与视觉件：降级成静态符号、空态白名单、内联可视化的沙箱前置

> 证据根（只读）：`motion.rs`（152）、`system_motion.rs`（83）、`shimmer.rs`（80）、`summary_shimmer.rs`、
> `empty_state_animation/{policy,renderer,geometry,paths,lighting,sequence}.rs`、
> `tooltips.rs`（681）、`transcript_image.rs`（169）、`inline_visualization.rs`（314）、`turn_tip.rs`（33）、
> `pets/*`（10 文件 ~130KB，彩蛋层）。
> 对照根（只读）：`frontend/src/gaea/lib/gsapAnimations.ts`、`gaea/lib/useGSAPCollapse.ts`、`components/chat/{WelcomeScreen,MessageList}.tsx`、
> `frontend/src/genui/*`（回答即 UI）。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、降级不是"关掉"，是**换成静态符号**

`motion.rs:18-38`：`MotionMode{Animated, Reduced}` + `ReducedMotionIndicator{Hidden, StaticBullet}`。

`loading_glyph_with_delay`（`:51-70`）：

```
Reduced            -> 直接返回静态 "◌"
Animated + delay   -> 返回 ""（空串）并 schedule_frame_in(delay - elapsed)
Animated           -> 每 100ms 一帧，10 帧盲文 ["⠋"…"⠏"] 循环
```

三条可直接搬的行为：
1. **Reduced 有替代符号**（`◌`），不是隐藏——用户仍然知道"在跑"；
2. **延迟期内返回空串**并只排一帧：不提前闪一下再消失；
3. 帧率与帧表是常量（`FRAME_DURATION=100ms`、`FRAMES` 10 项），不在渲染处硬编码。

`activity_indicator(start_time, motion_mode, indicator)`（`:72-`）把"起始时间 + 动效模式 + 降级符号"三件事合成一个函数，
所有活动行共用——单一实现，没有第二个转圈。

## 二、系统偏好只在启动读一次，且**不改持久配置**

`system_motion.rs:1-30` 文件头三条纪律：

> 「启动时读一次宿主无障碍偏好；**检测永不修改已持久化的配置、也不询问 app server**；
> 偏好不可用时保持配置行为；显式要求"少动效"时压制所有受 `tui.animations` 控制的效果；**改 OS 偏好后需重启**。」

实现：`static SYSTEM_MOTION: OnceLock<MotionMode>`（`:11`），`initialize()` 探测一次失败则 `Animated`（`:13-16`），
`mode()` 只读缓存（`:18-20`）。Windows 用 `SPI_GETCLIENTAREAANIMATION`（`:34-40`），macOS 用 `accessibilityDisplayShouldReduceMotion`。

> 对 Web：等价物是 `matchMedia('(prefers-reduced-motion: reduce)')`。
> 但 codex 的三条纪律值得保留：**读一次（或订阅）**、**不改用户配置**、**不可用则回落配置**。
> gaea 现在有 `--dur-fast/base/slow` 与 `gsapAnimations.ts:5-12` 双轨常量，缺的是"系统偏好 → 该用哪一档"的单一裁决点。

## 三、空态动画有**白名单**，未知类型默认不动

`empty_state_animation/policy.rs:1-28`：

```
enum Presentation { Hidden, Animated, Faded }
fn is_startup_cell(cell) -> bool {
    SessionHeader | SessionInfo | StartupWarnings | DeprecationNotice
    | UpdateAvailable | SessionNotice | (WarningHistoryCell && server_version_notice)
}
```

注释两句就是全部判据：
- 「**启动元信息是完整白名单；新的内容类型默认不动**」；
- 「未知的 history cell 类型算 activity，**即使它不渲染任何可见文本**」。

即：动画的开门条件是**正向列举**，不是"不是消息就动"。
`Presentation` 三档（Hidden/Animated/Faded）让"有内容了但还没提交完"能淡出而不是突然消失。

## 四、内联可视化：沙箱前置 + 私用区占位 + 2MiB 上限

`inline_visualization.rs:29-41`：

| 常量 | 值 | 作用 |
|---|---|---|
| `DIRECTIVE_PREFIX` | `::codex-inline-vis{` | 模型输出里的指令前缀（后接 JSON） |
| `CONTENT_REFERENCE_PREFIX/SUFFIX` | `\u{e200}visualize\u{e202}` / `\u{e201}` | 正文里的**私用区字符**包裹，避免与普通文本撞 |
| `MAX_FRAGMENT_BYTES` | `2 * 1024 * 1024` | 单片段上限 |
| `LINK_PLACEHOLDER_PREFIX` | `https://codex.invalid/inline-visualization/` | 伪域名占位（**不指向外网**，`.invalid` 保留域） |

`from_config`（`:49-70`）先做权限判断：
**有全盘写权限 → 直接 `None`（不做内联可视化）**；否则取可写根，并检查能否写 viewer 缓存目录。
即"要落盘生成可视化"之前先问沙箱，不是先生成再报错。

`transcript_image.rs:24-41`：`FIRST_IMAGE_ID = 0xC100`、`ImagePlacement{…}`、`TranscriptImage{…}`——
终端图片用固定起始 id 与放置结构（协议相关，Web 侧不搬，但"图片有稳定 id + 位置"的思路可留）。

`tooltips.rs:16-27`：`RAW_TOOLTIPS = include_str!("../assets/tooltips.txt")`（文案外置成数据文件），
并按平台给**可执行建议**：Windows/`macOS`/Linux 各一条 `WINDOWS_APP_TOOLTIP`/`MACOS_APP_TOOLTIP`/`LINUX_APP_TOOLTIP`——
不是"提示一下"，而是"去这个地址装这个"。

## 五、gaea 现状核对

| gaea | 现状 | 差在哪 |
|---|---|---|
| `gaea/lib/gsapAnimations.ts`（CSS 变量镜像 + GSAP 常量）、`useGSAPCollapse.ts` | 动效有统一常量与折叠 hook | 无"系统 prefers-reduced-motion → 选哪一档"的裁决点；无"Reduced 时换静态符号"的约定 |
| `components/chat/WelcomeScreen.tsx` | 欢迎屏（静态） | 无空态动画/淡出三档；也不需要（Web 可用 CSS 动画，但需白名单判据） |
| `genui/*`（回答即 UI）+ `ChatMermaid` + `MermaidBlock` | 富可视化已存在（比 codex 强） | 缺"指令 → 片段 → 上限"的护栏：`::codex-inline-vis` 式的显式指令格式、2MiB 上限、伪域名占位 |
| `gaea/lib/tooltips`? | 提示以 `title`/`Tooltip` 内联为主 | 无外置文案表与平台化建议 |
| `EmptyState`（多页复用） | 空态统一组件 | 无"加载中/失败/空"三态区分（见 F38 §五同源问题） |

## 六、三档判定

**档一（照搬语义）**

1. **Reduced 换符号**：`prefers-reduced-motion` 下所有活动指示换成静态符号（`◌`/静止点），不隐藏。
2. **延迟期不显示**：任何 spinner 有 `delay`，期内不渲染（防闪）。
3. **动效白名单**：空态/装饰动画只对已知"空"类型开启，未知类型默认不动。
4. **系统偏好不改配置**：读系统偏好只决定本次会话的档位，不写回用户设置。

**档二（改造搬）**

5. **内联可视化护栏**：显式指令前缀 + 私用区占位 + 2MiB 上限 + `.invalid` 式伪域名（不外链）。
6. **文案外置**：提示语放数据文件（`tooltips.txt` 的等价物），平台化建议各自一条。
7. **三档 Presentation**（Hidden/Animated/Faded）用于"内容到达"过渡。

**档三（不搬）**

8. `pets/*`（终端彩蛋层，含 sixel/iterm 图片协议）、`transcript_image` 的图片协议细节、
9. 盲文帧表与 100ms 帧率（Web 用 CSS 动画曲线）。

## 七、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F45-A 动效裁决点** | 新 `gaea/lib/motionPolicy.ts` | `motionMode()` 读一次 `matchMedia`（可订阅）+ `indicatorFor(mode)` | 单测：reduce ⇒ Reduced；`loadingGlyph(mode)` 返回静态符号 |
| **F45-B 白名单** | `EmptyState`/欢迎屏 | `isDecorationEligible(kind)` 正向列举 | 单测：未知 kind ⇒ 不动；消息类 ⇒ 不动 |
| **F45-C GenUI 护栏** | `genui/*` 解析层 | 指令前缀 + 2MiB 上限 + 伪域名占位（禁止外链预览） | 单测：>2MiB 片段被拒并给原因；外部 URL 不被当作可视化源 |
| **F45-D 提示外置** | 新 `gaea/assets/tips.json` | 平台化建议 + 文案表 | 单测：缺 key 回落默认；无硬编码长文案 |

**开关**：A 为纯函数（先接活动指示）；B/C 需产品确认（GenUI 是 gaea 强项，改解析要谨慎）；D 只动文案来源。

## 八、风险

1. **GenUI 是 gaea 的核心能力**（回答即 UI）：F45-C 只加"上限 + 占位 + 不引外链"三条护栏，不改渲染模型。
2. **双轨常量**：`--dur-*` 与 `gsapAnimations.ts` 已并存，F45-A 的裁决结果必须同时驱动两边，否则 reduced-motion 下不一致（F33 已记）。
3. **`perf-guards`**：`matchMedia` 若在渲染期读，会触发新订阅告警；应一次性 hook + `useSyncExternalStore`。

## 九、证据自查

- 动效模式与降级：`motion.rs:18-70`；系统偏好：`system_motion.rs:1-40`。
- 空态白名单：`empty_state_animation/policy.rs:1-28`。
- 内联可视化：`inline_visualization.rs:29-70`；图片：`transcript_image.rs:24-41`。
- 提示外置与平台建议：`tooltips.rs:16-27`。
