# 斑马进度 ZPert8 逆向 · 总索引（2026-11-14）

> 本文件是这轮逆向的**唯一入口**：11 份报告 + 60 份证据文件 + 全部关键结论的地址溯源。
> 后续任何人接手，从这一页出发即可，不必重读所有报告。

---

## 0 一句话现状

| 子系统 | 状态 |
|---|---|
| **CPM 计算内核（六个时标）** | ✅ **完全闭环**（正推/逆推/时差/关键判定/日历底座，全部实测） |
| 视图库 / 列模型 / 分组 / 筛选 | ✅ 闭环（对象布局 + 序列化契约 + show_type 规则） |
| 命令派发 / 埋点 / 导出族 | ✅ 闭环（注册制 + 6 条导出路径 + MS Project XML 规格） |
| 时标刻度（横轴怎么画） | ✅ 闭环（fov 枚举 + 分段表 + 自动联动 + 像素换算） |
| 网络图 / 布图 | ✅ 闭环（见 network-re，布图 = 迭代拟合器） |
| 文件容器 / ViewDB 解密 | ✅ 闭环（AES-256-CBC + brotli，密钥已取） |
| 登录 / 授权 | ⏸ 已搁置（下一层是加密锁驱动，需驱动级方案） |

---

## 1 样本与工程环境

| 项 | 值 |
|---|---|
| 样本 | `C:\Program Files\ZebraTech\ZPert8\ZPert.exe` 7.0.706.0 / 22.3 MB |
| SHA256 | `C61972FAA6A1A08AB846037564CAAD7B2AB23ABCEE3A11B27885DBD24E094E68` |
| Ghidra | `C:\Tools\ghidra_12.1.4_PUBLIC` + `C:\Tools\jdk-21.0.13+9` |
| 工程 | `C:\Tools\ghidra-proj\ZPertProj`（已全量分析，反编译秒级出结果） |
| 取证区 | `C:\AI\zpert-re\`（`evidence/`、`tools/`、`ghidra/`、`mockup/`） |
| 红线 | 取证区内容**不得**复制回 gaea 仓库；仓库内只放文字报告 |

**节区**（`.pdata` 异常 → 函数边界只能靠 capstone 切块或 Ghidra）：
`.text` 0x140001000 / `.rdata` 0x1407c1000 / `.data` 0x141096000 / 异常节 `.l'&` 0x14116f000

---

## 2 报告清单（11 份，建议阅读顺序）

| # | 报告 | 主题 | 关键结论 |
|---|---|---|---|
| 1 | [distill](gaea-schedule-zpert8-distill-20261114.md) | 全产品面 | 排程内核口径、命令面、文件容器 |
| 2 | [network-re](gaea-schedule-zpert8-network-re-20261114.md) | 网络图/布图/协作 | 5 模式枚举、点-箭模型、布图=迭代拟合器、关系合法性矩阵 |
| 3 | [uiux-re](gaea-schedule-zpert8-uiux-re-20261114.md) | 前端 UX/UI | 功能区 916 串、20 个布局树、6 套主题、命中枚举、CEF 桥 |
| 4 | [uiux-port-plan](gaea-schedule-zpert8-uiux-port-plan-20261114.md) | UI 移植计划 | 视觉令牌与控件映射 |
| 5 | [replication-assess](gaea-schedule-zpert8-replication-assess-20261114.md) | 复刻判定 | 各模块改造量与优先级 |
| 6 | [timescale-re](gaea-schedule-zpert8-timescale-re-20261114.md) | 时标刻度 | fov 1..6、分段表类型 日·一~六、自动联动、像素出口 |
| 7 | [timeparams-re](gaea-schedule-zpert8-timeparams-re-20261114.md) | 六个时间参数 | 列 10170–10220、键池、双套字段、is_keywork 位 |
| 8 | [viewbackend-re](gaea-schedule-zpert8-viewbackend-re-20261114.md) | 视图库后端 | view/列/分组三层布局、`show_type`、9 视图=列预设 |
| 9 | [commanddispatch-re](gaea-schedule-zpert8-commanddispatch-re-20261114.md) | 命令派发 | 注册制、无总表、埋点定位、导出族 6 路径 |
| 10 | [triple-re](gaea-schedule-zpert8-triple-re-20261114.md) | 埋点/导出/六参数 | MS Project XML 规格、双套字段、CPM 定位过程 |
| 11 | **[cpm-re](gaea-schedule-zpert8-cpm-re-20261114.md)** | **CPM 内核** | **正推/逆推/时差/关键判定三 bit/日历底座 —— 本文最核心** |

