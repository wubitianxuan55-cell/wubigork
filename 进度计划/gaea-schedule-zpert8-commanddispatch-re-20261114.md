# 斑马进度 ZPert8 · 命令派发机制（前端命令表 → 后端注册制）· 2026-11-14

> **动机**：前端逆向拿到 916 条"命令/埋点注册表"（`region_140017b00.txt`）与 602 条
> 「命令名 ↔ 动作 GUID」映射（`region_140014e30.txt`），但**命令点下去后端做了什么**没人看过。
> 本文把这个缺口填掉，并**纠正一处数字**。
>
> **样本**：`C:\Program Files\ZebraTech\ZPert8\ZPert.exe`（7.0.706.0）
> **方法**：字符串表定位 → `lea`/`call` 引用扫描 → 反编译 → 顺调用链推到绑定点。

---

## 0 一页结论

| # | 结论 | 强度 |
|---|---|---|
| 1 | **命令表是纯数据，不含任何函数指针**：`FUN_140017b00`（34 KB）整个函数里只有 **1 处代码地址赋值**，还是安全 cookie（`local_18[1] = 0x140017b22`） | 实测 |
| 2 | **动作表 299 条** `{内部名, 动作GUID}`，由 `FUN_140014e30` 构建，收进全局 `std::map` `DAT_141126b38`（节点 0x60：键 +0x20、值 +0x40） | 实测 |
| 3 | **查找器 `FUN_14032a3b0`（216 B）做双映射查找**：先查 `DAT_141126b90`，未命中再查 `DAT_141126b38`，返回对应字符串 | 实测 |
| 4 | **它全程序只有 1 个调用方** → `FUN_140576bc0`（动作工厂，2788 B），即"名字→GUID"只在这一条路上发生 | 实测 |
| 5 | ~~动作工厂在 页对象+0x2438 的 map 里查条目~~ → **已更正（见 §6 勘误）**：`+0x2438` 是 **`ZQtPertDoc`（文档对象）的一个 QString 字段**，`FUN_1400db990`/`FUN_1400ddfb0` 是 **QString 移动/拷贝赋值**，不是 map 查找 | 已更正 |
| 6 | `+0x2438` 全 `.text` 只有 **2 处**访问：`0x1405a2571`（`ZQtPertDoc` 构造器置空）、`0x140576c56`（动作工厂写入 GUID） | 实测 |
| 7 | **派发是"注册制"，不是表驱动**：命令键↔处理函数没有静态对应表，取决于哪段代码把 handler 注册进了哪个页对象 | 推断（强） |
| 8 | **数字纠正**：`region_140017b00` 的 **916 是字符串引用条数，不是命令条数**；去重后命令 ID 263 个、命令键 89 个，与动作表 **299 条**同量级 | 实测 |
| 9 | **`FUN_140576bc0` 是"动作埋点"而非动作工厂**：命令执行时由 handler 自己调用，把命令键→GUID 写进 `doc+0x2438` 并经 `ZGatewayServer` 上报（§7） | 实测 |
| 10 | **命令 → 处理函数没有总表**：派发在编译期绑死在各自 handler 里；映射的正确取法是"扫埋点调用方 + 其命令键字面量"（§7.3） | 实测 + 推断 |

---

## 1 三层结构

```
① 命令表（FUN_140017b00，916 个字符串引用）
     { 路径, 命令键, 命令ID }            ← 纯元数据，无实现
                 ↓  (供功能区/埋点/快捷键/自定义引用)
② 动作表（FUN_140014e30，299 条）
     { 内部名, 动作GUID"5000000000…" }   ← 全局 std::map DAT_141126b38
                 ↓  (FUN_14032a3b0 双映射查找)
③ 动作工厂（FUN_140576bc0）
     名字 → GUID → 页对象+0x2438 map → entry+0x10（handler 槽）
```

### 1.1 命令表为什么"没有实现"

反编译 `FUN_140017b00` 的完整统计：

```
code-addr assignments: 1
    = 0x140017b22;   ← 安全 cookie（函数入口后一条），不是函数指针
```

916 个引用的目标全是 `.rdata` 里的字符串。条目构造形态（实测样本）：

