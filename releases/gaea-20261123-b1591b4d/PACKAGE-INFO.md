# gaea-20261123-b1591b4d（树上在途件全收口包）

- 交付提交：`b1591b4d`（chore）——本包覆盖当天四笔提交：
  `5245fdec` feat 方案档参照知识库接线 → `b4b189a9` docs 包记录 →
  `28933717` feat 文学图文私密保护四形态+novel-chapter-branches 技能种子 →
  `9168f07a` fix 编程板块 web 遗孤回收根修 → `b1591b4d` chore 走查双栈+缓存探针
- 分支：`main`（ahead origin 39，未 push）
- 产品版本：`4.544.0`（日常预览包，不抬版本）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-b1591b4d`，detached HEAD
  （干净树；主树仅余运行态 probe-memory-out.jsonl 与 releases 大件，零混入）
- 前端：pnpm `11.21.0` install `24.7s` + `pnpm run build` Vite `25.23s`
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc、PATH 前置
  `build/bin`；`wails build -s -ldflags "-s -w" -trimpath`，bindings 未跳过，
  Wails `55.574s`；exe 61,733,888 B
- 漂移：Wails 再生成 wailsjs 26 文件+go.mod，`--ignore-cr-at-eol` 内容差 `0`
  （纯 CRLF 噪音）
- 冒烟：正式 payload `/api/health` **200**（HTTP 桥 18999 避让活壳），
  PID `35160` 已回收
- 三 DLL 与既有基线一致（`68D0D5F5…` / `408F9AB6…` / `DBAC55CE…`）

## 提交前验证（树上在途件全量）

- 归属核验：app.go/gaea_ui_extra/gaea_unified_search/jailbreak/main.go 全部
  属文学私密线（私密会话态/图文剥离/9333 取证注），无第四条线
- Go：vet（app/config/maturecraft）净；定向 `Privacy|Orphan` 绿；
  `WenxueSkillSeed|WenxueSkills|WenxueWorkspace|NovelChapterBranch` 绿；
  config+maturecraft 全量绿
- 前端：`tsc -b` 0；私密组件定向 `PrivacyMask+PrivacySidebarSection` 8/8

## 各提交要点

- `28933717`（49 文件 +2678/−111）：图文私密保护四形态（PIN 锁/隐藏恒滤/
  失焦遮罩/搜索日志隔离）——wenxue_privacy 四态后端（重启回锁、内存不持久化）
  +PrivacyMask/PrivacySidebarSection/useWenxuePrivacy 前端+unified_search
  图文剥离+config 私密键+9333 CDP 恒开取证注（env 反制无效结论）；
  novel-chapter-branches 技能种子双落（.gaea/skills+skill_seed 目录）；
  wenxue 绑定面再生（WenxueB/models/gaea lib 三件套）
- `9168f07a`：编程板块外部浏览器遗孤回收——netstat 探 3091 监听者 PID+
  等端口释放（100ms 步进带超时）+树杀；三探针可替换缝单测钉死
- `b1591b4d`：cdp-walk 双栈试连（WebView2 有时绑 [::1]:9333）+
  GAEA_WALK_HOST 钉壳防驱动打错实例；cache_probe_live.go（build ignore）
  DeepSeek 缓存命中率实测探针随批入库
- `5245fdec`/`b4b189a9`：方案档参照知识库接线+其包记录（详录
  releases/gaea-proposal-kbref-20261123-5245fdec/PACKAGE-INFO.md）

## 已知边界

- 未装机、未 push；安装按 `scripts\gaea_install.ps1`（会杀现有壳——
  当前 build\bin 旧壳 PID 62656 仍在跑，装机前先关）
- CDP 实机走查被活壳挡（9333 占用），壳关后补

## SHA256

| 文件 | SHA256 |
| --- | --- |
| payload/gaea.exe | `1413DA5DF8CF6147CDF24D78AC3D72269106DEE3B655B457197EBEC29942646A` |
| payload/onnxruntime.dll | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| payload/sherpa-onnx-c-api.dll | `408F9AB683D1B77CBF7F06D48BD3AF543ED783CC9093C19F520C7C8524619C8F` |
| payload/sherpa-onnx-cxx-api.dll | `DBAC55CECA8383FE28BD7C656FD125BEC60DD35D7DF1A3A28EA36A503EF6C562` |
| gaea-20261123-b1591b4d.zip | `C2D5293B962D8099E6A048F6F089D4B75CBD6772D554374A93CEF4AD14E6A17C` |
| gaea-20261123-b1591b4d-src.zip | `17646CD7BFEF451EABDFB55FBA826A4400B956DC64C6847204F907EB528A9353` |
