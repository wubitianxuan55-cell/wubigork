// Package comhelper 把 dsh-office 的 COM helper 移植进 gaea：一个常驻小进程，
// 通过 COM 附着到正在运行的 Word / Excel / PowerPoint / WPS，让文档在用户窗口里
// 被实时编辑（改动即时上屏、可协同、格式不走样）——这是无头 OOXML 手术管线
// （docxedit/pptxedit/xlsxedit）之外的「实时协同档」，二者互补不替代。
//
// 上游 provenance（2026-11-13 调研批，docs/gaea-dsh-office-distill-research-2026-11.md）：
//
//   - 源=github.com/copylee711/dsh-office v1.2.0（2069af2d），MIT 许可。
//   - helper/OfficeHelper.cs 逐字节照搬（本目录 OfficeHelper.cs，与上游
//     cmp 一致；升级=重新拷贝+跑协议测试），UPSTREAM-LICENSE 为其 MIT 全文。
//   - 宿主侧（上游 src/ 的 TS）不照搬：工具层按 gaea 工具框架重写于
//     internal/app/office_com_tools.go，协议层（编译+spawn+行协议）重写于本包。
//
// 与上游的三处有据偏离（其余协议逐字节对齐）：
//
//  1. 编译加 /codepage:65001——上游裸编译依赖系统 ANSI 代码页，源文件无 BOM
//     且含中文字面量（Points 的 磅/厘米/毫米/英寸 正则、卡片文案），非中文
//     locale 或未来 Windows 默认页变化时会静默 mojibake；强制 UTF-8 读入
//     在任何 locale 下都正确。
//  2. 波1 不带 math 预转腿（上游 omml.ts 依赖 npm temml；Go 侧无等价物）。
//     MS Word 公式走 Word 自带 LaTeX 解析器完整可用；WPS 公式（helper 需要
//     预转好的 OMML）为已知缺口挂波2。edit 调用不传 math 字段即走该缺省。
//  3. 体验开关暂不接 gaea 设置（波3）：follow=true / typing=true /
//     card=false / silent=false / suite=auto 硬编码在工具层（上游默认值，
//     仅 card 上游默认 true——迷你卡片挂拍板池第 3 项波二）。
//
// 生命周期：helper 进程随 gaea——父进程退出（哪怕崩溃）stdin 管道关闭，
// helper ReadLine 返回 null 自行退出（上游同语义），无需 JobObject 兜底；
// spawn 带 proc.HideWindow 防黑框。单命令串行（COM 忙等由 helper 的
// MessageFilter + BUSY 错误码收口），Go 侧调用互斥串行化。
//
// 编译：运行时用 Windows 自带的 .NET Framework csc.exe（Win10/11 自带），
// 产物按源码 sha256 前 12 位缓存在 %LOCALAPPDATA%\gaea\office-helper\，
// 升级 helper 源码即自然重编（与上游同策略，缓存目录换 gaea 命名）。
package comhelper