```c
FUN_1400decb0(local_e178, &DAT_1408e77f0, 7);   // 命令键 "文件-打开向导"（宽字符）
FUN_1400da890(local_dcb8, "10001", 5);           // 命令ID（窄字符）
FUN_1400decb0(local_dc98, &DAT_1408e77dc, 2);    // 路径段 "文件"（宽字符）
FUN_140186570(local_71b8, local_e178);           // push 字段1
FUN_1400ddfb0(local_7198, local_dcb8);           // push 字段2
FUN_140186570(local_7178, local_dc98);           // push 字段3
```

→ 三个 QSTRING，没有第四个字段。**命令表只回答"这个命令叫什么、挂在哪、ID 多少"，不回答"做什么"。**

### 1.2 动作表 299 条

`FUN_140014e30` 尾部：

```c
local_4ae8 = local_4ad8;
local_4ae0 = local_18;
FUN_1403291e0(&DAT_141126b38, &local_4ae8);          // 整表插入 map
_eh_vector_destructor_iterator_(local_4ad8, 0x40, 299, FUN_1403293e0);   // 299 × 0x40
atexit(FUN_14078e850);
```

每条用 `FUN_140328890(slot, 名称串, GUID串)` 装填，**299 条**。
节点 0x40 字节 = 两个 QString（各 0x20）：名称 + GUID。

### 1.3 双映射查找

```c
undefined8 *FUN_14032a3b0(undefined8 *out, undefined8 key) {
  FUN_1401e0260(&DAT_141126b90, local_28, key);       // 先查 ①
  if (*(char *)(local_18 + 0x19) == '\0') {           // 命中？
    cVar1 = FUN_1401e19a0(&DAT_141126b90, key, local_18 + 0x20);
    if (cVar1 == '\0' && lVar2 != DAT_141126b90) goto LAB_14032a404;
  }
  FUN_1401e0260(&DAT_141126b38, local_28, key);       // 再查 ②（动作表）
  if (*(char *)(local_18 + 0x19) == '\0') {
    cVar1 = FUN_1401e19a0(&DAT_141126b38, key, local_18 + 0x20);
    if (cVar1 == '\0' && local_18 != DAT_141126b38) {
LAB_14032a404:
      FUN_1400ddfb0(out, lVar2 + 0x40);               // 拷出值（+0x40）
      return out;
    }
  }
  ... return 空串;
}
```

- `DAT_141126b90` / `DAT_141126b38` 都是 `std::map<QString,QString>`（`FUN_1401e0260` = find，节点红黑标记在 `+0x19`）；
- 先查的那张（`b90`）优先 → 说明存在**覆盖表**（可被运行期注册的内容盖住静态动作表）；
- 两张表都未命中 → 返回空串。

**这是全程序唯一"名字→GUID"的入口**（`callers=1`）。

### 1.4 动作工厂：拿不到 handler 就不建动作

```c
void FUN_140576bc0(longlong page, undefined8 name) {
  ...
  uVar9 = FUN_14032a3b0(local_240, name);              // 名字 → GUID
  lVar10 = FUN_1400db990(page + 0x2438, uVar9);        // GUID → 页对象内的条目
  ...
  if (*(longlong *)(lVar10 + 0x10) == 0) return;        // ★ handler 槽为空 → 放弃
  ...
}
```

`entry+0x10` 就是 handler 槽。**为空则这个命令在界面上不会被创建**（或创建了也不触发）。

全 `.text` 里 `lea ..., [reg+0x2438]` **只有 1 处**（`0x140576c56`），即这张"GUID → 条目"的 map
只在取动作时被读；写入发生在页签构建期（由各 feature 模块注册）。

---

## 2 数字纠正：916 不是命令数

`region_140017b00.txt` 共 917 行（含首行标题）＝ **916 个字符串引用**。按首次出现形态统计：

| 形态 | 条数 | 去重 |
|---|---|---|
| 纯 5 位数字（命令 ID / 上级 ID） | 503 | **263** |
| 含 `-` 的命令键 | 135 | **89** |
| 其余（路径段、显示名） | 余下 | — |

命令 ID 首位分布（**含重复引用**）：

```
1:58   2:189  3:60   4:59   5:9   6:32   7:22   8:15   9:58   0("00007"):1
```

- 与前端报告的"ID 分段"结论吻合（1 文件 / 2 计划 / 3 资源 / 4 管控 / 5 帮助 / 6 导航 / 7 检查 / 8 关系 / 9 授权云 AI）；
- 但 **263 个去重 ID / 89 个去重键** 才是真实命令规模，和**动作表 299 条**同量级；
- 所以"916 条命令"应改为"**916 个字符串引用，约 263–299 条命令**"。

