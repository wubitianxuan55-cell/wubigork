// schedule/index.go — 进度计划多工程索引（v4.139 差距 #15 刀1+刀2）。
//
// 设计：docs/gaea-schedule-multi-project-design-2026-09.md §3.1/§3.3。
// 布局=每文件一工程：进度计划/*.gsched.json 是工程事实真相（权威），
// .gaea/schedule/index.json 只是注册表+当前指针（缓存，可随时重建）。
// 索引缺失/损坏/与目录漂移 → 扫描重建，宁重建勿报错；Load 时轻扫目录
// diff 收编游离文件（name 从文件内 project.name 轻量读取，不全文校验）。

package schedule

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ScheduleProjectsDir 工程文件目录（工作区相对，canonical 用 / 分隔）。
const ScheduleProjectsDir = "进度计划"

// ScheduleFileSuffix 工程文件后缀（与前端 isScheduleFilePath 同口径）。
const ScheduleFileSuffix = ".gsched.json"

// IndexRelPath 工程索引文件（工作区相对；.gaea/ 元数据惯例，与用户资产
// 目录「进度计划/」分离防误删）。
const IndexRelPath = ".gaea/schedule/index.json"

// ScheduleIndexEntry 索引条目：注册表一行 + 归档标记。
type ScheduleIndexEntry struct {
	Rel       string    `json:"rel"`       // 工作区相对路径（/ 分隔）
	Name      string    `json:"name"`      // 显示名（文件内 project.name 的登记副本）
	Archived  bool      `json:"archived"`  // 归档只标索引，文件不动（agent 显式 path 引用不失效）
	UpdatedAt time.Time `json:"updatedAt"` // 最近保存/登记时间
}

// ScheduleIndex 工程注册表+当前指针。Version=1；Current 为空回落
// DefaultRelPath（兼容老用户单文件形态，零感知）。
type ScheduleIndex struct {
	Version  int                  `json:"version"`
	Current  string               `json:"current"`
	Projects []ScheduleIndexEntry `json:"projects"`
}

// LoadScheduleIndex 装载索引（含自愈）：
//  1. 索引可读 → 轻扫目录 diff：收编游离文件（含 name 回填）、摘除已消失
//     条目、指针失效时重选；有变化则原子写回。
//  2. 索引缺失/损坏 → 扫描重建（current 优先 DefaultRelPath，否则第一个条目）。
//  3. 目录都不可读（极端环境）→ 返回空索引与错误，调用方回落 DefaultRelPath。
func LoadScheduleIndex(dir string) (ScheduleIndex, error) {
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(IndexRelPath)))
	if err == nil {
		var idx ScheduleIndex
		if json.Unmarshal(raw, &idx) == nil && idx.Version > 0 {
			return adoptIndex(dir, idx)
		}
		// 损坏（解析失败/版本缺省）→ 落到扫描重建
	}
	return rebuildIndex(dir)
}

// SaveScheduleIndex 原子写索引（临时文件+改名；MkdirAll 建父目录）。
func SaveScheduleIndex(dir string, idx ScheduleIndex) error {
	if idx.Version <= 0 {
		idx.Version = 1
	}
	if idx.Projects == nil {
		idx.Projects = []ScheduleIndexEntry{}
	}
	raw, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	target := filepath.Join(dir, filepath.FromSlash(IndexRelPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("创建索引目录失败：%w", err)
	}
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("写入索引临时文件失败：%w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("替换索引文件失败：%w", err)
	}
	return nil
}

// SetCurrentSchedule 切换当前工程指针：校验 rel 合法（在注册表内，或文件
// 在盘且后缀 .gsched.json）→ 未登记的顺带收编 → 原子写索引。不搬数据、
// 不改工程文件。
func SetCurrentSchedule(dir, rel string) error {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return fmt.Errorf("工程路径为空")
	}
	if err := validateScheduleRel(rel); err != nil {
		return err
	}
	idx, err := LoadScheduleIndex(dir)
	if err != nil {
		return err
	}
	registered := false
	for i := range idx.Projects {
		if idx.Projects[i].Rel == rel {
			registered = true
			break
		}
	}
	if !registered {
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); err != nil {
			return fmt.Errorf("工程不存在或未登记：%s", rel)
		}
		idx.Projects = append(idx.Projects, ScheduleIndexEntry{
			Rel:       rel,
			Name:      readProjectName(abs, relBaseName(rel)),
			UpdatedAt: time.Now(),
		})
	}
	idx.Current = rel
	return SaveScheduleIndex(dir, idx)
}

