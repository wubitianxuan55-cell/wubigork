package tool

// PublishGuard — P1b 发布边界（2026-11-09 众神殿 pan_1794234593182 裁决 action-4）。
//
// 办公侧主对话 bash 无 OS 沙箱（gaea_handler.go 显式 Sandbox.Bash="off"），扩展名
// 字符串扫描可被重定向/脚本/改名绕过（裁决风险条），因此硬禁落在「工作区发布边界」：
// Begin 对工作区根做 size+mtime 快照 → 工具执行 → Scan 比对出 durable 档
// （v1=docx/xlsx/pptx 三格式，D1 口径）的新增/修改 → Quarantine 移出正式工作区，
// 调用方 fail-loud 回显期望路径集（备忘 :23 fail-loud 继承）。
//
// 三分档口径（备忘 :21 继承）：durable 硬禁（本文件）；derived（渲染 PDF/PNG 挂父
// 哈希）与 ephemeral（结束查残留）属 P2/后续档，本文件不冒充全量治理。
//
// 边界（action-2 spike 验证对象）：前台路径由调用方在执行后 Scan；后台 bash 跨回合
// 存活，须在 job 完成回调（jobs.Manager.SetOnCompletion）再 Scan——前台 Scan 对
// 「尚未落盘的后台写」恒为 clean，spike 实证。

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// DefaultDurableExts v1 durable 档扩展名（D1：v1 只承诺 Office 三格式；非 Office
// durable（bash 写 .md/.csv 交付物）明确列范围外风险，P3/P4 再按交付意图扩展）。
func DefaultDurableExts() map[string]bool {
	return map[string]bool{".docx": true, ".xlsx": true, ".pptx": true}
}

// PublishViolation 一次 Scan 发现的 durable 越界写。
type PublishViolation struct {
	Path    string // 工作区内绝对路径
	RelPath string // 相对所属根的路径（ToSlash 规范化）
	Root    string // 所属根
	Kind    string // "created"（快照中不存在）/ "modified"（size 或 mtime 变化）
	Size    int64
}

// PublishGuard 工作区 durable 写发布边界。单次生命周期：Begin →（执行）→ Scan →
// Quarantine。非并发安全：同一线程内顺序调用；后台路径可在 job 完成回调线程
// 复用同一实例（Begin 在启动时做过快照即可）。
type PublishGuard struct {
	roots          []string
	quarantine     string
	durable        map[string]bool
	base           map[string]publishFileState
	baseDirs       map[string]bool // 执行前已存在的目录集（目录级原子隔离判定）
	beginTime      time.Time       // Begin 时刻（随迁旧文件判据：mv 不改 mtime）
	source         string          // D6 账本来源标注（foreground/background）
	commandPreview string
	lastLedger     *quarantineLedger
	// lastTouched 最近一次 Scan 见到的全部 durable 新增/修改（含豁免的过程件）——
	// 调用方拿它做事后体检（脚本写出的 xlsx 有没有缓存值），与隔离判定无关。
	lastTouched []PublishViolation
}

// WithSource 标注账本来源与命令预览（D6：归属证据——谁在何时因何命令隔离）。
func (g *PublishGuard) WithSource(source, command string) *PublishGuard {
	g.source = source
	g.commandPreview = command
	return g
}

// LastQuarantineDest 最近一次隔离的目的地 stamp 目录（fail-loud 回显用）。
func (g *PublishGuard) LastQuarantineDest() string {
	if g.lastLedger == nil {
		return ""
	}
	return filepath.Join(g.quarantine, g.lastLedger.rec.Stamp)
}

type publishFileState struct {
	size  int64
	mtime time.Time
}

// NewPublishGuard roots 为受守卫的工作区根（绝对路径）；quarantine 为越界件
// 隔离目录（建议落在工作区内 .gaea/ 下，保证 Rename 同卷）；durable 为空时取
// DefaultDurableExts。
func NewPublishGuard(roots []string, quarantine string, durable map[string]bool) *PublishGuard {
	if durable == nil {
		durable = DefaultDurableExts()
	}
	return &PublishGuard{
		roots:      append([]string(nil), roots...),
		quarantine: quarantine,
		durable:    durable,
	}
}

