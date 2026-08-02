package memory

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Type classifies a memory, mirroring the auto-memory taxonomy.
type Type string

const (
	TypeUser      Type = "user"      // who the user is: role, preferences, expertise
	TypeFeedback  Type = "feedback"  // guidance on how to work (with why + how-to-apply)
	TypeProject   Type = "project"   // ongoing work / goals / constraints not in the code
	TypeReference Type = "reference" // pointers to external resources (URLs, tickets)
)

// Kind classifies a memory by cognitive function (LangMem-inspired).
// Orthogonal to Type — Kind controls how the memory is injected and retrieved.
type Kind string

const (
	KindSemantic   Kind = "semantic"   // facts, preferences, constraints → L1 prefix + search
	KindEpisodic   Kind = "episodic"   // past experiences, solutions → tag-triggered injection
	KindProcedural Kind = "procedural" // rules, best practices → always-on injection
)

// validTypes is the closed set the `remember` tool accepts; anything else
// normalises to TypeProject.
var validTypes = map[Type]bool{TypeUser: true, TypeFeedback: true, TypeProject: true, TypeReference: true}

// validKinds is the closed set for the Kind field.
var validKinds = map[Kind]bool{KindSemantic: true, KindEpisodic: true, KindProcedural: true}

// NormalizeType coerces an arbitrary string to a known Type, defaulting to
// TypeProject so a sloppy tool argument never blocks a save.
func NormalizeType(s string) Type {
	t := Type(strings.ToLower(strings.TrimSpace(s)))
	if validTypes[t] {
		return t
	}
	return TypeProject
}

// NormalizeKind coerces an arbitrary string to a known Kind, defaulting to
// KindSemantic so existing memories without an explicit kind stay semantic.
func NormalizeKind(s string) Kind {
	k := Kind(strings.ToLower(strings.TrimSpace(s)))
	if validKinds[k] {
		return k
	}
	return KindSemantic
}

// Memory is one stored fact.
type Memory struct {
	Name        string   // kebab-case slug; also the file stem (<name>.md)
	Title       string   // human-readable index label; falls back to a de-kebabed Name
	Description string   // one-line summary used for the index and recall
	Type        Type     // category: user / feedback / project / reference
	Kind        Kind     // cognitive function: semantic / episodic / procedural
	Tags        []string // trigger tags for episodic memories (empty for others)
	Body        string   // the fact itself (Markdown)
}

// ArchivedMemory is a saved fact that has been removed from active memory but
// kept on disk for traceability. The ArchivedAt timestamp records when it was
// archived; Path is the absolute path to the archived .md file.
type ArchivedMemory struct {
	Memory
	Path       string    `json:"path"`
	ArchivedAt time.Time `json:"archivedAt"`
}

// backend is the storage engine behind Store. Two implementations exist:
// fileBackend (Markdown files + MEMORY.md index) and sqliteBackend (gaea.db).
// A nil backend in Store falls back to a fileBackend built from Dir/GlobalDir,
// which keeps zero-value `Store{Dir: ...}` constructions (used throughout the
// test suite) working unchanged.
type backend interface {
	Index() string
	Path(name string) string
	Save(m Memory) (string, error)
	Archive(name string) (string, error)
	Delete(name string) error
	ChangeType(name string, newType Type) error
	List() []Memory
	ListArchived() []ArchivedMemory
	Get(name string) (Memory, bool)
}

// Store is the memory storage facade — one per project working dir. It stays a
// value struct (not an interface) so call sites and zero-value constructions
// keep compiling; the actual engine is the internal backend.
type Store struct {
	Dir       string // file backend: ...gaea/projects/<slug>/memory (project-scoped)
	GlobalDir string // file backend: global memory directory (cross-project facts)
	backend   backend
}

// engine returns the active backend, falling back to a file backend derived
// from the legacy Dir/GlobalDir fields (test constructions, zero Store).
func (s Store) engine() backend {
	if s.backend != nil {
		return s.backend
	}
	return &fileBackend{Dir: s.Dir, GlobalDir: s.GlobalDir}
}

