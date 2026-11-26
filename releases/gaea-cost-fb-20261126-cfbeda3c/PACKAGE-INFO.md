# gaea-cost-fb-20261126-cfbeda3c

- 交付提交：`cfbeda3c`（`chore(skill): 内置测算样表换版——两层 ROUND 预置`）
- 内容提交：`3257f87e`（bindings 同步）、`1a13d339`（文件工具与闸门）、`1495f811`（workcost 库层）、
  `4cc6c1af`（app 造价工具面）、`2ce3a532`（交付门+clear_where）、`96767a43`（技能与 playbook 文案）
- 产品版本：日常装机包，不抬版本（wails.json productVersion 未动，4.544.0）
- 构建工作树：`C:\AI\wubigrok\.worktrees\pkg-cfbeda3c`（detached HEAD @ cfbeda3c，
  frontend/node_modules 为主树 junction）
- 前端：`npm.cmd run build`，Vite 27s
- 桌面：系统默认 `TMP/TEMP`、`CGO_ENABLED=1`、ucrt64 gcc；PATH 前置 ucrt64 bin、
  后置主树 `build/bin`（首次构建漏后置 → gen_bindings 0xc0000135，补上即过）；
  `wails build -s -ldflags "-s -w" -trimpath`，12s（缓存热）
- 冒烟：正式 payload `/api/health` 返回 200（`{"status":"ok"}`，冒烟进程 PID 38076 已回收）
- 装机：`gaea_install.ps1 -SourceDir <payload> -Launch` → INSTALL OK，装机位
  `C:\AI\gaea-app`，新壳 PID 61892（20:25:50 启动，晚于构建时刻 20:25:33），
  诊断口 18123 `/healthz` 回 `ok`
- 回写：包版四件已回写主树 `build/bin`（payload / 装机位 / build/bin 三处 SHA256 一致）
- exe SHA256 `710B33E65B320C6EBD355B8ADAAF37C43619DE70052F8E79CAA180120B3BA118`
  （zip `77C55CAA…`，src.zip `88FF7AA4…`，全量见 SHA256SUMS.txt）；三个 DLL 与
  上一包逐字节相同
- 源码包：`git archive cfbeda3c -- . ':(exclude)releases'`（5453 文件）

## 交付内容（造价反馈三批：太原 #15 / 百锦路 #13 / 旺平 #12，24 条对码+对会话日志后归并落刀）

### 文件工具与闸门（1a13d339）
write_file 的 overwrite 进模型可见 compact schema（同回合闸豁免参数此前模型发不出来，
三场 26 次调用 0 次带上）；md_validate/cad_ask 绑工作区（漏绑实例相对路径必被读边界拒）；
发布边界豁免 .gaea/work/ 过程目录、ASCII 特征词要求左边界（latest.xlsx 不再误放）、
回执逐件给隔离件完整落点与工作区内取回序列（旧「移到工作区外」实测被写根拦死）、
quarantine_list 按盘面报在不在；move_file 占用预检点名并给 office_edit 出路；
bash 空参数体单列话术；office_edit 操作名归一（wrİte_range 类模型输出怪癖）；
agent 包存量红（move_file stale 豁免）测试对齐。

### workcost 库层（1495f811）
地区并入裁决收窄（regionMergeDecision：缺失并入继承原行地区；不同规范地区=分行新建原行不动；
铜陵 #18 与百锦路 #3 对撞的收口），暂存流同裁决+暂存码别名落含量行（修掉并入后悬空）；
控制价只认纯控制价行+疑似误读告警；行类漏填明细行显形；机检四项（quota_gap 按身份归并、
dangling 回存骨架、region_mismatch、费率文案）；检索 region 过滤与总数；落地预演；dims 待整理明细。

### app 造价工具面（4cc6c1af）
列表 limit 钳位+总数+还有更多；summary 综合单价取到分再乘工程量（与工作簿同口径）、
分部小计、导入快照逐行对账、取费链走 ComposeProjectFees；save_resource 尊重显式编码、
同身份跨地区放行；import_project 显式 control_price；映射回执上限 12→60。

### 交付门与 xlsx_apply（2ce3a532）
审计区分「未重算」与「重算过、结果为空串」（读原始 XML 形态），后者不阻塞——样表 480 个
未用槽位公式此前把三场全部判「未达」；未重算按表列与公式模式聚类；新 op clear_where
条件清格（替代实测误删实算公式的手写清扫脚本），顺带修「=文本」比较永不命中的判定。

### 技能文案与样表（96767a43 / cfbeda3c）
cost-estimate 技能与 playbook：两层 ROUND、E/J/K 列位、定额块判定三问+1 块合法、
组数增减配方（末尾裁组 xlsx_apply 操作集）、清单口径（资料优先级/物理单位/能源进定额）、
返工修订记录；内置样表换用户 gaea 改版（fa584a4b，两层 ROUND 预置，逐格比对值与样式零差异，
真实 LibreOffice 重算 1298 公式零错误）。样表修改一律经 gaea，不再手搓。

### 门禁
build / vet / gofmt 净；golangci --new-from-rev 0 issues；tool / builtin / workcost /
skill / xlsxedit / agent / boot 全绿；app 1380 例四分片全绿（98s）；工具面预算三档在线内
（standard 16741/16750）；真实 LibreOffice 冒烟三条过。

### 余项
清单超 30 组仍需脚本追加（可让 gaea 把样表扩到 60 组，待令）；真实模型实弹未做；
office_edit 未知名就近提示未做（操作名单在 C# 侧）；版本对比、工艺标签挂拍板池。