// Begin 执行前快照。对每个根做整树遍历，记录 durable 扩展名文件的 size+mtime。
// 目录不进快照（durable 判据只看文件），遍历剪掉软链目录防环、隔离目录防自查、
// `~$` Office 锁件目录防误捕。
func (g *PublishGuard) Begin() error {
	g.base = make(map[string]publishFileState)
	g.baseDirs = make(map[string]bool)
	g.beginTime = time.Now()
	for _, root := range g.roots {
		if err := g.walk(root, func(path string, st fs.FileInfo) {
			g.base[path] = publishFileState{size: st.Size(), mtime: st.ModTime()}
		}); err != nil {
			return err
		}
		// 目录集快照（目录级原子隔离判定用）：执行后新出现的目录=本批目录级
		// 移动/新建的候选面（P1-12：目录骨架搬走、durable 被逐件隔离=中间态）。
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err == nil && d.IsDir() && !g.withinQuarantine(path) {
				g.baseDirs[path] = true
			}
			return nil
		})
	}
	return nil
}

// Scan 执行后比对。返回 durable 档的新增/修改清单；不存在越界时返回 nil。
// 过程件豁免（2026-11-21 反馈 P0-1 建议三）：探针/草稿类不是交付件——
// `_draft/` 段路径与文件名含探针字样（probe/test/tmp/draft/草稿/临时）不判
// durable（实测往 _draft/probe/ 写一个探针 docx 也被拦）。
func (g *PublishGuard) Scan() ([]PublishViolation, error) {
	if g.base == nil {
		return nil, fmt.Errorf("publishguard: Scan before Begin")
	}
	cur := make(map[string]publishFileState)
	for _, root := range g.roots {
		if err := g.walk(root, func(path string, st fs.FileInfo) {
			cur[path] = publishFileState{size: st.Size(), mtime: st.ModTime()}
		}); err != nil {
			return nil, err
		}
	}
	var out, touched []PublishViolation
	for path, st := range cur {
		before, existed := g.base[path]
		if !existed {
			if v := g.violationFor(path, st, "created"); !v.exempt() {
				out = append(out, v)
			}
			touched = append(touched, g.violationFor(path, st, "created"))
			continue
		}
		if before.size != st.size || !before.mtime.Equal(st.mtime) {
			if v := g.violationFor(path, st, "modified"); !v.exempt() {
				out = append(out, v)
			}
			touched = append(touched, g.violationFor(path, st, "modified"))
		}
	}
	sort.Slice(touched, func(i, j int) bool { return touched[i].Path < touched[j].Path })
	g.lastTouched = touched
	return out, nil
}

// Touched 最近一次 Scan 见到的全部 durable 新增/修改（含 .gaea/work/ 等豁免面）。
// 体检用：豁免的过程件不隔离，但脚本写出的 xlsx 没缓存值这类「合法但退化」要在
// 回执里提一句（2026-11-27 AI 反馈 #7）。
func (g *PublishGuard) Touched() []PublishViolation {
	return append([]PublishViolation(nil), g.lastTouched...)
}

// PublishExempt 报告一个工作区相对路径是否属过程件豁免面（.gaea/work/、_draft/、
// 探针字样等）。调用方拿它判断「这里是不是正式交付面」——例：move_file 覆盖前
// 只给正式交付件留版本快照，过程件不留（2026-11-27 AI 反馈 #4）。
func PublishExempt(relPath string) bool {
	return PublishViolation{RelPath: filepath.ToSlash(relPath)}.exempt()
}

// QuarantinePathOf 一件越界件隔离后的落点：工作区根内给工作区相对写法，否则给
// 绝对路径（逐件隔离与整目录隔离同式）。
func QuarantinePathOf(stamp string, v PublishViolation) string {
	return quarantineFilePath(stamp, v)
}

// publishExemptNameTokens 探针/草稿文件名特征（小写匹配）。
var publishExemptNameTokens = []string{"probe", "test", "tmp", "draft", "草稿", "临时"}