---

## 3 关键结论速查（按主题，附地址溯源）

### 3.1 CPM 内核（六个时标的求值链）

| 结论 | 地址 / 证据 |
|---|---|
| **总控** `FUN_1402730c0`（4755 B）：复位 → 起点播种 → 正推 → 逆推 → 关键判定 → 三开关 | `decomp_cpm_cand.c` |
| **正推驱动** `FUN_14026ed70`（2281 B，memo：`work[0x80]!=哨兵` 则跳过） | `decomp_forward.c` |
| **单工作正推** `FUN_14026e360`（2565 B）：ES 对紧前取 **max**，四条时间线并行 | `decomp_fwdwork.c` |
| **逆推取 min** `FUN_14026fbd0`（2689 B）+ **算 LS** `FUN_140270660`（1296 B） | `decomp_lfprep.c` / `decomp_backward.c` |
| **日历感知加减** `FUN_1401d8610`（2333 B）：连续日序索引 + 非工作日位图 | `decomp_rellag.c` |
| **求差** `FUN_1401d9940(cal,out,a,b,0)`；**比较** `FUN_1403125d0(a,b)` | — |
| **TF/FF 求值器** `FUN_1402def30`（2776 B）：沿父子树取子项极值再求差 | `decomp_fftf.c` |
| **`Work` 类**：vtable `0x1408c9c68`（6 项）；ctor `FUN_1402d2130` | `vtables.txt` |
| **`ZQtPertDoc` 类**：vtable `0x140c5fb98`（104 项） | `vtables.txt` |
| **关键判定三 bit**：bit0 计划完成==LF普通；bit2 预测完成==LF投标；bit1 预测完成≤LF普通 | `FUN_1402730c0` 内 |
| **环检测**：位图 + `idx = workCount + work[1]`（区分节点类型） | 正推/逆推同款 |
| **状态短路**：进行中/已完成不参与正推，`ES = 实际开始(+0x58)`；逆推 `LF = 实际完成(+0x60)` | — |
| **工期字段** `+0x68`（三处互证：计划开始=计划完成−工期、预测开始=预测完成−工期、LS=LF−工期） | — |
| **关系类型** `+0x7c`：1=FS（用 pred EF）；2/3=改基准（SS/FF/SF 族） | `decomp_fwdwork.c` |

**`Work` 时间字段全表**（双源验证：ctor 初始化序列 + 序列化器取值）：

```
+0x48 要求开始      +0x50 要求完成
+0x58 实际开始 ★     +0x60 实际完成 ★
+0x68 工期
+0x80 计划开始      +0x88 计划完成
+0x90 预测开始      +0x98 预测完成
+0xB0 ES普通        +0xB8 EF普通      +0xC0 LS普通      +0xC8 LF普通
+0xD0 ES投标        +0xD8 EF投标      +0xE0 LS投标      +0xE8 LF投标
+0xF0 原计划开始    +0xF8 原计划完成
+0x79 标志位：bit0 关键工作 / bit1 预测不超期 / bit2 投标关键工作
+0x248 → 文档指针；文档 +0x11b6 = 普通/投标口径开关
+0x34c 状态（0 未开始 / 1 进行中 / 2 已完成）
```

### 3.2 六个时间参数（列与键）

| 结论 | 地址 |
|---|---|
| 六列 10170/10180/10190/10200/10210/10220（连续、`type=4`、宽 110/140） | 列定义派发 `FUN_1400d10f0` |
| 键池 `earlist_start_time`(0x1408d05c0，**拼写是 earlist**) / `earlist_stop_time`(0x1408d05d8) / `latest_start_time`(0x1408d05f0) / `latest_stop_time`(0x1408d0608) / `latest_start_free_time`(0x1408d0458) / `latest_stop_free_time`(0x1408d0470) | `.rdata` |
| 派生列：自由时差 10290 → `free_time_difference`(0x1408d0590)；总时差 10300 → `total_time_difference`(0x1408d05a8) | 同上 |
| 六参数取用唯一入口 `FUN_1402f1490`（计划变动分析模型） | `decomp_cpm7.c` |
| UI 分组标题 = **「计算时间（预测）」**，8 项顺序：最早开始→最早完成→最晚开始→最晚完成→自由版×2→自由时差→总时差 | `region_0x140678710` |
| 关键工作不是查表算 TF，而是 `work+0x79` 的 bit（`FUN_1402efed0`，37 B） | `decomp_iskey.c` |

