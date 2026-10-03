package project

// project_chapters.go — 章节正文与章节摘要的文件读写（批 28 IN1-01 文件
// 拆分，自 project.go 原位搬移，零逻辑改动）：WriteChapter/ReadChapter 主线
// 与 branch 变体、章节摘要族（含 mainline 口径与 before 查询）、MaxChapter
// 探测、章节文件名解析单源 ParseChapterFileName（批 21 收敛的保留站点
// 口径对照表在此）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/types"
)

// WriteChapter 写章节文件 chapters/NNN.md（自动补零，原子写）
func (m *Manager) WriteChapter(num int, content string) error {
	return fileutil.AtomicWrite(m.ChapterPath(num), []byte(content), 0o600)
}

// ReadChapter 读章节文件
func (m *Manager) ReadChapter(num int) (string, error) {
	data, err := os.ReadFile(m.ChapterPath(num))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ChapterPath 返回章节文件路径 chapters/NNN.md
func (m *Manager) ChapterPath(num int) string {
	return filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d.md", num))
}

// ChapterBranchPath 返回分支章节文件路径 chapters/NNN{a,b,c}.md
func (m *Manager) ChapterBranchPath(num int, branch string) string {
	return filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d%s.md", num, branch))
}

// WriteChapterBranch 写分支章节（原子写）
func (m *Manager) WriteChapterBranch(num int, branch string, content string) error {
	return fileutil.AtomicWrite(m.ChapterBranchPath(num, branch), []byte(content), 0o600)
}

// ReadChapterBranch 读分支章节
func (m *Manager) ReadChapterBranch(num int, branch string) (string, error) {
	data, err := os.ReadFile(m.ChapterBranchPath(num, branch))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteChapterSummary 写章节摘要 chapters/NNN-summary.json
func (m *Manager) WriteChapterSummary(num int, summary *types.ChapterSummary) error {
	return writeJSON(m.ChapterSummaryPath(num), summary)
}

// ReadChapterSummary 读章节摘要
func (m *Manager) ReadChapterSummary(num int) (*types.ChapterSummary, error) {
	return loadJSON[types.ChapterSummary](m.ChapterSummaryPath(num))
}

func (m *Manager) ChapterSummaryPath(num int) string {
	return filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d-summary.json", num))
}

// WriteChapterBranchSummary 写分支章节摘要 chapters/NNN{a,b,c}-summary.json
func (m *Manager) WriteChapterBranchSummary(num int, branch string, summary *types.ChapterSummary) error {
	return writeJSON(m.ChapterBranchSummaryPath(num, branch), summary)
}

// ReadChapterBranchSummary 读分支章节摘要
func (m *Manager) ReadChapterBranchSummary(num int, branch string) (*types.ChapterSummary, error) {
	return loadJSON[types.ChapterSummary](m.ChapterBranchSummaryPath(num, branch))
}

func (m *Manager) ChapterBranchSummaryPath(num int, branch string) string {
	return filepath.Join(m.Dir, "chapters", fmt.Sprintf("%03d%s-summary.json", num, branch))
}

// chapterSummaryWithFile 一次扫描得到的一条章摘要及其来源文件名。
type chapterSummaryWithFile struct {
	File    string // chapters/ 下的摘要文件名（章号由此解析）
	Summary types.ChapterSummary
}

// readChapterSummariesWithFiles 扫描 chapters/ 取全部章摘要及其文件名，按文件名
// 升序（零填充数字 = 章号升序；分支章的字母后缀排在数字之后）。单个文件读失败
// 或 JSON 损坏跳过不中断（摘要缺失是增强面，不因脏档丢掉全库）。
func (m *Manager) readChapterSummariesWithFiles() ([]chapterSummaryWithFile, error) {
	entries, err := os.ReadDir(filepath.Join(m.Dir, "chapters"))
	if err != nil {
		return nil, err
	}

	// 收集匹配的文件名并排序（零填充数字，字典序即数值序）
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "-summary.json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := make([]chapterSummaryWithFile, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(m.Dir, "chapters", name))
		if err != nil {
			continue
		}
		var s types.ChapterSummary
		if json.Unmarshal(data, &s) == nil {
			out = append(out, chapterSummaryWithFile{File: name, Summary: s})
		}
	}
	return out, nil
}

// ReadAllChapterSummaries 一次扫描读取所有章节摘要（替代逐个文件探测）。
// 只返回摘要本身；需要按章号过滤的调用方走 ReadLatestChapterSummaryBefore。
// 注意：结果含分支摘要（NNN{a,b,c}-summary.json）——分支浏览器等需要分支
// 剧情线的调用方用本入口；主线口径（大纲续写前情等）走
// ReadMainlineChapterSummaries，否则分支剧情会被当主线参考注入（N5）。
func (m *Manager) ReadAllChapterSummaries() ([]types.ChapterSummary, error) {
	items, err := m.readChapterSummariesWithFiles()
	if err != nil {
		return nil, err
	}
	summaries := make([]types.ChapterSummary, 0, len(items))
	for _, it := range items {
		summaries = append(summaries, it.Summary)
	}
	return summaries, nil
}

// ReadMainlineChapterSummaries 读取全部**主线**章节摘要（NNN-summary.json），
// 分支摘要（NNN{a,b,c}-summary.json）不参与——主线生成/续写的前情参考里
// 混入分支剧情线会污染走向（N5）。
func (m *Manager) ReadMainlineChapterSummaries() ([]types.ChapterSummary, error) {
	items, err := m.readChapterSummariesWithFiles()
	if err != nil {
		return nil, err
	}
	summaries := make([]types.ChapterSummary, 0, len(items))
	for _, it := range items {
		if mainChapterSummaryNum(it.File) <= 0 {
			continue
		}
		summaries = append(summaries, it.Summary)
	}
	return summaries, nil
}

// ReadLatestChapterSummaryBefore 取章号严格小于 chapterNum 的最近一章摘要；
// 没有合格摘要（含 chapterNum<=1 的第一章）返回 (nil, nil)。
//
// 章号取自**文件名**（chapters/NNN-summary.json / NNN{a,b,c}-summary.json 的前导
// 数字）：ChapterSummary 本身没有章号字段，而按文件名升序读出来的「最后一个」
// 是全书最后一章——生成第 1 章时若拿它当「上一章」，prompt 里会塞进全书结局
// （G5）。故这里显式解析文件名过滤，不用「取最后一个」的兜底。
// chapters/ 目录缺失视为「无摘要」（新项目正常态），不报错。
func (m *Manager) ReadLatestChapterSummaryBefore(chapterNum int) (*types.ChapterSummary, error) {
	if chapterNum <= 1 {
		return nil, nil
	}
	items, err := m.readChapterSummariesWithFiles()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	// 文件名升序 = 章号升序，正向扫描时「最后一个合格者」即最近一章。
	// 只认主线摘要（NNN-summary.json）：分支摘要 NNN{a,b,c}-summary.json 与主线
	// 同章号却排在后面，收进来会把分支剧情当成主线的「上一章」（N5 的放大器）。
	var best *types.ChapterSummary
	for i := range items {
		cn := mainChapterSummaryNum(items[i].File)
		if cn > 0 && cn < chapterNum {
			best = &items[i].Summary
		}
	}
	return best, nil
}

// mainChapterSummaryNum 解析主线章摘要文件名（"006-summary.json" → 6）；
// 分支摘要（"006a-summary.json"）与其它形状一律返回 0（不参与主线口径）。
func mainChapterSummaryNum(name string) int {
	const suffix = "-summary.json"
	if !strings.HasSuffix(name, suffix) {
		return 0
	}
	num := strings.TrimSuffix(name, suffix)
	if num == "" {
		return 0
	}
	for _, r := range num {
		if r < '0' || r > '9' {
			return 0
		}
	}
	return leadDigits(num)
}

// MaxChapterBodyNum 扫描 chapters/ 目录，返回**磁盘上已有正文**的最大章号；
// 一个正稿都没有时返回 0。
//
// 「已有正文」= 主线正稿 chapters/NNN.md 非空，或 v4 场景制下该章已有承载非空
// 正文的场景（只有场景、没有 blob 的章同样算占用）。分支章 NNNa.md 不计入
// 主线章号口径。目录缺失 = 0（新项目正常态）。
//
// 供补下等「续编章号」场景做磁盘侧基准（G2）：只按大纲 OrderIndex 续编会在
// 大纲被「续写」重排后撞上已有正稿。
func (m *Manager) MaxChapterBodyNum() (int, error) {
	return m.maxChapterNumMatching(m.chapterHasBodyContent)
}

// MaxChapterNum 扫描 chapters/ 目录，返回任意章文件（NNN.md / NNNa.md，含空文件）
// 的最大前导章号；无文件返回 0。目录缺失 = 0（新项目正常态）。
func (m *Manager) MaxChapterNum() (int, error) {
	return m.maxChapterNumMatching(func(int) bool { return true })
}

// maxChapterNumMatching 扫描 chapters/，返回满足 keep 的最大章号。章号有两个来源：
//   - 章文件 NNN[字母].md 的前导章号；
//   - v4 场景制章目录 chapters/NNN/（只有场景、没有 blob 的章同样占用章号——
//     只看 .md 会让「场景承载正文的章」在补下基准里消失，G2 又回到撞车）。
//
// 目录缺失视为空集（新项目正常态），不报错。
func (m *Manager) maxChapterNumMatching(keep func(num int) bool) (int, error) {
	entries, err := os.ReadDir(filepath.Join(m.Dir, "chapters"))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	maxNum := 0
	consider := func(num int) {
		if num <= 0 || num <= maxNum || !keep(num) {
			return
		}
		maxNum = num
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			// 纯数字目录名 = 主线章目录（场景制，%03d）；「001a」这类分支目录不算主线章号。
			num := leadDigits(name)
			if num <= 0 || fmt.Sprintf("%03d", num) != name {
				continue
			}
			if m.chapterHasBodyContent(num) {
				consider(num)
			}
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		consider(leadDigits(name))
	}
	return maxNum, nil
}

// chapterHasBodyContent 该章是否已有非空正文（主线 blob 或 v4 场景承载）。
func (m *Manager) chapterHasBodyContent(num int) bool {
	if blob, err := m.ReadChapter(num); err == nil && strings.TrimSpace(blob) != "" {
		return true
	}
	sm := m.SceneManager(num)
	metas, err := sm.List()
	if err != nil {
		return false
	}
	for _, meta := range metas {
		if sc, rerr := sm.Read(meta.ID); rerr == nil && strings.TrimSpace(sc.Content) != "" {
			return true
		}
	}
	return false
}

// leadDigits 取字符串前导十进制数字；没有前导数字返回 0
// （"006-summary.json" → 6，"006a-summary.json" → 6）。
func leadDigits(s string) int {
	n, started := 0, false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
			started = true
			continue
		}
		if started {
			break
		}
		return 0
	}
	return n
}