// exempt 报告本越界件是否属过程件豁免面（P0-1 建议三）：路径含 `_draft/`
// 段或 `.gaea/work/`，或文件名含探针字样。豁免=不隔离不拦（过程件不是交付件）；
// 真交付物落这些名字属命名自误，非本边界职责。
func (v PublishViolation) exempt() bool {
	rel := strings.ToLower(v.RelPath)
	segs := strings.Split(rel, "/")
	for i, seg := range segs {
		switch seg {
		case "_draft", "_probe":
			return true
		case ".gaea":
			// .gaea/work/ 过程目录豁免（2026-11-26 造价反馈三批，百锦路 #1 /
			// 太原 #10）：系统提示词明令「中间表格」写 .gaea/work/<任务名>/，本边界
			// 却把脚本落在那里、或改过那里的 xlsx 一律隔离——提示词与闸门自相
			// 矛盾，两场各被拦两次。过程目录不是交付面：正式路径照拦，move_file
			// 仍是 durable 落进正式名的唯一出口。
			if i+1 < len(segs) && segs[i+1] == "work" {
				return true
			}
		}
	}
	base := rel
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		base = rel[i+1:]
	}
	for _, tok := range publishExemptNameTokens {
		if nameHasToken(base, tok) {
			return true
		}
	}
	return false
}

// nameHasToken 报告小写文件名 base 是否含特征词 tok。ASCII 特征词要求左侧是
// 词边界（开头或非字母）：直接子串匹配会把 latest.xlsx、contest.docx 里的
// test 当成探针件放过（2026-11-26 对码发现）。中文特征词没有分隔符，仍按
// 子串算。
func nameHasToken(base, tok string) bool {
	if tok == "" || !strings.Contains(base, tok) {
		return false
	}
	if tok[0] >= utf8.RuneSelf {
		return true
	}
	for i := 0; ; {
		j := strings.Index(base[i:], tok)
		if j < 0 {
			return false
		}
		at := i + j
		if at == 0 || !isASCIILetter(base[at-1]) {
			return true
		}
		i = at + 1
	}
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// Quarantine 把越界件移出正式工作区（隔离目录下按时间戳子目录保留相对结构），
// 返回已移动的原始绝对路径（调用方 fail-loud 回显）。移动失败（文件被占用等）
// 时 error 非空且未移动件仍留在正式工作区——调用方必须把 error 当硬失败上报。
// D6（dsh-ops 蒸馏 2026-11-10）：先记账后放句柄——任何 MkdirAll/Rename 之前
// 先落 pending 账本记录，移动后补终态；账本失败不阻断隔离（硬边界优先），
// 该 stamp 视为无主永不自动删。
func (g *PublishGuard) Quarantine(vs []PublishViolation) ([]string, error) {
	var moved []string
	var errs []string
	led := beginQuarantineRecord(g.quarantine, g.source, g.commandPreview, vs)
	g.lastLedger = led
	stamp := led.rec.Stamp
	// 目录级原子隔离（2026-11-21 反馈 P1-12：mv 目录→骨架搬走+durable 被逐件
	// 隔离=「半完成」中间态，事后要写脚本三方对账）。判定三要件：①violation
	// 的父目录是本批新建（不在 Begin 目录快照）；②目录内全部 durable 文件都
	// 属本批越界件；③目录内非 durable 伴行件的 mtime 早于 Begin（mv 不改
	// mtime=随迁旧文件；新建目录里现写的 md 是新文件，不该被整目录带走——
	// 进行中的编稿不是目录移动）。三要件齐→整个目录一次搬进隔离区（保持相对
	// 结构，不留空壳目录）；否则维持逐件。
	claimed := map[string]bool{}
	vset := map[string]bool{}
	for _, v := range vs {
		vset[v.Path] = true
	}
	dirMoved := func(dir string) ([]string, bool) {
		if g.baseDirs[dir] {
			return nil, false // 既有目录：非整目录新建形态
		}
		var files []string
		ok := true
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			st, serr := d.Info()
			if serr != nil {
				ok = false
				return nil
			}
			files = append(files, path)
			isDurable := g.isDurable(path)
			switch {
			case isDurable && !vset[path]:
				ok = false // durable 却不在越界集（快照前已有？防御面）
			case !isDurable && !st.ModTime().Before(g.beginTime):
				ok = false // 非随迁旧文件=目录内现写的进行中内容，不整搬
			}
			return nil
		})
		if !ok || len(files) == 0 {
			return nil, false
		}
		return files, true
	}
	// 按路径深度升序尝试（外层目录先整搬，内层 violation 随之外搬后自然 claimed）。
	sorted := append([]PublishViolation(nil), vs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	for _, v := range sorted {
		if claimed[v.Path] {
			continue
		}
		parent := filepath.Dir(v.Path)
		if files, ok := dirMoved(parent); ok {
			rel, rerr := filepath.Rel(g.roots[0], parent)
			if rerr == nil && !strings.HasPrefix(rel, "..") {
				dst := filepath.Join(g.quarantine, stamp, filepath.FromSlash(filepath.ToSlash(rel)))
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err == nil {
					if err := os.Rename(parent, dst); err == nil {
						moved = append(moved, files...)
						for _, f := range files {
							claimed[f] = true
						}
						continue // 整目录原子搬完
					}
				}
			}
			// 整目录搬失败（跨卷等）→ 落回逐件路径，如实报错面不变。
		}
		dst := filepath.Join(g.quarantine, stamp, filepath.FromSlash(v.RelPath))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", v.Path, err))
			continue
		}
		if err := os.Rename(v.Path, dst); err != nil {
			// 跨卷/占用兜底：copy+remove。remove 失败=越界件仍在正式工作区，如实报错。
			if cerr := copyPublishFile(v.Path, dst); cerr != nil {
				errs = append(errs, fmt.Sprintf("%s: rename=%v copy=%v", v.Path, err, cerr))
				continue
			}
			if rerr := os.Remove(v.Path); rerr != nil {
				errs = append(errs, fmt.Sprintf("%s: copy 成功但移除失败 %v（越界件仍在正式工作区）", v.Path, rerr))
				continue
			}
		}
		moved = append(moved, v.Path)
		claimed[v.Path] = true
	}
	led.finish(moved, errs)
	if len(errs) > 0 {
		return moved, fmt.Errorf("quarantine 部分失败：%s", strings.Join(errs, "; "))
	}
	return moved, nil
}

