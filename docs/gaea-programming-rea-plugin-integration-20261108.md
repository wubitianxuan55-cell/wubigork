# gaea 编程板块：DSH Web、REA 与桌面插件迁移记录（2026-11-08）

> 状态：✅ 已落地并装机。产品版本 `4.534.0`；本记录覆盖隔离 DSH Web、
> REA MCP、桌面版插件迁移、同源认证以及 cmd 黑窗问题。

## 结果摘要

- gaea「编程」板块直接内嵌最新版 DSH Web `0.2.2-rc.1`。
- REA `6.8.0` 已作为隔离 MCP 接入，实测暴露 `229` 个工具。
- 桌面 DSH 的 9 个外部依赖已迁入独立 Web profile；按桌面现状启用其中 8 个。
- DSH 与官方桌面版在安装、home、agents、profile、插件和补丁层面完全分离。
- 已修复编程板块运行时周期出现的 PowerShell/cmd 黑窗。
- 最终 gaea 冒烟通过并安装到 `C:\AI\gaea-app`。

## 隔离 DSH

| 项目 | 路径或值 |
|---|---|
| 安装根 | `C:\AI\dsh` |
| DSH 版本 | `0.2.2-rc.1` |
| `DSH_HOME` | `C:\AI\dsh\home` |
| `DSH_AGENTS_HOME` | `C:\AI\dsh\home\agents` |
| `DSH_BASE` | `C:\AI\dsh\node_modules\@deepseek-ai` |
| Web profile | `C:\AI\dsh\home\profiles\web` |
| 监听 | `127.0.0.1:3091` |
| iframe authority | `http://dsh.wails.localhost:3091` |
| dsh-wubi 源码 | `C:\AI\wubigrok\clones\dsh-wubi` |

DSH 绑定仍是回环地址。浏览器侧使用 `dsh.wails.localhost:3091`，与 gaea
WebView 的 `wails.localhost` 保持同站点，避免 `SameSite=Strict` 登录 Cookie
在跨站 iframe 中被拦截。

## REA

| 项目 | 路径或值 |
|---|---|
| npm 包 | `C:\AI\dsh\rea\node_modules\rea-agents` |
| 版本 | `6.8.0` |
| 源码克隆 | `C:\AI\wubigrok\clones\rea` |
| Node | `C:\AI\dsh\runtime\node-v24.22.1-win-x64\node.exe` |
| MCP 注册 | `C:\AI\dsh\home\cordis.patch.yml` |
| 技能 | `C:\AI\dsh\home\agents\skills\reverse-engineer-anything` |
| MCP 命名空间 | `mcp__rea__*` |
| 实测工具数 | `229` |

REA 的静态 JavaScript/Electron、桌面包、PE/ELF、模型文件、.NET、APK
签名分析可直接使用。Ghidra、IDA、JADX、Kuna 等可选原生分析引擎未安装，
因此不承诺原生反编译能力。

## 桌面插件迁移

桌面 profile 原位置：`C:\Users\wubi\.dsh\profiles\desktop`。只读取其
`package.json`、`cordis.patch.yml` 和已安装包元数据，没有写入桌面 profile。

| 包 | 桌面版 | Web 安装版 | Web bundle |
|---|---:|---:|---|
| `dshmarket` | `1.73.3` | `1.73.15` | 启用 |
| `@changfenhuang/dsh-genui` | `0.15.0` | `0.15.0` | 启用 |
| `dsh-ops` | `0.4.4` | `0.5.1` | 启用 |
| `@michengai/dsh-skills-manager` | `1.2.33` | `1.3.0` | 启用 |
| `dsh-better-sidebar` | `0.30.1` | `0.30.1` | 启用 |
| `dsh-context` | `0.82.0` | `0.82.2` | 启用 |
| `dsh-unlag` | `1.17.3` | `1.17.5` | 启用 |
| `dsh-dream-skin` | `11.4.1` | `11.7.0` | 启用 |
| `@linxin666/dsh-client-ui-skill-explorer` | `0.4.14` | `0.4.16` | 仅依赖，不单独启用 |

`dsh-ops@0.4.4` 的宿主范围是 `<0.2.2-0`，与当前 DSH
`0.2.2-rc.1` 冲突；`0.5.1` 明确声明支持 `>=0.2.2-rc.1 <0.2.3-0`，因此
采用 `0.5.1`。其余插件采用当前兼容版本。

Web bundle 顺序：

```text
@deepseek-ai/dsh-base
@deepseek-ai/dsh-web-app
dshmarket
@changfenhuang/dsh-genui
dsh-ops
@michengai/dsh-skills-manager
dsh-better-sidebar
dsh-context
dsh-unlag
dsh-dream-skin
dsh-wubi
```