### 3.3 视图库 / 列 / 分组

| 结论 | 地址 |
|---|---|
| 视图读 `FUN_1402ca1b0`（3348 B）/ 写 `FUN_1402cb540`（1477 B） | `decomp_viewio.c` |
| 视图对象：`+0x20 view_mode / +0x24 view_source / +0x68,0x69 显示开关 / +0x6c pert_mode / +0x90 display_mode / +0x98 groupby / +0x70,0x78 列向量 / +0x88 lock_col / +0x8c 索引列宽 / +0x178 筛选向量` | 同上 |
| 列对象 **0x58 字节**：`+0 id / +0x04 is_visible / +0x05 is_readonly / +0x06 show_type / +0x08 name / +0x28 alias / +0x48 width / +0x4C data_type / +0x50 align / +0x54 is_required_column` | `decomp_viewcol.c` |
| **`show_type` 推导**：`data_type 1/3/6/7→0`，`2/5→1`，`4/8→2` | 同上 |
| 分组对象 **0xD8**：`+0x68 interval_type / +0x70 interval_start / +0x98 interval_step / +0xC0 time_unit / +0xC8 interval_values` | `decomp_groupby.c` |
| **9 个内置视图 = 列可见性预设**（7/17/7/7/9/20/19/17/8 列），四个模式字段全为 `0/0/1/0` | ViewDB.gdb.json |
| 只有「里程碑列表」带 `column_filters`（`filter_info` 是**二次编码的 JSON 字符串**） | 同上 |

### 3.4 时标刻度（横轴）

| 结论 | 地址 |
|---|---|
| 六档 `fov`：1 分钟 / 1 小时 / 1 工作日 / 1 工作周 / 1 工程月 / 自然日档 | `FUN_1402c5e50` |
| 间隔集：年-月{1,2,3,4,6} / 月-天{1..7,10,15} / 工程月{1,2,3,6} | `FUN_14069c2c0` |
| 分段表「类型」= **日 / 一 / 二 / 三 / 四 / 五 / 六**（7 个单字） | 类型名表 `0x1410f12e0`，init `FUN_1400b94a0` |
| 分段记录 **0x68**，向量在 `ZoomTime+0x128/+0x130` | — |
| 自动联动 `FUN_14069a4d0`：比例 → scale（0.04..0.4，`×0.5` 收尾） | `decomp_scale5.c` |
| 刻度→几何唯一出口 `FUN_1402c4b40`（`pos[i] = Σ step×scale`，24 处调用） | `decomp_scale6.c` |
| 常量：`2.4` 每日工作小时、`0.08/0.10/0.12/0.24/0.40` 密度档、`0.5` 收尾系数 | `.rdata 0x1408bd210..250 / 0x1407f8ba0` |

### 3.5 命令派发 / 埋点 / 导出

| 结论 | 地址 |
|---|---|
| **命令表纯数据**（916 串 = 路径+键+ID，**无函数指针**） | `FUN_140017b00`（34,208 B） |
| 数量纠正：916 是**字符串引用数**；去重命令 ID **263**、键 89 | `decomp_cmdreg.c` |
| **动作表 299 条** `{内部名, GUID"5000000000…"}` → 全局 map `DAT_141126b38` | `FUN_140014e30` |
| 双映射查找（先覆盖表 `DAT_141126b90`，再静态表），**callers = 1** | `FUN_14032a3b0`（216 B） |
| **埋点** `FUN_140576bc0`：命令键 → GUID → 写 `doc+0x2438`（QString），并经 `ZGatewayServer` 上报 | `decomp_actuse.c` |
| 埋点只覆盖导出族（6 个调用方）；`11xxxxx/12xxxxx` 只是命令 ID 分段 | — |
| 导出 6 路径：Excel(libxl) / SVG(QSvgGenerator) / PDF(QPrinter) / 删旧再写 / 按 MIME / 向导 | `decomp_handlers.c` |
| **MS Project XML 规格**：单个 WorkWeek(1984→2049「综合周日历」) + 12 个任务字段 + 4 个关系字段 | `msproject_xml.txt`（模板在文件偏移 `0xc88270`） |

### 3.6 视图库解密（前端轮成果）