// ExpectedPathsNote fail-loud 提示文案（备忘 :23：回显期望路径集）。expected 为
// 声明写路径（bash 场景为空=无声明）；dest（可选）为隔离目的地 stamp 目录
// （D6：会话 JSONL 成为第二份归属证据，用户找得到文件）；输出直接进工具回执。
func ExpectedPathsNote(violations []PublishViolation, expected []string, dest ...string) string {
	var b strings.Builder
	b.WriteString("发布边界拦截：以下 Office 交付文件（durable 档）不得经本工具写入正式工作区，已隔离：")
	stamp := ""
	if len(dest) > 0 {
		stamp = dest[0]
	}
	for _, v := range violations {
		b.WriteString("\n  - " + v.RelPath + "（" + v.Kind + "）")
		if stamp != "" {
			// 2026-11-26 造价反馈三批（百锦路 #1、太原 #10）：回执此前只给批次
			// 目录，而隔离件落在批次目录下**镜像原相对路径**的子目录里——模型
			// 照回执路径取回直接 source not found，再花三次调用（ls/ls/find）
			// 才定位。逐件给完整落点；在工作区根内时给 move_file 能直接用的
			// 工作区相对写法。
			b.WriteString("\n    隔离件：" + quarantineFilePath(stamp, v))
		}
	}
	if stamp != "" {
		b.WriteString("\n隔离位置：" + stamp)
	} else {
		// 实测反馈⑤：dest 缺席时不得只剩「已隔离」裸句——给出可定位的兜底。
		b.WriteString("\n隔离位置：本次回执未取到具体 stamp 目录；隔离件按时间戳子目录保存在工作区 .gaea/publish-quarantine/ 下，会话 JSONL 隔离账本（D6）有完整路径与命令预览。")
	}
	if len(expected) > 0 {
		b.WriteString("\n声明写路径集：" + strings.Join(expected, "; "))
	}
	// 出路指引恒显（2026-11-20 实测修正）：此前 expected 非空时本句被「声明
	// 写路径集」替换——bash 带声明路径的拦截只见归因不见出路，实测用户绕行
	// 多轮才找到合规入口。声明路径是归因信息，与合规出路不互斥，两句都给。
	b.WriteString("\ndurable 交付物只能经结构化入口：office_generate（从零生成）或 docx_apply/pptx_apply/xlsx_apply（编辑）。")
	// 实测反馈⑤补两件：能不能取回、允许的落点是什么（此前用户试错一轮才找到
	// 临时目录中转）。隔离件是原件（Quarantine 只 move 不改内容）。
	// 2026-11-26 造价反馈三批改写：原句「move_file 移到工作区外暂存位（如临时
	// 目录）」实测走不通——move_file 的终点必须在写根内，两场都被 path access
	// denied 拦。改成在工作区内取回的可行序列（.gaea/work/ 已是过程目录豁免面）。
	b.WriteString("\n取回：隔离件是原件。在工作区内取回——move_file 的终点必须在写根内，移到工作区外会被拦：")
	b.WriteString("\n  move_file source=<上面的隔离件路径> destination=.gaea/work/<文件名>")
	b.WriteString("\n.gaea/work/ 是过程目录（脚本在其中读写 xlsx 不受本边界限制）；复核、改完后再 move_file 到正式名。")
	b.WriteString("\n允许落点：md/txt/png 等非 durable 档不受本边界限制；durable 档（docx/xlsx/pptx）走上述结构化入口。")
	// 2026-11-21 造价门实测 2.1：自由脚本（openpyxl/python-docx 等）生成 durable
	// 是合法场景，但合规路径此前只能试错撞出——Temp→copy→move_file。明示出口；
	// 同时告示 request_permission 的路径授权与本边界是两个独立闸门（实测用户
	// 批准了持久规则仍被拦，误以为授权无效）。
	b.WriteString("\n自由脚本生成 durable 的合规路径：直接落 .gaea/work/（过程目录，不受限），或先落系统临时目录（或非 durable 扩展名如 .xlsx.tmp），再用 move_file 移入工作区正式名——move_file 是 durable 落盘的合法出口。")
	b.WriteString("\n注意：request_permission 批准的路径规则不改变本边界（独立闸门）——durable 交付物一律走结构化入口或上述 move_file 路径。")
	return b.String()
}