`dsh-wubi` 保持最后，以维持其覆盖优先级。`pnpm peers check` 会报告缺少
DSH 宿主 peers，这是 `autoInstallPeers: false` 的预期形态：这些包由 DSH
运行时提供，不应在 profile 内重复安装。

## cmd 黑窗根因与修复

### gaea GPU 遥测（主因）

编程板块打开后，gaea 的 GPU 遥测会周期启动 PowerShell：

- `gpu_perf_windows.go` 每 15 秒左右调用一次 `Get-Counter`；
- `image_comfyui_proc.go` 通过注册表查询 AMD VRAM，并提供 CIM 兜底。

原实现使用裸 `syscall.SysProcAttr{HideWindow: true}`。Windows 会先创建
控制台再尝试隐藏，因此出现短暂黑色窗口。已把两个文件内共 6 处子进程启动
统一改为 `proc.HideWindow(...)`；该封装同时设置 `CREATE_NO_WINDOW`，从创建
阶段就不产生可见控制台。

覆盖的进程包括：

- `powershell.exe` GPU 计数器与 VRAM 查询
- `nvidia-smi`
- `netstat`
- ComfyUI Python 进程

### dshmarket 热重启 helper（次因）

`dshmarket@1.73.15` 的 `lib/restart.js` 在 Windows 上 detached 启动
重启 helper 时漏了 `windowsHide: true`。已在 Web profile 内用 pnpm patch
修复：

```text
C:\AI\dsh\home\profiles\web\patches\dshmarket@1.73.15.patch
```

补丁文件哈希与桌面原包不同，说明修复没有穿透 pnpm 硬链接去改桌面共享包：

- Web 修复后 `restart.js` SHA256：
  `AE6ADF4793D85266F145880209B7985C11C74C894655E04C337858A7D6CAF015`
- 桌面原版 `restart.js` SHA256：
  `FFB8420E6BCF3D018D0902538EC9E9F7A1234877F16C030E3A91A0AF39B11DA3`

### 验收证据

真实打开编程板块并监听进程创建，观察到 GPU `powershell.exe`、DSH 插件
启动的 `reg.exe` 以及对应 `conhost.exe`，`MainWindowHandle` 均为 0，
`visible=False`。GPU 遥测仍正常返回读数，功能没有为了消除黑窗而关闭。

## 构建与产物

按 `docs/gaea-build-guide.md` 执行：

1. `frontend/npm.cmd run build`
2. 使用系统默认 `TMP/TEMP`，前置 MSYS2 ucrt64 GCC，执行：
   `wails build -s -ldflags "-s -w" -trimpath`
3. 复制三个 voice DLL 到 `build\bin`
4. 冒烟 `/api/health` 返回 200
5. `scripts/gaea_install.ps1` 返回 `INSTALL OK`

最终产物：

```text
path: C:\AI\gaea-app\gaea.exe
version: 4.534.0
size: 61,556,736 bytes
sha256: 9D0178BE67E6D501397B0A8CCC3E8BB0F10F56960855176F15E531B8898F7FDD
```

## 桌面隔离核对

- 桌面 DSH 安装目录未写入。
- 桌面 profile 未出现 `.dshwubi.bak`。
- `C:\Users\wubi\.dsh\prompt-inject.md` 不存在。
- `C:\Users\wubi\.agents\skills\reverse-engineer-anything` 不存在。
- REA 技能只存在于 `C:\AI\dsh\home\agents`。
- Web 的 `dshmarket` patch 与桌面原文件哈希不同。

## 更新注意事项

1. 升级 DSH 后先跑 REA：

   ```powershell
   $env:DSH_HOME = 'C:\AI\dsh\home'
   $env:DSH_AGENTS_HOME = 'C:\AI\dsh\home\agents'
   & 'C:\AI\dsh\runtime\node-v24.22.1-win-x64\node.exe' `
     'C:\AI\dsh\rea\node_modules\rea-agents\scripts\rea.mjs' `
     doctor --client dsh --skill --json
   ```

2. 升级或重装 `dshmarket` 后检查 patch 是否仍存在；上游修复该 helper
   后可删除本地 patch。
3. 新增任何 Windows 控制台子进程时，必须使用
   `internal/gaea/proc.HideWindow`，禁止裸 `SysProcAttr{HideWindow:true}`。
4. 不要把桌面 profile 的 `dsh-ops@0.4.4` 直接复制到 DSH `0.2.2-rc.1`。
