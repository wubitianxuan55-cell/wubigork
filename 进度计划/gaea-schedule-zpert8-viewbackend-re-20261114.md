# 斑马进度 ZPert8 · 前端产物 → 后端补齐（视图库 / 列模型 / 分组）· 2026-11-14

> **定位**：前端逆向（`…-uiux-re…` §16–§19）已经把 `ViewDB.gdb` 解开，拿到了 JSON 结构；
> 但**这些字段在后端是什么类型、偏移多少、怎么被消费**，前端那轮看不到。
> 本文按"前端拿到的 JSON 键"逐条回填**后端对象布局与序列化契约**，并纠正一处前端推断。
>
> **样本**：`C:\Program Files\ZebraTech\ZPert8\ZPert.exe`（7.0.706.0）
> **方法**：用前端已知的 JSON 键名（`view_columns_info`/`column_filters`/`groupby_setting`/`is_visible`…）
> 在 `.rdata` 定位键串 → 扫 `lea` 引用 → 反编译序列化/反序列化函数。

---

## 0 一页结论

| # | 结论 | 强度 |
|---|---|---|
| 1 | **视图写入器 `FUN_1402cb540`（1477 B）与读取器 `FUN_1402ca1b0`（3348 B）成对**，字段写入顺序与前端解出的 JSON 键序**逐字一致** → 两侧独立取证互证 | 实测 |
| 2 | **视图对象布局**：`+0x20 view_mode / +0x24 view_source / +0x68 is_show_grid_editor / +0x69 is_show_pert_editor / +0x6c pert_mode / +0x90 display_mode / +0x98 groupby_setting / +0x70..0x78 列向量 / +0x88 lock_col / +0x8c index_column_width / +0x178 筛选向量` | 实测 |
| 3 | **列向量元素步长 `0x58`（88 字节）**，单列由 `FUN_1402cb1a0`（882 B）序列化，字段偏移见 §2 | 实测 |
| 4 | **`show_type` 是后端独有字段**：前端解出的 JSON 里没有，它由 `data_type` 推导 —— `1/3/6/7→0`、`2/5→1`、`4/8→2` | 实测 |
| 5 | **9 个内置视图的"差异"其实是列可见性预设**：`view_mode/view_source/pert_mode/display_mode` 九个视图**全部是 0/0/1/0**；真正不同的是可见列数 7/17/7/7/9/20/19/17/8 与 `is_show_pert_editor`、`lock_col` | 实测（纠正前端 §19.3） |
| 6 | 只有 **里程碑列表** 带 `column_filters`（`search_filter` + `date_filter`，含 `checked_item=-2`、`end_date=32535014400`） | 实测 |
| 7 | **`groupby_setting` 对象布局**（0xD8 字节）+ `time_unit` 与"六个时间参数/时标刻度"**同一套枚举族** | 实测 |

---

## 1 视图对象：布局与序列化顺序

写入器 `FUN_1402cb540` 的键名出现顺序（即序列化顺序）：

```
id  →  name  →  view_mode  →  view_source  →  is_show_grid_editor
    →  is_show_pert_editor  →  pert_mode  →  display_mode
    →  groupby_setting  →  [ view_columns_info … ]  →  lock_col
    →  index_column_width  →  [ column_filters … ]
```

与前端解出的 `ViewDB.gdb.json` 键序**逐字一致**（前端是解密后直接读 JSON，后端是内存对象写 JSON，
两条独立路径给出同一顺序）→ 契约成立。

| 偏移 | 类型 | 字段 | JSON 键 |
|---|---|---|---|
| +0x20 | int | 视图模式 | `view_mode` |
| +0x24 | int | 视图来源 | `view_source` |
| +0x68 | byte | 显示表格编辑区 | `is_show_grid_editor` |
| +0x69 | byte | 显示图编辑区 | `is_show_pert_editor` |
| +0x6c | int | 网络图模式 | `pert_mode` |
| +0x90 | int | 显示模式 | `display_mode` |
| +0x98 | 对象 0xD8 | 分组设置 | `groupby_setting` |
| +0x70 / +0x78 | 指针对 | **列向量 begin/end**（元素 0x58） | `view_columns_info` |
| +0x88 | int | 冻结列数 | `lock_col` |
| +0x8c | int | 索引列宽 | `index_column_width` |
| +0x178 | 指针对 | **筛选向量** | `column_filters` |

