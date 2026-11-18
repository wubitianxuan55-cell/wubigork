# gaea-board-prompts-20261119-cff32e23

**打包点**：`cff32e233e2d5f4f4b1a1f0e3a68e91c9db0b1e0`（HEAD 三连提交）
**提交**：
- `838a58cd` feat(gaea): 办公/文学系统提示词分板 v1.0（众神殿 pan_1795007052552 冻结稿）
- `f7323e13` feat(pantheon): 席位上下文账本断链根修+席位窗口三刀
- `cff32e23` docs(codex): hotspot-07 反编译管线 C 级发现档+backend 覆盖台账跟进
**版本**：4.544.0（不抬号；日常构建）

## 本包内容

- 办公身份 `DefaultSystemPrompt` → 「办公 A」五段短稿（身份/范围/任务判断/
  事实纪律/边界；不点名工具，能力=schema 单源+执行常量两行路由兜底，
  53 行工具手册淘汰）；文学身份 → 「文学 A」（覆盖各类小说）。
- 新增 `agent.WenxueSingleModelPrompt` 独立执行常量；办公执行常量增量=
  remember 确认制+能力发现两行；boot `Options.ExecBoard`/`Options.SpaceAnchor`
  板块缝；`Controller.Schemas()` 暴露面+schema 三态测试。
- 众神殿四席 usage 竖切（grok/T15 引擎事件、codex turn.completed、zcode/dsh
  原链）+席位窗口三刀（轨迹并工具/上下文 tab/观点默认折叠）。
- 运维面：装机位 `%APPDATA%\gaea\config.toml` 持久化 `system_prompt` 幽灵旧稿
  （土壤修复长稿）盖住新默认——已置空回落（备份
  `config.toml.bak-20261118-promptfix`）；旧稿点名的 spec_judge/spec_query
  已不存在于代码；task 工具 CompactDescription §7.3 委派口径无损。

## 构建与验收

- 干净工作树：`.tmp/gaea-pkg-cff32e23`（detached @ cff32e23）
- 前端：Vite 23.88s。**坑：工作树全新 `npm install`（repo 不含 package-lock.json，
  lockfile 为 untracked）解析出的依赖集确定性触发
  `[vite:esbuild-transpile] remove %TEMP%\esbuild-*: Access is denied`
  （esbuild 0.25.12 同版也中招；主树同刻重跑绿，判别实验定性为依赖集差异）——
  处置=删除工作树 node_modules，junction 主树
  `frontend/node_modules` 后构建即绿。建议后续把 package-lock.json 入库。**
- Wails：UCRT64 GCC、`CGO_ENABLED=1`、系统 TMP/TEMP，
  `wails build -s -ldflags "-s -w" -trimpath`，产物 55,198,720 B
- 正式 payload 冒烟：`/api/health` 200，冒烟 PID 51468 已回收
- 装机不重装：装机位 23:30 主树构建（791FE398）已含同源三提交内容+冒烟 200，
  本包 exe 为干净树独立产物留档

## payload

| 文件 | 字节 | SHA256 |
|---|---:|---|
| gaea.exe | 55,198,720 | 0F7102C851CF57EE6D2ED104232052CC70B56C8FCED8F077ADA8A19B42F58A93 |
| onnxruntime.dll | 19,288,888 | 68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B |
| sherpa-onnx-c-api.dll | 4,294,656 | 5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8 |
| sherpa-onnx-cxx-api.dll | 264,192 | 4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE |

三份 DLL 与既有基线哈希一致，未重编。

## 归档

- 二进制 ZIP：`gaea-board-prompts-20261119-cff32e23.zip`
  - 29,560,322 B
  - SHA256 `566451E985F8A7A2F184AEE56C464940D5EF992DCDD20A5EACE62B92E23F10D9`
- 源码 ZIP：`gaeacff32e23-source.zip`（git archive @ cff32e23）
  - 47,802,635 B
  - SHA256 `EDB743AC80824B1133C613DAA9D261E3F1FF8C638F5572452C504426CA9BDEBF`

源码 ZIP 与二进制 ZIP 留盘，不纳入源码提交；`PACKAGE-INFO.md` 与
`SHA256SUMS.txt` 入库。