密钥（运行期抓取，对所有文件/机器相同）：
`013a156cb1660e3c469c69bfaf4acf19017f79ebb7c60e390aaca6cb8160ca0f`
链路：`AES-256-CBC(key, iv=容器内) → 去填充 → brotli`；抓取法：Frida hook `FUN_14031ed50(ctx, key32, iv16)`（RVA 0x31ed50）

---

## 4 证据文件分类索引（`C:\AI\zpert-re\evidence\`）

### 4.1 反编译产物（按子系统）

| 子系统 | 文件 |
|---|---|
| **CPM 内核** | `decomp_cpm_cand.c`（总控+Work ctor）`decomp_forward.c` `decomp_fwdwork.c` `decomp_backward.c` `decomp_lfprep.c` `decomp_rellag.c` `decomp_fftf.c` `decomp_iskey.c` |
| 六个时间参数 | `decomp_time6.c`（列注册）`decomp_timedic.c`（119 列派发）`decomp_cpm7.c` / `decomp_cpm8.c`（分析/加载） |
| 视图库 | `decomp_viewio.c` `decomp_viewcol.c` `decomp_groupby.c` `decomp_view.c` `decomp_viewdb.c` |
| 时标刻度 | `decomp_scale.c` ~ `decomp_scale8.c`（8 份） |
| 命令派发 | `decomp_cmdreg.c` `decomp_actions.c` `decomp_actlookup.c` `decomp_actuse.c` `decomp_ribbon.c` `decomp_ribbonhelper.c` `decomp_handlers.c` |
| 文档/页对象 | `decomp_page2438.c`（ZQtPertDoc ctor）`decomp_mapops.c`（QString 原语） |
| UI | `decomp_ui1.c` `decomp_ui2.c` `decomp_hit.c` |
| 授权/加密 | `decomp_aes*.c` `decomp_kdf*.c` `decomp_auth1.c` |
| 早期 | `decomp1.c` ~ `decomp5.c` |

### 4.2 结论汇编（可 grep，纯文本）

| 文件 | 内容 |
|---|---|
| **`cpm_core.txt`** | CPM 三层结构 + 正推/逆推伪码 + 字段表 + 三 bit + 五项补齐 |
| `cmd_dispatch.txt` | 命令表/动作表/查找器/埋点 + 勘误 |
| `view_model.txt` | 视图/列/分组三层布局 + 消费点地址 |
| `scale_units.txt` | 六档串池 + 控件表 + 状态结构 + 常量 |
| `time_params.txt` / `timeparams_dual.txt` | 六参数列/键 + 双套字段 + 总时差片段 |
| `work_fields.txt` | 工作信息 4 页签字段表 + 三方互证 |
| `msproject_xml.txt` | MS Project XML 模板与结构 |
| `vtables.txt` | **ZQtPertDoc(104) / ZBasePertDoc / Work(6) vtable 索引** |

### 4.3 原始/派生数据

| 文件 | 内容 |
|---|---|
| `zpert_strings.txt`（1.7 MB） | 原始 ASCII 串 |
| `cn2/3/4.txt` | 清洗后的中文 UI 文案（3 千余条） |
| `rtti.txt` / `rtti_plain.txt` | 772 个 RTTI 类名 |
| `qmeta.txt` / `qmeta_sig.txt` | Qt 元对象 |
| `zpbdata_schema.json` / `.txt` | zpbdata.proto 描述符（RBP/RestrictedFunctions/ProductInfo） |
| `ascii_ids.txt` / `grouped.txt` | 蛇形键标识符 / 字符串簇 |
| `net_cn.json` / `net_ids.txt` / `net_va.json` / `net_xref.json` | 网络图取证 |
| `ui/index3.json`（5.5 MB） | **15,265 个代码块 ↔ 字符串引用索引**（所有 `--str` 查询靠它） |
| `ui/region_*.txt` | 按区块导出的字符串（命令注册表、工作信息字段、刻度设置…） |
| `ui/ui_*.txt` | 页签命令 / 布局树 / QSS / 对话框 / 对象名 |
| `rcc/*.qss` `rcc/svg_*` `rcc/png_*` | 6 套主题 QSS + 图标（从 exe 内 Qt 资源还原） |
| `viewdb/*.gdb` `.gzp` + `*.json` | 视图库原容器 + 解密后 JSON |

---

## 5 工具链（`C:\AI\zpert-re\tools\`）