---

## 2 列对象：88 字节布局

单列写入器 `FUN_1402cb1a0`（入口 0x1402cb1a0，882 B）：

| 偏移 | 类型 | 字段 | JSON 键 | 备注 |
|---|---|---|---|---|
| +0x00 | int | 列 ID | `id` | 10010 分区 / 10060 计划开始 …（见 `…-timeparams-re…` §1） |
| +0x04 | byte | 可见 | `is_visible` | 9 个内置视图的**唯一主要差异** |
| +0x05 | byte | 只读 | `is_readonly` | |
| +0x06 | byte | 显示类别 | **`show_type`** | **后端推导，前端 JSON 无此键** |
| +0x08 | QString(0x20) | 列名 | `name` | |
| +0x28 | QString(0x20) | 别名 | `alias` | |
| +0x48 | int | 列宽 | `width` | 与前端 §19.4 的实测宽度对齐（110/120/200…） |
| +0x4C | int | 数据类型 | `data_type` | 1 文本 / 2 整数 / 3 枚举 / 4 计算 / 6 / 7 / 8 日期 |
| +0x50 | int | 对齐 | `align` | 示例 JSON 里恒 0（默认左/按类型） |
| +0x54 | byte | 必填列 | `is_required_column` | |

### 2.1 `show_type` 推导表（后端规则，前端拿不到）

仅当"按系统内置视图导出"（`param_4 != 0`）时写入，否则沿用列自带值：

```c
switch (data_type) {
  case 1: case 3: case 6: case 7:  show_type = 0;  break;   // 文本/枚举/…
  case 2: case 5:                  show_type = 1;  break;   // 整数/…
  case 4: case 8:                  show_type = 2;  break;   // 计算值/日期
}
```

→ `show_type` 把 8 种 `data_type` **收敛成 3 类显示形态**。
这解释了为什么六时间参数是 `4`、计划开始/完成是 `8`，二者同为 `show_type=2`（值型/日期型统一按"可格式化数值"渲染）。

---

## 3 分组设置：0xD8 字节布局

序列化器 `FUN_140207a00`（1556 B）：

| 偏移 | 类型 | 字段 | JSON 键 |
|---|---|---|---|
| +0x00 | int | 分组项 ID | `id` |
| +0x04 | int | 分组列 ID | `column_id` |
| +0x08 | QString | 分组名 | `name` |
| +0x28 | QString | 列名 | `column_name` |
| +0x48 | QString | 后缀 | `suffix` |
| +0x68 | int | 间隔类型 | `interval_type` |
| +0x70 | 变体 | 间隔起点 | `interval_start{type,value}` |
| +0x98 | 变体 | 间隔步长 | `interval_step{type,value}` |
| +0xC0 | int | 时间单位 | `time_unit` |
| +0xC8 / +0xD0 | 指针对 | 间隔取值表 | `interval_values[]` |

**`time_unit` 与「六个时间参数」专题里的单位编码同族**（`…-timeparams-re…` §3.3）：
分组可以按时间间隔切块（每隔 N 个 天/周/月 分一组），`interval_type` 决定切法，
`time_unit` 决定单位。示例视图里全是 `interval_type=0 / time_unit=0 / step=1.0` → 不分组。

---

## 4 纠正：9 个内置视图不是 9 套引擎

前端 §19.3 列了一张"表格/图/冻结列/索引列宽"表，容易误读成"每个视图不同实现"。实测九视图：

| 视图 | view_mode | view_source | pert_mode | display_mode | 可见列数 | 表格 | 图 | lock_col |
|---|---|---|---|---|---|---|---|---|
| 计划编制 | 0 | 0 | 1 | 0 | **7** | ✅ | ✅ | 3 |
| 进度管控 | 0 | 0 | 1 | 0 | **17** | ✅ | ✅ | 3 |
| 期间计划 | 0 | 0 | 1 | 0 | **7** | ✅ | ✅ | 1 |
| 期间进度报告 | 0 | 0 | 1 | 0 | **7** | ✅ | ✅ | 1 |
| 资源 | 0 | 0 | 1 | 0 | **9** | ✅ | ✅ | 3 |
| 工程量计划 | 0 | 0 | 1 | 0 | **20** | ✅ | ✅ | 3 |
| 进度对比分析 | 0 | 0 | 1 | 0 | **19** | ✅ | ❌ | 3 |
| 计划变动分析 | 0 | 0 | 1 | 0 | **17** | ✅ | ❌ | 3 |
| 里程碑列表 | 0 | 0 | 1 | 0 | **8** | ✅ | ❌ | 1 |