// ScanScheduleProjects 扫描 进度计划/*.gsched.json（权威口径）。条目按 rel
// 稳定排序；name 轻量读文件内 project.name（只取 name 字段，不全文校验；
// 读不出来回落文件名）；UpdatedAt 取文件修改时间。目录不存在=空集非错误。
func ScanScheduleProjects(dir string) ([]ScheduleIndexEntry, error) {
	root := filepath.Join(dir, filepath.FromSlash(ScheduleProjectsDir))
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScheduleIndexEntry{}, nil
		}
		return nil, err
	}
	out := make([]ScheduleIndexEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if !strings.HasSuffix(e.Name(), ScheduleFileSuffix) {
			continue
		}
		var mod time.Time
		if info, ierr := e.Info(); ierr == nil {
			mod = info.ModTime()
		}
		abs := filepath.Join(root, e.Name())
		out = append(out, ScheduleIndexEntry{
			Rel:       path.Join(ScheduleProjectsDir, e.Name()),
			Name:      readProjectName(abs, relBaseName(e.Name())),
			UpdatedAt: mod,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, nil
}

// SafeSlugName 工程名 → 安全文件名 slug（不含目录与后缀）：中文与常规字母
// 数字保留；Windows 非法路径字符（\/:*?"<>|）与控制符替换为「-」；去首尾
// 空白与 - . ；清洗后为空回落「未命名」；与 existing 冲突加序号 -2/-3…，
// 选中结果登记回 existing（nil 则只判重不登记）。上限 60 字符防超 Windows
// 路径长度。此后文件名不再改（agent 显式 path 引用稳定）。
func SafeSlugName(name string, existing map[string]bool) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(`\/:*?"<>|`, r) {
			b.WriteRune('-')
			continue
		}
		b.WriteRune(r)
	}
	slug := strings.Trim(b.String(), "-. ")
	if runes := []rune(slug); len(runes) > 60 {
		slug = strings.Trim(string(runes[:60]), "-. ")
	}
	if slug == "" {
		slug = "未命名"
	}
	if !existing[slug] {
		if existing != nil {
			existing[slug] = true
		}
		return slug
	}
	for n := 2; ; n++ {
		cand := fmt.Sprintf("%s-%d", slug, n)
		if !existing[cand] {
			if existing != nil {
				existing[cand] = true
			}
			return cand
		}
	}
}

// ── 内部：自愈与轻量读取 ─────────────────────────────────────────────

// adoptIndex 索引可读时的轻扫收编：以索引元数据（archived/name/updatedAt）
// 为准，目录只做 diff——收编游离、摘除消失、修指针。
func adoptIndex(dir string, idx ScheduleIndex) (ScheduleIndex, error) {
	scanned, serr := ScanScheduleProjects(dir)
	if serr != nil {
		// 目录扫不动时索引原样可用（缓存语义），下次 Load 再收编。
		return idx, nil
	}
	scannedSet := make(map[string]ScheduleIndexEntry, len(scanned))
	for _, e := range scanned {
		scannedSet[e.Rel] = e
	}
	kept := make([]ScheduleIndexEntry, 0, len(idx.Projects)+len(scanned))
	keptSet := make(map[string]bool, len(idx.Projects)+len(scanned))
	for _, e := range idx.Projects {
		if se, ok := scannedSet[e.Rel]; ok {
			if strings.TrimSpace(e.Name) == "" {
				e.Name = se.Name // name 回填
			}
		} else if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(e.Rel))); err != nil {
			continue // 文件已消失（用户手删）→ 摘除
		}
		// 注：扫描集外的在盘文件（理论不该有）保守保留，不吃合法条目。
		if !keptSet[e.Rel] {
			keptSet[e.Rel] = true
			kept = append(kept, e)
		}
	}
	changed := len(kept) != len(idx.Projects)
	for _, se := range scanned {
		if !keptSet[se.Rel] {
			keptSet[se.Rel] = true
			kept = append(kept, se)
			changed = true
		}
	}
	idx.Projects = kept
	if idx.Current == "" || !keptSet[idx.Current] {
		idx.Current = pickCurrentRel(kept)
		changed = true
	}
	if changed {
		if err := SaveScheduleIndex(dir, idx); err != nil {
			return idx, err
		}
	}
	return idx, nil
}

// rebuildIndex 扫描重建（宁重建勿报错）：current 优先 DefaultRelPath（老
// 用户零感知），否则第一个条目；都没有也落 DefaultRelPath 指针（文件尚不
// 存在时由调用方的空态/落盘分支兜住）。
func rebuildIndex(dir string) (ScheduleIndex, error) {
	scanned, err := ScanScheduleProjects(dir)
	if err != nil {
		return ScheduleIndex{Version: 1, Current: DefaultRelPath, Projects: []ScheduleIndexEntry{}}, err
	}
	idx := ScheduleIndex{Version: 1, Current: pickCurrentRel(scanned), Projects: scanned}
	if err := SaveScheduleIndex(dir, idx); err != nil {
		return idx, err
	}
	return idx, nil
}

// pickCurrentRel 重建/指针失效时选当前：DefaultRelPath 优先，否则第一个条目。
func pickCurrentRel(entries []ScheduleIndexEntry) string {
	for _, e := range entries {
		if e.Rel == DefaultRelPath {
			return DefaultRelPath
		}
	}
	if len(entries) > 0 {
		return entries[0].Rel
	}
	return DefaultRelPath
}

// validateScheduleRel 相对路径合法性：后缀必须 .gsched.json，且禁止 .. 逃逸
// 工作区（Open/Delete 等绑定入参的最低防线）。
func validateScheduleRel(rel string) error {
	if !strings.HasSuffix(rel, ScheduleFileSuffix) {
		return fmt.Errorf("非法计划文件后缀（须 .gsched.json）：%s", rel)
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".." {
			return fmt.Errorf("非法计划文件路径（禁止 .. 逃逸）：%s", rel)
		}
	}
	return nil
}

// readProjectName 轻量读文件内 project.name（只取 name 字段，不全文校验）；
// 读不出/为空回落 fallback。
func readProjectName(absPath, fallback string) string {
	raw, err := os.ReadFile(absPath)
	if err == nil {
		var head struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(raw, &head) == nil && strings.TrimSpace(head.Name) != "" {
			return strings.TrimSpace(head.Name)
		}
	}
	return fallback
}

// relBaseName 相对路径 → 去目录去后缀的基础名（slug 去重与空工程命名用）。
func relBaseName(rel string) string {
	return strings.TrimSuffix(path.Base(filepath.ToSlash(rel)), ScheduleFileSuffix)
}