---

## 3 派发为什么是"注册制"

三条互相印证的证据：

1. 命令表**无函数指针**（§1.1）；
2. "名字→GUID"只有**一个调用方**（§1.3），说明 GUID 不是用来查实现的键，只是转一手的中间标识；
3. handler 槽 `entry+0x10` 的存在与"为空即放弃"的判定（§1.4），说明动作的**可执行性由注册时的填表决定**。

→ 结论：**ZPert 的命令执行 = 各 feature 在构建期把自己的 handler 注册进页对象，
功能区的命令键只负责"引用"它。** 没有一张"命令键/命令ID → 处理函数"的总表可抄。

这对复刻有直接影响：**不能照着命令表 1:1 生成 299 个处理函数占位**，
而应按"功能区条目 → 打开哪个对话框 / 调哪个模型操作"分组。

---

## 4 对 gaea 的落地

1. **命令表照抄为纯元数据**：`{path, cmdKey, cmdId, displayName}`，不含 handler。
2. **动作标识用稳定 GUID**（`5000000000…`）而非命令 ID —— 斑马就是这么做的，好处是改名/挪位置不破坏埋点。
3. **两层 map**：静态动作表 + 运行期覆盖表（先查覆盖表），便于云配置/自定义功能区覆盖内置项。
4. **handler 槽允许为空**：斑马据此实现"未授权的命令不出现"。gaea 可用来做能力门控。

---

## 5 待证

| # | 项 |
|---|---|
| 1 | 页对象 `+0x2438` map 的**写入点**（本轮只找到读点；写入应在页签构建期，需按 `+0x2438` 的 store 扫描或追 `FUN_1400db990` 的兄弟 insert 函数） |
| 2 | `DAT_141126b90`（覆盖表）的填充来源：云配置？自定义功能区？授权档位？ |
| 3 | `entry+0x10` 到底是裸函数指针还是 `std::function`/虚表 —— 决定能否批量枚举 handler |
| 4 | `FUN_14032b400`（103 个调用方的单例取值器）返回的是什么全局管理器 |

---

## 6 勘误（第二轮复核，2026-11-14 续）

上一轮把 `动作工厂 FUN_140576bc0` 里的 `page + 0x2438` 判成「页对象内的 GUID→条目 map」，
**这是错的**。复核证据如下。

### 6.1 `+0x2438` 属于 `ZQtPertDoc`，不是"页对象"

`FUN_1405a2340`（647 B）构造器里：

```c
QObject::QObject(param_1, 0);
pQVar1 = param_1 + 0x10;
FUN_14021fdd0(pQVar1);
*(undefined ***)pQVar1 = ZBasePertDoc::vftable;     // ← 基类
...
*(undefined ***)param_1 = ZQtPertDoc::vftable;       // ← 实际类型
*(undefined ***)pQVar1 = ZQtPertDoc::vftable;
...
*(undefined8 *)(param_1 + 0x2438) = 0;              // ← 就是这里
*(undefined8 *)(param_1 + 0x2448) = 0;
*(undefined8 *)(param_1 + 0x2450) = 0xf;
```

→ `param_1` 是 **`ZQtPertDoc`（计划文档对象）**，`+0x2438` 是它的成员，不是某个"页"的。

### 6.2 `FUN_1400db990` / `FUN_1400ddfb0` 是 QString 赋值，不是 map 操作

```c
longlong *FUN_1400db990(longlong *dst, longlong *src) {   // 139 B
  if (dst != src) {
    if (0xf < (ulonglong)dst[3]) free(*(void**)dst);       // 释放旧缓冲
    dst[3] = 0xf;  dst[2] = 0;  *(char*)dst = 0;           // 置空（SSO 容量 15）
    dst[1] = src[1];  dst[0] = src[0];
    dst[2] = src[2];  dst[3] = src[3];                     // 搬四字
    src[2] = 0;  src[3] = 0xf;  *(char*)src = 0;           // 源置空 → 移动语义
  }
  return dst;
}
```

`FUN_1400ddfb0`（219 B）结构相同但**分配 + memcpy**，是深拷贝版本。
两者都操作 `{d, ?, ?, 容量}` 四字结构、都以 `0xf` 作 SSO 容量、都会 free 旧缓冲
→ **就是 QString 的移动赋值 / 拷贝赋值**，不是 `std::map` 的 find。

### 6.3 修正后的动作工厂语义

