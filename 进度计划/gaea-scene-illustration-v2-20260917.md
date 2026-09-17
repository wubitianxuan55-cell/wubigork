# gaea 绘梦·阶段一刀 D 核心片：章节配图管线 v2（角色参考图 + 风格槽）规格书

> 2026-09-17 立项。来源：用户指令「继续」。规格 docs/gaea-dream-studio-nextgen-
> 2026-09.md §4 阶段一刀 D：「GenerateSceneIllustration / GaeaGenerateBookCover
> 支持传角色参考图与风格槽」。本刀取核心片=GenerateSceneIllustration 参考图+
> 风格槽贯通（书封参考/素材库变体/灯箱共用留观察池后续片）。
> 目标版本：v4.328.0。

## 1. 论点

现状场景插图=纯文生图：prompt 拼世界观/场景/角色**文字描述**，风格句写死
（「数字油画，电影级光影，高细节」），角色一致性只靠外貌文字——刀 B 建成的
参考槽（RefImages/RefMethod）没有接到配图链；风格也不可换。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | **签名扩展**（v4.139 Save 先例）：`GenerateSceneIllustration(chapterNum int, optsJSON string)`；optsJSON 空串=零行为变化；绑定面 705 不变 |
| Q2 | opts 形状 `{"characterIds":["id"],"style":"水墨国风"}`；坏 JSON 报错 |
| Q3 | 参考图来源=项目角色 `PortraitURL`（data URL 直用；本地路径读盘转 data URL）；无 PortraitURL 的角色诚实跳过并在返回 `refNote` 说明 |
| Q4 | **参考路由按后端能力**（刀 B 口径）：comfyui/herdsman 附 RefImages（img2img，denoise=SceneRefDenoise 0.6——结构引导优先兼顾场景重排）；其它后端不附参考并在返回 `refNote` 提示「当前引擎不支持参考图，本次纯文生图」——诚实降级提示不静默 |
| Q5 | 风格槽：opts.style 非空时替换风格句（构图「16:9」固定保留）；空=默认风格句 |
| Q6 | 返回 map 增量键：`refNote`（参考使用情况说明，恒回便于前端提示）；其余键零变化 |

## 3. 落地

- **internal/chapter**：`Agent.GenerateSceneIllustration` +两参（refs []string、
  style string）——refs 非空时请求带 RefImages+Mode="img2img"+Denoise；风格句
  可替换。
- **internal/app chapter_handler.go**：签名扩展+opts 解析+角色 PortraitURL
  收集（readImageFileAsDataURL 小助手）+后端能力路由+refNote。
- **前端 ChapterIllustration.tsx**：生成区上方加「风格」输入（placeholder 默认
  风格）+ 角色勾选（GetCharacters 拉名单，勾选传 characterIds；无角色/未勾=
  旧行为）；返回 refNote 展示为浅色说明行；bridge/mock 同步双参。
- 测试：Go（风格注入/refs 附着与 refNote 路由三态/坏 opts/空 opts 零变化/
  PortraitURL 路径转 dataURL）+前端（opts 序列化传递/refNote 展示）。

## 4. 观察池（本刀不做）

GaeaGenerateBookCover 参考图与风格槽；素材库变体/替换/设为书封；小说侧与
绘梦页共用灯箱；denoise 用户可调；多参考图权重。

## 5. 门禁

go build/vet + 触面包测试、tsc -b、eslint、vitest 全量、ci.ps1、drift OK@705
（签名扩展不增名）、版本三处 4.328.0。