| 工具 | 用途 | 常用命令 |
|---|---|---|
| `uiindex3.py` | 建/查「代码块 ↔ 字符串」索引 | `--str 前锋线` / `--region 0x1400d10f0` / `--build` |
| **`ghidra_decomp.ps1`** | **Ghidra headless 反编译** | `-Out out.c -Addrs "0x1402730c0"`（地址**必须加引号**） |
| `rccdump.py` / `rccraw.py` | Qt 资源还原（zlib 暴力 inflate + 魔数分类） | 直接运行 |
| `viewdb_open.py` | **ViewDB 最终解密器** | AES-256-CBC + brotli |
| `disasm.py` / `dis_around.py` / `xref_global.py` | 定点反汇编 / 周边 / 全局引用 | — |
| `pdata.py` / `vtable.py` / `vtabref.py` | 函数边界 / RTTI 探测 | — |
| `protodesc.py` | proto 描述符提取 | — |
| `uicmd.py` / `uilayouts.py` / `uiqss.py` / `uiscan.py` | UI 命令表 / 布局树 / 样式 / 扫描 | — |

**本轮新增的有效手法**（写下来避免下次踩坑）：

1. **扫立即数定字符串引用**：`.rdata` 里找键串 → 扫 `.text` 的 `lea r64,[rip+disp]`（`48/4C 8D 05` 或 `4C 8D 05`）→ 得消费函数。
2. **小偏移要用 disp8**：`0x79` 之类只占 1 字节，编码是 `80 4b 79 01`，**用 disp32 形态扫会 0 命中**（本轮在此卡了两轮）。
3. **从类入手而非偏移**：大函数里偏移复用严重，按 disp 盲扫会把 UI 构造器全捞上来；先取 vtable（构造器里 `lea` 到 `X::vftable`）再逐方法筛，命中率高得多。
4. **宽字符 vs 窄字符**：中文串是 UTF-16（`FUN_1400decb0`），键/ID/GUID 是 ASCII（`FUN_1400da890`）。用错编码搜不到（`estimate_stop_time` 就因此漏过）。
5. **写盘函数族反查**：`FUN_1400db990`（QString 移动）/`FUN_1400ddfb0`（拷贝）这类原语，看字节形态就能认出来，别当成 map 操作（本轮误判过一次）。

---

## 6 全局未闭环清单

| # | 项 | 影响 | 建议手法 |
|---|---|---|---|
| 1 | `cal` 跨年回绕的精确处理（`idx<0` 时 `+99`） | 时间精度边缘情况 | 读 `FUN_1401d8f30` / `FUN_1401db4d0` |
| 2 | `FUN_1402def30` 内部极值选择的逐条条件 | 汇总行时差 | 读该函数 2776 B 全文 |
| 3 | 起点的 LF 初值（`FUN_14026fbd0` 起点分支） | 起点时间 | 读该函数后半 |
| 4 | 登录/授权 | 商业功能 | 需驱动级方案（加密锁） |
| 5 | CEF 侧 AI 面板协议 | 云功能 | 需看 `QCefView.dll` + 运行期消息 |
| 6 | 运行时观感（动画/真实间距） | UI 像素级还原 | 真机截图逐像素比对 |

---

## 7 复刻优先级建议（按投入产出）

| 优先级 | 模块 | 依据 |
|---|---|---|
| **P0** | **CPM 内核 + 六个时标** | 已有完整伪码，可直接实现；错一处则全表错 |
| P0 | 视图库（列可见性预设 + 列宽 + 分组） | 布局与契约已明确，数据在 `.gdb` 里可直接读 |
| P1 | 命令注册表 + 动作 GUID | 纯数据，照抄即可；注意"无总表"这个结构事实 |
| P1 | MS Project XML 导出 | 12 字段规格已明确，通道要留 |
| P2 | 时标刻度（fov + 分段表 + 自动联动） | 公式完整；`刻度比例%` 与 scale 的换算系数待运行期确认 |
| P2 | 网络图布图 | 迭代拟合器，参数在 network-re |
| P3 | 主题/图标 | QSS 原文已还原，可直接取用 |

**三条最容易踩的坑**（都写在 cpm-re §8）：
1. 进行中/已完成的工作**不参与正推**，ES 直接取实际开始；
2. 计划开始 / 预测开始与 ES **同源回写**，不是三个独立字段；
3. 关键判定用「完成 == LF」而非「LS − ES == 0」，跨假期时两者不等价。