```c
uVar9  = FUN_14032a3b0(tmp, name);              // 名字 → GUID（返回 QString 指针）
lVar10 = FUN_1400db990(doc + 0x2438, uVar9);    // ★ 把 GUID 移动赋值进 文档+0x2438
if (*(longlong *)(lVar10 + 0x10) == 0) return;  // 对该 QString 的第三字做判空 → 空则退出
```

即：**动作工厂把解析出的 GUID 记到文档对象的一个 QString 上**（用途待证：疑为
"当前动作上下文"，供埋点/重放/云协作比对使用），而不是"在 map 里查 handler"。

### 6.4 上一轮哪些结论仍然成立

| 结论 | 状态 |
|---|---|
| 命令表是纯数据、无函数指针（`FUN_140017b00` 全函数仅 1 处代码地址，是安全 cookie） | **成立** |
| 动作表 299 条 `{内部名, 动作GUID}` → 全局 map `DAT_141126b38` | **成立** |
| `FUN_14032a3b0` 双映射查找（先 `b90` 覆盖表，再 `b38` 静态表），全程序仅 1 调用方 | **成立** |
| 916 是字符串引用数、命令 ID 去重 263、动作表 299 | **成立** |
| "派发是注册制，不是表驱动" | **成立**（但"注册在哪里"本轮未定位） |
| ~~`+0x2438` 是 GUID→条目 map，`entry+0x10` 是 handler 槽~~ | **推翻** |
| ~~handler 槽为空 → 不建动作~~ | **推翻**（那是 QString 判空，不是 handler 判空） |

### 6.5 下一步该往哪查（纠正后的路线）

1. **别再顺着 `+0x2438` 追** —— 它不是 handler 表。
2. 复核 `FUN_140576bc0` 内 `FUN_140259ce0(param_1 + 0x10)` → `(**(*plVar8 + 0x78))(plVar8)`
   这一串：那是取"当前文档/视图"的虚函数调用（vtable+0x78），
   **真正的 action 创建应在该虚函数返回对象之后的代码里**（`FUN_14029c250(local_a0, lVar7)` 附近）。
3. 从 `FUN_14052d7d0(&PTR_vftable_1410e51c0, ...)` + `__RTDynamicCast(..., ZGatewayServer)` 一段看，
   工厂里还牵扯网关/云服务，说明这个函数不只是建按钮 —— **名字取错了，它更可能是"动作上报/埋点"路径**。
4. 若目标是"命令 → 处理函数"，更稳的入口是**页签构建器**（`FUN_14058b080` 等 5 个）
   调用的 `FUN_1405ead40` / `FUN_140594e30`，那里才是 QAction 与槽的连接点。
## 7 第三轮：`FUN_140576bc0` 是**埋点**，不是动作工厂（导出族实证）

顺着页签 helper 追下去，两个 helper 都不是连接点：`FUN_1405ead40` 用
`xlCreateXMLBookW`（libxl）做 Excel 写盘，`FUN_140594e30` 用 `QPainter/QPixmap` 画图。
**它们是被调用的业务代码，不是绑定器。**

真正的位置反而在"调用方"那一侧。

### 7.1 6 个调用方全是"导出"命令的实现

`FUN_140576bc0` 全程序只有 **6 个调用方**：

| handler | 特征 API | 实际职责 |
|---|---|---|
| `FUN_14058b080` | `QFileDialog::getSaveFileName` + `xlCreateXMLBookW` | **导出 Excel**（libxl 写 xls/xlsx） |
| `FUN_14058bfa0` | `QSvgGenerator` + `setFileName` + `save` | **导出 SVG** |
| `FUN_14058c340` | `QPrinter` + `setOutputFileName` + `setOutputFormat` | **导出 PDF / 打印到文件** |
| `FUN_14058c610` | `getSaveFileName` + `QDir::remove` + `absoluteFilePath` | **导出（先删旧文件再写）** |
| `FUN_140594890` | `QFile` + `QMimeDatabase::defaultTypeFor` | **按 MIME 选格式导出** |
| `FUN_1405e1189` | `FUN_1405b5450` / `FUN_1405dbc90` 建向导 | **导出向导** |

每个 handler 的**第一件事**都是同一句（字面量实测）：

```c
FUN_1400decb0(local_XX, &DAT_1408e7888, 5);   // "文件-导出"
FUN_140576bc0(doc, local_XX);                 // ← 记录动作
```

