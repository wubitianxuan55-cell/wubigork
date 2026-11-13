# gaea-dsh-context-fleet-20261113-uncommitted

- 源码提交：`18fb5dcb7bb794a10179280f701a5644b4f5ab9c`
- 父提交：`5c83cf026433dc0e65d9496bcec1241a451d503a`
- 提交规模：76 files，13,784 insertions，2,575 deletions
- 范围：全部当时未提交的源码、文档、测试、截图、未提交工作树快照记录，以及此前的灵犀 batch3 改动
- 快照基准：父提交 `5c83cf02`
- 快照文件：36 个，原始体积 500,033,756 bytes
- ZIP 条目：42 个，包含 `tracked.patch`、完整工作树状态、文件清单和 33 个当时未跟踪文件
- 归档边界：14 个既有发布 ZIP/TAR 保留在工作树快照包中，不进入 Git 历史
- 验证：13 个定向测试文件 161 例通过；`tsc -b` 通过；ESLint 0 error；`docs` 卫生守卫通过
- 前端生产构建：`npm.cmd run build` 通过，Vite 24.68s
- 未执行：Wails 构建、装机、push

## 产物

| 文件 | Bytes | SHA256 |
|---|---:|---|
| `gaea-uncommitted-20261113.zip` | 526,813,847 | `E262BDAAB32DF21D6DDFC3D498401B4FB29A9AAEF97AEFD9902F2A78D4996A2F` |
| `gaea-18fb5dcb-source.tar.gz` | 42,625,937 | `FED4F578F7990C4619527DCC962DC45FDDC41F4B24DA7A617A32BC747FEA777E` |
| `tracked.patch` | 327,756 | `54F468525F35CDCD4E51A41B77E1C0A91C7023591D67E13EC080EFFB491AA336` |

## 文件记录

- `UNCOMMITTED-FILES.tsv`：36 个快照文件的路径、大小、SHA256。
- `TRACKED-FILES.txt`：相对父提交的已跟踪文件变更清单。
- `WORKTREE-STATUS.txt`：打包时完整 Git 状态。
- `HEAD.txt`：快照基准提交。
- `SHA256SUMS.txt`：两个归档和补丁的发布校验和。

## 恢复方式

1. 检出父提交 `5c83cf02`。
2. 应用 `tracked.patch`：`git apply --binary tracked.patch`
3. 将 `worktree-files/` 内容覆盖仓库根目录。

ZIP/TAR 仍遵守项目既有“磁盘保留、不入 Git”纪律。
