# gaea-bettersidebar-20261112-f49ea150

> 桌面预览包与源码归档记录。2026-11-12 23:09 CST 生成。

## 提交

- 功能提交：`f49ea1500026526b822c80be39cfb08d06026869`
- 短 SHA：`f49ea150`
- 提交标题：`feat-office-dsh-better-sidebar-workbench`
- 父提交：`daaa00654d1db7b1e0676bace326c878a1f83cae`
- 提交时间：`2026-11-12T23:04:17+08:00`
- 改动规模：80 files，7,786 insertions，374 deletions
- 分支：`main`
- 推送状态：**未推送**。构建时相对 `origin/main` ahead 1；本包元数据提交后 ahead 2。

## 构建

- 干净工作树：`C:\AI\wubigrok\.worktrees\pkg-f49ea150`
- 工作树模式：detached HEAD，精确检出 `f49ea1500026526b822c80be39cfb08d06026869`
- 构建前核验：`git status --porcelain` 为空
- 前端依赖：`frontend\node_modules` 从主工作树建立 junction 复用
- 前端：`npm.cmd run build`，通过，Vite `20.38s`
- Wails：`v2.13.0`；完整构建 `50.158s`；bindings 未跳过
- Go：`go1.26.10 windows/amd64`
- C 工具链：ucrt64 `gcc 16.1.0`；`CGO_ENABLED=1`；`CC=C:\msys64\ucrt64\bin\gcc.exe`
- 临时目录：系统默认 `C:\Users\wubi\AppData\Local\Temp`
- Wails 命令：`wails build -s -ldflags "-s -w" -trimpath`
- 已知提示：生成器打印 `Not found: time.Time`；构建成功，冒烟成功

## 冒烟

- 同一构建四件套放入包内临时 `payload/` 正式路径
- 命令：`scripts\smoke.ps1 -ExePath <绝对路径>\payload\gaea.exe`
- 结果：`/api/health` 返回 200，响应 `{"status":"ok"}`
- 冒烟 PID：30100，已回收
- 说明：本机直接从 `.worktrees` 启动 exe 会被系统策略拦截；不是构建失败，
  因此以 releases 正式路径的同一 payload 执行验收。

## 产物

| 文件 | 大小（bytes） | SHA256 |
|---|---:|---|
| `gaea-f49ea150-preview.zip` | 32,502,355 | `7038DDBE20F3B5A6821BE9D7140BF69257E9037940F3E6EAEB6C7D1E0E2A172A` |
| `gaea-f49ea150-source.tar.gz` | 39,080,619 | `776686C69D939C4DB8720531705C0CFEFCB6D62E6F5845B7222AE7F411C236FC` |

### 预览 ZIP 内容

ZIP 根目录只有以下四个文件，已逐项读取并复核 SHA256：

| 文件 | 大小（bytes） | SHA256 |
|---|---:|---|
| `gaea.exe` | 62,486,528 | `986C89666AB1588D10E0881AAEBA8129B35EC62577034FA88C1A2155F419EE96` |
| `onnxruntime.dll` | 19,288,888 | `68D0D5F502BE4509E2A16F932627A95137D748B2FAE519027F80125C0B1F550B` |
| `sherpa-onnx-c-api.dll` | 4,294,656 | `5DD36DBD3572846B72C4A40A4017C5A770160EE8E00DA5053BEBC8B2632501E8` |
| `sherpa-onnx-cxx-api.dll` | 264,192 | `4C56327796CDB980DCBF90025C28093D89CA286088FB74C3B107F2F633E67FFE` |

### 源码包

- 生成命令：
  `git archive --format=tar.gz --prefix=gaea-f49ea150/ -o <target> f49ea1500026526b822c80be39cfb08d06026869`
- 内容来自 Git 提交，不包含主工作树平行测试优化、构建生成的绑定差异或任何未跟踪文件。
- TAR 根目录前缀：`gaea-f49ea150/`

## 交付状态

- 预览 ZIP：磁盘保留，按仓库 P2-3 纪律不入库
- 源码 TAR：磁盘保留，按仓库 P2-3 纪律不入库
- 包记录：`PACKAGE-INFO.md` 与 `SHA256SUMS-f49ea150.txt` 入库
- 装机：**未执行**，未覆盖 `C:\AI\gaea-app`
- push：**未执行**
- 临时 `payload/`：仅用于同字节正式路径冒烟；包核验后已无损移入忽略的
  `.worktrees/pkg-f49ea150/.package-payload-smoke/` 留存构建证据，不进入交付目录

## 结论

`f49ea150` 的功能代码与桌面预览产物已完成从提交身份、干净构建、正式路径冒烟、
压缩包内容、源码归档到 SHA256 的闭合记录。发布动作仍等待用户明确决定。
