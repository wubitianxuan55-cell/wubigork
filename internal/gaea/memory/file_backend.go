package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/frontmatter"
)

// fileBackend stores memories as Markdown files (one per fact) with a MEMORY.md
// index of one line per fact, under a per-project directory. The whole thing is
// plain files the user can edit by hand.
type fileBackend struct {
	Dir       string // ...gaea/projects/<slug>/memory (project-scoped)
	GlobalDir string // global memory directory (optional, for cross-project facts)
}

func (b *fileBackend) Index() string {
	if b.Dir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(b.Dir, indexFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (b *fileBackend) Path(name string) string {
	return filepath.Join(b.Dir, slug(name)+".md")
}

func (b *fileBackend) Save(m Memory) (string, error) {
	if b.Dir == "" {
		return "", fmt.Errorf("memory store unavailable (no user config dir)")
	}
	name := slug(m.Name)
	if name == "" {
		return "", fmt.Errorf("memory needs a name")
	}
	if err := os.MkdirAll(b.Dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(b.Dir, name+".md")
	if err := os.WriteFile(path, []byte(render(m, name)), 0o644); err != nil {
		return "", err
	}
	if err := b.reindex(name, m); err != nil {
		return path, err
	}
	return path, nil
}

// Archive moves a memory file to .archive/ instead of permanently deleting it,
// so wrong memories remain traceable and recoverable. The MEMORY.md index line
// is still removed. A missing file is not an error.
func (b *fileBackend) Archive(name string) (string, error) {
	if b.Dir == "" {
		return "", fmt.Errorf("memory store unavailable (no user config dir)")
	}
	name = slug(name)
	if name == "" {
		return "", fmt.Errorf("memory needs a name")
	}
	file := name + ".md"
	src := filepath.Join(b.Dir, file)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return "", nil // nothing to archive
	}
	archiveDir := filepath.Join(b.Dir, ".archive")
	if err := os.MkdirAll(archiveDir, 0o700); err != nil {
		return "", err
	}
	ts := time.Now().UTC().Format("20060102-150405.000")
	dest := filepath.Join(archiveDir, ts+"-"+file)
	if err := os.Rename(src, dest); err != nil {
		return "", err
	}
	if err := b.flushIndex(b.indexLinesExcept(name)); err != nil {
		return dest, err
	}
	return dest, nil
}

// Delete removes a memory — it archives first, then removes the index line.
// Uses Archive internally so wrong memories remain traceable in .archive/.
func (b *fileBackend) Delete(name string) error {
	_, err := b.Archive(name)
	return err
}

// ChangeType changes the Type of a saved memory (e.g. promote to "user" level
// or demote to "project"/"feedback"). The memory is reloaded from disk, its
// Type updated, and re-saved — all other fields are preserved.
func (b *fileBackend) ChangeType(name string, newType Type) error {
	if b.Dir == "" {
		return fmt.Errorf("memory store unavailable (no user config dir)")
	}
	name = slug(name)
	if name == "" {
		return fmt.Errorf("memory needs a name")
	}
	var target *Memory
	for _, m := range b.List() {
		if m.Name == name {
			copy := m
			target = &copy
			break
		}
	}
	if target == nil {
		return fmt.Errorf("memory %q not found", name)
	}
	target.Type = newType
	_, err := b.Save(*target)
	return err
}

// Touch 是 file 后端的生命周期 no-op：文件记忆不记录使用时间。
func (b *fileBackend) Touch(string) error { return nil }

// List returns the saved memories parsed from their files, sorted by name. Used
// by `/memory` and the desktop memory panel. Files that fail to parse are
// skipped so one bad file never hides the rest.
func (b *fileBackend) List() []Memory {
	if b.Dir == "" {
		return nil
	}
	entries, err := os.ReadDir(b.Dir)
	if err != nil {
		return nil
	}
	var out []Memory
	for _, e := range entries {
		if e.IsDir() || e.Name() == indexFile || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if m, ok := loadMemory(filepath.Join(b.Dir, e.Name())); ok {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ListArchivedPaged 返回归档文件的分页视图（按归档时间倒序）：总量 + 当前页。
// 与 sqliteBackend 同语义；文件后端归档时间取文件名时间戳前缀，缺失时回退
// 文件修改时间。
func (b *fileBackend) ListArchivedPaged(limit, offset int) ([]ArchivedMemory, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	all := b.ListArchived()
	total := len(all)
	if offset >= total {
		return []ArchivedMemory{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// CleanupArchived 硬删除归档超过 cutoff 的 .archive 文件（生命周期清理，
// T6-8.2）：返回被删除的归档条目供审计。归档时间取文件名时间戳前缀，
// 解析失败按文件修改时间判断。
func (b *fileBackend) CleanupArchived(cutoff time.Time) ([]ArchivedMemory, error) {
	if b.Dir == "" {
		return nil, nil
	}
	archiveDir := filepath.Join(b.Dir, ".archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doomed []ArchivedMemory
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		full := filepath.Join(archiveDir, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		at := parseArchiveTimestamp(e.Name(), info.ModTime())
		if !at.Before(cutoff) {
			continue
		}
		am := ArchivedMemory{
			Memory:     Memory{Name: strings.TrimSuffix(e.Name(), ".md")},
			Path:       full,
			ArchivedAt: at,
		}
		// 尽量读回文件内容作溯源（解析失败保留文件名级信息）
		if m, ok := loadMemory(full); ok {
			am.Memory = m
		}
		if err := os.Remove(full); err != nil {
			return doomed, err
		}
		doomed = append(doomed, am)
	}
	return doomed, nil
}

// parseArchiveTimestamp 解析归档文件名时间戳前缀（20060102-150405.000-<name>.md，
// 前缀固定 19 字符）；失败回退 modTime。
func parseArchiveTimestamp(name string, fallback time.Time) time.Time {
	if len(name) >= 19 {
		if t, err := time.ParseInLocation("20060102-150405.000", name[:19], time.UTC); err == nil {
			return t
		}
	}
	return fallback
}
// dirs returns the non-empty store directories to scan. Project-scoped Dir
// takes priority; GlobalDir (when set) is also included for cross-project facts.
func (b *fileBackend) dirs() []string {
	var out []string
	if b.Dir != "" {
		out = append(out, b.Dir)
	}
	if b.GlobalDir != "" && b.GlobalDir != b.Dir {
		out = append(out, b.GlobalDir)
	}
	return out
}

// ListArchived returns archived memories parsed from .archive/, newest first.
// Archived files stay out of List() and the prompt index, so stale facts remain
// inspectable without being reused as active truth.
func (b *fileBackend) ListArchived() []ArchivedMemory {
	if b.Dir == "" && b.GlobalDir == "" {
		return nil
	}
	var out []ArchivedMemory
	for _, base := range b.dirs() {
		if base == "" {
			continue
		}
		dir := filepath.Join(base, ".archive")
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			m, ok := loadMemory(path)
			if !ok {
				continue
			}
			when := archiveTimeFromName(e.Name())
			if when.IsZero() {
				if info, err := e.Info(); err == nil {
					when = info.ModTime()
				}
			}
			out = append(out, ArchivedMemory{Memory: m, Path: path, ArchivedAt: when})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].ArchivedAt.Equal(out[j].ArchivedAt) {
			return out[i].ArchivedAt.After(out[j].ArchivedAt)
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// render serializes a memory to frontmatter + body. The frontmatter mirrors the
// auto-memory shape (name / description / metadata.type / metadata.kind) so the
// files are interchangeable with that ecosystem and re-readable by loadMemory.
func render(m Memory, name string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: " + name + "\n")
	if t := oneLine(m.Title); t != "" {
		b.WriteString("title: " + t + "\n")
	}
	b.WriteString("description: " + oneLine(m.Description) + "\n")
	b.WriteString("metadata:\n")
	b.WriteString("  type: " + string(NormalizeType(string(m.Type))) + "\n")
	if k := NormalizeKind(string(m.Kind)); k != KindSemantic {
		b.WriteString("  kind: " + string(k) + "\n")
	}
	if len(m.Tags) > 0 {
		b.WriteString("  tags: [" + strings.Join(m.Tags, ", ") + "]\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(m.Body))
	b.WriteString("\n")
	return b.String()
}

// indexLineRe matches a managed index line so reindex/Delete can target the line
// for one memory by its filename without disturbing the rest of a hand-edited
// MEMORY.md.
var indexLineRe = regexp.MustCompile(`\]\(([^)]+)\.md\)`)

// indexLinesExcept returns the managed MEMORY.md lines keyed by filename stem,
// dropping the entry for name (a missing index → empty map).
func (b *fileBackend) indexLinesExcept(name string) map[string]string {
	existing, _ := os.ReadFile(filepath.Join(b.Dir, indexFile))
	keep := map[string]string{}
	for _, line := range strings.Split(string(existing), "\n") {
		if mt := indexLineRe.FindStringSubmatch(line); mt != nil && mt[1] != name {
			keep[mt[1]] = strings.TrimRight(line, "\r")
		}
	}
	return keep
}

// flushIndex rewrites MEMORY.md from the managed lines, sorted by filename.
func (b *fileBackend) flushIndex(lines map[string]string) error {
	names := make([]string, 0, len(lines))
	for n := range lines {
		names = append(names, n)
	}
	sort.Strings(names)

	var buf strings.Builder
	buf.WriteString("# Memory\n\n")
	for _, n := range names {
		buf.WriteString(lines[n])
		buf.WriteString("\n")
	}
	return os.WriteFile(filepath.Join(b.Dir, indexFile), []byte(buf.String()), 0o644)
}

// reindex rewrites the MEMORY.md line for name, preserving every other managed
// line. The line is "- [<title>](<name>.md) — <description>"; title falls back
// to a de-kebabed name so the index reads as a label, never a bare slug.
// Kind is shown as a prefix tag when non-semantic: [E] for episodic, [P] for procedural.
func (b *fileBackend) reindex(name string, m Memory) error {
	lines := b.indexLinesExcept(name)
	kindTag := ""
	switch NormalizeKind(string(m.Kind)) {
	case KindEpisodic:
		kindTag = "[E] "
	case KindProcedural:
		kindTag = "[P] "
	}
	lines[name] = fmt.Sprintf("- %s[%s](%s.md) — %s", kindTag, displayTitle(m.Title, name), name, oneLine(m.Description))
	return b.flushIndex(lines)
}

// archiveTimeFromName extracts the timestamp from an archived filename, which is
// prefixed with "20060102-150405.000-" by Archive.
func archiveTimeFromName(name string) time.Time {
	const stampLen = len("20060102-150405.000")
	if len(name) <= stampLen || name[stampLen] != '-' {
		return time.Time{}
	}
	t, err := time.Parse("20060102-150405.000", name[:stampLen])
	if err != nil {
		return time.Time{}
	}
	return t
}

// loadMemory parses one fact file back into a Memory. It tolerates the minimal
// frontmatter render writes; a file without frontmatter still loads with its
// body and a name derived from the filename.
func loadMemory(path string) (Memory, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Memory{}, false
	}
	fm, body := splitFrontmatter(string(b))
	m := Memory{
		Name:        fm["name"],
		Title:       fm["title"],
		Description: fm["description"],
		Type:        NormalizeType(fm["type"]),
		Kind:        NormalizeKind(fm["kind"]),
		Tags:        parseTags(fm["tags"]),
		Body:        strings.TrimSpace(body),
	}
	if m.Name == "" {
		m.Name = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	return m, true
}
// splitFrontmatter is a thin wrapper; the real parser lives in
// internal/frontmatter.
func splitFrontmatter(s string) (map[string]string, string) {
	return frontmatter.Split(s)
}

// Get returns one active fact by name. For the file backend this reads the
// single <name>.md; missing or unparsable files yield (Memory{}, false).
func (b *fileBackend) Get(name string) (Memory, bool) {
	if b.Dir == "" {
		return Memory{}, false
	}
	name = slug(name)
	if name == "" {
		return Memory{}, false
	}
	return loadMemory(filepath.Join(b.Dir, name+".md"))
}