`DAT_1408e7888` 解码 = **`文件-导出`** —— 正是命令表里的**命令键**。

### 7.2 因此 `FUN_140576bc0` 的真实语义

```c
void FUN_140576bc0(doc, 命令键) {
  命令键 ──FUN_14032a3b0──> 动作GUID("5000000000…")
  GUID   ──FUN_1400db990──> 写入 doc+0x2438（QString，即"当前动作"）
  （另有 FUN_14052d7d0 + __RTDynamicCast(..., ZGatewayServer) → 云侧上报路径）
}
```

→ **它是"动作埋点/上报"，在命令执行时被 handler 自己调用**，不是"建按钮/绑槽"。

这也解释了之前所有对不上的现象：

| 现象 | 现在说得通 |
|---|---|
| 名字→GUID 查询只有 1 个调用方 | 只有埋点需要把命令键翻成稳定 GUID |
| 工厂里出现 `ZGatewayServer` 动态转换 | 埋点要经网关上报（云侧统计） |
| `+0x2438` 是 doc 的 QString 且被写成 GUID | 记"最近一次动作"，供重放/协作比对 |
| 6 个 handler 都传同一个 `文件-导出` | 一个命令键对应多条实现路径（6 种导出格式） |

### 7.3 命令 → 处理函数：没有总表，靠 handler 自报

由此得出确定结论：

- **命令键只是标识**（命令表里纯数据）；
- **实现分散在各 handler 函数里**，每个 handler 在开头用一句 `FUN_140576bc0(doc, 命令键)` **自报家门**；
- **不存在**"命令 ID/键 → 函数指针"的总表（第一轮已证命令表无函数指针，本轮补上原因：派发在编译期就绑死在 handler 里了）。

所以"命令→handler 映射"的正确取法是：
**扫 `FUN_140576bc0` 的全部调用方，取每个调用方传入的命令键字面量**。
当前只找到 6 个 → 说明**埋点只覆盖"导出"一族**（其余命令可能走别的埋点函数或不上报），
这也与前端报告里"11xxxxx/12xxxxx AI 交互埋点族"是**另一套**埋点相符。

### 7.4 本轮顺带取到的产品事实（导出族）

| 文案 | 含义 |
|---|---|
| `本计算机没有检测到MS Project，仅能导出XML格式，该格式可以在Project中转换打开` | **导出支持 MS Project XML**，但需要本机装 Project 才能转 MPP；否则降级为 XML |
| `导出失败，请尝试重新导出` | 统一失败提示 |
| `创建文件失败！请检查是否有可写权限。` | 写盘权限失败单独提示 |
| `创建临时文件失败` | 有临时文件中转（先写临时再改名） |
| `xlCreateXMLBookW(..., L"windows-…")` | Excel 侧按 Windows 代码页写，中文不乱码 |

→ 复刻口径：**导出必须给 MS Project XML 通道**，且"本机有无 Project"要作为能力探测而不是硬失败。
## 8 产物

| 位置 | 内容 |
|---|---|
| `re-zpert/evidence/decomp_cmdreg.c` | 命令表构建 `FUN_140017b00`（34 KB，916 串） |
| `re-zpert/evidence/decomp_actions.c` | 动作表构建 `FUN_140014e30`（299 条 {名, GUID}） |
| `re-zpert/evidence/decomp_actlookup.c` | 双映射查找 `FUN_14032a3b0` |
| `re-zpert/evidence/decomp_actuse.c` | 动作工厂 `FUN_140576bc0`（handler 槽判定） |
| `re-zpert/evidence/decomp_ribbon.c` | 两个页签构建器 `FUN_14058b080` / `FUN_14058bfa0` |
| `re-zpert/evidence/decomp_page2438.c` | `ZQtPertDoc` 构造器 `FUN_1405a2340`（`+0x2438` 来源） |
| `re-zpert/evidence/decomp_mapops.c` | `FUN_1400db990` / `FUN_1400ddfb0`（QString 移动/拷贝赋值） |
| `re-zpert/evidence/decomp_ribbonhelper.c` | `FUN_1405ead40`（libxl 导出）/ `FUN_140594e30`（QPainter 绘制） |
| `re-zpert/evidence/decomp_handlers.c` | 导出族 6 个 handler + 埋点调用点 |
| `re-zpert/evidence/cmd_dispatch.txt` | 三层结构、勘误、埋点定位与全部地址（grep 版） |