- **四个模式字段全同**（0/0/1/0）→ 不是"四套模式"，是同一模式的四种取值域都没被用到；
- 真正区分九个视图的是**列可见性集合**（7/17/7/7/9/20/19/17/8）与三个附属开关；
- `is_show_pert_editor` 只有"进度对比分析/计划变动分析/里程碑列表"为 false → 这三个视图不显示右侧图区；
- `lock_col` 只有"期间计划/期间进度报告/里程碑列表"为 1，其余 3。

**结论**：内置视图 = **列可见性 + 少量开关的预设**，共用同一套表格/图渲染引擎。
复刻时不需要为每个视图写引擎，只需要切列集。

---

## 5 筛选：唯一带规则的视图

九个视图里只有 **里程碑列表** 带 `column_filters`，结构（前端已解出，后端消费点本轮定位）：

```json
[{"column_id": 10020,
  "filter_info": "{\"search_filter\":{\"filter_only_child\":true,\"filter_label\":0,
                    \"attribute_to_match\":\"\",\"search_str\":\"\"},
                  \"date_filter\":{\"is_total_include\":false,\"checked_item\":-2,
                    \"start_date\":0,\"end_date\":32535014400,\"filter_only_child\":true,…}}"}]
```

后端消费点（`lea` 引用实测）：

| 键 | 消费函数 | 位置 |
|---|---|---|
| `filter_only_child` | `0x1401fbb4f` / `0x1401fef80` | 筛选模型 |
| `attribute_to_match` | `0x1401fbc98` / `0x1401fbd26` / `0x1401ff007` | 筛选模型 |
| `checked_item` | `0x1401fb70f` 等 6 处 | 筛选模型 |
| `search_filter` / `date_filter` | `0x14044abbf` / `0x14044ac5c` | 筛选设置对话框 |
| `is_total_include` | `0x14044aaf5` / `0x14044facd` | 筛选设置对话框 |

- `checked_item = -2` → 日期筛选的"预设区间"枚举值（-2 疑为"全部/自定义"）；
- `end_date = 32535014400` → 秒级时间戳（≈ 3029 年，作"无上限"用）。

---

## 6 对 gaea 的落地（复刻口径）

1. **视图 = 列可见性预设**：不必为 9 个内置视图建模，建一个 `ViewPreset { name, visibleColumnIds[], showPert, lockCol, indexWidth, filters[] }` 即可。
2. **列模型 88 字节契约**按 §2 建；`show_type` 用 §2.1 的推导表，**不要**让用户填。
3. **序列化顺序**严格按 §1 的键序（否则与原 `.gdb` 不兼容）。
4. **分组**按 §3 建 0xD8 结构，`time_unit` 复用六时标/时间参数的同一枚举。
5. **筛选**只在需要"区间筛选"的视图启用；`filter_info` 是**二次编码的 JSON 字符串**（不是嵌套对象）——这点必须照抄，否则读不回原文件。

---

## 7 待证

| # | 项 |
|---|---|
| 1 | `checked_item = -2` 的完整枚举表（-1/0/1/2… 各代表什么区间） |
| 2 | `interval_type` 与 `time_unit` 的完整取值域（示例全是 0，需运行期造分组） |
| 3 | `view_mode / view_source / display_mode` 的取值语义（九个内置视图都是 0，说明还有别的来源） |
| 4 | `column_filters` 的执行时机（打开视图时一次性过滤，还是行级动态） |

---

## 8 产物

| 位置 | 内容 |
|---|---|
| `re-zpert/evidence/decomp_viewio.c` | 视图写入 `FUN_1402cb540` + 读取 `FUN_1402ca1b0` |
| `re-zpert/evidence/decomp_viewcol.c` | 单列序列化 `FUN_1402cb1a0`（含 `show_type` 推导） |
| `re-zpert/evidence/decomp_groupby.c` | 分组序列化 `FUN_140207a00` |
| `re-zpert/evidence/view_model.txt` | 视图/列/分组三层布局 + 消费点地址表（grep 版） |