// quarantineFilePath 报告一件隔离件的落点：批次目录下镜像原相对路径。工作区根
// 在场且隔离目录在根内时给工作区相对写法（move_file / bash 都能直接用），
// 否则给绝对路径。
func quarantineFilePath(stamp string, v PublishViolation) string {
	full := filepath.Join(stamp, filepath.FromSlash(v.RelPath))
	if v.Root != "" {
		if rel, err := filepath.Rel(v.Root, full); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return full
}

// --- internals ---

// publishPrunedDirs 依赖/构建目录剪枝（大仓快照开销控制：node_modules 万级
// 条目会拖慢每次 bash 调用）。这些目录不是交付物落点，其中的 durable 写不做
// 发布边界判据（如实记档的范围裁剪，非漏拦）。
var publishPrunedDirs = map[string]bool{
	".git": true, "node_modules": true, "__pycache__": true,
	".venv": true, "venv": true, "target": true,
}

func (g *PublishGuard) walk(root string, record func(path string, st fs.FileInfo)) error {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 根尚未创建=空快照，合法
		}
		return err
	}
	if !info.IsDir() {
		if g.isDurable(root) {
			record(root, info)
		}
		return nil
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil // 执行期间被删：比对自然缺席
			}
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path == root {
				return nil
			}
			// 隔离目录（防自查）、软链目录（防环）、依赖目录（开销）、~$ Office
			// 锁件目录剪枝。
			if g.withinQuarantine(path) || d.Type()&fs.ModeSymlink != 0 ||
				strings.HasPrefix(name, "~$") || publishPrunedDirs[name] {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if strings.HasPrefix(name, "~$") {
			return nil // Word/Excel/WPS 锁件=ephemeral，不入 durable 判据
		}
		if !g.isDurable(path) {
			return nil
		}
		st, serr := d.Info()
		if serr != nil {
			return serr
		}
		record(path, st)
		return nil
	})
}

func (g *PublishGuard) isDurable(path string) bool {
	return g.durable[strings.ToLower(filepath.Ext(path))]
}

func (g *PublishGuard) withinQuarantine(path string) bool {
	if g.quarantine == "" {
		return false
	}
	rel, err := filepath.Rel(g.quarantine, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func (g *PublishGuard) violationFor(path string, st publishFileState, kind string) PublishViolation {
	v := PublishViolation{Path: path, Kind: kind, Size: st.size}
	for _, root := range g.roots {
		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		v.Root = root
		v.RelPath = filepath.ToSlash(rel)
		break
	}
	return v
}

func copyPublishFile(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
}