// StoreFor resolves the auto-memory store for a project working dir under the
// user config root, e.g. ~/.config/gaea/projects/-Users-me-proj/memory.
// A "" userDir (config dir unresolvable) yields a zero Store, which all methods
// treat as a disabled no-op.
func StoreFor(userDir, cwd string) Store {
	if userDir == "" {
		return Store{}
	}
	dir := filepath.Join(userDir, "projects", slugify(absOf(cwd)), "memory")
	return Store{Dir: dir, backend: &fileBackend{Dir: dir}}
}

// SQLiteStoreFor returns a Store backed by the gaea.db SQLite database. Facts
// are scoped per project (slugified cwd). A nil db or empty userDir yields a
// disabled zero Store.
func SQLiteStoreFor(db *sql.DB, userDir, cwd string) Store {
	if db == nil || userDir == "" {
		return Store{}
	}
	return Store{backend: &sqliteBackend{db: db, project: slugify(absOf(cwd))}}
}

// indexFile is the human-readable index of saved memories (file backend).
const indexFile = "MEMORY.md"

// slugify turns an absolute project path into a single filesystem-safe segment,
// matching the auto-memory convention (path separators → '-'), e.g.
// "/Users/me/proj" → "-Users-me-proj".
func slugify(absPath string) string {
	r := strings.NewReplacer(string(os.PathSeparator), "-", "/", "-", "\\", "-", ":", "-")
	return r.Replace(absPath)
}

// Index returns the memory index text (MEMORY.md for the file backend; the
// rendered per-line index for SQLite), or "" if there are none yet. This is
// what loads into the cached prefix.
func (s Store) Index() string { return s.engine().Index() }

// Path returns the storage location for a memory with the given name. For the
// file backend this is the absolute .md path; for SQLite it is a logical
// "gaea.db:<project>/<name>" reference.
func (s Store) Path(name string) string { return s.engine().Path(name) }

// Save writes (or overwrites) a memory and refreshes the index. It is the
// single mutation entry point — the `remember` tool, the desktop editor, and
// any future importer all go through here so the index never drifts. Returns
// the location written.
func (s Store) Save(m Memory) (string, error) { return s.engine().Save(m) }

// Archive moves a memory out of active memory instead of permanently deleting
// it, so wrong memories remain traceable and recoverable. A missing memory is
// not an error.
func (s Store) Archive(name string) (string, error) { return s.engine().Archive(name) }

// Delete removes a memory — it archives first, so wrong memories remain
// traceable.
func (s Store) Delete(name string) error { return s.engine().Delete(name) }

// ChangeType changes the Type of a saved memory (e.g. promote to "user" level
// or demote to "project"/"feedback"). All other fields are preserved.
func (s Store) ChangeType(name string, newType Type) error {
	return s.engine().ChangeType(name, newType)
}

// List returns the saved memories, sorted by name. Used by /memory and the
// desktop memory panel.
func (s Store) List() []Memory { return s.engine().List() }

// ListArchived returns archived memories, newest first. Archived facts stay
// out of List() and the prompt index, so stale facts remain inspectable without
// being reused as active truth.
func (s Store) ListArchived() []ArchivedMemory { return s.engine().ListArchived() }

// Get returns one active fact by name. Used by the memory_get tool (SQLite
// backend has no readable file to open with read_file).
func (s Store) Get(name string) (Memory, bool) { return s.engine().Get(name) }

// slugRe strips everything but lowercase alphanumerics and dashes.
var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slug normalises a name into a kebab-case, filesystem-safe stem.
func slug(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-"), "-")
}

// oneLine collapses whitespace so a description can't break the single-line
// index or frontmatter format.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// displayTitle is the index link label: the given title, or a de-kebabed name
// when none was supplied, so a bare slug never leaks into the index.
func displayTitle(title, name string) string {
	if t := oneLine(title); t != "" {
		return t
	}
	return strings.ReplaceAll(name, "-", " ")
}

// parseTags parses a frontmatter tags value. Accepts JSON array syntax
// [a, b, c] or comma-separated plain text. Returns nil for empty input.
func parseTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		inner := strings.TrimSpace(raw[1 : len(raw)-1])
		if inner == "" {
			return nil
		}
		parts := strings.Split(inner, ",")
		var out []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.Trim(p, "\"'")
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	parts := strings.Split(raw, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