// ParseChapterFileName 解析章节正稿文件名（chapters/ 下 NNN.md / NNN[a-z].md 家族），
// 返回章号、分支字母（无分支为空串）与是否为合法章文件名。口径与历史正则
// `^(\d{3})([a-z]?)\.md$`（原 export.listChapters / stats.Collect /
// internal/graph/consistency.go 三处重复）逐字段一致：恰好三位十进制章号
// （"000.md" 合法，章号 0）+ 至多一个小写字母分支 + 严格 ".md" 结尾。
//
// 这是仓库内章节文件名解析的**正稿家族唯一实现**（审计 IN1-03）。其余四套
// 口径因文件族/容忍度差异**刻意保留**（行为逐字段不变优先，见对照表）：
//   - mainChapterSummaryNum：NNN[a-z]-summary.json 家族，纯数字主干、分支刻意
//     排除（N5：分支剧情不得混入主线前情），谓词保留；
//   - leadDigits：前导数字宽口径（maxChapterNumMatching 接受 "001a.md"/"12.md"
//     等任意形状），谓词保留；
//   - v4 迁移 fmt.Sscanf(name, "%03d.md")：实测容忍 1-3 位数字、拒绝分支文件、
//     容忍首个 ".md" 之后的尾随内容——与本函数互不包含，保留；
//   - novelcontext.chapterNumFromFile：首段连续数字（容忍非数字前缀，大纲
//     ChapterFile 宽容口径），保留。
func ParseChapterFileName(name string) (num int, branch string, ok bool) {
	// 章号：恰好三位十进制数字（与 ChapterPath 的 fmt.Sprintf("%03d") 落盘口径对齐）
	if len(name) < 3 || !isASCIIDigits(name[:3]) {
		return 0, "", false
	}
	num = int(name[0]-'0')*100 + int(name[1]-'0')*10 + int(name[2]-'0')
	rest := name[3:]
	// 分支：至多一个小写字母
	if len(rest) > 1 && rest[0] >= 'a' && rest[0] <= 'z' {
		branch = rest[:1]
		rest = rest[1:]
	}
	if rest != ".md" {
		return 0, "", false
	}
	return num, branch, true
}

// isASCIIDigits 判断 s 是否为非空的纯 ASCII 十进制数字串。
func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
